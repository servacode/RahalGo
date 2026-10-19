-- ══════════════════════════════════════════════════════════════════════
-- **ساعي الأمين على كمبيوتر المول — حالُه و«مزامنة الآن»** (قرارُ المالك ٢٠٢٦-١٠-١٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- **الساعي يسأل كلَّ دقيقة** (`/public/price-sync/poll`) — فيُعرف أنّه حيّ، **ويُقال له
-- «زامن الآن» إن ضغطها صاحبُ المتجر** في لوحته. ولا يُدفع إليه شيءٌ من الخادم: كمبيوترُ
-- المول خلفَ شبكته، **والسؤالُ منه لا إليه.**
--
--   price_sync_agent_seen_at    آخرُ سؤالٍ من الساعي — «شغّال، آخرُ اتّصالٍ قبل دقيقتين»
--   price_sync_agent_version    نسختُه — لنعرف من يحتاج تحديثاً
--   price_sync_requested_at     «مزامنة الآن» — ويُمحى حين يزامن الساعي
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_agent_seen_at timestamptz;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_agent_version text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_requested_at timestamptz;
