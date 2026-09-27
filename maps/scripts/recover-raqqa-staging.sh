#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **إنقاذُ بناءِ خرائطِ الرقّة التجهيزيّ — إيقافٌ آمنٌ ثمّ تشغيلٌ مصلَّح**
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-٠٩-٢٧: «لا لصقَ أمرٍ طويلٍ على الصندوق المشترَك —
#  سكربتٌ مخصَّصٌ للتجهيز وحدَه، وأمرٌ واحدٌ قصيرٌ يستدعيه».)
#
# # ما يفعله — بالترتيب
#
#   ١) يتحقّق أنّ البناءَ الجارَ `build-raqqa-staging.sh` بجانبه موجود
#      وتحت `/srv/rahalgo-staging` — **ويرفض أيَّ مسار إنتاجٍ رفضاً قاطعاً**.
#   ٢) يكشف بناءَ الرقّةِ التجهيزيَّ العالقَ **بتوقيعاتٍ تجهيزيّةٍ فريدة**
#      (مخرَجُ `maps-work/region-raqqa.pmtiles` أو مسارُ سكربتِ البناء
#      تحت التجهيز)، ويوقف **مجموعةَ عمليّاته كاملةً** (الغلافُ + جافا
#      + أيُّ ابنٍ) عبر مُعرّف المجموعة — **لا `pkill java` عريض**، ولا
#      لمسَ مجموعتِنا نحن ولا أيَّ جافا لا يخصّ هذا البناء.
#   ٣) ينظّف **الآثارَ الجزئيّةَ التجهيزيّةَ وحدَها** (المخرَجُ الناقص
#      و`tmp` وملفّاتُ `.part` وملفُّ الرقم) — **ويُبقي `syria.osm.pbf`
#      و`planetiler.jar` المخبّأين** لإعادةٍ سريعة.
#   ٤) يشغّل البناءَ المصلَّح (بقفله ومُهله الداخليّة)، ويعرض ناتجَه حيّاً
#      **وفي سجلٍّ مطلق** معاً، ثمّ يعلن **SUCCESS/FAILURE** بحسب خروجه.
#
# # ما لا يفعله
#
#   • لا يكتب ولا يقرأ تحت `/srv/rahalgo` (الإنتاج) — حارسٌ يُسقطه.
#   • لا يشغّل `maps-sync.sh` (تنسخ الإنتاجَ فوقَ التجهيز فتمحو عملَنا).
#   • لا يعيد تشغيلَ أيّ خدمة. **الإنتاجُ لا يُمَسّ.**
#
#   الاستعمال (أمرٌ واحدٌ قصيرٌ مطلق):
#     sh /srv/rahalgo-staging/incoming/<SHA>/maps/scripts/recover-raqqa-staging.sh
set -eu

STAGE_ROOT=/srv/rahalgo-staging
PROD_ROOT=/srv/rahalgo
WORK="$STAGE_ROOT/maps-work"
LOG="$STAGE_ROOT/raqqa-staging.log"
OUT="$WORK/region-raqqa.pmtiles"

# ── مسارُ البناءِ المصلَّح: جارُنا في المجلّد نفسِه (مستقلٌّ عن الـSHA) ──
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
BUILD="$SCRIPT_DIR/build-raqqa-staging.sh"

log() { echo "[$(date -u +%FT%TZ)] recover: $*"; }

# ── حارسٌ صارم: لا شيءَ تحت الإنتاج، وكلُّ شيءٍ تحت التجهيز ───────────
case "$BUILD" in
  "$PROD_ROOT"|"$PROD_ROOT"/*) echo "✗ مسارُ بناءٍ إنتاجيّ — رفضٌ قاطع: $BUILD" >&2; exit 2 ;;
esac
case "$WORK" in
  "$PROD_ROOT"|"$PROD_ROOT"/*) echo "✗ عملُ بناءٍ إنتاجيّ — رفضٌ قاطع" >&2; exit 2 ;;
esac
case "$BUILD" in
  "$STAGE_ROOT"/*) : ;;
  *) echo "✗ البناءُ خارجَ شجرة التجهيز — رفض: $BUILD" >&2; exit 2 ;;
esac
[ -d "$STAGE_ROOT" ] || { echo "✗ لا جذرَ تجهيزٍ في $STAGE_ROOT" >&2; exit 2; }
[ -f "$BUILD" ] || { echo "✗ لا سكربتَ بناءٍ في $BUILD" >&2; exit 2; }

log "════ RECOVERY START ════  (build: $BUILD)"

# ── ١) كشفُ مجموعةِ عمليّاتِ البناء العالق بتوقيعاتٍ تجهيزيّةٍ فريدة ──
# **لا نطابق `java` مجرَّداً**؛ بل مخرَجَ التجهيز الفريدَ أو مسارَ سكربتِ
# البناء تحت التجهيز، ثمّ نأخذ مُعرّفَ مجموعةِ كلِّ مطابَقةٍ لنُنهيَ الشجرةَ
# كاملةً — فلو بقي الغلافُ (الصدفة) حيّاً دون جافا أُنهيَ هو أيضاً.
MYPGID=$(ps -o pgid= -p "$$" 2>/dev/null | tr -d ' ')
targets=""
add_pg() {
  pg=$(ps -o pgid= -p "$1" 2>/dev/null | tr -d ' ')
  [ -n "$pg" ] || return 0
  [ "$pg" = "$MYPGID" ] && return 0        # لا نمسّ مجموعتَنا
  [ "$pg" = "0" ] || [ "$pg" = "1" ] && return 0
  case " $targets " in *" $pg "*) ;; *) targets="$targets $pg";; esac
}
for pid in $(pgrep -f "$WORK/region-raqqa\.pmtiles" 2>/dev/null || true); do add_pg "$pid"; done
for pid in $(pgrep -f "$STAGE_ROOT/.*/maps/scripts/build-raqqa-staging\.sh" 2>/dev/null || true); do add_pg "$pid"; done

if [ -n "$targets" ]; then
  for pg in $targets; do
    log "PROGRESS: إيقافُ مجموعةِ بناءٍ تجهيزيّةٍ عالقة pgid=$pg (TERM)"
    kill -TERM -"$pg" 2>/dev/null || true
  done
  # مهلةُ خروجٍ لطيفٍ حتّى ١٥ث، ثمّ إنهاءٌ قسريٌّ لما بقي
  i=0
  while [ "$i" -lt 15 ]; do
    alive=0
    for pg in $targets; do kill -0 -"$pg" 2>/dev/null && alive=1; done
    [ "$alive" -eq 0 ] && break
    sleep 1; i=$((i+1))
  done
  for pg in $targets; do
    if kill -0 -"$pg" 2>/dev/null; then
      log "PROGRESS: إنهاءٌ قسريٌّ pgid=$pg (KILL)"
      kill -KILL -"$pg" 2>/dev/null || true
    fi
  done
else
  log "PROGRESS: لا بناءَ رقّةٍ تجهيزيٍّ عالقٌ جارٍ — متابعةٌ لتشغيلٍ نظيف"
fi

# ── ٢) تنظيفُ الآثارِ الجزئيّةِ التجهيزيّةِ وحدَها — والمخبّآتُ باقية ──
rm -rf "$WORK/tmp" 2>/dev/null || true
rm -f "$OUT" "$OUT.part" 2>/dev/null || true
rm -f "$WORK/syria.osm.pbf.part" "$WORK/planetiler.jar.part" 2>/dev/null || true
rm -f "$WORK/raqqa-build.pid" 2>/dev/null || true
log "PROGRESS: نُظّفت الآثارُ الجزئيّةُ (أُبقيَ syria.osm.pbf و planetiler.jar)"

# ── ٣) تشغيلُ البناءِ المصلَّح — حيٌّ على الشاشةِ وفي السجلِّ المطلقِ معاً ──
log "PROGRESS: تشغيلُ البناءِ المصلَّح — الناتجُ حيٌّ وفي $LOG"
echo "[$(date -u +%FT%TZ)] recover: ==== launch build-raqqa-staging.sh ====" >> "$LOG"
RCF="$WORK/.recover.rc"
rm -f "$RCF" 2>/dev/null || true
# **التقاطُ خروجِ البناء الحقيقيّ عبر ملفٍّ** (لا PIPESTATUS في POSIX sh)
{ sh "$BUILD"; echo $? > "$RCF"; } 2>&1 | tee -a "$LOG"
rc=$(cat "$RCF" 2>/dev/null || echo 1)
rm -f "$RCF" 2>/dev/null || true

if [ "$rc" = "0" ]; then
  log "════ RECOVERY SUCCESS ════  الرقّةُ التجهيزيّةُ بُنيت ونُشرت (سجلّ: $LOG)"
  exit 0
else
  log "════ RECOVERY FAILURE ════  البناءُ خرج rc=$rc — راجع $LOG (نُظّفت الآثارُ الجزئيّة)"
  exit "$rc"
fi
