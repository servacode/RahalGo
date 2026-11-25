-- «سلّتك ناطرتك» — تذكيرُ السلّة المتروكة (قرارُ المالك ٢٠٢٦-١١-٢٥).
--
-- # لماذا جدولٌ في الخادم
--
-- **السلّةُ عند العميل فقط** (SharedPreferences في التطبيق، localStorage في /shop) — فالخادمُ لا
-- يعرف أنّ في سلّة الزبون شيئاً. فيُبلِّغ العميلُ حالَها عند كلّ تغيير (`PUT /api/v1/my/cart-state`)
-- ويُفرغها عند الإفراغ أو الطلب (`DELETE`). **صفٌّ واحدٌ لكلّ زبون.**
--
-- # مرّةً لكلّ سلّة
--
-- `signature` بصمةُ محتوى السلّة (تُخزَّن مُجزَّأةً): بلاغٌ بالمحتوى نفسِه لا يلمس الصفَّ، فإعادةُ فتح
-- التطبيق لا تُعيد المؤقّت ولا تُعيد التذكير. **وتغييرُ المحتوى وحدَه** يمسح `reminded_at`
-- ويُرجع `changed_at` إلى الآن، فيصير تذكيرٌ جديدٌ ممكناً.
--
-- # ولا تذكيرَ بعد الطلب
--
-- المكنسةُ تتخطّى من طلب بعد `changed_at` (بغضّ النظر عن العميل)، وتختم الصفَّ `reminded_at`
-- بالتقاطٍ ذرّيٍّ واحد — فخادمان معاً أو إعادةُ التشغيل لا ترسلان مرّتين.

CREATE TABLE IF NOT EXISTS cart_reminders (
    user_id     uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    items_count integer NOT NULL CHECK (items_count > 0),
    signature   text NOT NULL DEFAULT '',
    -- آخرُ تغييرٍ في محتوى السلّة — منه يُحسب «بعد ساعة».
    changed_at  timestamptz NOT NULL DEFAULT now(),
    -- متى أُرسل التذكير؛ فارغٌ = لم يُرسَل بعد.
    reminded_at timestamptz
);

-- ما تلتقطه المكنسة: غيرُ المُذكَّر مرتَّباً بوقت التغيير.
CREATE INDEX IF NOT EXISTS cart_reminders_pending
    ON cart_reminders (changed_at) WHERE reminded_at IS NULL;
