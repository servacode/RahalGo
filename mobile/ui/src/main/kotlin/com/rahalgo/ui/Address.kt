package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.activity.compose.BackHandler
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Address

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عنوانُ التوصيل — قطعةٌ واحدةٌ تُستدعى ولا تُبنى**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «المفروض تصير طريقةُ الخريطة مركزيّة مشان ما
 *  نظلّ نبنيها بكلّ هالصفحات… مركزيّ ونستدعيها، وربّما تلزمنا بمكانٍ
 *  آخرَ بتطبيقٍ آخر — وهيك نحلّت مشكلةُ الخريطة وتحديد العنوان بشكلٍ
 *  كامل».)
 *
 * # ما كان متفرّقا
 *
 * **أربعُ شاشاتٍ تسأل السؤالَ نفسَه**: الشريطُ العلويُّ، و«حسابي»،
 * والطلبُ الخاصّ، والسلّة. **وكلٌّ تحمل رايتَها وتفتح لوحتَها وتقرأ
 * الافتراضيَّ بنفسها.**
 *
 * **وأربعُ نسخٍ من سؤالٍ واحدٍ تفترق حتماً** — تُزاد ميزةٌ في واحدةٍ
 * وتُنسى في ثلاث.
 *
 * # وما صار
 *
 * **حاملٌ ساكنٌ يعرف ما هو مفتوح** (`DeliveryAddress`)، **ومضيفٌ واحدٌ
 * يرسمه** (`AddressHost`) يُركَّب مرّةً في جذر التطبيق — **كما `Flash`
 * سواءً بسواء.**
 *
 * **والشاشةُ تنادي `DeliveryAddress.open()` ولا تعرف ما يقع بعدها.**
 *
 * # ولماذا حاملٌ ساكنٌ لا حالُ شاشة
 *
 * **الشاشةُ تُتلف حين ينتقل** — ومن فتح اللوحةَ من السلّة ثمّ رجع
 * تُلفت معه. **والساكنُ يبقى**، فتُفتح من أيّ موضعٍ وتُغلق إلى حيث كان.
 *
 * # ولا يعرف حساباً ولا خريطة
 *
 * **`AddressHost` يأخذ النموذجَ والمنتقيَ من مُركِّبه** — **و`:ui` لا
 * تعرف الخريطة**: `:map` تعتمد عليها، **ولو عكسنا لصار الاعتمادُ
 * دائريّاً** ولَحمل كلُّ تطبيقٍ مكتبةَ خرائطَ لا يرسمها.
 */
object DeliveryAddress {

    /** **أمفتوحةٌ لوحةُ الاختيار؟** */
    var picking by mutableStateOf(false)
        private set

    /**
     * **ما يُحرَّر الآن** — فارغٌ مغلق، و`new` جديد، وما عداهما معرّفُ
     * عنوانٍ يُعدَّل.
     *
     * **ورايتان (يُضاف · يُعدَّل) ترتفعان معاً حالٌ لا معنى لها.**
     */
    var editing by mutableStateOf("")
        private set

    /** **يفتح لوحةَ الاختيار** — من أيّ شاشة. */
    fun open() {
        picking = true
    }

    /** **ويفتح المحرّرَ رأساً** — لعنوانٍ بعينه أو لجديد. */
    fun edit(id: String?) {
        picking = false
        editing = id ?: "new"
    }

    fun close() {
        picking = false
        editing = ""
    }
}

/**
 * **مضيفُ العنوان — يُركَّب مرّةً في جذر التطبيق.**
 *
 * **ويُرسَم قبل الهيكل** — صفحةُ الخريطة تملأ الشاشةَ **ولا تُرسم داخل
 * عمودٍ يمرّر**: ارتفاعٌ لا حدَّ له داخل تمريرٍ يُسقط الرسم.
 *
 * **ويردّ `true` إن كان يغطّي الشاشة** — فيمتنع مُركِّبُه عن رسم ما
 * تحته: **شريطٌ علويٌّ فوق خريطةٍ يجعلها جزءاً من شاشةٍ أخرى.**
 */
@Composable
fun AddressHost(vm: AccountViewModel, picker: PointPicker?): Boolean {
    if (DeliveryAddress.editing.isNotEmpty() && picker != null) {
        AddressEditor(
            vm = vm,
            s = vm.state,
            picker = picker,
            existing = vm.state.addresses.firstOrNull { it.id == DeliveryAddress.editing },
            onDone = { DeliveryAddress.close() },
            standalone = true,
        )
        return true
    }
    if (DeliveryAddress.picking) {
        AddressSheet(
            addresses = vm.state.addresses,
            busy = vm.state.busy,
            onPick = {
                vm.makeDefault(it.id)
                DeliveryAddress.close()
            },
            onAdd = { DeliveryAddress.edit(null) },
            onClose = { DeliveryAddress.close() },
        )
    }
    return false
}

/** **العنوانُ المختار** — الافتراضيُّ في حسابه. */
fun selectedAddress(addresses: List<Address>): Address? =
    addresses.firstOrNull { it.isDefault }

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أين تريد التوصيل؟ — نافذةٌ منبثقةٌ من الأسفل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (صورةُ المالك المرجعيّة ٢٠٢٦-٠٨-١٨، وتصحيحُه: «نافذةٌ منبثقة».)
 *
 * # ولماذا منبثقةٌ لا شاشة
 *
 * **اختيارُ العنوان قرارٌ لا رحلة** — **ومن خرج إلى شاشةٍ كاملةٍ ليختار
 * سطراً فقد ما كان يفعله**، فيعود ولا يجد موضعَه. **والمنبثقةُ تُغلق
 * فيبقى حيث كان.**
 *
 * **وما خلفها يُعتَّم ولا يُمحى** — يرى أنّه لم يغادر.
 *
 * # وألوانُها من لوحتنا لا من الصورة
 *
 * (قاعدةُ المالك ٢٠٢٦-٠٨-١٨: «الألوانُ نأخذها من تصميمنا ولا نضيف
 *  ألواناً جديدة».)
 *
 * **والبرتقاليُّ عندنا `accent`** — وهو `secondary` في السمة و`onSecondary`
 * مقابلُه: **زوجٌ معرَّفٌ منذ بنيت اللوحة، لا لونٌ يُخترع هنا.**
 *
 * # ولا تعديلَ فيها ولا حذف
 *
 * (قرارُ المالك: «قسمُ حسابي **فقط** يُظهر العناوينَ المحفوظة ويمكن
 *  تعديلها أو حذفها».)
 *
 * **وهذه لوحةُ اختيارٍ لا لوحةُ إدارة** — **ومن فتحها ليطلب فوجد أزرارَ
 * حذفٍ حذف بالخطأ وهو مستعجل.**
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddressSheet(
    addresses: List<Address>,
    busy: Boolean,
    /** **يُختار عنوانٌ** — فيصير الافتراضيّ. */
    onPick: (Address) -> Unit,
    /** **ويُضاف جديدٌ** — يفتح الخريطةَ ثمّ ورقةَ الوصف. */
    onAdd: () -> Unit,
    onClose: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Rahal.colors.canvas,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp)) {

            // ══════════════════════════════════════════════════════════
            // **ترويسةٌ تسأل ثمّ تشرح**
            // ══════════════════════════════════════════════════════════
            //
            // **«أين تريد التوصيل؟» سؤالٌ لا عنوانُ قسم** — **وعنوانٌ
            // يصف ما تحته يُقرأ ولا يُجاب**، والسؤالُ يُجاب.
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    Modifier
                        .size(48.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.warnTint),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_pin),
                        contentDescription = null,
                        tint = Rahal.colors.accent,
                        modifier = Modifier.size(24.dp),
                    )
                }
                Spacer(Modifier.size(12.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.addr_sheet_title),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleLarge,
                    )
                    Text(
                        stringResource(R.string.addr_sheet_hint),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }

            Spacer(Modifier.height(20.dp))

            if (addresses.isEmpty()) {
                EmptyAddresses()
            } else {
                addresses.forEach { a -> AddressChoice(a, busy, onPick) }
            }

            // ══════════════════════════════════════════════════════════
            // **وزرُّ الإضافة ممتلئٌ لا مفرَّغ**
            // ══════════════════════════════════════════════════════════
            //
            // **وهو الفعلُ الوحيدُ لمن لا عنوانَ له** — **وزرٌّ مفرَّغٌ
            // يُقرأ خياراً ثانياً**، والممتلئُ يقول: هذا ما تفعله.
            Spacer(Modifier.height(20.dp))
            RahalButton(
                onClick = onAdd,
                enabled = !busy,
                // **و`accent` هي `secondary` في سمتنا** — والنبرةُ تقول
                // المعنى، **واللونُ يقول الشكلَ فيصير موضعاً يُبحث عنه
                // يومَ يتبدّل.**
                tone = Tone.Accent,
                modifier = Modifier.fillMaxWidth().height(52.dp),
            ) { Text(stringResource(R.string.addr_sheet_add)) }

            Spacer(Modifier.height(24.dp))
        }
    }
}

/**
 * **حالُ الفراغ — دبّوسٌ وسطران.**
 *
 * **وسطرٌ رماديٌّ وحدَه يُقرأ عطباً** — **ومن فتح لوحةً فوجدها فارغةً
 * بلا كلمةٍ ظنّ أنّها لم تُحمَّل بعد.** والسطرُ الثاني يقول ماذا يفعل.
 */
@Composable
private fun EmptyAddresses() {
    Column(
        Modifier.fillMaxWidth().padding(vertical = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(96.dp)
                .clip(CircleShape)
                .background(Rahal.colors.field),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_pin),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(44.dp),
            )
        }
        Spacer(Modifier.height(16.dp))
        Text(
            stringResource(R.string.addr_sheet_empty),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.addr_sheet_empty_hint),
            color = Rahal.colors.inkMuted,
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/** **عنوانٌ يُختار** — والمختارُ يُعرف بلونه لا بعلامةٍ ثانية. */
@Composable
private fun AddressChoice(a: Address, busy: Boolean, onPick: (Address) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(enabled = !busy) { onPick(a) }
            .padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = if (a.isDefault) Rahal.colors.accent else Rahal.colors.inkMuted,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = stringResource(addressKindLabel(a.kind)),
                fontWeight = FontWeight.Medium,
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                text = a.text,
                color = Rahal.colors.inkMuted,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
    HorizontalDivider()
}

/**
 * **اسمُ نوع العنوان** — رمزٌ في القاعدة وكلمةٌ في الشاشة.
 *
 * **وهنا لا في شاشةِ الحساب وحدَها** — **ونسختان من الخريطة نفسِها
 * تفترقان يومَ يُزاد نوعٌ ثالث.**
 */
fun addressKindLabel(kind: String): Int = when (kind) {
    "home" -> R.string.addr_kind_home
    "work" -> R.string.addr_kind_work
    else -> R.string.addr_kind_other
}

/**
 * **إضافةُ عنوانٍ من الشريط** — خريطةٌ ثمّ ورقةُ وصف.
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يصبح إضافةُ العنوان بسهولةٍ من الأعلى».)
 *
 * **وهي المحرّرُ نفسُه الذي في «حسابي»** — **ونسختان من نموذجٍ بأربعة
 * حقولٍ تعنيان موضعين يُصلَح فيهما العيبُ ويُنسى ثانيهما.**
 */
@Composable
fun AddAddressFlow(
    vm: AccountViewModel,
    picker: PointPicker,
    onDone: () -> Unit,
    /** **عنوانٌ يُعدَّل** — أو فارغٌ لجديد. */
    existing: Address? = null,
) {
    // **ولا حشوةَ ولا عمودٌ حولها** — **الخريطةُ تملأ ما يُعطى لها**،
    // وحشوةٌ بستّةَ عشرَ نقطةً تجعلها بطاقةً في صفحةٍ بيضاء.
    //
    // **والوصفُ بعدها يحتاج حشوةً وتمريرا** — فيتولّاهما `AddressEditor`
    // بنفسه حين يكون قائماً بذاته (`standalone`).
    AddressEditor(vm, vm.state, picker, existing, onDone, standalone = true)
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ عنوان التوصيل — تُضغط فتفتح اللوحةَ نفسَها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نغيّر دورَ الزرّ أيضاً بالطلب الخاصّ بحيث
 *  يظهر العنوانُ المحفوظ أو إضافةُ عنوان، ويعمل بنفس الدور — إمّا يختار
 *  عنوانَه الافتراضيَّ أو يضيف عنواناً جديداً».)
 *
 * # ولماذا لا حقلُ نصٍّ ولا خريطةٌ هنا
 *
 * **الطلبُ الخاصُّ كان يسأل عنواناً نصّاً ونقطةً على خريطة** — **فيكتب
 * الزبونُ عنوانَه في كلّ طلبٍ من جديد**، ويلتقط نقطتَه من جديد.
 *
 * **وعنوانُه محفوظٌ في حسابه** — **وسؤالُ ما هو معروفٌ يُقرأ عدمَ ثقةٍ
 * لا حرصا.**
 *
 * # وبابٌ واحدٌ للاختيار وللإضافة
 *
 * **يُضغط فتُفتح لوحةُ العناوين**: إن كان له عنوانٌ اختاره، وإن لم يكن
 * أضافه. **وبابان أحدُهما «اختر» والآخرُ «أضف» يجعلان من لا عنوانَ له
 * يضغط الأوّلَ فيجد فراغا.**
 */
@Composable
fun AddressCard(
    /** **العنوانُ المختار** — أو فارغٌ فلا عنوانَ بعد. */
    address: Address?,
    onOpen: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .border(1.dp, Rahal.colors.line, RoundedCornerShape(14.dp))
            .clickable(onClick = onOpen)
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.top_deliver_to),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
            // **واسمُ العنوان لا سطرُه** — (طلبُ المالك ٢٠٢٦-٠٨-١٨):
            // **وسطرٌ فيه منطقةٌ وشارعٌ وطابقٌ يُقرأ نصفُه**، والاسمُ
            // يُقرأ بنظرة.
            Text(
                text = address?.let { stringResource(addressKindLabel(it.kind)) }
                    ?: stringResource(R.string.addr_card_none),
                fontWeight = FontWeight.Bold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun AddressEditor(
    vm: AccountViewModel,
    s: AccountState,
    picker: PointPicker?,
    /** **عنوانٌ يُعدَّل** — أو فارغٌ لجديد. */
    existing: Address?,
    onDone: () -> Unit,
    /**
     * **أهو صفحةٌ قائمةٌ بذاتها؟**
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «تفتح صفحةُ الخريطة بشكلٍ كاملٍ ومنفصل»
     *  ثمّ «بعد اختيار العنوان يظهر بنافذةٍ فوق الخريطة… وليس بصفحةٍ
     *  معزولةٍ كما هي الآن».)
     *
     * **ومن الشريط: خريطةٌ تملأ الشاشةَ ونافذةٌ تطفو فوقها** — **والخريطةُ
     * تبقى مرئيّةً وهو يصف**: يقرأ ما تحت الدبّوس فيصحّح وصفَه، **ومن
     * وُضع في صفحةٍ بيضاء نسي أين كانت نقطتُه.**
     *
     * **ومن «حسابي»: جزءٌ من شاشة** — **وعمودٌ يمرّر داخل عمودٍ يمرّر
     * يُجمّد أحدَهما.**
     */
    standalone: Boolean = false,
) {
    var area by rememberSaveable(existing?.id) { mutableStateOf(existing?.areaBuilding ?: "") }
    var street by rememberSaveable(existing?.id) { mutableStateOf(existing?.street ?: "") }
    var floor by rememberSaveable(existing?.id) { mutableStateOf(existing?.floor ?: "") }
    var kind by rememberSaveable(existing?.id) { mutableStateOf(existing?.kind ?: "home") }
    var lat by rememberSaveable(existing?.id) { mutableStateOf(existing?.lat) }
    var lng by rememberSaveable(existing?.id) { mutableStateOf(existing?.lng) }
    // **والجديدُ يفتح الخريطةَ أوّلاً** — الموضعُ قبل الوصف.
    var onMap by rememberSaveable(existing?.id) { mutableStateOf(existing == null) }

    val save: () -> Unit = {
        if (existing == null) {
            vm.addAddress(area, street, floor, kind, lat, lng)
        } else {
            vm.updateAddress(existing.id, area, street, floor, kind, lat, lng)
        }
        onDone()
    }
    // **والحفظُ لا يُتاح بلا نقطةٍ ولا بلا وصف** — نقطةٌ صفريّةٌ تُرسل
    // سائقاً إلى لا مكان، **وعنوانٌ بلا منطقةٍ لا يُقرأ.**
    val canSave = !s.busy && area.isNotBlank() && street.isNotBlank() &&
        lat != null && lng != null

    // ══════════════════════════════════════════════════════════════════
    // **صفحةً كان أم جزءاً — الخريطةُ أوّلاً**
    // ══════════════════════════════════════════════════════════════════
    if (standalone) {
        BackHandler { if (onMap) onDone() else onMap = true }

        Box(Modifier.fillMaxSize()) {
            picker?.invoke(
                { la, ln, name ->
                    lat = la
                    lng = ln
                    // **واسمُ المكان يملأ المنطقةَ إن كانت فارغة** — ولا
                    // يمحو ما كتبه بيده.
                    if (area.isBlank() && name.isNotBlank()) area = name
                    onMap = false
                },
                { onDone() },
            )
        }

        // **والنافذةُ تطفو فوقها** — **وإغلاقُها يعيده إلى الخريطة لا
        // يُخرجه**: من سحبها ليقرأ ما تحتها لا يريد أن يبدأ من جديد.
        if (!onMap) {
            ModalBottomSheet(
                onDismissRequest = { onMap = true },
                sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
                containerColor = Rahal.colors.canvas,
            ) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = 20.dp),
                ) {
                    AddressFields(
                        area, { area = it },
                        street, { street = it },
                        floor, { floor = it },
                        kind, { kind = it },
                    )
                    Spacer(Modifier.height(20.dp))
                    RahalButton(
                        onClick = save,
                        enabled = canSave,
                        tone = Tone.Accent,
                        // **وارتفاعُه من حشوة الزرّ لا من رقمٍ هنا** —
                        // **و٥٢ مكتوبةً في موضعٍ واحدٍ تجعل هذا الزرَّ
                        // أعلى من كلّ زرٍّ آخرَ في التطبيق** بلا سبب.
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(R.string.addr_save)) }
                    Spacer(Modifier.height(24.dp))
                }
            }
        }
        return
    }

    // ══════════════════════════════════════════════════════════════════
    // **وجزءاً من «حسابي» — بلا نافذةٍ ولا تمرير**
    // ══════════════════════════════════════════════════════════════════
    if (onMap && picker != null) {
        picker(
            { la, ln, name ->
                lat = la
                lng = ln
                if (area.isBlank() && name.isNotBlank()) area = name
                onMap = false
            },
            { if (existing == null) onDone() else onMap = false },
        )
        return
    }

    AddressFields(
        area, { area = it },
        street, { street = it },
        floor, { floor = it },
        kind, { kind = it },
    )

    // **وتغييرُ الموضع بابٌ ظاهرٌ في التعديل.**
    if (picker != null) {
        Spacer(Modifier.height(8.dp))
        RahalTextButton(onClick = { onMap = true }) {
            Text(stringResource(R.string.addr_change_point))
        }
    }

    Spacer(Modifier.height(12.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        RahalButton(onClick = save, enabled = canSave, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.addr_save))
        }
        RahalOutlineButton(onClick = onDone, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.acc_delete_cancel))
        }
    }
}

/**
 * **حقولُ العنوان — واحدةٌ للنافذة ولشاشة الحساب.**
 *
 * **ونسختان من نموذجٍ بأربعة حقولٍ تعنيان موضعين يُصلَح فيهما العيبُ
 * ويُنسى ثانيهما.**
 */
@Composable
private fun AddressFields(
    area: String,
    onArea: (String) -> Unit,
    street: String,
    onStreet: (String) -> Unit,
    floor: String,
    onFloor: (String) -> Unit,
    kind: String,
    onKind: (String) -> Unit,
) {
    Text(
        stringResource(R.string.addr_confirm_title),
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.titleLarge,
    )
    Text(
        stringResource(R.string.addr_confirm_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodyMedium,
    )

    Spacer(Modifier.height(16.dp))
    FieldLabel(R.string.addr_area, required = true)
    OutlinedTextField(
        value = area,
        onValueChange = onArea,
        placeholder = { Text(stringResource(R.string.addr_area_hint)) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth(),
    )

    Spacer(Modifier.height(12.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Column(Modifier.weight(2f)) {
            FieldLabel(R.string.addr_street, required = true)
            OutlinedTextField(
                value = street,
                onValueChange = onStreet,
                placeholder = { Text(stringResource(R.string.addr_street_hint)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }
        Column(Modifier.weight(1f)) {
            FieldLabel(R.string.addr_floor, required = false)
            OutlinedTextField(
                value = floor,
                onValueChange = { v -> onFloor(v.filter { c -> c.isDigit() }) },
                placeholder = { Text(stringResource(R.string.addr_floor_hint)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **ونوعُ العنوان ثلاثُ بطاقاتٍ بأيقونات**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وحقلٌ حرٌّ يجعل لكلّ زبونٍ تسميتَه** — «البيت» و«بيتي» و«المنزل»
    // ثلاثةُ أسماءٍ لشيءٍ واحد، **ولا يُفرز ولا يُعرض بأيقونة.**
    Spacer(Modifier.height(16.dp))
    Text(
        stringResource(R.string.addr_kind),
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.bodyLarge,
    )
    Spacer(Modifier.height(8.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        KindCard("home", R.drawable.ic_home, kind == "home", Modifier.weight(1f)) { onKind("home") }
        KindCard("work", R.drawable.ic_work, kind == "work", Modifier.weight(1f)) { onKind("work") }
        KindCard("other", R.drawable.ic_star, kind == "other", Modifier.weight(1f)) { onKind("other") }
    }
}

/** **اسمُ الحقل ونجمتُه** — والنجمةُ تقول ما لا يُترك. */
@Composable
private fun FieldLabel(text: Int, required: Boolean) {
    Row {
        Text(
            stringResource(text),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyLarge,
        )
        if (required) {
            Text(
                " " + stringResource(R.string.addr_required),
                color = Rahal.colors.accent,
                style = MaterialTheme.typography.bodyLarge,
            )
        }
    }
    Spacer(Modifier.height(6.dp))
}

/**
 * **بطاقةُ نوع** — أيقونةٌ فوق اسمها.
 *
 * **والمختارُ يُعرف بحدّه وأرضه ولونه** — **ولونٌ وحدَه لا يكفي**: من
 * لا يفرّق الألوانَ لا يرى أيَّها اختير.
 */
@Composable
private fun KindCard(
    kind: String,
    icon: Int,
    on: Boolean,
    modifier: Modifier = Modifier,
    onPick: () -> Unit,
) {
    Column(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(if (on) Rahal.colors.warnTint else Rahal.colors.canvas)
            .border(
                width = if (on) 2.dp else 1.dp,
                color = if (on) Rahal.colors.accent else Rahal.colors.line,
                shape = RoundedCornerShape(14.dp),
            )
            .clickable(onClick = onPick)
            .padding(vertical = 14.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = if (on) Rahal.colors.accent else Rahal.colors.inkMuted,
            modifier = Modifier.size(24.dp),
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(addressKindLabel(kind)),
            color = if (on) Rahal.colors.accent else Rahal.colors.inkMuted,
            fontWeight = FontWeight.Medium,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}
