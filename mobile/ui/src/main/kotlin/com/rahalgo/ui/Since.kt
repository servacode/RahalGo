package com.rahalgo.ui

import android.content.Context
import java.time.Duration
import java.time.Instant

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كم مضى — بحرفٍ واحدٍ لا بجملة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(طلبُ المالك ٢٠٢٦-٠٩-٠٢:** أُضيف «عمرُ الطلب» على بطاقة المتجر.)
 *
 * # ولماذا في `:ui` لا في تطبيق المتجر
 *
 * **وكانت نصوصُه الثلاثةُ في تطبيق المتجر وحدَه** (`ord_since` و
 * `ord_min` و`ord_hr`) — **مكتوبةً منذ زمنٍ ولا تُستعمل.**
 *
 * **والوقتُ يمضي على كلّ شيءٍ لا على الطلب وحدَه**: شكوى تنتظر
 * جواباً، وسائقٌ لم يتحرّك، وطلبُ صرفٍ لم يُبتّ. **فمن كتبها في
 * تطبيقٍ كتبها في الأربعة بعد شهر.**
 *
 * # والحرفُ لا الكلمة
 *
 * **«منذ ٧ د» لا «منذ سبع دقائق»** — البطاقةُ ضيّقةٌ وفيها الرقمُ
 * والمبلغُ والحالة، **وجملةٌ طويلةٌ تدفع ما بعدها إلى سطرٍ ثانٍ.**
 *
 * # وأقلُّها دقيقة
 *
 * **وما دون الدقيقة «منذ ١ د»** — لا «منذ ٠ د» ولا ثوانٍ تتغيّر
 * أمام العين. **والطلبُ الذي وصل الآن لا يُقاس عمرُه، يُقرأ جديداً.**
 *
 * # وما لا يُقرأ لا يُخمَّن
 *
 * **ووقتٌ لا يُفهم يردّ فراغاً** — لا «منذ ٠» ولا تاريخَ اليوم.
 * **ومن اخترع قيمةً من نصٍّ فاسدٍ كتب على الشاشة كذباً هادئاً.**
 */
object Since {

    /** **أطولُ ما يُعرض بالساعات** — وما فوقه يُقاس بالأيّام. */
    private const val HOURS_IN_DAY = 24L

    /**
     * **النصُّ الجاهز** — «منذ ٧ د» أو «منذ ٣ س»، وفارغٌ إن تعذّر.
     *
     * @param iso وقتُ الإنشاء بصيغة `ISO-8601` كما يرسله المحرّك.
     */
    fun text(context: Context, iso: String, now: Instant = Instant.now()): String {
        val minutes = minutes(iso, now) ?: return ""
        val amount = when {
            minutes < 60 -> "$minutes " + context.getString(R.string.ord_min)
            minutes < 60 * HOURS_IN_DAY -> "${minutes / 60} " + context.getString(R.string.ord_hr)
            else -> "${minutes / (60 * HOURS_IN_DAY)} " + context.getString(R.string.ord_day)
        }
        return context.getString(R.string.ord_since, amount)
    }

    /**
     * **الدقائقُ الماضية** — أو فارغٌ إن لم يُفهم الوقت.
     *
     * **ووقتٌ في المستقبل يُقرأ دقيقةً** — ساعةُ الجهاز قد تسبق
     * ساعةَ الخادم بثوانٍ، **ولا يُكتب «منذ ناقص ثلاث».**
     */
    fun minutes(iso: String, now: Instant = Instant.now()): Long? {
        val at = runCatching { Instant.parse(iso) }.getOrNull()
            ?: runCatching { java.time.OffsetDateTime.parse(iso).toInstant() }.getOrNull()
            ?: return null
        return Duration.between(at, now).toMinutes().coerceAtLeast(1L)
    }
}
