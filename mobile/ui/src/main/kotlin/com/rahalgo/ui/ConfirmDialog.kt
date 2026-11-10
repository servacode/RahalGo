package com.rahalgo.ui

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight

/**
 * **سؤالُ «متأكّد؟» قبل فعلٍ لا يُرجَع** — نافذةٌ واحدةٌ للتطبيقات كلِّها.
 *
 * **والزرُّ الخطِرُ بلونه** (`danger`) — **ومن قرأ «ألغِ» بالأحمر عرف ما يضغط.**
 * **و«تراجع» هو الافتراضُ الآمن** — يُغلق النافذةَ بلا أثر.
 */
@Composable
fun ConfirmDialog(
    title: String,
    body: String,
    confirm: String,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    danger: Boolean = true,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title, fontWeight = FontWeight.Bold) },
        text = { Text(body) },
        confirmButton = {
            DialogConfirmButton(
                onClick = { onDismiss(); onConfirm() },
                tone = if (danger) Tone.Danger else Tone.Brand,
            ) {
                Text(confirm, fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            DialogDismissButton(onClick = onDismiss) {
                Text(stringResource(R.string.act_back_off))
            }
        },
    )
}
