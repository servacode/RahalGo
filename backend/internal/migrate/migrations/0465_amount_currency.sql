-- ══════════════════════════════════════════════════════════════════════
-- **عملةُ مبلغ «لدي توصيلة» و«إحضار»** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «المبلغ هذا امانة فقط يحضرها السائق للتاجر مارح تدخل حساباتنا… هو يريد المبلغ
-- بالسوري او الدولار». **فالمتجرُ يختار العملة**، والسائقُ يقبض بها كما كُتبت.
--
--   amount_currency  'SYP' (الافتراض — لا يتغيّر شيءٌ لطلبٍ قائم) أو 'USD'
--
-- **والدولارُ يُحفظ بالسنتات** في `collect_amount` و`fetch_amount` نفسيهما (١٢٫٥٠$ ⇒ ١٢٥٠)
-- — فلا يتغيّر منطقُ الإحضار ولا حارسُه. **ولا يمسّ الدفتر**: المبلغان خارجه أصلاً.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS amount_currency text NOT NULL DEFAULT 'SYP';

DO $$ BEGIN
    ALTER TABLE orders ADD CONSTRAINT orders_amount_currency_chk
        CHECK (amount_currency IN ('SYP', 'USD'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
