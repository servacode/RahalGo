package com.rahalgo.ui.menu

import androidx.compose.foundation.selection.selectable
import androidx.compose.ui.semantics.Role
import com.rahalgo.ui.money
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

    /**
     * **أقُرئت القائمةُ؟** (تدقيقُ ٢٠٢٦-١٠-٢٨) — كان «احفظ» يُضغط قبل أن تصل،
     * **فإن فشلت القراءةُ أُرسلت قائمةٌ فارغةٌ تمحو اختيارَ المتجر.**
     */
    var loaded by mutableStateOf(false)
        private set
    var loading by mutableStateOf(false)
        private set

    /** **ما وصل من المحرّك** — فحفظٌ لم يُغيّر شيئاً لا يُرسَل. */
    private var savedPicked: Set<String> = emptySet()

    /**
     * **رقمُ الجيل** — يزيد مع كلّ `reset()`، **فردٌّ متأخّرٌ لعميلٍ سابقٍ يُرمى**
     * ولا يكتب خياراتِه فوق خيارات العميل الجديد.
     */
    private var gen = 0

    private fun fill(b: OptionPresetsBody) {
        all = b.presets
        chosen = b.presets.filter { it.chosen }
        custom = b.custom
        picked = chosen.orEmpty().map { it.id }.toSet()
        savedPicked = picked
        loaded = true
    }

    /** **يُقرأ مع القائمة** — فيجد المحرّرُ الخياراتِ حين يُفتح. */
    fun refresh() {
        val g = gen
        scope.launch { runCatching { load() }.onSuccess { if (g == gen) fill(it) } }
    }

    /**
     * **يمحو ما قُرئ** — عند تبديل المتجر (المندوبُ بين عملائه)، **فلا تطلع
     * خياراتُ العميل السابق في محرّر العميل الجديد.**
     */
    fun reset() {
        gen++
        all = emptyList()
        chosen = null
        custom = false
        picked = emptySet()
        savedPicked = emptySet()
        loaded = false
        loading = false
        error = ""
        open = false
        busy = false
    }

    fun show() {
        error = ""
        open = true
        loading = true
        val g = gen
        scope.launch {
            runCatching { load() }
                .onSuccess { if (g == gen) fill(it) }
                .onFailure { if (g == gen) error = errText(it) }
            if (g == gen) loading = false
        }
    }

    fun close() {
        open = false
    }

    fun toggle(id: String) {
        picked = if (id in picked) picked - id else picked + id
    }

    /** **يُحفظ؟** — بعد القراءة وحدَها، **وبقائمةٍ فيها شيءٌ يُختار.** */
    val canSave: Boolean get() = loaded && !loading && !busy && all.isNotEmpty()

    private fun send(body: SetOptionPresets) {
        busy = true
        val g = gen
        scope.launch {
            runCatching { save(body) }
                .onSuccess {
                    if (g != gen) return@onSuccess
                    fill(it)
                    open = false
                    Flash.ok(AppCore.get().app.getString(R.string.presets_saved))
                }
                .onFailure { if (g == gen) error = errText(it) }
            if (g == gen) busy = false
        }
    }

    /**
     * **ولا يُرسَل ما لم يتغيّر** — كان حفظٌ بلا تبديلٍ يقلب المتجرَ من «خياراتِ
     * تصنيفه» إلى «اختياره» بالقائمة نفسِها، **فلا يتبع تصنيفَه بعدها.**
     */
    fun submit() {
        if (!canSave) return
        if (picked == savedPicked) {
            open = false
            return
        }
        send(SetOptionPresets(presetIds = all.map { it.id }.filter { it in picked }))
    }

    /** **يعيده إلى خيارات تصنيفه.** */
    fun resetToCategory() {
        if (!loaded || busy) return
        send(SetOptionPresets(reset = true))
    }
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

    /**
     * **رقمُ الطلب** (تدقيقُ ٢٠٢٦-١٠-٢٨) — معاينةٌ قديمةٌ تصل بعد تبديل النسبة
     * **كانت تُعرض ثمّ يُؤكَّد بالنسبة الجديدة.** فكلُّ تبديلٍ يزيده، وما وصل
     * برقمٍ قديمٍ يُرمى.
     */
    private var seq = 0

    /** **أيُّ تبديلٍ يُسقط المعاينة** — فلا يُؤكَّد ما لم يُرَ. */
    fun changed() {
        seq++
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
        if (busy) return
        val body = input(false) ?: return
        val mine = seq
        busy = true
        scope.launch {
            runCatching { run(body) }
                .onSuccess { if (mine == seq) preview = it }
                .onFailure { if (mine == seq) error = errText(it) }
            busy = false
        }
    }

    fun apply() {
        if (busy || preview == null) return
        val body = input(true) ?: return
        busy = true
        scope.launch {
            runCatching { run(body) }
                .onSuccess {
                    open = false
                    Flash.ok(AppCore.get().app.getString(R.string.bulk_done, it.count.toString()))
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
        if (state.loading && !state.loaded) {
            Text(stringResource(R.string.presets_loading), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        }
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
            RahalButton(onClick = { state.submit() }, enabled = state.canSave) { Text(stringResource(R.string.act_save)) }
            RahalTextButton(onClick = { state.close() }) { Text(stringResource(R.string.act_cancel)) }
        }
        if (state.custom && state.loaded) {
            Spacer(Modifier.height(8.dp))
            RahalTextButton(onClick = { state.resetToCategory() }, enabled = !state.busy) {
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
        // **والمدخلاتُ تُقفل والطلبُ ماشٍ** — فلا تتبدّل النسبةُ تحت معاينةٍ لم تصل.
        val free = !state.busy
        Row(verticalAlignment = Alignment.CenterVertically) {
            RadioRow(state.up, free, stringResource(R.string.bulk_up), Modifier.weight(1f)) { state.up = true; state.changed() }
            RadioRow(!state.up, free, stringResource(R.string.bulk_down), Modifier.weight(1f)) { state.up = false; state.changed() }
        }
        OutlinedTextField(
            value = state.percent,
            onValueChange = { v -> state.percent = v.filter { it in '0'..'9' }.take(3); state.changed() },
            label = { Text(stringResource(R.string.bulk_percent)) },
            singleLine = true,
            enabled = free,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(10.dp))
        Text(stringResource(R.string.bulk_scope), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelLarge)
        RadioRow(state.sectionId.isEmpty(), free, stringResource(R.string.bulk_all), Modifier.fillMaxWidth()) {
            state.sectionId = ""; state.changed()
        }
        sections.forEach { s ->
            RadioRow(state.sectionId == s.id, free, s.name, Modifier.fillMaxWidth()) {
                state.sectionId = s.id; state.changed()
            }
        }
        state.preview?.let { pv ->
            Spacer(Modifier.height(10.dp))
            Card {
                Column(Modifier.padding(4.dp)) {
                    if (pv.count == 0) {
                        Text(stringResource(R.string.bulk_none))
                    } else {
                        Text(stringResource(R.string.bulk_preview_count, pv.count.toString()), fontWeight = FontWeight.Bold)
                        pv.samples.forEach { sm ->
                            Text(
                                stringResource(R.string.bulk_sample, sm.name, money(sm.oldPrice), money(sm.newPrice)),
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

/** **سطرُ اختيارٍ يُضغط كلُّه** — لا الدائرةُ الصغيرةُ وحدَها. */
@Composable
private fun RadioRow(selected: Boolean, enabled: Boolean, label: String, modifier: Modifier, onPick: () -> Unit) {
    Row(
        modifier.selectable(selected = selected, enabled = enabled, role = Role.RadioButton, onClick = onPick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = selected, onClick = null, enabled = enabled)
        Spacer(Modifier.width(4.dp))
        Text(label)
    }
}
