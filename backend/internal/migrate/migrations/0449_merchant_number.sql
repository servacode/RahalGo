-- ══════════════════════════════════════════════════════════════════════
-- **رقمُ المتجر — قصيرٌ يُقرأ، للوحة وحدَها** (طلبُ المالك ٢٠٢٦-١٠-١٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- **متجران باسمٍ عربيٍّ واحد** («القصور» و«القصور») **لا يُفرَّق بينهما في
-- اللوحة إلّا بفتح كلٍّ منهما.** فصار لكلّ متجرٍ رقمٌ يُكتب بجانب اسمه:
-- «القصور #M-1045» و«القصور #M-1062».
--
-- **وهو للموظّفين لا للزبون** — لا يُرسَل في أيّ ردٍّ لتطبيقٍ ولا لصفحةٍ عامّة.
--
-- **ويبدأ من ١٠٠١** كرقم الطلب، **والقائمُ يُرقَّم بترتيب ولادته**
-- (`created_at` ثمّ المعرّفُ لكسر التعادل)، **والجديدُ يأخذ التاليَ من
-- التسلسل وحدَه** — لا شيفرةَ تكتبه، فلا بابَ ينساه.

CREATE SEQUENCE IF NOT EXISTS merchants_number_seq START WITH 1001;

ALTER TABLE merchants ADD COLUMN number bigint;

-- **الترقيمُ بالترتيب لا بـ`nextval` داخل استعلام**: ترتيبُ استدعاء
-- التسلسل في `UPDATE` غيرُ مضمون، **والرقمُ هنا يجب أن يقول من وُلد أوّلاً.**
UPDATE merchants m
   SET number = 1000 + r.rn
  FROM (SELECT id, row_number() OVER (ORDER BY created_at, id) AS rn FROM merchants) r
 WHERE r.id = m.id;

-- **والتسلسلُ يُكمل بعد آخر رقمٍ أُعطي** — وإلّا اصطدم أوّلُ متجرٍ جديدٍ بالقائم.
SELECT setval('merchants_number_seq', COALESCE((SELECT max(number) FROM merchants), 1000) + 1, false);

ALTER TABLE merchants
    ALTER COLUMN number SET DEFAULT nextval('merchants_number_seq'),
    ALTER COLUMN number SET NOT NULL,
    ADD CONSTRAINT merchants_number_key UNIQUE (number);

ALTER SEQUENCE merchants_number_seq OWNED BY merchants.number;
