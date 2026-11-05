package com.rahalgo.shared.driver

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رقمُ العدّاد كما يُكتب** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الرقمُ عليه الحساب** — فيُقرأ كما يكتبه السائق: أرقامٌ غربيّةٌ أو عربيّةٌ (٠-٩ و۰-۹)،
 * وفاصلٌ عشريٌّ واحد (`.` أو `,` أو الفاصلُ العربيّ) ومنزلةٌ عشريّةٌ واحدة، وسبعُ خاناتٍ
 * صحيحةٍ على الأكثر. **ولا يُستعمل حقلُ المبلغ** (`isDigit`) — يُمرّر الأرقامَ العربيّة بلا تحويل.
 *
 * والحكمُ الأخير للمحرّك (`odometer_bad_km`) — هذا حتّى لا يُكتب ما سيُردّ.
 */
object OdometerInput {
    /** أطولُ رقمٍ صحيح — `numeric(8,1)` في القاعدة. */
    const val MAX_INT_DIGITS = 7

    /**
     * **أيُقبل هذا النصّ أثناء الكتابة؟** — يردّه مُوحَّداً (أرقامٌ غربيّةٌ ونقطة) أو فارغاً
     * إن كان فيه ما لا يُقبل (حرفٌ أو فاصلان أو منزلتان أو أكثرُ من سبع خانات).
     */
    fun accept(raw: String): String? {
        val out = StringBuilder()
        var sep = false
        var ints = 0
        var decimals = 0
        for (c in raw) {
            val d = digit(c)
            when {
                d != null && !sep -> {
                    if (ints >= MAX_INT_DIGITS) return null
                    ints++
                    out.append(d)
                }
                d != null -> {
                    if (decimals >= 1) return null
                    decimals++
                    out.append(d)
                }
                c == '.' || c == ',' || c == '٫' -> {
                    if (sep || ints == 0) return null
                    sep = true
                    out.append('.')
                }
                else -> return null
            }
        }
        return out.toString()
    }

    /** **النصُّ الذي يُرسَل** — «1250.» تصير «1250»، والفارغُ أو المرفوضُ فارغ. */
    fun normalize(raw: String): String? = accept(raw.trim())?.trimEnd('.')?.ifEmpty { null }

    /** **الرقمُ كما يُعرض** — «1250» أو «1250.5»، بأرقامٍ غربيّة. */
    fun show(km: Double): String =
        if (km % 1.0 == 0.0) km.toLong().toString()
        else String.format(java.util.Locale.US, "%.1f", km)

    private fun digit(c: Char): Char? = when (c) {
        in '0'..'9' -> c
        in '٠'..'٩' -> '0' + (c - '٠')
        in '۰'..'۹' -> '0' + (c - '۰')
        else -> null
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ماذا يفعل مفتاحُ الدوام** — من حكم المحرّك (`odometer_mode`) لا من رأي التطبيق
 * ══════════════════════════════════════════════════════════════════════
 *
 *	off      ← يُرسَل الفتحُ والسكّرُ مباشرةً كما كان
 *	ask      ← ورقةُ العدّاد، ويُسمح بـ«لم أستطع التصوير»
 *	require  ← ورقةُ العدّاد بلا تخطٍّ — **وصورةُ الليل حصراً** («مشان الصبح نعرف إذا هو
 *	           ماشي بالموتور»)؛ وسماحُ العمليّات يفتح الدوامَ بلا صورةٍ مرّةً واحدة
 *
 * **وسكّرٌ وبيده طلب** يُرسَل مباشرةً — والمحرّكُ يردّه برسالته (`has_active_orders`) قبل أن
 * يُطلب منه تصويرُ شيء.
 */
object OdometerFlow {
    sealed interface Step

    /** يُرسَل مباشرةً بلا ورقة. */
    data object Direct : Step

    /** ورقةُ العدّاد: `start` للفتح و`end` للسكّر. */
    data class Sheet(val phase: String, val skippable: Boolean) : Step

    fun onToggle(on: Boolean, mode: String, exempt: Boolean, activeOrders: Int): Step {
        if (!on && activeOrders > 0) return Direct
        val phase = if (on) "start" else "end"
        return when (mode) {
            "require" -> if (on && exempt) Direct else Sheet(phase, skippable = false)
            "ask" -> Sheet(phase, skippable = true)
            else -> Direct
        }
    }

    /** **الخروجُ من التطبيق وهو بدوامه سكّرٌ للدوام** — يُسأل عن العدّاد كما يُسأل عند المفتاح. */
    fun onLogout(onShift: Boolean, mode: String, activeOrders: Int): Step =
        if (!onShift) Direct else onToggle(false, mode, exempt = false, activeOrders = activeOrders)
}
