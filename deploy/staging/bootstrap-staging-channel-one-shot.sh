#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  ONE-SHOT self-contained installer — RahalGo staging deploy channel
# ══════════════════════════════════════════════════════════════════════
#
# GENERATED — do not edit by hand. The wrapper embedded below is byte-identical
# to deploy/staging/rahalgo-staging-deploy.sh, and deploy/staging/wrapper-selftest.sh
# fails if they ever drift apart.
#
# Needs NO git checkout, NO repo, NO manual file copying. Embeds the root-owned
# wrapper + the deploy PUBLIC key + the sudoers policy. NO secret values are
# embedded (the deploy PRIVATE key lives only in the GitHub `staging` secret;
# the persistent staging .env stays only on the box).
#
# Owner runs it once as root (typically piped from wget). Prints BOOTSTRAP RESULT.
set -euo pipefail
[ "$(id -u)" -eq 0 ] || { echo "x must run as root (use: sudo bash)"; exit 1; }

DEPLOY_USER=rahalgo-staging-deploy
STAGING_ROOT=/srv/rahalgo-staging
WRAPPER_DST=/usr/local/sbin/rahalgo-staging-deploy
SUDOERS=/etc/sudoers.d/rahalgo-staging-deploy
PUBKEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGStRuQImBvzFLw3oaLt1I/iaV0CO1YBm4GXtLszsV8Y rahalgo-staging-deploy'

echo "== RahalGo staging deploy channel — one-shot install (hardened) =="
[ -d "$STAGING_ROOT" ] || { echo "x $STAGING_ROOT does not exist — refusing"; exit 1; }
[ -f "$STAGING_ROOT/deploy/staging/.env.staging" ] || \
  echo "! warning: $STAGING_ROOT/deploy/staging/.env.staging not found yet — deploys fail until it exists"

# ── install the root-owned wrapper (embedded verbatim) ────────────────
umask 022
cat > "$WRAPPER_DST" <<'RG_WRAPPER_EOF'
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

# sudo resets the environment — restore what the build needs:
#  - PATH: secure_path drops Go's dir (needed for stagingctl guard) and others
#  - HOME: `go` needs a writable module/build cache (GOCACHE/GOMODCACHE default under HOME)
# STAGING_GO_BIN in .env.staging can override if Go lives somewhere unusual.
export PATH="/usr/local/go/bin:/snap/bin:/root/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"
export HOME="${HOME:-/root}"

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
RG_WRAPPER_EOF
chown root:root "$WRAPPER_DST"; chmod 0755 "$WRAPPER_DST"
echo "installed wrapper: $WRAPPER_DST (root:root 0755)"

# ── restricted identity: locked password, NO docker group, real shell ─
if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  useradd --system --create-home --home-dir "/home/$DEPLOY_USER" --shell /bin/bash "$DEPLOY_USER"
  echo "created user: $DEPLOY_USER"
fi
passwd -l "$DEPLOY_USER" >/dev/null 2>&1 || true
if id -nG "$DEPLOY_USER" | tr ' ' '\n' | grep -qx docker; then
  gpasswd -d "$DEPLOY_USER" docker || true; echo "removed $DEPLOY_USER from docker group"
fi

# ── sudoers: permit ONLY the wrapper; keep SSH_ORIGINAL_COMMAND ───────
cat > "$SUDOERS" <<EOF
Defaults:$DEPLOY_USER !requiretty
Defaults:$DEPLOY_USER env_keep += "SSH_ORIGINAL_COMMAND"
$DEPLOY_USER ALL=(root) NOPASSWD: $WRAPPER_DST
EOF
chmod 0440 "$SUDOERS"
visudo -cf "$SUDOERS" >/dev/null || { echo "x sudoers syntax invalid — removing"; rm -f "$SUDOERS"; exit 1; }
echo "installed sudoers: $SUDOERS (wrapper-only)"

# ── forced-command key (the ONLY thing this key can do) ───────────────
SSH_DIR="/home/$DEPLOY_USER/.ssh"
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0700 "$SSH_DIR"
RESTRICT='command="sudo -n /usr/local/sbin/rahalgo-staging-deploy",no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding,no-user-rc'
printf '%s %s\n' "$RESTRICT" "$PUBKEY" > "$SSH_DIR/authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "$SSH_DIR/authorized_keys"; chmod 600 "$SSH_DIR/authorized_keys"
echo "installed forced-command key for $DEPLOY_USER"

# ── staging-only writable paths ───────────────────────────────────────
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0755 "$STAGING_ROOT/incoming" "$STAGING_ROOT/artifacts"

# ── VALIDATE — PASS/FAIL, fail closed ─────────────────────────────────
echo; echo "== validation =="
fail=0
chk(){ if eval "$2" >/dev/null 2>&1; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }
chk "wrapper is root-owned"                 "[ \"\$(stat -c '%U' $WRAPPER_DST)\" = root ]"
chk "wrapper not group/other writable"      "[ -z \"\$(find $WRAPPER_DST -perm /022 -print)\" ]"
chk "deploy user exists"                    "id $DEPLOY_USER"
chk "deploy user password locked"           "passwd -S $DEPLOY_USER | grep -Eq ' L | LK '"
chk "deploy user NOT in docker group"       "! id -nG $DEPLOY_USER | tr ' ' '\n' | grep -qx docker"
chk "sudoers valid + wrapper-only"          "visudo -cf $SUDOERS && grep -q 'NOPASSWD: $WRAPPER_DST' $SUDOERS"
chk "sudo -l lists ONLY the wrapper"        "sudo -l -U $DEPLOY_USER | grep -q '$WRAPPER_DST' && ! sudo -l -U $DEPLOY_USER | grep -Eq '\(ALL\) *ALL|NOPASSWD: *ALL'"
chk "authorized_keys forces sudo wrapper"   "grep -q 'command=\"sudo -n /usr/local/sbin/rahalgo-staging-deploy\"' $SSH_DIR/authorized_keys"
chk "authorized_keys forbids pty/forwarding" "grep -q 'no-pty' $SSH_DIR/authorized_keys && grep -q 'no-port-forwarding' $SSH_DIR/authorized_keys"
chk "exactly one authorized key"            "[ \"\$(grep -c . $SSH_DIR/authorized_keys)\" -eq 1 ]"
chk "deploy user cannot edit the wrapper"   "! su -s /bin/sh -c \"test -w $WRAPPER_DST\" $DEPLOY_USER"
chk "deploy user has NO direct docker"      "! su -s /bin/sh -c 'docker ps' $DEPLOY_USER"
chk "wrapper rejects malformed SHA"         "! $WRAPPER_DST --check deadbeef"
chk "wrapper accepts a well-formed SHA"     "$WRAPPER_DST --check 0000000000000000000000000000000000000000"
chk "wrapper refuses production-tainted env" "! DATABASE_URL=postgres://u@h:5432/rahalgo_prod?sslmode=disable $WRAPPER_DST --check 0000000000000000000000000000000000000000"
echo
if [ "$fail" -eq 0 ]; then
  echo "BOOTSTRAP RESULT: PASS — hardened staging deploy channel installed. Routine deploys need no terminal work."
  exit 0
else
  echo "BOOTSTRAP RESULT: FAIL — see FAIL lines above. Channel NOT ready."
  exit 1
fi
