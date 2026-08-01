# سكربتات التشغيل

| الملف | ماذا يفعل |
|---|---|
| [`wipe.sql`](wipe.sql) | **يمسح كل الحسابات وما يخصّها** من قاعدة التطوير، ويُبقي بنية المنصة (التصنيفات، مناطق التوصيل، الإعدادات، الأدوار، أكواد الخصم) |
| [`e2e/cycle.py`](e2e/) | يمشي بالمنصة ستّاً وأربعين خطوة من أول لحظة إلى آخرها |

## بدايةٌ نظيفة

```bash
# ١) نسخةٌ احتياطية أولاً — الحذف لا رجعة فيه
docker exec rahalgo-postgres pg_dump -U rahalgo -d rahalgo -Fc -f /tmp/before-wipe.dump

# ٢) المسح (داخل معاملة واحدة: إمّا كلُّه أو لا شيء)
docker cp scripts/wipe.sql rahalgo-postgres:/tmp/wipe.sql
docker exec rahalgo-postgres psql -U rahalgo -d rahalgo -v ON_ERROR_STOP=1 -f /tmp/wipe.sql

# ٣) ملفات الوسائط اليتيمة على القرص
rm -rf backend/uploads

# ٤) بابٌ للدخول — قاعدةٌ بلا أدمن قاعدةٌ لا يُدخَل إليها
cd backend && go run ./cmd/seed -staff
```

**ولا يُشغَّل على إنتاج.** لا حارس في `wipe.sql` نفسه — هو نصُّ SQL خام لا
يعرف بيئته، وحمايتُه أن يُنسخ إلى الحاوية يدوياً في كل مرّة.
