#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  Off-box self-test of the staging deploy wrapper's SECURITY boundary
# ══════════════════════════════════════════════════════════════════════
#
# Exercises the input-validation / production-refusal layer via `--check`
# (no docker, no ssh, no network) so the channel can be proven before use.
# Runtime paths (missing/wrong archive, identity/health mismatch) fail closed
# in the wrapper and are additionally verified by the GitHub workflow's
# independent identity check.
#
#     deploy/staging/wrapper-selftest.sh
set -uo pipefail

W="$(cd "$(dirname "$0")" && pwd)/rahalgo-staging-deploy.sh"
VALID=0000000000000000000000000000000000000000
pass=0; fail=0

# run <expected-exit> <description> -- <env assignments...> -- <args...>
run() {
	local want="$1" desc="$2"; shift 2
	local -a envs=() args=()
	while [ "${1:-}" != "--" ]; do envs+=("$1"); shift; done; shift
	args=("$@")
	env "${envs[@]}" bash "$W" "${args[@]}" >/dev/null 2>&1
	local got=$?
	if [ "$got" -eq "$want" ]; then echo "PASS ($got) $desc"; pass=$((pass+1))
	else echo "FAIL (got $got, want $want) $desc"; fail=$((fail+1)); fi
}

echo "== staging deploy wrapper self-test =="

# ── malformed SHA -> 10 ───────────────────────────────────────────────
run 10 "blank SHA rejected"                 -- --check ""
run 10 "short SHA rejected"                 -- --check deadbeef
run 10 "non-hex SHA rejected"               -- --check 000000000000000000000000000000000000000z
run 10 "uppercase SHA rejected"             -- --check 000000000000000000000000000000000000000A
run 10 "41-char SHA rejected"               -- --check 00000000000000000000000000000000000000000

# ── well-formed SHA, clean env -> 0 ───────────────────────────────────
run 0  "well-formed SHA accepted"           -- --check "$VALID"

# ── production-tainted env -> 11 ──────────────────────────────────────
run 11 "production DB name refused"         DATABASE_URL=postgres://u@h:5432/rahalgo_prod?sslmode=disable -- --check "$VALID"
run 11 "production path refused"            COMPOSE_FILE=/srv/rahalgo/deploy/compose.yml -- --check "$VALID"
run 11 "production api host refused"        REDIS_URL=redis://api.rahalgo.com:6379/0 -- --check "$VALID"

echo
echo "self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
