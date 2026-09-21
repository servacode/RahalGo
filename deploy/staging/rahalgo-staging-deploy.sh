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

# ── restore a sane runtime under sudo (PATH/HOME) + the Go dir bootstrap found/installed
GO_CONF=/etc/rahalgo-staging-deploy.conf
load_runtime(){
	export PATH="/usr/local/go/bin:/opt/rahalgo-go/bin:/snap/bin:/root/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"
	export HOME="${HOME:-/root}"
	# bootstrap writes GO_BIN_DIR here after detecting/installing a suitable Go
	[ -r "$GO_CONF" ] && . "$GO_CONF" 2>/dev/null || true
	[ -n "${GO_BIN_DIR:-}" ] && export PATH="$GO_BIN_DIR:$PATH"
}

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

# ── on-box readiness: prove the REAL sudo runtime can deploy — no mutation ──
# Invoked by bootstrap as: sudo -u <deploy-user> sudo -n <wrapper> --readiness
# so it exercises the exact deploy path. Builds nothing on staging, touches no
# staging/production container, prints no secret values.
readiness(){
	local rf=0
	rok(){ echo "  PASS $1"; }; rno(){ echo "  FAIL $1"; rf=1; }; rw(){ echo "  WARN $1"; }
	echo "== on-box readiness (no staging mutation) =="

	if [ "$(id -u)" -eq 0 ]; then rok "wrapper runs as root via deploy-user -> sudo path"
	else echo "  FAIL not root — sudo/forced-command path broken"; echo "READINESS: FAIL"; return 1; fi

	load_runtime

	# persistent env: normalize + parse WITHOUT printing any value
	if [ -f "$STAGING_ENV" ]; then
		local e; e="$(mktemp)"; tr -d '\r' < "$STAGING_ENV" > "$e"
		if ( set -a; . "$e"; set +a; [ -n "${STAGING_DB_PASSWORD:-}" ] && [ -n "${STAGING_API_URL:-}" ] ); then
			rok ".env.staging normalizes + parses (required keys present)"
		else rno ".env.staging missing required keys (DB password / API URL)"; fi
		local gob; gob="$( set -a; . "$e"; set +a; printf '%s' "${STAGING_GO_BIN:-}" )"
		[ -n "$gob" ] && export PATH="$gob:$PATH"
		rm -f "$e"
	else rno "persistent .env.staging missing: $STAGING_ENV"; fi

	# HOME writable (Go cache)
	if ( t="$HOME/.rg-readiness.$$"; : > "$t" && rm -f "$t" ) 2>/dev/null; then rok "HOME set and writable ($HOME)"
	else rno "HOME not writable ($HOME) — Go cache would fail"; fi

	# required tools
	local t
	for t in git tar curl docker; do command -v "$t" >/dev/null 2>&1 && rok "found: $t" || rno "missing: $t"; done
	docker compose version >/dev/null 2>&1 && rok "found: docker compose (v2)" || rno "missing: docker compose (v2)"

	# go toolchain actually builds (stagingctl compiles from the bundle at deploy time)
	if command -v go >/dev/null 2>&1; then
		local d; d="$(mktemp -d)"; printf 'package main\nfunc main(){}\n' > "$d/m.go"
		if ( cd "$d" && go mod init rg_readiness >/dev/null 2>&1 && go build -o /dev/null . >/dev/null 2>&1 ); then
			rok "go toolchain builds (proves the Go env; stagingctl compiles at deploy)"
		else rno "go present but cannot build — cache/HOME/toolchain problem"; fi
		rm -rf "$d"
	else rno "go not found on PATH — set STAGING_GO_BIN in .env.staging"; fi

	# docker works as root (through the wrapper) — the ONLY way this identity gets docker
	docker ps >/dev/null 2>&1 && rok "docker usable as root (via wrapper)" || rno "docker not usable as root"

	# capacity for the known build
	local ak; ak="$(df -Pk "$STAGING_ROOT" 2>/dev/null | awk 'NR==2{print $4}')"
	if [ -n "$ak" ]; then
		if   [ "$ak" -lt 3145728 ]; then rno "disk critically low: $((ak/1024/1024))G free (<3G) — build will fail"
		elif [ "$ak" -lt 8388608 ]; then rw  "disk low: $((ak/1024/1024))G free (<8G recommended)"
		else rok "disk ok: $((ak/1024/1024))G free"; fi
	else rw "could not read disk free"; fi
	local ma; ma="$(awk '/MemAvailable/{print $2}' /proc/meminfo 2>/dev/null)"
	if [ -n "$ma" ]; then
		if [ "$ma" -lt 786432 ]; then rw "memory low: $((ma/1024))M available (Next build may OOM — ensure swap)"
		else rok "memory ok: $((ma/1024))M available"; fi
	else rw "could not read memory"; fi

	# fail-closed guards still hold on this box
	"$0" --check deadbeef >/dev/null 2>&1 && rno "malformed SHA NOT rejected" || rok "malformed SHA fails closed"
	"$0" --check 0000000000000000000000000000000000000000 >/dev/null 2>&1 && rok "well-formed SHA accepted" || rno "well-formed SHA wrongly rejected"
	DATABASE_URL=postgres://u@h:5432/rahalgo_prod?sslmode=disable "$0" --check 0000000000000000000000000000000000000000 >/dev/null 2>&1 \
		&& rno "production-tainted NOT refused" || rok "production-tainted fails closed"

	echo
	if [ "$rf" -eq 0 ]; then echo "READINESS: PASS"; return 0; else echo "READINESS: FAIL"; return 1; fi
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
	--readiness)
		readiness; exit $? ;;
esac

# ── real deploy ───────────────────────────────────────────────────────
validate_sha "$REQ"; refuse_production
SHA="$REQ"; SHORT="${SHA:0:8}"
[ "$(id -u)" -eq 0 ] || die "must run as root (via sudo forced command)" 15
[ -f "$STAGING_ENV" ] || die "persistent staging env missing: $STAGING_ENV" 12

# sudo resets the environment — restore PATH/HOME and the Go dir the bootstrap
# detected or installed (via $GO_CONF). STAGING_GO_BIN in .env.staging still overrides.
load_runtime

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
[ -n "${STAGING_GO_BIN:-}" ] && export PATH="$STAGING_GO_BIN:$PATH"
command -v go >/dev/null 2>&1 || die "go not found on PATH — set STAGING_GO_BIN in .env.staging (dir containing 'go')" 3
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
