package com.rahalgo.driver.home

import com.rahalgo.ui.DialogDismissButton
import com.rahalgo.ui.DialogConfirmButton
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverTerms
import com.rahalgo.shared.model.parseBlocks
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ شروط العمل — قبل فتح الدوام** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **البندُ الجديدُ أوّلاً في بطاقةٍ ملوّنة** — «أخذ زبون أو رقمه للتعامل برّا المنصّة
 * يعني إيقاف الحساب» — **ثمّ الشروطُ كاملةً بكتلها** (`parseBlocks`، القارئُ نفسُه الذي
 * في صفحة الشروط)، **وتحتها «قرأت الشروط وموافق».**
 *
 * **والنصُّ من المحرّك لا من التطبيق** — وثيقةٌ في نسختين تفترقان.
 */
@Composable
fun TermsSheet(
    terms: DriverTerms,
    busy: Boolean,
    error: String,
    onAccept: () -> Unit,
    onClose: () -> Unit,
) {
    val blocks = parseBlocks(text = terms.text, name = terms.name, phone = terms.phone)
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = { if (!busy) onClose() },
        title = { Text(stringResource(R.string.terms_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(
                Modifier
                    .heightIn(max = 460.dp)
                    .verticalScroll(rememberScrollState()),
            ) {
                Text(stringResource(R.string.terms_body), style = MaterialTheme.typography.bodyMedium)
                if (terms.clause.isNotBlank()) {
                    Spacer(Modifier.height(12.dp))
                    Column(
                        Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.md)
                            .background(Rahal.colors.brand.copy(alpha = 0.10f))
                            .padding(14.dp),
                    ) {
                        Text(
                            text = stringResource(R.string.terms_new_clause),
                            color = Rahal.colors.brand,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelLarge,
                        )
                        Spacer(Modifier.height(4.dp))
                        Text(terms.clause, fontWeight = FontWeight.Bold)
                    }
                }
                blocks.forEach { b ->
                    Spacer(Modifier.height(12.dp))
                    if (b.head.isNotEmpty()) {
                        Text(b.head, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        Spacer(Modifier.height(4.dp))
                    }
                    b.body.forEach { p ->
                        Text(p, style = MaterialTheme.typography.bodyMedium)
                    }
                }
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(12.dp))
                    Text(error, color = Rahal.colors.danger)
                }
            }
        },
        confirmButton = {
            DialogConfirmButton(onClick = onAccept, enabled = !busy) {
                Text(
                    stringResource(R.string.terms_accept),
                    fontWeight = FontWeight.Bold,
                )
            }
        },
        dismissButton = {
            DialogDismissButton(onClick = onClose, enabled = !busy) {
                Text(stringResource(R.string.terms_close))
            }
        },
    )
}
