package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepCommissionApi
import com.rahalgo.shared.rep.RepMerchantHead
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.apiError
import kotlinx.coroutines.launch
import java.math.BigDecimal

/** «7.5» و«10» — بلا أصفارٍ زائدة. */
internal fun pctText(v: Double): String =
    BigDecimal.valueOf(v).stripTrailingZeros().toPlainString()

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نسبةُ عمولة متجر العميل — يضبطها المندوب** (قرارُ المالك ٢٠٢٦-١١-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بين ال0 وال10 بالمية» — **والحدّان من المحرّك** يُعرضان تحت الخانة، وتسري فوراً
 * بلا موافقة المكتب. **وما يراه نسبةُ المتجر وحدَها** لا ربحَ المنصّة.
 */
@Composable
fun StoreCommissionRow(merchantId: String, head: RepMerchantHead) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var current by remember(merchantId, head.storeCommissionPercent) { mutableStateOf(head.storeCommissionPercent) }
    var lo by remember(merchantId, head.storeCommissionMin) { mutableStateOf(head.storeCommissionMin) }
    var hi by remember(merchantId, head.storeCommissionMax) { mutableStateOf(head.storeCommissionMax) }
    var editing by remember(merchantId) { mutableStateOf(false) }
    var text by remember(merchantId, current) { mutableStateOf(pctText(current)) }
    var saving by remember { mutableStateOf(false) }

    Spacer(Modifier.height(8.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.cd_commission_title) + ": " + stringResource(R.string.cd_commission_value, pctText(current)),
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        if (!editing) {
            RahalTextButton(onClick = { editing = true }) {
                Text(stringResource(R.string.cd_commission_edit), color = Rahal.colors.brand)
            }
        }
    }
    if (!editing) return
    Column {
        Text(
            stringResource(R.string.cd_commission_range, pctText(lo), pctText(hi)),
            style = MaterialTheme.typography.bodySmall,
            color = Rahal.colors.inkMuted,
        )
        Spacer(Modifier.height(6.dp))
        val normalized = text.replace('٫', '.').replace(',', '.')
        val value = normalized.toBigDecimalOrNull()
        val valid = value != null && value.scale() <= 2 &&
            value >= BigDecimal.valueOf(lo) && value <= BigDecimal.valueOf(hi)
        Row(verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = text,
                onValueChange = { v -> text = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(6) },
                label = { Text(stringResource(R.string.cd_commission_field)) },
                singleLine = true,
                isError = text.isNotBlank() && !valid,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            RahalButton(
                onClick = {
                    saving = true
                    scope.launch {
                        try {
                            val got = RepCommissionApi(AppCore.get().api).setStoreCommission(merchantId, normalized)
                            current = got.percent
                            lo = got.min
                            hi = got.max
                            editing = false
                            Flash.ok(context.getString(R.string.cd_commission_saved))
                        } catch (e: Exception) {
                            Flash.fail(apiError(context, e))
                        }
                        saving = false
                    }
                },
                enabled = valid && !saving,
            ) { Text(stringResource(com.rahalgo.ui.R.string.act_save)) }
        }
        RahalTextButton(onClick = { editing = false; text = pctText(current) }) {
            Text(stringResource(R.string.cd_commission_cancel), color = Rahal.colors.inkMuted)
        }
    }
}
