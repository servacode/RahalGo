package com.rahalgo.customer.shop

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ModifierGroup
import com.rahalgo.shared.model.ModifierOption
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خياراتُ الصنف — تُختار قبل أن يدخل السلّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٢: «الأحجام… حركةٌ ذكيّةٌ بالأكل».)
 *
 * # ولماذا نافذةٌ لا شاشة
 *
 * **الاختيارُ فعلٌ من ثوانٍ في وسط التصفّح** — **وشاشةٌ كاملةٌ تُخرجه
 * من القسم** فيعود إليه وقد فقد موضعَ تمريره.
 *
 * # و«أضف» لا يعمل حتّى يكتمل الإلزاميّ
 *
 * **المحرّكُ يرفض الطلبَ كلَّه** إن نقص اختيارٌ إلزاميّ
 * (`orders/service.go`: `chosen < min` → `ErrBadItems`) — **لا الصنفَ
 * وحدَه.** **فزرٌّ يُضغط ثمّ يسقط الطلبُ عند الدفع أسوأُ من زرٍّ مطفأٍ
 * يقول لماذا.**
 *
 * # والسعرُ يتحرّك مع الاختيار
 *
 * **«أضف — ٣٥٠» تصير «أضف — ٤٤٠» حين يختار كبيراً** — **ورقمٌ ثابتٌ
 * بينما تتبدّل الخياراتُ يجعل الفارقَ مفاجأةً في السلّة.**
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ItemOptionsSheet(
    item: Item,
    api: CustomerApi,
    /** **ما اختاره والعدد** — «أضف» يحمل الكمّيّةَ كلَّها بضغطةٍ واحدة. */
    onAdd: (List<ModifierOption>, Int) -> Unit,
    onClose: () -> Unit,
    /**
     * **اسمُ المتجر بابُ صفحته** (قرارُ المالك ٢٠٢٦-١٠-١٨) — كما في البطاقة.
     * و`null` في صفحة المتجر نفسِه وحيث لا صفحة (`StoreLink.target`).
     */
    onStore: (() -> Unit)? = null,
    /** **عنوانُ الصور** — وبه تُرسم صورةُ الصنف عريضةً فوق النافذة، و`null` بلا صورة. */
    media: ((String?) -> String?)? = null,
) {
    var groups by remember(item.id) { mutableStateOf<List<ModifierGroup>?>(null) }
    var failed by remember(item.id) { mutableStateOf(false) }
    // **والعددُ هنا** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — من أراد ثلاثةً كان يفتح النافذةَ ثلاثاً.
    var qty by remember(item.id) { mutableStateOf(1) }

    // **المختارُ محفوظٌ بالمجموعة** — فمجموعةٌ تُبدَّل لا تمسّ أختَها.
    var chosen by remember(item.id) { mutableStateOf<Map<String, List<ModifierOption>>>(emptyMap()) }

    LaunchedEffect(item.id) {
        runCatching { api.itemDetail(item.id) }
            .onSuccess { d ->
                groups = d.modifiers
                // **والمجموعةُ الإلزاميّةُ بخيارٍ واحدٍ يُختار أوّلُ متوفّرٍ فيها** — مرآةُ الويب
                // (`items.tsx`): «الحجم: عادي» بلا ضغطةٍ إضافيّة.
                chosen = d.modifiers.mapNotNull { g ->
                    val first = g.options.firstOrNull { it.available }
                    if (g.minSelect > 0 && g.maxSelect == 1 && first != null) g.id to listOf(first) else null
                }.toMap()
            }
            .onFailure { failed = true }
    }

    val picked = chosen.values.flatten()
    val unit = item.price + picked.sumOf { it.priceDelta }

    // **وناقصٌ واحدٌ يكفي لإطفاء الزرّ.**
    val ready = groups?.all { g -> (chosen[g.id]?.size ?: 0) >= g.minSelect } == true

    // **ومجموعةٌ إلزاميّةٌ كلُّ خياراتها نافدةٌ تُطفئ الزرَّ للأبد** (تدقيقُ ٢٠٢٦-١٠-٢٨) —
    // **فيُقال لماذا** بدل زرٍّ مطفأٍ بلا سبب.
    val allOut = groups?.firstOrNull { g -> g.minSelect > 0 && g.options.none { it.available } }

    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Rahal.colors.canvas,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp)) {
            // **وصورةُ الصنف عريضةٌ فوق** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — يرى ما يختار له.
            if (media != null && (item.imageThumbUrl ?: item.imageUrl) != null) {
                // **والصورةُ كاملةً بأبعادها، والنقرُ يفتحها بملء الشاشة** (قرارُ المالك ٢٠٢٦-١٠-٢٩).
                com.rahalgo.ui.FullPhoto(
                    url = media(item.imageUrl ?: item.imageThumbUrl),
                    name = item.name,
                )
                Spacer(Modifier.height(12.dp))
            }
            Text(
                item.name,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleLarge,
            )
            // **واسمُ المتجر تحت اسم الصنف** — ملوَّنٌ حين يفتح صفحتَه، رماديٌّ حين لا.
            item.merchantName?.takeIf { it.isNotBlank() }?.let { name ->
                Spacer(Modifier.height(4.dp))
                Text(
                    text = name,
                    color = if (onStore != null) Rahal.colors.brand else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = if (onStore != null) {
                        Modifier
                            .clip(Rahal.shape.sm)
                            .clickable(
                                onClickLabel = stringResource(R.string.store_open_page),
                                onClick = onStore,
                            )
                            .padding(vertical = 2.dp)
                    } else {
                        Modifier
                    },
                )
            }
            Spacer(Modifier.height(16.dp))

            when {
                failed -> Text(
                    stringResource(R.string.options_failed),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                )

                groups == null -> Box(
                    Modifier.fillMaxWidth().height(120.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(color = Rahal.colors.brand)
                }

                else -> Column(
                    Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(18.dp),
                ) {
                    groups.orEmpty().forEach { g ->
                        GroupBlock(
                            group = g,
                            itemPrice = item.price,
                            itemPriceUsd = item.priceUsd,
                            chosen = chosen[g.id].orEmpty(),
                            onPick = { opt ->
                                val now = chosen[g.id].orEmpty()
                                val next = when {
                                    // **واحدٌ لا أكثر: الاختيارُ يستبدل** —
                                    // **ورفعُ القديم بضغطتين حيث تكفي واحدةٌ
                                    // يُقرأ عطباً.**
                                    g.maxSelect <= 1 -> listOf(opt)
                                    now.any { it.id == opt.id } -> now.filter { it.id != opt.id }
                                    now.size >= g.maxSelect -> now
                                    else -> now + opt
                                }
                                chosen = chosen + (g.id to next)
                            },
                        )
                    }
                }
            }

            Spacer(Modifier.height(20.dp))
            allOut?.let { g ->
                Text(
                    stringResource(R.string.options_all_out, g.name),
                    color = Rahal.colors.danger,
                    style = MaterialTheme.typography.bodyMedium,
                )
                Spacer(Modifier.height(8.dp))
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
            // **أداةُ الكمّيّة جنب الزرّ** — والسعرُ على الزرّ يتبعها.
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .border(1.dp, Rahal.colors.line, Rahal.shape.pill),
            ) {
                SheetQty(R.drawable.ic_minus, stringResource(R.string.cart_qty_less), enabled = qty > 1) { qty -= 1 }
                Text(qty.toString(), fontWeight = FontWeight.Bold, modifier = Modifier.padding(horizontal = 4.dp))
                SheetQty(com.rahalgo.ui.R.drawable.ic_plus, stringResource(R.string.cart_qty_more), enabled = qty < 99) { qty += 1 }
            }
            Spacer(Modifier.size(12.dp))
            RahalButton(
                onClick = { onAdd(picked, qty) },
                modifier = Modifier.weight(1f),
                enabled = ready,
            ) {
                // **وبالدولار لمتجرٍ اختاره، والليرةُ بجانبه** — هي ما يُدفع (٢٠٢٦-١٠-١٢).
                // **والدولارُ جمعُ دولار الصنف وإضافاته** — لا الليرةُ مقسومة، فيطابق
                // ما رآه على البطاقة والخيارات (ملاحظةُ المالك ٢٠٢٦-١٠-١٢).
                val usdUnit = item.priceUsd?.let { base ->
                    base + picked.sumOf { it.priceDeltaUsd ?: 0.0 }
                }
                Text(
                    stringResource(
                        R.string.options_add,
                        if (usdUnit != null) {
                            com.rahalgo.ui.dollars(usdUnit * qty) + " (≈ " + money(unit * qty) + ")"
                        } else money(unit * qty),
                    ),
                    maxLines = 2,
                )
            }
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

/** **زرُّ عددٍ في النافذة** — دائرةٌ ٤٠ بأيقونة، وباهتٌ حين لا يعمل. */
@Composable
private fun SheetQty(icon: Int, label: String, enabled: Boolean, onClick: () -> Unit) {
    Box(
        Modifier
            .size(40.dp)
            .clip(CircleShape)
            .clickable(enabled = enabled, onClickLabel = label, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        androidx.compose.material3.Icon(
            painter = androidx.compose.ui.res.painterResource(icon),
            contentDescription = label,
            tint = if (enabled) Rahal.colors.brand else Rahal.colors.line,
            modifier = Modifier.size(18.dp),
        )
    }
}

/** **مجموعةٌ واحدةٌ وخياراتُها** — وعنوانُها يقول أإلزاميّةٌ هي. */
@Composable
private fun GroupBlock(
    group: ModifierGroup,
    itemPrice: Long,
    itemPriceUsd: Double?,
    chosen: List<ModifierOption>,
    onPick: (ModifierOption) -> Unit,
) {
    Column {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                group.name,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.size(8.dp))
            // **«مطلوب» تُقال قبل أن يُضغط الزرُّ المطفأ** — **ومن رأى
            // زرّاً لا يعمل بلا سببٍ ظنّ التطبيقَ عطبان.**
            Text(
                stringResource(
                    if (group.minSelect > 0) R.string.options_required else R.string.act_optional,
                ),
                color = if (group.minSelect > 0) Rahal.colors.accent else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
        }
        if (group.minSelect > 0 && group.options.none { it.available }) {
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.options_all_out, group.name),
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(8.dp))
        group.options.forEach { o ->
            OptionRow(
                option = o,
                // **«حسب الوزن» بسعره كاملاً** لا «+فرق» (قرارُ المالك ٢٠٢٦-١٠-٢٨) —
                // «كيلو 6,000 ل.س» يُفهم، و«+5,400» لا يُفهم.
                fullPrice = if (group.priceMode == "weight") itemPrice + o.priceDelta else null,
                fullPriceUsd = if (group.priceMode == "weight") itemPriceUsd?.let { b -> o.priceDeltaUsd?.let { b + it } } else null,
                selected = chosen.any { it.id == o.id },
                onClick = { if (o.available) onPick(o) },
            )
        }
    }
}

/** **سطرُ خيارٍ** — واسمُه وفرقُ سعره، والمختارُ يُعرف بلونه. */
@Composable
private fun OptionRow(
    option: ModifierOption,
    selected: Boolean,
    onClick: () -> Unit,
    fullPrice: Long? = null,
    fullPriceUsd: Double? = null,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(enabled = option.available, onClick = onClick)
            .padding(vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(20.dp)
                .clip(CircleShape)
                .background(if (selected) Rahal.colors.brand else Rahal.colors.line),
        )
        Spacer(Modifier.size(12.dp))
        // **والنافدُ مشطوبٌ وبجنبه «غير متوفر»** — مرآةُ الويب (تدقيقُ ٢٠٢٦-١٠-٢٨)؛
        // كان رماديّاً وحدَه فيُظنّ معطّلاً بلا سبب.
        Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
            Text(
                option.name,
                color = if (option.available) Rahal.colors.ink else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyLarge,
                textDecoration = if (option.available) null else androidx.compose.ui.text.style.TextDecoration.LineThrough,
            )
            if (!option.available) {
                Spacer(Modifier.size(8.dp))
                Text(
                    stringResource(R.string.options_unavailable),
                    color = Rahal.colors.danger,
                    style = MaterialTheme.typography.labelSmall,
                )
            }
        }
        if (fullPrice != null) {
            Text(
                fullPriceUsd?.let { com.rahalgo.ui.dollars(it) } ?: money(fullPrice),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
        } else if (option.priceDelta != 0L) {
            // **والصفرُ لا يُكتب «+٠»** — **رقمٌ بلا معنًى يُقرأ سعراً.**
            Text(
                // **وبالدولار لمتجرٍ يعرض به** (٢٠٢٦-١٠-١٢).
                "+" + (option.priceDeltaUsd?.let { com.rahalgo.ui.dollars(it) } ?: money(option.priceDelta)),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}
