package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
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
            Button(
                onClick = onAdd,
                enabled = !busy,
                colors = ButtonDefaults.buttonColors(
                    // **و`accent` هو `secondary` في سمتنا** — انظر أعلى
                    // الملفّ: لا لونَ يُخترع هنا.
                    containerColor = MaterialTheme.colorScheme.secondary,
                    contentColor = MaterialTheme.colorScheme.onSecondary,
                ),
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
) {
    // **ولا حشوةَ ولا عمودٌ حولها** — **الخريطةُ تملأ ما يُعطى لها**،
    // وحشوةٌ بستّةَ عشرَ نقطةً تجعلها بطاقةً في صفحةٍ بيضاء.
    //
    // **والوصفُ بعدها يحتاج حشوةً وتمريرا** — فيتولّاهما `AddressEditor`
    // بنفسه حين يكون قائماً بذاته (`standalone`).
    AddressEditor(vm, vm.state, picker, null, onDone, standalone = true)
}
