#!/usr/bin/env bash
# استبقاءُ أرشيفات الإصدار — سياسةٌ واحدةٌ تُقاس وتُختبَر.
#
# ══════════════════════════════════════════════════════════════════════
#  لماذا وُجد هذا الملفّ
# ══════════════════════════════════════════════════════════════════════
#
# وقع ٢٠٢٦-٠٩-٣٠: القرص امتلأ ١٠٠٪ (75G) فسقطت قاعدة التجهيز في
# recovery mode وفشلت هجرة 0166 بـ«No space left on device»، والإنتاج على
# القرص نفسه.
#
# والسبب الجذريّ لم يكن غياب سياسة — بل سياسةٌ تنظر إلى غير موضعها:
#
#   rahalgo-staging-deploy.sh كان يُسند ARTIFACT_DIR بلا export
#   فلا ترثه build-artifact.sh (عمليّةٌ ابنة)، فتهبط إلى افتراضها
#   /srv/rahalgo/artifacts — وهو مجلّد الإنتاج.
#   والتنقيةُ تحرس المسار بـcase /srv/rahalgo-staging/* فلا تراه أبداً.
#
# فتراكم ٢٤٤ أرشيفاً = 28G في مجلّدٍ لا يُنقّى.
#
# ══════════════════════════════════════════════════════════════════════
#  القاعدة
# ══════════════════════════════════════════════════════════════════════
#
#   يُحتفظ بآخر KEEP_N إصداراً (افتراضُه ٥)
#   ويُحتفظ دائماً بكلّ إصدارٍ محميّ — العامل الآن وأيُّ هدفِ رجوع
#   ولا يُحذف أرشيفٌ لإصدارٍ مستعمَل ولو شاخ تاريخُه
#   والحذفُ بمجموعاتٍ لا بملفّات: أرشيفُ الإصدار وبيانُه يمضيان معاً
#
# ══════════════════════════════════════════════════════════════════════
#  ونسخُ القاعدة ليست أرشيفاتِ إصدار
# ══════════════════════════════════════════════════════════════════════
#
# لها سياستُها ولا تدخل هنا بحال. والحارسُ يرفض أيّ مجلّدٍ في مساره
# backup أو backups أو pgdata أو postgres — ويرفض ما اسمُه الأخير ليس
# artifacts. فمن نادى هذه الأداةَ على مجلّد نسخٍ خرج بخطأٍ لا بحذف.
set -euo pipefail

# ما يُعَدّ أرشيفَ إصدار — ولا شيءَ غيره
#
#   rahalgo-api-release-<short>.tar   الصورةُ محفوظة
#   release-api-<short>.json          بيانُها
TAR_GLOB='rahalgo-*-release-*.tar'
MAN_GLOB='release-*-*.json'

die() { printf 'retention: %s\n' "$*" >&2; exit 1; }

# guard_dir يرفض قبل أن يقرأ — ولا يُحذف شيءٌ في مجلّدٍ لم يُصرَّح به
guard_dir() {
	local d="$1"
	case "$d" in
		/*) : ;;
		*) die "المسار ليس مطلقاً: $d" ;;
	esac
	[ -d "$d" ] || die "لا مجلّدَ بهذا المسار: $d"
	# **ونسخُ القاعدة ممنوعةٌ قطعاً** — ولو مُرّرت بالخطأ
	case "$d" in
		*/backup/*|*/backups/*|*backup|*backups|*/pgdata/*|*pgdata|*/postgres/*|*postgres)
			die "هذا مسارُ نسخٍ أو بيانات قاعدة — ولها سياستُها: $d" ;;
	esac
	[ "$(basename "$d")" = artifacts ] || die "اسمُ المجلّد ليس artifacts: $d"
}

# tokens_of يطبع رمزَ الإصدار لكلّ ملفٍّ مؤهَّل مع وقتِ تعديله
#   <mtime> <token> <path>
tokens_of() {
	local d="$1" f base token
	for f in "$d"/$TAR_GLOB "$d"/$MAN_GLOB; do
		[ -f "$f" ] || continue
		base="$(basename "$f")"
		case "$base" in
			*.tar)  token="${base%.tar}";  token="${token##*-release-}" ;;
			*.json) token="${base%.json}"; token="${token##*-}" ;;
			*) continue ;;
		esac
		[ -n "$token" ] || continue
		printf '%s\t%s\t%s\n' "$(stat -c %Y "$f" 2>/dev/null || echo 0)" "$token" "$f"
	done
}

# plan يطبع ما يجب حذفُه — ولا يحذف حرفاً
#   plan <dir> <keep_n> [protected...]
plan() {
	local d="$1" keep="$2"; shift 2
	guard_dir "$d"
	case "$keep" in ''|*[!0-9]*) die "عددُ الاستبقاء ليس رقماً: $keep" ;; esac
	[ "$keep" -ge 1 ] || die "عددُ الاستبقاء أقلُّ من واحد: $keep"

	local rows protected_re=""
	rows="$(tokens_of "$d")"
	[ -n "$rows" ] || return 0

	# الرموزُ المحميّة — تُطابَق باحتواء، فوسمُ صورةٍ قد يحمل سبعةَ أحرفٍ
	# ورمزُ الملفّ ثمانية
	local p
	for p in "$@"; do
		[ -n "$p" ] || continue
		protected_re="$protected_re $p"
	done

	# أحدثُ وقتٍ لكلّ رمز، ثمّ ترتيبٌ تنازليّ
	local keep_tokens
	keep_tokens="$(printf '%s\n' "$rows" | awk -F'\t' '
		{ if ($1 > m[$2]) m[$2] = $1 }
		END { for (t in m) printf "%s\t%s\n", m[t], t }
	' | sort -rn -k1,1 | head -n "$keep" | cut -f2)"

	local t f mt prot q
	while IFS=$'\t' read -r mt t f; do
		[ -n "$f" ] || continue
		# محميٌّ؟ لا يُحذف ولو شاخ
		prot=0
		for q in $protected_re; do
			case "$q" in *"$t"*) prot=1 ;; esac
			case "$t" in *"$q"*) prot=1 ;; esac
		done
		[ "$prot" -eq 1 ] && continue
		# داخلَ نافذة الاستبقاء؟
		if printf '%s\n' "$keep_tokens" | grep -qxF "$t"; then continue; fi
		# ولا يُطبَع إلّا ما كان تحتَ المجلّد نفسِه
		case "$f" in "$d"/*) printf '%s\n' "$f" ;; esac
	done <<< "$rows"
}

# free_mb المساحةُ الحرّةُ على نظام ملفّات المجلّد، بالميغابايت
free_mb() { df -Pk "$1" 2>/dev/null | awk 'NR==2{printf "%d", $4/1024}'; }

# apply يحذف ما خطّطه — ويسجّل المساحةَ قبل وبعد
#   apply <dir> <keep_n> [protected...]
apply() {
	local d="$1" keep="$2"; shift 2
	guard_dir "$d"
	local before after n=0 f
	before="$(free_mb "$d")"
	printf 'retention: قبل = %sM حرّة في %s\n' "$before" "$d" >&2
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		printf 'retention: حذف %s\n' "$(basename "$f")" >&2
		rm -f -- "$f"
		n=$((n + 1))
	done < <(plan "$d" "$keep" "$@")
	after="$(free_mb "$d")"
	printf 'retention: بعد = %sM حرّة · حُذف %d ملفّاً · تحرّر %sM\n' \
		"$after" "$n" "$((after - before))" >&2
}

# guard يسقط إن كانت المساحةُ تحتَ الحدّ — فلا يُبنى ما لا مكانَ له
#   guard <dir> <min_free_mb>
guard() {
	local d="$1" min="$2" have
	[ -d "$d" ] || d="$(dirname "$d")"
	have="$(free_mb "$d")"
	printf 'retention: المساحةُ الحرّة %sM · الحدُّ %sM\n' "$have" "$min" >&2
	[ "$have" -ge "$min" ] || die "مساحةٌ حرجة: ${have}M دونَ الحدّ ${min}M — ولا يُنشَر على قرصٍ ممتلئ"
}

case "${1:-}" in
	plan)  shift; plan "$@" ;;
	apply) shift; apply "$@" ;;
	guard) shift; guard "$@" ;;
	*) die "الاستعمال: retention.sh plan|apply <dir> <keep_n> [protected...] | guard <dir> <min_free_mb>" ;;
esac
