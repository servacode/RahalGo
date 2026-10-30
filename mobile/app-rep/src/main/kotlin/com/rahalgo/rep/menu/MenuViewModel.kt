package com.rahalgo.rep.menu

import com.rahalgo.ui.Refresh
import com.rahalgo.ui.menu.ItemDraft
import com.rahalgo.ui.menu.modifiersForSave
import com.rahalgo.ui.menu.optionsError
import com.rahalgo.ui.menu.withUsdText
import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.rep.ItemInput
import com.rahalgo.shared.rep.MenuItem
import com.rahalgo.shared.rep.MenuSection
import com.rahalgo.shared.rep.ModifierGroup
import com.rahalgo.shared.rep.ModifierOption
import com.rahalgo.shared.rep.PlatformSection
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import com.rahalgo.rep.R
import com.rahalgo.ui.apiError
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أصنافُ عميلٍ يبنيها المندوبُ نيابةً عنه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يقوم المندوبُ بإضافة أصناف المنتجات
 *  الموجودة لدى المتجر بدلاً عنه… ويستطيع تعديلَ السعر وجعلَ المنتج
 *  متاحاً أو غيرَ متاح».)
 *
 * # ولماذا في التطبيق لا في اللوحة وحدَها
 *
 * **المندوبُ في السوق لا في مكتب** — يقف عند صاحب المتجر فيصوّر الصنفَ
 * بهاتفه ويكتب سعرَه. **ومن طُلب منه أن يعود إلى حاسوبٍ ليُدخلها لا
 * يُدخلها.**
 *
 * # ولا أقسامَ تُنشأ
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الأدمنُ هو من يزرع الأقسام».)
 *
 * **القسمُ في القائمة هو قسمُ السوق** — يظهر لأنّ فيه صنفاً ويختفي حين
 * يخرج آخرُ صنف. **وأوّلُ ما بنيتُ هذه الشاشةَ وضعتُ فيها «إضافة قسم»
 * فكان المندوبُ يضيف قسماً ولا يظهر** — يكتبه المحرّكُ في جدولٍ لا
 * تقرؤه `GetMenu`. **وزرٌّ يعمل بلا أثرٍ أسوأُ من زرٍّ لا وجودَ له.**
 *
 * # وإعادةُ الجلب بعد كلّ كتابة
 *
 * **ولا تُعدَّل النسخةُ في الذاكرة** — **سعرُ البيع تحسبه المنصّةُ من
 * سعر المتجر والهامش، ولا يعرفه الجهاز.** ومن كتب الرقمَ بيده في
 * القائمة أرى المندوبَ سعراً ليس هو ما سيراه الزبون.
 */
class MenuViewModel(app: Application) : AndroidViewModel(app) {

    private val api = RepApi(AppCore.get().api)

    /** **معرّفُ العميل** — يُضبَط عند الفتح، وفارغُه يعني «لا شاشة». */
    var merchantID by mutableStateOf("")
        private set

    var merchantName by mutableStateOf("")
        private set

    var sections by mutableStateOf<List<MenuSection>?>(null)

        private set

    // ══════════════════════════════════════════════════════════════════
    // **والأصنافُ تُعرض بأقسام السوق — كما في تطبيق المتجر**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «الأصناف لازم أوّل شي تُعرض بشكل أقسام
    //  وندخل بالقسم مثل المتجر وليس بشكل مختلف».)
    //
    // **وكانت تُعرض كتلاً متتابعةً** — قسمُ المتجر الداخليّ ثمّ أصنافُه،
    // ثمّ الذي يليه. **والمندوبُ يحرّر قائمةَ متجرٍ كما يحرّرها صاحبُه**،
    // فشاشتان مختلفتان لعملٍ واحدٍ تجعلان من يعرف إحداهما يتعثّر بالأخرى.
    //
    // **والتقسيمُ بقسم السوق لا بقسم المتجر**: هو ما يراه الزبونُ،
    // **وهو ما يجب أن يراه من يملأ القائمة.**
    var groups by mutableStateOf<List<ItemGroup>>(emptyList())
        private set

    /** **القسمُ المفتوح** — وفارغٌ يعني الشبكة. */
    var open by mutableStateOf<ItemGroup?>(null)
        private set

    /** **البحثُ ومرشّحُ «غير المتوفرة»** — كشبكة المتجر (٢٠٢٦-١٠-٢٩). */
    var query by mutableStateOf("")
    var onlyOff by mutableStateOf(false)
        private set

    fun openGroup(g: ItemGroup) {
        open = g
        onlyOff = false
    }

    fun showUnavailable() {
        open = null
        onlyOff = true
    }

    fun back() {
        open = null
        onlyOff = false
    }

    /**
     * **يجمع الأصنافَ بأقسام السوق.**
     *
     * **ويُعاد ضبطُ المفتوح بعد كلّ تحميل** — وإلّا بقي المندوبُ داخلَ
     * قسمٍ يحمل أصنافاً قديمة.
     */
    private fun regroup() {
        val items = sections.orEmpty().flatMap { it.items }
        // ══════════════════════════════════════════════════════════════
        // **ووجهُ القسم من القسم نفسِه — لا من أوّل صنفٍ فيه**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «صورة القسم تأتي من صور الأقسام حسب
        //  سوق المنصّة… وما تصير صورة الصنف هي صورة القسم».)
        //
        // **وكان `list.firstNotNullOfOrNull { it.imageThumbUrl }`** —
        // **فصورةُ أوّل صنفٍ تصير وجهَ القسم.** ومتجرٌ جديدٌ أصنافُه بلا
        // صورٍ **يظهر قسمُه حروفاً**، ومتجرٌ صوّر صنفاً واحداً يصير ذاك
        // الصنفُ عنوانَ القسم كلِّه.
        //
        // **وأقسامُ السوق مصوَّرةٌ كلُّها في القاعدة**، والمحرّكُ يردّ
        // صورتَها منذ ٢٠٢٦-٠٨-٢٣ — **وكان العقدُ في التطبيق يُسقطها.**
        //
        // **ولا احتياطَ بصورة صنف**: **قسمٌ بلا صورةٍ يُرسَم بحرفه**،
        // وهو صادقٌ — **وصورةُ صنفٍ في موضع القسم كذبٌ صغيرٌ يتكرّر.**
        val faceOf = platformSections.associate { it.name to (it.imageThumbUrl ?: it.imageUrl) }
        groups = items
            .groupBy { it.platformSectionName.ifBlank { unsorted() } }
            .map { (name, list) ->
                ItemGroup(
                    name = name,
                    items = list.sortedBy { it.name },
                    imageUrl = faceOf[name],
                    // **ومعرّفُ القسم من أصنافه** (٢٠٢٦-١٠-٢٧) — فيُمرَّر إلى
                    // «إضافة صنف» من داخله **ولا يُبحث عنه بالاسم**؛ و«بلا قسم» فارغ.
                    sectionId = list.firstNotNullOfOrNull { it.platformSectionId?.takeIf(String::isNotBlank) }.orEmpty(),
                )
            }
            .sortedByDescending { it.items.size }
        open = open?.let { cur -> groups.firstOrNull { it.name == cur.name } }
    }

    private fun unsorted(): String = com.rahalgo.ui.AppCore.get().app
        .getString(com.rahalgo.rep.R.string.mn_unsorted)

    /** **أقسامُ السوق** — **وبلاها لا يظهر الصنفُ في التصفّح.** */
    var platformSections by mutableStateOf<List<PlatformSection>>(emptyList())

    /** **ما اختاره المتجرُ من أقسام السوق** — ومنها وحدَها يختار المندوبُ قسمَ الصنف. */
    var storeSections by mutableStateOf<List<PlatformSection>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    /**
     * **ما يُحرَّر الآن** — وفارغٌ يعني القائمةَ معروضة.
     *
     * **وحالٌ واحدةٌ لا رايتان**: راية «مفتوح» ونسخةُ المُحرَّر تفترقان
     * فتُفتح شاشةٌ فارغة.
     */
    var editing by mutableStateOf<ItemDraft?>(null)
        private set

    /** **سببُ رفض الحفظ — فوق زرّه** كما عند المتجر (تدقيقُ ٢٠٢٦-١٠-٢٨). */
    var editError by mutableStateOf("")
        private set

    /** **المسوّدةُ كما فُتحت** — فيُعرف أفيها ما لم يُحفظ. */
    private var loadedDraft: ItemDraft? = null

    /** **في المحرّر تعديلٌ لم يُحفظ؟** — فلا يُنسخ الصنفُ المحفوظُ فوقه. */
    val dirty: Boolean get() = editing != null && editing != loadedDraft

    fun open(id: String, name: String) {
        // **وخياراتُ العميل السابق تُمحى** (تدقيقُ ٢٠٢٦-١٠-٢٨) — كانت تطلع في
        // محرّر العميل الجديد حتّى تصل قراءتُه.
        if (id != merchantID) presets.reset()
        merchantID = id
        merchantName = name
        sections = null
        storeSections = emptyList()
        open = null
        onlyOff = false
        query = ""
        editing = null
        loadedDraft = null
        editError = ""
        error = ""
        load()
    }

    // **قائمةُ المشروبات كما عند المتجر** (طلبُ المالك ٢٠٢٦-١٠-١٩).
    val drinks = com.rahalgo.ui.menu.DrinksState(
        scope = viewModelScope,
        load = { api.drinksInfo(merchantID) },
        save = { api.setDrinks(merchantID, it) },
        itemCount = { groups.sumOf { g -> g.items.size } },
        errText = { apiError(getApplication(), it) },
        upload = { bytes ->
            api.uploadItemImage("drink.jpg", bytes).let { r -> com.rahalgo.shared.merchant.MediaRef(r.id, r.thumbUrl) }
        },
    )

    // **خياراتُ الأصناف والأسعارُ دفعةً واحدة كما عند المتجر** (٢٠٢٦-١٠-٢٧).
    val presets = com.rahalgo.ui.menu.OptionPresetsState(
        scope = viewModelScope,
        load = { api.optionPresets(merchantID) },
        save = { api.setOptionPresets(merchantID, it) },
        errText = { apiError(getApplication(), it) },
    )
    val bulk = com.rahalgo.ui.menu.BulkPriceState(
        scope = viewModelScope,
        run = { api.bulkPrice(merchantID, it) },
        onApplied = { load(); Refresh.bump() },
        errText = { apiError(getApplication(), it) },
    )

    /** **العميلُ الذي قُرئت خياراتُه** — فلا تُقرأ مع كلّ نبضة. */
    private var presetsFor: String? = null

    /** **نسخُ الصنف ثمّ فتحُ النسخة** — كما عند المتجر (٢٠٢٦-١٠-٢٧). */
    fun duplicate(itemId: String) {
        if (busy || dirty) return
        busy = true
        viewModelScope.launch {
            try {
                val newId = api.duplicateItem(itemId)
                sections = api.menu(merchantID)
                regroup()
                // **أُغلق المحرّرُ والنسخُ ماشٍ؟** — فلا يُفتح من جديد بلا طلب.
                if (editing != null) {
                    sections.orEmpty().flatMap { it.items }.firstOrNull { it.id == newId }?.let { editItem(it) }
                    Flash.ok(getApplication<Application>().getString(com.rahalgo.ui.R.string.item_duplicated))
                }
                Refresh.bump()
            } catch (e: Exception) {
                if (editing != null) editError = apiError(getApplication(), e)
                else Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }

    init {
        // ══════════════════════════════════════════════════════════════
        // **والقائمةُ تتحدّث لحظيّاً كما عند المتجر** (طلبُ المالك ٢٠٢٦-١٠-١٩)
        // ══════════════════════════════════════════════════════════════
        //
        // **كانت لا تسمع النبضة** — فصاحبُ المتجر يعدّل صنفاً والمندوبُ يرى القديم
        // حتّى يخرج ويعود. **والنبضةُ تأتي من قناة الزمن الحقيقيّ** (`ShellViewModel`).
        viewModelScope.launch {
            Refresh.tick.collect { if (it > 0L && merchantID.isNotEmpty() && editing == null && !busy) load() }
        }
    }

    fun close() {
        merchantID = ""
        editing = null
        sections = null
        storeSections = emptyList()
    }

    /** **سعرُ صرف المتجر** — يُقرأ مع القائمة ويُكتب من بطاقته (٢٠٢٦-١٠-١٢). */
    var exchangeRate by mutableStateOf<Double?>(null)
        private set

    fun load() {
        if (merchantID.isEmpty()) return
        busy = true
        viewModelScope.launch {
            try {
                sections = api.menu(merchantID)
                regroup()
                error = ""
                runCatching { exchangeRate = api.merchant(merchantID).merchant.exchangeRate }
                // **مرّةً لكلّ عميل** — لا مع كلّ نبضة (تدقيقُ ٢٠٢٦-١٠-٢٨).
                if (presetsFor != merchantID || !presets.loaded) {
                    presetsFor = merchantID
                    presets.refresh()
                    drinks.refresh()
                }
            } catch (e: Exception) {
                // **والخطأُ يُكتب في حاله لا في ومضةٍ وحدَها** (٢٠٢٦-١٠-٢٧) — **وكانت
                // الدوّارةُ تبقى عشرين ثانيةً** والمندوبُ واقفٌ في المتجر ينتظر،
                // لأنّ `error` لا يُملأ. **ومع قائمةٍ ظاهرةٍ تكفي الومضة.**
                val msg = apiError(getApplication(), e)
                if (sections == null) error = msg else Flash.fail(msg)
            }
            // **وأقسامُ السوق على حدة** — **سقوطُها لا يُخفي القائمة**،
            // ويبقى المندوبُ يقرأ ما هو قائمٌ ولو تعذّر عليه أن يضيف.
            runCatching { platformSections = api.platformSections() }
            runCatching { storeSections = api.storeSections(merchantID) }
            busy = false
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الأصناف**
    // ══════════════════════════════════════════════════════════════════

    /** **صنفٌ جديد** — وقسمُ السوق يُختار في النموذج. */
    fun newItem(platformSectionId: String = "") {
        // **صنفٌ جديدٌ محاولةٌ جديدة** — فلا يحمل مفتاحَ صنفٍ سابقٍ مجهولِ
        // المصير فيُعادَ ردُّه بدل إدراجه.
        com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.REP_ITEM)
        editError = ""
        editing = ItemDraft(platformSectionId = platformSectionId)
        loadedDraft = editing
    }

    fun editItem(item: MenuItem) {
        editError = ""
        editing = ItemDraft(
            itemId = item.id,
            name = item.name,
            // **وسعرُ المتجر لا سعرُ البيع** — هو ما يملك تغييرَه.
            // **والصفرُ يُعرض صفراً** (قرارُ المالك ٢٠٢٦-١٠-٢٨: «مسموح للكل»).
            price = if (item.merchantPrice >= 0) item.merchantPrice.toString() else "",
            platformSectionId = item.platformSectionId.orEmpty(),
            modifiers = item.modifiers.map { it.withUsdText() },
            imageThumb = item.imageThumbUrl,
            // **وعملتُه كما حُفظت** (٢٠٢٦-١٠-١٢).
            currency = item.priceCurrency,
            priceUsd = com.rahalgo.ui.menu.usdText(item.merchantPriceUsd),
            storeDrinks = item.storeDrinks,
        )
        loadedDraft = editing
    }

    fun editDraft(block: (ItemDraft) -> ItemDraft) {
        editing = editing?.let(block)
        editError = ""
    }

    // ══════════════════════════════════════════════════════════════════
    // **المُعدِّلات — الحجمُ والإضافاتُ وما شابه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «مجموعاتٌ ومعدّلاتٌ غيرُ موجودةٍ
    //  بالتطبيق — لازم يكون كلُّ شيءٍ مطابقاً ١٠٠٪».)
    //
    // **والقيمُ الابتدائيّةُ كما في الويب حرفا**: مجموعةٌ جديدةٌ بأدنى
    // صفرٍ وأقصى واحد. **ورقمان مختلفان بين الشاشتين يعنيان صنفاً
    // يُبنى في الويب غيرَ الذي يُبنى في الجوّال.**

    fun addGroup() = editDraft { d ->
        d.copy(modifiers = d.modifiers + ModifierGroup(name = "", minSelect = 0, maxSelect = 1))
    }

    fun removeGroup(index: Int) = editDraft { d ->
        d.copy(modifiers = d.modifiers.filterIndexed { i, _ -> i != index })
    }

    fun updateGroup(index: Int, block: (ModifierGroup) -> ModifierGroup) = editDraft { d ->
        d.copy(modifiers = d.modifiers.mapIndexed { i, g -> if (i == index) block(g) else g })
    }

    fun addOption(groupIndex: Int) = updateGroup(groupIndex) { g ->
        g.copy(options = g.options + ModifierOption(name = "", priceDelta = 0))
    }

    fun removeOption(groupIndex: Int, optionIndex: Int) = updateGroup(groupIndex) { g ->
        g.copy(options = g.options.filterIndexed { i, _ -> i != optionIndex })
    }

    fun updateOption(
        groupIndex: Int,
        optionIndex: Int,
        block: (ModifierOption) -> ModifierOption,
    ) = updateGroup(groupIndex) { g ->
        g.copy(options = g.options.mapIndexed { i, o -> if (i == optionIndex) block(o) else o })
    }

    fun cancelEdit() {
        editing = null
        loadedDraft = null
        editError = ""
    }

    /**
     * **يقلب «متاح» من القائمة مباشرة.**
     *
     * **و«نفد الصنف» تقع عشرَ مرّاتٍ في اليوم** — **ومن فتح لها نموذجاً
     * بثمانية حقولٍ لم يقلبها**، فيبقى الصنفُ يُطلب وهو ناقص.
     */
    /**
     * **يقلب توفّرَ صنفٍ واحدٍ فوراً ثمّ يُرسل** (بلاغُ المالك ٢٠٢٦-١٠-٣٠: «يتأثّر الزرُّ الثاني») —
     * كان يُعطّل مفاتيحَ الأصناف كلِّها وينتظر الشبكةَ ثمّ يُعيد قراءةَ القائمة، فتهتزّ.
     * **كما في تطبيق المتجر**: لا يُمسّ غيرُه، **وإن ردّ المحرّكُ خطأً عاد وقيلت الرسالة.**
     */
    fun toggleAvailable(item: MenuItem) {
        val next = !item.available
        val before = sections
        sections = sections?.map { s ->
            s.copy(items = s.items.map { if (it.id == item.id) it.copy(available = next) else it })
        }
        regroup()
        viewModelScope.launch {
            runCatching { api.updateItem(item.id, ItemInput(available = next)) }
                .onSuccess { Refresh.bump() }
                .onFailure {
                    sections = before
                    regroup()
                    Flash.fail(apiError(getApplication(), it as? Exception ?: Exception(it)))
                }
        }
    }

    /**
     * **يرفع الصورةَ أوّلاً ويحتفظ بمعرّفها.**
     *
     * **ولا تُرسَل مع الصنف بايتاتٍ** — المحرّكُ يقرأ الصورةَ من نموذجٍ
     * متعدّد الأجزاء، **وصورةٌ في JSON تكبر الثلثَ وتُقرأ في الذاكرة
     * مرّتين.**
     */
    fun pickImage(bytes: ByteArray) {
        val draft = editing ?: return
        busy = true
        viewModelScope.launch {
            try {
                // **وتُعرض المصغّرةُ فورَ الرفع** — كان `imageThumb`
                // يُصفَّر، **فيُرسم الحرفُ الاحتياطيُّ بعد رفعٍ ناجح**
                // ويُظنّ أنّ الصورةَ ضاعت (بلاغُ المالك ٢٠٢٦-٠٨-٢٩).
                //
                // **وتُنسَخ من `editing` الحاليّ لا من رسمٍ قديم.**
                val ref = api.uploadItemImage("item.jpg", bytes)
                editing = editing?.copy(
                    imageMediaId = ref.id,
                    imageThumb = ref.thumbUrl.ifBlank { null },
                )
                error = ""
            } catch (e: Exception) {
                Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }

    /** **يمحو صورةَ الصنف** — والفراغُ الصريحُ يعني «أزِلها». */
    fun clearImage() {
        editing = editing?.copy(imageMediaId = "", imageThumb = null)
    }

    /**
     * **يحفظ المسوّدة.**
     *
     * **وحقلُ الصورة يُرسَل فقط إن مُسّ** — `null` يعني «لا تمسّها»،
     * **ومن أرسله في كلّ حفظٍ محا صورةَ صنفٍ كلَّما بُدّل اسمُه.**
     */
    fun saveItem() {
        val d = editing ?: return
        // ══════════════════════════════════════════════════════════════
        // **والرفضُ يُقال ولا يُسكَت عنه**
        // ══════════════════════════════════════════════════════════════
        //
        // **كان `return` صامتاً**: يضغط «حفظ» فلا يُغلق النموذجُ ولا
        // تظهر رسالةٌ ولا يقع شيء — **فيظنّ الزرَّ معطوباً** فيضغطه
        // مرّةً بعد مرّة. **وشاشةٌ لا تردّ أسوأُ من شاشةٍ ترفض.**
        //
        // (كشفه جردُ الحالات ٢٠٢٦-٠٨-٣٠.)
        if (busy) return
        // **وأسبابُ الرفض فوق زرّ الحفظ** كما عند المتجر (تدقيقُ ٢٠٢٦-١٠-٢٨).
        if (d.name.isBlank()) {
            editError = getApplication<Application>().getString(R.string.mn_need_name)
            return
        }
        // **وبالدولار يُفحص الدولار** (٢٠٢٦-١٠-١٢) — والليرةُ يحسبها الخادم.
        // **والصفرُ مسموح** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — والفارغُ لا.
        val usd = if (d.isUsd) com.rahalgo.ui.menu.parseUsd(d.priceUsd) else null
        val price = if (d.isUsd) null else d.price.trim().toLongOrNull()
        if ((d.isUsd && usd == null) || (!d.isUsd && (price == null || price < 0))) {
            editError = getApplication<Application>().getString(R.string.mn_need_price)
            return
        }
        // **مجموعةٌ بلا اسم، أو إلزاميّةٌ بلا خيارٍ متوفّر** — يُقال بوضوح قبل الإرسال.
        d.optionsError()?.let { editError = it; return }
        // **وبلا قسمٍ يُحفظ ولا يُمنع** — لوحةُ الويب تسمح به، ومنعُ
        // التطبيقِ ما يسمح به الويبُ فرقٌ بين الشاشتين. **والتنبيهُ في
        // النموذج تحت مختار القسم** — كما في الويب حرفاً.
        val input = ItemInput(
            name = d.name.trim(),
            price = price,
            // **والفراغُ الصريحُ يرفع التصنيف** — يقرؤه المحرّك «ارفع»
            // لا «بلا تغيير»، كما ترسله لوحةُ الويب حرفا.
            platformSectionId = d.platformSectionId,
            // **ولا تُرسَل الإتاحةُ من النموذج** — الويبُ يقلبها من
            // السطر وحدَه. **وحقلٌ يُكتب من موضعين يمحو أحدُهما ما فعله
            // الآخر**: من أوقف صنفاً ثمّ عدّل اسمَه أعاده متوفّرا.
            imageMediaId = d.imageMediaId,
            // **وتُرسَل الشجرةُ كاملةً من هنا** — الويبُ يرسلها في كلّ
            // حفظٍ كذلك، **والحذفُ لا باب له غير الاستبدال**: من أزال
            // مجموعةً ولم تُرسَل الشجرةُ بقيت في القاعدة.
            //
            // **وتُنظَّف قبل الإرسال**: مجموعةٌ بلا اسمٍ أو بلا خيارٍ
            // واحدٍ تُسقَط، **وخيارٌ بلا اسمٍ يُسقَط** — **وصنفٌ يُعرض
            // للزبون بمجموعةٍ بلا اسمٍ يُوقفه عن الطلب.**
            modifiers = d.modifiersForSave()
                .filter { it.name.isNotBlank() && it.options.isNotEmpty() },
            priceCurrency = d.currency,
            priceUsd = usd,
            storeDrinks = d.storeDrinks,
        )
        // **ومفتاحُ المحاولة للإنشاء وحدَه** (`DUP-LEAD`) — يثبت حتّى ينجح
        // أو يحسمه الخادم، **فإعادةٌ بعد انقطاعٍ لا تُدرج صنفاً ثانياً.**
        // والتعديلُ يُعاد بلا ضرر: يكتب القيمَ نفسَها.
        write(onFail = { e ->
            if (d.isNew && com.rahalgo.ui.isDecided(e)) {
                com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.REP_ITEM)
            }
        }) {
            if (d.isNew) {
                val key = com.rahalgo.ui.Attempt.key(com.rahalgo.ui.Attempt.REP_ITEM)
                api.createItem(merchantID, input, idempotencyKey = key)
                com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.REP_ITEM)
            } else {
                api.updateItem(d.itemId, input)
            }
            editing = null
            Flash.ok(getApplication<Application>().getString(com.rahalgo.rep.R.string.mn_saved))
        }
    }

    // **والحذفُ يُغلق المحرِّر** (الخطوة ١٥) — رُئي على الجهاز: حُذف الصنفُ
    // **وبقيت شاشةُ تعديله مفتوحةً عليه بلا كلمة**، فيظنّه المندوبُ لم يُحذف.
    fun deleteItem(id: String) = write {
        api.deleteItem(id)
        editing = null
        Flash.ok(getApplication<Application>().getString(com.rahalgo.rep.R.string.mn_deleted))
    }

    /**
     * **كلُّ كتابةٍ تُتبَع بجلب.**
     *
     * **ولا تُعدَّل النسخةُ في الذاكرة**: سعرُ البيع والقسمُ والموافقة
     * تحسبها المنصّة، **ونسخةٌ تُصلَح بيد الجهاز تفترق عمّا في القاعدة
     * فيقرأ المندوبُ رقماً لا وجودَ له.**
     */
    private fun write(onFail: (Exception) -> Unit = {}, block: suspend () -> Unit) {
        busy = true
        viewModelScope.launch {
            try {
                block()
                error = ""
                sections = api.menu(merchantID)
                regroup()
                // **وعدّاداتُ بطاقة العميل تتحدّث** (الخطوة ١٥) — رُئي على الجهاز:
                // صنفٌ أُضيف وبقيت «أصناف معروضة ٠».
                Refresh.bump()
            } catch (e: Exception) {
                onFail(e)
                Flash.fail(apiError(getApplication(), e))
            }
            busy = false
        }
    }
}


/** **قسمٌ من أقسام السوق وأصنافُه** — انظر `regroup`. */
data class ItemGroup(
    val name: String,
    val items: List<MenuItem>,
    val imageUrl: String?,
    /** **معرّفُ قسم السوق** — وفارغٌ لـ«بلا قسم». */
    val sectionId: String = "",
)
