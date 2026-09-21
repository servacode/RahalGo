#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  rahalgo-staging-deploy — the ONLY thing the staging deploy identity runs
# ══════════════════════════════════════════════════════════════════════
#
# Installed by bootstrap to:  /usr/local/sbin/rahalgo-staging-deploy
# Root-owned, mode 0755, NOT writable by the deploy identity.
# The deploy identity's authorized_keys pins:
#     command="/usr/local/sbin/rahalgo-staging-deploy",no-pty,no-port-forwarding,
#     no-agent-forwarding,no-X11-forwarding <key>
# so that key can do nothing but invoke this wrapper — no general shell.
#
# Contract:
#   - argument (or $SSH_ORIGINAL_COMMAND) = exactly ONE 40-char lowercase git SHA
#   - a `git archive` .tar.gz of that exact commit arrives on STDIN
#   - everything happens under /srv/rahalgo-staging ONLY
#   - production is NEVER built, migrated, restarted, or configured
#   - fail closed on any ambiguity
#
# Exit codes:
#   10 malformed/blank SHA        11 production-tainted target
#   12 missing persistent env     13 bad/absent source archive
#   3  envguard (stagingctl) refused the environment
#   5  post-deploy identity/env/commit mismatch
#   6  production container identity changed (isolation breach)
#   0  success (staging serves the exact SHA)
#
# `--check <sha>` runs ONLY the input validation (no docker, no network) so the
# channel can be self-tested off-box. See deploy/staging/wrapper-selftest.sh.
set -euo pipefail

# ── staging-only constants — NEVER production ─────────────────────────
STAGING_ROOT=/srv/rahalgo-staging
STAGING_ENV="$STAGING_ROOT/deploy/staging/.env.staging"
IDENTITY_URL="http://localhost:8080/api/v1/public/identity"
HEALTH_URL="http://localhost:8080/api/v1/healthz"
INCOMING="$STAGING_ROOT/incoming"
ARTIFACT_DIR="$STAGING_ROOT/artifacts"
PROD_PROJECT=rahalgo            # read-only, to PROVE it was not touched
STAGING_PROJECT=rahalgo-staging

log(){ printf '%s\n' "$*" >&2; }
die(){ printf 'x %s\n' "$*" >&2; exit "${2:-1}"; }

# ── 1 · resolve the requested SHA (arg, or forced-command SSH_ORIGINAL_COMMAND)
REQ="${1:-${SSH_ORIGINAL_COMMAND:-}}"
REQ="${REQ%% *}"                # first whitespace field only

CHECK=0
if [ "${REQ:-}" = "--check" ]; then CHECK=1; REQ="${2:-}"; fi

# ── 2 · strict SHA validation: exactly 40 lowercase hex ───────────────
case "$REQ" in
	"")           die "no SHA provided" 10 ;;
	*[!0-9a-f]*)  die "SHA rejected — not 40 lowercase hex: '$REQ'" 10 ;;
esac
[ "${#REQ}" -eq 40 ] || die "SHA rejected — length ${#REQ} != 40" 10
SHA="$REQ"
SHORT="${SHA:0:8}"

# ── 3 · staging-only / production-refusal guard ───────────────────────
[ "$STAGING_ROOT" = /srv/rahalgo-staging ] || die "staging root misconfigured — fail closed" 11
# Refuse if the caller tries to steer us at production via the environment.
for v in DATABASE_URL REDIS_URL COMPOSE_FILE ENV_FILE STAGING_ROOT; do
	case "${!v:-}" in
		*rahalgo_prod*|*/srv/rahalgo/*|*api.rahalgo.com*|*:5432/rahalgo?sslmode*)
			die "refusing production-tainted \$$v" 11 ;;
	esac
done

if [ "$CHECK" = 1 ]; then
	if [ -f "$STAGING_ENV" ]; then log "check: persistent staging env present"; else
		log "check(note): persistent staging env not on this host ($STAGING_ENV) — expected only on the box"; fi
	log "OK check passed: SHA=$SHORT accepted, staging-only, production refused"
	exit 0
fi

# ── 4 · from here we are on the box: require the persistent staging env
[ -f "$STAGING_ENV" ] || die "persistent staging env missing: $STAGING_ENV" 12

# ── 5 · snapshot PRODUCTION identities BEFORE (read-only; never mutate) ─
prod_snapshot(){ docker ps -a --filter "label=com.docker.compose.project=$PROD_PROJECT" \
	--format '{{.Names}}={{.Image}}={{.State}}' 2>/dev/null | sort; }
PROD_BEFORE="$(prod_snapshot || true)"
log "-- production containers before: $(printf '%s' "$PROD_BEFORE" | tr '\n' ' ')"

# ── 6 · receive + extract the git archive (stdin) into a fresh staging dir
mkdir -p "$INCOMING" "$ARTIFACT_DIR"
SRC="$INCOMING/$SHA"
rm -rf "$SRC"; mkdir -p "$SRC"
tar -xzf - -C "$SRC" 2>/dev/null || die "could not extract source archive from stdin (expect git tar.gz)" 13
# The archive must actually be this repo.
[ -f "$SRC/deploy/build-artifact.sh" ] && \
[ -f "$SRC/deploy/staging/compose.staging.yml" ] && \
[ -f "$SRC/deploy/promote.sh" ] || die "archive is not the RahalGo repo — refused" 13

# ── 7 · staging environment for build + guard (mirrors deploy/staging/deploy.sh)
set -a; . "$STAGING_ENV"; set +a
export APP_ENV=staging RAHALGO_STAGING=1
export DATABASE_URL="postgres://rahalgo:${STAGING_DB_PASSWORD}@localhost:5534/rahalgo_staging?sslmode=disable"
export REDIS_URL="redis://localhost:6580/0"
export NEXT_PUBLIC_API_URL="${STAGING_API_URL}"
# Final production-refusal, now that the env is loaded.
case "$DATABASE_URL" in *5534/rahalgo_staging*) ;; *) die "DATABASE_URL is not the staging DB — fail closed" 11 ;; esac

# ── 8 · envguard: the app itself refuses a non-staging environment ────
( cd "$SRC/backend" && go run ./cmd/stagingctl guard ) || die "stagingctl guard refused the environment — fail closed" 3

# ── 9 · the target must already say it is staging (before any mutation) ─
STRICT=0 "$SRC/deploy/preflight-env.sh" "$IDENTITY_URL" staging || die "preflight: target is not staging — fail closed" 11

# ── 10 · build the exact SHA in ARCHIVE mode (identity injected + verified)
export SOURCE_COMMIT="$SHA" SRC_ROOT="$SRC"
ART="$("$SRC/deploy/build-artifact.sh")"
printf '%s\n' "$ART" | sed 's/^/   build: /' >&2
API_TAG="$(printf '%s\n' "$ART" | sed -n 's/^API_IMAGE_TAG=//p')"
API_ID="$(printf '%s\n'  "$ART" | sed -n 's/^API_IMAGE_ID=//p')"
WEB_TAG="$(printf '%s\n' "$ART" | sed -n 's/^WEB_IMAGE_TAG=//p')"
[ -n "$API_TAG" ] && [ -n "$WEB_TAG" ] || die "build produced no image tags — fail closed" 4

# ── 11 · bring up the staging stack (non-api), then promote api by artifact
export RAHALGO_API_IMAGE="$API_TAG" RAHALGO_WEB_IMAGE="$WEB_TAG"
docker compose -p "$STAGING_PROJECT" -f "$SRC/deploy/staging/compose.staging.yml" --env-file "$STAGING_ENV" \
	up -d --no-build --no-deps caddy web postgres redis
TARGET_ENV=staging "$SRC/deploy/promote.sh" \
	"$SRC/deploy/staging/compose.staging.yml" "$STAGING_ENV" "$API_TAG" "$IDENTITY_URL" "$API_ID"

# ── 12 · verify the LIVE staging runtime is exactly what we built ──────
for _ in $(seq 1 60); do curl -fsS "$HEALTH_URL" >/dev/null 2>&1 && break; sleep 2; done
curl -fsS "$HEALTH_URL" >/dev/null 2>&1 || die "staging /healthz did not come up" 5
ID="$(curl -fsS "$IDENTITY_URL")" || die "staging identity did not answer" 5
printf 'IDENTITY=%s\n' "$ID" >&2
case "$ID" in *'"environment":"staging"'*) ;; *) die "live env is not staging — fail closed" 5 ;; esac
case "$ID" in *"$SHA"*) ;; *) die "live source_commit != requested SHA — fail closed" 5 ;; esac

# ── 13 · PROVE production was not replaced ─────────────────────────────
PROD_AFTER="$(prod_snapshot || true)"
if [ "$PROD_BEFORE" != "$PROD_AFTER" ]; then
	log "-- production before: $PROD_BEFORE"
	log "-- production after : $PROD_AFTER"
	die "PRODUCTION CONTAINER IDENTITY CHANGED — isolation breach" 6
fi

MIG="$(printf '%s' "$ID" | grep -o '"migration_version":"[^"]*"' | cut -d'"' -f4)"
log "OK staging now serves $SHORT · migration=$MIG · production unchanged"
printf 'DEPLOYED_SHA=%s\nMIGRATION_VERSION=%s\nPRODUCTION_MUTATIONS=0\n' "$SHA" "$MIG"
