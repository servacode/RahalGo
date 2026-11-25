package com.rahalgo.customer.cart

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FilterChip
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.TipsConfig
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.money

/**
 * **إعدادُ الإكرامية من المنصّة** (`public/platform` ← `tips`) — يُجلب مرّةً ويبقى.
 *
 * **وغيابُه أو سقوطُ النداء ⇒ لا واجهةَ إكرامية** (خادمٌ قديم): الخادمُ هو الحاكم، والتطبيقُ يعرض فقط.
 */
object TipConfig {
    var tips by mutableStateOf<TipsConfig?>(null)
        private set

    private var loading = false

    suspend fun ensure() {
        if (tips != null || loading) return
        loading = true
        try {
            runCatching { AppCore.get().auth.platform() }
                .onSuccess { tips = it.tips ?: TipsConfig() }
        } finally {
            loading = false
        }
    }
}

/**
 * **مختارُ إكرامية الكابتن** — واحدٌ للسلّة وتأكيد العرض وتبديل الدفع.
 *
 * **يظهر فقط حين تكون الإكرامية مفعّلة**، والمنادي يضعه حين يكون الدفعُ من المحفظة.
 * `tip` بالليرة الجديدة؛ `0` = «لا شكراً».
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun TipChooser(tip: Long, onTip: (Long) -> Unit, modifier: Modifier = Modifier) {
    LaunchedEffect(Unit) { TipConfig.ensure() }
    val cfg = TipConfig.tips ?: return
    if (!cfg.enabled) return

    // **«مبلغ تاني» مفتوحٌ حين المبلغُ المختار ليس من الجاهزة.**
    var customOpen by rememberSaveable { mutableStateOf(tip > 0 && tip !in cfg.presets) }
    var text by rememberSaveable { mutableStateOf(if (tip > 0 && tip !in cfg.presets) tip.toString() else "") }

    Column(modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.tip_title),
            color = Rahal.colors.ink,
            style = Rahal.type.body,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        ) {
            cfg.presets.forEach { p ->
                FilterChip(
                    selected = !customOpen && tip == p,
                    onClick = {
                        customOpen = false
                        onTip(p)
                    },
                    label = { Text(money(p)) },
                )
            }
            FilterChip(
                selected = customOpen,
                onClick = {
                    customOpen = true
                    onTip(text.toLongOrNull()?.takeIf { it in 1..cfg.max } ?: 0L)
                },
                label = { Text(stringResource(R.string.tip_other)) },
            )
            FilterChip(
                selected = !customOpen && tip == 0L,
                onClick = {
                    customOpen = false
                    onTip(0L)
                },
                label = { Text(stringResource(R.string.tip_none)) },
            )
        }
        if (customOpen) {
            Spacer(Modifier.height(Rahal.space.sm))
            val parsed = text.toLongOrNull()
            val bad = text.isNotEmpty() && (parsed == null || parsed !in 1..cfg.max)
            OutlinedTextField(
                value = text,
                onValueChange = {
                    text = it.filter(Char::isDigit).take(9)
                    onTip(text.toLongOrNull()?.takeIf { v -> v in 1..cfg.max } ?: 0L)
                },
                label = { Text(stringResource(R.string.tip_custom_label)) },
                isError = bad,
                supportingText = if (bad) {
                    { Text(stringResource(R.string.tip_custom_error, money(cfg.max))) }
                } else {
                    null
                },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
            )
        }
        Spacer(Modifier.height(Rahal.space.xs))
        Text(
            stringResource(R.string.tip_hint),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.caption,
        )
    }
}
