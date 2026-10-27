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

    // ══════════════════════════════════════════════════════════════════
    // **والمعلَّقُ غيرُ المرفوض** (٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان المرفوضُ يُعدّ في «ينتظرون البتّ» ويبقى فيه للأبد** — فبعد
    // شهرين يرى المندوبُ عشرين «بانتظار» وهم مرفوضون من زمن. **ورقمُ
    // اللوحة (`pending_leads`) يعدّ `new` وحدَها**، فيتناقض الرقمان.
    //
    // **فالمعلَّقُ ما ينتظر الإدارةَ أو تنتظره هي** (`new` و`needs_info`)،
    // **والمرفوضُ في قسمه.**

    /** **ما ينتظر البتّ** — جديدٌ أو تطلب الإدارةُ فيه معلومات. */
    val pendingLeads: List<Lead> get() = leads.filter { it.status != "rejected" }

    /** **ما رُفض** — بسببه، في قسمٍ مطويٍّ تحت المعلَّقين. */
    val rejectedLeads: List<Lead> get() = leads.filter { it.status == "rejected" }

    /** **أتُعرض المرفوضة؟** — مطويّةٌ افتراضاً فلا تزحم القائمة. */
    var showRejected by mutableStateOf(false)

    // **البحثُ والفلترة** — لمن له عشراتُ العملاء (٢٠٢٦-١٠-٢٧).
    var query by mutableStateOf("")
    var filter by mutableStateOf(ClientFilter.All)

    /** **العملاءُ بعد البحث والفلترة** — والترتيبُ ترتيبُ المحرّك. */
    fun visible(list: List<RepMerchant>): List<RepMerchant> {
        // **بالاسم أو الصنف أو رقم صاحب المتجر** — والأرقامُ العربيّةُ ومفتاحُ سوريا يُفهمان.
        val q = normalizeQuery(query)
        val qd = digitsOnly(query)
        return list.filter { m ->
            (q.isEmpty() || normalizeQuery(m.name).contains(q) || normalizeQuery(m.categoryName).contains(q) ||
                (qd.isNotEmpty() && digitsOnly(m.ownerPhone.orEmpty()).contains(qd))) &&
                when (filter) {
                    ClientFilter.All -> true
                    ClientFilter.Missing -> m.goalState == "pending"
                    ClientFilter.NoItems -> m.itemsCount == 0
                    ClientFilter.Stopped -> m.status != "active"
                }
        }
    }

    /** **أتُجلب صفحةٌ تالية؟** — فلا يُضغط «المزيد» مرّتين. */
    var loadingMore by mutableStateOf(false)
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

    /**
     * **شعارُ متجر عميله** (طلبُ المالك ٢٠٢٦-١٠-١٩: «والمندوب يقدر كمان يضيف لوغو للمتجر»)
     * — يُرفع ثمّ يُربط ثمّ يُعاد جلبُ التفصيل فيظهر.
     */
    fun setLogo(bytes: ByteArray) {
        val id = openId
        if (id.isEmpty()) return
        busy = true
        viewModelScope.launch {
            try {
                val ref = api.uploadLogo("logo.jpg", bytes)
                api.setLogo(id, ref.id)
                detail = api.merchant(id)
                com.rahalgo.ui.Flash.ok(getApplication<android.app.Application>().getString(com.rahalgo.ui.R.string.ok_logo_saved))
            } catch (e: Exception) {
                com.rahalgo.ui.Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }

    /**
     * **يُعيد جلبَ التفصيل والتفصيلُ ظاهر** — سحبٌ للتحديث.
     *
     * **ولا يُفرغ ما يُرى قبل أن يصل الجديد** — وإلّا رأى دوّارةً مكانَ
     * شاشته ثمّ رجعت.
     */
    fun refreshDetail() {
        val id = openId
        if (id.isEmpty()) return
        busy = true
        viewModelScope.launch {
            try {
                detail = api.merchant(id)
                error = ""
            } catch (e: Exception) {
                com.rahalgo.ui.Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **وطلباتُ العميل كلُّها لا أوّلُ عشرين** (٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **المحرّكُ يقسمها صفحاتٍ ويرسل `total`** — والتطبيقُ كان يجلب
    // الأولى وحدَها، **فالمربّعاتُ فوق تقول ١٥٠ والقائمةُ عشرون.**

    /** **أبقيت طلباتٌ لم تُجلب؟** */
    val hasMoreOrders: Boolean
        get() = detail?.let { it.orders.size < it.total } ?: false

    /** **يجلب الصفحةَ التالية ويضمّها** — والملخّصُ يبقى من الأولى. */
    fun loadMoreOrders() {
        val d = detail ?: return
        val id = openId
        if (id.isEmpty() || loadingMore || !hasMoreOrders) return
        loadingMore = true
        viewModelScope.launch {
            try {
                val next = api.merchantPage(id, d.page + 1)
                // **ولا يُضمّ ما جاء لعميلٍ أُغلق في الأثناء.**
                if (openId == id) {
                    // **ورقمُ الطلب مفتاحُه** — طلبٌ جديدٌ يُزيح الصفحاتِ سطراً فيتكرّر.
                    val seen = d.orders.map { it.number }.toSet()
                    detail = d.copy(
                        orders = d.orders + next.orders.filter { it.number !in seen },
                        page = next.page,
                        total = next.total,
                    )
                }
            } catch (e: Exception) {
                com.rahalgo.ui.Flash.fail(apiError(getApplication(), e))
            }
            loadingMore = false
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
            // **و«بحاجة معلومات» أوّلاً** — هي ما ينتظر يدَه هو (٢٠٢٦-١٠-٢٧).
            runCatching {
                leads = api.leads().filter { it.status != "converted" }
                    .sortedByDescending { it.status == "needs_info" }
            }
            busy = false
        }
    }
}

/** **فلاترُ قائمة العملاء** — الكلُّ · ناقصُه شروط · بلا أصناف · متوقّف. */
enum class ClientFilter { All, Missing, NoItems, Stopped }

/** **الأرقامُ العربيّةُ والفارسيّةُ لاتينيّة**، والحروفُ صغيرة. */
private fun normalizeQuery(s: String): String = buildString {
    for (c in s.trim().lowercase()) {
        append(
            when (c) {
                in '\u0660'..'\u0669' -> '0' + (c - '\u0660')
                in '\u06F0'..'\u06F9' -> '0' + (c - '\u06F0')
                else -> c
            },
        )
    }
}

/** **الرقمُ بلا رموز** — ويُقصّ «00» و«963» و«0» الأوّل. */
private fun digitsOnly(s: String): String {
    var d = normalizeQuery(s).filter { it.isDigit() }
    if (d.startsWith("00")) d = d.drop(2)
    if (d.startsWith("963") && d.length > 9) d = d.drop(3)
    if (d.startsWith("0")) d = d.drop(1)
    return d
}
