package com.rahalgo.driver.data

import android.content.Context
import android.util.Log
import com.rahalgo.driver.R
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
    "invalid_refresh" to R.string.err_invalid_refresh,
    "forbidden" to R.string.err_not_driver,
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
    "weak_password" to R.string.acc_weak_password,
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
    "complaint_window_passed" to R.string.err_complaint_window,
    "order_not_closed" to R.string.err_order_not_closed,
    "bad_complaint_reason" to R.string.err_bad_fail_reason,
)
