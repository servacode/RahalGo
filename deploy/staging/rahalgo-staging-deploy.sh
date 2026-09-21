#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  rahalgo-staging-deploy — the ONLY action the staging deploy identity runs
# ══════════════════════════════════════════════════════════════════════
#
# Installed by bootstrap to:  /usr/local/sbin/rahalgo-staging-deploy  (root:root 0755)
# Invoked ONLY as:            sudo -n /usr/local/sbin/rahalgo-staging-deploy
# (via the deploy identity's forced-command key; sudoers permits exactly this one
#  command and keeps SSH_ORIGINAL_COMMAND). The deploy identity is NOT in the
# docker group and has no shell — it can do nothing but trigger this wrapper,
# which then runs as root.
#
# Contract:
#   - SSH_ORIGINAL_COMMAND (or $1) = exactly ONE 40-char lowercase git SHA
#   - a `git bundle` (containing refs/heads/deploy-target = that SHA) on STDIN
#   - the wrapper CRYPTOGRAPHICALLY verifies the bundle resolves to that SHA
#     before building anything stamped with it
#   - everything under /srv/rahalgo-staging ONLY; production never mutated
#   - fail closed
#
# Modes (for off-box self-test; neither touches docker/network/production):
#   --check <sha>                 input validation only
#   --verify-bundle <sha> <file>  input validation + bundle<->SHA crypto binding
#
# Exit: 10 bad SHA · 11 production-tainted · 12 missing env · 13 bad bundle ·
#       14 bundle commit != requested SHA · 15 not root · 3 envguard refused ·
#       5 post-deploy identity/env/commit mismatch · 6 production identity changed · 0 ok
set -euo pipefail

STAGING_ROOT=/srv/rahalgo-staging
STAGING_ENV="$STAGING_ROOT/deploy/staging/.env.staging"
IDENTITY_URL="http://localhost:8080/api/v1/public/identity"
HEALTH_URL="http://localhost:8080/api/v1/healthz"
INCOMING="$STAGING_ROOT/incoming"
ARTIFACT_DIR="$STAGING_ROOT/artifacts"
PROD_PROJECT=rahalgo
STAGING_PROJECT=rahalgo-staging

log(){ printf '%s\n' "$*" >&2; }
die(){ printf 'x %s\n' "$*" >&2; exit "${2:-1}"; }

# ── validate a 40-char lowercase-hex SHA ──────────────────────────────
validate_sha(){
	case "$1" in
		"")          die "no SHA provided" 10 ;;
		*[!0-9a-f]*) die "SHA rejected — not 40 lowercase hex: '$1'" 10 ;;
	esac
	[ "${#1}" -eq 40 ] || die "SHA rejected — length ${#1} != 40" 10
}

# ── refuse anything that points at production ─────────────────────────
refuse_production(){
	[ "$STAGING_ROOT" = /srv/rahalgo-staging ] || die "staging root misconfigured — fail closed" 11
	local v
	for v in DATABASE_URL REDIS_URL COMPOSE_FILE ENV_FILE STAGING_ROOT; do
		case "${!v:-}" in
			*rahalgo_prod*|*/srv/rahalgo/*|*api.rahalgo.com*|*:5432/rahalgo\?sslmode*)
				die "refusing production-tainted \$$v" 11 ;;
		esac
	done
}

# ── verify a git bundle resolves EXACTLY to the requested SHA, then ────
#    extract that commit's tree.  git content-addressing binds source<->SHA:
#    a bundle carrying commit A cannot satisfy a request for SHA B.
verify_bundle_to_src(){ # <sha> <bundle> <destdir>
	local sha="$1" bundle="$2" dest="$3" work got
	command -v git >/dev/null 2>&1 || return 13
	work="$(mktemp -d)"
	git init -q "$work"                                                || { rm -rf "$work"; return 13; }
	git -C "$work" bundle verify "$bundle"      >/dev/null 2>&1        || { rm -rf "$work"; return 13; }
	git -C "$work" fetch -q "$bundle" refs/heads/deploy-target 2>/dev/null || { rm -rf "$work"; return 13; }
	got="$(git -C "$work" rev-parse FETCH_HEAD 2>/dev/null || true)"
	[ "$got" = "$sha" ]                                               || { rm -rf "$work"; return 14; }
	[ "$(git -C "$work" cat-file -t "$sha" 2>/dev/null)" = commit ]   || { rm -rf "$work"; return 14; }
	mkdir -p "$dest"
	git -C "$work" archive "$sha" | tar -x -C "$dest"                 || { rm -rf "$work"; return 13; }
	rm -rf "$work"
	return 0
}

# ── production snapshot: container ID + image ID + state (identity) ────
prod_snapshot(){
	local id
	for id in $(docker ps -aq --filter "label=com.docker.compose.project=$PROD_PROJECT" 2>/dev/null | sort); do
		docker inspect --format '{{.Id}}={{.Image}}={{.State.Status}}' "$id" 2>/dev/null
	done | sort
}

# ── resolve the request ───────────────────────────────────────────────
REQ="${1:-${SSH_ORIGINAL_COMMAND:-}}"; REQ="${REQ%% *}"

case "${REQ:-}" in
	--check)
		validate_sha "${2:-}"; refuse_production
		[ -f "$STAGING_ENV" ] && log "check: persistent staging env present" \
			|| log "check(note): persistent staging env not on this host (expected only on the box)"
		log "OK check passed: staging-only, production refused"; exit 0 ;;
	--verify-bundle)
		validate_sha "${2:-}"; refuse_production
		d="$(mktemp -d)"; rc=0; verify_bundle_to_src "${2:-}" "${3:?bundle path}" "$d" || rc=$?; rm -rf "$d"
		[ "$rc" -eq 0 ] && { log "OK bundle is bound to ${2:0:8}"; exit 0; }
		[ "$rc" -eq 14 ] && die "bundle commit != requested SHA — rejected" 14
		die "bundle invalid — rejected" 13 ;;
esac

# ── real deploy ───────────────────────────────────────────────────────
validate_sha "$REQ"; refuse_production
SHA="$REQ"; SHORT="${SHA:0:8}"
[ "$(id -u)" -eq 0 ] || die "must run as root (via sudo forced command)" 15
[ -f "$STAGING_ENV" ] || die "persistent staging env missing: $STAGING_ENV" 12

PROD_BEFORE="$(prod_snapshot || true)"
log "-- production containers before: $(printf '%s' "$PROD_BEFORE" | wc -l) recorded"

# receive + cryptographically bind the source to the SHA
mkdir -p "$INCOMING" "$ARTIFACT_DIR"
BUNDLE="$INCOMING/$SHA.bundle"; SRC="$INCOMING/$SHA"
rm -rf "$SRC" "$BUNDLE"; mkdir -p "$SRC"
cat > "$BUNDLE"          # git bundle from stdin
rc=0; verify_bundle_to_src "$SHA" "$BUNDLE" "$SRC" || rc=$?
[ "$rc" -eq 14 ] && die "received source is NOT commit $SHORT — refused before build" 14
[ "$rc" -eq 0 ]  || die "source bundle invalid — refused" 13
[ -f "$SRC/deploy/build-artifact.sh" ] && [ -f "$SRC/deploy/staging/compose.staging.yml" ] \
	|| die "verified commit is not the RahalGo repo — refused" 13

# staging environment (mirrors deploy/staging/deploy.sh), then envguard.
# **Normalize CRLF -> LF**: an owner-edited .env.staging may carry Windows line
# endings; sourcing it raw fails ($'\r': command not found) and --env-file would
# fold a trailing \r into secret values (e.g. the DB password). We use a sanitized
# LF copy for sourcing AND for compose/promote --env-file.
ENVLF="$(mktemp)"; trap 'rm -f "$ENVLF"' EXIT
tr -d '\r' < "$STAGING_ENV" > "$ENVLF"; chmod 600 "$ENVLF"
set -a; . "$ENVLF"; set +a
export APP_ENV=staging RAHALGO_STAGING=1
export DATABASE_URL="postgres://rahalgo:${STAGING_DB_PASSWORD}@localhost:5534/rahalgo_staging?sslmode=disable"
export REDIS_URL="redis://localhost:6580/0"
export NEXT_PUBLIC_API_URL="${STAGING_API_URL}"
case "$DATABASE_URL" in *5534/rahalgo_staging*) ;; *) die "DATABASE_URL is not the staging DB — fail closed" 11 ;; esac
( cd "$SRC/backend" && go run ./cmd/stagingctl guard ) || die "stagingctl guard refused the environment — fail closed" 3
STRICT=0 "$SRC/deploy/preflight-env.sh" "$IDENTITY_URL" staging || die "preflight: target is not staging — fail closed" 11

# build the verified commit in archive mode (identity injected + verified)
export SOURCE_COMMIT="$SHA" SRC_ROOT="$SRC"
ART="$("$SRC/deploy/build-artifact.sh")"; printf '%s\n' "$ART" | sed 's/^/   build: /' >&2
API_TAG="$(printf '%s\n' "$ART" | sed -n 's/^API_IMAGE_TAG=//p')"
API_ID="$(printf '%s\n'  "$ART" | sed -n 's/^API_IMAGE_ID=//p')"
WEB_TAG="$(printf '%s\n' "$ART" | sed -n 's/^WEB_IMAGE_TAG=//p')"
[ -n "$API_TAG" ] && [ -n "$WEB_TAG" ] || die "build produced no image tags — fail closed" 4

# bring up staging (non-api) then promote api by artifact
export RAHALGO_API_IMAGE="$API_TAG" RAHALGO_WEB_IMAGE="$WEB_TAG"
docker compose -p "$STAGING_PROJECT" -f "$SRC/deploy/staging/compose.staging.yml" --env-file "$ENVLF" \
	up -d --no-build --no-deps caddy web postgres redis
TARGET_ENV=staging "$SRC/deploy/promote.sh" \
	"$SRC/deploy/staging/compose.staging.yml" "$ENVLF" "$API_TAG" "$IDENTITY_URL" "$API_ID"

# verify the LIVE staging runtime is exactly what we built
for _ in $(seq 1 60); do curl -fsS "$HEALTH_URL" >/dev/null 2>&1 && break; sleep 2; done
curl -fsS "$HEALTH_URL" >/dev/null 2>&1 || die "staging /healthz did not come up" 5
ID="$(curl -fsS "$IDENTITY_URL")" || die "staging identity did not answer" 5
printf 'IDENTITY=%s\n' "$ID" >&2
case "$ID" in *'"environment":"staging"'*) ;; *) die "live env is not staging — fail closed" 5 ;; esac
case "$ID" in *"$SHA"*) ;; *) die "live source_commit != requested SHA — fail closed" 5 ;; esac

# PROVE production was not replaced (container-ID + image-ID + state)
PROD_AFTER="$(prod_snapshot || true)"
if [ "$PROD_BEFORE" != "$PROD_AFTER" ]; then
	log "-- production before:"; printf '%s\n' "$PROD_BEFORE" | sed 's/^/     /' >&2
	log "-- production after :"; printf '%s\n' "$PROD_AFTER"  | sed 's/^/     /' >&2
	die "PRODUCTION CONTAINER IDENTITY CHANGED — isolation breach" 6
fi

MIG="$(printf '%s' "$ID" | grep -o '"migration_version":"[^"]*"' | cut -d'"' -f4)"
log "OK staging now serves $SHORT · migration=$MIG · production unchanged"
printf 'DEPLOYED_SHA=%s\nMIGRATION_VERSION=%s\nPRODUCTION_MUTATIONS=0\n' "$SHA" "$MIG"
