-- ٠٢٠٠ · «السوق» في لوحة الإدارة — قراراتُ المالك ٢٠٢٦-١٠-٠٤
--
-- # ١ · لا موافقةَ على الصنف
--
-- «ما في داعي للموافقة على الصنف أساساً»: يُضيفه المتجرُ فيظهر في السوق
-- فوراً، **وتُشال المراجعةُ وإعدادُها من كلّ مكان.**
--
-- **والعمودُ `approved` يبقى صادقاً لا حارساً**: كلُّ صفٍّ اليومَ مُقَرّ،
-- ولا بابَ يكتب فيه `false` بعد اليوم. **ولم يُحذف** لأنّ قراءاتٍ في
-- الطلبات والتحويل تذكره — **وحذفُه يمسّ حزماً أخرى في دفعةٍ واحدة.**
UPDATE menu_items
   SET approved = true, review_note = ''
 WHERE NOT approved OR review_note <> '';

ALTER TABLE menu_items ALTER COLUMN approved SET DEFAULT true;

-- **وطابورُ المراجعة ذهب** — فلا فهرسَ له.
DROP INDEX IF EXISTS menu_items_pending_idx;

-- **ومفتاحُه ذهب من الكتالوج** — فلا تبقى قيمتُه المحفوظةُ يتيمة.
DELETE FROM app_settings WHERE key = 'merchants.menu_requires_approval';

-- # ٢ · «المضافُ حديثاً» — آخرُ فتحٍ للسوق لكلّ موظّف
--
-- «لازم أعرف الأصناف المضافة حديثاً» — **وعدّادٌ في القائمة الجانبيّة
-- يختفي حين يُفتح.** **والعدُّ لكلّ موظّفٍ وحدَه**: من فتح السوق صُفِّر
-- عدّادُه هو، **ولا يُخفى الجديدُ عن زميلٍ لم يره.**
CREATE TABLE IF NOT EXISTS admin_market_seen (
    user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    seen_at timestamptz NOT NULL DEFAULT now()
);

-- **والقراءةُ الساخنة: «كم صنفاً أُضيف بعد لحظة كذا»** — مع كلّ صفحة.
CREATE INDEX IF NOT EXISTS menu_items_created_idx ON menu_items (created_at DESC);

-- # ٣ · مفتاحُ اسم القسم — «حلويات» = «حلويّات»
--
-- **والمقارنةُ بعد التطبيع**: تُرفع الحركاتُ والشدّةُ والتطويل، **وتوحَّد
-- الهمزاتُ على الألف**، والتاءُ المربوطةُ هاءً، والألفُ المقصورةُ ياءً،
-- **والمسافاتُ المتكرّرة مسافةً واحدة.**
--
-- **ومصدرُها هنا وحدَه** — المحرّكُ يسأل القاعدةَ بها ولا يعيد كتابتَها،
-- **وتطبيعان في موضعين يفترقان يوماً.**
CREATE OR REPLACE FUNCTION section_name_key(t text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT lower(btrim(regexp_replace(
        translate(
            regexp_replace(t, '[ً-ٰٟـ‌-‏]', '', 'g'),
            'أإآٱةىؤئ', 'ااااهيوي'),
        '\s+', ' ', 'g')))
$$;

-- # ٤ · الأقسامُ المكرّرةُ تُدمج — ويبقى واحد
--
-- «الأقسامُ المكرّرةُ بالاسم نفسه تُحذف ويبقى واحد». **والباقي**: الفعّالُ
-- أوّلاً، ثمّ الأكثرُ أصنافاً، ثمّ الأقدم. **وأصنافُ المكرَّر تُنقل إليه**
-- (لا يُحذف صنف)، **وإعلانُ المتاجر أنّها تبيع فيه يُنقل معها**، **وصورتُه
-- تُورَث إن لم تكن للباقي صورة.**
--
-- **ودالّةٌ لا سطرٌ واحد** — ليُفحص أثرُها في اختبارٍ يبني التكرارَ ثمّ
-- يُرجعه.
CREATE OR REPLACE FUNCTION merge_duplicate_platform_sections() RETURNS integer
LANGUAGE plpgsql AS $$
DECLARE
    r record;
    n integer := 0;
BEGIN
    FOR r IN
        WITH ranked AS (
            SELECT ps.id,
                   first_value(ps.id) OVER (
                       PARTITION BY section_name_key(ps.name)
                       ORDER BY ps.active DESC,
                                (SELECT count(*) FROM menu_items i
                                  WHERE i.platform_section_id = ps.id) DESC,
                                ps.created_at, ps.id) AS keeper
              FROM platform_sections ps
        )
        SELECT id, keeper FROM ranked WHERE id <> keeper
    LOOP
        UPDATE menu_items SET platform_section_id = r.keeper
         WHERE platform_section_id = r.id;
        INSERT INTO store_sections (store_id, section_id)
        SELECT store_id, r.keeper FROM store_sections WHERE section_id = r.id
        ON CONFLICT DO NOTHING;
        UPDATE platform_sections k
           SET image_media_id = d.image_media_id
          FROM platform_sections d
         WHERE k.id = r.keeper AND d.id = r.id
           AND k.image_media_id IS NULL AND d.image_media_id IS NOT NULL;
        DELETE FROM platform_sections WHERE id = r.id;
        n := n + 1;
    END LOOP;
    RETURN n;
END
$$;

SELECT merge_duplicate_platform_sections();

-- # ٥ · ولا يُنشأ قسمٌ باسمٍ موجود — ولو اختلف التشكيل
--
-- **والفهرسُ هو الحارسُ الأخير** — المحرّكُ يسأل قبل الكتابة ليقول السببَ
-- بالعربيّة، **وطلبان متزامنان يمرّان من السؤال معاً فيردّهما الفهرس.**
CREATE UNIQUE INDEX IF NOT EXISTS platform_sections_name_key_uq
    ON platform_sections (section_name_key(name));
