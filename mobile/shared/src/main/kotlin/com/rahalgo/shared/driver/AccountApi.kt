package com.rahalgo.shared.driver

import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.AddressInput
import com.rahalgo.shared.model.AvatarResult
import com.rahalgo.shared.model.MeSummary
import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إدارةُ الحساب من التطبيق — لا من متصفّح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ومن قال لك إنّه سيفتحها وهو يقود؟ كيف
 *  سيدير حسابَه إذا يغيّر كلمة سرّه، يبدّل صورتَه، إذا أراد حذف حسابه،
 *  وإذا أراد إضافة عنوان».)
 *
 * # ولا نداءَ جديدٌ في المحرّك
 *
 * **كلُّها المسارات التي تناديها شاشةُ الويب حرفيّا** — `me/summary`
 * و`me/name` و`me/avatar` و`auth/password` و`my/addresses`
 * و`auth/account/delete`.
 *
 * **ومسارٌ ثانٍ للتطبيق كان سيفترق**: تُشدَّد قاعدةٌ في الويب وتُنسى في
 * الهاتف، **فيحذف السائقُ حسابَه من طريقٍ بلا رمزِ تأكيد.**
 *
 * # والحذفُ بخطوتين لا بضغطة
 *
 * **`request` ثمّ `confirm` برمزٍ يصل الرقم** — وهو ما تفعله الشاشة
 * الأخرى. **وفعلٌ لا يُستدرَك لا يُترك لإصبعٍ زلّ.**
 */
class AccountApi(private val api: ApiClient) {

    suspend fun summary(): MeSummary = api.call("/api/v1/me/summary")

    /** **يبدّل الاسم** — `PATCH` لأنّه حقلٌ من الحساب لا الحسابُ كلُّه. */
    suspend fun setName(name: String) {
        api.call<Ack>("/api/v1/me/name", HttpMethod.Patch, mapOf("full_name" to name))
    }

    /**
     * **يرفع الصورة** — بايتاتٍ لا نصّاً.
     *
     * **وصورةٌ في JSON تكبر الثلث** بترميز ستّةٍ وستّين، **وتُقرأ
     * كلُّها في الذاكرة مرّتين** — وهاتفُ السائق ليس خادما.
     */
    suspend fun setAvatar(fileName: String, bytes: ByteArray): AvatarResult {
        api.upload("/api/v1/me/avatar", fileName, bytes)
        // **والمسارُ يُقرأ من الملخّص بعده** — الرفعُ لا يردّ جسماً
        // مقروءاً في هذا العميل، **وقراءةٌ ثانيةٌ أصدقُ من تخمين.**
        return AvatarResult(summary().avatarThumbUrl.orEmpty())
    }

    suspend fun removeAvatar() {
        api.call<Ack>("/api/v1/me/avatar", HttpMethod.Delete, null)
    }

    /**
     * **يبدّل كلمة المرور** — بالقديمة معها.
     *
     * **وهاتفٌ يُترك مفتوحاً لحظةً** يكفي لمن أراد أن يقفل الحسابَ على
     * صاحبه. **والقديمةُ هي ما يعرفه هو وحدَه.**
     */
    suspend fun setPassword(current: String, next: String) {
        api.call<Ack>(
            "/api/v1/auth/password",
            HttpMethod.Post,
            // **وأسماءُ الحقول كما يقرؤها المحرّك** — `password`
            // للجديدة لا `new_password`. **واسمٌ يُخمَّن يُردّ بأربعمئة**
            // بلا كلمةٍ تقول أيَّ حقلٍ نقص.
            mapOf("password" to next, "current_password" to current),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **وتبديلُ الرقم بخطوتين — والتوثيقُ معه لا بعده بيوم**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «تغيير رقم الهاتف يجب أن يكون موجوداً
    //  أيضاً كما طلبتُ منك… الويب لا ينفع السائق مثل التطبيق».)
    //
    // **والرمزُ يصل الرقمَ الجديد لا القديم** — وهو ما يُثبت أنّه له:
    // من كتب رقمَ غيره لا يصله شيء.
    //
    // **وتبديلُ الرقم يُسقط توثيقَ واتساب** (أُصلح ٢٠٢٦-٠٨-١٣) — **فلو
    // أُعطي التبديلُ بلا التوثيق لَفقد صاحبُه بابَ استعادة حسابه ولا
    // سبيلَ لإعادته من التطبيق.** فهما معاً أو لا واحدَ منهما.

    suspend fun phoneChangeRequest(phone: String) {
        api.call<Ack>(
            "/api/v1/auth/phone/request",
            HttpMethod.Post,
            mapOf("phone" to phone),
        )
    }

    suspend fun phoneChangeConfirm(phone: String, code: String) {
        api.call<Ack>(
            "/api/v1/auth/phone/confirm",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code),
        )
    }

    suspend fun whatsappRequest(phone: String) {
        api.call<Ack>(
            "/api/v1/auth/whatsapp/request",
            HttpMethod.Post,
            mapOf("phone" to phone),
        )
    }

    suspend fun whatsappConfirm(phone: String, code: String) {
        api.call<Ack>(
            "/api/v1/auth/whatsapp/confirm",
            HttpMethod.Post,
            mapOf("phone" to phone, "code" to code),
        )
    }

    suspend fun addresses(): List<Address> = api.call("/api/v1/my/addresses")

    /**
     * **يضيف عنواناً** — بنوعٍ لا بخريطةٍ مختلطة.
     *
     * **`Map<String, Any>` لا يُسلسَل**: المُسلسِلُ يحتاج نوعاً معروفاً
     * لكلّ قيمة، **وخريطةٌ فيها نصٌّ وعددٌ تُسقط النداءَ وقتَ التشغيل**
     * لا وقتَ البناء — **فلا يُمسك إلّا على جهاز.**
     */
    suspend fun addAddress(input: AddressInput) {
        api.call<Ack>("/api/v1/my/addresses", HttpMethod.Post, input)
    }

    /**
     * **يعدّل عنواناً محفوظاً** — (قرارُ المالك ٢٠٢٦-٠٨-١٨).
     *
     * **ومن أخطأ في طابقه كان يحذف وينشئ** — فيفقد كونَه الافتراضيَّ
     * ويعيد التقاطَ نقطته على الخريطة.
     */
    suspend fun updateAddress(id: String, input: AddressInput) {
        api.call<Ack>("/api/v1/my/addresses/" + id, HttpMethod.Patch, input)
    }

    suspend fun deleteAddress(id: String) {
        api.call<Ack>("/api/v1/my/addresses/" + id, HttpMethod.Delete, null)
    }

    suspend fun makeDefaultAddress(id: String) {
        api.call<Ack>(
            "/api/v1/my/addresses/" + id + "/default",
            HttpMethod.Post,
            mapOf<String, String>(),
        )
    }

    /** **يطلب رمزَ حذف الحساب** — يصل رقمَه. */
    suspend fun deleteRequest() {
        api.call<Ack>("/api/v1/auth/account/delete/request", HttpMethod.Post, mapOf<String, String>())
    }

    /** **يؤكّد الحذف بالرمز** — ولا رجعةَ بعده. */
    suspend fun deleteConfirm(code: String) {
        api.call<Ack>(
            "/api/v1/auth/account/delete/confirm",
            HttpMethod.Post,
            mapOf("code" to code),
        )
    }
}
