-- ══════════════════════════════════════════════════════════════════════
-- «أمانة» في قائمة بوت الواتساب (قرار المالك ٢٠٢٦-١١-٢٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- الخيار السابع: إرسال أمانة أو استلامها — أسئلة واحداً واحداً ثم طلب معلّق بانتظار موافقة الإدارة.
-- حالُ المحادثة الجديدة `amanah` تُضاف إلى قيد `wa_bot_chats.state`، وإلى فهرس التذكير الجزئيّ
-- (التذكير بعد ٣ دقائق ثم الإسقاط يشملها كما يشمل الطلب والتسجيل).
--
-- ولا عمود جديد على الطلب: الأمانة تُكتب بأعمدتها القائمة (`wa_amanah` و`amanah_direction` و
-- `collect_amount`/`amount_currency`).
--
-- **والأمانةُ بمبلغ صارت مسموحة** (قرار المالك ٢٠٢٦-١١-٢٧، يلغي «لا ثمن في الأمانة» ٢٠٢٦-١١-١١):
-- الكابتن يدفع المبلغ عند الاستلام ويقبضه عند التسليم، كثمن البضاعة في طلب الواتساب تماماً
-- (`collect_amount` خارج الدفتر). فيُرفع شرط `collect_amount IS NULL` من قيد `orders_wa_amanah_ck`
-- ويبقى أنّ الأمانة طلبُ واتساب وحده.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_wa_amanah_ck;
ALTER TABLE orders ADD CONSTRAINT orders_wa_amanah_ck
    CHECK (NOT wa_amanah OR origin = 'whatsapp');

ALTER TABLE wa_bot_chats DROP CONSTRAINT IF EXISTS wa_bot_chats_state_check;
ALTER TABLE wa_bot_chats ADD CONSTRAINT wa_bot_chats_state_check
    CHECK (state IN ('idle', 'menu', 'closed', 'order', 'amanah', 'track', 'admin', 'signup', 'done'));

DROP INDEX IF EXISTS wa_bot_chats_flow_idx;
CREATE INDEX IF NOT EXISTS wa_bot_chats_flow_idx ON wa_bot_chats (step_at)
    WHERE state IN ('order', 'amanah', 'signup');
