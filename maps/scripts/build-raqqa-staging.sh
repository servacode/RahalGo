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
# # الرصدُ والأمان — أُضيف ٢٠٢٦-٠٩-٢٧ بعد تعلّقِ بناءٍ صامت
#
#   • **قفلٌ للتجهيز وحدَه** (`flock` على `raqqa-build.lock`): بناءان لا
#     يعملان معاً — الثاني يرفض بلا لمسِ عملِ الأوّل، ويُكتب رقمُ العملية.
#   • **سجلٌّ بطوابع زمنيّةٍ لكلّ خطوة** عبر `log()` — فيُعرف أين علِق.
#   • **مُهلٌ صريحة**: تنزيلاتُ `curl` (اتّصالٌ ٣٠ث، أقصى ٣٠د، ٣ محاولات)،
#     وخطوةُ `planetiler` كلُّها في `timeout 45m` — **فلا تعلّقَ أبديّ**
#     (المشتبَهُ الأوّل: `--download` يجلب بياناتٍ مساعِدةً خارجيّةً بلا حدّ).
#   • **تنظيفٌ آمنٌ للآثار الجزئيّة** عند أيّ فشلٍ أو مقاطعة: يُحذف
#     `region-raqqa.pmtiles` الناقصُ و`tmp` — **تحت التجهيز وحدَه** —
#     فتبقى إعادةُ التشغيل نظيفة. **ويُبقى المقتطفُ والأداةُ المخبّآن.**
#   • **علامةُ نجاحٍ** تُكتب في النهاية فقط.
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
#     setsid nohup sh <path>/build-raqqa-staging.sh \
#       >> /srv/rahalgo-staging/raqqa-staging.log 2>&1 &
set -eu

# ── مساراتُ التجهيز وحدَها ───────────────────────────────────────────
STAGING=/srv/rahalgo-staging/maps
PROD=/srv/rahalgo/maps                 # للقراءةِ فقط، ويُحرَس ضدَّ الكتابة
WORK=/srv/rahalgo-staging/maps-work
STAGE_ROOT=/srv/rahalgo-staging
LOCKFILE="$STAGE_ROOT/raqqa-build.lock"
PIDFILE="$WORK/raqqa-build.pid"
OUT="$WORK/region-raqqa.pmtiles"
TMPDIR_BUILD="$WORK/tmp"
SUCCESS=0                              # يصير ١ عند النجاح فقط

# ── سجلٌّ بطابعٍ زمنيّ — كلُّ خطوةٍ مؤرَّخة ────────────────────────────
log() { echo "[$(date -u +%FT%TZ)] $*"; }

# ── حارسٌ: لا كتابةَ في الإنتاج مهما كان ─────────────────────────────
case "$STAGING" in
  "$PROD"|"$PROD"/*) echo "✗ الهدفُ داخلَ آثار الإنتاج — رفضٌ مطلق" >&2; exit 2 ;;
esac
case "$WORK" in
  "$PROD"|"$PROD"/*) echo "✗ عملُ البناء داخلَ الإنتاج — رفضٌ مطلق" >&2; exit 2 ;;
esac
[ -d "$STAGING" ] || { echo "✗ لا آثارَ تجهيزٍ في $STAGING" >&2; exit 2; }

# ── قفلُ التجهيز: بناءٌ واحدٌ لا أكثر ────────────────────────────────
# **لو علِق بناءٌ آخرُ ممسِكاً بالقفل، هذا يرفض بلا لمسِ عمله** — والأمرُ
# الإنقاذيُّ (خارجَ السكربت) هو من يوقف العالقَ ثمّ يعيد التشغيل.
exec 9>"$LOCKFILE"
if ! flock -n 9; then
  echo "✗ بناءُ رقّةٍ تجهيزيٌّ آخرُ يعمل (القفل ممسوك). لا تشغيلَ ثانٍ." >&2
  [ -s "$PIDFILE" ] && echo "  رقمُ العملية المُمسِكة (تقريباً): $(cat "$PIDFILE" 2>/dev/null)" >&2
  exit 4
fi

mkdir -p "$WORK"
echo $$ > "$PIDFILE"

# ── تنظيفٌ آمنٌ للآثار الجزئيّة عند الخروج بلا نجاح ─────────────────
cleanup() {
  rc=$?
  rm -rf "$TMPDIR_BUILD" 2>/dev/null || true
  if [ "$SUCCESS" -ne 1 ]; then
    log "✗ فشلٌ/مقاطعةٌ (rc=$rc) — تنظيفُ الآثار الجزئيّة تحت التجهيز وحدَه"
    # **حذفُ المخرَجِ الناقص فقط** — لا المقتطف، لا الأداة، لا الإنتاج.
    rm -f "$OUT" 2>/dev/null || true
  fi
  rm -f "$PIDFILE" 2>/dev/null || true
  # القفلُ يُحرَّر تلقائيّاً بإغلاق الواصف ٩ عند خروج العملية.
}
trap cleanup EXIT
trap 'log "✗ خطأ عند السطر $LINENO"; exit 1' INT TERM

# ── الحدود والمدى من الوصفة نفسِها ───────────────────────────────────
RAQQA_BBOX="38.92,35.88,39.12,36.03"   # REGION_RAQQA في config/build.env
RMIN=10
RMAX=16
DV=2026-08-24                          # نسخةُ بيانات الأساس القائم

[ -s "$STAGING/base/$DV/syria.pmtiles" ] || { echo "✗ لا أساسَ تجهيزٍ في $STAGING/base/$DV" >&2; exit 2; }

cd "$WORK"
log "════ RAQQA-STAGING START ════"
log "يكتب في: $STAGING (التجهيز) — ولا يمسّ $PROD (الإنتاج)"

# ── مقتطفُ OSM: في عمل التجهيز وحدَه — **لا لمسَ لـ/srv/rahalgo إطلاقاً** ──
# **ولا يُقرأ من عمل الإنتاج** (٢٠٢٦-٠٩-٢٧): كلُّ شيءٍ تحت التجهيز، ولو
# كلّف تنزيلاً ثانياً — **فصفرُ لمسٍ للإنتاج أوضحُ من قراءةٍ آمنة.**
#
# **المصدرُ مستقرٌّ لا مؤرَّخ** (إصلاحُ ٢٠٢٦-٠٩-٢٧): كان الرابطُ يشير إلى
# لقطةٍ يوميّةٍ مؤرَّخةٍ (`syria-260820`)، **وGeofabrik تُسقط اللقطاتِ
# اليوميّةَ القديمةَ فيردّ الرابطُ 404** — وهو ما أوقف البناء. **فصار
# `syria-latest.osm.pbf` (رابطٌ ثابتٌ لا يشيخ).**
#
# **والسلامةُ لم تُضعَّف بل صارت دائمة**: بدل بصمةٍ مجمَّدةٍ لملفٍّ زال،
# **نتحقّق ضدَّ md5 الذي ينشره المزوّدُ نفسُه** (`.md5`)، ونسقط عند أيّ
# عدم تطابق. **بوّابةُ سلامةٍ صارمةٌ باقية** — والمقايضةُ الوحيدةُ أنّ
# البيانات «أحدثُ لقطة» لا لقطةً بعينها (وهي مقايضةٌ حتميّةٌ إذ زال الملفّ).
OSM="$WORK/syria.osm.pbf"
OSM_URL="https://download.geofabrik.de/asia/syria-latest.osm.pbf"
OSM_MD5_URL="$OSM_URL.md5"
OSM_MIN_BYTES=40000000                 # أرضيّةُ عقلٍ (~٤٠م)؛ latest نحو ٨٢م
need_dl=1
if [ -s "$OSM" ]; then
  sz=$(stat -c %s "$OSM")
  if [ "$sz" -ge "$OSM_MIN_BYTES" ]; then
    need_dl=0
    log "مقتطفٌ مخبّأٌ صالحُ الحجم ($sz bytes) — بلا تنزيلٍ ولا لمسِ شبكة"
  else
    log "مقتطفٌ مخبّأٌ صغيرٌ مريبٌ ($sz bytes) — يُحذف ويُعاد تنزيله"
    rm -f "$OSM"
  fi
fi
if [ "$need_dl" -eq 1 ]; then
  # **فحصٌ مسبقٌ يسقط باكراً وبوضوحٍ لو شاخ الرابط** — لا تعلّقَ، لا 404 مبهم.
  code=$(curl -sIL -o /dev/null -w '%{http_code}' --connect-timeout 30 --max-time 60 "$OSM_URL" 2>/dev/null || echo 000)
  [ "$code" = "200" ] || { echo "✗ مصدرُ OSM غيرُ متاحٍ (HTTP $code) — رابطٌ شائخ؟: $OSM_URL" >&2; exit 3; }
  log "تنزيلُ مقتطف OSM المستقرّ: $OSM_URL (مُهلة: اتّصال ٣٠ث · أقصى ٣٠د · ٣ محاولات)"
  if ! curl -L --fail --connect-timeout 30 --max-time 1800 --retry 3 --retry-delay 10 \
        -o "$OSM.part" "$OSM_URL"; then
    rm -f "$OSM.part"; echo "✗ تعذّر تنزيلُ مقتطف OSM (شبكة/مُهلة)" >&2; exit 3
  fi
  # **تحقّقُ السلامةِ ضدَّ md5 المنشورِ من المزوّد** — بوّابةٌ صارمةٌ تسقط عند أيّ خلل.
  PUB=$(curl -fsSL --connect-timeout 30 --max-time 120 --retry 3 "$OSM_MD5_URL" 2>/dev/null | awk '{print $1}')
  [ -n "$PUB" ] || { rm -f "$OSM.part"; echo "✗ تعذّر جلبُ md5 المنشور — لا تحقّقَ بلا مرجع" >&2; exit 3; }
  GOTMD5=$(md5sum "$OSM.part" | awk '{print $1}')
  [ "$GOTMD5" = "$PUB" ] || { rm -f "$OSM.part"; echo "✗ عدمُ تطابق md5: $GOTMD5 != $PUB" >&2; exit 3; }
  mv -f "$OSM.part" "$OSM"
  log "سلامةُ المقتطفِ مؤكَّدةٌ ضدَّ md5 المزوّد ($PUB)"
fi
# **بصمةُ sha256 تُسجَّل للتدقيق** (لا تحكم البناءَ بل تُبقي الأثرَ قابلاً للمراجعة).
log "sha256 المقتطف: $(sha256sum "$OSM" | cut -d' ' -f1)"

# ── الأداة (في عمل التجهيز) ──────────────────────────────────────────
JAR="$WORK/planetiler.jar"
if [ ! -s "$JAR" ]; then
  log "تنزيلُ planetiler.jar (مُهلة: اتّصال ٣٠ث · أقصى ٣٠د · ٣ محاولات)"
  if ! curl -L --fail --connect-timeout 30 --max-time 1800 --retry 3 --retry-delay 10 \
        -o "$JAR.part" \
        https://github.com/onthegomap/planetiler/releases/latest/download/planetiler.jar; then
    rm -f "$JAR.part"; echo "✗ تعذّر تنزيلُ planetiler.jar (شبكة/مُهلة)" >&2; exit 3
  fi
  mv -f "$JAR.part" "$JAR"
fi

# ── بناءُ حزمةِ الرقّة بالحدود (لا استخراج) ──────────────────────────
# **المشتبَهُ الأوّلُ في التعلّق**: `--download` يجلب بياناتٍ مساعِدةً
# خارجيّةً (مياهٌ/حدود) بلا حدٍّ زمنيّ، وقد يبطئ أو يعلق على شبكة الصندوق.
# فكلُّ الخطوةِ في `timeout 45m` — فإن تجاوزتها ماتت ونُظِّفت الآثار.
log "── بناءُ region-raqqa.pmtiles  z$RMIN–z$RMAX  bbox=$RAQQA_BBOX  (timeout 45m, -Xmx3g)"
rm -rf "$TMPDIR_BUILD"
set +e
timeout --signal=TERM --kill-after=60 45m \
  java -Xmx3g -jar "$JAR" \
  --download \
  --osm-path="$OSM" \
  --output="$OUT" \
  --force \
  --languages=ar,en \
  --transliterate=false \
  --minzoom="$RMIN" --maxzoom="$RMAX" \
  --bounds="$RAQQA_BBOX" \
  --nodemap-type=sortedtable \
  --nodemap-storage=mmap \
  --tmpdir="$TMPDIR_BUILD"
jrc=$?
set -e
rm -rf "$TMPDIR_BUILD"
if [ "$jrc" -eq 124 ] || [ "$jrc" -eq 137 ]; then
  echo "✗ planetiler تجاوز المُهلة (45m) وأُنهي — المخرَجُ الجزئيُّ سيُنظَّف" >&2; exit 5
fi
[ "$jrc" -eq 0 ] || { echo "✗ planetiler فشل (rc=$jrc)" >&2; exit 5; }
[ -s "$OUT" ] || { echo "✗ لا مخرَجَ رغم نجاحِ الأداة" >&2; exit 5; }
log "الحجم: $(stat -c %s "$OUT") bytes"

# ── النشرُ في التجهيز وحدَه ──────────────────────────────────────────
mkdir -p "$STAGING/regions/$DV"
cp -f "$OUT" "$STAGING/regions/$DV/region-raqqa.pmtiles"

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
SUCCESS=1
log "════ RAQQA-STAGING DONE ════"
echo "   تذكير: **لا تُشغّل maps-sync.sh بعد هذا** — تنسخ الإنتاجَ فوقَ التجهيز."
