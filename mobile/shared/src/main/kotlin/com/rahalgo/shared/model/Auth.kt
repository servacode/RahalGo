package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نماذج المصادقة — أسماؤها من عقد الـAPI لا من الذاكرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`api/contract.json` — مولَّد من شيفرة المحرّك بـ`go run ./cmd/apidoc`.)
 *
 * **والاسم في الشبكة غير الاسم في Kotlin**: `full_name` هناك و`fullName`
 * هنا. **ومن كتبه من رأسه يبني حقلا لا وجود له** — ولا خطأ في بناء ولا
 * سطر في سجل، **إنما شاشة تعرض فراغا.**
 *
 * # وكل رد يأتي داخل `data`
 *
 * المحرّك يلفّ كل رد ناجح في `{"data": …}` والخطأ في `{"error": …}`
 * (`httpx.JSON` و`httpx.Error`). **فلا يُفكّ الرد مباشرة** — يُفكّ الغلاف
 * أولا. وهذا ما تفعله `Envelope`.
 */

/** غلاف كل رد من المحرّك. */
@Serializable
data class Envelope<T>(
    val data: T? = null,
    val error: ApiErrorBody? = null,
)

/**
 * جسم الخطأ كما يرسله المحرّك.
 *
 * **و`message_key` مفتاح لا نص**: المحرّك لا يكتب نصوصا للمستخدم
 * (`GROUND-RULES.md` §1.1)، **والواجهة تترجمه.**
 */
@Serializable
data class ApiErrorBody(
    val code: String = "",
    @SerialName("message_key") val messageKey: String = "",
)

@Serializable
data class User(
    val id: String = "",
    val phone: String = "",
    @SerialName("full_name") val fullName: String = "",
    val roles: List<String> = emptyList(),
    @SerialName("avatar_thumb_url") val avatarThumbUrl: String? = null,
)

@Serializable
data class Tokens(
    @SerialName("access_token") val accessToken: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
)

/**
 * نتيجة الدخول.
 *
 * **و`pin_required` تخص الأدمن وحده** (رمز رباعي ثانٍ) — **وتصل التطبيق
 * أيضا**: صاحب المنصة قد يفتح تطبيق السائق بحسابه. **فتُقرأ ولا تُهمل**،
 * وإلا رأى شاشة دخول ناجحة بلا توكن.
 */
@Serializable
data class AuthResult(
    val user: User = User(),
    val tokens: Tokens = Tokens(),
    @SerialName("pin_required") val pinRequired: Boolean = false,
    @SerialName("pin_setup") val pinSetup: Boolean = false,
    val challenge: String = "",
)

/**
 * ما تقوله المنصّة عن نفسها **قبل أن يكون هناك حساب.**
 *
 * (`GET /api/v1/public/platform` — مفتوح بلا توثيق لأن شاشة الدخول
 * تحتاجه وهي أوّل ما يُرى.)
 *
 * **وباب الرمز يُطفأ من الإعدادات لا من شيفرة كلّ تطبيق**
 * (`auth.otp_login`): **فتطبيق يعرض تبويب الرمز والمنصّة أطفأته** يرسل
 * صاحبه إلى باب مغلق.
 */
@Serializable
data class Platform(
    /**
     * **اسمُ المنصّة وشعارُها** — لترويسة الأوراق المطبوعة.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «قالبُ الطباعة المستخدم بالويب لازم
     *  يُطبَّق بالتطبيق، مشان الكشف يكون رسميّاً للمنصّة».)
     *
     * **ولا يُكتب الاسمُ في الشيفرة** (قاعدةُ المالك: «لا أريد أن تكتب
     * اسم المنصة بأيّ مكانٍ أبدا») — **يبدّله من لوحته فتتبدّل الورقة.**
     */
    val name: String = "",
    val logo: String = "",
    @SerialName("otp_login") val otpLogin: Boolean = true,
    /**
     * **أيُطلب رمزٌ عند إنشاء الحساب؟** (`auth.signup_verify`).
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٥.)
     *
     * **وافتراضُه `false` لا `true`** — **وتطبيقٌ قديمٌ لا يعرف الحقلَ
     * يجب أن يتصرّف كالجديد**: يمرّ بلا رمز. **ولو كان الافتراضُ
     * `true` لحبس نفسَه على شاشةٍ لا يصلها رمز.**
     */
    @SerialName("signup_verify") val signupVerify: Boolean = false,
    @SerialName("password_min_length") val passwordMinLength: Int = 8,
    @SerialName("support_phone") val supportPhone: String = "",
    /**
     * **عنوانُ المكتب وموقعُه وحساباتُه** — لشاشة «تواصل معنا».
     *
     * **والمحرّكُ يرسلها منذ اليوم الأوّل** (`public/platform`) — وكانت
     * تُهمَل هنا. **ولا يُكتب رقمُ الدعم في الشيفرة** (قاعدةُ المالك):
     * يبدّله من لوحته فتتبدّل الشاشة.
     */
    val address: String = "",
    /** **إحداثيّاتُ المكتب** — نصّاً كما تُحفظ (`lat,lng`). */
    val location: String = "",
    val social: Social = Social(),
)

/** **حساباتُ المنصّة** — وفارغُها لا يُعرض: أيقونةٌ لا تفتح شيئاً عطب. */
@Serializable
data class Social(
    val facebook: String = "",
    val instagram: String = "",
    val telegram: String = "",
    val whatsapp: String = "",
)

/**
 * **تذكرةُ واتساب** — انظر `AuthApi.waTicket`.
 *
 * `waUrl` رابطٌ يفتح واتساب على رقم المنصّة برسالةٍ جاهزة، **و`text`
 * نسختُه النصّيّة** لمن أراد نسخَها بيده حين لا يُفتح الرابط.
 */
@kotlinx.serialization.Serializable
data class WaTicket(
    val tag: String = "",
    val text: String = "",
    @kotlinx.serialization.SerialName("wa_url") val waUrl: String = "",
)
