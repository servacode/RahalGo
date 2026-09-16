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

    /** **تقرؤه النافذةُ لتجلب خياراتِ الصنف** — ونداءٌ واحدٌ لا اثنان. */
    val customerApi: CustomerApi get() = api

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
                //
                // **ويأخذ الرقمَ كغيرِه** — **فالسحبةُ أحدثُ من نداءٍ
                // سابقٍ لم يردّ بعد**: **وإلّا دهس القديمُ الجديد.**
                pick?.let {
                    // **والسحبةُ تخلُف النداءَ الجاري لا تجاوره** —
                    // **وإلّا بقي يعمل وقد أُبطلت تذكرتُه.**
                    fetching?.cancel()
                    val ticket = latest.begin()
                    val got = api.sectionItems(it).items
                    if (latest.isCurrent(ticket)) items = got
                }
                error = ""
            }.onFailure { error = apiError(getApplication(), it as Exception) }
            refreshing = false
            // ══════════════════════════════════════════════════════════
            // **ومن أبطل تذكرةَ غيرِه ورث دوّارتَه**
            // ══════════════════════════════════════════════════════════
            //
            // **والدوّارةُ تُطفأ بتذكرةٍ سارية** — **فمن سُحبت تذكرتُه
            // تركها لمن أبطلها.** **والسحبةُ تُبطل ولا تملك `busy`**:
            // **فلو سكتت هنا لَدارت الدوّارةُ إلى الأبد** — **وهو
            // عينُ ما يمنعه شاهدُ الجهاز «لا دوّارةَ دائمة».**
            //
            // **إلّا أن يكون قد بدأ نداءُ قسمٍ أحدثُ** — **فهي له.**
            if (fetching?.isActive != true) busy = false
        }
    }

    var error by mutableStateOf("")
        private set

    /** **أهذه نتيجةُ بحثٍ أم أصنافُ قسم** — والفارغُ يقول غيرَ ما يقول. */
    var searching by mutableStateOf(false)
        private set

    private var typing: Job? = null

    /**
     * **نداءُ أصنافِ القسم الجاري** — **ويُلغى قبل أن يُبدأ غيرُه.**
     *
     * # وقسمٌ يُفتح فتُعرض بضاعةُ غيرِه
     *
     * **وكان كلُّ ضغطةِ رقاقةٍ تُطلق نداءً ولا تُلغي سابقَه** — فمن
     * نقر «شاورما» ثمّ «حلويات» قبل أن يردّ الأوّل **أطلق نداءين
     * يتسابقان على `items` نفسِها.** **والرابحُ أسرعُهما لا آخرُهما.**
     *
     * **فيقف الشريطُ على «حلويات» وتحتَه شاورما** — **ويطلبها الزبونُ
     * وهو يظنّ أنّه في قسمٍ آخر.** **ولا خطأَ يُرى ولا سطرَ في سجلّ.**
     *
     * **وشبكةُ الرقّة تجعل هذا هو الحالَ الغالبَ لا النادر**: **ردٌّ
     * بطيءٌ لقسمٍ صغيرٍ يسبقه ردٌّ أبطأُ لقسمٍ كبير.**
     */
    private var fetching: Job? = null

    /**
     * **نداءُ الصفحة الجاري** — **وضغطتان على «أعد المحاولة» نداءٌ واحد.**
     *
     * **ومن ضغط خمساً على شبكةٍ منقطعةٍ أطلق خمسةَ نداءاتٍ متزامنةً**
     * — **كلُّها تكتب `error` و`busy` بلا ترتيب**، **وآخرُها انتهاءً
     * يحكم الشاشةَ لا آخرُها ضغطاً.**
     */
    private var loading: Job? = null

    /** **حكمُ التسابق** — انظر [Latest]: **آخرُ من طُلب هو من يكتب.** */
    private val latest = Latest()

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

    /**
     * **الأقسامُ التي لها محتوىً يخصّ هذا الزبون** — **وهي بنيةُ
     * السوق كما يراها.**
     *
     * **وقسمٌ خاوٍ يُعرَض ثمّ يُفتح فارغاً يُقرأ عطباً** — **ومن
     * فتح تسعةَ أقسامٍ كلُّها خاليةٌ ظنّ التطبيقَ معطوباً.**
     *
     * **ولا يُبنى على الدوام**: **متجرٌ نائمٌ يبقى قسمُه** (قرارُ
     * المالك ٢٠٢٦-٠٩-١٦) — **والخادمُ يفصل العدّين.**
     */
    val visibleSections: List<Section> get() = sections.filter { it.count > 0 }

    /** **أسوقٌ لم تمتلئ بعد؟** — **لا قسمَ فيه محتوىً.** */
    val marketEmpty: Boolean get() = sections.isNotEmpty() && visibleSections.isEmpty()

    fun load() {
        // **ونداءٌ واحدٌ في الطريق** — **وإعادةُ المحاولة تُعيد المحاولةَ
        // لا تُكوّم محاولات.** **والردُّ على شبكةٍ منقطعةٍ هو الردُّ عينُه
        // مهما تكرّر**، **فخمسُ ضغطاتٍ تستنزف الحزمةَ ولا تغيّر حرفا.**
        if (loading?.isActive == true) return
        busy = true
        error = ""
        loading = viewModelScope.launch {
            try {
                val home = api.home()
                sections = home.sections
                banners = home.banners
                bannerAuto = home.bannerAuto
                bannerEveryMs = home.bannerEveryMs
                // **وأوّلُ قسمٍ يُفتح** — إلّا أن يكون قد اختار قبل الدوران.
                // **ويُفتح أوّلُ قسمٍ له محتوىً** — **لا أوّلُ قسمٍ في القائمة**،
            // **فقد يكون خاوياً فيُستقبَل الزبونُ بفراغ.**
            val first = pick ?: home.sections.firstOrNull { it.count > 0 }?.id
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
        // **ويُلغى نداءُ القسم السابق** — انظر `fetching`: **نداءان
        // يتسابقان يضعان بضاعةَ قسمٍ تحت عنوانِ قسمٍ آخر.**
        fetching?.cancel()
        // **ولا يُوثَق بالإلغاء وحدَه** — **نداءٌ تمّ ولم يُستأنَف بعدُ
        // يكتب وهو ملغى.** **والرقمُ يفصل**: **آخرُ من طُلب هو وحدَه
        // من يكتب**، **ولو نُقرت الرقاقةُ نفسُها مرّتين.**
        val ticket = latest.begin()
        busy = true
        error = ""
        fetching = viewModelScope.launch {
            try {
                val got = api.sectionItems(id).items
                if (!latest.isCurrent(ticket)) return@launch
                items = got
            } catch (e: CancellationException) {
                // **والإلغاءُ ليس خطأً** — **وهو هنا فعلُنا نحن**:
                // **رسالةُ عطبٍ على ضغطةِ قسمٍ ثانيةٍ تُقرأ تطبيقاً
                // مكسوراً.** (وهي حكايةُ `gs1` نفسُها في `type`.)
                throw e
            } catch (e: Exception) {
                if (!latest.isCurrent(ticket)) return@launch
                error = apiError(getApplication(), e)
            }
            // **ولا تُطفأ الدوّارةُ إلّا للأخير** — **ومن سبقه نداءٌ
            // أحدثُ تركها له**: **فلا تنطفئ والبضاعةُ في الطريق.**
            if (latest.isCurrent(ticket)) busy = false
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
            // **والبحثُ يكتب `items` كما يكتبها القسم** — **فيدخل
            // الرقمَ نفسَه.** **وإلّا جاء ردُّ قسمٍ بطيءٍ بعد نتيجةِ
            // بحثٍ فمحاها**: **يكتب الزبونُ كلمةً فيرى بضاعةً لا
            // تمتّ إليها.**
            val ticket = latest.begin()
            busy = true
            error = ""
            try {
                val got = api.search(q).items
                if (!latest.isCurrent(ticket)) return@launch
                items = got
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
                if (!latest.isCurrent(ticket)) return@launch
                error = apiError(getApplication(), e)
            }
            if (latest.isCurrent(ticket)) busy = false
        }
    }

    private companion object {
        const val MIN_CHARS = 2
        const val QUIET_MS = 300L
    }
}
