-- ══════════════════════════════════════════════════════════════════════
-- **عناوينُ المتاجر القائمة: الشارعُ أوّلاً** (قرارُ المالك ٢٠٢٦-١٠-١٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- «شارع تل ابيض - الرقة - مركز المدينة» — وكان العنوانُ يُبنى عند تحويل المرشَّح
-- «المنطقة، المدينة — التفصيل» فيُقرأ مقلوباً، **وتتكرّر المدينةُ** إن كتبها المندوبُ
-- في تفصيله («مركز الرقة، الرقة — الرقة — شارع تل أبيض»). البناءُ صُحّح
-- (`leadAddress`)، **وهذه للمتاجر التي بُنيت قبله.**
--
-- **ولا يُمسّ إلّا ما طابق الشكلَ القديمَ بعينه**: منطقةٌ موجودةٌ في `districts`،
-- ثمّ «، »، ثمّ مدينةٌ موجودةٌ في `governorates`، ثمّ « — ». **وما كتبه أحدٌ بيده
-- بغير هذا الشكل يبقى كما هو.**
WITH old AS (
    SELECT m.id, regexp_match(m.address_text, '^(.+?)، (.+?) — (.*)$') AS p
      FROM merchants m
     WHERE m.address_text LIKE '%، % — %'
), fixed AS (
    SELECT o.id,
           concat_ws(' - ',
               CASE WHEN btrim(o.p[3]) = o.p[2] THEN NULL
                    ELSE NULLIF(btrim(regexp_replace(btrim(o.p[3]), '^' || o.p[2] || '\s*[—،-]\s*', '')), '') END,
               o.p[2], o.p[1]) AS addr
      FROM old o
     WHERE o.p IS NOT NULL
       AND EXISTS (SELECT 1 FROM districts d WHERE d.name = o.p[1])
       AND EXISTS (SELECT 1 FROM governorates g WHERE g.name = o.p[2])
)
UPDATE merchants m
   SET address_text = f.addr, updated_at = now()
  FROM fixed f
 WHERE m.id = f.id AND f.addr <> '';
