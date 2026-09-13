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
ROOT="${SRC_ROOT:-$(cd .. && pwd)}"
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

# ── ٢ · وهويّةُ المصدر — من المستودع أو من بيانٍ صريح ────────────────
#
# ══════════════════════════════════════════════════════════════════════
# **ولماذا وضعُ الأرشيف** (٢٠٢٦-٠٩-١٣)
# ══════════════════════════════════════════════════════════════════════
#
# **والمستودعُ ليس على خادم الإنتاج** — يصل المصدرُ أرشيفَ `git`
# بعينه. **فهذا الملفُّ كان لا يعمل هناك**، **فكُتبت سكربتاتٌ عارضةٌ
# بديلةٌ بيدٍ عجلى** — **وواحدةٌ منها نسيت `--build-arg` فرُقّي إلى
# الإنتاج أثرٌ بهويّةٍ فارغة.** (قِيس ٢٠٢٦-٠٩-١٣.)
#
# **وحارسٌ يُتجاوَز ليس حارساً** — **والعلاجُ أن يُستغنى عن التجاوز
# لا أن يُضاف حارسٌ ثالث.**
#
#	في المستودع:  deploy/build-artifact.sh
#	على الخادم :  SOURCE_COMMIT=<sha40> SRC_ROOT=<مجلّد> \
#	              deploy/build-artifact.sh
if [ -n "${SOURCE_COMMIT:-}" ]; then
	case "$SOURCE_COMMIT" in
		*[!0-9a-f]*|"") echo "✗ **SOURCE_COMMIT ليس بصمةً كاملة**: $SOURCE_COMMIT" >&2; exit 2 ;;
	esac
	[ "${#SOURCE_COMMIT}" -eq 40 ] || { echo "✗ **SOURCE_COMMIT طولُه ${#SOURCE_COMMIT} لا ٤٠**" >&2; exit 2; }
else
	if [ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null)" ]; then
		echo "✗ شجرةُ العمل متّسخة — **وأثرٌ من مصدرٍ لا يُعرَف محتواه لا يثبت شيئاً.**" >&2
		git -C "$ROOT" status --short >&2
		exit 2
	fi
	SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null)" \
		|| { echo "✗ **لا مستودعَ هنا ولا SOURCE_COMMIT صريح** — ولا يُبنى أثرٌ مجهولُ المصدر." >&2; exit 2; }
fi
SHORT="${SOURCE_COMMIT:0:8}"
BUILD_ID="${BUILD_ID:-build-$(date -u +%Y%m%dT%H%M%SZ)-$SHORT}"

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
	# ══════════════════════════════════════════════════════════════
	# **ولا يُقبَل أثرٌ أجوفُ الهويّة** (٢٠٢٦-٠٩-١٣)
	# ══════════════════════════════════════════════════════════════
	#
	# **ويُسأل الأثرُ نفسُه لا الوسمُ ولا البيان**: `-identity` يخرج
	# بـ٣ إن كان الختمُ فارغاً. **وقِيس ٢٠٢٦-٠٩-١٣: أثرٌ رُقّي إلى
	# الإنتاج بهويّةٍ فارغة** لأنّ سكربتاً عارضاً تجاوز هذا الملفَّ
	# ولم يمرّر `--build-arg`، **ولم يمنعه شيء.**
	if [ "$comp" = "api" ]; then
		verify_identity "$tag" "$SOURCE_COMMIT" "$BUILD_ID"
	fi
	archive_and_manifest "$comp" "$tag" "$(docker image inspect "$tag" --format '{{.Id}}')"
}

# verify_identity **الأثرُ يقول التزامَه وبناءَه — أم يُرفض.**
verify_identity() { # <tag-or-image> <want-commit> <want-build>
	local img="$1" wantc="$2" wantb="$3" out rc
	out="$(docker run --rm --entrypoint /app/api "$img" -identity 2>&1)" && rc=0 || rc=$?
	if [ "${rc:-0}" -ne 0 ]; then
		echo "✗ **أثرٌ بلا هويّة**: $img — **ولا يُرقَّى ما لا يقول ما هو.**" >&2
		echo "$out" | sed 's/^/    /' >&2
		exit 3
	fi
	local gotc gotb
	# **ولا مجموعةَ التقاطٍ في هذا الملفّ** — **محرفُ الهروب يُؤكَل عبر
	# طبقات التحرير فتخرج القيمةُ فارغةً والحارسُ يمرّ كاذباً.** (وقع
	# ٢٠٢٦-٠٩-١٣ مرّتين.) **فيُقطَع النصُّ بأدواتٍ لا تحتاج هروباً.**
	gotc="$(printf '%s' "$out" | grep -o '"source_commit": *"[^"]*"' | cut -d'"' -f4)"
	gotb="$(printf '%s' "$out" | grep -o '"build_id": *"[^"]*"' | cut -d'"' -f4)"
	if [ -n "$wantc" ] && [ "$gotc" != "$wantc" ]; then
		echo "✗ **ختمُ الالتزام لا يطابق المصدر**: في الأثر=$gotc · المنتظَر=$wantc" >&2
		exit 3
	fi
	if [ -n "$wantb" ] && [ "$gotb" != "$wantb" ]; then
		echo "✗ **ختمُ البناء لا يطابق**: في الأثر=$gotb · المنتظَر=$wantb" >&2
		exit 3
	fi
	echo "   ✓ الهويّةُ مختومةٌ في الأثر: ${gotc:0:8} · $gotb"
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
