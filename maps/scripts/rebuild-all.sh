#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **أمرٌ واحدٌ من التنزيل إلى البلاطات — لأنّ الوصلَ يُقطع**
# ══════════════════════════════════════════════════════════════════════
#
# (طلبُ المالك ٢٠٢٦-٠٨-٢٤.)
#
# # ولماذا سكربتٌ واحدٌ لا خمسة
#
# **السيرفرُ يحظر عنوانَ المطوّر بعد كلّ اتّصالٍ أو اتّصالين**
# (`fail2ban`, قِيس ٢٠٢٦-٠٨-٢٤: انتظارٌ من خمسٍ إلى تسع دقائقَ بين
# أمرٍ وأمر). **فخمسُ خطواتٍ في خمس جلساتٍ نصفُ ساعةِ انتظارٍ صافية.**
#
# **وهذا يُشغَّل مرّةً ويمضي وحدَه** — ويُقرأ سجلُّه حين يتيسّر الوصل.
#
#   الاستعمال:  setsid nohup sh rebuild-all.sh > all.log 2>&1 &
set -eu

M=/srv/rahalgo/maps
W=$M/work
PIN=0a8d6878a3c0da48a8311e8c54ebcce49b4b6d4de5a1a7fccb56cb2a7f9db7ac
DV=2026-08-24

mkdir -p "$W"
cd "$M"
echo "════ START $(date -u +%FT%TZ) ════"

step() { echo "── $1 · $(date -u +%T) · disk=$(df -h / | awk 'NR==2{print $4}') ──"; }

# ── 1 · buildings ────────────────────────────────────────────────────
if [ ! -s "$W/buildings.osm.pbf" ]; then
  step "fetch buildings"
  WORK="$M" REGION=Syria sh fetch-buildings.sh

  step "convert to osm"
  python3 buildings-to-osm.py "$W/buildings.osm" "$M"/buildings/qk-*.geojsonl

  step "pack to pbf"
  osmium cat "$W/buildings.osm" -o "$W/buildings.osm.pbf" -f pbf --overwrite
  # **ويُحذف فورَ ضغطه** — **والقرصُ لا يحتمل النسختين** (٢٫٦ غيغا
  # مقابل ١١٧ ميغا).
  rm -f "$W/buildings.osm"
  rm -rf "$M/buildings"
fi
step "buildings ready: $(stat -c %s "$W/buildings.osm.pbf") bytes"

# ── 2 · snapshot ─────────────────────────────────────────────────────
cd "$W"
if [ ! -f syria.osm.pbf ]; then
  step "download snapshot"
  curl -sL --fail -o syria.osm.pbf https://download.geofabrik.de/asia/syria-260820.osm.pbf
fi
GOT=$(sha256sum syria.osm.pbf | cut -d' ' -f1)
[ "$GOT" = "$PIN" ] || { echo "!! snapshot hash mismatch: $GOT"; exit 3; }
step "snapshot verified"

# ── 3 · merge ────────────────────────────────────────────────────────
# **و`merge` لا `cat`** — **فـ`cat` تلصق بلا ترتيب**، وPlanetiler
# يقرأ مرتّباً.
rm -f merged.osm.pbf
step "merge"
osmium merge syria.osm.pbf buildings.osm.pbf -o merged.osm.pbf --overwrite
step "merged: $(stat -c %s merged.osm.pbf) bytes"

# ── 4 · tiles ────────────────────────────────────────────────────────
if [ ! -f planetiler.jar ]; then
  step "download planetiler"
  curl -sL --fail -o planetiler.jar \
    https://github.com/onthegomap/planetiler/releases/latest/download/planetiler.jar
fi

step "build z0-16"
rm -rf "$W/tmp"
# **و`--download` تجلب البياناتِ المساعدة** — بحيراتٌ ومياهٌ وحدود.
# **وبلاها يقف البناءُ في أوّل ثانية** (قِيس ٢٠٢٦-٠٨-٢٤).
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

step "output: $(stat -c %s syria.pmtiles) bytes"
sha256sum syria.pmtiles

mkdir -p "$M/base/$DV"
mv -f syria.pmtiles "$M/base/$DV/syria.pmtiles"
rm -rf "$W/tmp" merged.osm.pbf
step "installed at $M/base/$DV/syria.pmtiles"
echo "════ DONE $(date -u +%FT%TZ) ════"
