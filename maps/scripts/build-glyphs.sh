#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **بناءُ الحروف — والعربيّةُ تحتاج أشكالَ عرضها**
# ══════════════════════════════════════════════════════════════════════
#
# (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ١٣ و١٤.)
#
# # ولماذا لا تكفي حروفُ العربيّة الأساسيّة
#
# **مسحُ `libmaplibre.so` في PREFLIGHT وجد `u_shapeArabic_61`** — يحوّل
# الحرفَ إلى شكلِ عرضه بحسب موضعه:
#
#	ش (U+0634) → ﺷ (U+FEB7) في الوسط
#
# **ثمّ يُبحث عن ذلك الشكل في نطاقات الحروف.** فمجموعةٌ فيها
# `U+0600–06FF` وحدَها **تُظهر العربيّةَ مربّعاتٍ والخطُّ سليم** —
# وهو عطبٌ يبدو خطأَ خطٍّ وهو خطأُ نطاق.
#
# **والنطاقاتُ مقيسةٌ لا مفترضة**: `node glyph-ranges.mjs`.
#
# # وليست هذه أداةَ التوليد
#
# **بناءُ `.pbf` يحتاج أداةً متخصّصة** (`build_pbf_glyphs` أو
# `font-maker`). **وهذا السكربتُ يستدعيها ويتحقّق من ناتجها** — ولا
# يعيد كتابتها.
#
#   الاستعمال:  sh build-glyphs.sh <مجلّد الموارد>
set -eu

OUT=${1:?الاستعمال: build-glyphs.sh <مجلّد الموارد>}
MAPS=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

# **الأداةُ تُختار بما هو متاحٌ على آلة البناء.**
if command -v build_pbf_glyphs >/dev/null 2>&1; then
  echo "  · build_pbf_glyphs"
  build_pbf_glyphs "$OUT/fonts" "$OUT/glyphs"
elif command -v font-maker >/dev/null 2>&1; then
  echo "  · font-maker"
  for f in "$OUT/fonts"/*.ttf; do
    font-maker "$f" "$OUT/glyphs"
  done
else
  echo "  ✗ لا أداةَ حروفٍ مثبَّتة على آلة البناء." >&2
  echo "    ثبّت إحداهما:  cargo install build_pbf_glyphs  |  npm i -g font-maker" >&2
  exit 3
fi

# ── التحقّق ─────────────────────────────────────────────────────────
# **ولا يمرّ ناتجٌ بلا نطاقات العربيّة** — انظر أعلى الملفّ.
#
# **والقائمةُ من `glyph-ranges.mjs` لا مدوّنةً هنا** — كانت مدوَّنةً
# فسقط منها `64512-64767` و`64768-65023` **وزِيد `65280-65535`**،
# **فكان الفاحصُ يمرّ وفي الأسماء ثقوب.** (كشفه المالكُ ٢٠٢٦-٠٨-٢١.)
RANGES=$(node "$MAPS/scripts/glyph-ranges.mjs" --list)

missing=0
stacks=0
for stack in "$OUT"/glyphs/*/; do
  [ -d "$stack" ] || continue
  stacks=$((stacks + 1))
  name=$(basename "$stack")
  for range in $RANGES; do
    if [ ! -f "$stack/$range.pbf" ]; then
      echo "  ✗ $name: النطاقُ $range مفقود" >&2
      missing=$((missing + 1))
    elif [ ! -s "$stack/$range.pbf" ]; then
      # **والفارغُ كالمفقود** — يردّ ٢٠٠ ويرسم مربّعات.
      echo "  ✗ $name: النطاقُ $range فارغ" >&2
      missing=$((missing + 1))
    fi
  done
  echo "  · $name: $(ls "$stack" | wc -l) نطاقاً"
done

[ "$stacks" -gt 0 ] || { echo "  ✗ لم تُبنَ أيُّ رصّة" >&2; exit 4; }
[ "$missing" -eq 0 ] || { echo "  ✗ $missing نطاقاً ناقصاً — العربيّةُ ستظهر مربّعات" >&2; exit 4; }
echo "  ✓ الحروف — $stacks رصّةً × $(printf '%s' "$RANGES" | wc -w) نطاقاً مطلوباً"
