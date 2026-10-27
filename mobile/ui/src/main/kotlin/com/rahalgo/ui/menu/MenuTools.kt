package com.rahalgo.ui.menu

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Checkbox
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.BulkPriceInput
import com.rahalgo.shared.merchant.BulkPriceResult
import com.rahalgo.shared.merchant.OptionPreset
import com.rahalgo.shared.merchant.OptionPresetsBody
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.shared.merchant.SetOptionPresets
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.Flash
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أدواتُ القائمة — خياراتُ الأصناف والأسعارُ دفعةً واحدة** (طلبُ المالك ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مشتركةٌ بين المتجر والمندوب** كقائمة المشروبات (`Drinks.kt`) — **والفرقُ
 * بابُ الشبكة وحدَه.** ومرآتُها في لوحة المتجر على الويب.
 */
class OptionPresetsState(
    private val scope: CoroutineScope,
    private val load: suspend () -> OptionPresetsBody,
    private val save: suspend (SetOptionPresets) -> OptionPresetsBody,
    private val errText: (Throwable) -> String,
) {
    /** **القائمةُ كلُّها** — للاختيار في شاشة «خيارات الأصناف». */
    var all by mutableStateOf<List<OptionPreset>>(emptyList())
        private set

    /**
     * **ما يطلع في محرّر الصنف** — و`null` لم تُقرأ بعد (فتبقى القوالبُ القديمة).
     */
    var chosen by mutableStateOf<List<OptionPreset>?>(null)
        private set

    /** **أهو اختيارُه أم تصنيفُه؟** */
    var custom by mutableStateOf(false)
        private set

    /** **المعلَّمُ في الشاشة** قبل الحفظ. */
    var picked by mutableStateOf<Set<String>>(emptySet())
        private set

    var open by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set

    private fun fill(b: OptionPresetsBody) {
        all = b.presets
        chosen = b.presets.filter { it.chosen }
        custom = b.custom
        picked = chosen.orEmpty().map { it.id }.toSet()
    }

    /** **يُقرأ مع القائمة** — فيجد المحرّرُ الخياراتِ حين يُفتح. */
    fun refresh() {
        scope.launch { runCatching { load() }.onSuccess { fill(it) } }
    }

    fun show() {
        error = ""
        open = true
        scope.launch {
            runCatching { load() }
                .onSuccess { fill(it) }
                .onFailure { error = errText(it) }
        }
    }

    fun close() {
        open = false
    }

    fun toggle(id: String) {
        picked = if (id in picked) picked - id else picked + id
    }

    private fun send(body: SetOptionPresets) {
        busy = true
        scope.launch {
            runCatching { save(body) }
                .onSuccess {
                    fill(it)
                    open = false
                    Flash.ok(AppCore.get().app.getString(R.string.presets_saved))
                }
                .onFailure { error = errText(it) }
            busy = false
        }
    }

    fun submit() = send(SetOptionPresets(presetIds = all.map { it.id }.filter { it in picked }))

    fun reset() = send(SetOptionPresets(reset = true))
}

/** **تغييرُ الأسعار دفعةً واحدة** — معاينةٌ أوّلاً ثمّ تأكيد. */
class BulkPriceState(
    private val scope: CoroutineScope,
    private val run: suspend (BulkPriceInput) -> BulkPriceResult,
    /** **بعد التطبيق** — تُقرأ القائمةُ من جديد. */
    private val onApplied: () -> Unit,
    private val errText: (Throwable) -> String,
) {
    var open by mutableStateOf(false)
        private set
    var percent by mutableStateOf("")
    var up by mutableStateOf(true)
    /** **قسمٌ واحد** — وفارغٌ كلُّ الأصناف. */
    var sectionId by mutableStateOf("")
    var preview by mutableStateOf<BulkPriceResult?>(null)
        private set
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set

    fun show() {
        percent = ""
        up = true
        sectionId = ""
        preview = null
        error = ""
        open = true
    }

    fun close() {
        open = false
    }

    /** **أيُّ تبديلٍ يُسقط المعاينة** — فلا يُؤكَّد ما لم يُرَ. */
    fun changed() {
        preview = null
        error = ""
    }

    private fun input(confirm: Boolean): BulkPriceInput? {
        val p = percent.filter { it.isDigit() }.toIntOrNull()
        if (p == null || p <= 0 || (up && p > 100) || (!up && p > 50)) {
            error = AppCore.get().app.getString(R.string.bulk_bad_percent)
            return null
        }
        return BulkPriceInput(
            percent = (if (up) p else -p).toDouble(),
            platformSectionId = sectionId.ifBlank { null },
            confirm = confirm,
        )
    }

    fun look() {
        val body = input(false) ?: return
        busy = true
        scope.launch {
            runCatching { run(body) }
                .onSuccess { preview = it }
                .onFailure { error = errText(it) }
            busy = false
        }
    }

    fun apply() {
        val body = input(true) ?: return
        busy = true
        scope.launch {
            runCatching { run(body) }
                .onSuccess {
                    open = false
                    Flash.ok(AppCore.get().app.getString(R.string.bulk_done, it.count))
                    onApplied()
                }
                .onFailure { error = errText(it) }
            busy = false
        }
    }
}

/** **زرّا الأدوات** تحت زرّ المشروبات. */
@Composable
fun MenuToolsButtons(presets: OptionPresetsState, bulk: BulkPriceState) {
    Spacer(Modifier.height(6.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
        RahalOutlineButton(onClick = { presets.show() }, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.presets_title))
        }
        RahalOutlineButton(onClick = { bulk.show() }, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.bulk_title))
        }
    }
}

@Composable
fun OptionPresetsEditor(state: OptionPresetsState) {
    Screen {
        ScreenTitle(stringResource(R.string.presets_title), stringResource(R.string.presets_hint))
        Text(
            stringResource(if (state.custom) R.string.presets_custom else R.string.presets_from_category),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(8.dp))
        state.all.forEach { p ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = p.id in state.picked, onCheckedChange = { state.toggle(p.id) })
                Column(Modifier.weight(1f)) {
                    Text(p.name, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                    val opts = p.options.joinToString(" · ") { it.name }
                    Text(
                        if (opts.isBlank()) stringResource(R.string.presets_empty_opts) else opts,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(state.error, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(onClick = { state.submit() }, enabled = !state.busy) { Text(stringResource(R.string.act_save)) }
            RahalTextButton(onClick = { state.close() }) { Text(stringResource(R.string.act_cancel)) }
        }
        if (state.custom) {
            Spacer(Modifier.height(8.dp))
            RahalTextButton(onClick = { state.reset() }, enabled = !state.busy) {
                Text(stringResource(R.string.presets_reset))
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
fun BulkPriceEditor(state: BulkPriceState, sections: List<PlatformSectionRef>) {
    Screen {
        ScreenTitle(stringResource(R.string.bulk_title), stringResource(R.string.bulk_hint))
        Row(verticalAlignment = Alignment.CenterVertically) {
            RadioButton(selected = state.up, onClick = { state.up = true; state.changed() })
            Text(stringResource(R.string.bulk_up))
            Spacer(Modifier.width(16.dp))
            RadioButton(selected = !state.up, onClick = { state.up = false; state.changed() })
            Text(stringResource(R.string.bulk_down))
        }
        OutlinedTextField(
            value = state.percent,
            onValueChange = { v -> state.percent = v.filter { it.isDigit() }.take(3); state.changed() },
            label = { Text(stringResource(R.string.bulk_percent)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(10.dp))
        Text(stringResource(R.string.bulk_scope), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelLarge)
        Row(verticalAlignment = Alignment.CenterVertically) {
            RadioButton(selected = state.sectionId.isEmpty(), onClick = { state.sectionId = ""; state.changed() })
            Text(stringResource(R.string.bulk_all))
        }
        sections.forEach { s ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                RadioButton(selected = state.sectionId == s.id, onClick = { state.sectionId = s.id; state.changed() })
                Text(s.name)
            }
        }
        state.preview?.let { pv ->
            Spacer(Modifier.height(10.dp))
            Card {
                Column(Modifier.padding(4.dp)) {
                    if (pv.count == 0) {
                        Text(stringResource(R.string.bulk_none))
                    } else {
                        Text(stringResource(R.string.bulk_preview_count, pv.count), fontWeight = FontWeight.Bold)
                        pv.samples.forEach { sm ->
                            Text(
                                stringResource(R.string.bulk_sample, sm.name, sm.oldPrice, sm.newPrice),
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(state.error, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            val pv = state.preview
            if (pv == null || pv.count == 0) {
                RahalButton(onClick = { state.look() }, enabled = !state.busy) { Text(stringResource(R.string.bulk_preview)) }
            } else {
                RahalButton(onClick = { state.apply() }, enabled = !state.busy) { Text(stringResource(R.string.bulk_apply)) }
            }
            RahalTextButton(onClick = { state.close() }) { Text(stringResource(R.string.act_cancel)) }
        }
        Spacer(Modifier.height(24.dp))
    }
}
