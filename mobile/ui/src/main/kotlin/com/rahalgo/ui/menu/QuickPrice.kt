package com.rahalgo.ui.menu

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import com.rahalgo.design.Rahal
import com.rahalgo.ui.DialogConfirmButton
import com.rahalgo.ui.DialogDismissButton
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalDialog

/**
 * **تعديلُ السعر السريع** (قرارُ المالك ٢٠٢٦-١١-١١) — لمسةٌ على سعر الصنف تفتح نافذةً صغيرةً فيها
 * **حقلُ السعر وحدَه**، **وبعملة الصنف**: دولارٌ لصنفٍ بالدولار، وليرةٌ لصنفٍ بالليرة. ولا شيءَ غيرُ السعر يُمسّ.
 *
 * @param usd عملةُ الصنف دولار؟
 * @param initial السعرُ الحاليّ كما يُكتب في الحقل
 * @param error ما ردّه الخادمُ إن فشل الحفظ — يظهر في النافذة نفسها
 * @param onSave نصُّ السعر الجديد (صالحٌ دائماً: الزرُّ لا يعمل قبل ذلك)
 */
@Composable
fun QuickPriceDialog(
    itemName: String,
    usd: Boolean,
    initial: String,
    busy: Boolean,
    error: String,
    onSave: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    var text by remember { mutableStateOf(initial) }
    val valid = if (usd) parseUsd(text) != null else (text.trim().toLongOrNull() ?: -1) >= 0
    RahalDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.price_quick_title), fontWeight = FontWeight.Bold) },
        text = {
            Column {
                Text(itemName, color = Rahal.colors.inkMuted, style = Rahal.type.caption, maxLines = 2)
                OutlinedTextField(
                    value = text,
                    onValueChange = { text = if (usd) it.take(12) else it.filter { c -> c.isDigit() }.take(12) },
                    label = { Text(stringResource(if (usd) R.string.price_quick_usd else R.string.price_quick_syp)) },
                    singleLine = true,
                    shape = Rahal.shape.md,
                    keyboardOptions = KeyboardOptions(keyboardType = if (usd) KeyboardType.Decimal else KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                )
                if (error.isNotEmpty()) {
                    Text(error, color = Rahal.colors.danger, style = Rahal.type.caption)
                }
            }
        },
        confirmButton = {
            DialogConfirmButton(onClick = { onSave(text.trim()) }, enabled = valid && !busy) {
                Text(stringResource(R.string.act_save), fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            DialogDismissButton(onClick = { if (!busy) onDismiss() }) { Text(stringResource(R.string.act_cancel)) }
        },
    )
}
