package com.rahalgo.rep.clients

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.rep.Lead
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.shared.rep.RepMerchantDetail
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.apiError
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عملاءُ المندوب — المقبولون والمعلَّقون معا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا في شاشةٍ واحدة
 *
 * **من سجّل عميلاً يريد أن يعرف أين وصل** — **ولو كان المعلَّقون في
 * شاشةٍ أخرى لَظنّ أنّ عملَه ضاع** حين لا يجده في «عملائي».
 *
 * **والمعلَّقون فوق**: هم ما ينتظر عملاً — **والمقبولون يعملون وحدَهم.**
 *
 * # ونداءان لا واحد
 *
 * **والفشلُ في أحدهما لا يُخفي الآخر** — **ومن سقط نداءُ المعلَّقين
 * فرأى قائمةً فارغةً ظنّ أنّ الإدارة رفضتهم كلَّهم.**
 */
class ClientsViewModel(app: Application) : AndroidViewModel(app) {

    private val api = RepApi(AppCore.get().api)

    var merchants by mutableStateOf<List<RepMerchant>?>(null)
        private set

    var leads by mutableStateOf<List<Lead>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    /**
     * **العميلُ المفتوح** — وفارغٌ يعني القائمة معروضة.
     *
     * **ولا رايةٌ ثانيةٌ للفتح**: **حالٌ واحدةٌ لا تتناقض مع نفسها.**
     */
    var detail by mutableStateOf<RepMerchantDetail?>(null)
        private set

    var openId by mutableStateOf("")
        private set

    /** **يفتح تفصيلَ عميل** — ويُجلب عند الفتح لا مع القائمة. */
    fun openDetail(id: String) {
        if (id.isEmpty()) return
        openId = id
        detail = null
        busy = true
        viewModelScope.launch {
            try {
                detail = api.merchant(id)
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    fun closeDetail() {
        openId = ""
        detail = null
    }

    init {
        load()
        // **وما يتبدّل يُقرأ** — الإدارةُ تقبل عميلاً فيظهر عنده.
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        busy = true
        viewModelScope.launch {
            try {
                merchants = api.merchants()
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            // ══════════════════════════════════════════════════════════
            // **والمقبولُ يخرج من المعلَّقين — اسمُ حاله `converted`**
            // ══════════════════════════════════════════════════════════
            //
            // (شكوى المالك ٢٠٢٦-٠٨-١٨: «عميلٌ بانتظار موافقة الإدارة مع
            //  العلم بالأسفل مكتوبٌ يعمل، والإدارةُ فعليّاً وافقت».)
            //
            // **كان المرشِّحُ يصفّي `approved` — وهي كلمةٌ لا يكتبها
            // المحرّكُ في الطلبات أبداً.** حالاتُه ثلاثٌ: `new`
            // و`converted` و`rejected`، **والموافقةُ تكتب `converted`.**
            //
            // **فكان يصفّي قيمةً لا وجودَ لها — أي لا يصفّي شيئا**:
            // فيبقى المتجرُ المقبولُ في «ينتظرون البتّ» **ويظهر مرّةً
            // ثانيةً تحت في «يعملون».** والشيءُ الواحدُ في مكانين بحالين
            // متناقضين يُفقد الثقةَ بالشاشة كلِّها.
            //
            // **والكلمةُ جاءت من مفردات المحفظة** (`approved` هناك حالُ
            // حوالةِ دفع) — **ولفظٌ يُستعار من سياقٍ لا يخصّه لا يُسقطه
            // المترجِمُ لأنّه نصٌّ صالحٌ نحوا.**
            //
            // **والمعلَّقون على حدة** — سقوطُهم لا يُفرغ المقبولين.
            runCatching { leads = api.leads().filter { it.status != "converted" } }
            busy = false
        }
    }
}
