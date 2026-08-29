#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **بناءُ الأساس بمبانيه — لقطةُ OSM + مبانٍ مستخرَجةٌ آليّاً**
# ══════════════════════════════════════════════════════════════════════
#
# (طلبُ المالك ٢٠٢٦-٠٨-٢٤: «أريد الخريطة تصبح مطابقة تقريباً إلى غوغل
#  ماب».)
#
# # ولماذا الدمج قبل Planetiler
#
# **المبانيُ تصير مبانيَ OSM عاديّة** — فتمرّ إلى طبقة `building`
# القياسيّة. **ولا يتغيّر سطرٌ في التطبيق ولا في النمط ولا في الفهرس.**
#
# # وبصمةُ اللقطة تُفحص قبل الدمج لا بعده
#
# **والدمجُ يبدّل الملفَّ فتسقط البصمة** — فيُفحص المصدرُ أوّلاً، **ثمّ
# يُدمج في ملفٍّ ثالث.** ومن دمج ثمّ فحص أسقط بناءه على لا شيء.
#
# # والقرصُ ضيّق
#
# **ثلاثةَ عشرَ غيغا حرّةً** (قِيس ٢٠٢٦-٠٨-٢٤) — فيُحذف كلُّ ملفٍّ
# فورَ انتهاء دوره، **ولا تجتمع نسختان.**
#
#   الاستعمال:  sh build-with-buildings.sh
set -eu

W=/srv/rahalgo/maps/work
OUT=/srv/rahalgo/maps
# **بصمةُ لقطة Geofabrik المثبَّتة** — كما في `build-maps.sh`.
PIN=0a8d6878a3c0da48a8311e8c54ebcce49b4b6d4de5a1a7fccb56cb2a7f9db7ac
# **ونسخةُ بياناتٍ جديدةٌ** — البياناتُ تبدّلت فعلاً، **والعملاءُ
# يُنزّلون بالنسخة لا بالتاريخ.**
DV=2026-08-24

cd "$W"

echo "════ بدأ $(date -u +%FT%TZ) ════"

# ── 1 · the pinned snapshot ──────────────────────────────────────────
if [ ! -f syria.osm.pbf ]; then
  echo "-- downloading snapshot"
  curl -sL --fail -o syria.osm.pbf https://download.geofabrik.de/asia/syria-260820.osm.pbf
fi
GOT=$(sha256sum syria.osm.pbf | cut -d' ' -f1)
if [ "$GOT" != "$PIN" ]; then
  echo "!! snapshot hash mismatch"
  echo "   got: $GOT"
  echo "   pin: $PIN"
  exit 3
fi
echo "-- snapshot verified"

[ -s buildings.osm.pbf ] || { echo "!! buildings.osm.pbf missing"; exit 4; }
echo "-- buildings: $(stat -c %s buildings.osm.pbf) bytes"

# ── 2 · merge ────────────────────────────────────────────────────────
# **و`merge` لا `cat`** — **فـ`cat` تلصق بلا ترتيب** وتُخرج ملفّاً
# يقرؤه Planetiler خطأً أو يرفضه.
if [ ! -s merged.osm.pbf ]; then
  echo "-- merging"
  osmium merge syria.osm.pbf buildings.osm.pbf -o merged.osm.pbf --overwrite
fi
echo "-- merged: $(stat -c %s merged.osm.pbf) bytes"

# ── 3 · tiles ────────────────────────────────────────────────────────
if [ ! -f planetiler.jar ]; then
  echo "-- downloading planetiler"
  curl -sL --fail -o planetiler.jar \
    https://github.com/onthegomap/planetiler/releases/latest/download/planetiler.jar
fi

echo "-- building z0-16"
rm -rf "$W/tmp"
# **و`--download` تجلب البياناتِ المساعدة** — خطوطَ البحيرات ومضلّعاتِ
# المياه وحدودَ Natural Earth. **وهي محذوفةٌ من البناء السابق** (قِيس
# ٢٠٢٦-٠٨-٢٤: `lake_centerline.shp.zip does not exist`)، **وبلاها يقف
# البناءُ في أوّل ثانية** ولا يُنتج شيئاً.
java -Xmx3g -jar planetiler.jar \
  --download \
  --osm-path=merged.osm.pbf \
  --output=syria.pmtiles \
  --force \
  --languages=ar,en \
  --transliterate=false \
  --minzoom=0 --maxzoom=16 \
  --nodemap-type=sortedtable \
  --nodemap-storage=mmap \
  --tmpdir="$W/tmp"

stat -c "-- output %s bytes" syria.pmtiles
sha256sum syria.pmtiles

# ── 4 · install ──────────────────────────────────────────────────────
mkdir -p "$OUT/base/$DV"
mv -f syria.pmtiles "$OUT/base/$DV/syria.pmtiles"
rm -rf "$W/tmp" merged.osm.pbf
echo "-- installed at $OUT/base/$DV/syria.pmtiles"
df -h / | tail -1
echo "════ done $(date -u +%FT%TZ) ════"
