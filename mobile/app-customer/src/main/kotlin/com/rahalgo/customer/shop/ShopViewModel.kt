package com.rahalgo.customer.shop

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Section
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.apiError
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ التسوّق — القسمُ المختارُ وأصنافُه والبحث**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (وترتيبُ الشاشة كترتيب الويب حرفاً: **بحثٌ ← شريطُ أقسامٍ ← أصنافُ
 *  المختار.**)
 *
 * # ولماذا أوّلُ قسمٍ يُفتح وحدَه
 *
 * **شاشةٌ تُفتح فارغةً تنتظر ضغطة** تُقرأ عطباً — **ومن فتح السوقَ جاء
 * ليرى بضاعةً لا ليختار قسما.**
 *
 * # والبحثُ ينتظر أن يسكت
 *
 * **نداءٌ عند كلّ حرفٍ يعني عشرةَ نداءاتٍ لكلمةٍ من عشرة أحرف** — تسعةٌ
 * منها تُرمى، **وحزمةُ من يتصفّح في الرقّة تُستنزف بلا فائدة.**
 *
 * **وثلاثُ مئةِ ملّي كما في الويب حرفا** — ورقمان يفترقان يجعلان الشاشةَ
 * الواحدةَ تُحسّ مختلفةً في جهازين.
 */
class ShopViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var sections by mutableStateOf<List<Section>>(emptyList())
        private set

    var pick by mutableStateOf<String?>(null)
        private set

    var items by mutableStateOf<List<Item>>(emptyList())
        private set

    var query by mutableStateOf("")
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    /** **أهذه نتيجةُ بحثٍ أم أصنافُ قسم** — والفارغُ يقول غيرَ ما يقول. */
    var searching by mutableStateOf(false)
        private set

    private var typing: Job? = null

    init {
        load()
    }

    fun load() {
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                val home = api.home()
                sections = home.sections
                // **وأوّلُ قسمٍ يُفتح** — إلّا أن يكون قد اختار قبل الدوران.
                val first = pick ?: home.sections.firstOrNull()?.id
                if (first != null) openSection(first) else busy = false
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
                busy = false
            }
        }
    }

    fun openSection(id: String) {
        pick = id
        searching = false
        query = ""
        typing?.cancel()
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                items = api.sectionItems(id).items
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    /**
     * **يكتب فيُبحث بعد أن يسكت.**
     *
     * **وحرفان حدُّه الأدنى** (كما في الويب) — **وحرفٌ واحدٌ يجلب السوقَ
     * كلَّه** فيبطئ ولا يفيد. **وما دونهما يعود إلى القسم المفتوح.**
     */
    fun type(text: String) {
        query = text
        typing?.cancel()
        val q = text.trim()
        if (q.length < MIN_CHARS) {
            searching = false
            pick?.let { openSection(it) }
            return
        }
        typing = viewModelScope.launch {
            delay(QUIET_MS)
            busy = true
            error = ""
            try {
                items = api.search(q).items
                searching = true
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    private companion object {
        const val MIN_CHARS = 2
        const val QUIET_MS = 300L
    }
}
