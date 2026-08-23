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
 * # والمجهولُ يُعرض برمزه
 *
 * **لا يُبتلع**: من رآه أبلغ عنه، **ومن ابتلعه ترك شاشةً صامتةً لا
 * يُعرف سببُها.**
 */
fun apiError(
    context: Context,
    e: Throwable,
    extra: Map<String, Int> = emptyMap(),
): String {
    // **ووسمٌ واحدٌ لا وسمٌ لكلّ شاشة** — أثرُ الاستدعاء في السجلّ يقول
    // من نادى، **ووسمٌ عربيٌّ يُمرَّر وسيطاً يقع خارج نداء السجلّ**
    // فيشكو منه حارسُ النصوص بحقّ: نصٌّ عربيٌّ في شيفرة.
    Log.e("RahalGo/خطأ", "فشل نداء", e)
    if (e !is ApiClient.ApiException) {
        return when (e) {
            is IOException, is HttpRequestTimeoutException, is SocketTimeoutException ->
                context.getString(R.string.err_network)
            else ->
                context.getString(R.string.err_unexpected) + " (" + e.javaClass.simpleName + ")"
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
    if (e.status >= 500 || (code !in CODES && code !in extra && code.isNotEmpty())) {
        Crash.soft(e, "api " + e.status + " " + code)
    }
    extra[code]?.let { return context.getString(it) }
    val res = CODES[code] ?: return if (code.isEmpty()) {
        context.getString(R.string.err_internal)
    } else {
        code
    }
    return context.getString(res)
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
    "too_many_requests" to R.string.err_rate_limited,
    "rate_limited" to R.string.err_rate_limited,
    "too_many_attempts" to R.string.err_rate_limited,
    "validation" to R.string.err_validation,
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
    "merchant_closed" to R.string.err_merchant_closed,
    // **والدفعُ نقداً موقوفٌ مؤقّتاً** — (٢٠٢٦-٠٨-٢٣). **والرسالةُ
    // تقول البديل**: المحفظةُ مفتوحةٌ له.
    "cash_blocked" to R.string.err_cash_blocked,
    "multi_source_order" to R.string.err_multi_source_order,
    "too_many_sources" to R.string.err_too_many_sources,
    "item_unavailable" to R.string.err_item_unavailable,
    "invalid_items" to R.string.err_invalid_items,
    "out_of_zone" to R.string.err_out_of_zone,
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
    "invalid_zone" to R.string.err_invalid_zone,
    "over_settle" to R.string.err_over_settle,
    "cash_limit_exceeded" to R.string.err_cash_limit_exceeded,
)
