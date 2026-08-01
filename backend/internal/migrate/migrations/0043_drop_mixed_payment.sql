-- إلغاء الدفع المختلط من القاعدة نفسها.
--
-- أُلغي من المحرّك بقرار المالك: كان يدفع ما في المحفظة ويترك الباقي نقداً،
-- فيصير للطلب الواحد **مصدرا دفعٍ ومسارا تسويةٍ ومسارا استرجاع** — تعقيدٌ في
-- أخطر جزء من النظام مقابل راحةٍ لا يطلبها أحد.
--
-- وبقي في القيد وفي قائمة السلّة بعده. **وخيارٌ يُعرض ويرفضه الخادم أسوأ من
-- خيارٍ غائب**: من اختاره ظنّ أن النظام انكسر لا أن اختياره ملغى.
--
-- والقيد يُشدَّد الآن كي لا يعود بابُه مفتوحاً: ما لا يُقبل في الشيفرة لا يُقبل
-- في القاعدة — وإلا صار الحارسان اثنين يختلفان.

-- لا صفوف مختلطة في القاعدة (أُلغي قبل أن يُستعمل)، والتحويل احتياطٌ لا أكثر:
-- طلبٌ مختلط دُفع بعضُه من المحفظة يُقرأ محفظةً، وما بقي نقداً باقٍ في cash_due.
UPDATE orders SET payment_method = 'wallet'
 WHERE payment_method = 'mixed' AND wallet_paid > 0;
UPDATE orders SET payment_method = 'cash'
 WHERE payment_method = 'mixed';

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_method_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
    CHECK (payment_method IN ('cash', 'wallet'));
