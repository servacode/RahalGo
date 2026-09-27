#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **حزمةُ الرقّةِ المدينيّةُ — للتجهيز وحدَه، ولا تمسّ الإنتاج**
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-٠٩-٢٧: «حزمةٌ بحجم الرقّة لا سوريا كلَّها ٤٧٩م ·
#  STAGING ONLY · ممنوع لمسُ الإنتاج».)
#
# # ما يفعله
#
#   يبني `region-raqqa.pmtiles` (~٢م) بحدود الرقّة من مقتطف OSM نفسِه
#   الذي بُني منه الأساس، ثمّ يضعه في آثار **التجهيز وحدَها** ويحدّث
#   فهرسَ التجهيز ليشير إليه — **لا `source:base`، ملفٌّ حقيقيّ.**
#
# # ما لا يفعله — حرّاسٌ صريحة
#
#   • لا يكتب في `/srv/rahalgo/maps` (الإنتاج) — حارسٌ يُسقطه إن حاول.
#   • لا يشغّل `maps-sync.sh` — فهي تنسخ الإنتاجَ فوقَ التجهيز (`--delete`)
#     فتمحو ما نضعه. **فبعد هذا السكربت لا تُشغَّل المزامنة.**
#   • لا يمسّ `maps.rahalgo.com` ولا أيَّ خدمةِ إنتاج.
#
# # المضيفُ/المساراتُ التي يلمسها — **صفرُ لمسٍ لـ/srv/rahalgo والإنتاج**
#
#   يقرأ  : الأساسَ القائمَ في التجهيز فقط، ومقتطفَ OSM من عمل التجهيز
#           (يُنزَّل إليه إن غاب). **ولا يقرأ ولا يكتب تحت /srv/rahalgo.**
#   يكتب  : `/srv/rahalgo-staging/maps/regions/<dv>/region-raqqa.pmtiles`
#           و`/srv/rahalgo-staging/maps/manifest.json` (بنسخةٍ احتياطيّة)
#           و`/srv/rahalgo-staging/maps-work/` (عملٌ مؤقّت). **لا غير.**
#   يخدَم : `https://staging-api.rahalgo.com/maps` (Caddyfile.staging).
#   لا يُعيد تشغيلَ أيِّ خدمة، ولا يتّصل بـ`maps.rahalgo.com`.
#
#   الاستعمال (بمسارٍ مطلق):
#     setsid nohup sh /srv/rahalgo-staging/build-raqqa-staging.sh \
#       > /srv/rahalgo-staging/raqqa-staging.log 2>&1 &
set -eu

# ── الحدود والمدى من الوصفة نفسِها ───────────────────────────────────
RAQQA_BBOX="38.92,35.88,39.12,36.03"   # REGION_RAQQA في config/build.env
RMIN=10
RMAX=16
DV=2026-08-24                          # نسخةُ بيانات الأساس القائم

# ── مساراتُ التجهيز وحدَها ───────────────────────────────────────────
STAGING=/srv/rahalgo-staging/maps
PROD=/srv/rahalgo/maps                 # للقراءةِ فقط، ويُحرَس ضدَّ الكتابة
WORK=/srv/rahalgo-staging/maps-work

# ── حارسٌ: لا كتابةَ في الإنتاج مهما كان ─────────────────────────────
case "$STAGING" in
  "$PROD"|"$PROD"/*) echo "✗ الهدفُ داخلَ آثار الإنتاج — رفضٌ مطلق" >&2; exit 2 ;;
esac
[ -d "$STAGING" ] || { echo "✗ لا آثارَ تجهيزٍ في $STAGING" >&2; exit 2; }
[ -s "$STAGING/base/$DV/syria.pmtiles" ] || { echo "✗ لا أساسَ تجهيزٍ في $STAGING/base/$DV" >&2; exit 2; }

mkdir -p "$WORK"
cd "$WORK"
echo "════ RAQQA-STAGING START $(date -u +%FT%TZ) ════"
echo "   يكتب في: $STAGING (التجهيز) — ولا يمسّ $PROD (الإنتاج)"

# ── مقتطفُ OSM: في عمل التجهيز وحدَه — **لا لمسَ لـ/srv/rahalgo إطلاقاً** ──
# **ولا يُقرأ من عمل الإنتاج** (٢٠٢٦-٠٩-٢٧): كلُّ شيءٍ تحت التجهيز، ولو
# كلّف تنزيلاً ثانياً — **فصفرُ لمسٍ للإنتاج أوضحُ من قراءةٍ آمنة.**
PIN=0a8d6878a3c0da48a8311e8c54ebcce49b4b6d4de5a1a7fccb56cb2a7f9db7ac
OSM="$WORK/syria.osm.pbf"
if [ ! -s "$OSM" ]; then
  echo "   تنزيلُ مقتطف OSM إلى عمل التجهيز: $OSM"
  curl -sL --fail -o "$OSM" https://download.geofabrik.de/asia/syria-260820.osm.pbf
fi
GOT=$(sha256sum "$OSM" | cut -d' ' -f1)
[ "$GOT" = "$PIN" ] || { echo "!! تعذّرت مطابقةُ بصمة المقتطف: $GOT" >&2; exit 3; }

# ── الأداة (في عمل التجهيز) ──────────────────────────────────────────
JAR="$WORK/planetiler.jar"
[ -f "$JAR" ] || curl -sL --fail -o "$JAR" \
  https://github.com/onthegomap/planetiler/releases/latest/download/planetiler.jar

# ── بناءُ حزمةِ الرقّة بالحدود (لا استخراج) ──────────────────────────
echo "── بناءُ region-raqqa.pmtiles  z$RMIN–z$RMAX  bbox=$RAQQA_BBOX"
rm -rf "$WORK/tmp"
java -Xmx3g -jar "$JAR" \
  --download \
  --osm-path="$OSM" \
  --output="$WORK/region-raqqa.pmtiles" \
  --force \
  --languages=ar,en \
  --transliterate=false \
  --minzoom="$RMIN" --maxzoom="$RMAX" \
  --bounds="$RAQQA_BBOX" \
  --nodemap-type=sortedtable \
  --nodemap-storage=mmap \
  --tmpdir="$WORK/tmp"
rm -rf "$WORK/tmp"
echo "   الحجم: $(stat -c %s "$WORK/region-raqqa.pmtiles") bytes"

# ── النشرُ في التجهيز وحدَه ──────────────────────────────────────────
mkdir -p "$STAGING/regions/$DV"
cp -f "$WORK/region-raqqa.pmtiles" "$STAGING/regions/$DV/region-raqqa.pmtiles"

# ── تحديثُ فهرس التجهيز: منطقةُ الرقّة تشير إلى ملفِّها لا إلى الأساس ──
cp -f "$STAGING/manifest.json" "$STAGING/manifest.json.$(date -u +%Y%m%d%H%M).bak"
python3 - "$STAGING" "$DV" "$RAQQA_BBOX" "$RMIN" "$RMAX" <<'PY'
import hashlib, json, os, sys
staging, dv, bbox, rmin, rmax = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4]), int(sys.argv[5])

def stamp(p):
    h = hashlib.sha256(); n = 0
    with open(p, "rb") as f:
        for c in iter(lambda: f.read(1 << 20), b""):
            n += len(c); h.update(c)
    return n, h.hexdigest()

mpath = os.path.join(staging, "manifest.json")
man = json.load(open(mpath, encoding="utf-8"))
rp = os.path.join(staging, "regions", dv, "region-raqqa.pmtiles")
size, digest = stamp(rp)
w, s, e, n = [float(x) for x in bbox.split(",")]
# **منطقةُ الرقّة — ملفٌّ حقيقيٌّ مدينيٌّ، لا القاعدةُ كلُّها** (٢٠٢٦-٠٩-٢٧)
raqqa = {
    "id": "raqqa", "name": "الرقّة",
    "url": f"regions/{dv}/region-raqqa.pmtiles",
    "bytes": size, "sha256": digest,
    "bbox": [w, s, e, n], "minZoom": rmin, "maxZoom": rmax, "dataVersion": dv,
}
# **يبقى الأساسُ للاتّصال الحيّ** — والمناطقُ تصير الرقّةَ المدينيّةَ وحدَها هنا.
man["regions"] = [raqqa]
tmp = mpath + ".tmp"
json.dump(man, open(tmp, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
open(tmp, "a", encoding="utf-8").write("\n")
os.replace(tmp, mpath)
print(f"   raqqa: {size} bytes · bbox={bbox} · z{rmin}-{rmax}")
PY

echo "-- نُشر في التجهيز:"
grep -E '"id"|"bytes"|"url"' "$STAGING/manifest.json" | sed -n '1,12p'
echo "════ RAQQA-STAGING DONE $(date -u +%FT%TZ) ════"
echo "   تذكير: **لا تُشغّل maps-sync.sh بعد هذا** — تنسخ الإنتاجَ فوقَ التجهيز."
