package com.rahalgo.driver.orders

import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow

/**
 * **نبضةُ «اقرأ الطابورَ من جديد» بعد فتح الورديّة أو إغلاقها.**
 *
 * **رُئي على المحاكي ٢٠٢٦-١٠-٠٥**: طلبٌ في الطابور قبل أن يفتح السائقُ
 * ورديّتَه، **ففتحها فبقيت «الطلبات» تقول «لا يوجد طلب في هذه اللحظة»
 * خمسَ دقائق** والخادمُ يعرضه عليه — لأنّ القائمةَ تُقرأ مع أحداث الوصلة
 * وحدَها، **ودفعُ الطلب ذهب لمن كان في ورديّته ساعةَ نزل.** ولم يظهر
 * إلّا بعد إعادة فتح التطبيق.
 */
object QueuePulse {
    private val _flow = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val flow: SharedFlow<Unit> = _flow.asSharedFlow()

    fun bump() {
        _flow.tryEmit(Unit)
    }
}
