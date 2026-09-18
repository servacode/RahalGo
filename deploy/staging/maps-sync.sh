#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  آثارُ الخرائط للتجهيز — نسخةٌ مستقلّة ونمطٌ يسمّي مضيفَ التجهيز
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-٠٩-١٩ · `ADMIN-STAGING-PRIMARY-PARITY.md` G1.)
#
# # ولماذا
#
# **لوحةُ التجهيز كانت بلا خرائط**: `mapStyleUrl` فارغ، **و`maps.rahalgo.com`
# مضيفُ إنتاجٍ لا يسمّيه التجهيز** (`envguard.ProductionHosts`). **فالحلُّ
# خرائطُ للتجهيز نفسِه** — لا إضعافُ الحارس.
#
# **ولا سجلَّ DNS لـ`staging-maps.rahalgo.com`** — **فتُخدَم على مضيف التجهيز
# القائم** تحت `/maps/` (`Caddyfile.staging`).
#
# # ولماذا نسخةٌ لا حاملٌ مشترك
#
# **العزلُ**: خرائطُ الإنتاج تتبدّل بلا أن تمسّ التجهيز، والعكس. **والآثارُ
# مؤرَّخةٌ لا تتبدّل بعد النشر** — فالنسخُ مرّةً يكفي حتّى تُنشَر خريطةٌ جديدة.
#
# # والنمطُ وحدَه يسمّي المضيف
#
# **قِيس ٢٠٢٦-٠٩-١٩**: من كلّ الآثار **ملفٌّ واحدٌ** يذكر `maps.rahalgo.com` —
# `map-resources/1/style.online.json` (النقوشُ والخطوطُ ومصدرُ `pmtiles`).
# **فيُعاد كتابتُه في النسخة وحدَها** — **ولا يُمَسّ ملفُّ الإنتاج.**
set -euo pipefail

SRC="${MAPS_SRC:-/srv/rahalgo/maps}"
DST="${STAGING_MAPS_DIR:-/srv/rahalgo-staging/maps}"
FROM="https://maps.rahalgo.com"
TO="${STAGING_MAPS_BASE:-https://staging-api.rahalgo.com/maps}"

# **وما يُخدَم وحدَه يُنسخ** — كما في كتلة الإنتاج (`Caddyfile`).
PARTS=(manifest.json base regions map-resources style sprite)

[ -d "$SRC" ] || { echo "✗ لا آثارَ في $SRC" >&2; exit 2; }
case "$DST" in
	"$SRC"|"$SRC"/*) echo "✗ الهدفُ داخلَ آثار الإنتاج — **ولا يُكتب فيها**" >&2; exit 2 ;;
esac

mkdir -p "$DST"
for p in "${PARTS[@]}"; do
	# **`--delete` داخلَ الجزء وحدَه** — فلا يبقى في التجهيز ما حُذف من المصدر.
	if [ -d "$SRC/$p" ]; then
		rsync -a --delete "$SRC/$p/" "$DST/$p/"
	else
		rsync -a "$SRC/$p" "$DST/$p"
	fi
done

STYLE="$DST/map-resources/1/style.online.json"
[ -f "$STYLE" ] || { echo "✗ لا نمطَ في $STYLE" >&2; exit 3; }
sed -i "s#${FROM}#${TO}#g" "$STYLE"

# **والمضغوطُ مسبقاً يحمل المضيفَ القديم** — و`precompressed br gzip` يقدّمه
# على الأصل لمن يقبله، **فيبقى النمطُ القديمُ لأكثر المتصفّحات.**
rm -f "$STYLE.gz" "$STYLE.br"

# ── ولا يُنهى والإنتاجُ مذكور ────────────────────────────────────────
#
# **والملفّاتُ الثنائيّةُ لا تُفتَّش** (`pmtiles` · `png`) — **فيها بيانُ
# الأرض لا عناوين.**
if grep -rIl "maps\.rahalgo\.com" "$DST/manifest.json" "$DST/map-resources" \
	"$DST/style" "$DST/sprite" 2>/dev/null | grep -q .; then
	echo "✗ **بقي مضيفُ الإنتاج في آثار التجهيز**:" >&2
	grep -rIl "maps\.rahalgo\.com" "$DST" >&2
	exit 4
fi

echo "✓ آثارُ الخرائط في $DST · والنمطُ يشير إلى $TO"
du -sh "$DST" | awk '{print "  الحجم = "$1}'
