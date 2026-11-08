-- ══════════════════════════════════════════════════════════════════════
-- **عمولةُ متجرنا على طلب الواتساب** — قرار المالك ٢٠٢٦-١١-٠٨
-- ══════════════════════════════════════════════════════════════════════
--
-- طلبُ واتسابٍ من متاجرنا بثمنٍ مكتوب: **عند التسليم تُخصم عمولةُ المتجر على الثمن من محفظة صاحبه**
-- إلى الخزينة (`orders/whatsapp_commission.go`). **وما لم يكفِه المتاحُ دينٌ عليه** بسببٍ جديد
-- `whatsapp_order_commission` — ولا محفظةَ سالبة.
--
-- **ولا يُمسّ `orders_whatsapp_shape`**: الخصمُ قيدُ `order_payment` بمرجع الطلب بلا `wallet_paid`
-- (الأجرةُ نقدٌ كلُّها)، والخزينةُ تقرأ القيدَ لا العمود.
ALTER TABLE financial_obligations DROP CONSTRAINT IF EXISTS financial_obligations_cause_check;
ALTER TABLE financial_obligations ADD CONSTRAINT financial_obligations_cause_check
    CHECK (cause IN (
        'refund_merchant_earning',
        'refund_rep_commission',
        'returned_goods',
        'legacy_opening',
        'merchant_delivery_fee',
        'penalty_remainder',
        'dispute_remainder',
        'failed_order_driver',
        'delivery_failed_recipient',
        'refund_rep_share',
        'waived_rep_commission',
        'undelivered_order',
        'order_edit',
        'whatsapp_order_commission'));
