package com.rahalgo.ui

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.State
import androidx.compose.runtime.produceState
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
        // **كلماتٌ كاملةٌ لا حروفٌ** (٢٠٢٦-١١-٠٤): «منذ 1 ي» لم يفهمها الزبون — «منذ دقيقة / ٥ دقائق /
        // ساعتين / يومين»، **وما جاوز يومين تاريخٌ ووقتٌ بتوقيت دمشق.**
        val age = age(minutes)
        val amount = when (age.unit) {
            Part.DATE -> return runCatching {
                java.time.OffsetDateTime.parse(iso).atZoneSameInstant(SYRIA_ZONE).format(DATE_TIME)
            }.getOrNull() ?: runCatching {
                Instant.parse(iso).atZone(SYRIA_ZONE).format(DATE_TIME)
            }.getOrDefault("")
            Part.MIN -> plural(
                context, age.n, R.string.since_min_one, R.string.since_min_two,
                R.string.since_min_few, R.string.since_min_many,
            )
            Part.HR -> plural(
                context, age.n, R.string.since_hr_one, R.string.since_hr_two,
                R.string.since_hr_few, R.string.since_hr_many,
            )
            Part.DAY -> plural(
                context, age.n, R.string.since_day_one, R.string.since_day_two,
                R.string.since_day_few, R.string.since_day_many,
            )
        }
        return context.getString(R.string.ord_since, amount)
    }

    /** **وحدةُ العمر** — تُحسَب بلا سياقٍ لتُختبر. */
    enum class Part { MIN, HR, DAY, DATE }

    data class Age(val unit: Part, val n: Long)

    /** **دقائقٌ ← وحدةٌ وعدد**: دقائق دون الساعة، ساعاتٌ دون اليوم، يومان كحدٍّ أقصى، وبعدها تاريخ. */
    fun age(minutes: Long): Age = when {
        minutes < 60 -> Age(Part.MIN, minutes)
        minutes < 60 * HOURS_IN_DAY -> Age(Part.HR, minutes / 60)
        minutes < 60 * HOURS_IN_DAY * 3 -> Age(Part.DAY, minutes / (60 * HOURS_IN_DAY))
        else -> Age(Part.DATE, 0)
    }

    /** **صيغةُ العدد العربيّة**: ١ مفرد، ٢ مثنّى، ٣–١٠ جمع، وما فوق تمييز مفرد. */
    fun form(n: Long): Int = when {
        n == 1L -> 1
        n == 2L -> 2
        n in 3..10 -> 3
        else -> 4
    }

    private fun plural(c: Context, n: Long, one: Int, two: Int, few: Int, many: Int): String =
        when (form(n)) {
            1 -> c.getString(one)
            2 -> c.getString(two)
            3 -> c.getString(few, n)
            else -> c.getString(many, n)
        }

    private val DATE_TIME = java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")

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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الساعةُ تمشي ما دامت الشاشةُ مفتوحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: «منذ 1 د» بقيت كما هي دقائقَ طويلةً والشاشةُ
 *  مفتوحة.)
 *
 * **وكان `Since.text` يُنادى بـ`Instant.now()` لحظةَ الرسم وحدَها** — ولا
 * شيءَ يُعيد الرسم إن لم يتبدّل الطلب، **فيقف العمرُ عند أوّل قراءة.**
 *
 * **فحالٌ تتجدّد كلَّ نصف دقيقة** — والعمرُ يُقاس بالدقيقة، **فلا يتأخّر
 * أكثرَ من نصفها.** **وتقف حين تخرج الشاشةُ من التركيب** — لا ساعةَ تدقّ
 * لشاشةٍ لا يراها أحد.
 */
@Composable
fun rememberNow(periodMs: Long = NOW_TICK_MS): State<Instant> =
    produceState(Instant.now(), periodMs) {
        while (true) {
            kotlinx.coroutines.delay(periodMs)
            value = Instant.now()
        }
    }

/** **نصفُ دقيقة** — والعمرُ بالدقيقة. */
const val NOW_TICK_MS = 30_000L
