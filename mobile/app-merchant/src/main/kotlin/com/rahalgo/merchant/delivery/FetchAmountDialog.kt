package com.rahalgo.merchant.delivery

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.DialogProperties
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.Delivery
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«إحضار»: نافذةُ موافقتك على المبلغ — تظهر من أيّ شاشة** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كنافذة تأكيد السعر عند الزبون**: لا «لاحقاً» ولا إغلاقَ بلمسةٍ خارجها — **فالسائقُ واقفٌ
 * بمكان الاستلام ينتظر**، ولا «استلمت» قبل قرارك. **زرٌّ أخضرُ ممتلئٌ للموافقة وأحمرُ للرفض.**
 * وبعد الرفض يكتب السائقُ مبلغاً جديداً فتعود النافذةُ به.
 */
@Composable
fun FetchAmountApproveDialog(
    d: Delivery,
    busy: Boolean,
    error: String,
    onApprove: () -> Unit,
    onReject: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = {},
        properties = DialogProperties(dismissOnBackPress = false, dismissOnClickOutside = false),
        title = {
            Text(stringResource(R.string.md_amount_title, d.number.toString()), fontWeight = FontWeight.Bold)
        },
        text = {
            Column(
                Modifier.fillMaxWidth().verticalScroll(rememberScrollState()),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(stringResource(R.string.md_amount_paid), color = Rahal.colors.inkMuted)
                Spacer(Modifier.height(6.dp))
                Text(
                    stringResource(R.string.md_amount_value, (d.fetchAmount ?: 0L).toString()),
                    color = Rahal.colors.accent,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineMedium,
                )
                d.collectAmount?.takeIf { it > 0 }?.let {
                    Spacer(Modifier.height(4.dp))
                    Text(stringResource(R.string.md_amount_estimate, it.toString()), color = Rahal.colors.inkMuted)
                }
                Spacer(Modifier.height(10.dp))
                Text(
                    stringResource(R.string.md_amount_body),
                    color = Rahal.colors.ink,
                    textAlign = TextAlign.Center,
                    style = MaterialTheme.typography.bodyMedium,
                )
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(8.dp))
                    Text(error, color = Rahal.colors.danger, textAlign = TextAlign.Center)
                }
                Spacer(Modifier.height(14.dp))
                RahalButton(
                    onClick = onApprove,
                    enabled = !busy,
                    tone = Tone.Success,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (busy) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text(stringResource(R.string.md_amount_approve), fontWeight = FontWeight.Bold)
                    }
                }
                Spacer(Modifier.height(8.dp))
                RahalButton(
                    onClick = onReject,
                    enabled = !busy,
                    tone = Tone.Danger,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(stringResource(R.string.md_amount_reject), fontWeight = FontWeight.Bold)
                }
            }
        },
        confirmButton = {},
    )
}
