package com.rahalgo.merchant.store

import com.rahalgo.merchant.noStoreMsg
import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.Store
import com.rahalgo.shared.merchant.StoreReport
import com.rahalgo.shared.merchant.StoreSettingsInput
import com.rahalgo.ui.err
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متجري — حالُه وساعاتُه وأرقامُه وما عليه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣.)
 *
 * # وأربعةٌ في شاشةٍ واحدةٍ لا أربعُ شاشات
 *
 * **كلُّها تُقرأ مرّةً في اليوم** — الحالُ صباحاً، والساعاتُ مرّةً في
 * الشهر، والأرقامُ مساءً. **وأربعُ تبويباتٍ لأربعة أشياءَ نادرةٍ تجعل
 * التطبيقَ يبدو أكبرَ ممّا يفعل.**
 *
 * # والإنذاراتُ خرجت منها
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «الإنذارات احذفها من هنا وستكون بقسمٍ
 *  مستقلٍّ بالقائمة الجانبيّة».)
 *
 * **وما عليه ليس من حالِ متجره** — انظر `drawer/DrawerScreens.kt`.
 */
class StoreViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var store by mutableStateOf<Store?>(null)
        private set

    var hours by mutableStateOf<List<DayHours>>(emptyList())

    /**
     * ══════════════════════════════════════════════════════════════════
     * **أقسامُ السوق — كلُّها وما اختاره منها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «مو معقول كلُّ الأقسام تطلع عند كلّ
     *  المتاجر… مطعمٌ شو علاقتُه بالأحذية؟»)
     *
     * **و`allSections` للاختيار، و`mySections` لنموذج الصنف.**
     */
    var allSections by mutableStateOf<List<PlatformSectionRef>>(emptyList())
    var mySections by mutableStateOf<List<PlatformSectionRef>>(emptyList())
        private set

    var report by mutableStateOf<StoreReport?>(null)
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    var saving by mutableStateOf(false)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **ونقطةٌ اختيرت ولم تُحفظ بعد**
    // ══════════════════════════════════════════════════════════════════
    //
    // **والخريطةُ تُغلق فيعود إلى الشاشة** — **ولو كُتبت في `store`
    // مباشرةً لَبدت محفوظةً وهي لم تصل المحرّك**، فيخرج ويظنّ دبّوسَه
    // انتقل.
    var pickedLat by mutableStateOf<Double?>(null)
        private set

    var pickedLng by mutableStateOf<Double?>(null)
        private set

    /** **يستقبل ما اختاره على الخريطة** — ولا يحفظ: الحفظُ بزرّه. */
    fun setPoint(lat: Double, lng: Double) {
        pickedLat = lat
        pickedLng = lng
    }

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            runCatching {
                val mine = api.stores().stores.firstOrNull()
                if (mine == null) {
                    error = noStoreMsg()
                    loading = false
                    return@launch
                }
                store = mine
                // ══════════════════════════════════════════════════════
                // **وكلٌّ يُجلب على حدة، وإخفاقُ واحدٍ لا يُسقط الشاشة**
                // ══════════════════════════════════════════════════════
                //
                // **التقاريرُ والإنذاراتُ زينةٌ حول الحال** — ومن سقطت
                // شاشتُه كلُّها لأنّ نداءَ تقريرٍ تعثّر **لم يستطع أن
                // يفتح متجرَه صباحاً.**
                runCatching { hours = api.hours(mine.id) }
                // **والأقسامُ زينةٌ حول الحال مثلُها** — ومن سقطت
                // شاشتُه لأنّ نداءَ أقسامٍ تعثّر لم يفتح متجرَه.
                runCatching { allSections = api.platformSections().sections }
                runCatching { mySections = api.storeSections(mine.id).sections }
                // **واليومُ وحدَه** — **ومدًى من سبعة أيّامٍ يُقرأ «اليوم»
                // فيظنّ صاحبُ المتجر يومَه أكبرَ ممّا هو.**
                runCatching {
                    val today = java.time.LocalDate.now().toString()
                    report = api.reports(mine.id, from = today, to = today)
                }
                error = ""
            }.onFailure { error = err(it) }
            loading = false
        }
    }

    /**
     * **يفتح المتجرَ أو يغلقه** — وهو أهمُّ مفتاحٍ في يده.
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «المتجرُ لا يستقبل طلبات — إذا كان
     *  مفتوحاً يستقبل وإذا كان مغلقاً لا يستقبل».)
     *
     * **وهو الحقُّ في المحرّك**: `emergency_closed` جزءٌ من معادلة
     * «مفتوح» نفسِها (`orders/hours.go`) — **فليس إيقافَ استقبالٍ في
     * متجرٍ مفتوح، بل إغلاقُ المتجر.**
     *
     * **وأشرفُ من رفض كلّ طلبٍ يصله**: الرفضُ يُحسب مخالفةً، **والإغلاقُ
     * إعلانٌ صادقٌ لا عقوبةَ فيه.**
     */
    fun setOpen(on: Boolean) {
        val id = store?.id ?: return
        saving = true
        viewModelScope.launch {
            // **والبابُ `/emergency` لا `/settings`** — انظر `MerchantApi`:
            // **كنتُ أرسل حقلاً لا يقرؤه المحرّكُ فيُطرح صامتاً**، فيقلب
            // المفتاحُ نفسَه في الشاشة والمتجرُ مفتوحٌ كما كان.
            runCatching { api.setClosed(id, closed = !on) }
                .onSuccess { ack -> store = store?.copy(emergencyClosed = ack.closed) }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    fun setPrepMinutes(minutes: Int) {
        val id = store?.id ?: return
        if (minutes <= 0) return
        saving = true
        viewModelScope.launch {
            runCatching { api.settings(id, StoreSettingsInput(prepMinutes = minutes)) }
                // **والردُّ `{"updated": true}` لا المتجر** — فيُبدَّل
                // الحالُ محلّيّاً، **ونداءُ قراءةٍ كاملٍ لرقمٍ واحدٍ حملٌ
                // بلا سبب.**
                .onSuccess { store = store?.copy(prepMinutes = minutes) }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    /**
     * **يبدّل اسمَ متجره** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣).
     *
     * **وكان للأدمن وحدَه** — **فمن أخطأ حرفاً يومَ سُجّل بقي الخطأُ في
     * كلّ إشعارٍ يصل سائقَه.**
     */
    fun rename(name: String) {
        val id = store?.id ?: return
        val trimmed = name.trim()
        if (trimmed.isEmpty()) {
            Flash.fail(com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.v_store_name))
            return
        }
        saving = true
        viewModelScope.launch {
            runCatching { api.settings(id, StoreSettingsInput(name = trimmed)) }
                .onSuccess { store = store?.copy(name = trimmed) }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **عنوانُه ودبّوسُه — بيده هو**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-٢٣، وقرارُه قبله ٢٠٢٦-٠٨-١٢: «يجب إضافة
     *  عنوان المتجر بالإعدادات».)
     *
     * **وكانا يُضبطان من لوحة الإدارة وحدَها** — فمن انتقل أو وجد
     * دبّوسَه في الشارع المجاور **ينتظر من يفتح له اللوحة**، والسائقُ
     * يقف على بابٍ ليس بابَه.
     *
     * **والنقطةُ تُرسَل كاملةً أو لا تُرسَل** — المحرّكُ يردّ نصفَها
     * خطأً (`merchant_ops_handlers.go`): **من أرسل `lat` وحدَه نقل
     * متجرَه إلى خطّ الاستواء.**
     */
    fun saveAddress(text: String, lat: Double?, lng: Double?) {
        val id = store?.id ?: return
        val trimmed = text.trim()
        saving = true
        viewModelScope.launch {
            runCatching {
                api.settings(
                    id,
                    StoreSettingsInput(
                        addressText = trimmed.ifEmpty { null },
                        lat = lat,
                        lng = lng,
                    ),
                )
            }
                .onSuccess {
                    store = store?.copy(
                        addressText = trimmed.ifEmpty { store?.addressText ?: "" },
                        lat = lat ?: store?.lat,
                        lng = lng ?: store?.lng,
                    )
                    // **وتُمحى المسوّدةُ بعد الحفظ** — **ونقطةٌ تبقى
                    // معلّقةً تُرسَل ثانيةً مع أيّ حفظٍ تالٍ.**
                    pickedLat = null
                    pickedLng = null
                    Flash.ok(com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.ok_address_saved))
                }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    /**
     * **يحفظ أقسامَ المتجر** — انظر `mySections`.
     *
     * **ولا يُحفظ فارغاً بلا قصد**: من أزال الكلَّ عاد يرى القائمةَ
     * كاملةً كما قبل الاختيار، **وذاك سلوكٌ مقصودٌ لا عطب.**
     */
    fun saveSections(ids: List<String>) {
        val id = store?.id ?: return
        saving = true
        viewModelScope.launch {
            runCatching { api.setStoreSections(id, ids) }
                .onSuccess {
                    mySections = allSections.filter { it.id in ids }
                    Flash.ok(com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.ok_sections_saved))
                }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    fun saveHours(days: List<DayHours>) {
        val id = store?.id ?: return
        saving = true
        viewModelScope.launch {
            runCatching { api.setHours(id, days) }
                .onSuccess { hours = days }
                .onFailure { Flash.fail(err(it)) }
            saving = false
        }
    }

    /**
     * **الطارئ — إغلاقٌ فوريٌّ يُبلَّغ به.**
     *
     * **وهو الإغلاقُ نفسُه** (`/emergency`) — **والمحرّكُ يُشعر الإدارةَ
     * به**، فلا نداءَ ثانٍ ولا بابَ آخر.
     */
    fun emergency() {
        val id = store?.id ?: return
        viewModelScope.launch {
            runCatching { api.setClosed(id, closed = true) }
                .onSuccess { ack ->
                    store = store?.copy(emergencyClosed = ack.closed)
                    Flash.ok(com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.merchant.R.string.ok_store_closed))
                }
                .onFailure { Flash.fail(err(it)) }
        }
    }
}
