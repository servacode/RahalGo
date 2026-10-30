package com.rahalgo.customer.live

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * **«شارك موقعي المباشر مع السائق أثناء الطلب»** — تفضيلٌ على الجهاز (قرارُ المالك ٢٠٢٦-١٠-٣٠).
 *
 * **مفعّلٌ افتراضاً.** وما دام مفعّلاً تبدأ المشاركةُ وحدَها حين يصير الطلبُ «بالطريق»
 * (أو يطلبها السائق) — بلا نافذة سؤال إن كان الإذنُ والموقعُ جاهزَين. **والإيقافُ باقٍ**:
 * من بطاقة الطلب أو من الإشعار، ومن أوقفها لطلبٍ لا تعود له.
 */
object LiveAuto {
    private const val FILE = "rahalgo_live_auto"
    private const val KEY = "on"

    /** **الحالُ للشاشات** — تُقرأ من التخزين أوّلَ مرّة. */
    var on by mutableStateOf(true)
        private set

    private var loaded = false

    fun load(context: Context): Boolean {
        if (!loaded) {
            loaded = true
            on = context.applicationContext
                .getSharedPreferences(FILE, Context.MODE_PRIVATE)
                .getBoolean(KEY, true)
        }
        return on
    }

    fun set(context: Context, value: Boolean) {
        loaded = true
        on = value
        context.applicationContext
            .getSharedPreferences(FILE, Context.MODE_PRIVATE)
            .edit().putBoolean(KEY, value).apply()
    }
}
