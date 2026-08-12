package com.rahalgo.shared.net

import io.ktor.client.plugins.websocket.WebSockets
import io.ktor.client.plugins.websocket.webSocket
import io.ktor.websocket.Frame
import io.ktor.websocket.readText
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البثّ الحيّ — الشاشة تتحدّث بنفسها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`GET /api/v1/ws?token=…` — والمحرّك يشترك للسائق في موضوعين:
 * `drivers:queue` أي «الطابور تغيّر»، و`driver:<id>` أي «طلبك تغيّر».)
 *
 * # ولماذا مع الدفع لا بدلا منه
 *
 * **الدفع يوقظ الجيب المغلق، والبثّ يحدّث الشاشة المفتوحة.** ومن اكتفى
 * بالدفع **أبقى القائمة على حالها والتطبيق أمام عين صاحبه**، فيرى طلبا
 * أخذه غيره ويضغطه فيُقال له «سبقك غيرك».
 *
 * # والحمولة لا تُقرأ
 *
 * **المحرّك يرسل إشارة لا محتوى** («تغيّر شيء») — **وما يراه كلّ سائق
 * تقرّره نقطة الطابور** بحسب ورديّته ونمط التوزيع. **ومن بنى القائمة من
 * الحمولة** بنى نسخة ثانية من قواعد لا يملكها.
 *
 * # ويعيد الوصل بنفسه
 *
 * **الاتّصال ينقطع كثيرا**: نفق، أو تبديل شبكة، أو نوم الجهاز. **ومن لم
 * يُعد الوصل** بقي التطبيق مفتوحا وصامتا — وهو أسوأ من مغلق: **صاحبه
 * يظنّه يعمل.**
 *
 * **والمهلة تتضاعف** — فلا يُستنزف الجهاز بمحاولةٍ كلّ ثانية حين يكون
 * الخادم نائما.
 */
class LiveSocket(
    private val baseUrl: String,
    private val session: SessionStore,
    private val client: String,
) {

    private val http = io.ktor.client.HttpClient {
        install(WebSockets)
    }

    private var job: Job? = null

    /**
     * يبدأ الإصغاء — **و`onEvent` تُنادى عند كلّ إشارة.**
     *
     * **وتُنادى مرّة واحدة لكلّ حياة شاشة** — والبدء مرّتين يفتح وصلتين.
     */
    fun start(
        scope: CoroutineScope,
        // **وحال الوصلة يُرى** — «لا انهيار» ليس دليل اتّصال: **وصلة
        // صامتة تبدو كوصلة هادئة**، ولا يُعرف الفرق إلّا بأثر.
        onState: (Boolean) -> Unit = {},
        onEvent: () -> Unit,
    ) {
        if (job?.isActive == true) return
        job = scope.launch {
            var wait = FIRST_RETRY_MS
            while (isActive) {
                try {
                    // **والتوكن في الرابط لا في ترويسة** — مقبس الويب لا
                    // يحمل ترويسة تفويض في المتصفّح، **والمحرّك يقرأه من
                    // الاستعلام** (`ws.go`).
                    val url = baseUrl.replace("https://", "wss://").replace("http://", "ws://") +
                        "/api/v1/ws?token=" + session.accessToken()
                    http.webSocket(url) {
                        wait = FIRST_RETRY_MS
                        onState(true)
                        for (frame in incoming) {
                            if (frame is Frame.Text) {
                                frame.readText()
                                onEvent()
                            }
                        }
                    }
                } catch (e: Exception) {
                    // **وانقطاع الوصل ليس عطبا** — يقع كلّ يوم عشرات
                    // المرّات، **ولا يُكتب في السجلّ كخطأ** فيغرقه.
                }
                onState(false)
                delay(wait)
                wait = (wait * 2).coerceAtMost(MAX_RETRY_MS)
            }
        }
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    private companion object {
        const val FIRST_RETRY_MS = 2_000L
        const val MAX_RETRY_MS = 30_000L
    }
}
