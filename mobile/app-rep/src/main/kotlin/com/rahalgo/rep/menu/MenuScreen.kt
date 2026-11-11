package com.rahalgo.rep.menu

import com.rahalgo.ui.menu.ItemEditor
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأصناف — يبنيها المندوبُ نيابةً عن عميله**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نفس الفورم الموجود عند مدير المنصّة
 *  والموجود عند المتجر موجودٌ عند المندوب… باللوحة والتطبيق أيضا».)
 *
 * # وهي نسخةُ `MenuManager` في الويب — لا شاشةٌ تشبهها
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لا يجوز أن يشعر الشخصُ بالفرق بين الويب
 *  والتطبيق أصلاً — أيَّهما يفتح يكون العملُ موحّدا».)
 *
 * **فالكلماتُ من `shared.menuEditor` حرفا**، **وترتيبُ الحقول ترتيبَه**
 * (الاسم · السعر · قسم المنصة · الوصف · الصورة)، **والإتاحةُ زرٌّ في
 * السطر لا مفتاحاً في النموذج** — كما هي هناك.
 *
 * **ومن غيّر أحدَهما وحدَه أعاد الفرق.**
 *
 * # وقسمُ المنصة ليس شرطاً — إنّما يُحذَّر منه
 *
 * **الويبُ يحفظ بلا قسمٍ ويكتب تحذيراً**، **ومنعُ الحفظ يوقف المندوبَ
 * في السوق أمام صاحب متجرٍ ينتظر** — والصنفُ يبقى قابلاً للطلب من صفحة
 * متجره، ويراه الأدمنُ بلا قسمٍ فيصنّفه.
 *
 * # ولا زرَّ لإنشاء قسم
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٨: «الأقسامُ الإدارةُ هي التي تضعها، والمتجرُ
 *  أو المندوبُ يختار منتجَه بأيّ قسمٍ سينزل».)
 */
@Composable
fun MenuScreen(vm: MenuViewModel, onOffers: () -> Unit = {}) {
    val ctx = LocalContext.current
    // **والرجوعُ درجةً درجة** (الخطوة ١٥) — رُئي على الجهاز: «رجوع» من داخل
    // قسمٍ أغلق المتجرَ كلَّه وقفز فوق قائمة الأقسام.
    BackHandler {
        when {
            vm.editing != null -> vm.cancelEdit()
            vm.open != null -> vm.back()
            else -> vm.close()
        }
    }

    // **وقائمةُ المشروبات كما عند المتجر** (طلبُ المالك ٢٠٢٦-١٠-١٩) — المحرّرُ نفسُه.
    if (vm.drinks.open) {
        BackHandler { vm.drinks.close() }
        com.rahalgo.ui.menu.DrinksEditor(vm.drinks, hint = com.rahalgo.ui.R.string.drinks_hint_rep)
        return
    }

    // **خياراتُ الأصناف وتغييرُ الأسعار كما عند المتجر** (٢٠٢٦-١٠-٢٧).
    if (vm.presets.open) {
        BackHandler { vm.presets.close() }
        com.rahalgo.ui.menu.OptionPresetsEditor(vm.presets)
        return
    }
    if (vm.bulk.open) {
        BackHandler { vm.bulk.close() }
        val mine = vm.sections.orEmpty().map { it.id }.toSet()
        com.rahalgo.ui.menu.BulkPriceEditor(
            vm.bulk,
            vm.platformSections.filter { it.id in mine }.map {
                com.rahalgo.shared.merchant.PlatformSectionRef(id = it.id, name = it.name)
            },
        )
        return
    }

    if (vm.editing != null) {
        // **والمحرّرُ من الوحدة المشتركة** — نسخةٌ واحدةٌ للمتجر
        // والمندوب (`ui.menu.ItemEditor`). **وكان نموذجان بمئتين وستّين
        // سطراً بينهما يفعلان الشيءَ نفسَه.**
        ItemEditor(
            d = vm.editing!!,
            // **المشروباتُ للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
            drinksAllowed = vm.drinks.allowed,
            exchangeRate = vm.exchangeRate,
            rateMissing = com.rahalgo.ui.R.string.item_usd_need_rate_rep,
            // ══════════════════════════════════════════════════════════
            // **وأقسامُ متجرِه أوّلاً — لا كلُّ أقسام المنصّة بترتيبها**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «لا يلزم يُعرض كلّ الأقسام، لازم
            //  تطلع أقسام المتجر نفسه مشان المندوب ما يتعذّب ويدور بين
            //  الأقسام».)
            //
            // **ومتجرٌ يبيع في قسمين، وأقسامُ المنصّة عشرات** — فيبحث
            // المندوبُ في صفٍّ طويلٍ عن قسمٍ يعرفه سلفاً، **مرّةً لكلّ
            // صنفٍ يضيفه.**
            //
            // **ولا تُحذف البقيّة**: أوّلُ صنفٍ لمتجرٍ جديدٍ لا قسمَ له
            // بعد، **ومن أراد قسماً جديداً لم يجد إليه سبيلاً.** **بل
            // تُقدَّم أقسامُه ويبقى ما سواها خلفها** — فينال السرعةَ ولا
            // يفقد القدرة.
            // **وأقسامُ المتجر المختارةُ وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٢٩: «ماتطلع كل الأقسام
            // للمندوب») — كما في تطبيق المتجر. **ومتجرٌ لم يختر بعدُ يرى الكلّ** فلا يُحبس.
            sections = run {
                vm.storeSections.ifEmpty { vm.platformSections }
                    .map {
                        // **وصورةُ القسم من القسم** — **وكانت تُمرَّر
                        // فارغةً فيرسم المحرّرُ حرفاً بدل وجهِ السوق.**
                        com.rahalgo.shared.merchant.PlatformSectionRef(
                            id = it.id, name = it.name,
                            imageUrl = it.imageUrl, imageThumbUrl = it.imageThumbUrl,
                        )
                    }
            },
            busy = vm.busy,
            media = { path -> Backend.of(ctx).media(path) },
            onEdit = { f -> vm.editDraft(f) },
            onPickImage = { bytes -> vm.pickImage(bytes) },
            onClearImage = { vm.clearImage() },
            onSave = { vm.saveItem() },
            onCancel = { vm.cancelEdit() },
            onDelete = { vm.editing?.itemId?.let(vm::deleteItem) },
            // **وألفاظُ المندوب هي ألفاظُ المتجر** (مراجعةُ المالك ٢٠٢٦-١٠-٣٠):
            // «السعر» و«قسم السوق» و«وصف مختصر» — الافتراضيّةُ في المحرّر، ولا
            // سطرَ تنبيهٍ تحت القسم. ويحرسها `pnpm check:menu-parity`.
            presets = vm.presets.chosen,
            onDuplicate = { vm.editing?.itemId?.let(vm::duplicate) },
            dirty = vm.dirty,
            error = vm.editError,
        )
        return
    }

    val list = vm.sections
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load() }
        return
    }

    // ══════════════════════════════════════════════════════════════════
    // **شبكةُ الأصناف نفسُها التي عند المتجر** — `ui.menu.MenuItemsGrid`
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-١٠-٢٩: «لازم تطبيق المندوب يطبق عليه كلشي تطبق على
    //  المتجر».) **واسمُ المتجر وسعرُ صرفه فوقها** — ومندوبٌ له عشرةُ عملاء
    // لا يعرف في أيّهم يُضيف بلا اسمه.
    val media = { path: String? -> Backend.of(ctx).media(path) }
    val byId = list.flatMap { it.items }.associateBy { it.id }
    val open = vm.open
    BackHandler(enabled = open != null || vm.onlyOff) { vm.back() }
    com.rahalgo.ui.menu.MenuItemsGrid(
        groups = vm.groups.map { g ->
            com.rahalgo.ui.menu.GridGroup(
                g.name,
                g.items.map { item ->
                    val merchant = if (item.merchantPrice > 0) item.merchantPrice else item.price
                    com.rahalgo.ui.menu.GridItem(
                        id = item.id,
                        name = item.name,
                        description = item.modifiers.map { it.name }.filter { it.isNotBlank() }.joinToString(" · "),
                        imageUrl = media(item.imageThumbUrl),
                        // **والدولاريُّ بدولاره وليرته** (٢٠٢٦-١٠-١٢).
                        price = com.rahalgo.ui.menu.priceLine(merchant, item.merchantPriceUsd),
                        priceUsd = item.merchantPriceUsd,
                        priceSyp = merchant,
                        // **والسعرُ الثاني لا يُعرض إلّا إن كان أعلى** — كما في الويب.
                        customerPrice = if (item.merchantPrice > 0 && item.price > item.merchantPrice) money(item.price) else null,
                        available = item.available,
                        approved = item.approved,
                        rejected = !item.approved && item.reviewNote.isNotEmpty(),
                        reviewNote = if (item.approved) "" else item.reviewNote,
                        closedNow = item.sourceClosed,
                    )
                },
            )
        },
        query = vm.query,
        onQuery = { vm.query = it },
        openName = open?.name,
        onlyOff = vm.onlyOff,
        onAll = { vm.back() },
        onOnlyOff = { vm.showUnavailable() },
        onOpen = { name -> vm.groups.firstOrNull { it.name == name }?.let(vm::openGroup) },
        showTools = true,
        onPresets = { vm.presets.show() },
        onBulk = { vm.bulk.show() },
        onDrinks = { vm.drinks.show() },
        drinksAllowed = vm.drinks.allowed,
        addLabel = stringResource(R.string.mn_add_item),
        onAdd = {
            // **ومن داخل قسمٍ يُضاف إليه** — بمعرّفه، والاسمُ احتياطٌ لقسمٍ قديم.
            if (open != null) {
                vm.newItem(
                    open.sectionId.ifEmpty {
                        vm.platformSections.firstOrNull { it.name == open.name }?.id.orEmpty()
                    },
                )
            } else {
                vm.newItem()
            }
        },
        offerLabel = stringResource(R.string.mn_make_offer),
        onOffer = onOffers,
        emptyText = stringResource(R.string.mn_empty),
        toggleEnabled = true,
        onToggle = { g -> byId[g.id]?.let(vm::toggleAvailable) },
        onEdit = { g -> byId[g.id]?.let(vm::editItem) },
        header = {
            item(span = { androidx.compose.foundation.lazy.grid.GridItemSpan(maxLineSpan) }) {
                Column {
                    // **وسهمُ رجوعٍ إلى العميل** (مراجعةُ المالك ٢٠٢٦-١٠-٣٠) — ومن داخل قسمٍ يرجع درجة.
                    com.rahalgo.ui.PageHeader(
                        title = if (vm.merchantName.isNotBlank()) {
                            stringResource(R.string.mn_items_of, vm.merchantName)
                        } else {
                            stringResource(R.string.mn_items)
                        },
                        onBack = { if (vm.open != null || vm.onlyOff) vm.back() else vm.close() },
                    )
                    Spacer(Modifier.height(Rahal.space.sm))
                    // **وسعرُ صرف المتجر للقراءة** — يكتبه صاحبُه (قرارُ المالك ٢٠٢٦-١٠-١٢).
                    com.rahalgo.ui.menu.ExchangeRateCard(rate = vm.exchangeRate, busy = vm.busy, editable = false)
                }
            }
        },
    )
}
