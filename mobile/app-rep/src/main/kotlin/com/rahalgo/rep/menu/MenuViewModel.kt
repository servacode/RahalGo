package com.rahalgo.rep.menu

import com.rahalgo.ui.menu.ItemDraft
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

    fun openGroup(g: ItemGroup) {
        open = g
    }

    fun back() {
        open = null
    }

    /**
     * **يجمع الأصنافَ بأقسام السوق.**
     *
     * **ويُعاد ضبطُ المفتوح بعد كلّ تحميل** — وإلّا بقي المندوبُ داخلَ
     * قسمٍ يحمل أصنافاً قديمة.
     */
    private fun regroup() {
        val items = sections.orEmpty().flatMap { it.items }
        // **ووجهُ القسم من أوّل صنفٍ فيه** — `PlatformSection` في واجهة
        // المندوب تحمل الاسمَ والمعرّفَ فقط، **بلا صورة.**
        groups = items
            .groupBy { it.platformSectionName.ifBlank { unsorted() } }
            .map { (name, list) ->
                ItemGroup(
                    name = name,
                    items = list.sortedBy { it.name },
                    imageUrl = list.firstNotNullOfOrNull { it.imageThumbUrl },
                )
            }
            .sortedByDescending { it.items.size }
        open = open?.let { cur -> groups.firstOrNull { it.name == cur.name } }
    }

    private fun unsorted(): String = com.rahalgo.ui.AppCore.get().app
        .getString(com.rahalgo.rep.R.string.mn_unsorted)

    /** **أقسامُ السوق** — **وبلاها لا يظهر الصنفُ في التصفّح.** */
    var platformSections by mutableStateOf<List<PlatformSection>>(emptyList())
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

    fun open(id: String, name: String) {
        merchantID = id
        merchantName = name
        sections = null
        editing = null
        error = ""
        load()
    }

    fun close() {
        merchantID = ""
        editing = null
        sections = null
    }

    fun load() {
        if (merchantID.isEmpty()) return
        busy = true
        viewModelScope.launch {
            try {
                sections = api.menu(merchantID)
                regroup()
                error = ""
            } catch (e: Exception) {
                Flash.fail(apiError(getApplication(), e))
            }
            // **وأقسامُ السوق على حدة** — **سقوطُها لا يُخفي القائمة**،
            // ويبقى المندوبُ يقرأ ما هو قائمٌ ولو تعذّر عليه أن يضيف.
            runCatching { platformSections = api.platformSections() }
            busy = false
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الأصناف**
    // ══════════════════════════════════════════════════════════════════

    /** **صنفٌ جديد** — وقسمُ السوق يُختار في النموذج. */
    fun newItem(platformSectionId: String = "") {
        editing = ItemDraft(platformSectionId = platformSectionId)
    }

    fun editItem(item: MenuItem) {
        editing = ItemDraft(
            itemId = item.id,
            name = item.name,
            // **وسعرُ المتجر لا سعرُ البيع** — هو ما يملك تغييرَه.
            price = if (item.merchantPrice > 0) item.merchantPrice.toString() else "",
            description = item.description,
            platformSectionId = item.platformSectionId.orEmpty(),
            modifiers = item.modifiers,
            imageThumb = item.imageThumbUrl,
        )
    }

    fun editDraft(block: (ItemDraft) -> ItemDraft) {
        editing = editing?.let(block)
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
    }

    /**
     * **يقلب «متاح» من القائمة مباشرة.**
     *
     * **و«نفد الصنف» تقع عشرَ مرّاتٍ في اليوم** — **ومن فتح لها نموذجاً
     * بثمانية حقولٍ لم يقلبها**، فيبقى الصنفُ يُطلب وهو ناقص.
     */
    fun toggleAvailable(item: MenuItem) = write {
        api.updateItem(item.id, ItemInput(available = !item.available))
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
        if (d.name.isBlank()) {
            Flash.fail(getApplication<Application>().getString(R.string.mn_need_name))
            return
        }
        val price = d.price.trim().toLongOrNull()
        if (price == null) {
            Flash.fail(getApplication<Application>().getString(R.string.mn_need_price))
            return
        }
        // **وبلا قسمٍ يُحفظ ولا يُمنع** — لوحةُ الويب تسمح به، ومنعُ
        // التطبيقِ ما يسمح به الويبُ فرقٌ بين الشاشتين. **والتنبيهُ في
        // النموذج تحت مختار القسم** — كما في الويب حرفاً.
        val input = ItemInput(
            name = d.name.trim(),
            description = d.description.trim(),
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
            modifiers = d.modifiers
                .map { g -> g.copy(options = g.options.filter { it.name.isNotBlank() }) }
                .filter { it.name.isNotBlank() && it.options.isNotEmpty() },
        )
        write {
            if (d.isNew) api.createItem(merchantID, input) else api.updateItem(d.itemId, input)
            editing = null
        }
    }

    fun deleteItem(id: String) = write { api.deleteItem(id) }

    /**
     * **كلُّ كتابةٍ تُتبَع بجلب.**
     *
     * **ولا تُعدَّل النسخةُ في الذاكرة**: سعرُ البيع والقسمُ والموافقة
     * تحسبها المنصّة، **ونسخةٌ تُصلَح بيد الجهاز تفترق عمّا في القاعدة
     * فيقرأ المندوبُ رقماً لا وجودَ له.**
     */
    private fun write(block: suspend () -> Unit) {
        busy = true
        viewModelScope.launch {
            try {
                block()
                error = ""
                sections = api.menu(merchantID)
                regroup()
            } catch (e: Exception) {
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
)
