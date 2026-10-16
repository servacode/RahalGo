package com.rahalgo.customer.live

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«شارك موقعك المباشر» — نافذةٌ فوق أيّ شاشة** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يركّبها الغلاف** (`MainActivity`) كنافذة تأكيد السعر — فمن كان في السوق حين
 * صار طلبُه «بالطريق» يُسأل وهو فيه. **وجوابان لا ثالث**: «شارك موقعي» أو
 * «لا، العنوان صحيح» — **وكلاهما يُذكر فلا يُسأل عن الطلب نفسِه ثانيةً.**
 *
 * **والرجوعُ يُقرأ «العنوان صحيح»** — نافذةٌ تعود كلّما أُغلقت تُزعج من قرّر.
 */
@Composable
fun LiveShareDialog(
    onShare: () -> Unit,
    onDecline: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDecline,
        title = {
            Text(stringResource(R.string.live_ask_title), fontWeight = FontWeight.Bold)
        },
        text = {
            Text(
                text = stringResource(R.string.live_ask_body),
                color = Rahal.colors.ink,
                style = MaterialTheme.typography.bodyMedium,
            )
        },
        confirmButton = {
            RahalTextButton(onClick = onShare) {
                Text(stringResource(R.string.live_ask_share), fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDecline) {
                Text(stringResource(R.string.live_ask_no))
            }
        },
    )
}
