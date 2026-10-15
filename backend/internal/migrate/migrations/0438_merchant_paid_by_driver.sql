-- ══════════════════════════════════════════════════════════════════════
-- **السائقُ يدفع للمتجر عند الاستلام** (قرارُ المالك ٢٠٢٦-١٠-١٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- كان مستحقُّ المتجر «نقداً» يُقيَّد دَيناً (`cash_due`) يدفعه الأدمنُ لاحقاً من
-- درج المكتب. **والواقعُ أنّ السائقَ يدفعه للمتجر عند الاستلام** من سلفةٍ يعطيه
-- إيّاها مديرُ المكتب من جيبه (والسلفةُ خارجَ النظام). **فالتسويةُ تصير مدفوعةً
-- لحظةَ نشأتها**، ويُعرَف من دفع:
--
--   office  الأدمنُ دفع من درج المكتب (`MarkCashSettlementPaid`) — وله سطرٌ في الصندوق
--   driver  السائقُ دفع عند الاستلام — **ولا سطرَ في الدرج**: النقدُ لم يمرّ به
--
-- **و`paid_by` يبقى «من دفع»** (الأدمنُ أو السائق) — وهذا العمودُ «من أين دُفع».
ALTER TABLE merchant_settlements
    ADD COLUMN IF NOT EXISTS paid_via text
    CHECK (paid_via IN ('office', 'driver'));

-- **والمدفوعُ قبل اليوم دفعه الأدمنُ من الدرج** — لا سائقَ دفع شيئاً قبل هذا القرار.
UPDATE merchant_settlements SET paid_via = 'office'
 WHERE state = 'cash_paid' AND paid_via IS NULL;

-- **ومدفوعٌ بلا جهةٍ خرقٌ تحرسه القاعدة.**
ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_paid_via_ck;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_paid_via_ck
    CHECK (state <> 'cash_paid' OR paid_via IS NOT NULL);
