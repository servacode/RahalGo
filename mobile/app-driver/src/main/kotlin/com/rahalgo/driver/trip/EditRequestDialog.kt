package com.rahalgo.driver.trip

import com.rahalgo.ui.RahalDivider
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.clickable
import androidx.compose.foundation.background
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
 * **نافذة «طلب تعديل»** (قرار المالك ٢٠٢٦-١١-١٠، وشكلُها ٢٠٢٦-١١-١٢) — كل صنف بسطره وأيقونتان
 * بآخره: **حذف** (ناقص بالمتجر) و**إلغاء** (الزبون لغاه). وتحتها «إضافة صنف»: الاسمُ والكمّيةُ
 * بسطر، والسعرُ بالليرة أو الدولار بسطر. **والمجموعُ سطرٌ واحد** — بلا شرحٍ زائد.
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
    // **ل.س أو $** — والدولارُ يُكتب بكسوره: «2.5».
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
    // **والمكتوبُ في الخانات يُحسب ويُرسَل ولو لم يُضغط «+ أضف الصنف».**
    val effective = draft.withPendingAdd(addName, q, p, addStoreRow?.merchantId, addStoreRow?.name.orEmpty(), addUsd)
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.edit_req_title), fontWeight = FontWeight.Bold) },
        text = {
            Column(
                Modifier.fillMaxWidth().heightIn(max = 520.dp).verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                for (it in lines) {
                    val d = draft.lines[it.id]
                    fun toggle(a: EditLineAction) {
                        val next = if (d?.action == a) null else EditLineDraft(a, d?.remove ?: it.qty)
                        draft = draft.copy(lines = if (next == null) draft.lines - it.id else draft.lines + (it.id to next))
                    }
                    // **الصنفُ وأيقونتاه بسطرٍ واحد** — والمختارُ يُلوَّن، ولمسةٌ ثانيةٌ تُلغيه.
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                "${it.name} × ${it.qty}",
                                color = Rahal.colors.ink,
                                style = Rahal.type.bodyStrong,
                                textDecoration = if (d != null) androidx.compose.ui.text.style.TextDecoration.LineThrough else null,
                            )
                            it.unitPrice?.let { up ->
                                Text(money(up * it.qty), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
                            }
                        }
                        LineToggle(
                            icon = com.rahalgo.ui.R.drawable.ic_trash,
                            description = stringResource(R.string.edit_req_missing),
                            selected = d?.action == EditLineAction.MISSING,
                            tint = Rahal.colors.danger,
                        ) { toggle(EditLineAction.MISSING) }
                        Spacer(Modifier.width(Rahal.space.xs))
                        LineToggle(
                            icon = R.drawable.ic_close_circle,
                            description = stringResource(R.string.edit_req_cancelled),
                            selected = d?.action == EditLineAction.CANCELLED,
                            tint = Rahal.colors.accent,
                        ) { toggle(EditLineAction.CANCELLED) }
                    }
                    // **ومعنى المختار بكلمة** — فلا تُحزر الأيقونة.
                    if (d != null) {
                        Text(
                            stringResource(if (d.action == EditLineAction.MISSING) R.string.edit_req_missing else R.string.edit_req_cancelled),
                            color = if (d.action == EditLineAction.MISSING) Rahal.colors.danger else Rahal.colors.accent,
                            style = Rahal.type.meta,
                            fontWeight = FontWeight.Bold,
                        )
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
                    RahalDivider()
                }
                // **المضافُ بسطره** — الاسمُ والكمّيةُ، وسعرُه تحته، وزرُّ شيله بآخره.
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
                            Text(
                                (if (a.usd) amountText(a.price * a.qty, "USD") else money(a.price * a.qty)) +
                                    (if (a.merchantName.isBlank()) "" else " · " + a.merchantName),
                                style = Rahal.type.meta,
                                color = Rahal.colors.inkMuted,
                            )
                        }
                        com.rahalgo.ui.IconAction(
                            icon = com.rahalgo.ui.R.drawable.ic_trash,
                            description = stringResource(R.string.edit_req_remove),
                            tint = Rahal.colors.danger,
                            onClick = { draft = draft.copy(adds = draft.adds.filterIndexed { j, _ -> j != i }) },
                        )
                    }
                }
                Text(stringResource(R.string.edit_req_add_title), style = Rahal.type.sectionTitle, color = Rahal.colors.ink)
                // **الاسمُ والكمّيةُ بسطرٍ واحد** (قرارُ المالك ٢٠٢٦-١١-١٢).
                Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        value = addName,
                        onValueChange = { addName = it.take(120) },
                        label = { Text(stringResource(R.string.edit_req_add_name), maxLines = 1) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                        keyboardActions = KeyboardActions(onNext = { focus.moveFocus(FocusDirection.Down) }),
                        modifier = Modifier.weight(1f),
                    )
                    Spacer(Modifier.width(Rahal.space.xs))
                    com.rahalgo.ui.IconAction(
                        icon = com.rahalgo.ui.R.drawable.ic_minus,
                        description = "−",
                        onClick = { addQty = (q - 1).coerceAtLeast(1).toString() },
                    )
                    Text(
                        addQty.ifBlank { "1" },
                        style = Rahal.type.sectionTitle,
                        color = Rahal.colors.ink,
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                        modifier = Modifier.widthIn(min = 28.dp),
                    )
                    com.rahalgo.ui.IconAction(
                        icon = com.rahalgo.ui.R.drawable.ic_plus,
                        description = "+",
                        onClick = { addQty = (q + 1).coerceIn(1, 50).toString() },
                    )
                }
                if (stores.isNotEmpty()) {
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
                // **والسعرُ وعملتُه بسطر.**
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    OutlinedTextField(
                        value = addPrice,
                        onValueChange = { v ->
                            addPrice = if (addUsd) {
                                // **رقمٌ وفاصلةٌ واحدةٌ ومنزلتان** — «2.5» لا «2..5».
                                val clean = v.filter { c -> c.isDigit() || c == '.' }
                                val dot = clean.indexOf('.')
                                (if (dot < 0) clean else clean.substring(0, dot + 1) + clean.substring(dot + 1).replace(".", "").take(2)).take(9)
                            } else {
                                v.filter { c -> c.isDigit() }.take(9)
                            }
                        },
                        label = { Text(stringResource(R.string.edit_req_price_label), maxLines = 1, overflow = TextOverflow.Ellipsis) },
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
                // **المجموعُ سطرٌ واحدٌ واضح** — القديمُ ← الجديد (والدولارُ المضافُ بجانبه).
                com.rahalgo.ui.RahalCard(
                    variant = com.rahalgo.ui.CardVariant.Tinted,
                    tone = com.rahalgo.ui.StatusTone.Info,
                ) {
                    val usd = editDraftUsd(effective)
                    com.rahalgo.ui.InvoiceRow(
                        stringResource(R.string.edit_req_total_label),
                        money(order.total) + " ← " + money(editDraftTotal(order, effective)) +
                            (if (usd > 0) " + " + amountText(usd, "USD") else ""),
                        strong = true,
                    )
                }
                if (error.isNotEmpty()) {
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

/** **أيقونةُ فعلٍ على سطر الصنف** — قرصٌ بلون معناه إن اختير، وإطارٌ باهتٌ إن لم يُختر. */
@Composable
private fun LineToggle(icon: Int, description: String, selected: Boolean, tint: androidx.compose.ui.graphics.Color, onClick: () -> Unit) {
    androidx.compose.foundation.layout.Box(
        Modifier
            .size(Rahal.size.touch)
            .clip(androidx.compose.foundation.shape.CircleShape)
            .background(if (selected) tint else tint.copy(alpha = 0.10f))
            .clickable(onClickLabel = description, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = description,
            tint = if (selected) Rahal.colors.onPanel else tint,
            modifier = Modifier.size(Rahal.icon.md),
        )
    }
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
