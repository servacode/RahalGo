package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حال السائق — كما يقوله المحرّك لا كما نتخيّله**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`GET /api/v1/driver/me` — `server/driver_handlers.go`.)
 *
 * **وكلّ حقل هنا له مقابل هناك بالاسم نفسه.** والاسم في الشبكة غير الاسم
 * في Kotlin (`on_shift` هناك و`onShift` هنا)، **ومن كتبه من رأسه بنى
 * حقلا لا وجود له** — ولا خطأ في بناء ولا سطر في سجلّ، **إنّما شاشة
 * تعرض صفرا.**
 *
 * # والمبالغ أعداد صحيحة
 *
 * **لا كسور في الليرة** — والمحرّك يرسلها `int64`. **ومن قرأها عشريّة**
 * أظهر «12500.0» في شاشة لا كسر فيها.
 */
@Serializable
data class DriverMe(
    @SerialName("full_name") val fullName: String = "",

    // ── الوردية ──
    /** **مفتاح كلّ شيء**: من ليس على وردية لا يصله طلب. */
    @SerialName("on_shift") val onShift: Boolean = false,
    @SerialName("shift_started_at") val shiftStartedAt: String? = null,

    // ── المال ──
    /** النقد الذي بذمّته — **يجمعه من الزبائن ويسلّمه للمكتب.** */
    @SerialName("cash_held") val cashHeld: Long = 0,
    /** **سقفه**: من بلغه لا يصله طلب نقدي حتّى يسلّم. */
    @SerialName("cash_limit") val cashLimit: Long = 0,
    val balance: Long = 0,

    // ── اليوم ──
    @SerialName("today_delivered") val todayDelivered: Int = 0,
    @SerialName("today_failed") val todayFailed: Int = 0,
    @SerialName("today_earned") val todayEarned: Long = 0,
    @SerialName("today_compensated") val todayCompensated: Long = 0,

    // ── الطلبات ──
    @SerialName("active_orders") val activeOrders: Int = 0,
    @SerialName("max_active_orders") val maxActiveOrders: Long = 0,

    // ── ما يحكم سلوك التطبيق ──
    /** أيلزم إثبات تسليم بصورة؟ — **يقرّره المحرّك لا التطبيق.** */
    @SerialName("require_photo") val requirePhoto: Boolean = false,
    /** **كلّ كم ثانية يُرسل الموقع** — رقم من الإعدادات لا من الشيفرة. */
    @SerialName("location_ping_sec") val locationPingSec: Long = 0,
    @SerialName("avg_speed_kmh") val avgSpeedKmh: Long = 0,

    // ── تقييمه ──
    /** **متوسّط نجومه** — وصفر يعني لم يُقيَّم بعد. */
    val rating: Double = 0.0,
    /** **كم قيّمه** — به يُفرَّق «لا تقييم» عن «تقييم منخفض». */
    @SerialName("rating_count") val ratingCount: Int = 0,
)

/**
 * **نقطة مسار** — كما يقبلها `driver/location/batch`.
 *
 * **ووقتها من الجهاز لا من الخادم**: الدفعة تصل بعد أن تعود الشبكة،
 * **ولو خُتمت بوقت الوصول** لظهر السائق قافزاً من مكان إلى مكان في
 * لحظة واحدة.
 *
 * **والمحرّك يرفض ما جاوز ساعتين** أو ما كان في المستقبل بأكثر من خمس
 * دقائق (`batchMaxAge` و`futureSkew`).
 */
@Serializable
data class TrackPoint(
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    /** وقت الالتقاط بصيغة `ISO-8601` — **لا وقت الإرسال.** */
    val at: String = "",
    @SerialName("speed_mps") val speedMps: Double? = null,
    @SerialName("accuracy_m") val accuracyM: Double? = null,
)
