#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **مبانٍ لسوريا كلِّها — من بيانات مايكروسوفت المفتوحة**
# ══════════════════════════════════════════════════════════════════════
#
# (طلبُ المالك ٢٠٢٦-٠٨-٢٤: «أريد الخريطة تصبح مطابقة تقريباً إلى غوغل
#  ماب».)
#
# # ولماذا مصدرٌ ثانٍ
#
# **قِيس ٢٠٢٦-٠٨-٢٤**: بلاطةٌ عند التقريب ١٦ فوق مركز الرقّة فيها
# **مبنًى واحد**. وتغطيةُ OpenStreetMap للأبنية في الرقّة قريبةٌ من
# الصفر — **وأكبرُ فارقٍ بين خريطتنا وخرائط غوغل هو المباني.**
#
# **ومايكروسوفت أطلقت ١٫٤ مليار مبنًى** مستخرَجةً من صور الأقمار
# برخصةٍ متساهلة. **وسوريا مغطّاةٌ بسبعةٍ وخمسين ملفّاً** (٣٧٧ م.ب،
# محدَّثةً ٢٠٢٦-٠٨-١٣).
#
# # ولا يُنسخ من غوغل حرفٌ واحد
#
# **شروطُ غوغل تمنع استعمالَ محتواه لبناء خريطةٍ منافسة**، ومجتمعُ
# OpenStreetMap يحذف ما مصدرُه غوغل ويحظر صاحبَه. **وهذا مصدرٌ آخرُ
# مفتوحٌ صراحةً.**
#
# # والقرصُ ضيّق
#
# **ثلاثةَ عشرَ غيغا حرّةً على السيرفر** (قِيس ٢٠٢٦-٠٨-٢٤). فيُعالَج
# ملفٌّ ملفّاً ويُحذف مضغوطُه فور فكّه — **ولا تجتمع النسختان.**
#
#   الاستعمال:  WORK=/srv/maps sh fetch-buildings.sh
set -eu

WORK=${WORK:-/srv/maps}
INDEX_URL=${INDEX_URL:-https://bfppub.blob.core.windows.net/\$web/2026-08-13/dataset-links.csv}
REGION=${REGION:-Syria}
OUT="$WORK/buildings"

mkdir -p "$OUT"
cd "$OUT"

# ── 1 · the index ────────────────────────────────────────────────────
if [ ! -f dataset-links.csv ]; then
  echo "> index"
  curl -sL --fail -o dataset-links.csv "$INDEX_URL"
fi

TOTAL=$(grep -c "^$REGION," dataset-links.csv || echo 0)
[ "$TOTAL" -gt 0 ] || { echo "  x no files for $REGION" >&2; exit 3; }
echo "> $REGION: $TOTAL files"

# ── 2 · fetch and expand, one at a time ──────────────────────────────
# **ويُستأنَف**: من انقطع تنزيلُه يُعاد ولا يُعاد ما تمّ.
: > "$OUT/.progress"
N=0
grep "^$REGION," dataset-links.csv | while IFS=, read -r region qk url size date; do
  N=$((N + 1))
  DEST="$OUT/qk-$qk.geojsonl"
  if [ -s "$DEST" ]; then
    echo "  = $qk (kept)"
    continue
  fi
  echo "  > $N/$TOTAL  $qk  $size"
  curl -sL --fail -o "$OUT/part.gz" "$url" || { echo "    x download failed" >&2; continue; }
  # ══════════════════════════════════════════════════════════════════
  # **واللاحقةُ `.csv.gz` تكذب — المحتوى GeoJSON سطراً سطراً**
  # ══════════════════════════════════════════════════════════════════
  #
  # **قِيس ٢٠٢٦-٠٨-٢٤** بتنزيل ملفٍّ وفكِّه: كلُّ سطرٍ كائنٌ كاملٌ
  # بذاته يبدأ بـ`{"type": "Feature"`. **ولا فواصلَ ولا أعمدة.**
  #
  # **وأوّلُ تشغيلٍ قرأها جداولَ CSV فأخرج صفراً من المباني** — سبعةٌ
  # وخمسون ملفّاً فارغةً **بلا خطأٍ واحدٍ في السجلّ**: القارئُ لم يجد
  # عموداً ثانياً فسكت. **وصفرٌ صامتٌ أخبثُ من عطبٍ يصرخ.**
  #
  # **فيُمرَّر ما يبدأ بقوس** — وسطرٌ لا يبدأ به ليس معلماً.
  gunzip -c "$OUT/part.gz" | sed -n '/^{/p' > "$DEST"
  [ -s "$DEST" ] || { echo "    x empty after convert" >&2; rm -f "$DEST"; }
  rm -f "$OUT/part.gz"
done

# ── 3 · the tally ────────────────────────────────────────────────────
FILES=$(ls "$OUT"/qk-*.geojsonl 2>/dev/null | wc -l)
LINES=$(cat "$OUT"/qk-*.geojsonl 2>/dev/null | wc -l)
BYTES=$(du -sh "$OUT" | cut -f1)
echo "> done: $FILES files · $LINES buildings · $BYTES"
