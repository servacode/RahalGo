#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  **فحصُ الاسترجاع — نسخةٌ لم تُسترجَع ليست نسخة** (٢٠٢٦-١٠-١٠)
# ══════════════════════════════════════════════════════════════════════
#
# يأخذ أحدثَ `db-*.dump` ويسترجعه في حاويةِ بوستغرس **مؤقّتةٍ منفصلة**
# (بالصورة عينِها، بلا منفذٍ ولا حجم) — **ولا يمسّ القاعدةَ الحيّة إلّا
# قراءةً**. ثمّ يقارن عددَ الصفوف جدولاً جدولاً:
#
#   **المسترجَعُ ≤ الحيّ** في كلّ جدول — فالحيّةُ تكتب بعد النسخة.
#   **جدولٌ ناقصٌ أو أكبرُ في النسخة** ⇒ فشل.
#
# **وتُحذف الحاويةُ المؤقّتةُ في كلّ حال** (`trap`).
set -euo pipefail

DIR=${BACKUPS_DIR:-/srv/rahalgo/backups}
PG=${PG_CONTAINER:-rahalgo-postgres-1}
TMPC=rahalgo-restore-test

dump=$(ls -1t "$DIR"/db-*.dump 2>/dev/null | head -1)
[ -n "$dump" ] || { echo "!! لا نسخة في $DIR" >&2; exit 1; }
echo "== فحص $dump"

img=$(docker inspect -f '{{.Config.Image}}' "$PG")
docker rm -f "$TMPC" >/dev/null 2>&1 || true
trap 'docker rm -f "$TMPC" >/dev/null 2>&1 || true' EXIT
docker run -d --name "$TMPC" -e POSTGRES_USER=rahalgo -e POSTGRES_PASSWORD=verify -e POSTGRES_DB=rahalgo "$img" >/dev/null
# **وصورةُ بوستغرس تُقلع خادماً مؤقّتاً للتهيئة ثمّ تعيد تشغيله** — و`pg_isready`
# يجيب «نعم» للمؤقّت فيُقطع الاسترجاعُ في منتصفه. **فيُنتظر سطرُ انتهاء التهيئة.**
for _ in $(seq 1 120); do
	docker logs "$TMPC" 2>&1 | grep -q "PostgreSQL init process complete" && break
	sleep 1
done
for _ in $(seq 1 60); do
	docker exec "$TMPC" pg_isready -U rahalgo -d rahalgo >/dev/null 2>&1 && break
	sleep 1
done

# **وفي قاعدةٍ نظيفةٍ من `template0`** — صورةُ PostGIS تخلق إضافاتها (ومنها مخطّطُ
# `tiger`) في القاعدة الافتراضيّة، **فيسقط الاسترجاعُ عندها بـ«موجودٌ أصلاً».**
docker exec "$TMPC" createdb -U rahalgo -T template0 verify
docker exec -i "$TMPC" pg_restore -U rahalgo -d verify --no-owner --exit-on-error <"$dump"
echo "استُرجعت"

counts() {
	docker exec "$1" psql -U rahalgo -d "$2" -At -F' ' -c "
		SELECT format('SELECT %L, count(*) FROM %I.%I', table_name, table_schema, table_name)
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY 1" |
		while read -r q; do docker exec "$1" psql -U rahalgo -d "$2" -At -F' ' -c "$q"; done
}

live=$(counts "$PG" rahalgo)
rest=$(counts "$TMPC" verify)
bad=0
tables=0
while read -r t n; do
	tables=$((tables + 1))
	r=$(printf '%s\n' "$rest" | awk -v t="$t" '$1==t{print $2}')
	if [ -z "$r" ]; then
		echo "!! جدول ناقص في النسخة: $t"; bad=1
	elif [ "$r" -gt "$n" ]; then
		echo "!! $t: النسخة $r > الحيّ $n"; bad=1
	elif [ "$r" -lt "$n" ]; then
		echo "   $t: النسخة $r · الحيّ $n (كُتب بعد النسخة)"
	fi
done <<<"$live"

echo "الجداول: $tables"
for t in users orders merchants menu_items wallets wallet_transactions; do
	printf '%s\n' "$rest" | awk -v t="$t" '$1==t{printf "   %-22s %s\n", $1, $2}'
done
if [ "$bad" -ne 0 ]; then
	echo "RESTORE = FAIL"; exit 1
fi
echo "RESTORE = PASS"
