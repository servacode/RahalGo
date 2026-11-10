package com.rahalgo.merchant.delivery

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
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
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = {},
        properties = DialogProperties(dismissOnBackPress = false, dismissOnClickOutside = false),
        title = {
            Text(
                stringResource(R.string.md_amount_title, d.number.toString()),
                style = Rahal.type.sectionTitle,
                color = Rahal.colors.ink,
            )
        },
        text = {
            Column(
                Modifier.fillMaxWidth().verticalScroll(rememberScrollState()),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(stringResource(R.string.md_amount_paid), color = Rahal.colors.inkMuted, style = Rahal.type.body)
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    // **بعملة التوصيلة** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — والدولارُ بالسنتات.
                    DeliveryTexts.amount(d, d.fetchAmount ?: 0L),
                    color = Rahal.colors.accentInk,
                    style = Rahal.type.headline,
                )
                d.collectAmount?.takeIf { it > 0 }?.let {
                    Spacer(Modifier.height(Rahal.space.xs))
                    Text(stringResource(R.string.md_amount_estimate, DeliveryTexts.amount(d, it)), color = Rahal.colors.inkMuted, style = Rahal.type.caption)
                }
                Spacer(Modifier.height(Rahal.space.md))
                Text(
                    stringResource(R.string.md_amount_body),
                    color = Rahal.colors.ink,
                    textAlign = TextAlign.Center,
                    style = Rahal.type.body,
                )
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Text(error, color = Rahal.colors.danger, textAlign = TextAlign.Center, style = Rahal.type.body)
                }
                Spacer(Modifier.height(Rahal.space.lg))
                RahalButton(
                    onClick = onApprove,
                    enabled = !busy,
                    tone = Tone.Success,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (busy) {
                        CircularProgressIndicator(Modifier.size(Rahal.icon.sm), strokeWidth = Rahal.stroke.focus)
                    } else {
                        Text(stringResource(R.string.md_amount_approve))
                    }
                }
                Spacer(Modifier.height(Rahal.space.sm))
                RahalButton(
                    onClick = onReject,
                    enabled = !busy,
                    tone = Tone.Danger,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(stringResource(R.string.md_amount_reject))
                }
            }
        },
        confirmButton = {},
    )
}
