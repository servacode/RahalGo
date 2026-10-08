package com.rahalgo.ui

import android.content.Context

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رمزٌ أُرسل ولم يُكتب بعد — يبقى إن أغلق أندرويد التطبيق** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «إذا شخص وصله كود راح ع واتس جاب الكود ورجع يظل بنفس الصفحة مايضطر يبعث
 * كود جديد.» — **والخطوةُ في النموذج تبقى ما دام التطبيقُ حيّاً**، لكنّ الجوالَ
 * الضعيفَ يُغلقه في الخلفية وصاحبُه في واتساب، **فيعود إلى أوّل الشاشة ويطلب
 * رمزاً ثانياً وينتظر مهلةَ الإعادة.**
 *
 * فيُحفظ على الجهاز **نوعُ الشاشة والرقمُ ولحظةُ الإرسال** — لا الرمز —
 * **ويُنسى بعد عشر دقائق** (الرمزُ نفسُه انتهى) أو حين يُدخَل أو يُغلق الباب.
 */
object PendingCode {
    const val LOGIN = "login"
    const val SIGNUP = "signup"
    const val RESET = "reset"

    private const val PREFS = "rahal_pending_code"
    private const val TTL_MS = 10 * 60 * 1000L

    data class Pending(val kind: String, val phone: String, val sentAt: Long, val referral: String)

    private fun prefs() =
        AppCore.get().app.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun save(kind: String, phone: String, sentAt: Long, referral: String = "") {
        runCatching {
            prefs().edit()
                .putString("kind", kind)
                .putString("phone", phone)
                .putLong("sent_at", sentAt)
                .putString("referral", referral)
                .apply()
        }
    }

    fun load(): Pending? = runCatching {
        val p = prefs()
        val kind = p.getString("kind", null) ?: return@runCatching null
        val sentAt = p.getLong("sent_at", 0L)
        val age = System.currentTimeMillis() - sentAt
        if (age !in 0..TTL_MS) {
            clear()
            return@runCatching null
        }
        Pending(kind, p.getString("phone", "").orEmpty(), sentAt, p.getString("referral", "").orEmpty())
    }.getOrNull()

    fun clear() {
        runCatching { prefs().edit().clear().apply() }
    }
}
