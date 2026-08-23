package com.rahalgo.customer.shop

import androidx.compose.foundation.background
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
    onAdd: (List<ModifierOption>) -> Unit,
    onClose: () -> Unit,
) {
    var groups by remember(item.id) { mutableStateOf<List<ModifierGroup>?>(null) }
    var failed by remember(item.id) { mutableStateOf(false) }

    // **المختارُ محفوظٌ بالمجموعة** — فمجموعةٌ تُبدَّل لا تمسّ أختَها.
    var chosen by remember(item.id) { mutableStateOf<Map<String, List<ModifierOption>>>(emptyMap()) }

    LaunchedEffect(item.id) {
        runCatching { api.itemDetail(item.id) }
            .onSuccess { groups = it.modifiers }
            .onFailure { failed = true }
    }

    val picked = chosen.values.flatten()
    val unit = item.price + picked.sumOf { it.priceDelta }

    // **وناقصٌ واحدٌ يكفي لإطفاء الزرّ.**
    val ready = groups?.all { g -> (chosen[g.id]?.size ?: 0) >= g.minSelect } == true

    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Rahal.colors.canvas,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp)) {
            Text(
                item.name,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleLarge,
            )
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
            RahalButton(
                onClick = { onAdd(picked) },
                modifier = Modifier.fillMaxWidth(),
                enabled = ready,
            ) {
                Text(stringResource(R.string.options_add, money(unit)))
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

/** **مجموعةٌ واحدةٌ وخياراتُها** — وعنوانُها يقول أإلزاميّةٌ هي. */
@Composable
private fun GroupBlock(
    group: ModifierGroup,
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
                    if (group.minSelect > 0) R.string.options_required else R.string.options_optional,
                ),
                color = if (group.minSelect > 0) Rahal.colors.accent else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
        }
        Spacer(Modifier.height(8.dp))
        group.options.forEach { o ->
            OptionRow(
                option = o,
                selected = chosen.any { it.id == o.id },
                onClick = { if (o.available) onPick(o) },
            )
        }
    }
}

/** **سطرُ خيارٍ** — واسمُه وفرقُ سعره، والمختارُ يُعرف بلونه. */
@Composable
private fun OptionRow(option: ModifierOption, selected: Boolean, onClick: () -> Unit) {
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
        Text(
            option.name,
            modifier = Modifier.weight(1f),
            color = if (option.available) Rahal.colors.ink else Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyLarge,
        )
        // **والصفرُ لا يُكتب «+٠»** — **رقمٌ بلا معنًى يُقرأ سعراً.**
        if (option.priceDelta != 0L) {
            Text(
                "+" + money(option.priceDelta),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}
