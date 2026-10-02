package com.rahalgo.driver.orders

import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
import kotlinx.coroutines.delay

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خطوةٌ لا تُكرَّر ولا تُكذَّب** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ردٌّ ضاع في الشبكة والخطوةُ ثبتت** — كان السائقُ يرى خطأً، ثمّ يضغط ثانيةً
 * فيُردّ «انتقالٌ غيرُ جائز» **وهو لا يعرف أوقعت أم لا.**
 *
 * **فمفتاحٌ واحدٌ للمحاولة يُعاد بعينه مرّةً حين تنقطع الشبكة** — والمحرّكُ يردّ
 * على الإعادة بالطلب كما هو (`replayOwnTransition`). **وما عدا الانقطاع لا يُعاد**:
 * رفضُ المحرّك جوابٌ يُقال لا عطلٌ يُتجاوز.
 */
object StepRetry {

    /** **كم يُنتظر قبل الإعادة** — شبكةُ السوق تعود في ثانية أو لا تعود. */
    const val WAIT_MS = 1_500L

    /** **انقطاعٌ أو مهلة** — وهما وحدَهما يحتملان «ثبتت وضاع ردُّها». */
    fun retriable(e: Throwable): Boolean = e is IOException || e is HttpRequestTimeoutException

    /** **يُرسل الخطوةَ بمفتاحها، ويُعيدها مرّةً بالمفتاح نفسِه إن انقطعت الشبكة.** */
    suspend fun send(key: String, call: suspend (String) -> Unit) {
        try {
            call(key)
        } catch (e: Exception) {
            if (!retriable(e)) throw e
            delay(WAIT_MS)
            call(key)
        }
    }

    fun newKey(): String = java.util.UUID.randomUUID().toString()
}
