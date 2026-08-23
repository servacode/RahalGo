#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **الأيقونات — أربعٌ لا أربعون**
# ══════════════════════════════════════════════════════════════════════
#
# (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢٠.)
#
# **ولا مطاعمَ ولا متاجرَ ولا أسواق** — أمرُ المالك نصّاً.
#
# **والسببُ ليس الازدحامَ وحدَه**: **رحّال غو يعرض الأصنافَ لا
# المتاجر**، **وخريطةٌ مزدحمةٌ بالمطاعم تنافس سوقَنا في شاشتنا.**
#
# **وأيقوناتُ السائق والمتجر والدبّوس ليست هنا** — تلك مواردُ التطبيق
# يمرّرها بنفسه (`MarkerIcons`)، **ولا تدخل بلاطةَ خريطة.**
#
#   الاستعمال:  sh build-sprite.sh <مجلّد الموارد>
set -eu

OUT=${1:?الاستعمال: build-sprite.sh <مجلّد الموارد>}
MAPS=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SRC="$MAPS/sprite"

# **والأيقوناتُ الأربع** — انظر `poi-label` في النمط.
for icon in poi-hospital poi-pharmacy poi-fuel poi-landmark; do
  [ -f "$SRC/$icon.svg" ] || { echo "  ✗ أيقونةٌ مفقودة: $icon.svg" >&2; exit 3; }
done

if command -v spreet >/dev/null 2>&1; then
  echo "  · spreet"
  spreet "$SRC" "$OUT/sprite"
  spreet --retina "$SRC" "$OUT/sprite@2x"
else
  echo "  ✗ لا أداةَ أيقوناتٍ مثبَّتة." >&2
  echo "    ثبّتها:  cargo install spreet" >&2
  exit 3
fi

for f in "$OUT/sprite.json" "$OUT/sprite.png" "$OUT/sprite@2x.json" "$OUT/sprite@2x.png"; do
  [ -f "$f" ] || { echo "  ✗ ناتجٌ مفقود: $f" >&2; exit 4; }
done
echo "  ✓ الأيقونات"
