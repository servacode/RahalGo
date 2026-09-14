-- ══════════════════════════════════════════════════════════════════════
-- **هدفُ الاشتراك مكانٌ لا خليّة** (`SI`، ٢٠٢٦-٠٩-١٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- # العطبُ الذي كان
--
-- **وكان التفرّدُ بالخليّة لكلّ نوع** — **والخليّةُ ≈ ١٫١ كم ودمشقُ
-- نصفُ قطرها ٢٥ كم**: **فألفُ خليّةٍ وأكثرُ داخلَ مدينةٍ واحدة.**
--
-- **فمن ضغط «أخبرني عند توفّر الخدمة في دمشق» من عنوانه في المزّة ثمّ
-- من عنوان عمله في باب توما صار هدفَين** — **ويُخبَر مرّتين يومَ
-- تُطلَق دمشق.**
--
-- **والزرُّ قال «في دمشق» لا «في هذه الخليّة»** — **فهويّةُ الاشتراك
-- يجب أن تكون ما وعد به الزرُّ.**
--
-- # والنوعان يفترقان في الهويّة
--
--	service_interest + مدينةٌ معروفة  ⇒  هدفُه المدينة
--	service_interest + موضعٌ مجهول    ⇒  هدفُه الخليّة (لا مكانَ له)
--	coverage_request                 ⇒  هدفُه الخليّة **دائماً**
--
-- **وطلبُ التغطية مكانيٌّ بطبعه** — **«أيُّ حيٍّ خارجَ نطاق الرقّة عليه
-- أكبرُ طلب؟» سؤالُ خلايا لا سؤالُ مدن.** **ولو جُمع بالمدينة لصار
-- «الرقّة» صفّاً واحداً لا يقول أين يُوسَّع.**
--
-- # ولمَ عمودٌ صريحٌ لا عمودٌ مولَّد
--
-- **والمولَّدُ يُعاد حسابُه إن تبدّل `city_id`** — **وصفُّ خليّةٍ حُلَّت
-- مدينتُه لاحقاً يقفز إلى هويّةِ مدينةٍ قد تكون مشغولةً بصفٍّ آخر**،
-- **فيسقط التحديثُ بخرقِ تفرّد.**
--
-- **والصريحُ يُكتب مرّةً ولا يتبدّل** — **وهويّةٌ ثابتةٌ هي المقصودة.**

ALTER TABLE coverage_requests ADD COLUMN IF NOT EXISTS target_key text;

-- ── الملءُ للصفوف القائمة ─────────────────────────────────────────────
UPDATE coverage_requests
   SET target_key = CASE
       WHEN kind = 'service_interest' AND city_id IS NOT NULL
            THEN 'city:' || city_id::text
       ELSE 'cell:' || cell_y::text || ',' || cell_x::text
       END
 WHERE target_key IS NULL;

-- ── وتُدمَج المكرّراتُ قبل أن يُفرَض التفرّد ──────────────────────────
--
-- **وصفّان لهدفٍ واحدٍ قد يكونان موجودَين** — **كتبهما التفرّدُ القديمُ
-- بالخليّة.** **فيُبقى أقدمُهما وتُجمَع مرّاتُه**، **ولا تُمحى شدّةُ
-- الطلب.**
--
-- **والسريانُ يُجمَع بـ«أيٌّ منها سارٍ»** — **ومن ألغى واحداً وأبقى
-- آخرَ ما زال مشترِكاً.**
WITH ranked AS (
    SELECT id, user_id, kind, target_key, requests, active, created_at,
           first_value(id) OVER w AS keep_id
    FROM coverage_requests
    WHERE user_id IS NOT NULL
    WINDOW w AS (PARTITION BY user_id, kind, target_key ORDER BY created_at, id)
), merged AS (
    SELECT keep_id,
           sum(requests) AS total,
           bool_or(active) AS any_active,
           max(created_at) AS last_at
    FROM ranked
    GROUP BY keep_id
    HAVING count(*) > 1
)
UPDATE coverage_requests r
   SET requests = m.total,
       active = m.any_active,
       last_seen_at = GREATEST(r.last_seen_at, m.last_at)
  FROM merged m
 WHERE r.id = m.keep_id;

DELETE FROM coverage_requests r
 USING (
    SELECT id, first_value(id) OVER (
               PARTITION BY user_id, kind, target_key ORDER BY created_at, id) AS keep_id
    FROM coverage_requests
    WHERE user_id IS NOT NULL
 ) d
 WHERE r.id = d.id AND d.id <> d.keep_id;

-- ── والتفرّدُ بالهدف لا بالخليّة ──────────────────────────────────────
DROP INDEX IF EXISTS coverage_requests_dedupe_idx;

CREATE UNIQUE INDEX IF NOT EXISTS coverage_requests_target_idx
    ON coverage_requests (user_id, kind, target_key)
 WHERE user_id IS NOT NULL;

-- **وسؤالُ الدفعة الثامنة**: «من يُخبَر يومَ تُطلَق هذه المدينة؟»
CREATE INDEX IF NOT EXISTS coverage_requests_target_active_idx
    ON coverage_requests (kind, target_key) WHERE active;

-- ── ولا صفَّ بلا هويّة ────────────────────────────────────────────────
--
-- **وصفٌّ بهويّةٍ فارغةٍ لا يُلغى ولا يُدمَج فيه جديد** — **ويبقى
-- يتيماً يُعَدّ في الكثافة ولا يُخبَر صاحبُه.** **فالعمودُ مُلزِم.**
ALTER TABLE coverage_requests ALTER COLUMN target_key SET NOT NULL;
