#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
# **تنظيفُ صور الإصدارات القديمة — يومياً** (قرارُ المالك ٢٠٢٦-١٠-٠٧: «ننظّف كلّشي بشكل صحيح
# ليصير قوي سيرفر الإنتاج وسريع»)
# ══════════════════════════════════════════════════════════════════════
#
# كلُّ رفعٍ على التجهيز يبني صورتين (api + web، ~1.5GB) ولا يحذف القديم — **فامتلأ القرصُ مرّتين
# في يومين** (٨٣٪)، وأوقف الرفعَ وأبطأ القاعدة.
#
# يبقى: **كلُّ صورةٍ تعمل الآن** (إنتاجاً أو تجهيزاً) + **أحدثُ ثلاثة إصدارات** للرجوع.
# يُحذف: ما سواها من `rahalgo-(api|web):release-*`، والصورُ المعلّقة، وذاكرةُ البناء.
# **ولا يمسّ الأحجام ولا الحاويات ولا قواعد البيانات أبداً.**
#
# التثبيت (مرّةً، root): cp image-prune.sh /etc/cron.daily/rahalgo-image-prune && chmod +x …
set -u
KEEP_NEWEST=3
running="$(docker ps --format '{{.Image}}' | grep -oE 'release-[0-9a-f]+' | sort -u)"
newest="$(docker images --format '{{.CreatedAt}}\t{{.Tag}}' 'rahalgo-api' \
	| grep -E 'release-' | sort -r | head -n "$KEEP_NEWEST" | cut -f2)"
keep="$(printf '%s\n%s\n' "$running" "$newest" | sort -u)"
n=0
for img in $(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^rahalgo-(api|web):release-'); do
	tag="${img##*:}"
	if printf '%s\n' "$keep" | grep -qxF "$tag"; then continue; fi
	docker rmi "$img" >/dev/null 2>&1 && n=$((n + 1))
done
docker image prune -f >/dev/null 2>&1
docker builder prune -f >/dev/null 2>&1
journalctl --vacuum-size=200M >/dev/null 2>&1
logger -t rahalgo-image-prune "removed=$n kept=$(printf '%s' "$keep" | tr '\n' ' ') $(df -h / | tail -1 | awk '{print "disk="$5}')"
