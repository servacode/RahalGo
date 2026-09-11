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
#   deploy/build-artifact.sh              # المحرّكُ والويبُ من HEAD النظيف
#   deploy/build-artifact.sh --force-tag  # لإعادة إسنادٍ مقصودٍ ومُعلَن
#   COMPONENT=api|web deploy/build-artifact.sh   # واحدٌ منهما
set -euo pipefail

cd "$(dirname "$0")"
ROOT="$(cd .. && pwd)"
FORCE=""
[ "${1:-}" = "--force-tag" ] && FORCE="yes"
# **والمكوّنُ افتراضُه الاثنان** — **وأثرٌ ناقصٌ يُنشَر نصفَ إصدار.**
COMPONENT="${COMPONENT:-all}"

# ── ١ · شجرةٌ نظيفة — **ولا مرشَّحَ من شجرةٍ تتبدّل** (البند ٣٩) ──────
if [ -n "$(git -C "$ROOT" status --porcelain)" ]; then
	echo "✗ شجرةُ العمل متّسخة — **وأثرٌ من مصدرٍ لا يُعرَف محتواه لا يثبت شيئاً.**" >&2
	git -C "$ROOT" status --short >&2
	exit 2
fi

SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD)"
SHORT="${SOURCE_COMMIT:0:8}"
BUILD_ID="build-$(date -u +%Y%m%dT%H%M%SZ)-$SHORT"

# ── ٣ · بناءُ المحرّك — بهويّةٍ محقونةٍ وقتَ البناء ──────────────────
#
# **ولا تُقرأ الهويّةُ من متغيّرٍ يُبدَّل بعد النشر** (البند ١٠):
# **هويّةٌ تُغيَّر وقتَ التشغيل ليست هويّة.**
#
# **وهذا للمحرّك وحدَه**: **الويبُ لا هويّةَ بيئةٍ فيه أصلاً** (دورةُ
# ٧١و) — **ولو حُقنت لعاد العطبُ الذي أُغلق.**
OUT="${ARTIFACT_DIR:-/srv/rahalgo/artifacts}"
mkdir -p "$OUT"

archive_and_manifest() { # <component> <tag> <image-id>
	local comp="$1" tag="$2" iid="$3"
	local ar="$OUT/rahalgo-$comp-release-$SHORT.tar"
	[ -f "$ar" ] || docker image save "$tag" -o "$ar"
	local sha
	sha="$(sha256sum "$ar" | cut -d' ' -f1)"
	cat > "$OUT/release-$comp-$SHORT.json" <<JSON
{
  "component": "$comp",
  "source_commit": "$SOURCE_COMMIT",
  "build_id": "$BUILD_ID",
  "image_tag": "$tag",
  "image_id": "$iid",
  "archive": "$ar",
  "archive_sha256": "$sha",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
JSON
	echo "${comp^^}_IMAGE_TAG=$tag"
	echo "${comp^^}_IMAGE_ID=$iid"
	echo "${comp^^}_ARCHIVE=$ar"
	echo "${comp^^}_ARCHIVE_SHA256=$sha"
}

build_one() { # <component> <context> <tag> [build-args...]
	local comp="$1" ctx="$2" tag="$3"; shift 3
	if EXISTING="$(docker image inspect "$tag" --format '{{.Id}}' 2>/dev/null)"; then
		if [ -z "$FORCE" ]; then
			echo "✓ أثرُ $comp مبنيٌّ من قبل — **ولا يُعاد بناؤه.**"
			archive_and_manifest "$comp" "$tag" "$EXISTING"
			return 0
		fi
		echo "⚠ إعادةُ إسنادٍ مقصودةٌ لوسمٍ قائم: $tag (كان $EXISTING)" >&2
	fi
	echo "── يُبنى $tag من $SHORT"
	docker build "$@" -t "$tag" "$ctx"
	archive_and_manifest "$comp" "$tag" "$(docker image inspect "$tag" --format '{{.Id}}')"
}

echo "SOURCE_COMMIT=$SOURCE_COMMIT"
echo "BUILD_ID=$BUILD_ID"

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "api" ]; then
	build_one api "$ROOT/backend" "rahalgo-api:release-$SHORT" 		--build-arg "SOURCE_COMMIT=$SOURCE_COMMIT" 		--build-arg "BUILD_ID=$BUILD_ID"
fi

# ── ٤ · بناءُ الويب — بلا هويّةِ بيئةٍ إطلاقاً ────────────────────────
#
# **ولا `--build-arg` لعنوانٍ ولا لبيئة** — **وكانت تُحقَن فتُخبَز في
# الحزمة**: **صفرُ ذكرٍ لـ`api.rahalgo.com` في حزمة التجهيز مقابل ستٍّ
# في حزمة الإنتاج.** **فصورتان من التزامٍ واحدٍ ليستا أثراً واحداً.**
#
# **والبيئةُ تصل وقتَ التشغيل** (`/config.js`) — **فالصورةُ الواحدةُ
# تصلح للبيئتين، وما يُختبَر هنا يُرقَّى هناك.**
if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "web" ]; then
	build_one web "$ROOT/web" "rahalgo-web:release-$SHORT"
fi
