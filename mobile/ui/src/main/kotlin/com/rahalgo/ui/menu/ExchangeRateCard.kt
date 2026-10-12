package com.rahalgo.ui.menu

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.Card
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ سعر الصرف — للمندوب** (٢٠٢٦-١٠-١٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المندوبُ يجهّز المتجرَ قبل صاحبه** — وأصنافٌ دولاريّةٌ بلا سعر صرفٍ تُرفض.
 * فيكتبه هنا لمتجره، **وتتبعه أصنافُه الدولاريّةُ لأقرب ٥ ليرات** والليريّةُ
 * لا تتأثّر. (وصفحةُ «متجري» عند المتجر تحمل مثلَها.)
 */
@Composable
fun ExchangeRateCard(rate: Double?, busy: Boolean, editable: Boolean = true, onSave: (String) -> Unit = {}) {
    var text by rememberSaveable(rate) { mutableStateOf(usdText(rate)) }
    Card {
        // **وعند المندوب للقراءة وحدَها** (قرارُ المالك ٢٠٢٦-١٠-١٢) — يكتبه صاحبُ المتجر.
        if (!editable) {
            Text(
                stringResource(R.string.rate_title) + ": " +
                    (rate?.let { stringResource(R.string.rate_value, usdText(it)) } ?: stringResource(R.string.rate_none)),
                fontWeight = FontWeight.Bold,
            )
            Text(stringResource(R.string.rate_owner_sets), style = MaterialTheme.typography.bodySmall, color = Rahal.colors.inkMuted)
            return@Card
        }
        Text(
            stringResource(R.string.rate_title) + ": " +
                (rate?.let { stringResource(R.string.rate_value, usdText(it)) } ?: stringResource(R.string.rate_none)),
            fontWeight = FontWeight.Bold,
        )
        Text(stringResource(R.string.rate_hint), style = MaterialTheme.typography.bodySmall, color = Rahal.colors.inkMuted)
        Spacer(Modifier.height(6.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = text,
                onValueChange = { v -> text = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(10) },
                label = { Text(stringResource(R.string.rate_field)) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            RahalButton(onClick = { onSave(text) }, enabled = !busy && text.isNotBlank()) {
                Text(stringResource(R.string.rate_save))
            }
        }
    }
}

/** **قفزةٌ في سعر الصرف** — يُسأل قبل الحفظ بمثالٍ من الأصناف. */
@Composable
fun RateJumpDialog(oldRate: String, newRate: String, samples: String, onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.rate_jump_title)) },
        text = {
            Column {
                Text(stringResource(R.string.rate_jump_body, oldRate, newRate))
                if (samples.isNotBlank()) {
                    Spacer(Modifier.height(8.dp))
                    Text(stringResource(R.string.rate_jump_samples), fontWeight = FontWeight.Bold)
                    Text(samples)
                }
            }
        },
        confirmButton = { RahalTextButton(onClick = onConfirm) { Text(stringResource(R.string.rate_jump_yes)) } },
        dismissButton = { RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.rate_jump_no)) } },
    )
}

/** **سطرُ السعر في القائمة** — «$1.5 · 175 ل.س» للدولاريّ، والليرةُ وحدَها لغيره. */
fun priceLine(syp: Long, usd: Double?): String =
    if (usd != null) "$" + usdText(usd) + " · " + com.rahalgo.ui.money(syp) else com.rahalgo.ui.money(syp)
