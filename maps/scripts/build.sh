#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **بناءُ الخرائط — أمرٌ واحدٌ يُعاد تشغيلُه**
# ══════════════════════════════════════════════════════════════════════
#
# (المرحلة ٦أ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٢ و٢٨.)
#
#   OSM  →  Planetiler  →  تحقُّق  →  أثرٌ مؤرَّخ  →  فهرس  →  نشر
#
# **ولا يعتمد على خطواتٍ يدويّةٍ من جهاز مطوّر** — أمرُ المالك نصّاً.
#
#   الاستعمال:
#     WORK=/srv/maps sh maps/scripts/build.sh
#
#   المتطلّبات: Java 21+ · curl · sha256sum · node
#
# **ومجلّدُ العمل خارجَ المستودع** — والأثرُ ثلاثُ مئةِ ميغابايت،
# **ولا يدخل Git** (البند ٢٩).
set -eu

MAPS=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
WORK=${WORK:-/srv/maps}
. "$MAPS/config/build.env"

PLANETILER_URL="https://github.com/onthegomap/planetiler/releases/latest/download/planetiler.jar"
JAR="$WORK/planetiler.jar"

mkdir -p "$WORK/out"
cd "$WORK"

# ── ١ · الأداة ──────────────────────────────────────────────────────
# **تُنزَّل مرّةً وتبقى** — تسعون ميغابايت.
if [ ! -f "$JAR" ]; then
  echo "▸ تنزيلُ Planetiler"
  curl -sL --fail -o "$JAR" "$PLANETILER_URL"
fi

JAVA=${JAVA:-java}
COMMON="--area=$AREA --languages=$LANGUAGES --transliterate=$TRANSLITERATE \
  --nodemap-type=$NODEMAP --storage=$STORAGE --force"

# ── ٢ · الأرشيفُ الأساسيّ ───────────────────────────────────────────
# **z0–z16 لكلّ سوريا** — والقياسُ حسم المدى، انظر `build.env`.
echo "▸ بناءُ الأساس  z$BASE_MINZOOM–z$BASE_MAXZOOM"
# shellcheck disable=SC2086
"$JAVA" -Xmx"$JAVA_HEAP" -jar "$JAR" $COMMON \
  --minzoom="$BASE_MINZOOM" --maxzoom="$BASE_MAXZOOM" \
  --output="$WORK/out/syria.pmtiles"

# ── ٣ · الحزمُ الإقليميّة ───────────────────────────────────────────
# **بالحدود لا بالاستخراج** — أمرُ المالك: «ممنوع `pmtiles extract`».
for var in $(grep -o '^REGION_[A-Z_]*' "$MAPS/config/build.env"); do
  spec=$(eval "printf '%s' \"\$$var\"")
  id=$(printf '%s' "$var" | sed 's/^REGION_//' | tr 'A-Z' 'a-z')
  name=$(printf '%s' "$spec" | cut -d'|' -f1)
  bbox=$(printf '%s' "$spec" | cut -d'|' -f2)
  rmin=$(printf '%s' "$spec" | cut -d'|' -f3)
  rmax=$(printf '%s' "$spec" | cut -d'|' -f4)
  echo "▸ بناءُ حزمةِ $name  ($id)  z$rmin–z$rmax"
  # shellcheck disable=SC2086
  "$JAVA" -Xmx"$JAVA_HEAP" -jar "$JAR" $COMMON \
    --minzoom="$rmin" --maxzoom="$rmax" --bounds="$bbox" \
    --output="$WORK/out/region-$id.pmtiles"
done

# ── ٤ · موارِدُ الخريطة ─────────────────────────────────────────────
# **مرّةً واحدةً لكلّ المناطق** — انظر `resources.sh` والبند ٧.
sh "$MAPS/scripts/resources.sh"

# ── ٥ · التحقّقُ قبل النشر ──────────────────────────────────────────
# **ولا يُنشَر أثرٌ لم يُفحص** — أمرُ المالك، البند ٢٧.
echo "▸ التحقّق"
node "$MAPS/scripts/validate.mjs" all "$WORK/out"

# ── ٦ · الفهرس ──────────────────────────────────────────────────────
# **يُكتب آخرَ شيء** — فلا يشير إلى ما لم يُرفَع بعد. (البند ١٠.)
echo "▸ الفهرس"
node "$MAPS/scripts/manifest.mjs" "$WORK/out" > "$WORK/out/manifest.json"

echo "✓ تمّ — الأثرُ في $WORK/out"
ls -la "$WORK/out"
