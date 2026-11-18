package com.rahalgo.driver.trip

import androidx.compose.foundation.layout.Arrangement
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
import com.rahalgo.driver.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.money

/**
 * **نافذة «اطلب تعديل الأجرة»** (قرار المالك ٢٠٢٦-١١-١٨) — طلب الواتساب: الأجرة الجديدة والسبب،
 * **والإدارة توافق** بمعاينة المحرّك، وبعدها بيتعدّل المبلغ اللي بيقبضه الكابتن من الزبون.
 */
@Composable
internal fun FeeRequestDialog(
    currentFee: Long,
    error: String,
    busy: Boolean,
    onSend: (Long, String) -> Unit,
    onDismiss: () -> Unit,
) {
    var fee by remember { mutableStateOf("") }
    var reason by remember { mutableStateOf("") }
    val n = fee.trim().toLongOrNull()
    val valid = n != null && n > 0 && n != currentFee && reason.isNotBlank()
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.fee_req_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                Text(stringResource(R.string.fee_req_hint, money(currentFee)))
                OutlinedTextField(
                    value = fee,
                    onValueChange = { v -> fee = v.filter { it.isDigit() }.take(9) },
                    label = { Text(stringResource(R.string.fee_req_fee)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = reason,
                    onValueChange = { reason = it.take(500) },
                    label = { Text(stringResource(R.string.fee_req_reason)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                if (error.isNotEmpty()) Text(error, color = Rahal.colors.danger)
            }
        },
        confirmButton = {
            RahalButton(
                onClick = { if (n != null) onSend(n, reason.trim()) },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy && valid,
            ) { Text(stringResource(R.string.fee_req_send), style = Rahal.type.button) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.act_back))
            }
        },
    )
}
