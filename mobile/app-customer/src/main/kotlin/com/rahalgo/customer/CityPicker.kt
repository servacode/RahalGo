package com.rahalgo.customer

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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.City
import com.rahalgo.ui.R as UiR

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مدينةُ التسوّق — تُقال في القائمة وتُبدَّل بضغطة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «يختار مدينةً من أوّل ما يفتح، أو بطريقةٍ
 *  احترافيّةٍ نحدّد عنوانَه تلقائيّا».)
 *
 * # ولماذا تُقال أصلاً
 *
 * **من لم يجد مطعمَه يسأل «أين ذهب؟»** — **ولا جوابَ في شاشةٍ لا تقول
 * من أين تعرض.** والسطرُ يجيب قبل أن يُسأل: «تتسوّق في الرقّة».
 *
 * **وهذه علّةُ وجوده الأولى** — والتبديلُ ثانيها.
 *
 * # ولماذا في القائمة لا في الشريط
 *
 * **الشريطُ فيه «التوصيل إلى» والجرسُ وثلاثةُ خطوط** — **ورقاقةٌ رابعةٌ
 * تدفع الجرسَ خارجَ الشاشة** في هاتفٍ ضيّق. (وقع مثلُه ٢٠٢٦-٠٨-١٨.)
 *
 * **والمدينةُ تُبدَّل مرّةً في السنة والعنوانُ في كلّ طلب** — **وما
 * يُفعل مرّةً لا يأخذ موضعَ ما يُفعل كلَّ يوم.**
 *
 * # و«تلقائيّاً» خيارٌ لا غياب
 *
 * **من ثبّت مدينةً ثمّ سافر يبقى على القديمة** — **ولا سبيلَ للرجوع إن
 * لم يكن للتلقائيّ سطرٌ يُضغط.** فهو أوّلُ السطور.
 */

/** **صفُّ «تتسوّق في…»** — يوضع فوق قائمة الزبون الجانبيّة. */
@Composable
fun CityRow(name: String, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(UiR.drawable.ic_pin),
            contentDescription = null,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(20.dp),
        )
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.city_shopping_in),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
            // **واسمٌ فارغٌ يقول «لم تُحدَّد»** — **وسطرٌ فارغٌ يُقرأ
            // عطباً في الشاشة لا حالاً في الحساب.**
            Text(
                text = name.ifEmpty { stringResource(R.string.city_unknown) },
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
        Icon(
            painter = painterResource(UiR.drawable.ic_chevron_down),
            contentDescription = null,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(18.dp),
        )
    }
}

/**
 * **ورقةُ اختيار المدينة.**
 *
 * **و`null` في `onPick` تعني «تلقائيّاً»** — لا «ألغِ».
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CitySheet(
    cities: List<City>,
    chosenID: String?,
    onPick: (City?) -> Unit,
    onClose: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Rahal.colors.canvas,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    Modifier
                        .size(48.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.warnTint),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(UiR.drawable.ic_pin),
                        contentDescription = null,
                        tint = Rahal.colors.accent,
                        modifier = Modifier.size(24.dp),
                    )
                }
                Spacer(Modifier.size(12.dp))
                Column(Modifier.weight(1f)) {
                    // **سؤالٌ لا عنوانُ قسم** — كورقة العنوان.
                    Text(
                        stringResource(R.string.city_sheet_title),
                        color = Rahal.colors.ink,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleLarge,
                    )
                    Text(
                        stringResource(R.string.city_sheet_hint),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }

            Spacer(Modifier.height(20.dp))

            // **والتلقائيُّ أوّلاً** — انظر أعلاه: **بلاه لا رجوعَ لمن
            // ثبّت مدينةً ثمّ سافر.**
            Choice(
                title = stringResource(R.string.city_auto),
                subtitle = stringResource(R.string.city_auto_hint),
                selected = chosenID == null,
                onClick = { onPick(null) },
            )
            cities.forEach { c ->
                Choice(
                    title = c.name,
                    subtitle = "",
                    selected = c.id == chosenID,
                    onClick = { onPick(c) },
                )
            }

            Spacer(Modifier.height(24.dp))
        }
    }
}

/**
 * **سطرُ اختيارٍ واحد** — والمختارُ يُعرف بلونه لا بعلامةٍ ثانية.
 *
 * **وهو نمطُ ورقةِ العنوان نفسُه** (`AddressChoice`) — **وورقتان
 * تُفتحان من قائمةٍ واحدةٍ بشكلين تجعلان التطبيقَ يبدو تطبيقين.**
 */
@Composable
private fun Choice(title: String, subtitle: String, selected: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(UiR.drawable.ic_pin),
            contentDescription = null,
            tint = if (selected) Rahal.colors.accent else Rahal.colors.inkMuted,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                title,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
                style = MaterialTheme.typography.bodyMedium,
            )
            if (subtitle.isNotEmpty()) {
                Text(
                    subtitle,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
    HorizontalDivider()
}
