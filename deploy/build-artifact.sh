#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  بناءُ أثرِ المحرّك — مرّةً واحدةً، بوسمٍ لا يُعاد إسنادُه
# ══════════════════════════════════════════════════════════════════════
#
# (دورةُ ٧١ — إغلاقُ ثغرةِ «كلُّ بيئةٍ تبني لنفسها».)
#
# # ولماذا خطوةٌ منفصلةٌ عن النشر
#
# **وكان `compose` يبني عند كلّ نشر** — **فأثرٌ اختُبر على التجهيز
# ليس الأثرَ الذي يعمل في الإنتاج ولو كان الالتزامُ واحداً.**
# **وفرقٌ في طبقةِ أساسٍ أو في ساعةِ بناءٍ يصنع ثنائيّاً آخر.**
#
#     بناءٌ  ←  وسمٌ ثابت  ←  اختبارٌ على التجهيز  ←  ترقيةُ الأثر عينِه
#
# # والوسمُ يحمل الالتزامَ لا الهجرة
#
# **والهجرةُ تتكرّر بين إصداراتٍ** (`target-0146` قد يعني اثنين)،
# **والالتزامُ لا يتكرّر.** **فوسمُ الإصدار من الالتزام، و`target-*`
# و`rollback-*` أدوارٌ تشير إليه.**
#
# # ولا يُكتَب فوق وسمٍ قائمٍ بمحتوىً مختلف
#
# **ودوكر يُعيد إسنادَ الوسم بلا سؤال** — **ولا يمنعه محلّيّاً شيء.**
# **فمن أعاد بناءَ وسمٍ اعتُمد محا الدليلَ ولم يُخطره أحد.** فيُرفض
# هنا صراحةً.
#
# الاستعمال:
#   deploy/build-artifact.sh              # من HEAD النظيف
#   deploy/build-artifact.sh --force-tag  # لإعادة إسنادٍ مقصودٍ ومُعلَن
set -euo pipefail

cd "$(dirname "$0")"
ROOT="$(cd .. && pwd)"
FORCE=""
[ "${1:-}" = "--force-tag" ] && FORCE="yes"

# ── ١ · شجرةٌ نظيفة — **ولا مرشَّحَ من شجرةٍ تتبدّل** (البند ٣٩) ──────
if [ -n "$(git -C "$ROOT" status --porcelain)" ]; then
	echo "✗ شجرةُ العمل متّسخة — **وأثرٌ من مصدرٍ لا يُعرَف محتواه لا يثبت شيئاً.**" >&2
	git -C "$ROOT" status --short >&2
	exit 2
fi

SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD)"
SHORT="${SOURCE_COMMIT:0:8}"
BUILD_ID="api-$(date -u +%Y%m%dT%H%M%SZ)-$SHORT"
TAG="rahalgo-api:release-$SHORT"

# ── ٢ · أوسمُ الإصدار لا تُعاد كتابتُها ──────────────────────────────
if EXISTING="$(docker image inspect "$TAG" --format '{{.Id}}' 2>/dev/null)"; then
	if [ -z "$FORCE" ]; then
		echo "✓ الأثرُ مبنيٌّ من قبل — **ولا يُعاد بناؤه.**"
		echo "SOURCE_COMMIT=$SOURCE_COMMIT"
		echo "IMAGE_TAG=$TAG"
		echo "IMAGE_ID=$EXISTING"
		exit 0
	fi
	echo "⚠ إعادةُ إسنادٍ مقصودةٌ لوسمٍ قائم: $TAG (كان $EXISTING)" >&2
fi

# ── ٣ · البناءُ بهويّةٍ محقونةٍ وقتَ البناء ──────────────────────────
#
# **ولا تُقرأ الهويّةُ من متغيّرٍ يُبدَّل بعد النشر** (البند ١٠):
# **هويّةٌ تُغيَّر وقتَ التشغيل ليست هويّة.**
echo "── يُبنى $TAG من $SHORT"
docker build \
	--build-arg "SOURCE_COMMIT=$SOURCE_COMMIT" \
	--build-arg "BUILD_ID=$BUILD_ID" \
	-t "$TAG" \
	"$ROOT/backend"

IMAGE_ID="$(docker image inspect "$TAG" --format '{{.Id}}')"

# ── ٤ · بيانُ الإصدار — **والأرشيفُ يُحفَظ خارجَ مخزن دوكر** ─────────
#
# **ومخزنُ دوكر يذهب مع الخادم** — **فأرشيفٌ ببصمةٍ يُبقي الإصدارَ
# قابلاً للاستعادة ولو ضاع الخادم.** (وليس بديلاً عن سجلٍّ بعيدٍ
# حين تتعدّد الخوادم — انظر خطّةَ الدورة.)
OUT="${ARTIFACT_DIR:-/srv/rahalgo/artifacts}"
mkdir -p "$OUT"
ARCHIVE="$OUT/rahalgo-api-release-$SHORT.tar"
if [ ! -f "$ARCHIVE" ]; then
	docker image save "$TAG" -o "$ARCHIVE"
fi
SHA="$(sha256sum "$ARCHIVE" | cut -d' ' -f1)"

cat > "$OUT/release-$SHORT.json" <<JSON
{
  "source_commit": "$SOURCE_COMMIT",
  "build_id": "$BUILD_ID",
  "image_tag": "$TAG",
  "image_id": "$IMAGE_ID",
  "archive": "$ARCHIVE",
  "archive_sha256": "$SHA",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
JSON

echo "SOURCE_COMMIT=$SOURCE_COMMIT"
echo "BUILD_ID=$BUILD_ID"
echo "IMAGE_TAG=$TAG"
echo "IMAGE_ID=$IMAGE_ID"
echo "ARCHIVE=$ARCHIVE"
echo "ARCHIVE_SHA256=$SHA"
echo "MANIFEST=$OUT/release-$SHORT.json"
