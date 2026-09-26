package com.rahalgo.ui

import android.content.Context
import android.util.Log
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.network.sockets.SocketTimeoutException
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رموزُ المحرّك بعربيّة — خريطةٌ واحدةٌ لا سبع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك ٢٠٢٦-٠٨-١٣: «ركّز جيّداً على المركزيّة بكلّ شيء».)
 *
 * # العطبُ الذي كشفه الجرد
 *
 * **كانت سبعُ خرائطَ في سبعة نماذج**، وكلٌّ تعرف رموزَ شاشتها وحدَها.
 * **ورمزٌ يخرج من نقطةٍ تناديها شاشتان يُترجَم في واحدةٍ ويُعرض خامّاً
 * في الأخرى.**
 *
 * **وقيس على الجهاز**: `whatsapp_required` مترجَمٌ في «حسابي»
 * **والمحرّكُ يردّه من `driver/shift`** — أي من مفتاح الورديّة في
 * «لوحتي». **فمن حاول فتحَ دوامه بلا توثيقٍ قرأ `whatsapp_required`
 * بالإنكليزيّة** ولم يعرف ما يفعل.
 *
 * # ولماذا خريطةٌ واحدة
 *
 * **الرمزُ ملكُ المحرّك لا ملكُ الشاشة** — ومعناه واحدٌ حيثما خرج.
 * **وسبعُ خرائطَ تعني أنّ إضافةَ رمزٍ في المحرّك تحتاج سبعَ إضافات**،
 * فتُنسى ستّ.
 *
 * # وما يبقى للشاشة
 *
 * **رمزٌ عامٌّ يعني في شاشةٍ شيئاً وفي أخرى غيرَه** — `validation` في
 * المحفظة «مبلغٌ غير صالح»، وفي غيرها «حقلٌ ناقص». **فتُمرَّر
 * استثناءاتُها وحدَها** (`extra`)، ويبقى الباقي مركزيّا.
 *
 * # والمجهولُ رسالةٌ عامّةٌ ويُبلَّغ عنه
 *
 * **كان يُعرض برمزه ليُرى ويُبلَّغ** — **وكان يُقرأ لاتينيّةً على شاشةٍ
 * عربيّة.** (قرارُ المالك، دورةُ ٤٢.)
 *
 * **ولا يُبتلع**: `Crash.soft` يرفعه كما كان — **فالبلاغُ باقٍ
 * والمستخدمُ لا يُطالَب بقراءة رمزٍ ليخدمنا.**
 */
fun apiError(
    context: Context,
    e: Throwable,
    extra: Map<String, Int> = emptyMap(),
): String {
    // **ووسمٌ واحدٌ لا وسمٌ لكلّ شاشة** — أثرُ الاستدعاء في السجلّ يقول
    // من نادى، **ووسمٌ عربيٌّ يُمرَّر وسيطاً يقع خارج نداء السجلّ**
    // فيشكو منه حارسُ النصوص بحقّ: نصٌّ عربيٌّ في شيفرة.
    // ══════════════════════════════════════════════════════════════════
    // **وحالٌ متوقَّعةٌ ليست عطباً يُسجَّل بأثره** (`B9`، ٢٠٢٦-٠٩-١٦)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وقِيس على المحاكي**: **فتحةُ ضيفٍ واحدةٌ تكتب ثلاثةَ أخطاءٍ
    // بأثرِ نداءٍ كاملٍ لكلٍّ منها** — **و`not_signed_in` حارسُنا نحن**
    // (لا ردُّ خادم): **يمنع رحلةَ شبكةٍ لا معنى لها.**
    //
    // **وأثرُ نداءٍ كاملٌ في الإقلاع كلفةُ معالجٍ ونصٍّ** — **ويغرق
    // السجلَّ فيخفي عطباً حقيقيّاً بجانبه.**
    if (e is ApiClient.ApiException && e.body.code == "not_signed_in") {
        Log.d("RahalGo/خطأ", "بابٌ مصادَقٌ نُودي بلا جلسة")
    } else {
        Log.e("RahalGo/خطأ", "فشل نداء", e)
    }
    if (e !is ApiClient.ApiException) {
        return when (e) {
            is IOException, is HttpRequestTimeoutException, is SocketTimeoutException ->
                context.getString(R.string.err_network)
            // ══════════════════════════════════════════════════════
            // **ولا يُطبَع اسمُ صنفٍ برمجيٍّ على شاشة** (`AB-24`)
            // ══════════════════════════════════════════════════════
            //
            // **وكان يُلحَق `e.javaClass.simpleName`** — **فقُرئ على
            // الشاشة**: «تعذّر إتمام الطلب
            // (NoTransformationFoundException)» — **وقِيس على المحاكي
            // ٢٠٢٦-٠٩-١٦** حين ردّ الخادمُ جسماً غيرَ JSON.
            //
            // **واسمُ الصنف لا يقول لصاحبه ما يفعل** — **ويقول لغيره
            // ما نستعمله.** **فيبقى في السجلّ** (`Log.e` أعلاه فيه
            // الأثرُ كاملاً) **ويُرفَع عن الشاشة.**
            else -> context.getString(R.string.err_unexpected)
        }
    }
    val code = e.body.code
    // ══════════════════════════════════════════════════════════════════
    // **وخطأُ الخادم يُسجَّل وإن لم يُسقط التطبيق**
    // ══════════════════════════════════════════════════════════════════
    //
    // **خمسمئةٌ تُعرض للسائق جملةً مهذّبةً وتمضي** — فيعيد الفعلَ ويعيد،
    // **ولا يعلم أحدٌ أنّ باباً في الخادم مكسور.**
    //
    // **ورمزٌ لا ترجمةَ له كذلك**: يُعرض خامّاً على شاشةٍ عربيّة — **وقد
    // وقع فعلاً** (`whatsapp_required`)، ولم يُكتشف إلّا بجردٍ يدويّ.
    // ══════════════════════════════════════════════════════════════════
    // **وبابٌ لم يُفتح بعد ليس عطباً** (٢٠٢٦-٠٩-١٣)
    // ══════════════════════════════════════════════════════════════════
    //
    // **و٥٠٣ تُقرأ عطبَ خادمٍ فتُسجَّل حادثةً** — **ووضعُ الإطلاق حالٌ
    // مقصودةٌ يضبطها المالك، لا انكسارٌ يُبلَّغ عنه.** **ولو سُجِّل
    // لَغرِق السجلُّ بمئاتٍ يوميّاً فيُفقَد فيه ما يعني شيئاً.**
    // ══════════════════════════════════════════════════════════════════
    // **وإغلاقُ الاستقبال ليس عطباً كذلك** (`PH`، ٢٠٢٦-٠٩-١٤)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وهما حالان يضبطهما المالك**: إيقافٌ مؤقّتٌ ودوامٌ انتهى —
    // **ولا يُسجَّلان حادثةً كما لا يُسجَّل بابُ الإطلاق.**
    //
    // **ونصُّ المالك يغلب نصَّ التطبيق** — والمفتاحُ `notice` عينُه.
    //
    // **وموعدُ العودة يُقرأ من الخادم ويُنسَّق هنا** — **والأهليّةُ
    // قُضيت هناك**: **هذا عرضٌ لا حكم.**
    if (code == "temporarily_unavailable" || code == "platform_closed_now" ||
        code == "zone_closed_now"
    ) {
        val notice = e.body.details["notice"].orEmpty().trim()
        val base = if (notice.isNotEmpty()) notice
        else context.getString(resolveErrorRes(code, e.body.messageKey, extra))
        val back = backAtText(e.body.details["next_available_at"])
        return if (back == null) base
        else context.getString(R.string.err_back_at, base, back)
    }
    if (code == "launch_closed") {
        // **ونصُّ المالك يغلب نصَّ التطبيق** — **ونصٌّ مكتوبٌ في حزمةٍ
        // لا يُصحَّح إلّا بنشرٍ في المتجر.** وفارغُه يقع على نصّ الرمز.
        val notice = e.body.details["notice"].orEmpty().trim()
        if (notice.isNotEmpty()) return notice
        return context.getString(R.string.err_launch_closed)
    }
    if (e.status >= 500 || (code !in CODES && code !in extra && code.isNotEmpty())) {
        Crash.soft(e, "api " + e.status + " " + code)
    }
    return context.getString(resolveErrorRes(code, e.body.messageKey, extra))
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحُ الرسالة قارئٌ له — والمجهولُ لا يُعرض خامّاً** — `XG-45`
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما كان يقع
 *
 * **الخريطةُ تعرف الرمزَ ولا تعرف مفتاحَ الرسالة**، **ورمزٌ لا ترجمةَ
 * له كان يُعرض كما هو**: `auth_unavailable` لاتينيّةً على شاشةٍ
 * عربيّة. **وقِيس: خمسةَ عشرَ رمزاً يبلغ الهاتفَ بلا نصّ.**
 *
 * # والترتيبُ ثلاثةٌ لا واحد
 *
 *	١ استثناءُ الشاشة  — `validation` في المحفظة غيرُها في نموذج
 *	٢ رمزُ المحرّك     — `code`
 *	٣ مفتاحُ رسالته    — `errors.x` ثمّ ذيلُه
 *
 * **والثالثُ وُلد هنا**: **المحرّكُ يرسل الاثنين**، ولو تبدّل رمزٌ
 * وبقي مفتاحُه لَبقي النصُّ يصل.
 *
 * # والمجهولُ رسالةٌ عامّةٌ لا رمزٌ خام
 *
 * **وكان يُعرض برمزه ليُبلَّغ عنه** — **والبلاغُ باقٍ في `Crash.soft`
 * أعلاه**، **والمستخدمُ لا يُطالَب بقراءة لاتينيّةٍ ليخدمنا.**
 *
 * **ودالّةٌ صافيةٌ بلا سياق** — **فتُقاس على المُفسِّر بلا جهاز.**
 */
fun resolveErrorRes(
    code: String,
    messageKey: String = "",
    extra: Map<String, Int> = emptyMap(),
): Int {
    extra[code]?.let { return it }
    CODES[code]?.let { return it }
    if (messageKey.isNotEmpty()) {
        val tail = messageKey.substringAfterLast('.')
        extra[tail]?.let { return it }
        CODES[tail]?.let { return it }
    }
    return R.string.err_internal
}

/**
 * **كلُّ رمزٍ يمكن أن يصل هذا التطبيق.**
 *
 * **ومصدرُها المحرّكُ لا الشاشة** — `httpx.NewError(...)` في الأبواب
 * التي يناديها السائق.
 */
private val CODES: Map<String, Int> = mapOf(
    // ── الجلسة والصلاحيّة ──────────────────────────────────────────────
    "unauthorized" to R.string.err_invalid_refresh,
    // **ومن لم يدخل قطّ لا «انتهت جلستُه»** — يُقال له إنّ هذا يحتاج
    // حسابا. (قِيس ٢٠٢٦-٠٨-١٤.)
    "not_signed_in" to R.string.err_not_signed_in,
    "invalid_refresh" to R.string.err_invalid_refresh,
    // **ودخولٌ من جهازٍ آخر يُقال بصريحه** (Obs 3) — لا «انتهت جلستُك» العامّة.
    "session_superseded" to R.string.err_session_superseded,
    // ══════════════════════════════════════════════════════════════════
    // **و«ممنوع» لا تعني «لستَ سائقا»**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كانت `forbidden` تُترجَم «هذا الحسابُ ليس حساب سائق»** — وهذا
    // المعجمُ تقرؤه التطبيقاتُ الثلاثة. **فقرأها المندوبُ وهو يرفع صورةَ
    // صنفٍ لعميله** (قِيس على الجهاز ٢٠٢٦-٠٨-١٨).
    //
    // **ورسالةٌ تقول شيئاً غيرَ ما وقع أسوأُ من رسالةٍ عامّة**: تُرسل
    // صاحبَها يبحث في حسابه عن عطبٍ ليس فيه.
    "forbidden" to R.string.err_forbidden,
    "user_blocked" to R.string.err_user_blocked,
    "user_suspended" to R.string.err_user_suspended,
    "internal" to R.string.err_internal,
    // **تعذّر بناءُ حمولةٍ آمنة** — والخادمُ يسقط مغلقاً بدل أن
    // يُرسل ما لا يجوز. **ونصُّه نصُّ العطب**: لا شيءَ يفعله صاحبُ
    // الجهاز، **ووصفُ داخلِنا يُقلق ولا يُفيد.**
    "payload_unsafe" to R.string.err_internal,
    "too_many_requests" to R.string.err_rate_limited,
    "rate_limited" to R.string.err_rate_limited,
    "too_many_attempts" to R.string.err_rate_limited,
    "validation" to R.string.err_validation,
    "update_required" to R.string.err_update_required,
    // **وكلُّ سببٍ باسمه** — انظر `orders/models.go`.
    "no_items" to R.string.err_no_items,
    "no_address" to R.string.err_no_address,
    "bad_merchant" to R.string.err_bad_merchant,
    "bad_payment" to R.string.err_bad_payment,
    "bad_qty" to R.string.err_bad_qty,
    "item_gone" to R.string.err_item_gone,
    // ── الدخول ────────────────────────────────────────────────────────
    "invalid_credentials" to R.string.err_unauthorized,
    "otp_send_failed" to R.string.err_otp_send_failed,
    // **ورقمٌ بلا واتساب لا يصله رمز** — وهو أوّلُ ما يلقاه من يُسجّل،
    // **فظهر رمزُه خاماً (`no_whatsapp`) في أوّل تجربة** (٢٠٢٦-٠٨-١٤).
    "no_whatsapp" to R.string.err_no_whatsapp,
    // ── الورديّة والطلبات ─────────────────────────────────────────────
    "whatsapp_required" to R.string.err_whatsapp_required,
    "has_active_orders" to R.string.err_has_active_orders,
    "order_taken" to R.string.err_order_taken,
    "offer_not_yours" to R.string.err_offer_not_yours,
    "not_on_shift" to R.string.err_not_on_shift,
    "cash_limit_reached" to R.string.err_cash_limit,
    "too_many_active_orders" to R.string.err_too_many_active,
    "not_your_order" to R.string.err_not_your_order,
    "delivery_proof_required" to R.string.err_proof_required,
    "bad_fail_reason" to R.string.err_bad_fail_reason,
    // ── الحساب ────────────────────────────────────────────────────────
    "wrong_password" to R.string.acc_wrong_password,
    // **وكلمةُ المرور الحاليّة بعينها** — كان المحرّكُ يردّ
    // `invalid_credentials` فتُقرأ «رقمُ الهاتف أو كلمةُ المرور غير
    // صحيحة»، **ولا رقمَ في نموذج التبديل أصلا.**
    "wrong_current_password" to R.string.acc_wrong_password,
    "weak_password" to R.string.acc_weak_password,
    // ── موانعُ حذف الحساب ─────────────────────────────────────────────
    //
    // **خمسةٌ كانت تُعرض خامّةً** — والزرُّ يبدو معطَّلاً لصاحبه.
    "wallet_not_empty" to R.string.err_wallet_not_empty,
    "cash_not_settled" to R.string.err_cash_not_settled,
    "owns_merchants" to R.string.err_owns_merchants,
    "open_orders" to R.string.err_open_orders,
    "admin_cannot_delete" to R.string.err_admin_cannot_delete,
    // **ورمزُ المحرّك `invalid_otp` لا `otp_invalid`** — كان الاسمُ
    // مقلوباً في الخريطة، **فيسقط إلى عرض الرمز الخام** ويقرأ صاحبُه
    // إنجليزيّةً لا تعني له شيئاً. (شكوى المالك ٢٠٢٦-٠٨-١٨.)
    "invalid_otp" to R.string.acc_bad_code,
    "otp_invalid" to R.string.acc_bad_code,
    "invalid_code" to R.string.acc_bad_code,
    "phone_taken" to R.string.acc_phone_taken,
    "phone_exists" to R.string.acc_phone_taken,
    "invalid_phone" to R.string.err_invalid_phone,
    "bad_phone" to R.string.err_invalid_phone,
    "invalid_image" to R.string.acc_photo_bad,
    "image_dimensions" to R.string.acc_photo_too_large,
    "image_too_large" to R.string.acc_photo_too_large,
    "upload_failed" to R.string.acc_photo_failed,
    // ── المال ─────────────────────────────────────────────────────────
    "insufficient_balance" to R.string.wal_not_enough,
    "payout_pending" to R.string.wal_pending_exists,
    // ── السجلّ والبلاغات ──────────────────────────────────────────────
    "complaint_already_open" to R.string.hist_already_open,
    "already_rated" to R.string.hist_already_rated,
    // **ولا متجرَ في الطلب الخاصّ** — ورمزٌ يصل بلا ترجمةٍ يُعرض
    // لاتينيّاً على شاشةٍ عربيّة.
    "no_merchant" to R.string.err_no_merchant,
    "complaint_window_passed" to R.string.err_complaint_window,
    "order_not_closed" to R.string.err_order_not_closed,
    "bad_complaint_reason" to R.string.err_bad_fail_reason,
    // ── الباقي: كلُّ رمزٍ يردّه المحرّكُ ويصل هاتفا ──────────────────
    //
    // **كشفها `check-app-error-codes`** — وكانت تُعرض خامّة.
    "invalid_role" to R.string.err_invalid_role,
    "self_action" to R.string.err_self_action,
    // **ورمزا حارسِ الدور المحميّ** (٢٠٢٦-٠٩-١٢) — **بابُهما إدارةٌ
    // لا هاتف**، **ونصٌّ هنا ثمنُه سطران** ويمنع لاتينيّةً خامّةً على
    // شاشةٍ عربيّةٍ إن بلغها يوماً.
    "owner_role_protected" to R.string.err_owner_role_protected,
    "last_owner" to R.string.err_last_owner,
    "role_not_creatable" to R.string.err_role_not_creatable,
    "role_grant_retired" to R.string.err_role_grant_retired,
    "pin_required" to R.string.err_pin_required,
    "pin_invalid" to R.string.err_pin_invalid,
    "pin_locked" to R.string.err_pin_locked,
    "pin_format" to R.string.err_pin_format,
    "pin_challenge" to R.string.err_pin_challenge,
    "pin_not_set" to R.string.err_pin_not_set,
    "whatsapp_unverified" to R.string.err_whatsapp_unverified,
    "role_conflict" to R.string.err_role_conflict,
    "merchant_needs_store" to R.string.err_merchant_needs_store,
    "name_too_short" to R.string.err_name_too_short,
    "invalid_invite_code" to R.string.err_invalid_invite_code,
    "not_custom_order" to R.string.err_not_custom_order,
    "custom_not_assigned" to R.string.err_custom_not_assigned,
    "custom_not_agreed" to R.string.err_custom_not_agreed,
    // ── عرضُ السعر المخصَّص وتأكيدُه (Batch 2a) ──
    "quote_changed" to R.string.err_quote_changed,
    "quote_not_confirmed" to R.string.err_quote_not_confirmed,
    "custom_locked" to R.string.err_custom_locked,
    "merchant_closed" to R.string.err_merchant_closed,
    // **والدفعُ نقداً موقوفٌ مؤقّتاً** — (٢٠٢٦-٠٨-٢٣). **والرسالةُ
    // تقول البديل**: المحفظةُ مفتوحةٌ له.
    "cash_blocked" to R.string.err_cash_blocked,
    // **وسقفُ نقد الزبون غيرِ المسدَّد** — غيرُ سقف السائق (`cash_limit_exceeded`):
    // **قيمةُ الطلب تعبر ما يُسمح به نقداً عند الاستلام**، والمحفظةُ بديلٌ.
    "cod_limit_exceeded" to R.string.err_cod_limit_exceeded,
    "multi_source_order" to R.string.err_multi_source_order,
    "too_many_sources" to R.string.err_too_many_sources,
    "item_unavailable" to R.string.err_item_unavailable,
    "invalid_items" to R.string.err_invalid_items,
    "out_of_zone" to R.string.err_out_of_zone,
    // ── السلطةُ الإداريّةُ عند الإنشاء (Batch 3a) ──
    //
    // **محافظةٌ/مدينةٌ لم تُطلَق، أو موضعٌ لا مدينةَ تحويه** — صار الإنشاءُ
    // يردّها (كانت للقراءة وحدَها)، **وبنصوصِ الإتاحة نفسِها** (`ServiceReason`)
    // فلا يفترق رفضُ الإنشاء عن شرحِ الشاشة.
    "province_not_supported" to R.string.av_province_not_supported,
    "city_not_supported" to R.string.av_city_not_supported,
    "area_not_supported" to R.string.av_area_not_supported,
    "below_min_order" to R.string.err_below_min_order,
    "too_many_open_orders" to R.string.err_too_many_open_orders,
    "invalid_promo" to R.string.err_invalid_promo,
    "invalid_transition" to R.string.err_invalid_transition,
    "driver_required" to R.string.err_driver_required,
    "cancel_window_passed" to R.string.err_cancel_window_passed,
    "not_delivered" to R.string.err_not_delivered,
    "invalid_stars" to R.string.err_invalid_stars,
    "rate_own_client" to R.string.err_rate_own_client,
    "invalid_amount" to R.string.err_invalid_amount,
    "invalid_category" to R.string.err_invalid_category,
    "merchant_location_required" to R.string.err_merchant_location_required,
    "owner_required" to R.string.err_owner_required,
    "invalid_hours" to R.string.err_invalid_hours,
    "code_taken" to R.string.err_code_taken,
    "section_not_empty" to R.string.err_section_not_empty,
    "section_required" to R.string.err_section_required,
    // **سببُ نقل المتجر بين المندوبَين إلزاميّ** — بابُ أدمن (لا يبلغ الهاتفَ
    // عمليّاً)، لكنّ حارسَ الرموز يفحص حزمةَ `catalog` فيلزمه تعيينٌ ونصّ.
    "transfer_reason_required" to R.string.err_transfer_reason_required,
    "invalid_zone" to R.string.err_invalid_zone,
    "over_settle" to R.string.err_over_settle,
    "cash_limit_exceeded" to R.string.err_cash_limit_exceeded,
    // ── رموزٌ تولد في `internal/server` وتبلغ الهاتف — `XG-45` ────────
    //
    // **وحارسُ الرموز كان يتخطّى `server` كلَّها** لأنّ فيها أبوابَ
    // الإدارة — **فسقط منها ما يمرّ به كلُّ نداءٍ من هاتف.**
    //
    // **وأوّلُها `auth_unavailable`**: يخرج من وسيط التوثيق نفسِه
    // (`R16`)، **فيبلغ التطبيقاتِ الأربعةَ جميعاً.**
    // **وبابُ الإطلاق** — **يبلغ الأربعةَ كلَّها**: الزبونُ عند الطلب،
    // والسائقُ عند بدء الدوام، والمندوبُ عند ضمّ متجر.
    //
    // **وهو ٥٠٣ لا ٤٠٣**: **«ليس الآن» لا «لستَ أهلاً»** — ومن خلطهما
    // أخبر الزبونَ أنّه ممنوعٌ وهو مسموحٌ غداً.
    "coverage_unavailable" to R.string.err_coverage_unavailable,
    "bad_point" to R.string.err_bad_point,
    "launch_closed" to R.string.err_launch_closed,
    "temporarily_unavailable" to R.string.err_temporarily_unavailable,
    "platform_closed_now" to R.string.err_platform_closed_now,
    "zone_closed_now" to R.string.err_zone_closed_now,
    "service_now_available" to R.string.err_service_now_available,
    "reason_mismatch" to R.string.err_reason_mismatch,
    "password_change_required" to R.string.err_password_change_required,
    "auth_unavailable" to R.string.err_auth_unavailable,
    // **CUST-DEF-002: «قيد المعالجة» يُقال صريحاً لا «تعذّر الاتصال».**
    // **وكان `in_progress` بلا خانةٍ فيقع على `err_internal` («تعذّر
    // الاتصال — حاول بعد قليل») — فيُغري بإعادةٍ تُنشئ طلباً ثانياً**،
    // **ويُسجَّل حادثةً زائفةً** (`code !in CODES`). والرسالةُ الصحيحةُ
    // «العملية قيد التنفيذ — انتظر قليلا ولا تعدها».
    "in_progress" to R.string.err_in_progress,
    "idempotency_reclaimed" to R.string.err_in_progress,
    // **CAF-02: «مفتاحٌ لجسمين» يُقال صريحاً لا «تعذّر الاتصال».**
    // **عُدِّلت السلّةُ ومحاولةٌ سابقةٌ لم تُحسَم** — **لا يُنشأ طلبٌ
    // ثانٍ فوق سابقٍ قد نجح**؛ يُوجَّه إلى «طلباتي» قبل محاولةٍ جديدة.
    "idempotency_key_reused" to R.string.err_key_reused,
    "too_many_addresses" to R.string.err_too_many_addresses,
    "already_returned" to R.string.err_already_returned,
    "merchant_no_returns" to R.string.err_merchant_no_returns,
    "order_not_returnable" to R.string.err_order_not_returnable,
    "not_readyable" to R.string.err_not_readyable,
    "reason_required" to R.string.err_reason_required,
    "never_picked_up" to R.string.err_never_picked_up,
    "payout_below_min" to R.string.err_payout_below_min,
    "payout_closed" to R.string.err_payout_closed,
    "payout_not_allowed" to R.string.err_payout_not_allowed,
    "invite_required" to R.string.err_invite_required,
    "lead_already_converted" to R.string.err_lead_already_converted,
    "bad_bbox" to R.string.err_bad_request,
    // ── المحادثةُ والوجود ─ `CAF-18` (`CUST-20-019`) ───────────────────
    //
    // **ثلاثةُ رموزٍ كانت تسقط على `err_internal` («تعذّر الاتصال — حاول
    // بعد قليل»)** — **ورسالةٌ تقول «تعذّر الاتصال» لعطبٍ ليس اتصالاً
    // تُرسل صاحبَها يفحص شبكتَه بلا داعٍ.** **فلكلٍّ نصُّه**: القناةُ
    // مغلقةٌ لأنّ الطلبَ انتهى، أو لا سائقَ بعدُ، أو لم يُوجَد المطلوب.
    "not_found" to R.string.err_not_found,
    "comms_closed" to R.string.err_comms_closed,
    "comms_no_driver" to R.string.err_comms_no_driver,
)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **`err` — سببُ الخطأ بالعربيّة، من أيّ مكان**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «لا أريد نصوصاً أجنبيّةً غير مفهومة ولا
 *  مفاتيح… وتأكّد أنّ رسائل النجاح والفشل مبنيّةٌ بشكلٍ مركزيٍّ
 *  وصحيح».)
 *
 * # ولماذا وُجدت
 *
 * **و`apiError` تحتاج `Context`** — فمن كان في `ViewModel` بلا
 * `Application` كتب `it.message ?: "تعذّر…"` **وانتهى.**
 *
 * **وقِيس في تطبيق المتجر ٢٠٢٦-٠٨-٢٦**: عشرون موضعاً هكذا،
 * **و`it.message` في نداءٍ فاشلٍ هو رمزُ الخادم الخام** — فيقرأ صاحبُ
 * المتجر `merchant_closed` أو `validation` بدل جملةٍ يفهمها.
 *
 * **والنصُّ الاحتياطيُّ كان يُخفي ذلك عنّا**: يبدو عربيّاً في الشيفرة،
 * **ولا يُعرض إلّا حين تكون الرسالةُ فارغة** — وهي نادراً ما تكون.
 *
 * # ولا تحتاج سياقا
 *
 * **وتقرأ السياقَ من التطبيق نفسِه** (`AppCore`) — فتُنادى من أيّ
 * موضع، **ولا يبقى عذرٌ لكتابة رسالةٍ بيد.**
 */
fun err(e: Throwable): String = apiError(AppCore.get().app, e)

/**
 * backAtText **موعدُ العودة نصّاً قصيراً** — و`null` إن لم يُعرَف.
 *
 * **ولا يُخترَع موعد**: **خادمٌ لا يعرف متى يعود لا يُنطَق عنه.**
 *
 * **ويُنسَّق بمنطقة الجهاز** — **وهي منطقةُ من يقرأ**: **والحكمُ وقع
 * في الخادم قبل أن يصل هذا النصّ.**
 *
 * **وتاريخٌ لا يُحلَّل يُبتلَع** — **ورسالةُ «ليس الآن» أنفعُ من رسالةِ
 * عطبٍ لأنّ حقلاً جاء مشوَّهاً.**
 */
fun backAtText(raw: String?): String? {
    val v = raw?.trim().orEmpty()
    if (v.isEmpty()) return null
    return try {
        val at = java.time.OffsetDateTime.parse(v)
        java.time.format.DateTimeFormatter
            .ofPattern("h:mm a", java.util.Locale("ar"))
            .format(at.atZoneSameInstant(java.time.ZoneId.systemDefault()))
    } catch (_: java.time.format.DateTimeParseException) {
        null
    }
}
