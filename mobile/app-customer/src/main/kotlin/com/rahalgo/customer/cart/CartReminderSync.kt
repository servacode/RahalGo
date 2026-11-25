package com.rahalgo.customer.cart

import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«سلّتك ناطرتك» — يُعلِم الخادمَ بحال السلّة** (قرارُ المالك ٢٠٢٦-١١-٢٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السلّةُ عندنا فقط** (`Cart`)، فالخادمُ لا يعرف أنّ في سلّة الزبون شيئاً. فبعد كلّ تغييرٍ
 * (وعند الإقلاع) يُبلَّغ بمتجرها وعددها وبصمة محتواها، **والخادمُ يرسل تذكيراً واحداً بعد ساعةٍ بلا طلب.**
 * وسلّةٌ فارغةٌ (إفراغٌ أو طلب) تُلغي التذكير.
 *
 * # ولا يُثقل
 *
 * **ثوانٍ قليلةٌ تُجمَّع فيها الضغطات** — عشرُ ضغطاتِ «زِد» نداءٌ واحد. **وفشلُه صمتٌ**: التذكيرُ
 * تحسينٌ لا يُوقف الطلب، والبلاغُ التالي يُصلحه. **والضيفُ لا نداءَ له** (لا جلسة).
 */
object CartReminderSync {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var job: Job? = null

    /** **انتظارُ تجميع الضغطات** قبل البلاغ. */
    private const val DEBOUNCE_MS = 3_000L

    /** **بصمةُ المحتوى** — مفتاحُ كلّ سطرٍ وعددُه، مرتّبةً فلا يغيّرها الترتيب. */
    fun signature(lines: List<Cart.Line>): String =
        lines.map { it.key + "x" + it.qty }.sorted().joinToString(";")

    /** **يُنادى بعد كلّ تغيير في السلّة** — يؤجّل ويجمع. */
    fun changed(lines: List<Cart.Line>) {
        val snapshot = lines
        job?.cancel()
        job = scope.launch {
            delay(DEBOUNCE_MS)
            runCatching { send(snapshot) }
        }
    }

    private suspend fun send(lines: List<Cart.Line>) {
        val core = AppCore.get()
        if (core.session.refreshToken().isEmpty()) return
        val api = CustomerApi(core.api)
        val count = lines.sumOf { it.qty }
        val merchant = lines.firstOrNull()?.item?.merchantId.orEmpty()
        if (count <= 0 || merchant.isEmpty()) {
            api.clearCartState()
        } else {
            api.reportCartState(merchant, count, signature(lines))
        }
    }
}
