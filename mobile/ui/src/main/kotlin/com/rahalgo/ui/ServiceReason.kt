package com.rahalgo.ui

import android.content.Context

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سياسةُ عرضِ الإتاحة — موضعٌ واحدٌ لا شاشة** (`AV`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # المسألة
 *
 * **وأسبابُ المنع عشرة** — وضعُ إطلاقٍ وإيقافٌ مؤقّتٌ ودوامُ منصّةٍ
 * ونقطةٌ مشوَّهةٌ وتغطيةٌ ومحافظةٌ ومدينةٌ ومنطقةٌ ووقتُها ودوامُ متجر.
 *
 * **ولو رسمت كلُّ شاشةٍ نصَّها بنفسها لَافترقت الشاشاتُ** — **فتقول
 * السلّةُ شيئاً وتقول شاشةُ المتجر غيرَه عن اللحظة نفسِها.** **ومن
 * أضاف سبباً حادي عشرَ غداً يضيفه في موضعٍ وينساه في ثلاثة.**
 *
 * # والفرقُ بين رسالتين ليس تجميلاً
 *
 * **«رحّال غو لم يصل إلى محافظتك بعد» دعوةٌ للانتظار** — **يُبقي
 * التطبيقَ مثبَّتاً.**
 *
 * **و«عنوانُك خارج نطاق التوصيل» حكمٌ على العنوان** — **يُقال لمن
 * الخدمةُ مُطلَقةٌ في مدينته وعنوانُه خارجَ الشكل، فينقل دبّوسَه أو
 * ينتظر توسّعاً قريباً.**
 *
 * **ومن خلطهما أخبر ساكنَ دمشقَ أنّ عنوانَه بعيد** — **وهو ليس بعيداً،
 * بل لم نصل إلى مدينته أصلاً** — **فحذف التطبيق.**
 *
 * # ولا يُخترَع موعد
 *
 * **وخادمٌ لا يعرف متى يعود لا يُنطَق عنه** — **و«نعود قريباً» أصدقُ
 * من ساعةٍ لا نفي بها.**
 */
object ServiceReason {

    /** **أسماءُ الأسباب كما يرسلها المحرّك** — عقدٌ لا زينة. */
    const val AVAILABLE = "service_available"
    const val LAUNCH_CLOSED = "launch_closed"
    const val TEMPORARILY_UNAVAILABLE = "temporarily_unavailable"
    const val PLATFORM_CLOSED_NOW = "platform_closed_now"
    const val INVALID_LOCATION = "invalid_location"
    const val COVERAGE_UNAVAILABLE = "coverage_unavailable"
    const val PROVINCE_NOT_SUPPORTED = "province_not_supported"
    const val CITY_NOT_SUPPORTED = "city_not_supported"
    const val AREA_NOT_SUPPORTED = "area_not_supported"
    const val ADDRESS_OUTSIDE_COVERAGE = "address_outside_coverage"
    const val ZONE_CLOSED_NOW = "zone_closed_now"
    const val MERCHANT_CLOSED_NOW = "merchant_closed_now"

    /**
     * text **نصُّ السبب كما يقرؤه الزبون — ومعه الموعدُ إن عُرف.**
     *
     * **ونصُّ المالك يغلب نصَّ الحزمة** — **ونصٌّ مكتوبٌ في تطبيقٍ لا
     * يُصحَّح إلّا بنشرٍ في المتجر.**
     *
     * **واسمُ المكان يُستعمل حين يُعرَف** — **و«لم نصل إلى الرقّة بعد»
     * أوضحُ من «لم نصل إلى محافظتك».** **ولا يُسمّى ما لا يُعرَف.**
     */
    fun text(
        ctx: Context,
        reason: String,
        message: String = "",
        placeName: String = "",
        nextAvailableAt: String = "",
    ): String {
        val own = message.trim()
        val base = when {
            own.isNotEmpty() -> own
            reason == PROVINCE_NOT_SUPPORTED || reason == CITY_NOT_SUPPORTED ->
                named(ctx, placeName, reason)
            reason == AREA_NOT_SUPPORTED -> ctx.getString(R.string.av_area_not_supported)
            reason == ADDRESS_OUTSIDE_COVERAGE ->
                ctx.getString(R.string.av_address_outside_coverage)
            reason == INVALID_LOCATION -> ctx.getString(R.string.av_invalid_location)
            reason == MERCHANT_CLOSED_NOW -> ctx.getString(R.string.av_merchant_closed_now)
            reason == ZONE_CLOSED_NOW -> ctx.getString(R.string.err_zone_closed_now)
            reason == PLATFORM_CLOSED_NOW -> ctx.getString(R.string.err_platform_closed_now)
            reason == TEMPORARILY_UNAVAILABLE ->
                ctx.getString(R.string.err_temporarily_unavailable)
            reason == COVERAGE_UNAVAILABLE -> ctx.getString(R.string.err_coverage_unavailable)
            reason == LAUNCH_CLOSED -> ctx.getString(R.string.err_launch_closed)
            else -> ctx.getString(R.string.err_unexpected)
        }
        // **ولا موعدَ لبابٍ لم يُفتح بعد** — **ولا لِما لا يُعرَف رفعُه.**
        val back = backAtText(nextAvailableAt)
        return if (back == null) base else ctx.getString(R.string.err_back_at, base, back)
    }

    /** **باسم المكان إن عُرف، وبالعامّ إن لم يُعرَف.** */
    private fun named(ctx: Context, place: String, reason: String): String {
        val p = place.trim()
        if (p.isNotEmpty()) return ctx.getString(R.string.av_place_not_supported_named, p)
        return if (reason == CITY_NOT_SUPPORTED) {
            ctx.getString(R.string.av_city_not_supported)
        } else {
            ctx.getString(R.string.av_province_not_supported)
        }
    }

    /**
     * expansionPending **أهذا سببُ «لم نصل بعد»؟**
     *
     * **وتفرّقه الشاشةُ عن سائر الأسباب** — **فهذا مكانُ دعوةِ
     * الانتظار**، **وسائرُها مكانُ تصحيحٍ أو صبرٍ قصير.**
     *
     * **ولا زرَّ هنا اليوم** — **وزرٌّ لا يفعل شيئاً أسوأُ من لا زرّ**،
     * وطلبُ التوسّع في دفعةٍ قادمة.
     */
    fun expansionPending(reason: String): Boolean =
        reason == PROVINCE_NOT_SUPPORTED ||
            reason == CITY_NOT_SUPPORTED ||
            reason == AREA_NOT_SUPPORTED
}
