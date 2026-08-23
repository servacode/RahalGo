package com.rahalgo.merchant.menu

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.merchant.MenuItem
import com.rahalgo.shared.merchant.MenuItemInput
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
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

    /** **أقسامُ السوق التي فيها أصنافُه** — مرتّبةً كما يراها الزبون. */
    var groups by mutableStateOf<List<MenuGroup>>(emptyList())
        private set

    /** **القسمُ المفتوح** — وفارغٌ يعني الشبكة. (كنمط الويب.) */
    var open by mutableStateOf<MenuGroup?>(null)
        private set

    var loading by mutableStateOf(true)
        private set

    var error by mutableStateOf("")
        private set

    private var storeId: String = ""

    /** **وجهُ كلّ قسمٍ باسمه** — من `platform-sections`، تُقرأ مرّةً. */
    private var faces: Map<String, String?> = emptyMap()

    /** **أقسامُ السوق كلُّها** — لاختيار موضعِ الصنف في المحرّر. */
    var sections by mutableStateOf<List<PlatformSectionRef>>(emptyList())
        private set

    /** **الصنفُ الذي يُحرَّر الآن** — وفارغٌ يعني لا محرّر. */
    var editing by mutableStateOf<Draft?>(null)
        private set

    var busy by mutableStateOf(false)
        private set

    init {
        load()
    }

    fun openGroup(g: MenuGroup) {
        open = g
    }

    fun back() {
        open = null
    }

    fun load() {
        viewModelScope.launch {
            runCatching {
                if (storeId.isEmpty()) {
                    storeId = api.stores().stores.firstOrNull()?.id ?: ""
                }
                if (storeId.isEmpty()) {
                    error = "لا متجر مرتبط بحسابك"
                    loading = false
                    return@launch
                }
                // **ووجوهُ الأقسام تُقرأ مرّةً** — ثمانيةٌ وثلاثون سطراً
                // لا تتبدّل في اليوم، **ونداءٌ لكلّ إنعاشٍ حملٌ بلا سبب.**
                if (faces.isEmpty()) {
                    runCatching {
                        sections = api.platformSections().sections
                        faces = sections.associate { it.name to (it.imageThumbUrl ?: it.imageUrl) }
                    }
                }
                regroup(api.menu(storeId).flatMap { it.items })
                error = ""
            }.onFailure { error = it.message ?: "تعذّر جلب الأصناف" }
            loading = false
        }
    }

    /**
     * **يجمع الأصنافَ بقسم السوق** — والقسمُ بلا اسمٍ يُجمع تحت «غير مصنّف».
     *
     * **ولا يُسقط ما لا قسمَ له**: صنفٌ يختفي من شاشة صاحبه **يُقرأ ضياعاً
     * في المنصّة** — ويُعرض ويُقال عنه.
     */
    private fun regroup(items: List<MenuItem>) {
        val byName = items.groupBy { it.platformSectionName.ifBlank { UNSORTED } }
        groups = byName.map { (name, list) ->
            MenuGroup(
                name = name,
                items = list.sortedBy { it.name },
                // ══════════════════════════════════════════════════════
                // **ووجهُ القسم من السوق لا من بضاعته**
                // ══════════════════════════════════════════════════════
                //
                // **كانت الشبكةُ تأخذ صورةَ أوّلِ صنفٍ فظهرت حروفاً** —
                // قِيس على الجهاز ٢٠٢٦-٠٨-٢٣: **أصنافُ المتجر بلا صورٍ
                // بعد**، وأقسامُ السوق مصوَّرةٌ كلُّها.
                //
                // **وصنفُه احتياطٌ لها** — إن دخل قسمٌ بلا صورةٍ يوماً.
                imageUrl = faces[name]
                    ?: list.firstNotNullOfOrNull { it.imageThumbUrl ?: it.imageUrl },
            )
        }.sortedByDescending { it.items.size }

        // **والمفتوحُ يُحدَّث لا يُغلق** — من بدّل توفّرَ صنفٍ وهو داخلَ
        // قسمٍ **لا تُقذف به الشبكةُ إلى الخارج.**
        open = open?.let { cur -> groups.firstOrNull { it.name == cur.name } }
    }

    /**
     * **يقلب التوفّرَ فوراً ثمّ يُرسل.**
     *
     * **وانتظارُ رحلةِ شبكةٍ عند كلّ ضغطةٍ يجعل التطبيقَ يبدو ثقيلاً** على
     * شبكة الرقّة. **وإن ردّ المحرّكُ خطأً عاد الحالُ وقيلت الرسالة** —
     * **وشاشةٌ تقول «متوفر» والمحرّكُ يقول غيرَه تجعل الزبونَ يطلب ما نفد.**
     */
    fun toggle(itemId: String) {
        val before = groups
        val beforeOpen = open
        var next = false
        groups = groups.map { g ->
            g.copy(
                items = g.items.map { item ->
                    if (item.id != itemId) item
                    else {
                        next = !item.available
                        item.copy(available = next)
                    }
                },
            )
        }
        open = open?.let { cur -> groups.firstOrNull { it.name == cur.name } }

        viewModelScope.launch {
            runCatching { api.setAvailability(itemId, next) }
                .onFailure {
                    groups = before
                    open = beforeOpen
                    Flash.fail(it.message ?: "تعذّر تبديل التوفر")
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
        val ps = sections.firstOrNull { it.name == sectionName }?.id.orEmpty()
        editing = Draft(platformSectionId = ps)
    }

    fun editItem(item: MenuItem) {
        editing = Draft(
            itemId = item.id,
            name = item.name,
            description = item.description,
            price = if (item.price > 0) item.price.toString() else "",
            platformSectionId = item.platformSectionId.orEmpty(),
            imageThumb = item.imageThumbUrl ?: item.imageUrl,
        )
    }

    fun editDraft(block: (Draft) -> Draft) {
        editing = editing?.let(block)
    }

    fun cancelEdit() {
        editing = null
    }

    /**
     * **يرفع الصورةَ فورَ اختيارها** — لا عند الحفظ.
     *
     * **ومن انتظر الحفظَ لا يعرف أنجحت أم لا** حتّى يُغلق النموذجَ،
     * **ورفعٌ يفشل عند الحفظ يُضيّع ما كتبه معها.**
     */
    fun pickImage(bytes: ByteArray) {
        val d = editing ?: return
        busy = true
        viewModelScope.launch {
            runCatching { api.uploadItemImage("item.jpg", bytes) }
                .onSuccess { editing = d.copy(imageMediaId = it, imageThumb = null) }
                .onFailure { Flash.fail(it.message ?: "تعذّر رفع الصورة") }
            busy = false
        }
    }

    /** **يمحو الصورة** — والفراغُ الصريحُ يعني «أزِلها» لا «لا تمسّها». */
    fun clearImage() {
        editing = editing?.copy(imageMediaId = "", imageThumb = null)
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
        if (d.name.isBlank()) {
            Flash.fail("اكتب اسم الصنف")
            return
        }
        val price = d.price.trim().toLongOrNull()
        if (price == null || price <= 0) {
            Flash.fail("اكتب سعر الشراء")
            return
        }
        if (d.platformSectionId.isBlank()) {
            Flash.fail("اختر قسم السوق — وصنف بلا قسم لا يراه زبون")
            return
        }
        busy = true
        viewModelScope.launch {
            val input = MenuItemInput(
                name = d.name.trim(),
                description = d.description.trim(),
                price = price,
                platformSectionId = d.platformSectionId,
                imageMediaId = d.imageMediaId,
            )
            runCatching {
                if (d.itemId.isEmpty()) api.createItem(storeId, input)
                else api.updateItem(d.itemId, input)
            }.onSuccess {
                editing = null
                load()
            }.onFailure { Flash.fail(it.message ?: "تعذّر حفظ الصنف") }
            busy = false
        }
    }

    fun deleteItem(id: String) {
        busy = true
        viewModelScope.launch {
            runCatching { api.deleteItem(id) }
                .onSuccess {
                    editing = null
                    load()
                }
                .onFailure { Flash.fail(it.message ?: "تعذّر حذف الصنف") }
            busy = false
        }
    }

    private companion object {
        const val UNSORTED = "غير مصنّف"
    }
}

/** **مسوّدةُ صنفٍ في المحرّر** — ومعرّفٌ فارغٌ يعني جديداً. */
data class Draft(
    val itemId: String = "",
    val name: String = "",
    val description: String = "",
    /** **سعرُ الشراء لا سعرُ البيع** — وهو ما يقبضه. */
    val price: String = "",
    val platformSectionId: String = "",
    /** `null` لم تُمسّ · `""` أزِلها · معرّفٌ صورةٌ جديدة. */
    val imageMediaId: String? = null,
    val imageThumb: String? = null,
)

/** **قسمُ سوقٍ فيه أصنافُه** — والاسمُ هو المفتاح، فهو ما يراه الزبون. */
data class MenuGroup(
    val name: String,
    val items: List<MenuItem>,
    val imageUrl: String?,
)
