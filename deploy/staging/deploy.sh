#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  نشرُ التجهيز — بهويّةٍ ثابتةٍ لا بنسخِ ملفّات
# ══════════════════════════════════════════════════════════════════════
#
# (`P-0` البنود ١٠ و٣٦ و٣٧ و٣٩.)
#
# # ولا `scp` لملفّاتٍ مجهولة (البند ٣٧)
#
# **من نسخ ملفّاتٍ بيده لا يعرف أيَّ شيفرةٍ تعمل** — **ولا تستطيع
# البوّابةُ (`P-10`) أن تحكم على مرشَّحٍ لا هويّةَ له.**
#
# **فالمسارُ**: التزامٌ نظيف ← وسمٌ ← بناءٌ بهويّةٍ محقونة ← هجرات ←
# فحصُ صحّة.
#
# # وشجرةٌ متّسخةٌ ليست مرشَّحاً (البند ٣٩)
#
# **ويُرفَض النشرُ منها** — **ودليلٌ من بناءٍ لا يُعرَف محتواه لا يثبت
# شيئاً.**
set -euo pipefail

cd "$(dirname "$0")"
ROOT="$(cd ../.. && pwd)"

# ── ١ · شجرةٌ نظيفة ───────────────────────────────────────────────────
if [ -n "$(git -C "$ROOT" status --porcelain)" ]; then
	echo "✗ شجرةُ العمل متّسخة — **ولا مرشَّحَ من شجرةٍ تتبدّل.**" >&2
	git -C "$ROOT" status --short >&2
	exit 2
fi

SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD)"
BUILD_ID="stg-$(date -u +%Y%m%dT%H%M%SZ)-${SOURCE_COMMIT:0:8}"

# ── ٢ · البيئة ────────────────────────────────────────────────────────
if [ ! -f .env.staging ]; then
	echo "✗ لا ملفَّ .env.staging — انسخ .env.staging.example واملأه." >&2
	exit 2
fi

# **ولا يُنشَر إلّا بعد أن يرضى حارسُ الإنتاج** — **ومتغيّرٌ واحدٌ خاطئٌ
# يجعل النشرَ يمسّ الإنتاج.**
set -a
# shellcheck disable=SC1091
. ./.env.staging
set +a
export APP_ENV=staging RAHALGO_STAGING=1
export DATABASE_URL="postgres://rahalgo:${STAGING_DB_PASSWORD}@localhost:5534/rahalgo_staging?sslmode=disable"
export REDIS_URL="redis://localhost:6580/0"
export NEXT_PUBLIC_API_URL="${STAGING_API_URL}"

if ! (cd "$ROOT/backend" && go run ./cmd/stagingctl guard); then
	echo "✗ حارسُ الإنتاج رفض النشر." >&2
	exit 3
fi

# ── ٢·٥ · **والهدفُ يقول من هو قبل أن يُمَسّ** ────────────────────────
#
# ══════════════════════════════════════════════════════════════════════
# **وكان أوّلُ مَسٍّ يسبق أوّلَ سؤال** (٢٠٢٦-٠٩-١٥)
# ══════════════════════════════════════════════════════════════════════
#
# **و`promote.sh` يسأل قبل التبديل** — **لكنّه يجيء بعد `up -d` الذي
# يرفع البوّابةَ والويبَ والقاعدةَ والذاكرة.** **فلو كان المكدّسُ
# المقصودُ إنتاجاً لَمُسّ قبل أن يسأله أحدٌ من هو.**
#
# **والسؤالُ الآن أوّلُ ما يقع بعد حارس البيئة** — **وقبل البناء وقبل
# `up`**: **ومن ردَّ «إنتاج» وقف النشرُ ولم يُمَسّ شيء.**
"$ROOT/deploy/preflight-env.sh" "http://localhost:8080/api/v1/public/identity" staging

# ── ٣ · بناءُ الأثر — مرّةً، بوسمٍ ثابت ───────────────────────────────
#
# ══════════════════════════════════════════════════════════════════════
# **والبناءُ صار خطوةً قائمةً بذاتها** (دورةُ ٧١)
# ══════════════════════════════════════════════════════════════════════
#
# **وكان `compose build` يبني عند كلّ نشر** — **فأثرُ التجهيز غيرُ
# أثر الإنتاج ولو كان الالتزامُ واحداً**، **ولا يُثبت اختبارٌ هنا
# شيئاً عن هناك.**
#
# **والوسمُ يحمل الالتزام**، **و`build-artifact.sh` لا يُعيد بناءَ
# وسمٍ قائم** — **فإعادةُ النشر لا تبدّل الأثر.**
echo "── يُبنى أثرُ ${SOURCE_COMMIT:0:8}"
ARTIFACT_OUT="$(../build-artifact.sh)"
echo "$ARTIFACT_OUT"
IMAGE_TAG="$(echo "$ARTIFACT_OUT" | sed -n 's/^IMAGE_TAG=//p')"
IMAGE_ID="$(echo "$ARTIFACT_OUT"  | sed -n 's/^IMAGE_ID=//p')"
[ -n "$IMAGE_TAG" ] || { echo "✗ لا وسمَ للأثر — **وقف.**" >&2; exit 4; }

# ── ٤ · بقيّةُ المكدّس ثمّ ترقيةُ المحرّك بالأثر عينِه ────────────────
#
# **والمحرّكُ يُرقّى بـ`promote.sh` لا بـ`up` مجرَّدةً** — **فيُقارَن
# المنتظَرُ بالفعليّ قبلَ النشر وبعدَه.**
export RAHALGO_API_IMAGE="$IMAGE_TAG"
docker compose -f compose.staging.yml --env-file .env.staging up -d --no-deps caddy web postgres redis
TARGET_ENV=staging ../promote.sh compose.staging.yml .env.staging "$IMAGE_TAG" 	http://localhost:8080/api/v1/public/identity "$IMAGE_ID"

# ── ٥ · الهجرات ثمّ الصحّة ───────────────────────────────────────────
#
# **والمحرّكُ يهاجر عند إقلاعه** — فيُنتظَر ثمّ يُسأل.
echo "── يُنتظَر المحرّك"
for i in $(seq 1 60); do
	if curl -fsS "http://localhost:8080/api/v1/healthz" >/dev/null 2>&1; then
		break
	fi
	sleep 2
done

echo "── الهويّة"
IDENTITY="$(curl -fsS http://localhost:8080/api/v1/public/identity)" || {
	echo "✗ بابُ الهويّة لا يردّ — **ومرشَّحٌ لا يقول من هو ليس مرشَّحاً.**" >&2
	exit 4
}
echo "$IDENTITY"

# **ويُتحقَّق أنّ ما يعمل هو ما بُني** — **لا ما كان يعمل قبل النشر.**
case "$IDENTITY" in
	*"\"environment\":\"staging\""*) ;;
	*) echo "✗ البيئةُ ليست staging — **وقف.**" >&2; exit 5 ;;
esac
case "$IDENTITY" in
	*"$SOURCE_COMMIT"*) ;;
	*) echo "✗ الالتزامُ الذي يعمل غيرُ الذي بُني." >&2; exit 5 ;;
esac

echo "✓ نُشر ${BUILD_ID}"
