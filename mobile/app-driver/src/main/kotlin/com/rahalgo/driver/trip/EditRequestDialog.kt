package com.rahalgo.driver.trip

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.money

/**
 * **نافذة «طلب تعديل»** (قرار المالك ٢٠٢٦-١١-١٠) — كل سطر: «ناقص» أو «الزبون لغاه» (وكم قطعة)،
 * و«انزاد صنف» باسمه وكمّيته وسعره، وملاحظة. **والمجموع التقريبي تحت**، والمكتب بيوافق.
 */
@Composable
internal fun EditRequestDialog(
    order: DriverOrder,
    error: String,
    busy: Boolean,
    onSend: (EditDraft) -> Unit,
    onDismiss: () -> Unit,
) {
    var draft by remember(order.id) { mutableStateOf(EditDraft()) }
    var addName by remember { mutableStateOf("") }
    var addQty by remember { mutableStateOf("1") }
    var addPrice by remember { mutableStateOf("") }
    val lines = editableLines(order)
    // **طلب بأكثر من متجر ⇒ الكابتن يختار متجر الصنف** (الافتراضي: الأول)، وغيره ما بيظهر اختيار.
    val stores = editStoreChoices(order)
    var addStore by remember(order.id) { mutableStateOf(stores.firstOrNull()?.merchantId) }
    val focus = LocalFocusManager.current
    val q = addQty.toIntOrNull() ?: 0
    val p = addPrice.toLongOrNull() ?: 0L
    val addStoreRow = stores.firstOrNull { it.merchantId == addStore }
    // **والمكتوبُ في الخانات يُحسب ويُرسَل ولو لم يُضغط «+ ضيف الصنف».**
    val effective = draft.withPendingAdd(addName, q, p, addStoreRow?.merchantId, addStoreRow?.name.orEmpty())
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.edit_req_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(
                Modifier.fillMaxWidth().heightIn(max = 520.dp).verticalScroll(rememberScrollState()),
            ) {
                Text(stringResource(R.string.edit_req_hint), color = Rahal.colors.inkMuted)
                Spacer(Modifier.height(8.dp))
                for (it in lines) {
                    val d = draft.lines[it.id]
                    Text("${it.name} ×${it.qty}", fontWeight = FontWeight.Bold, color = Rahal.colors.ink)
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                        for (a in EditLineAction.entries) {
                            FilterChip(
                                selected = d?.action == a,
                                onClick = {
                                    val next = if (d?.action == a) null else EditLineDraft(a, d?.remove ?: it.qty)
                                    draft = draft.copy(
                                        lines = if (next == null) draft.lines - it.id else draft.lines + (it.id to next),
                                    )
                                },
                                label = {
                                    Text(
                                        stringResource(
                                            if (a == EditLineAction.MISSING) R.string.edit_req_missing else R.string.edit_req_cancelled,
                                        ),
                                    )
                                },
                            )
                        }
                    }
                    // **وكم قطعة** — لسطر فيه أكتر من وحدة.
                    if (d != null && it.qty > 1) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(stringResource(R.string.edit_req_how_many), color = Rahal.colors.inkMuted)
                            Spacer(Modifier.width(8.dp))
                            RahalTextButton(onClick = {
                                draft = draft.copy(lines = draft.lines + (it.id to d.copy(remove = (d.remove - 1).coerceAtLeast(1))))
                            }) { Text("−", fontWeight = FontWeight.Bold) }
                            Text("${d.remove}", fontWeight = FontWeight.Bold)
                            RahalTextButton(onClick = {
                                draft = draft.copy(lines = draft.lines + (it.id to d.copy(remove = (d.remove + 1).coerceAtMost(it.qty))))
                            }) { Text("+", fontWeight = FontWeight.Bold) }
                        }
                    }
                    HorizontalDivider(Modifier.fillMaxWidth())
                }
                Spacer(Modifier.height(8.dp))
                for ((i, a) in draft.adds.withIndex()) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            if (a.merchantName.isBlank()) {
                                stringResource(R.string.edit_req_added_line, a.name, a.qty, money(a.price))
                            } else {
                                stringResource(R.string.edit_req_added_line_store, a.name, a.qty, money(a.price), a.merchantName)
                            },
                            modifier = Modifier.weight(1f),
                        )
                        RahalTextButton(onClick = {
                            draft = draft.copy(adds = draft.adds.filterIndexed { j, _ -> j != i })
                        }) { Text(stringResource(R.string.edit_req_remove)) }
                    }
                }
                Text(stringResource(R.string.edit_req_add_title), fontWeight = FontWeight.Bold)
                OutlinedTextField(
                    value = addName,
                    onValueChange = { addName = it.take(120) },
                    label = { Text(stringResource(R.string.edit_req_add_name)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                    keyboardActions = KeyboardActions(onNext = { focus.moveFocus(FocusDirection.Down) }),
                    modifier = Modifier.fillMaxWidth(),
                )
                if (stores.isNotEmpty()) {
                    Text(stringResource(R.string.edit_req_add_store), color = Rahal.colors.inkMuted)
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        for (st in stores) {
                            FilterChip(
                                selected = addStore == st.merchantId,
                                onClick = { addStore = st.merchantId },
                                label = { Text(st.name) },
                            )
                        }
                    }
                }
                // **الكمّيةُ بزرَّي − و+** — لا كتابةَ لرقمٍ صغير (المالك ٢٠٢٦-١١-١٠).
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(stringResource(R.string.edit_req_add_qty), color = Rahal.colors.inkMuted)
                    Spacer(Modifier.width(8.dp))
                    RahalTextButton(onClick = { addQty = (q - 1).coerceAtLeast(1).toString() }) {
                        Text("−", fontWeight = FontWeight.Bold)
                    }
                    Text(
                        addQty.ifBlank { "1" },
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                    )
                    RahalTextButton(onClick = { addQty = (q + 1).coerceIn(1, 50).toString() }) {
                        Text("+", fontWeight = FontWeight.Bold)
                    }
                }
                // **والسعرُ بعرضٍ كامل** — فلا تلتفّ التسميةُ على سطرين.
                OutlinedTextField(
                    value = addPrice,
                    onValueChange = { v -> addPrice = v.filter { it.isDigit() }.take(9) },
                    label = {
                        Text(
                            stringResource(R.string.edit_req_add_price),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { focus.clearFocus() }),
                    modifier = Modifier.fillMaxWidth(),
                )
                RahalTextButton(
                    onClick = {
                        draft = effective
                        addName = ""
                        addQty = "1"
                        addPrice = ""
                    },
                    enabled = addName.isNotBlank() && q in 1..50 && p > 0,
                ) { Text(stringResource(R.string.edit_req_add_button)) }
                Spacer(Modifier.height(8.dp))
                Text(
                    stringResource(R.string.edit_req_total, money(order.total), money(editDraftTotal(order, effective))),
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.ink,
                )
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(8.dp))
                    Text(error, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
                }
            }
        },
        confirmButton = {
            RahalButton(
                onClick = { onSend(effective) },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy && editDraftValid(order, effective),
            ) { Text(stringResource(R.string.edit_req_send), fontWeight = FontWeight.Bold) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.act_back))
            }
        },
    )
}
