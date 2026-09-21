#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  Off-box self-test of the staging deploy channel's security boundary
# ══════════════════════════════════════════════════════════════════════
#
# Proves, without a server: SHA validation, production refusal, and the
# git-bundle<->SHA cryptographic binding (via `--check` / `--verify-bundle`,
# neither touches docker/network/production). Also source-asserts the
# hardening in the wrapper + bootstrap and syntax-checks both scripts.
#
#     deploy/staging/wrapper-selftest.sh
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
W="$HERE/rahalgo-staging-deploy.sh"
BOOT="$HERE/bootstrap-staging-channel.sh"
VALID=0000000000000000000000000000000000000000
pass=0; fail=0
ok(){ echo "PASS $1"; pass=$((pass+1)); }
no(){ echo "FAIL $1"; fail=$((fail+1)); }

run(){ # <want-exit> <desc> <env...> -- <args...>
	local want="$1" desc="$2"; shift 2
	local -a envs=(); while [ "${1:-}" != "--" ]; do envs+=("$1"); shift; done; shift
	env "${envs[@]}" bash "$W" "$@" >/dev/null 2>&1
	[ "$?" -eq "$want" ] && ok "$desc" || no "$desc (want $want)"
}

echo "== staging deploy wrapper self-test =="

# ── SHA validation ────────────────────────────────────────────────────
run 10 "blank SHA rejected"        -- --check ""
run 10 "short SHA rejected"        -- --check deadbeef
run 10 "non-hex SHA rejected"      -- --check 000000000000000000000000000000000000000z
run 10 "uppercase SHA rejected"    -- --check 000000000000000000000000000000000000000A
run 0  "well-formed SHA accepted"  -- --check "$VALID"

# ── production refusal ────────────────────────────────────────────────
run 11 "prod DB name refused"      DATABASE_URL=postgres://u@h:5432/rahalgo_prod?sslmode=disable -- --check "$VALID"
run 11 "prod path refused"         COMPOSE_FILE=/srv/rahalgo/deploy/compose.yml -- --check "$VALID"
run 11 "prod api host refused"     REDIS_URL=redis://api.rahalgo.com:6379/0 -- --check "$VALID"

# ── git-bundle <-> SHA cryptographic binding ──────────────────────────
if command -v git >/dev/null 2>&1 && git -C "$HERE" rev-parse HEAD >/dev/null 2>&1; then
	TMP="$(mktemp -d)"
	A="$(git -C "$HERE" rev-parse HEAD)"
	B="$(git -C "$HERE" rev-parse HEAD~1)"
	# Must use the ref name the wrapper/workflow agree on: refs/heads/deploy-target
	git -C "$HERE" branch -f deploy-target "$A" >/dev/null 2>&1
	git -C "$HERE" bundle create "$TMP/A.bundle" deploy-target >/dev/null 2>&1
	git -C "$HERE" branch -D deploy-target >/dev/null 2>&1
	echo "not-a-bundle" > "$TMP/garbage.bundle"

	bash "$W" --verify-bundle "$A" "$TMP/A.bundle" >/dev/null 2>&1
	[ "$?" -eq 0 ] && ok "matching bundle+SHA accepted" || no "matching bundle+SHA accepted"

	bash "$W" --verify-bundle "$B" "$TMP/A.bundle" >/dev/null 2>&1
	[ "$?" -eq 14 ] && ok "mismatched bundle/commit rejected (content A + SHA B)" || no "mismatched bundle/commit rejected"

	bash "$W" --verify-bundle "$A" "$TMP/garbage.bundle" >/dev/null 2>&1
	[ "$?" -eq 13 ] && ok "corrupt bundle rejected" || no "corrupt bundle rejected"
	rm -rf "$TMP"
else
	echo "SKIP bundle tests (no git repo here)"
fi

# ── source-asserted hardening ─────────────────────────────────────────
grep -q "id -u" "$W" && grep -q "must run as root" "$W" && ok "wrapper requires root" || no "wrapper requires root"
grep -q "{{.Id}}=" "$W" && ok "prod snapshot includes container ID" || no "prod snapshot includes container ID"
grep -q 'readiness(){' "$W" && grep -q '\-\-readiness)' "$W" && ok "wrapper has --readiness mode" || no "wrapper has --readiness mode"
grep -q 'docker compose version' "$W" && grep -q 'go build -o /dev/null' "$W" && grep -q 'MemAvailable' "$W" && grep -q 'df -Pk' "$W" && ok "readiness checks tools/go/disk/memory" || no "readiness checks tools/go/disk/memory"
grep -q 'sudo -u "$DEPLOY_USER" sudo -n "$WRAPPER_DST" --readiness' "$BOOT" && ok "bootstrap gates PASS on on-box readiness" || no "bootstrap gates PASS on on-box readiness"
grep -q 'load_runtime' "$W" && grep -q 'GO_BIN_DIR' "$W" && ok "wrapper loads Go dir from config" || no "wrapper loads Go dir from config"
grep -q 'staging-[*].rahalgo.com' "$W" && grep -q 'STAGING_API_URL is a production host' "$W" && grep -q 'NEXT_PUBLIC_API_URL="http://localhost:8080"' "$W" && ok "wrapper accepts staging-api alias, refuses prod API host, feeds guard on-box target" || no "wrapper API-host handling"
grep -q 'find_go' "$BOOT" && grep -q 'install_go' "$BOOT" && grep -q '2852af0cb20a13139b3448992e69b868e50ed0f8a1e5940ee1de9e19a123b613' "$BOOT" && grep -q 'sha256sum -c' "$BOOT" && ok "bootstrap self-heals Go (detect or pinned+verified install)" || no "bootstrap self-heals Go"
GOMOD_V="$(grep -oE '^go [0-9.]+' "$HERE/../../backend/go.mod" 2>/dev/null | awk '{print $2}')"
[ -n "$GOMOD_V" ] && grep -q "REQUIRED_GO=$GOMOD_V" "$BOOT" && ok "installer Go version matches backend/go.mod ($GOMOD_V)" || no "installer Go version matches backend/go.mod"
grep -q "gpasswd -d .* docker" "$BOOT" && ok "bootstrap removes docker-group membership" || no "bootstrap removes docker-group membership"
grep -q "NOPASSWD: \$WRAPPER_DST" "$BOOT" && ok "bootstrap sudoers is wrapper-only" || no "bootstrap sudoers is wrapper-only"
grep -q 'command="sudo -n /usr/local/sbin/rahalgo-staging-deploy"' "$BOOT" && ok "bootstrap forces sudo wrapper command" || no "bootstrap forces sudo wrapper command"
grep -q "NO direct docker" "$BOOT" && ok "bootstrap validates no direct docker" || no "bootstrap validates no direct docker"

# ── one-shot installer: embeds the CURRENT wrapper, byte-for-byte ─────
ONESHOT="$HERE/bootstrap-staging-channel-one-shot.sh"
if [ -f "$ONESHOT" ]; then
	o="$(grep -n "<<'RG_WRAPPER_EOF'" "$ONESHOT" | head -1 | cut -d: -f1)"
	c="$(grep -n '^RG_WRAPPER_EOF$' "$ONESHOT" | head -1 | cut -d: -f1)"
	tmp="$(mktemp)"; sed -n "$((o+1)),$((c-1))p" "$ONESHOT" > "$tmp"
	diff -q "$tmp" "$W" >/dev/null 2>&1 && ok "one-shot embeds the current wrapper (no drift)" \
		|| no "one-shot embeds the current wrapper (no drift)"
	# embedded key is the PUBLIC key only (no PRIVATE key material)
	! grep -q 'BEGIN .*PRIVATE KEY' "$ONESHOT" && ok "one-shot embeds no private key" || no "one-shot embeds no private key"
	grep -q 'sudo -u "$DEPLOY_USER" sudo -n "$WRAPPER_DST" --readiness' "$ONESHOT" && ok "one-shot gates PASS on on-box readiness" || no "one-shot gates PASS on on-box readiness"
	grep -q 'find_go' "$ONESHOT" && grep -q '2852af0cb20a13139b3448992e69b868e50ed0f8a1e5940ee1de9e19a123b613' "$ONESHOT" && ok "one-shot includes Go self-heal (pinned+verified)" || no "one-shot includes Go self-heal"
	rm -f "$tmp"
else
	echo "SKIP one-shot checks (file absent)"
fi

# ── syntax ────────────────────────────────────────────────────────────
bash -n "$W"       && ok "wrapper syntax OK"   || no "wrapper syntax OK"
bash -n "$BOOT"    && ok "bootstrap syntax OK" || no "bootstrap syntax OK"
[ -f "$ONESHOT" ] && bash -n "$ONESHOT" && ok "one-shot syntax OK" || no "one-shot syntax OK"

echo
echo "self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
