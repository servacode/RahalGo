package com.rahalgo.shared.driver

import com.rahalgo.shared.model.CashPage
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.HistoryPage
import com.rahalgo.shared.model.MerchantRatingInput
import com.rahalgo.shared.model.ReportInput
import com.rahalgo.shared.model.ReportReasons
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.shared.model.FailReasons
import com.rahalgo.shared.model.OrderRoute
import com.rahalgo.shared.model.TrackPoint
import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.json.put
import kotlinx.serialization.json.buildJsonObject

/**
 * **أبواب السائق — كما هي في عقد الـAPI.**
 *
 * (`api/contract.json`: مسارات `driver`.)
 *
 * **ولا منطق عمل هنا** (`GROUND-RULES.md` §7.2 البند ٥): متى يُقبل طلب
 * ومتى تُفتح وردية يقرّره المحرّك — **وهذه تنادي وتُعيد ما قيل.**
 */
class DriverApi(private val api: ApiClient) {

    /** حال السائق — **الوردية والمال واليوم في نداء واحد.** */
    suspend fun me(): DriverMe = api.call("/api/v1/driver/me")

    /**
     * **سجلُّ ما نفّذه** — المغلقةُ وحدَها، الأحدثُ أوّلاً.
     *
     * (`GET /api/v1/driver/orders/history` — والنداءُ نفسُه الذي تقرؤه
     * شاشةُ الويب.)
     */
    suspend fun history(): HistoryPage = api.call("/api/v1/driver/orders/history")

    /**
     * **كشفُ صندوقه** — النقدُ الذي قبضه وما سلّمه.
     *
     * (`GET /api/v1/driver/cash` — نداءُ شاشة الويب نفسُه.)
     *
     * **ومالٌ في ذمّة إنسانٍ بلا كشفٍ يقرؤه خلافٌ ينتظر**: يقول
     * «سلّمتُ» وتقول المنصّةُ «لم يصل»، **ولا ورقةَ بينهما.**
     */
    suspend fun cash(page: Int = 1): CashPage =
        api.call("/api/v1/driver/cash?page=" + page)

    /**
     * **إعادةُ بضاعةِ طلبٍ تعذّر تسليمُه إلى متجرها.**
     *
     * (`POST /api/v1/driver/orders/{id}/return` — بابٌ في المحرّك منذ
     * زمنٍ **بلا زرٍّ يفتحه في الويب ولا في التطبيق.**)
     *
     * **وأثرُه مال**: يُعكَس مستحقُّ المتجر وتُعاد الخزينةُ إلى حسابها —
     * **فما دام لم يُسجَّل، الدفترُ يقول إنّ المتجر يستحقّ ثمنَ بضاعةٍ
     * رجعت إليه.**
     */
    suspend fun returnGoods(orderId: String): Ack =
        api.call(
            "/api/v1/driver/orders/" + orderId + "/return",
            HttpMethod.Post,
            mapOf<String, String>(),
        )

    /** **أسبابُ البلاغ** — من الخادم لا من التطبيق. */
    /**
     * **أسبابُ البلاغ لهذا الطلب بعينه.**
     *
     * **والطلبُ الخاصُّ بلا متجر** — فأسبابُ المتجر فيه سؤالٌ عمّا لا
     * وجودَ له، **ومن اختار واحداً منها فُتح بلاغٌ بلا مشتكًى عليه.**
     *
     * **والمحرّكُ هو من يصفّي** — لا الشاشة: قائمةٌ تُصفّى في العرض
     * وحدَه لا تمنع من ينادي الواجهةَ مباشرة.
     */
    suspend fun reportReasons(orderId: String): ReportReasons =
        api.call("/api/v1/driver/orders/report-reasons?order=" + orderId)

    suspend fun reportReasons(): ReportReasons =
        api.call("/api/v1/driver/orders/report-reasons")

    /** **يرفع بلاغاً على طلبٍ نفّذه** — يصل غرفةَ العمليات ويُتابَع. */
    suspend fun report(orderId: String, reason: String, note: String) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/report",
            HttpMethod.Post,
            ReportInput(reason, note),
        )
    }

    /** **يقيّم متجرَ طلبٍ وقف عند بابه** — سرعةً وتعاملا. */
    suspend fun rateMerchant(orderId: String, speed: Int, conduct: Int, comment: String) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/rate-merchant",
            HttpMethod.Post,
            MerchantRatingInput(speed, conduct, comment),
        )
    }

    /** ما هو معروض عليه الآن — **ولم يأخذه أحد بعد.** */
    suspend fun queue(): List<DriverOrder> = api.call("/api/v1/driver/queue")

    /** طلباته التي في يده — **ما لم يُغلق بعد.** */
    /**
     * **الطارئ** — ضغطةٌ واحدة: موضعُه يُلتقط، والعملياتُ تُنبَّه.
     *
     * **ولا نافذةَ يملؤها**: من كُسرت يدُه أو أُوقف في الطريق لا يكتب
     * شرحا، **وحقلٌ إلزاميٌّ في لحظةٍ كهذه** يجعله يترك الزرَّ ويتّصل.
     */
    suspend fun emergency(
        orderId: String,
        lat: Double?,
        lng: Double?,
        note: String = "",
    ): Ack =
        api.call(
            "/api/v1/driver/orders/" + orderId + "/emergency",
            HttpMethod.Post,
            buildJsonObject {
                if (lat != null) put("lat", lat)
                if (lng != null) put("lng", lng)
                put("note", note)
            },
        )

    /** **مسارُ الطرف الحاليّ** — خطُّ الشوارع ومسافتُه ومدّتُه. */
    suspend fun route(orderId: String): OrderRoute =
        api.call("/api/v1/driver/orders/" + orderId + "/route")

    suspend fun orders(): List<DriverOrder> = api.call("/api/v1/driver/orders")

    /**
     * **يأخذ الطلب.**
     *
     * **وقد يرفض المحرّك**: وردية مغلقة · طلبات أكثر من حدّه · نقد بلغ
     * السقف · **أو سبقه غيره إليه** (`order_taken`). **وكلّها أجوبة
     * طبيعيّة لا أعطال** — والشاشة تقولها كما قالها.
     */
    suspend fun accept(orderId: String) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/accept",
            HttpMethod.Post,
        )
    }

    /**
     * **يرفض العرض — فينتقل الدور فورا إلى من بعده.**
     *
     * **وكان الرفض صمتا**: يُترك حتّى تنقضي مهلته. **والزبون يقف دقيقةً
     * بلا سبب**، والسائق يعرف في الثانية الأولى أنّه لا يريده.
     *
     * **وفي «للجميع» لا رفض** — لا دور فيه أصلا، والمحرّك يردّ خطأ.
     */
    suspend fun decline(orderId: String) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/decline",
            HttpMethod.Post,
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يحرّك الطلب خطوة**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والسلسلة كما في آلة الحالات** (`orders/statuses.go`):
     * `assigned` ثمّ `at_pickup` ثمّ `picked_up` ثمّ `on_the_way` ثمّ
     * `at_dropoff` ثمّ `delivered`.
     *
     * **ولا تُحسب الخطوة هنا** — المحرّك يرفض أيّ قفزة، **وحساب التالي
     * في مكانين يجعلهما يفترقان** يوم يُضاف حال جديد.
     *
     * **والتعذّر يلزمه سبب مصنّف** لا نصّا حرّا: منه يُشتقّ الذنب الذي
     * يقرّر التعويض. (كُتب مرّة «الزبون لا يقبل او رفض او لم اجد احد» —
     * **أربعة أحكام في سطر**، وأحدها يستوجب مراجعة زبون والآخر لا.)
     */
    suspend fun transition(orderId: String, to: String, reason: String = "", note: String = "") {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/transition",
            HttpMethod.Post,
            mapOf("to" to to, "reason" to reason, "note" to note),
        )
    }

    /** **يعيد الطلب إلى الطابور** — ولا يبقى معلّقا في يد من لا يقدر. */
    suspend fun release(orderId: String, note: String = "") {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/release",
            HttpMethod.Post,
            mapOf("note" to note),
        )
    }

    /** أسباب التعذّر المسموحة في حال بعينه — **يقولها المحرّك.** */
    suspend fun failReasons(at: String): List<FailReasonItem> =
        api.call<FailReasons>("/api/v1/driver/fail-reasons?at=" + at).reasons

    /**
     * **يرسل نقطة موقع.**
     *
     * **والدقّة والسرعة تُرسلان معها**: المحرّك يتجاهل النقطة الرديئة،
     * **ونقطة بدقّة ٥٠٠ متر تُقرأ حركة وهي وقوف تحت سقف.**
     */
    suspend fun sendLocation(
        lat: Double,
        lng: Double,
        speedMps: Double? = null,
        accuracyM: Double? = null,
    ) {
        val body = buildMap<String, Any> {
            put("lat", lat)
            put("lng", lng)
            if (speedMps != null) put("speed_mps", speedMps)
            if (accuracyM != null) put("accuracy_m", accuracyM)
        }
        api.call<Ack>("/api/v1/driver/location", HttpMethod.Post, body)
    }

    /**
     * **يرسل ما تجمّع حين انقطعت الشبكة — دفعة واحدة.**
     *
     * **ولا تُرسل نقطة نقطة**: مئة نقطة مئة نداء، **وحزمة السائق تُدفع
     * من جيبه.** والمحرّك يقبل حتّى مئتين في الدفعة.
     */
    suspend fun sendBatch(points: List<TrackPoint>) {
        api.call<Ack>(
            "/api/v1/driver/location/batch",
            HttpMethod.Post,
            mapOf("points" to points),
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **صورة التسليم — بضغطتين لا أكثر**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وأهمّها الإحداثيات لا الصورة**: صورة بابٍ قد تكون لأيّ باب،
     * **والنقطة تقول أين وقف حين صوّر.**
     *
     * **والمحرّك يمنع «سُلّم» بلا إثبات** حين يُفعَّل الإعداد
     * (`drivers.require_delivery_photo`) — **والحارس فيه لا في الشاشة**:
     * زرّ يُخفى يُلتفّ عليه.
     */
    suspend fun sendProof(orderId: String, jpeg: ByteArray, lat: Double?, lng: Double?) {
        api.upload(
            "/api/v1/driver/orders/" + orderId + "/proof",
            fileName = "proof.jpg",
            bytes = jpeg,
            fields = buildMap {
                if (lat != null && lng != null) {
                    put("lat", lat.toString())
                    put("lng", lng.toString())
                }
            },
        )
    }

    /** **يتخطّى الصورة بسبب** — ولا يُقبل تخطٍّ بلا سبب. */
    suspend fun skipProof(orderId: String, reason: String) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/proof/skip",
            HttpMethod.Post,
            mapOf("reason" to reason),
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **توثيق ما اتُّفق عليه في الطلب الخاصّ**
     * ══════════════════════════════════════════════════════════════════
     *
     * (البند التاسع في قائمة المالك ٢٠٢٦-٠٨-١٢.)
     *
     * **وثمن البضاعة اختياريّ**: قد تكون أمانةً لا ثمن لها — **فأجرة
     * التوصيل وحدَها.** ومن ألزم بثمنٍ في كلّ طلب **جعل السائق يكتب
     * رقما من رأسه** ليمضي.
     *
     * **والمنصّة توثّق ولا تحاسب**: السائق يدفع من جيبه ويستردّ عند
     * التسليم. **والتوثيق هو ما يُرجع إليه** يوم يختلفان.
     */
    suspend fun agree(orderId: String, goodsAmount: Long, fee: Long) {
        api.call<Ack>(
            "/api/v1/driver/orders/" + orderId + "/agree",
            HttpMethod.Post,
            mapOf("goods_amount" to goodsAmount, "fee" to fee),
        )
    }

    /**
     * يفتح الوردية أو يغلقها.
     *
     * **والنتيجة تُقرأ من المحرّك لا تُفترض**: قد يرفض الفتح (نقد فوق
     * السقف مثلا)، **ومن بدّل الزرّ قبل الجواب** أرى صاحبه «متاح» وهو
     * ليس كذلك — فينتظر طلبا لا يأتي.
     */
    suspend fun setShift(on: Boolean): Boolean =
        api.call<Map<String, Boolean>>(
            "/api/v1/driver/shift",
            HttpMethod.Post,
            mapOf("on" to on),
        )["on_shift"] ?: on
}
