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
