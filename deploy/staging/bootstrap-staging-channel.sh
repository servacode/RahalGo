#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  ONE-TIME owner bootstrap — install the hardened staging deploy channel
# ══════════════════════════════════════════════════════════════════════
#
# Run ONCE, as root, on the Hetzner box, from a checkout of this repo:
#
#     sudo deploy/staging/bootstrap-staging-channel.sh
#
# The public deploy key is read from deploy/staging/rahalgo-staging-deploy.pub
# (committed; public keys are safe to commit). The matching PRIVATE key is stored
# only as the GitHub `staging` environment secret STAGING_DEPLOY_KEY.
#
# It creates an identity that can do EXACTLY ONE thing — `sudo` the root-owned
# staging deploy wrapper — and nothing else:
#   - password locked (no password login)
#   - forced-command SSH key (no shell, no pty/port/agent/X11 forwarding)
#   - NOT in the docker group (no host-root equivalence)
#   - a sudoers rule that permits ONLY the wrapper
#
# Idempotent. Prints a PASS/FAIL report and fails closed.
set -euo pipefail

[ "$(id -u)" -eq 0 ] || { echo "x must run as root" >&2; exit 1; }

DEPLOY_USER=rahalgo-staging-deploy
STAGING_ROOT=/srv/rahalgo-staging
HERE="$(cd "$(dirname "$0")" && pwd)"
WRAPPER_SRC="$HERE/rahalgo-staging-deploy.sh"
WRAPPER_DST=/usr/local/sbin/rahalgo-staging-deploy
SUDOERS=/etc/sudoers.d/rahalgo-staging-deploy
PUBKEY="${STAGING_DEPLOY_PUBKEY:-$(cat "$HERE/rahalgo-staging-deploy.pub" 2>/dev/null || true)}"
[ -n "$PUBKEY" ] || { echo "x no public key: set STAGING_DEPLOY_PUBKEY or commit rahalgo-staging-deploy.pub" >&2; exit 1; }

echo "== RahalGo staging deploy channel bootstrap (hardened) =="
[ -d "$STAGING_ROOT" ] || { echo "x $STAGING_ROOT does not exist — refusing"; exit 1; }
[ -f "$WRAPPER_SRC" ]  || { echo "x wrapper source not found: $WRAPPER_SRC"; exit 1; }
[ -f "$STAGING_ROOT/deploy/staging/.env.staging" ] || \
  echo "! warning: $STAGING_ROOT/deploy/staging/.env.staging not found yet — deploys fail until it exists"

# ── 1 · install the root-owned wrapper (not editable by the deploy user)
install -o root -g root -m 0755 "$WRAPPER_SRC" "$WRAPPER_DST"
echo "installed wrapper: $WRAPPER_DST (root:root 0755)"

# ── 2 · restricted identity: locked password, NO docker group, real shell
#        (shell is only ever reached through the forced command)
if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  useradd --system --create-home --home-dir "/home/$DEPLOY_USER" --shell /bin/bash "$DEPLOY_USER"
  echo "created user: $DEPLOY_USER"
fi
passwd -l "$DEPLOY_USER" >/dev/null 2>&1 || true
# Explicitly ensure NOT in the docker group (docker group == host root).
if id -nG "$DEPLOY_USER" | tr ' ' '\n' | grep -qx docker; then
  gpasswd -d "$DEPLOY_USER" docker || true
  echo "removed $DEPLOY_USER from docker group"
fi

# ── 3 · sudoers: permit ONLY the wrapper; keep SSH_ORIGINAL_COMMAND ────
cat > "$SUDOERS" <<EOF
Defaults:$DEPLOY_USER !requiretty
Defaults:$DEPLOY_USER env_keep += "SSH_ORIGINAL_COMMAND"
$DEPLOY_USER ALL=(root) NOPASSWD: $WRAPPER_DST
EOF
chmod 0440 "$SUDOERS"
visudo -cf "$SUDOERS" >/dev/null || { echo "x sudoers syntax invalid — removing"; rm -f "$SUDOERS"; exit 1; }
echo "installed sudoers: $SUDOERS (wrapper-only)"

# ── 4 · forced-command key — the ONLY thing this key can do ────────────
SSH_DIR="/home/$DEPLOY_USER/.ssh"
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0700 "$SSH_DIR"
RESTRICT='command="sudo -n /usr/local/sbin/rahalgo-staging-deploy",no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding,no-user-rc'
printf '%s %s\n' "$RESTRICT" "$PUBKEY" > "$SSH_DIR/authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "$SSH_DIR/authorized_keys"
chmod 600 "$SSH_DIR/authorized_keys"
echo "installed forced-command key for $DEPLOY_USER"

# ── 5 · staging-only writable paths for the deploy user ───────────────
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0755 "$STAGING_ROOT/incoming" "$STAGING_ROOT/artifacts"

# ── 6 · VALIDATE — PASS/FAIL, fail closed ─────────────────────────────
echo; echo "== validation =="
fail=0
chk(){ if eval "$2" >/dev/null 2>&1; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }

chk "wrapper is root-owned"                 "[ \"\$(stat -c '%U' $WRAPPER_DST)\" = root ]"
chk "wrapper not group/other writable"      "[ -z \"\$(find $WRAPPER_DST -perm /022 -print)\" ]"
chk "deploy user exists"                    "id $DEPLOY_USER"
chk "deploy user password locked"           "passwd -S $DEPLOY_USER | grep -Eq ' L | LK '"
chk "deploy user NOT in docker group"       "! id -nG $DEPLOY_USER | tr ' ' '\n' | grep -qx docker"
chk "sudoers valid + wrapper-only"          "visudo -cf $SUDOERS && grep -q 'NOPASSWD: $WRAPPER_DST' $SUDOERS"
chk "sudo -l lists ONLY the wrapper"        "sudo -l -U $DEPLOY_USER | grep -q '$WRAPPER_DST' && ! sudo -l -U $DEPLOY_USER | grep -Eq '\\(ALL\\) *ALL|NOPASSWD: *ALL'"
chk "authorized_keys forces sudo wrapper"   "grep -q 'command=\"sudo -n /usr/local/sbin/rahalgo-staging-deploy\"' $SSH_DIR/authorized_keys"
chk "authorized_keys forbids pty/forwarding" "grep -q 'no-pty' $SSH_DIR/authorized_keys && grep -q 'no-port-forwarding' $SSH_DIR/authorized_keys"
chk "exactly one authorized key"            "[ \"\$(grep -c . $SSH_DIR/authorized_keys)\" -eq 1 ]"
chk "deploy user cannot edit the wrapper"   "! su -s /bin/sh -c \"test -w $WRAPPER_DST\" $DEPLOY_USER"
chk "deploy user has NO direct docker"      "! su -s /bin/sh -c 'docker ps' $DEPLOY_USER"
chk "wrapper rejects malformed SHA"         "! $WRAPPER_DST --check deadbeef"
chk "wrapper accepts a well-formed SHA"     "$WRAPPER_DST --check 0000000000000000000000000000000000000000"
chk "wrapper refuses production-tainted env" "! DATABASE_URL=postgres://u@h:5432/rahalgo_prod?sslmode=disable $WRAPPER_DST --check 0000000000000000000000000000000000000000"

echo
if [ "$fail" -ne 0 ]; then
  echo "BOOTSTRAP RESULT: FAIL — install validation failed (see FAIL lines above). Channel NOT ready."
  exit 1
fi

# ── on-box readiness through the REAL deploy path (no staging mutation) ─
echo "== on-box readiness (deploy-user -> sudo -> wrapper as root) =="
if sudo -u "$DEPLOY_USER" sudo -n "$WRAPPER_DST" --readiness; then
  echo
  echo "BOOTSTRAP RESULT: PASS — channel installed AND on-box runtime is deploy-ready."
  exit 0
else
  echo
  echo "BOOTSTRAP RESULT: FAIL — on-box readiness failed (see above). Channel NOT deploy-ready."
  exit 1
fi
