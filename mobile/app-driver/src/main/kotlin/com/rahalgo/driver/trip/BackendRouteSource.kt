package com.rahalgo.driver.trip

import com.rahalgo.navigation.RerouteFailure
import com.rahalgo.navigation.RouteReply
import com.rahalgo.navigation.RouteSource
import com.rahalgo.driver.data.Backend
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بابُ الشبكة كما ينفّذه تطبيقُ السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣ب، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * **ووحدةُ الملاحة لا تعرف شيئاً ممّا هنا** — لا `Ktor` ولا `Backend`
 * ولا `OrderRoute` ولا JSON. **تعرف `RouteSource` ودالّةً واحدة.**
 *
 * **وهنا وحدَه يُعرف رقمُ الطلب** — والوحدةُ لا تعرف طلباً أصلاً.
 *
 * # ولا يُرسَل غيرُ النقطة
 *
 * **الجهازُ يرسل واقعةً والخادمُ يقرّر** — قاعدةُ المشروع. **فلا وجهةَ
 * تُبنى هنا ولا تُقرأ من الشاشة**، والخادمُ يقرأ طورَ الطلب من قاعدته
 * لحظةَ النداء.
 */
class BackendRouteSource(
    private val backend: Backend.Wired,
    private val scope: CoroutineScope,
    /** **رقمُ الطلب الآن** — دالّةٌ لا قيمة: **الطلبُ يتبدّل والباب واحد.** */
    private val orderId: () -> String?,
) : RouteSource {

    override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
        val id = orderId()
        if (id == null) {
            done(RouteReply.Failed(RerouteFailure.NO_ROUTE))
            return
        }
        scope.launch(Dispatchers.IO) {
            // **ومهلةٌ صريحةٌ لا انتظارٌ مفتوح** — **وطلبٌ لا يعود
            // يُبقي المحرّكَ في `REQUESTING` أبداً**، فلا تُعاد محاولةٌ
            // ولا يُعرض إخفاق.
            val reply = withTimeoutOrNull(TIMEOUT_MS) {
                runCatching { backend.driver.route(id, lat, lng) }.fold(
                    onSuccess = { r ->
                        val nav = NavRouteMapper.toNavRoute(r)
                        when {
                            !r.available -> RouteReply.Failed(RerouteFailure.NO_ROUTE)
                            // **ومسارٌ بلا بياناتِ ملاحةٍ لا يصلح
                            // لإعادة حساب** — يُرسم ولا يُرشِد.
                            nav == null -> RouteReply.Failed(RerouteFailure.INVALID_ROUTE)
                            else -> RouteReply.Ok(nav)
                        }
                    },
                    // **وكلُّ ما يرميه النداءُ شبكةٌ من ناحيتنا** —
                    // ولا يُفصَّل أكثرَ ممّا يفيد التهدئة.
                    onFailure = { RouteReply.Failed(RerouteFailure.NETWORK) },
                )
            } ?: RouteReply.Failed(RerouteFailure.TIMEOUT)
            done(reply)
        }
    }

    private companion object {
        /**
         * **عشرُ ثوانٍ.**
         *
         * **وأطولُ من مهلة المحرّك المعتادة** — والمخبأُ يردّ في
         * أجزاءٍ من الثانية، **والنداءُ الحيُّ يستغرق ثوانيَ على شبكةِ
         * جيلٍ ثالث.**
         */
        const val TIMEOUT_MS = 10_000L
    }
}
