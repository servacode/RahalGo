package com.rahalgo.driver.trip

import com.rahalgo.ui.RahalDivider
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Icon
import androidx.compose.ui.res.painterResource
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
import androidx.compose.material3.FilterChip
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
import com.rahalgo.ui.amountText

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
    // **ل.س أو $** (قرارُ المالك ٢٠٢٦-١١-١٢) — والدولارُ يُكتب بكسوره: «2.5».
    var addUsd by remember { mutableStateOf(false) }
    val lines = editableLines(order)
    // **طلب بأكثر من متجر ⇒ الكابتن يختار متجر الصنف** (الافتراضي: الأول)، وغيره ما بيظهر اختيار.
    val stores = editStoreChoices(order)
    var addStore by remember(order.id) { mutableStateOf(stores.firstOrNull()?.merchantId) }
    val focus = LocalFocusManager.current
    val q = addQty.toIntOrNull() ?: 0
    val p = if (addUsd) {
        addPrice.toDoubleOrNull()?.let { Math.round(it * 100) } ?: 0L
    } else {
        addPrice.toLongOrNull() ?: 0L
    }
    val addStoreRow = stores.firstOrNull { it.merchantId == addStore }
    // **والمكتوبُ في الخانات يُحسب ويُرسَل ولو لم يُضغط «+ ضيف الصنف».**
    val effective = draft.withPendingAdd(addName, q, p, addStoreRow?.merchantId, addStoreRow?.name.orEmpty(), addUsd)
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.edit_req_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(
                Modifier.fillMaxWidth().heightIn(max = 520.dp).verticalScroll(rememberScrollState()),
            ) {
                Text(stringResource(R.string.edit_req_hint), color = Rahal.colors.inkMuted, style = Rahal.type.body)
                Spacer(Modifier.height(Rahal.space.sm))
                for (it in lines) {
                    val d = draft.lines[it.id]
                    Spacer(Modifier.height(Rahal.space.sm))
                    // **الاسمُ والكمّيةُ بسطر، والسعرُ تحته** (قرارُ المالك ٢٠٢٦-١١-١٢).
                    Text("${it.name} × ${it.qty}", color = Rahal.colors.ink, style = Rahal.type.cardTitle)
                    it.unitPrice?.let { up ->
                        Text(
                            stringResource(R.string.edit_req_line_price, money(up), money(up * it.qty)),
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.meta,
                        )
                    }
                    Spacer(Modifier.height(Rahal.space.xs))
                    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm), verticalAlignment = Alignment.CenterVertically) {
                        for (a in EditLineAction.entries) {
                            FilterChip(
                                selected = d?.action == a,
                                onClick = {
                                    val next = if (d?.action == a) null else EditLineDraft(a, d?.remove ?: it.qty)
                                    draft = draft.copy(
                                        lines = if (next == null) draft.lines - it.id else draft.lines + (it.id to next),
                                    )
                                },
                                // **وأيقونةٌ لكلّ سبب** (قرارُ المالك ٢٠٢٦-١١-١٢: «نخليه احترافي اكثر») —
                                // صندوقٌ مشطوبٌ لما لم يجده بالمتجر، وشخصٌ لما رفضه الزبون.
                                leadingIcon = {
                                    Icon(
                                        painter = painterResource(
                                            if (a == EditLineAction.MISSING) com.rahalgo.ui.R.drawable.ic_block else com.rahalgo.ui.R.drawable.ic_user,
                                        ),
                                        contentDescription = null,
                                        modifier = Modifier.size(Rahal.icon.sm),
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
                        QtyStepper(
                            label = stringResource(R.string.edit_req_how_many),
                            value = "${d.remove}",
                            onMinus = {
                                draft = draft.copy(lines = draft.lines + (it.id to d.copy(remove = (d.remove - 1).coerceAtLeast(1))))
                            },
                            onPlus = {
                                draft = draft.copy(lines = draft.lines + (it.id to d.copy(remove = (d.remove + 1).coerceAtMost(it.qty))))
                            },
                        )
                    }
                    Spacer(Modifier.height(Rahal.space.sm))
                    RahalDivider()
                }
                Spacer(Modifier.height(Rahal.space.sm))
                for ((i, a) in draft.adds.withIndex()) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            painter = painterResource(com.rahalgo.ui.R.drawable.ic_plus),
                            contentDescription = null,
                            tint = Rahal.colors.success,
                            modifier = Modifier.size(Rahal.icon.sm),
                        )
                        Spacer(Modifier.width(Rahal.space.sm))
                        Column(Modifier.weight(1f)) {
                            Text("${a.name} × ${a.qty}", style = Rahal.type.bodyStrong, color = Rahal.colors.ink)
                            val unit = if (a.usd) amountText(a.price, "USD") else money(a.price)
                            val sum = if (a.usd) amountText(a.price * a.qty, "USD") else money(a.price * a.qty)
                            Text(
                                stringResource(R.string.edit_req_line_price, unit, sum) +
                                    (if (a.merchantName.isBlank()) "" else " · " + a.merchantName),
                                style = Rahal.type.meta,
                                color = Rahal.colors.inkMuted,
                            )
                        }
                        RahalTextButton(onClick = {
                            draft = draft.copy(adds = draft.adds.filterIndexed { j, _ -> j != i })
                        }, tone = com.rahalgo.ui.Tone.Danger) { Text(stringResource(R.string.edit_req_remove)) }
                    }
                }
                Spacer(Modifier.height(Rahal.space.sm))
                Text(stringResource(R.string.edit_req_add_title), style = Rahal.type.sectionTitle, color = Rahal.colors.ink)
                Spacer(Modifier.height(Rahal.space.xs))
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
                    Spacer(Modifier.height(Rahal.space.sm))
                    Text(stringResource(R.string.edit_req_add_store), color = Rahal.colors.inkMuted, style = Rahal.type.label)
                    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
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
                QtyStepper(
                    label = stringResource(R.string.edit_req_add_qty),
                    value = addQty.ifBlank { "1" },
                    onMinus = { addQty = (q - 1).coerceAtLeast(1).toString() },
                    onPlus = { addQty = (q + 1).coerceIn(1, 50).toString() },
                )
                // **والسعرُ بعرضٍ كامل** — فلا تلتفّ التسميةُ على سطرين.
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    OutlinedTextField(
                        value = addPrice,
                        onValueChange = { v ->
                            addPrice = if (addUsd) {
                                // **رقمٌ وفاصلةٌ واحدةٌ ومنزلتان** — «2.5» لا «2..5».
                                val clean = v.filter { it.isDigit() || it == '.' }
                                val dot = clean.indexOf('.')
                                (if (dot < 0) clean else clean.substring(0, dot + 1) + clean.substring(dot + 1).replace(".", "").take(2)).take(9)
                            } else {
                                v.filter { it.isDigit() }.take(9)
                            }
                        },
                        label = {
                            Text(
                                stringResource(if (addUsd) R.string.edit_req_add_price_usd else R.string.edit_req_add_price),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(
                            keyboardType = if (addUsd) KeyboardType.Decimal else KeyboardType.Number,
                            imeAction = ImeAction.Done,
                        ),
                        keyboardActions = KeyboardActions(onDone = { focus.clearFocus() }),
                        modifier = Modifier.weight(1f),
                    )
                    FilterChip(
                        selected = !addUsd,
                        onClick = { if (addUsd) { addUsd = false; addPrice = "" } },
                        label = { Text(stringResource(R.string.edit_req_cur_syp)) },
                    )
                    FilterChip(
                        selected = addUsd,
                        onClick = { if (!addUsd) { addUsd = true; addPrice = "" } },
                        label = { Text(stringResource(R.string.edit_req_cur_usd)) },
                    )
                }
                RahalTextButton(
                    onClick = {
                        draft = effective
                        addName = ""
                        addQty = "1"
                        addPrice = ""
                        addUsd = false
                    },
                    enabled = addName.isNotBlank() && q in 1..50 && p > 0,
                ) { Text(stringResource(R.string.edit_req_add_button)) }
                Spacer(Modifier.height(Rahal.space.sm))
                // **المجموعُ التقريبيُّ في بطاقةٍ ملوّنة** — آخرُ ما يُقرأ قبل الإرسال.
                com.rahalgo.ui.RahalCard(
                    variant = com.rahalgo.ui.CardVariant.Tinted,
                    tone = com.rahalgo.ui.StatusTone.Info,
                ) {
                    // **المجموعُ قبل وبعد بسطرين** (٢٠٢٦-١١-١٢) — والجديدُ أعرض، و«تقريباً» لأنّ المكتب يوافق.
                    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.xxs)) {
                        com.rahalgo.ui.InvoiceRow(stringResource(R.string.edit_req_total_before), money(order.total))
                        com.rahalgo.ui.InvoiceRow(
                            stringResource(R.string.edit_req_total_after),
                            money(editDraftTotal(order, effective)),
                            strong = true,
                        )
                        editDraftUsd(effective).takeIf { it > 0 }?.let { usd ->
                            Text(
                                stringResource(R.string.edit_req_total_usd, amountText(usd, "USD")),
                                style = Rahal.type.bodyStrong,
                                color = Rahal.colors.ink,
                            )
                        }
                        Text(stringResource(R.string.edit_req_total_note), style = Rahal.type.meta, color = Rahal.colors.inkMuted)
                    }
                }
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Text(error, color = Rahal.colors.danger, style = Rahal.type.caption)
                }
            }
        },
        confirmButton = {
            RahalButton(
                onClick = { onSend(effective) },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy && editDraftValid(order, effective),
            ) { Text(stringResource(R.string.edit_req_send), style = Rahal.type.button) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.act_back))
            }
        },
    )
}

/**
 * **عدّادُ كمّيّةٍ بزرَّين مدوَّرين** (إعادةُ التصميم ٢٠٢٦-١١-١١) — «−» و«+» بلمسةٍ ٤٨ والرقمُ بينهما،
 * **بدلَ حرفين صغيرين في أزرارٍ نصّيّة.** والفعلُ نفسُه كما كان.
 */
@Composable
private fun QtyStepper(label: String, value: String, onMinus: () -> Unit, onPlus: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = Rahal.space.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, color = Rahal.colors.inkMuted, style = Rahal.type.label, modifier = Modifier.weight(1f))
        com.rahalgo.ui.IconAction(
            icon = com.rahalgo.ui.R.drawable.ic_minus,
            description = "−",
            onClick = onMinus,
        )
        Text(
            value,
            style = Rahal.type.sectionTitle,
            color = Rahal.colors.ink,
            textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            modifier = Modifier.widthIn(min = Rahal.size.touch),
        )
        com.rahalgo.ui.IconAction(
            icon = com.rahalgo.ui.R.drawable.ic_plus,
            description = "+",
            onClick = onPlus,
        )
    }
}
