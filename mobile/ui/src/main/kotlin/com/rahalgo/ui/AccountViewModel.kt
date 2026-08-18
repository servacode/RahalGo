package com.rahalgo.ui

import android.app.Application
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.AddressInput
import com.rahalgo.shared.model.MeSummary
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.ByteArrayOutputStream
import java.io.IOException
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ شاشة الحساب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ ببنائها بالتطبيق كما هي بالويب، ولكن
 *  حذف حسابي اجعلها بالآخر بعد العناوين».)
 *
 * # ورسالةٌ واحدةٌ لا رسالةٌ لكلّ قسم
 *
 * **الشاشةُ أقسامٌ خمسة**، ولو حمل كلُّ قسمٍ رسالتَه لَبقيت رسالةُ قسمٍ
 * معلّقةً وصاحبُها يعمل في آخر. **فرسالةٌ واحدةٌ تُمحى مع كلّ فعلٍ
 * جديد** — يقرؤها حيث نظر.
 */
data class AccountState(
    val me: MeSummary? = null,
    val addresses: List<Address> = emptyList(),
    val loading: Boolean = true,
    val busy: Boolean = false,
    val error: String = "",
    /** **مرحلةُ الحذف**: لم يُطلب · وصل الرمزُ وينتظر · تمّ. */
    val deleteAsked: Boolean = false,
    val deleted: Boolean = false,
    /**
     * **الرقمُ الذي أُرسل إليه رمزُ تبديل الهاتف** — وفارغٌ يعني لم يُطلب.
     *
     * **ويُحفظ لأنّ التأكيد يحتاجه**: المحرّكُ يطلب الرقمَ مع الرمز
     * ليتحقّق أنّهما لطلبٍ واحد — **ولو قُرئ من الحقل لَبدّله صاحبُه
     * بعد الإرسال فأكّد رقماً لم يصله رمز.**
     */
    val phonePending: String = "",
    /** **ورقمُ توثيق واتساب المنتظِر** — والفارغُ لم يُطلب. */
    val waPending: String = "",
)

class AccountViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(AccountState())
        private set

    private val backend = AppCore.get()

    init {
        refresh()
        // **وتسمع النبضةَ كسائر الشاشات** — من بدّل اسمَه من الويب
        // **يراه هنا بلا أن يخرج ويعود.**
        viewModelScope.launch {
            Refresh.tick.drop(1).collect { refresh() }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            state = try {
                // **والعناوينُ مع الملخّص في نداءين متتاليين** — وسقوطُ
                // العناوين لا يُفرغ الشاشة: **قائمةٌ فارغةٌ أهونُ من
                // شاشةٍ لا تُفتح.**
                val me = backend.account.summary()
                val list = runCatching { backend.account.addresses() }.getOrDefault(emptyList())
                state.copy(me = me, addresses = list, loading = false, error = "")
            } catch (e: Exception) {
                state.copy(loading = false, error = describe(e))
            }
        }
    }

    /** **يجري فعلاً ثمّ يُنعش** — والرسالةُ تُمحى قبل أن يبدأ. */
    private fun act(okMsg: Int, block: suspend () -> Unit) {
        if (state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                block()
                val me = backend.account.summary()
                val list = runCatching { backend.account.addresses() }.getOrDefault(state.addresses)
                // ══════════════════════════════════════════════════════
                // **وما بدّله يُبَثّ ليراه في شريطه فورا**
                // ══════════════════════════════════════════════════════
                //
                // (شكوى المالك ٢٠٢٦-٠٨-١٣: «الصورة ما زالت حرف خ».)
                //
                // **والخادمُ لا يبثّ خبراً عن فعلٍ فعله صاحبُ الحساب
                // بنفسه** — فيبثّه من فعله. **وبدونه يبقى الشريطُ
                // العلويُّ على الاسم القديم والصورة القديمة** حتّى
                // يُقلع التطبيقُ من جديد.
                Refresh.bump()
                // **والرسالةُ تُبَثّ ولا تُخزَّن** — انظر `Flash`.
                Flash.ok(getApplication<Application>().getString(okMsg))
                state.copy(me = me, addresses = list, busy = false)
            } catch (e: Exception) {
                Flash.fail(describe(e))
                state.copy(busy = false)
            }
        }
    }

    fun setName(name: String) {
        val clean = name.trim()
        if (clean.isEmpty()) return
        act(R.string.acc_saved_name) { backend.account.setName(clean) }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يرفع صورةً — مضغوطةً قبل أن تخرج من الهاتف**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٣: «حاولتُ رفع صورةٍ ورفض، يقول خطأٌ
     *  بالرفع».)
     *
     * **وحدُّ المحرّك خمسةُ ميغا** (`media.max_upload_mb`)، **وصورةُ
     * كاميرا الهاتف بين أربعةٍ وتسعة** — فأكثرُها يُردّ.
     *
     * **ورفعُ الحدِّ ليس جوابا**: الصورةُ تُصغَّر في الخادم إلى ألفٍ
     * وستّمئة بكسل على أيّ حال، **فما زاد يُرسَل ليُرمى** — يستهلك حزمةَ
     * السائق ويُبطئ الرفعَ على شبكةٍ ضعيفة.
     *
     * **والضغطُ هنا لا هناك**: ما لا يُرسَل لا يُنتظَر.
     *
     * **وألفٌ وستّمئة على الضلع الأطول** — حدُّ الخادم نفسُه، فلا يخسر
     * شيئاً ولا يحمل زائدا.
     */
    /**
     * **يرفع صورةً قُرئت وصُغّرت في `ImagePick`.**
     *
     * **والبايتاتُ لا `Uri`** — (طلبُ المالك ٢٠٢٦-٠٨-١٨: الكاميرا
     * المباشرة). **وصورةُ الكاميرا تجيء من ملفٍّ مؤقّتٍ وصورةُ المعرض
     * من مزوّد المعرض** — **ونموذجٌ يفرّق بينهما يعرف عن النظام ما لا
     * يخصّه.** فيقرؤهما المنتقي ويردّ شيئاً واحدا.
     */
    fun setAvatarBytes(bytes: ByteArray) {
        act(R.string.acc_saved_photo) {
            backend.account.setAvatar("avatar.jpg", bytes)
        }
    }

    fun removeAvatar() =
        act(R.string.acc_removed_photo) { backend.account.removeAvatar() }

    fun setPassword(current: String, next: String) =
        act(R.string.acc_pw_changed) { backend.account.setPassword(current, next) }

    fun addAddress(label: String, text: String, lat: Double, lng: Double) =
        act(R.string.acc_saved_address) {
            backend.account.addAddress(AddressInput(label.trim(), text.trim(), lat, lng))
        }

    fun deleteAddress(id: String) =
        act(R.string.acc_deleted_address) { backend.account.deleteAddress(id) }

    fun makeDefault(id: String) =
        act(R.string.acc_default_address) { backend.account.makeDefaultAddress(id) }

    /** **يطلب رمزَ الحذف** — ولا يحذف. */
    fun askDelete() {
        if (state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.deleteRequest()
                state.copy(busy = false, deleteAsked = true)
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    /**
     * **يؤكّد الحذف** — ولا رجعةَ بعده.
     *
     * **ولا يُنعَش بعده**: الحسابُ ذهب، **ونداءُ ملخّصٍ يُردّ بأربعمئةٍ
     * وواحد** فتظهر رسالةُ خطأٍ لفعلٍ نجح.
     */
    fun confirmDelete(code: String) {
        if (state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.deleteConfirm(code.trim())
                state.copy(busy = false, deleted = true)
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **تبديلُ الرقم — وتوثيقُه على واتساب**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «الويب لا ينفع السائق مثل التطبيق…
    //  التطبيق يجب أن يكون تطبيقاً كاملاً، لا حاجةَ للمستخدم من الدخول
    //  إلى مكانٍ ثانٍ لإجراء تعديلات».)

    fun askPhone(phone: String) {
        val clean = phone.trim()
        if (clean.isEmpty() || state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.phoneChangeRequest(clean)
                state.copy(busy = false, phonePending = clean)
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    /**
     * **يؤكّد الرقمَ الجديد** — ثمّ يُنعش.
     *
     * **وتوثيقُ واتساب يسقط معه** (`SetPhone` في المحرّك) — **والشاشةُ
     * تقرؤه من الملخّص بعده** فيرى صاحبُه الحقيقةَ ولا يظنّ نفسَه
     * موثَّقا.
     */
    fun confirmPhone(code: String) {
        val phone = state.phonePending
        if (phone.isEmpty() || state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.phoneChangeConfirm(phone, code.trim())
                val me = backend.account.summary()
                Flash.ok(getApplication<Application>().getString(R.string.acc_phone_saved))
                state.copy(me = me, busy = false, phonePending = "")
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    fun cancelPhone() {
        state = state.copy(phonePending = "")
    }

    // ══════════════════════════════════════════════════════════════════
    // **وتوثيقُ واتساب صار شرطاً على السائق**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم بالطبع السائق يجب أن يوثّق حسابَه
    //  على واتساب… لا يمكنه استقبال الطلبات بدون توثيق حسابه، اعتبره
    //  شرطاً للسائق».)
    //
    // **وقد سألتُ عن دوره أمسِ فلم يكن له دور** — فقرّره المالكُ دورا.
    // **والمنعُ في المحرّك لا في الشاشة**: من لم يوثّق **لا يفتح دوامه**
    // (`drivers.require_whatsapp`)، فلا يُعرض عليه طلبٌ أصلا.
    //
    // **وعلى رقم الحساب نفسِه لا على رقمٍ ثانٍ** (قرارُ المالك
    // ٢٠٢٦-٠٨-١٢: «ما يصير رقم الهاتف مختلف عن واتساب»).

    fun askWhatsApp() {
        val phone = state.me?.phone.orEmpty()
        if (phone.isEmpty() || state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.whatsappRequest(phone)
                state.copy(busy = false, waPending = phone)
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    fun confirmWhatsApp(code: String) {
        val phone = state.waPending
        if (phone.isEmpty() || state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                backend.account.whatsappConfirm(phone, code.trim())
                val me = backend.account.summary()
                state.copy(
                    me = me,
                    busy = false,
                    waPending = "",
                ).also {
                    Flash.ok(getApplication<Application>().getString(R.string.acc_wa_done))
                }
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    fun cancelWhatsApp() {
        state = state.copy(waPending = "")
    }

    fun cancelDelete() {
        state = state.copy(deleteAsked = false)
    }

    /** **ولا مفتاحُ آلةٍ يُعرض** — إلّا ما لا ترجمةَ له، فيُعرض ليُعرف. */
    /** **الرمزُ بعربيّة** — من الخريطة المركزيّة (`data/ApiErrors.kt`). */
    /**
     * ══════════════════════════════════════════════════════════════════
     * **فحصُ الإشعارات — يقول لماذا لا يرنّ**
     * ══════════════════════════════════════════════════════════════════
     *
     * (وقع ٢٠٢٦-٠٨-١٤ على جهازٍ حقيقيّ: **لم يرنّ.** والجهازُ مسجَّلٌ
     *  والمفتاحُ مضبوطٌ والمشروعُ متطابق — **ولا شيءَ في المنصّة يقول
     *  أين وقفت الرسالة**، فبقي التخمينُ وحدَه.)
     *
     * **وإشعارٌ لا يصل لا يشتكي منه أحد**: السائقُ يظنّ أنّه لا طلبات،
     * **والمكتبُ يظنّه كسولا.**
     *
     * **والجوابُ يُقرأ بترتيب**: أمُهيَّأٌ الخادم؟ ثمّ أثمّة جهازٌ
     * مسجَّل؟ ثمّ أقبِلت الرسالة؟ — **وأوّلُ «لا» هو السبب.**
     */
    fun checkPush() {
        if (state.busy) return
        state = state.copy(busy = true)
        viewModelScope.launch {
            state = try {
                val r = backend.devices.test()
                // **ويُكتب في السجلّ أيضاً** — الشاشةُ تُقرأ بعين، **والسجلُّ
                // يُقرأ من حاسوبٍ موصولٍ حين لا يكون صاحبُ الجهاز حاضرا.**
                Log.i(
                    "RahalGo/فحص",
                    "مهيّأ=" + r.configured + " أجهزة=" + r.devices +
                        " حديثة=" + r.fresh + " قُبل=" + r.sent +
                        " مرفوض=" + r.dead + " خطأ=" + r.error,
                )
                val app = getApplication<Application>()
                when {
                    !r.configured -> Flash.fail(app.getString(R.string.push_check_off)).let { state.copy(busy = false) }
                    r.devices == 0 -> Flash.fail(app.getString(R.string.push_check_none)).let { state.copy(busy = false) }
                    r.fresh == 0 -> Flash.fail(app.getString(R.string.push_check_stale)).let { state.copy(busy = false) }
                    r.error.isNotEmpty() -> Flash.fail(app.getString(R.string.push_check_failed, r.error)).let { state.copy(busy = false) }
                    else -> Flash.ok(app.getString(R.string.push_check_ok, r.sent, r.fresh)).let { state.copy(busy = false) }
                }
            } catch (e: Exception) {
                Flash.fail(describe(e)).let { state.copy(busy = false) }
            }
        }
    }

    private fun describe(e: Exception): String = apiError(getApplication(), e)

    private companion object {
        /** **الضلعُ الأطول** — حدُّ الخادم نفسُه، فلا يُرسَل ما يُرمى. */
        const val MAX_EDGE = 1600

        /** **وجودةُ الضغط** — خمسٌ وثمانون لا تُرى بالعين وتنصّف الحجم. */
        const val JPEG_QUALITY = 85
    }
}