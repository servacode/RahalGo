-- ══════════════════════════════════════════════════════════════════════
-- **نقدُ المستلِم يرجع له إن فشلت «إحضار» بغير ذنبه** — قرارُ المالك ٢٠٢٦-١٠-٢٧
-- ══════════════════════════════════════════════════════════════════════
--
-- في «إحضار» يدفع المستلِمُ الأجرةَ نقداً للسائق عند الاستلام. **فإن فشلت التوصيلةُ بذنب
-- المنصّة أو السائق** ردّ السائقُ النقدَ للمستلِم بيده — **فلا يبقى في صندوقه ولا يُقيَّد
-- ربحاً للمنصّة.** والردُّ سطرٌ سالبٌ في صندوقه بمرجع التوصيلة (`recipient_cash_returned`)
-- يقابل سطرَ القبض — فيظهر في كشفه، **وتقرؤه الخزينةُ فلا تحسب المقبوضَ مدفوعاً لها.**
--
-- **وذنبُ المستلِم يبقى كما كان**: النقدُ للسائق والمنصّة كأنّها سُلّمت.

ALTER TABLE driver_cash_entries DROP CONSTRAINT IF EXISTS driver_cash_entries_kind_check;
ALTER TABLE driver_cash_entries ADD CONSTRAINT driver_cash_entries_kind_check
    CHECK (kind IN ('order_collection', 'settlement', 'adjustment', 'commission_waived',
                    'recipient_cash_returned'));

-- **ومرّةً واحدةً للتوصيلة** — نهايةٌ واحدةٌ لا تُردّ مرّتين.
CREATE UNIQUE INDEX IF NOT EXISTS driver_cash_entries_recipient_return_once
    ON driver_cash_entries (ref) WHERE kind = 'recipient_cash_returned';
