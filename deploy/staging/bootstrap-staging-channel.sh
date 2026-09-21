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

# ── 5.5 · self-heal Go: detect a suitable toolchain, else install pinned
REQUIRED_GO=1.25.0
GO_CONF=/etc/rahalgo-staging-deploy.conf
GO_SHA256=2852af0cb20a13139b3448992e69b868e50ed0f8a1e5940ee1de9e19a123b613
GO_TGZ="go${REQUIRED_GO}.linux-amd64.tar.gz"
go_ok(){ local v; v="$("$1" version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1 | sed 's/^go//')"; [ -n "$v" ] && [ "$(printf '%s\n%s\n' "$REQUIRED_GO" "$v" | sort -V | head -1)" = "$REQUIRED_GO" ]; }
find_go(){ local c; for c in /usr/local/go/bin/go /usr/bin/go /snap/bin/go /opt/go/bin/go /opt/rahalgo-go/bin/go /usr/lib/go/bin/go /root/sdk/go*/bin/go /home/*/sdk/go*/bin/go /root/go/bin/go; do [ -x "$c" ] || continue; go_ok "$c" && { dirname "$c"; return 0; }; done; return 1; }
install_go(){ local dest=/opt/rahalgo-go tmp; command -v curl >/dev/null 2>&1 || { echo "x curl required to install Go" >&2; return 1; }; tmp="$(mktemp -d)"; echo "downloading pinned Go $REQUIRED_GO (integrity-verified) ..." >&2; curl -fsSL "https://go.dev/dl/$GO_TGZ" -o "$tmp/$GO_TGZ" || { echo "x Go download failed" >&2; rm -rf "$tmp"; return 1; }; echo "$GO_SHA256  $tmp/$GO_TGZ" | sha256sum -c - >/dev/null 2>&1 || { echo "x Go checksum FAILED — refusing" >&2; rm -rf "$tmp"; return 1; }; rm -rf "$dest"; tar -C "$tmp" -xzf "$tmp/$GO_TGZ" || { echo "x Go extract failed" >&2; rm -rf "$tmp"; return 1; }; mv "$tmp/go" "$dest"; rm -rf "$tmp"; go_ok "$dest/bin/go" || { echo "x installed Go failed version check" >&2; return 1; }; printf '%s' "$dest/bin"; }
echo "== ensuring Go >= $REQUIRED_GO (never overwrites an existing Go) =="
GO_DIR="$(find_go || true)"
if [ -n "$GO_DIR" ]; then
  echo "detected suitable Go: $GO_DIR ($("$GO_DIR/go" version 2>/dev/null))"
else
  echo "no suitable Go found — installing pinned Go $REQUIRED_GO to /opt/rahalgo-go"
  GO_DIR="$(install_go)" || { echo "BOOTSTRAP RESULT: FAIL — could not provide Go $REQUIRED_GO"; exit 1; }
  echo "installed Go: $GO_DIR"
fi
printf 'GO_BIN_DIR=%s\n' "$GO_DIR" > "$GO_CONF"; chmod 0644 "$GO_CONF"
echo "recorded Go dir in $GO_CONF"

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
