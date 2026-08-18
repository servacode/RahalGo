package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يخصّ صاحبَ الحساب — كما يرسله المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بالطبع تفتح صفحةً كاملة… كيف سيدير حسابَه
 *  إذا يغيّر كلمة سرّه، يبدّل صورتَه، إذا أراد حذف حسابه، وإذا أراد
 *  إضافة عنوان».)
 *
 * **والنداءُ نفسُه الذي تقرؤه شاشةُ الويب** (`GET /api/v1/me/summary`)
 * — **ولا حقلَ يُحسب في التطبيق** فيفترق عمّا تعرضه اللوحة.
 */
@Serializable
data class MeSummary(
    @SerialName("full_name") val fullName: String = "",
    @SerialName("avatar_thumb_url") val avatarThumbUrl: String? = null,
    val phone: String = "",
    @SerialName("whatsapp_phone") val whatsappPhone: String? = null,
    @SerialName("whatsapp_verified") val whatsappVerified: Boolean = false,
    @SerialName("has_password") val hasPassword: Boolean = true,
)

/** **جوابُ رفع الصورة** — المسارُ الجديد ليُعرَض بلا إعادة تحميل. */
@Serializable
data class AvatarResult(
    @SerialName("avatar_thumb_url") val avatarThumbUrl: String = "",
)

/**
 * **عنوانُ الشخص** — (`GET /api/v1/my/addresses`).
 *
 * **والإحداثيّ مع النصّ**: النصُّ لمن يقرأ، **والإحداثيُّ لمن يوصّل** —
 * وعنوانٌ بلا نقطةٍ على الخريطة يُبحث عنه بالهاتف.
 */
@Serializable
data class Address(
    val id: String = "",
    /** **المنطقةُ والمبنى** — الإلزاميُّ الوحيد. */
    @SerialName("area_building") val areaBuilding: String = "",
    val street: String = "",
    val floor: String = "",
    /** **نوعُ العنوان** — `home` أو `work` أو `other`. */
    val kind: String = "other",
    /**
     * **السطرُ المركَّب** — يقرؤه السائقُ ويُحفظ لقطةً في الطلب.
     *
     * **مشتقٌّ في المحرّك لا يُرسَل** — **ولو أُرسل لصار للعنوان مصدران
     * يفترقان.**
     */
    @SerialName("address_text") val text: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("is_default") val isDefault: Boolean = false,
)

// **ولا غلافَ للقائمة** — المحرّكُ يردّ مصفوفةً عاريةً (`[]`)،
// **وغلافٌ يُفترض يجعل القراءةَ تسقط بلا سببٍ يُقرأ.**

/** **ما يُرسَل لإنشاء عنوان** — بأسماء حقول المحرّك. */
@Serializable
data class AddressInput(
    @SerialName("area_building") val areaBuilding: String,
    val street: String = "",
    val floor: String = "",
    val kind: String = "other",
    /** **والنقطةُ اختياريّةٌ في التعديل** — تصحيحُ حرفٍ لا يفتح خريطة. */
    val lat: Double? = null,
    val lng: Double? = null,
)
