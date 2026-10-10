#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
#  **النسخ الاحتياطيّ الليليّ للإنتاج** (قرارُ المالك ٢٠٢٦-١٠-١٠)
# ══════════════════════════════════════════════════════════════════════
#
# **كان كلُّ شيءٍ على قرصٍ واحدٍ بلا نسخة** — كشفه راصدُ المنصّة
# (`BACKUPS_DIR` فارغ). **ومنصّةٌ فيها محافظُ ودفترُ مالٍ لا تُطلَق هكذا.**
#
# ثلاثةُ أشياءَ كلَّ ليلة، في `/srv/rahalgo/backups`:
#   db-<وقت>.dump        القاعدةُ كاملةً (`pg_dump -Fc`) — والمنصّةُ شغّالة
#   uploads-<وقت>.tgz    الصورُ والوثائق (حجمُ `rahalgo_uploads`)
#   secrets-<وقت>.tgz    `.env` والأسرار — **وبصلاحيّة ٦٠٠ لا يقرؤها سوى root**
#
# **والنسخةُ تُفحص قبل أن تُعتمد**: `pg_restore -l` يقرأ فهرسَها، **ومن
# لم يُقرأ فهرسُه حُذف وسقط السكربت** — فلا تبقى نسخةٌ تالفةٌ تُحسب سليمة.
# **والكتابةُ إلى ملفٍّ مؤقّتٍ ثمّ نقلٌ** — فنسخةٌ انقطعت لا تحمل اسمَ نسخة.
#
# **ويُبقي سبعةَ أيّام** — وما أقدمُ منها يُحذف، **ومن أنماطه وحدَها.**
#
# التثبيت (مرّةً، root):
#   install -m 700 backup.sh /usr/local/bin/rahalgo-backup
#   echo '0 0 * * * root /usr/local/bin/rahalgo-backup >> /var/log/rahalgo-backup.log 2>&1' > /etc/cron.d/rahalgo-backup
# (منتصفُ الليل UTC = الثالثةُ فجراً بتوقيت دمشق.)
set -euo pipefail
umask 077

DIR=${BACKUPS_DIR:-/srv/rahalgo/backups}
PG=${PG_CONTAINER:-rahalgo-postgres-1}
UPLOADS_VOL=${UPLOADS_VOLUME:-rahalgo_uploads}
DEPLOY=${DEPLOY_DIR:-/srv/rahalgo/deploy}
SECRETS=${SECRETS_DIR:-/srv/rahalgo/secrets}
KEEP_DAYS=${KEEP_DAYS:-7}
TS=$(date -u +%Y%m%dT%H%MZ)

mkdir -p "$DIR"
chmod 700 "$DIR"
echo "== $(date -u +%FT%TZ) نسخة $TS"

# ── القاعدة ─────────────────────────────────────────────────────────
tmp="$DIR/.db-$TS.part"
docker exec "$PG" pg_dump -U rahalgo -d rahalgo -Fc >"$tmp"
if ! docker exec -i "$PG" pg_restore -l >/dev/null <"$tmp"; then
	rm -f "$tmp"
	echo "!! نسخةُ القاعدة تالفة — لم تُعتمد" >&2
	exit 1
fi
mv "$tmp" "$DIR/db-$TS.dump"
echo "db      $(du -h "$DIR/db-$TS.dump" | cut -f1)"

# ── الصور ───────────────────────────────────────────────────────────
src=$(docker volume inspect -f '{{.Mountpoint}}' "$UPLOADS_VOL")
tmp="$DIR/.uploads-$TS.part"
tar -czf "$tmp" -C "$src" .
mv "$tmp" "$DIR/uploads-$TS.tgz"
echo "uploads $(du -h "$DIR/uploads-$TS.tgz" | cut -f1)"

# ── الأسرار ─────────────────────────────────────────────────────────
tmp="$DIR/.secrets-$TS.part"
tar -czf "$tmp" -C / "${DEPLOY#/}/.env" "${SECRETS#/}"
mv "$tmp" "$DIR/secrets-$TS.tgz"
echo "secrets $(du -h "$DIR/secrets-$TS.tgz" | cut -f1)"

# ── الاحتفاظ ────────────────────────────────────────────────────────
find "$DIR" -maxdepth 1 -type f \( -name 'db-*.dump' -o -name 'uploads-*.tgz' -o -name 'secrets-*.tgz' \) \
	-mtime +"$KEEP_DAYS" -print -delete
find "$DIR" -maxdepth 1 -type f -name '.*.part' -mmin +120 -delete

echo "ok $TS"
