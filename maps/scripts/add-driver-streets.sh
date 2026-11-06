#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **إضافةُ شوارع مشي السائقين الموافَق عليها إلى ملفّ الخريطة**
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-١١-٠٦ — الخطوة ٤ من «الخريطة من مشي السائقين».)
#
# يأخذ ملفَّ خريطةٍ (`.osm.pbf`) وملفَّ التصدير من اللوحة (`rahalgo-streets.osm`)
# ويُخرج ملفّاً ثالثاً فيه الشوارعُ الجديدةُ **موصولةً بأقرب طريق** — فتُرسم في
# البلاطات (Planetiler) **ويمرّ بها التوجيه** (Valhalla/OSRM). الوصلُ والتصفيةُ
# في `driver-streets-to-osc.py`.
#
#   الاستعمال:
#     sh add-driver-streets.sh IN.osm.pbf rahalgo-streets.osm OUT.osm.pbf
#
# ثمّ يُبنى من OUT كما يُبنى من الأصل:
#   • البلاطات:  java -jar planetiler.jar --osm-path=OUT.osm.pbf …   (كما في rebuild-all.sh)
#   • التوجيه:   valhalla_build_tiles -c valhalla.json OUT.osm.pbf    (ملفُّ التوجيه route.osm.pbf يمرّ بالأمر نفسِه)
#
# **لا يلمس خادماً ولا يعيد تشغيلَ شيء** — ملفٌّ يدخل وملفٌّ يخرج.
# يحتاج: osmium-tool وpython3.
set -eu

[ $# -eq 3 ] || { echo "الاستعمال: sh add-driver-streets.sh IN.osm.pbf streets.osm OUT.osm.pbf" >&2; exit 2; }
IN=$1
ST=$2
OUT=$3
HERE=$(cd "$(dirname "$0")" && pwd)
[ -s "$IN" ] || { echo "✗ لا ملفَّ خريطة: $IN" >&2; exit 2; }
[ -s "$ST" ] || { echo "✗ لا ملفَّ شوارع: $ST" >&2; exit 2; }

if ! grep -q '<way ' "$ST"; then
  echo "driver-streets: لا شوارعَ موافَقاً عليها — نسخٌ بلا تغيير"
  cp -f "$IN" "$OUT"
  exit 0
fi

T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT

BBOX=$(python3 "$HERE/driver-streets-to-osc.py" bbox "$ST")
echo "driver-streets: bbox=$BBOX"
# **الطرقُ وحدَها حول الشوارع** — ملفٌّ صغيرٌ يقرؤه بايثون (`extract` يُبقي الطرقَ كاملةً بعقدها).
osmium tags-filter "$IN" w/highway -o "$T/hw.osm.pbf" --overwrite
osmium extract -b "$BBOX" "$T/hw.osm.pbf" -o "$T/hw.osm" --overwrite
python3 "$HERE/driver-streets-to-osc.py" build "$T/hw.osm" "$ST" "$T/streets.osc"
osmium apply-changes "$IN" "$T/streets.osc" -o "$OUT" --overwrite
echo "driver-streets: $(stat -c %s "$OUT") bytes → $OUT"
