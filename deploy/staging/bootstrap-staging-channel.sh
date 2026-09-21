#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  ONE-TIME owner bootstrap — install the staging deploy channel
# ══════════════════════════════════════════════════════════════════════
#
# Run ONCE, as root, on the Hetzner box, from a checkout of this repo:
#
#     sudo STAGING_DEPLOY_PUBKEY="$(cat rahalgo-staging-deploy.pub)" \
#          deploy/staging/bootstrap-staging-channel.sh
#
# It creates a restricted identity that can do exactly ONE thing — invoke the
# root-owned staging deploy wrapper — and nothing else. No general shell, no
# password login, no port/agent/X11 forwarding, staging-only.
#
# After this, routine staging deploys need NO owner terminal work: the GitHub
# Actions `Deploy Staging` workflow drives everything through this channel.
#
# Idempotent: safe to re-run (it rewrites the wrapper + authorized key).
set -euo pipefail

[ "$(id -u)" -eq 0 ] || { echo "x must run as root" >&2; exit 1; }

DEPLOY_USER=rahalgo-staging-deploy
STAGING_ROOT=/srv/rahalgo-staging
WRAPPER_SRC="$(cd "$(dirname "$0")" && pwd)/rahalgo-staging-deploy.sh"
WRAPPER_DST=/usr/local/sbin/rahalgo-staging-deploy
PUBKEY="${STAGING_DEPLOY_PUBKEY:?set STAGING_DEPLOY_PUBKEY to the deploy identity PUBLIC key}"

echo "== RahalGo staging deploy channel bootstrap =="

# ── 1 · refuse to run against a production-only layout ─────────────────
[ -d "$STAGING_ROOT" ] || { echo "x $STAGING_ROOT does not exist — refusing (staging root required)"; exit 1; }
[ -f "$STAGING_ROOT/deploy/staging/.env.staging" ] || \
  echo "! warning: $STAGING_ROOT/deploy/staging/.env.staging not found yet — deploys will fail until it exists"

# ── 2 · install the root-owned wrapper (not editable by the deploy user)
[ -f "$WRAPPER_SRC" ] || { echo "x wrapper source not found: $WRAPPER_SRC"; exit 1; }
install -o root -g root -m 0755 "$WRAPPER_SRC" "$WRAPPER_DST"
echo "installed wrapper: $WRAPPER_DST (root:root 0755)"

# ── 3 · create the restricted deploy identity (locked password, docker access)
if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  useradd --system --create-home --home-dir "/home/$DEPLOY_USER" --shell /bin/bash "$DEPLOY_USER"
  echo "created user: $DEPLOY_USER"
fi
passwd -l "$DEPLOY_USER" >/dev/null 2>&1 || true    # no password login, ever
# docker is required to build/deploy; the forced command still limits the key to one action.
getent group docker >/dev/null 2>&1 && usermod -aG docker "$DEPLOY_USER"

# ── 4 · pin the forced-command authorized key (the ONLY thing this key can do)
SSH_DIR="/home/$DEPLOY_USER/.ssh"
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0700 "$SSH_DIR"
RESTRICT='command="/usr/local/sbin/rahalgo-staging-deploy",no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding,no-user-rc'
# Exactly one authorized key — the restricted deploy key. Overwrites any prior.
printf '%s %s\n' "$RESTRICT" "$PUBKEY" > "$SSH_DIR/authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "$SSH_DIR/authorized_keys"
chmod 600 "$SSH_DIR/authorized_keys"
echo "installed forced-command key for $DEPLOY_USER"

# ── 5 · staging-only writable paths for the deploy user ───────────────
install -d -o "$DEPLOY_USER" -g "$DEPLOY_USER" -m 0755 "$STAGING_ROOT/incoming" "$STAGING_ROOT/artifacts"

# ── 6 · VALIDATE — print a PASS/FAIL report, fail closed ──────────────
echo
echo "== validation =="
fail=0
chk(){ if eval "$2" >/dev/null 2>&1; then echo "PASS $1"; else echo "FAIL $1"; fail=1; fi; }

chk "wrapper is root-owned"            "[ \"\$(stat -c '%U' $WRAPPER_DST)\" = root ]"
chk "wrapper is not group/other writable" "[ -z \"\$(find $WRAPPER_DST -perm /022 -print)\" ]"
chk "deploy user exists"               "id $DEPLOY_USER"
chk "deploy user password is locked"   "passwd -S $DEPLOY_USER | grep -Eq ' L | LK '"
chk "authorized_keys pins forced command" "grep -q 'command=\"/usr/local/sbin/rahalgo-staging-deploy\"' $SSH_DIR/authorized_keys"
chk "authorized_keys forbids pty/forwarding" "grep -q 'no-pty' $SSH_DIR/authorized_keys && grep -q 'no-port-forwarding' $SSH_DIR/authorized_keys"
chk "exactly one authorized key"       "[ \"\$(grep -c . $SSH_DIR/authorized_keys)\" -eq 1 ]"
chk "deploy user cannot edit the wrapper" "! su -s /bin/sh -c \"test -w $WRAPPER_DST\" $DEPLOY_USER"
chk "staging root present"             "[ -d $STAGING_ROOT ]"
# Sanity: the wrapper's own input validation works (no side effects).
chk "wrapper rejects a malformed SHA"  "! $WRAPPER_DST --check deadbeef"
chk "wrapper accepts a well-formed SHA" "$WRAPPER_DST --check 0000000000000000000000000000000000000000"

echo
if [ "$fail" -eq 0 ]; then
  echo "BOOTSTRAP RESULT: PASS — staging deploy channel installed. Routine deploys need no terminal work."
  exit 0
else
  echo "BOOTSTRAP RESULT: FAIL — see FAIL lines above. Channel NOT ready."
  exit 1
fi
