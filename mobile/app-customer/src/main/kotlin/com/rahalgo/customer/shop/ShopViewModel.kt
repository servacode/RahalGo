package com.rahalgo.customer.shop

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Banner
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Section
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.apiError
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.drop
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

    /**
     * **لافتاتُ صفحة التسوّق** — (طلبُ المالك ٢٠٢٦-٠٨-١٨).
     *
     * **والمحرّكُ يرسلها منذ زمنٍ ولا أحدَ يرسمها** — تصل في
     * `/public/home` مع مهلتها وحالِ دورانها، **فتُنشَر من اللوحة ولا
     * تُرى في الجوّال.**
     */
    var banners by mutableStateOf<List<Banner>>(emptyList())
        private set

    /** **دورانُها ومهلتُه من اللوحة لا من الشيفرة.** */
    var bannerAuto by mutableStateOf(false)
        private set

    var bannerEveryMs by mutableStateOf(0)
        private set

    var pick by mutableStateOf<String?>(null)
        private set

    var items by mutableStateOf<List<Item>>(emptyList())
        private set

    var query by mutableStateOf("")
        private set

    var busy by mutableStateOf(false)
        private set

    /**
     * **رايةُ السحب وحدَه** — لا `busy` العامّة.
     *
     * **و`busy` تُرفع عند كلّ فتحِ قسم** — فلو قادت دوّارةَ السحب
     * **لَظهرت في أعلى الشاشة كلَّما ضُغطت رقاقةُ قسم**، وهو ما لم
     * يطلبه أحد.
     */
    var refreshing by mutableStateOf(false)
        private set

    /** **يُنعش كلَّ شيءٍ بسحبةٍ** — (طلبُ المالك ٢٠٢٦-٠٨-١٨). */
    fun refresh() {
        if (refreshing) return
        refreshing = true
        viewModelScope.launch {
            runCatching {
                val home = api.home()
                sections = home.sections
                banners = home.banners
                bannerAuto = home.bannerAuto
                bannerEveryMs = home.bannerEveryMs
                // **والقسمُ المفتوحُ يُعاد جلبُه** — **وإنعاشٌ يُحدّث
                // الشريطَ ويترك البضاعةَ قديمةً نصفُ إنعاش.**
                pick?.let { items = api.sectionItems(it).items }
                error = ""
            }.onFailure { error = apiError(getApplication(), it as Exception) }
            refreshing = false
        }
    }

    var error by mutableStateOf("")
        private set

    /** **أهذه نتيجةُ بحثٍ أم أصنافُ قسم** — والفارغُ يقول غيرَ ما يقول. */
    var searching by mutableStateOf(false)
        private set

    private var typing: Job? = null

    init {
        load()

        // ══════════════════════════════════════════════════════════════
        // **وما تكتبه لوحةُ الإدارة يظهر بلا إعادة تشغيل**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٨: «أيُّ تعديلٍ من لوحة الأدمن فوراً
        //  يُطبَّق حتّى ولو الزبونُ فاتحٌ التطبيق، ما يلزم يحدّث أو يعيد
        //  تشغيل التطبيق».)
        //
        // **وكانت هذه الشاشةُ الوحيدةَ التي لا تسمع النبضة** — تسمعها
        // الطلباتُ والحسابُ والمحفظة، **والسوقُ — وهو ما تكتبه اللوحةُ
        // فعلاً — يبقى على ما جلبه عند إقلاعه.** فيُغيَّر سعرٌ أو
        // يُخفى صنفٌ **ويطلبه الزبونُ بسعرٍ لم يعد قائما.**
        //
        // # ولا يُقطع بحثٌ جارٍ
        //
        // **`load()` تُغلق البحثَ وتعود إلى القسم** — ومن كتب كلمةً
        // فانتُزعت من تحته لأنّ موظّفاً غيّر بانراً **يقرأ التطبيقَ
        // معطوباً لا محدَّثاً.** فيُؤجَّل حتّى يخرج من البحث.
        viewModelScope.launch {
            Refresh.tick.drop(1).collect {
                // **ولا يُنعَش وهو يكتب** — `load()` تنتهي إلى
                // `openSection` فتمحو الحقل: **كلمةٌ تُنتزع من
                // تحت إصبعه لأنّ موظّفاً غيّر بانرا.**
                if (!searching && query.isEmpty()) load()
            }
        }
    }

    fun load() {
        busy = true
        error = ""
        viewModelScope.launch {
            try {
                val home = api.home()
                sections = home.sections
                banners = home.banners
                bannerAuto = home.bannerAuto
                bannerEveryMs = home.bannerEveryMs
                // **وأوّلُ قسمٍ يُفتح** — إلّا أن يكون قد اختار قبل الدوران.
                val first = pick ?: home.sections.firstOrNull()?.id
                if (first != null) openSection(first) else busy = false
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
                busy = false
            }
        }
    }

    /**
     * **يفتح قسماً بأمرِ صاحبه** — ضغطةً على رقاقةٍ في الشريط.
     *
     * **ويمحو ما كُتب في البحث** — قصداً: من اختار قسماً ترك بحثَه.
     *
     * **ولا تُنادى من `type`** — انظرها: **الكتابةُ ليست اختيارَ قسم**،
     * ومحوُ الحقل فيها يمحو ما يكتبه صاحبُه تحت إصبعه.
     */
    fun openSection(id: String) {
        pick = id
        searching = false
        query = ""
        typing?.cancel()
        loadSection(id)
    }

    /**
     * **يجلب أصنافَ قسمٍ ولا يمسّ حقلَ البحث.**
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٨: «حقلُ البحث يفتح الكيبورد، أكتب ولكن لا
     *  يظهر أيُّ حرفٍ بالكتابة».)
     *
     * **وكان `type` ينادي `openSection` وهي تمحو `query`** — فيُكتب
     * الحرفُ الأوّلُ ثمّ يُمحى في النداء نفسِه. **وحرفٌ واحدٌ أقلُّ من
     * الحدّ الأدنى دائماً، فلا يُبلَغ الحرفُ الثاني أبدا** — والحقلُ
     * لا يمسك شيئا.
     *
     * **ولا خطأَ ولا رسالة**: لوحةُ المفاتيح تُفتح والحرفُ يُبتلع،
     * **فيُقرأ عطباً في الجهاز لا في التطبيق.**
     */
    private fun loadSection(id: String) {
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
            // **ولا يُعاد الجلبُ إلّا للعائد من نتائج بحث** — ومن يكتب
            // حرفَه الأوّلَ لم يغادر قسمَه، **ونداءُ شبكةٍ عند كلّ ضغطةِ
            // مفتاحٍ يستنزف حزمةَ من يتصفّح في الرقّة.**
            if (searching) {
                searching = false
                pick?.let { loadSection(it) }
            }
            return
        }
        typing = viewModelScope.launch {
            delay(QUIET_MS)
            busy = true
            error = ""
            try {
                items = api.search(q).items
                searching = true
            } catch (e: CancellationException) {
                // ══════════════════════════════════════════════════════
                // **والإلغاءُ ليس خطأً — يُعاد رميُه**
                // ══════════════════════════════════════════════════════
                //
                // (شكوى المالك ٢٠٢٦-٠٨-١٨ بصورةِ شاشة: «تعذّر إتمام
                //  الطلب (gs1)» تحت حقل البحث.)
                //
                // **و`gs1` اسمُ `CancellationException` بعد تشويش R8.**
                //
                // # ولماذا وقع
                //
                // **كلُّ حرفٍ يُكتب يُلغي بحثَ سابقِه** (`typing?.cancel()`)
                // — وهو مقصود. **فإن كان السابقُ قد تجاوز المهلةَ ودخل
                // في النداء، خرج بـ`CancellationException`.**
                //
                // **و`catch (e: Exception)` تبتلعها** — فتُعرض على أنّها
                // عطبُ خادم. **فيُقرأ التطبيقُ مكسوراً وهو يعمل كما
                // صُمّم**، ويرى المالكُ رمزاً لا معنى له.
                //
                // # ولماذا تُعاد لا تُهمَل
                //
                // **الإلغاءُ يسري في شجرة المهامّ بالاستثناء نفسِه** —
                // ومن ابتلعه قطع السريان: **يُلغى النطاقُ ويبقى ما فيه
                // يعمل** حتّى يُتلف النموذج.
                throw e
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
