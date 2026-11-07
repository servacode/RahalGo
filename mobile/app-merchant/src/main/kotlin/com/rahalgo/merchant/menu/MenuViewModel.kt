package com.rahalgo.merchant.menu

import com.rahalgo.ui.menu.ItemDraft
import com.rahalgo.ui.menu.modifiersForSave
import com.rahalgo.ui.menu.optionsError
import com.rahalgo.ui.menu.withUsdText
import com.rahalgo.merchant.noStoreMsg
import com.rahalgo.merchant.SelectedStore
import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.merchant.MenuItem
import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.ModifierOption
import com.rahalgo.shared.merchant.MenuItemInput
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.ui.err
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import com.rahalgo.shared.merchant.MenuSummary
import com.rahalgo.shared.merchant.MenuSummarySection
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أصنافُ المتجر — مرتَّبةً بأقسام السوق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «التسميةُ غلط، يجب أن تكون الأصناف·
 *  وتكون عبارةً عن الأقسام الموجودة بالسوق والتي يوجد لديّ أصنافٌ بها».)
 *
 * # ولماذا أقسامُ السوق لا أقسامُ مطبخه
 *
 * **المحرّكُ يردّ أقسامَ المتجر الداخليّة** (`menu_sections`) — «الرئيسيّة»
 * و«المشاوي» كما يرتّبها في مطبخه. **والزبونُ لا يراها إطلاقاً**: يتصفّح
 * السوقَ بأقسامه (`platform_sections`).
 *
 * **فمن رتّب شاشتَه بترتيب مطبخه لم يعرف أين تقع بضاعتُه في السوق** —
 * ولا لماذا لا يجدها زبونٌ يبحث في «شاورما».
 *
 * **والصنفُ يحمل قسمَه أصلاً** (`platform_section_name`) — فالتجميعُ هنا
 * بلا نداءٍ إضافيّ.
 *
 * # ولا يُعرض قسمٌ فارغ
 *
 * **ثمانيةٌ وثلاثون قسماً في السوق** ولا يملك في أكثرها شيئاً. **وشبكةٌ
 * أكثرُها فارغٌ تُخفي ما فيه.**
 */
class MenuViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    // ══════════════════════════════════════════════════════════════════
    // **Paged menu** (owner 2026-11-07: «ما يصير يكون بطيء»)
    // ══════════════════════════════════════════════════════════════════
    //
    // A mall of 5,311 items made the full menu 3 MB and the tab took 45 s on a weak link. So the
    // screen reads a light summary (section chips and counts) and then pages of 60 items; the
    // chips, «غير المتوفرة» and the search ask the server instead of filtering a full list.

    /** **Section chips with counts, unavailable and no-photo counts** — from `menu/summary`. */
    var summary by mutableStateOf(MenuSummary())
        private set

    /** **The items loaded so far for the current filter** (section / search / unavailable). */
    var items by mutableStateOf<List<MenuItem>>(emptyList())
        private set

    /** **More pages after these?** */
    var hasMore by mutableStateOf(false)
        private set

    /** **The first page of a new filter is on its way** — the grid shows a loader, not «لا شيء». */
    var listLoading by mutableStateOf(false)
        private set

    private var nextAfter = ""
    private var listJob: Job? = null
    private var moreJob: Job? = null
    private var searchJob: Job? = null

    /**
     * **بحثٌ في كلّ الأصناف** (تدقيقُ ٢٠٢٦-١٠-٢٧) — and now on the server, 350 ms after the last
     * keystroke. **وفارغٌ يعني الشبكة.**
     */
    var query by mutableStateOf("")
        private set

    fun search(text: String) {
        if (text == query) return
        query = text
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(SEARCH_DEBOUNCE_MS)
            reloadList()
        }
    }

    /** **القسمُ المفتوح** — وفارغٌ يعني «الكل». */
    var open by mutableStateOf<MenuSummarySection?>(null)
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    /**
     * **أموقوفٌ المتجرُ المختار؟** (B6) — فلا يُعدَّل قائمتُه.
     *
     * **والخادمُ هو الحُجّة** (A4)، **لكن تعطيلَ الأزرار أصدقُ** من تركها
     * تُضغط لتُردّ بخطأ.
     */
    var suspended by mutableStateOf(false)
        private set

    private var storeId: String = ""

    /** **سعرُ صرف المتجر** — لعرض «≈ ل.س» تحت السعر الدولاريّ (٢٠٢٦-١٠-١٢). */
    var exchangeRate by mutableStateOf<Double?>(null)

    /** **هل لديه برنامج الأمين؟** (٢٠٢٦-١١-٠٦) — بدونه لا حقلَ رمزٍ في الصنف. */
    var usesAmeen by mutableStateOf(false)
        private set

    /** **كم صنفاً من الأمين ينتظر قسماً** (٢٠٢٦-١١-٠٧) — يظهر زرُّ «أصناف بلا قسم» حين يزيد على صفر. */
    var pendingCount by mutableStateOf(0)
        private set

    /** **وجهُ كلّ قسمٍ باسمه** — من `platform-sections`، تُقرأ مرّةً. */
    private var faces: Map<String, String?> = emptyMap()

    /** **أقسامُ السوق كلُّها** — لاختيار موضعِ الصنف في المحرّر. */
    var sections by mutableStateOf<List<PlatformSectionRef>>(emptyList())
        private set

    /** **الصنفُ الذي يُحرَّر الآن** — وفارغٌ يعني لا محرّر. */
    /**
     * **سببُ رفض الحفظ — يُرسم فوق زرّه** (تقريرُ فحص المتجر) — كان `Flash`
     * عابراً أعلى الشاشة. **ويُمحى عند أوّل تعديل.**
     */
    var editError by mutableStateOf("")
        private set

    var editing by mutableStateOf<ItemDraft?>(null)
        private set

    var busy by mutableStateOf(false)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **والنبضةُ تقرأ الشاشةَ الظاهرةَ وحدَها** (تدقيقُ ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان كلُّ حدثٍ حيٍّ يعيد تحميلَ كلّ الشاشات معاً** — والمخفيّةُ منها أيضاً،
    // فوق أربعةَ عشرَ نداءً على باقة صاحب المتجر. **فالمخفيّةُ تُعلَّم «قديمة»
    // وتُقرأ لحظةَ تظهر** — فيبقى التحديثُ اللحظيُّ كما أراد المالك (٢٠٢٦-٠٨-٣١)
    // لما يراه، **بلا نداءٍ لما لا يراه.** وافتراضُه ظاهرٌ كما كان.
    private var stale = false
    var active: Boolean = true
        set(v) {
            val woke = v && !field
            field = v
            if (woke && stale) {
                stale = false
                load()
            }
        }

    // ══════════════════════════════════════════════════════════════════
    // **وتُعلَن قبل `init`** (تدقيقُ ٢٠٢٦-١٠-٢٨) — `init { load() }` كان يسبقها،
    // **فيُنادى `presets.refresh()` على حقلٍ لم يُبنَ بعد.**
    // ══════════════════════════════════════════════════════════════════

    /** **التصوير السريع للأصناف بلا صورة** (قرارُ المالك ٢٠٢٦-١١-٠٦). */
    val quick = com.rahalgo.merchant.quickphoto.QuickPhotoState(
        scope = viewModelScope,
        api = api,
        storeId = { storeId },
        dir = java.io.File(app.cacheDir, "quickphoto"),
        onClosed = { load() },
    )

    /** **أصناف بلا قسم** (قرارُ المالك ٢٠٢٦-١١-٠٧) — صاحبُ المتجر يستعمل الجوّالَ وحدَه. */
    val pendingSections = com.rahalgo.merchant.pendingsections.PendingSectionsState(
        scope = viewModelScope,
        api = api,
        storeId = { storeId },
        onClosed = { load() },
    )

    /**
     * **يفتح الشاشةَ من نقرة الإشعار** — والمتجرُ قد لا يكون قُرئ بعدُ إن فُتح التطبيقُ من الإشعار.
     */
    fun openPendingSections() {
        viewModelScope.launch {
            if (storeId.isEmpty()) {
                runCatching { SelectedStore.resolve(api.stores().stores)?.id.orEmpty() }
                    .onSuccess { storeId = it }
            }
            if (storeId.isNotEmpty()) pendingSections.show()
        }
    }

    // **قائمةُ المشروبات** (قرارُ المالك ٢٠٢٦-١٠-١٩) — محرّرُها مشتركٌ مع المندوب (`DrinksState`).
    val drinks = com.rahalgo.ui.menu.DrinksState(
        scope = viewModelScope,
        load = { api.drinksInfo(storeId) },
        save = { api.setDrinks(storeId, it) },
        itemCount = { summary.total },
        errText = { err(it) },
        upload = { api.uploadItemImage("drink.jpg", it) },
    )

    // **خياراتُ الأصناف والأسعارُ دفعةً واحدة** (طلبُ المالك ٢٠٢٦-١٠-٢٧) — مشتركةٌ مع المندوب.
    val presets = com.rahalgo.ui.menu.OptionPresetsState(
        scope = viewModelScope,
        load = { api.optionPresets(storeId) },
        save = { api.setOptionPresets(storeId, it) },
        errText = { err(it) },
    )
    val bulk = com.rahalgo.ui.menu.BulkPriceState(
        scope = viewModelScope,
        run = { api.bulkPrice(storeId, it) },
        onApplied = { load() },
        errText = { err(it) },
    )

    /** **المتجرُ الذي قُرئت خياراتُه** — فلا تُقرأ مع كلّ نبضةٍ حيّة، بل أوّلَ مرّةٍ وعند تبديله. */
    private var presetsFor: String? = null

    init {
        load()
        // ══════════════════════════════════════════════════════════════
        // **وتسمع النبضةَ** — انظر `StoreViewModel.saveSections`.
        //
        // **وأقسامُ المتجر تُحرَّر في شاشةٍ أخرى** — فمن حفظها هناك
        // يجد هذه الشاشةَ لا تعرف، **ويُقال له «اذهب إلى متجري» وهو
        // عائدٌ منه للتوّ.**
        viewModelScope.launch {
            com.rahalgo.ui.Refresh.tick.collect { if (it > 0L) { if (active) load() else stale = true } }
        }
    }

    /** **المسوّدةُ كما فُتحت** — فيُعرف أفيها ما لم يُحفظ. */
    private var loadedDraft: ItemDraft? = null

    /** **في المحرّر تعديلٌ لم يُحفظ؟** — فلا يُنسخ الصنفُ المحفوظُ فوقه. */
    val dirty: Boolean get() = editing != null && editing != loadedDraft

    /**
     * **نسخُ الصنف ثمّ فتحُ النسخة** (٢٠٢٦-١٠-٢٧) — قميصٌ بخمسة ألوان يُعمل مرّةً
     * ويُنسخ، **فيبدّل الصورةَ والاسمَ ويحفظ.**
     */
    fun duplicate(itemId: String) {
        if (busy || dirty) return
        busy = true
        viewModelScope.launch {
            runCatching {
                val newId = api.duplicateItem(itemId)
                // **The copy alone** — not the whole menu (a mall has thousands of items).
                val copy = api.menuItems(storeId, item = newId).items.first()
                refreshCounts()
                reloadList()
                copy
            }.onSuccess { copy ->
                // **أُغلق المحرّرُ والنسخُ ماشٍ؟** — فلا يُفتح من جديد بلا طلب.
                if (editing == null) return@onSuccess
                editItem(copy)
                Flash.ok(getApplication<android.app.Application>().getString(com.rahalgo.ui.R.string.item_duplicated))
            }.onFailure { if (editing != null) editError = err(it) }
            busy = false
        }
    }

    fun openSection(sectionId: String) {
        open = summary.sections.firstOrNull { it.id == sectionId } ?: return
        onlyOff = false
        reloadList()
    }

    fun back() {
        val changed = open != null || onlyOff
        open = null
        onlyOff = false
        if (changed) reloadList()
    }

    /**
     * **الأصنافُ غيرُ المتوفرة وحدَها** (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٨) — يفتحها زرُّ
     * «غير المتوفر» في «متجري» ورقاقتُها في «الأصناف». و«الكل» يمحوها.
     */
    var onlyOff by mutableStateOf(false)
        private set

    fun showUnavailable() {
        open = null
        query = ""
        searchJob?.cancel()
        onlyOff = true
        if (storeId.isNotEmpty()) reloadList()
    }

    fun load() {
        viewModelScope.launch {
            runCatching {
                // **والمتجرُ من الاختيار المشترك في كلّ تحميل** (B8) — لا
                // يُخبَّأ مرّةً: **من بدّل فرعَه وجب أن تُقرأ قائمةُ الفرع
                // الجديد، لا أن تبقى قائمةُ الأوّل محفوظةً في `storeId`.**
                val mine = SelectedStore.resolve(api.stores().stores)
                storeId = mine?.id.orEmpty()
                exchangeRate = mine?.exchangeRate
                usesAmeen = mine?.usesAmeen == true
                suspended = mine?.status == "suspended"
                if (storeId.isEmpty()) {
                    error = noStoreMsg()
                    loading = false
                    return@launch
                }
                // **ووجوهُ الأقسام تُقرأ مرّةً** — ثمانيةٌ وثلاثون سطراً
                // لا تتبدّل في اليوم، **ونداءٌ لكلّ إنعاشٍ حملٌ بلا سبب.**
                // **وتُقرأ في كلّ تحميلٍ لا مرّةً واحدة** — كان
                // الحارسُ `faces.isEmpty()`، **فمن أضاف قسماً بعد أوّل
                // فتحةٍ لم تُقرأ إضافتُه أبداً.**
                run {
                    runCatching {
                        // ══════════════════════════════════════════════
                        // **وأقسامُ هذا المتجر لا أقسامُ المنصّة كلُّها**
                        // ══════════════════════════════════════════════
                        //
                        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «مو معقول كلُّ
                        //  الأقسام تطلع عند كلّ المتاجر — مطعمٌ شو
                        //  علاقتُه بالأحذية؟»)
                        //
                        // **وقائمةٌ فيها ثمانيةٌ وثلاثون قسماً أكثرُها
                        // لا يعنيه تُختار منها بالخطأ** — وصنفٌ في قسمٍ
                        // غلطٍ يظهر للزبون في المكان الخطأ أو لا يظهر.
                        //
                        // **وفارغةٌ تعني الكلّ**: من لم يُعلن أقسامَه
                        // بعدُ يرى القائمةَ كاملةً كما قبل التحديث،
                        // **فلا يُحبس متجرٌ قائمٌ بلا أقسام.**
                        val mine = runCatching { api.storeSections(storeId).sections }
                            .getOrDefault(emptyList())
                        // ══════════════════════════════════════════
                        // **ولا ارتدادَ إلى أقسام المنصّة كلِّها**
                        // ══════════════════════════════════════════
                        //
                        // (قرارُ المالك ٢٠٢٦-٠٨-٢٩: «يجب أن يختار أقسامَه
                        //  ثمّ يستطيع إضافةَ صنفٍ أوّل».)
                        //
                        // **كان `mine.ifEmpty { الكلّ }`** — فمتجرٌ لم يختر
                        // أقسامَه يرى المطاعمَ والصيدليّةَ والبقالة.
                        //
                        // **وصاحبُ المتجر مستعجل**، فيضغط أوّلَ قسمٍ في
                        // القائمة — **فتنزل كنافتُه في قسم الصيدليّة.**
                        //
                        // **والخطأُ لا يُكتشف**: يرى صنفَه في تطبيقه
                        // سليماً، **ويظنّ أنّ الزبائن لا يشترون منه** —
                        // وقد يمضي شهرٌ قبل أن ينتبه أحد.
                        //
                        // **فالفارغُ يبقى فارغاً**، وشاشةُ الإضافة تقول
                        // له أين يذهب.
                        sections = mine
                        faces = sections.associate { it.name to (it.imageThumbUrl ?: it.imageUrl) }
                    }
                }
                // **The summary and the first page together** — two small answers instead of one
                // that grows with every item.
                coroutineScope {
                    val sum = async { api.menuSummary(storeId) }
                    val first = async { fetchPage("") }
                    summary = sum.await()
                    open = open?.let { cur -> summary.sections.firstOrNull { it.id == cur.id } }
                    applyFirst(first.await())
                }
                error = ""
                // **عددُ الأصناف بلا قسم** (٢٠٢٦-١١-٠٧) — للأمين وحدَه، وفشلُه لا يُسقط الشاشة.
                pendingCount = if (usesAmeen) {
                    runCatching { api.priceSyncPendingCount(storeId) }.getOrDefault(0)
                } else {
                    0
                }
                // **أوّلَ مرّةٍ وعند تبديل الفرع وحدَهما** — لا مع كلّ نبضة (تدقيقُ ٢٠٢٦-١٠-٢٨).
                if (presetsFor != storeId) {
                    if (presetsFor != null) presets.reset()
                    presetsFor = storeId
                    presets.refresh()
                    drinks.refresh()
                }
            }.onFailure { error = err(it) }
            loading = false
        }
    }

    /** **One page for the current filter** — section, search or unavailable. */
    private suspend fun fetchPage(after: String) = api.menuItems(
        storeId,
        section = if (query.isBlank()) open?.id.orEmpty() else "",
        query = query,
        filter = if (onlyOff && query.isBlank()) "unavailable" else "",
        after = after,
        limit = PAGE_SIZE,
    )

    private fun applyFirst(page: com.rahalgo.shared.merchant.MenuItemsPage) {
        items = page.items
        nextAfter = page.nextAfter
        hasMore = page.nextAfter.isNotEmpty()
    }

    /** **The filter changed** — drop the list and read its first page. */
    private fun reloadList() {
        if (storeId.isEmpty()) return
        listJob?.cancel()
        moreJob?.cancel()
        items = emptyList()
        hasMore = false
        listLoading = true
        listJob = viewModelScope.launch {
            runCatching { fetchPage("") }
                .onSuccess { applyFirst(it) }
                .onFailure { if (it !is kotlinx.coroutines.CancellationException) Flash.fail(err(it)) }
            listLoading = false
        }
    }

    /** **The grid reached its end** — the next page, once at a time. */
    fun loadMore() {
        if (!hasMore || nextAfter.isEmpty() || moreJob?.isActive == true || listJob?.isActive == true) return
        val after = nextAfter
        moreJob = viewModelScope.launch {
            runCatching { fetchPage(after) }
                .onSuccess { page ->
                    if (after != nextAfter) return@onSuccess
                    val have = items.mapTo(HashSet()) { it.id }
                    items = items + page.items.filter { it.id !in have }
                    nextAfter = page.nextAfter
                    hasMore = page.nextAfter.isNotEmpty()
                }
                .onFailure { if (it !is kotlinx.coroutines.CancellationException) Flash.fail(err(it)) }
        }
    }

    /** **Counts after a change** (duplicate, toggle failure…) — the summary only. */
    private suspend fun refreshCounts() {
        runCatching { api.menuSummary(storeId) }.onSuccess { sum ->
            summary = sum
            open = open?.let { cur -> sum.sections.firstOrNull { it.id == cur.id } }
        }
    }

    /**
     * **يقلب التوفّرَ فوراً ثمّ يُرسل.**
     *
     * **وانتظارُ رحلةِ شبكةٍ عند كلّ ضغطةٍ يجعل التطبيقَ يبدو ثقيلاً** على
     * شبكة الرقّة. **وإن ردّ المحرّكُ خطأً عاد الحالُ وقيلت الرسالة** —
     * **وشاشةٌ تقول «متوفر» والمحرّكُ يقول غيرَه تجعل الزبونَ يطلب ما نفد.**
     *
     * The item stays where it is in the list (also under «غير المتوفرة») until the next reload,
     * and the chip counts follow at once.
     */
    fun toggle(itemId: String) {
        val beforeItems = items
        val beforeSummary = summary
        val target = items.firstOrNull { it.id == itemId } ?: return
        val next = !target.available
        items = items.map { if (it.id == itemId) it.copy(available = next) else it }
        val delta = if (next) -1 else 1
        summary = summary.copy(
            unavailable = (summary.unavailable + delta).coerceAtLeast(0),
            sections = summary.sections.map { sec ->
                if (sec.id == target.platformSectionId) sec.copy(unavailable = (sec.unavailable + delta).coerceAtLeast(0)) else sec
            },
        )
        viewModelScope.launch {
            runCatching { api.setAvailability(itemId, next) }
                .onFailure {
                    items = beforeItems
                    summary = beforeSummary
                    Flash.fail(err(it))
                }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **المحرّر — إضافةٌ وتعديلٌ وصورةٌ وحذف**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «عندما أدخل إلى القسم يجب أن يكون بنفس
    //  طريقة الويب: إضافةُ صورةٍ وتعديلُ معلوماتٍ ومتوفّر/غير متوفّر
    //  وحذف».)

    /** **صنفٌ جديدٌ في قسمٍ معلوم** — فلا يُعيد اختيارَ ما هو فيه. */
    fun newItem(sectionName: String = "") {
        // **صنفٌ جديدٌ محاولةٌ جديدة** — لا يحمل مفتاحَ سابقٍ مجهولِ المصير.
        com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.MERCHANT_ITEM)
        editError = ""
        val ps = sections.firstOrNull { it.name == sectionName }?.id.orEmpty()
        editing = ItemDraft(platformSectionId = ps)
        loadedDraft = editing
    }

    fun editItem(item: MenuItem) {
        editError = ""
        editing = ItemDraft(
            itemId = item.id,
            name = item.name,
            // **والصفرُ يُعرض صفراً** (قرارُ المالك ٢٠٢٦-١٠-٢٨: «مسموح للكل»).
            price = if (item.price >= 0) item.price.toString() else "",
            platformSectionId = item.platformSectionId.orEmpty(),
            imageThumb = item.imageThumbUrl ?: item.imageUrl,
            imageFull = item.imageUrl?.takeIf { it.isNotBlank() },
            // **وتُحمَّل مع الصنف** — ومن حرّر صنفاً بلا حملها ثمّ حفظ
            // **محا إضافاتِه كلَّها**: المحرّكُ يستبدل لا يضيف
            // (`DELETE FROM modifier_groups WHERE item_id`).
            modifiers = item.modifiers.map { it.withUsdText() },
            // **وعملتُه كما حُفظت** (٢٠٢٦-١٠-١٢).
            currency = item.priceCurrency,
            priceUsd = com.rahalgo.ui.menu.usdText(item.merchantPriceUsd),
            storeDrinks = item.storeDrinks,
            externalCode = item.externalCode,
            barcode = item.barcode,
        )
        loadedDraft = editing
    }

    fun editDraft(block: (ItemDraft) -> ItemDraft) {
        editing = editing?.let(block)
        editError = ""
    }

    fun cancelEdit() {
        editError = ""
        editing = null
        loadedDraft = null
    }

    /**
     * **يرفع الصورةَ فورَ اختيارها** — لا عند الحفظ.
     *
     * **ومن انتظر الحفظَ لا يعرف أنجحت أم لا** حتّى يُغلق النموذجَ،
     * **ورفعٌ يفشل عند الحفظ يُضيّع ما كتبه معها.**
     */
    fun pickImage(bytes: ByteArray) {
        if (editing == null) return
        busy = true
        viewModelScope.launch {
            runCatching { api.uploadItemImage("item.jpg", bytes) }
                .onSuccess { ref ->
                    // ══════════════════════════════════════════════════
                    // **وتُعرض المصغّرةُ فورَ الرفع**
                    // ══════════════════════════════════════════════════
                    //
                    // **كان `imageThumb = null`** — فيرسم الحقلُ الحرفَ
                    // الاحتياطيَّ بعد رفعٍ ناجح، **ويظنّ صاحبُ المتجر
                    // أنّ الصورة ضاعت** فيعيد الاختيارَ مرّةً بعد مرّة.
                    //
                    // **والمصغّرةُ كانت تصل من المحرّك في الردّ نفسِه**
                    // (`thumb_url`) — وتُرمى.
                    //
                    // **وتُنسَخ من `editing` الحاليّ لا من نسخةٍ قديمة**:
                    // كان يُلتقَط الرسمُ قبل الرفع، **فما كُتب أثناءه
                    // يُطمس عند انتهائه.**
                    editing = editing?.copy(
                        imageMediaId = ref.id,
                        imageThumb = ref.thumbUrl.ifBlank { null },
                        imageFull = null,
                    )
                }
                .onFailure { Flash.fail(err(it)) }
            busy = false
        }
    }

    /** **يمحو الصورة** — والفراغُ الصريحُ يعني «أزِلها» لا «لا تمسّها». */
    fun clearImage() {
        editing = editing?.copy(imageMediaId = "", imageThumb = null, imageFull = null)
    }

    /**
     * **يحفظ المسوّدة.**
     *
     * **وحقلُ الصورة يُرسَل فقط إن مُسّ** — `null` يعني «لا تمسّها»،
     * **ومن أرسله في كلّ حفظٍ محا صورةَ صنفٍ كلّما بُدّل اسمُه.**
     *
     * **ولا تُرسَل الإتاحةُ من النموذج** — تُقلب من السطر وحدَه.
     * **وحقلٌ يُكتب من موضعين يمحو أحدُهما ما فعله الآخر**: من أوقف
     * صنفاً ثمّ عدّل اسمَه أعاده متوفّراً.
     */
    fun saveItem() {
        val d = editing ?: return
        if (busy || suspended) return
        // **وأسبابُ الرفض فوق زرّ الحفظ** (تدقيقُ ٢٠٢٦-١٠-٢٨) — كانت `Flash` عابرة.
        val app = com.rahalgo.ui.AppCore.get().app
        if (d.name.isBlank()) {
            editError = app.getString(com.rahalgo.merchant.R.string.v_item_name)
            return
        }
        // **وبالدولار يُفحص الدولار** (٢٠٢٦-١٠-١٢) — والليرةُ يحسبها الخادم.
        // **والصفرُ مسموح** (قرارُ المالك ٢٠٢٦-١٠-٢٨: «مسموح للكل») — والفارغُ لا.
        val usd = if (d.isUsd) com.rahalgo.ui.menu.parseUsd(d.priceUsd) else null
        val price = if (d.isUsd) null else d.price.trim().toLongOrNull()
        if ((d.isUsd && usd == null) || (!d.isUsd && (price == null || price < 0))) {
            editError = app.getString(com.rahalgo.merchant.R.string.v_item_price)
            return
        }
        if (d.platformSectionId.isBlank()) {
            editError = app.getString(com.rahalgo.merchant.R.string.v_item_section)
            return
        }
        d.optionsError()?.let { editError = it; return }
        busy = true
        viewModelScope.launch {
            val input = MenuItemInput(
                name = d.name.trim(),
                price = price,
                platformSectionId = d.platformSectionId,
                imageMediaId = d.imageMediaId,
                // **وتُرسل دائماً لا حين تتبدّل** — انظر `editItem`.
                modifiers = d.modifiersForSave(),
                priceCurrency = d.currency,
                priceUsd = usd,
                storeDrinks = d.storeDrinks,
                // **ورقمُ الأمين يُرسل دائماً** — والفارغُ يفصل الربط (٢٠٢٦-١١-٠٤).
                externalCode = d.externalCode.trim(),
                barcode = d.barcode.trim(),
            )
            runCatching {
                if (d.itemId.isEmpty()) {
                    val key = com.rahalgo.ui.Attempt.key(com.rahalgo.ui.Attempt.MERCHANT_ITEM)
                    api.createItem(storeId, input, idempotencyKey = key)
                    com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.MERCHANT_ITEM)
                } else {
                    api.updateItem(d.itemId, input)
                }
            }.onSuccess {
                editing = null
                editError = ""
                // **و«تمّ» يُقال** — كان النجاحُ صامتاً (تقريرُ فحص المتجر).
                Flash.ok(getApplication<android.app.Application>().getString(com.rahalgo.merchant.R.string.mn_item_saved))
                load()
            }.onFailure {
                if (d.itemId.isEmpty() && com.rahalgo.ui.isDecided(it)) {
                    com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.MERCHANT_ITEM)
                }
                editError = err(it)
            }
            busy = false
        }
    }

    fun deleteItem(id: String) {
        busy = true
        viewModelScope.launch {
            runCatching { api.deleteItem(id) }
                .onSuccess {
                    editing = null
                    editError = ""
                    Flash.ok(getApplication<android.app.Application>().getString(com.rahalgo.merchant.R.string.mn_item_deleted))
                    load()
                }
                .onFailure { editError = err(it) }
            busy = false
        }
    }

    private companion object {
        const val PAGE_SIZE = 60
        const val SEARCH_DEBOUNCE_MS = 350L
    }
}
