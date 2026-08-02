-- ٠٠٥٤ · الإنذارُ يُسجَّل ولا يمرّ
--
-- # المسألة
--
-- المتجرُ الذي لا يسلّم البضاعةَ للسائق — مغلقٌ، أو رافض، أو «لا أعرف هذا
-- الطلب» — **كان يُرسَل إليه إشعارٌ ويمضي.** والإشعارُ يُقرأ ويُنسى، **ولا
-- يُعدّ**: لا يُعرف كم مرّةً وقع منه ذلك هذا الشهر.
--
-- **وعدُّ المخالفات كان يرى نصفَ الصورة**: يعدّ ما ألغاه المتجرُ بيده
-- (`ended_by = 'merchant'`)، **ولا يعدّ ما أفشله بامتناعه** — والثاني أسوأ:
-- في الإلغاء يعرف الزبونُ باكراً، **وفي الامتناع يكون السائقُ قد قاد والزبونُ
-- قد انتظر.**
--
-- # ولماذا جدولٌ ولم تكفِ الطلبات
--
-- أكثرُ الإنذارات تُشتقّ من طلباتٍ وقعت — **وتلك لا تُنسخ هنا**: الطلبُ هو
-- سجلُّها، ونسخُها يخلق مصدرين للحقيقة الواحدة.
--
-- **لكنّ إنذاراً بلا طلب موجود**: متجرٌ رفع أسعارَه عن المتّفق، أو أساء إلى
-- سائق، أو تكرّر تأخيرُه بلا فشلٍ مسجَّل. **ومن لا مكانَ لإنذاره لا يُنذَر
-- إلّا بمكالمةٍ لا أثر لها.**
CREATE TABLE IF NOT EXISTS merchant_warnings (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id  uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    -- reason رمزٌ مصنَّف — **للعلّةِ نفسِها التي صُنّف بها سببُ تعذّر التسليم**.
    reason       text NOT NULL,
    note         text NOT NULL DEFAULT '',
    -- order_id الطلبُ الذي نشأ عنه — وفارغٌ في إنذارٍ يدويّ.
    order_id     uuid REFERENCES orders(id) ON DELETE SET NULL,
    issued_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS merchant_warnings_by_merchant
    ON merchant_warnings (merchant_id, created_at DESC);

-- **ولا إنذاران على طلبٍ واحد.**
--
-- بلا هذا يُنذَر المتجرُ مرّتين على الواقعة نفسِها إن أُعيد الانتقال أو
-- تكرّرت المناداة — **فيبلغ الحدَّ بضعفِ سرعته ويُحظر على نصف ما استحقّ.**
CREATE UNIQUE INDEX IF NOT EXISTS merchant_warnings_one_per_order
    ON merchant_warnings (order_id) WHERE order_id IS NOT NULL;
