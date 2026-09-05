#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
# **هويّةُ تشغيلٍ على جهازٍ حقيقيّ — `P-8` البندان ٢ و٣**
# ══════════════════════════════════════════════════════════════════════
#
# **ودليلٌ بلا هويّةٍ لا يُنسَب.** ثلاثةُ تشغيلاتٍ من ثلاثة بناءاتٍ تُقرأ
# سواءً، **ومن خلط بينها بنى حكماً على بناءٍ آخر.**
#
# **ولا يُكتب سطرٌ على جهازٍ قبل أن يُثبَت أنّه لا يشير إلى الإنتاج**
# (البند ٣).
#
#   ./testkit/devicerun.sh identity            هويّةُ التشغيل
#   ./testkit/devicerun.sh safety              حارسُ الإنتاج
#   ./testkit/devicerun.sh begin  <test-case>  يفتح تشغيلاً
#   ./testkit/devicerun.sh end    <result>     يختمه

set -u
OUT="${RUN_DIR:-$(cd "$(dirname "$0")/.." && pwd)/build/device-runs}"
mkdir -p "$OUT"

die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }
ok()  { printf '\033[32m✔\033[0m %s\n' "$*"; }

need_device() {
  local n
  n=$(adb devices | grep -cw 'device' || true)
  [ "$n" -ge 1 ] || die "لا جهازَ متّصل — DEVICE NOT AVAILABLE"
}

# ── هويّةُ التشغيل (البند ٢) ─────────────────────────────────────────
identity() {
  need_device
  local ser model rel fp bat net
  ser=$(adb get-serialno)
  model=$(adb shell getprop ro.product.model | tr -d '\r')
  rel=$(adb shell getprop ro.build.version.release | tr -d '\r')
  fp=$(adb shell getprop ro.build.fingerprint | tr -d '\r')
  bat=$(adb shell dumpsys battery | grep -E 'level|status' | tr -d '\r' | paste -sd' ')
  net=$(adb shell dumpsys connectivity | grep -m1 'NetworkAgentInfo' | tr -d '\r' | cut -c1-90)

  cat <<EOF
RUN_ID          = ${RUN_ID:-$(date +%Y%m%dT%H%M%S)}
SOURCE_COMMIT   = $(git -C "$(dirname "$0")/../.." rev-parse --short HEAD 2>/dev/null || echo '?')
APK_HASH        = ${APK_HASH:-'—'}
APP_VERSION     = ${APP_VERSION:-'—'}
BACKEND_COMMIT  = ${BACKEND_COMMIT:-'—'}
DEVICE_SERIAL   = $ser
DEVICE_MODEL    = $model
ANDROID_VERSION = $rel
BUILD_FINGERPRINT = $fp
BATTERY         = $bat
NETWORK         = $net
STARTED_AT      = $(date -Iseconds)
EOF
}

# ── حارسُ الإنتاج (البند ٣) ──────────────────────────────────────────
#
# **ولا يُكتب على جهازٍ يشير إلى الإنتاج.** يُقرأ العنوانُ من الحزمة
# المثبَّتة، **ومن وجد `api.rahalgo.com` توقّف.**
safety() {
  need_device
  local pkgs found=0 bad=0
  pkgs=$(adb shell pm list packages | tr -d '\r' | grep -E 'com\.rahalgo' || true)
  [ -n "$pkgs" ] || die "لا حزمةَ رحّال غو مثبَّتة"
  for p in $pkgs; do
    p=${p#package:}
    found=$((found+1))
    # **والعنوانُ يُقرأ من موارد الحزمة** — لا يُفترَض.
    local base out
    base=$(adb shell pm path "$p" | head -1 | tr -d '\r'); base=${base#package:}
    out=$(adb shell "strings $base 2>/dev/null | grep -m3 -E 'https?://[a-z0-9.-]+/api'" | tr -d '\r' || true)
    printf '  %-34s %s\n' "$p" "${out:-'—'}"
    case "$out" in
      *api.rahalgo.com*) bad=$((bad+1));;
    esac
  done
  [ "$bad" -eq 0 ] || die "PRODUCTION ENDPOINT DETECTED — لا يُكتب على هذا الجهاز"
  ok "PRODUCTION GUARD = PASS — $found حزمةً ولا واحدةَ تشير إلى الإنتاج"
}

case "${1:-}" in
  identity) identity ;;
  safety)   safety ;;
  begin)
    [ $# -ge 2 ] || die "الاستعمال: begin <test-case>"
    RUN_ID="${RUN_ID:-$(date +%Y%m%dT%H%M%S)}"
    d="$OUT/$RUN_ID"; mkdir -p "$d"
    identity > "$d/identity.txt"
    echo "TEST_CASE = $2" >> "$d/identity.txt"
    adb logcat -c
    ok "فُتح تشغيلٌ: $d"
    ;;
  end)
    [ $# -ge 2 ] || die "الاستعمال: end <result>"
    d="$OUT/${RUN_ID:?RUN_ID مطلوب}"
    adb logcat -d > "$d/logcat.txt" 2>/dev/null || true
    adb shell screencap -p /sdcard/qa.png >/dev/null 2>&1 && adb pull /sdcard/qa.png "$d/screen.png" >/dev/null 2>&1
    { echo "RESULT   = $2"; echo "ENDED_AT = $(date -Iseconds)"; } >> "$d/identity.txt"
    ok "خُتم التشغيلُ: $d"
    ;;
  *) die "الاستعمال: devicerun.sh {identity|safety|begin|end}" ;;
esac
