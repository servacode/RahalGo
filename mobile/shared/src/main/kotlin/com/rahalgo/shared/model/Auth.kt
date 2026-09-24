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
    /**
     * ══════════════════════════════════════════════════════════════════
     * **تفصيلٌ يضبطه المالكُ لا يُترجمه التطبيق** (٢٠٢٦-٠٩-١٣)
     * ══════════════════════════════════════════════════════════════════
     *
     * **والرمزُ عقدٌ يترجمه التطبيقُ**، **والتفصيلُ حالٌ يضبطها المالكُ
     * من اللوحة** — مثل `launch.notice`: «نفتح استقبالَ الطلبات بعد
     * العيد».
     *
     * **ونصٌّ مكتوبٌ في التطبيق لا يُصحَّح إلّا بنشرٍ في المتجر** —
     * **ويومَ يُفتح البابُ يبقى معروضاً أسبوعاً.**
     *
     * **وغيابُه يقع على نصّ الرمز** — فلا يُعرض فراغ.
     */
    val details: Map<String, String> = emptyMap(),
)

@Serializable
data class User(
    val id: String = "",
    val phone: String = "",
    @SerialName("full_name") val fullName: String = "",
    val roles: List<String> = emptyList(),
    @SerialName("avatar_thumb_url") val avatarThumbUrl: String? = null,
    // **بيانٌ يعرفه ثالثٌ — فلا وصولَ قبل تبديله** (`CUST-DEF-010`).
    // **المحرّكُ يسمح بـ`/auth/me` لمن يبدّل** فيصل هذا الحقلُ `true`،
    // **وما عداه من الأبواب يُردّ 403 `password_change_required`.**
    @SerialName("must_change_password") val mustChangePassword: Boolean = false,
)

@Serializable
data class Tokens(
    @SerialName("access_token") val accessToken: String = "",
    /**
     * ══════════════════════════════════════════════════════════════════
     * **ومتى ينتهي رمزُ الوصول — يقوله مُصدِرُه** (`D19`)
     * ══════════════════════════════════════════════════════════════════
     *
     * **كان المحرّكُ يرسله والتطبيقُ يرميه** (`access_expires_at` في
     * `identity/models.go`) — **فلم يكن للجهاز طريقٌ يعرف به أنّ رمزَه
     * انتهى إلّا أن يُردّ.**
     *
     * **وحلقةُ إعادة الوصل لا تُردّ ردّاً تقرؤه**: **مقبسٌ فشلت
     * مصافحتُه يرمي استثناءً نصُّه يختلف باختلاف المحرّك** — قِيس
     * (دورةُ ٥٨): **٤٠١ ترمي `WebSocketException` و٤٠٣ و٥٠٣ ترميان
     * `ProtocolException`** — **ورقمُ الحال في نصّ الرسالة لا في
     * حقل.** **فبناءُ القرار على قراءة نصٍّ هشٌّ**، **وهذا الحقلُ
     * سلطةٌ لا تخمين.**
     *
     * **ونصُّه `RFC3339`** — يُقرأ بـ`accessExpiresAtMs()`.
     */
    @SerialName("access_expires_at") val accessExpiresAt: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
) {
    /**
     * **متى ينتهي بالأجزاء من الألف** — **وصفرٌ يعني «لا يُعرف».**
     *
     * **وما لا يُقرأ لا يُخمَّن**: **صفرٌ يمنع التجديدَ الاستباقيّ
     * ولا يمنع الوصل** — **فالجهلُ يُبقي السلوكَ القديمَ ولا يكسره.**
     */
    fun accessExpiresAtMs(): Long =
        runCatching {
            java.time.OffsetDateTime.parse(accessExpiresAt).toInstant().toEpochMilli()
        }.getOrDefault(0L)
}

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
    /**
     * **شعارُ المنصّة — وقد لا يكون** (`B9-SC`، ٢٠٢٦-٠٩-١٦).
     *
     * **والمحرّكُ يردّ `null` صراحةً حين لا صورةَ مرفوعة**
     * (`settingMedia` ⇒ `*string`) — **وهو عقدٌ صحيحٌ لا عطب.**
     *
     * **وكان نوعُه هنا `String` غيرَ قابلٍ للعدم** — **فسقط فكُّ
     * الرسالة كلِّها**، **ومعها `otp_login` فاختفى بابُ الدخول
     * بالرمز.**
     *
     * **ويُقال في النوع لا يُداوى بتساهلٍ عامّ في المُفكِّك** —
     * **والتساهلُ العامُّ يُخفي عيباً حقيقيّاً في عقدٍ آخر.**
     */
    val logo: String? = null,
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
    /**
     * **أيُستقبَل طلبٌ جديدٌ الآن؟** (`PH`، ٢٠٢٦-٠٩-١٤).
     *
     * **ولا تُحسَب في الجهاز** — **والخادمُ يحكم**: جدولُ دوامٍ بمنطقةِ
     * دمشق وإيقافٌ مؤقّتٌ يضبطه المالك. **وساعةُ الهاتف تُبدَّل بإصبع.**
     *
     * **وافتراضُها «مفتوحٌ»** — **وتطبيقٌ يقرأ ردّاً لا حقلَ فيه يجب أن
     * يتصرّف كما كان يتصرّف**: **والمحرّكُ يردّ الطلبَ عند الإنشاء
     * على كلّ حال**، **فالأسوأُ أن يُمنَع من الطلب بلا سببٍ حقيقيّ.**
     */
    val ordering: Ordering = Ordering(),
    /**
     * **أيُّ أبواب الزبون مفتوحةٌ أصلاً؟** (`PL`، ٢٠٢٦-٠٩-١٦).
     *
     * **وكان التطبيقُ لا يعرف أنّ المنصّةَ لم تُفتح حتّى يطرق باباً**
     * **فيُردّ ٥٠٣** — **فيرسم شاشةَ سوقٍ ثمّ يبدّلها رسالةَ خطأ.**
     * **وحالٌ مقصودةٌ تُقرأ خطأً تُرى عطباً.**
     *
     * **وهي عرضٌ لا حكم** — كحال الاستقبال فوقَها: **المنعُ في
     * `launch_gate` بالمحرّك**، **ومن نادى باباً مغلقاً رُدّ وإن لم
     * يقرأ هذا الحقلَ قطّ.**
     */
    val launch: LaunchState = LaunchState(),
    /**
     * **سياسةُ أجرة الطلب المخصَّص عند الإنشاء** (`Batch 2c`).
     *
     * **يقرؤها نموذجُ الطلب الخاصّ ليقول للزبون قبل الإرسال**: إن كانت
     * الأجرةُ محدَّدةً من المنصة عُرضت فوراً، وإلّا قيل «تُحدَّد بعد قبول
     * السائق». **وهي عرضٌ لا حكم** — اللقطةُ على الطلب هي التي تُلزم.
     */
    @SerialName("custom_delivery") val customDelivery: CustomDelivery = CustomDelivery(),
)

/**
 * **سياسةُ أجرة الطلب المخصَّص** — `admin_defined` تعني أنّ المنصةَ
 * حدّدت الأجرةَ فتُعرَض؛ وما عداها يحددها السائق بعد قبوله.
 */
@Serializable
data class CustomDelivery(
    val source: String = "driver_defined",
    val fee: Long = 0,
) {
    /** **أحدّدتها المنصة؟** — فيُعرَض رقمُها للزبون عند الإنشاء. */
    val adminDefined: Boolean get() = source == "admin_defined"
}

/**
 * **حالُ الافتتاح كما يقولها الخادم.**
 *
 * **وافتراضُها «مفتوحٌ»** — **وتطبيقٌ قديمٌ يقرأ ردّاً لا حقلَ فيه يجب
 * أن يتصرّف كما كان يتصرّف**، **والمحرّكُ يردّه عند الطرق على كلّ
 * حال.** **ولو كان الافتراضُ مغلقاً لحبس نفسَه على شاشةٍ لا مخرجَ
 * منها بأوّل ردٍّ ناقص.** (وهي حجّةُ `signup_verify` نفسُها.)
 */
@Serializable
data class LaunchState(
    @SerialName("customer_signup") val signup: Boolean = true,
    @SerialName("customer_browse") val browse: Boolean = true,
    @SerialName("customer_orders") val orders: Boolean = true,
    @SerialName("customer_custom_orders") val customOrders: Boolean = true,
    /** **نصُّ المالك** — **يُبدَّل من اللوحة بلا نشرٍ ولا تحديث.** */
    val notice: String = "",
) {
    /**
     * **أهذه حالُ ما قبل الافتتاح؟**
     *
     * **والتصفّحُ هو الفيصل**: **سوقٌ لا يُرى لا شاشةَ له** — **وبابُ
     * الدخول يبقى لمن له حساب.**
     */
    val preLaunch: Boolean get() = !browse
}

/**
 * **حالُ الاستقبال كما يقولها الخادم.**
 *
 * **وهي عرضٌ لا حكم**: **الشاشةُ تُعطّل الزرَّ وتقول السببَ قبل
 * الضغط** — **والمنعُ الحقيقيُّ في المحرّك عند الإنشاء.**
 */
@Serializable
data class Ordering(
    @SerialName("ordering_available") val available: Boolean = true,
    /** `platform_closed_now` أو `temporarily_unavailable` أو فارغ. */
    val reason: String = "",
    /** **نصُّ المالك** — ويغلب نصَّ التطبيق حين يضبطه. */
    val message: String = "",
    /** **موعدُ العودة** — RFC 3339، وفارغٌ يعني «لا موعدَ معلوم». */
    @SerialName("next_available_at") val nextAvailableAt: String = "",
    /** **لحظةُ الخادم** — **وبها يُقاس لا بساعة الجهاز.** */
    @SerialName("server_time") val serverTime: String = "",
    val timezone: String = "",
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
