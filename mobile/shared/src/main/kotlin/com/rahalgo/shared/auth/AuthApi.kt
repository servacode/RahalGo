package com.rahalgo.shared.auth

import com.rahalgo.shared.model.AuthResult
import com.rahalgo.shared.model.Platform
import com.rahalgo.shared.model.SiteContact
import com.rahalgo.shared.model.User
import com.rahalgo.shared.model.WaTicket
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

    /**
     * ══════════════════════════════════════════════════════════════════
     * **تذكرةُ واتساب — والزبونُ يبدأ المحادثة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٥ بعد أن قُيّد رقمُ المنصّة مرّتين في يوم.)
     *
     * **وواتساب يمنع الحساباتِ الشخصيّةَ من مراسلة من لم يراسلها** —
     * **والردُّ مسموح.** فتُفتح تذكرةٌ، ويُرسل صاحبُها رسالةً جاهزةً
     * فيها وسمُها، **فيردّ البوتُ بالرمز.**
     *
     * **والرابطُ يأتي من المحرّك لا يُبنى هنا**: رقمُ المنصّة إعدادٌ
     * يُبدَّل في اللوحة، **ونصٌّ يُبنى في أربعة تطبيقاتٍ يختلف أربعَ
     * مرّات.**
     *
     * `purpose` — `verify` للتوثيق · `reset` لاستعادة كلمة المرور.
     */
    suspend fun waTicket(phone: String, purpose: String = "verify"): WaTicket =
        api.raw(
            "/api/v1/auth/wa/ticket",
            HttpMethod.Post,
            mapOf("phone" to phone, "purpose" to purpose),
        )

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

    // ══════════════════════════════════════════════════════════════════
    // **إنشاء حساب — ثلاث خطوات، وللزبون وحدَه**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا يُنشئ هذا البابُ سائقاً ولا متجراً** (`auth_handlers.go`:
    // «إنشاء حساب زبون فقط»): حساباتُ العاملين من المنصّة،
    // **ومن فتح بابَ التسجيل لكلّ دورٍ فتح بابَ من يُعطي نفسَه دورا.**
    //
    // **والخطوةُ الوسطى مقصودة** — تتحقّق من الرمز ولا تستهلكه: قرارُ
    // المالك ٢٠٢٦-٠٨-٠٦ «لا تظهر المعلوماتُ إلّا بعد التحقّق من الرمز».
    // **فلا يكتب اسمَه وكلمتَه ثمّ يُقال له إنّ رمزَه خطأ** فيعيد كلَّ
    // شيء.

    /** يطلب رمزَ تسجيل. */
    suspend fun signupRequest(phone: String): Unit =
        api.raw<Ack>(
            "/api/v1/auth/signup/request",
            HttpMethod.Post,
            mapOf("phone" to phone),
        ).let { }

    /** يتحقّق من الرمز **ولا يستهلكه** — الخطوة التالية تعيده. */
    suspend fun signupVerify(phone: String, code: String): Unit =
        api.raw<Ack>(
            "/api/v1/auth/signup/verify",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code),
        ).let { }

    /**
     * **يُنشئ الحسابَ ويفتح الجلسة** — فلا يدخل بما ضبطه للتوّ.
     *
     * @param ref رمزُ من دعاه — **اختياريّ**، ومن سجّل بلا دعوةٍ حسابُه
     *  كامل. **ورمزٌ خاطئٌ لا يُسقط تسجيلا**: يُحرَم المكافأةَ وحدَها.
     */
    suspend fun signupConfirm(
        phone: String,
        code: String,
        fullName: String,
        password: String,
        ref: String = "",
    ): AuthResult =
        api.raw(
            "/api/v1/auth/signup/confirm",
            HttpMethod.Post,
            mapOf(
                "phone" to phone,
                "code" to code,
                "full_name" to fullName,
                "password" to password,
                "ref" to ref,
            ),
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
