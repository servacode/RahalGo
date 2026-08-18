package com.rahalgo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.material3.Icon
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Address

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختيارُ عنوان التوصيل — ما يفتحه زرُّ الشريط**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «ضيف لي الزرَّ بالأعلى… يصبح إضافةُ العنوان
 *  بسهولةٍ من الأعلى، وقسمُ حسابي فقط يُظهر العناوينَ المحفوظة».)
 *
 * # والمختارُ هو الافتراضيّ
 *
 * **ولا رايةٌ ثانيةٌ في الجهاز** — **وعنوانٌ يُختار في الهاتف ولا يُحفظ
 * في الحساب يضيع يومَ يُقلع التطبيق**، ويختلف بين جهازين لصاحبٍ واحد.
 *
 * **والمحرّكُ يعرف الافتراضيَّ منذ اليوم الأوّل** — يُسنده إلى الطلب من
 * لم يختر. **فاختيارُ عنوانٍ هنا يجعله الافتراضيّ**، ومصدرٌ واحدٌ يحكم
 * الشاشةَ والطلب.
 *
 * # ولا تعديلَ هنا ولا حذف
 *
 * (قرارُ المالك: «قسمُ حسابي **فقط** يُظهر العناوينَ المحفوظة ويمكن
 *  تعديلها أو حذفها».)
 *
 * **وهذه لوحةُ اختيارٍ لا لوحةُ إدارة** — **ومن فتحها ليطلب فوجد أزرارَ
 * حذفٍ حذف بالخطأ وهو مستعجل.**
 */
@Composable
fun AddressPicker(
    addresses: List<Address>,
    busy: Boolean,
    /** **يُختار عنوانٌ** — فيصير الافتراضيّ. */
    onPick: (Address) -> Unit,
    /** **ويُضاف جديدٌ** — يفتح الخريطةَ ثمّ ورقةَ الوصف. */
    onAdd: () -> Unit,
) {
    Column(Modifier.fillMaxWidth().padding(16.dp)) {
        Text(
            stringResource(R.string.top_addresses_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(12.dp))

        if (addresses.isEmpty()) {
            Text(
                stringResource(R.string.acc_addr_empty),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            Spacer(Modifier.height(10.dp))
        }

        addresses.forEach { a ->
            Row(
                Modifier
                    .fillMaxWidth()
                    .clickable(enabled = !busy) { onPick(a) }
                    .padding(vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_pin),
                    contentDescription = null,
                    // **والمختارُ يُعرف بلونه** — **ولا علامةٌ ثانيةٌ
                    // بجانبه**: رمزان لمعنًى واحدٍ يفترقان.
                    tint = if (a.isDefault) Rahal.colors.brand else Rahal.colors.inkMuted,
                    modifier = Modifier.size(20.dp),
                )
                Spacer(Modifier.size(10.dp))
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

        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = onAdd,
            enabled = !busy,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.Center,
            ) {
                Text(stringResource(R.string.acc_addr_add))
            }
        }
        Spacer(Modifier.height(8.dp))
    }
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
    Column(Modifier.fillMaxWidth().padding(16.dp)) {
        AddressEditor(vm, vm.state, picker, null, onDone)
    }
}
