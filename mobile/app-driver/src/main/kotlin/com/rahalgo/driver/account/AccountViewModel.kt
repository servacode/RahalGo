package com.rahalgo.driver.account

import android.app.Application
import android.net.Uri
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.AddressInput
import com.rahalgo.shared.model.MeSummary
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
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
    val done: String = "",
    /** **مرحلةُ الحذف**: لم يُطلب · وصل الرمزُ وينتظر · تمّ. */
    val deleteAsked: Boolean = false,
    val deleted: Boolean = false,
)

class AccountViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(AccountState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        refresh()
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
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            state = try {
                block()
                val me = backend.account.summary()
                val list = runCatching { backend.account.addresses() }.getOrDefault(state.addresses)
                state.copy(
                    me = me,
                    addresses = list,
                    busy = false,
                    done = getApplication<Application>().getString(okMsg),
                )
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    fun setName(name: String) {
        val clean = name.trim()
        if (clean.isEmpty()) return
        act(R.string.acc_saved) { backend.account.setName(clean) }
    }

    /**
     * **يرفع صورةً من معرض الجهاز.**
     *
     * **وتُقرأ بايتاتُها هنا لا في الشاشة**: قراءةُ ملفٍّ عملُ قرصٍ،
     * **وفي خيط الرسم تُجمّد الواجهةَ** بقدر حجم الصورة.
     */
    fun setAvatar(uri: Uri) {
        act(R.string.acc_saved) {
            val bytes = getApplication<Application>().contentResolver
                .openInputStream(uri)?.use { it.readBytes() }
                ?: throw IOException("لا يُقرأ الملفّ")
            backend.account.setAvatar("avatar.jpg", bytes)
        }
    }

    fun removeAvatar() = act(R.string.acc_saved) { backend.account.removeAvatar() }

    fun setPassword(current: String, next: String) =
        act(R.string.acc_pw_changed) { backend.account.setPassword(current, next) }

    fun addAddress(label: String, text: String, lat: Double, lng: Double) =
        act(R.string.acc_saved) {
            backend.account.addAddress(AddressInput(label.trim(), text.trim(), lat, lng))
        }

    fun deleteAddress(id: String) = act(R.string.acc_saved) { backend.account.deleteAddress(id) }

    fun makeDefault(id: String) = act(R.string.acc_saved) { backend.account.makeDefaultAddress(id) }

    /** **يطلب رمزَ الحذف** — ولا يحذف. */
    fun askDelete() {
        if (state.busy) return
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            state = try {
                backend.account.deleteRequest()
                state.copy(busy = false, deleteAsked = true)
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
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
        state = state.copy(busy = true, error = "", done = "")
        viewModelScope.launch {
            state = try {
                backend.account.deleteConfirm(code.trim())
                state.copy(busy = false, deleted = true)
            } catch (e: Exception) {
                state.copy(busy = false, error = describe(e))
            }
        }
    }

    fun cancelDelete() {
        state = state.copy(deleteAsked = false, error = "")
    }

    /** **ولا مفتاحُ آلةٍ يُعرض** — إلّا ما لا ترجمةَ له، فيُعرض ليُعرف. */
    private fun describe(e: Exception): String {
        Log.e("RahalGo/حساب", "فشل نداء الحساب", e)
        val app = getApplication<Application>()
        return when {
            e is ApiClient.ApiException -> when (e.body.code) {
                "unauthorized", "invalid_refresh" -> app.getString(R.string.err_invalid_refresh)
                "wrong_password" -> app.getString(R.string.acc_wrong_password)
                "weak_password" -> app.getString(R.string.acc_weak_password)
                "otp_invalid" -> app.getString(R.string.acc_bad_code)
                "has_active_orders" -> app.getString(R.string.err_has_active_orders)
                "" -> app.getString(R.string.err_internal)
                else -> e.body.code
            }
            e is HttpRequestTimeoutException || e is IOException ->
                app.getString(R.string.err_network)
            else -> app.getString(R.string.err_internal)
        }
    }
}
