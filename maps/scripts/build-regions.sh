#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **حزمُ المناطق من الأساس الجديد — الرقّة ودمشق**
# ══════════════════════════════════════════════════════════════════════
#
# (طلبُ المالك ٢٠٢٦-٠٨-٢٤.)
#
# # والعنقدةُ مرّةً واحدةً للاثنتين
#
# **و`pmtiles cluster` تعمل في الملفّ نفسِه** — فتُنسخ نسخةٌ واحدةٌ
# وتُعنقد، **ثمّ يُستخرج منها إقليمان.** ومن عنقد لكلّ إقليمٍ نسخةً
# ضاعف العملَ والقرص.
#
# # والحدودُ كما في السكربتات السابقة
#
# **ولا تُخترع** — دمشق كما في `build-damascus.sh`، والرقّة كما في
# الفهرس القائم.
#
#   الاستعمال:  setsid nohup sh build-regions.sh > regions.log 2>&1 &
set -eu

M=/srv/rahalgo/maps
W=$M/work
DV=2026-08-24

cd "$W"
echo "════ START $(date -u +%FT%TZ) ════"
step() { echo "── $1 · $(date -u +%T) · disk=$(df -h / | awk 'NR==2{print $4}') ──"; }

BASE="$M/base/$DV/syria.pmtiles"
[ -s "$BASE" ] || { echo "!! no base at $BASE"; exit 3; }

step "copy base ($(stat -c %s "$BASE") bytes)"
cp -f "$BASE" clustered.pmtiles

step "cluster"
./pmtiles cluster clustered.pmtiles

mkdir -p "$M/regions/$DV"

step "extract raqqa"
./pmtiles extract clustered.pmtiles region-raqqa.pmtiles \
  --bbox="38.90,35.88,39.12,36.02" --maxzoom=16
stat -c "   raqqa %s bytes" region-raqqa.pmtiles
sha256sum region-raqqa.pmtiles
mv -f region-raqqa.pmtiles "$M/regions/$DV/region-raqqa.pmtiles"

step "extract damascus"
./pmtiles extract clustered.pmtiles region-damascus.pmtiles \
  --bbox="36.15,33.40,36.45,33.62" --maxzoom=16
stat -c "   damascus %s bytes" region-damascus.pmtiles
sha256sum region-damascus.pmtiles
mv -f region-damascus.pmtiles "$M/regions/$DV/region-damascus.pmtiles"

rm -f clustered.pmtiles
step "done"
ls -la "$M/regions/$DV/"
echo "════ DONE $(date -u +%FT%TZ) ════"
