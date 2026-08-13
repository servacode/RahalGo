package com.rahalgo.shared.auth

import com.rahalgo.shared.model.AuthResult
import com.rahalgo.shared.model.Platform
import com.rahalgo.shared.model.SiteContact
import com.rahalgo.shared.model.User
import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod

/**
 * **أبواب الدخول — كما هي في عقد الـAPI.**
 *
 * (`api/contract.json`: مسارات `auth` — الدخول والرمز ومَن أنا والخروج.)
 *
 * **ولا تُكتب نجمة بعد شرطة مائلة داخل تعليق**: `/` ثمّ `*` يفتحان
 * تعليقا متداخلا في Kotlin **فيبتلع بقية الملفّ** — والرسالة تقول
 * «تعليق لم يُغلق» عند آخر سطر، لا عند موضع العطب. (وقع مرّتين
 * ٢٠٢٦-٠٨-١١، ويحرسه `check-kotlin-comments`.)
 *
 * **ولا منطق عمل هنا** (`GROUND-RULES.md` §7.2 البند ٥): من يدخل وأين
 * يذهب يقرّره المحرّك والأدوار — **وهذه تنادي وتُعيد ما قيل.**
 */
class AuthApi(private val api: ApiClient) {

    /** دخول بكلمة المرور — **الباب الافتراضي، كما في الويب.** */
    suspend fun login(phone: String, password: String): AuthResult =
        api.raw(
            "/api/v1/auth/login",
            HttpMethod.Post,
            mapOf("phone" to phone, "password" to password),
        )

    /** طلب رمز لمرّة واحدة. */
    suspend fun requestOtp(phone: String): Unit =
        api.raw<Ack>(
            "/api/v1/auth/otp/request",
            HttpMethod.Post,
            mapOf("phone" to phone),
        ).let { }

    /** تأكيد الرمز — **يفتح الجلسة مباشرة.** */
    suspend fun verifyOtp(phone: String, code: String): AuthResult =
        api.raw(
            "/api/v1/auth/otp/verify",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code),
        )

    // ══════════════════════════════════════════════════════════════════
    // **استعادة كلمة المرور — ثلاث خطوات لا واحدة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **والفصل مقصود**: الرمز يُتحقّق منه **قبل** نموذج الكلمة الجديدة
    // (`auth/password/reset/verify`)، **فلا يكتب صاحبه كلمة جديدة ثمّ
    // يُقال له إن رمزه خطأ** فيعيد كل شيء.
    //
    // **والتأكيد يفتح جلسة مباشرة** (`ConfirmPasswordReset` يعيد
    // `AuthResult`) — فلا يُطلب منه أن يدخل بما ضبطه للتوّ.

    /** يطلب رمز استعادة. **ورقم غير مسجّل يردّ نجاحا صامتا** — لئلّا
     *  يُعرف من هذا الباب أيّ الأرقام لها حسابات. */
    suspend fun resetRequest(phone: String): Unit =
        api.raw<Ack>(
            "/api/v1/auth/password/reset/request",
            HttpMethod.Post,
            mapOf("phone" to phone),
        ).let { }

    /** يتحقّق من الرمز **ولا يستهلكه** — الخطوة التالية تعيده. */
    suspend fun resetVerify(phone: String, code: String): Unit =
        api.raw<Ack>(
            "/api/v1/auth/password/reset/verify",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code),
        ).let { }

    /** يضبط الكلمة الجديدة **ويفتح الجلسة.** */
    suspend fun resetConfirm(phone: String, code: String, password: String): AuthResult =
        api.raw(
            "/api/v1/auth/password/reset/confirm",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code, "password" to password),
        )

    /**
     * **من أنا؟** — يُنادى عند الإقلاع لاستعادة الجلسة.
     *
     * **ونجاحه هو الدليل على أن التوكن حيّ** — لا وجودُه في التخزين:
     * توكن مُبطَل من الإدارة يبقى مكتوبا، **ومن اكتفى بوجوده أرى صاحبَه
     * شاشة داخل ثمّ سقط كل نداء بعدها.**
     */
    suspend fun me(): User = api.call("/api/v1/auth/me")

    /** حال المنصّة — **يُنادى قبل رسم شاشة الدخول.** */
    suspend fun platform(): Platform = api.raw("/api/v1/public/platform")

    /**
     * **هويّةُ المنصّة ونصوصُ صفحاتها** — التعليماتُ ومن نحن والشروطُ
     * والخصوصيّة.
     *
     * (`GET /api/v1/public/contact` — النداءُ الذي تقرؤه صفحاتُ الويب
     * القانونيّة.)
     *
     * **وعامّةٌ لا محميّة**: من يُسأل أن يوافق على الشروط يقرؤها قبل أن
     * يدخل، **ووثيقةٌ لا تُقرأ إلّا بعد التسجيل يُوافَق عليها بلا
     * قراءة.**
     */
    suspend fun contact(): SiteContact = api.raw("/api/v1/public/contact")

    /** خروج — **يُبطل الجلسة في المحرّك لا في الجهاز وحده.** */
    suspend fun logout(refreshToken: String) {
        api.raw<Ack>(
            "/api/v1/auth/logout",
            HttpMethod.Post,
            mapOf("refresh_token" to refreshToken),
        )
    }
}
