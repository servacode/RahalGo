package com.rahalgo.ui.menu

import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.BulkPriceInput
import com.rahalgo.shared.merchant.BulkPriceResult
import com.rahalgo.shared.merchant.OptionPreset
import com.rahalgo.shared.merchant.OptionPresetsBody
import com.rahalgo.shared.merchant.PRICE_MODE_NONE
import com.rahalgo.shared.merchant.PRICE_MODE_WEIGHT
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.shared.merchant.SetOptionPresets
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Screen
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


/**
 * **تغييرُ الأسعار دفعةً واحدة** — بنسبةٍ أو بمبلغٍ ثابت (٢٠٢٦-١٠-٢٩).
 *
 * **ولا خطوةَ معاينةٍ منفصلة** (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩): يكتب القيمةَ
 * ويضغط «تطبيق» فيُسأل المحرّكُ بلا كتابةٍ عن العدد، **وتطلع نافذةُ تأكيدٍ
 * واحدة** «سيتغيّر سعر N صنفاً»، ثمّ يُطبَّق بالقيمة نفسِها.
 */
class BulkPriceState(
    private val scope: CoroutineScope,
    private val run: suspend (BulkPriceInput) -> BulkPriceResult,
    /** **بعد التطبيق** — تُقرأ القائمةُ من جديد. */
    private val onApplied: () -> Unit,
    private val errText: (Throwable) -> String,
) {
    var open by mutableStateOf(false)
        private set
    /** **القيمةُ كما تُكتب** — نسبةٌ أو مبلغٌ بحسب `byAmount`. */
    var value by mutableStateOf("")
    var up by mutableStateOf(true)
    /** **مبلغٌ ثابتٌ بالليرة** لا نسبة. */
    var byAmount by mutableStateOf(false)
    /** **قسمٌ واحد** — وفارغٌ كلُّ الأصناف. */
    var sectionId by mutableStateOf("")
    /** **ما تسأل عنه نافذةُ التأكيد** — وفراغُها لا نافذة. */
    var asking by mutableStateOf<BulkPriceInput?>(null)
        private set
    var askCount by mutableStateOf(0)
        private set
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set

    fun show() {
        value = ""
        up = true
        byAmount = false
        sectionId = ""
        asking = null
        error = ""
        open = true
    }

    fun close() {
        open = false
    }

    /**
     * **رقمُ الطلب** (تدقيقُ ٢٠٢٦-١٠-٢٨) — عددٌ قديمٌ يصل بعد تبديل القيمة
     * **كان يُعرض ثمّ يُؤكَّد بالقيمة الجديدة.** فكلُّ تبديلٍ يزيده، وما وصل
     * برقمٍ قديمٍ يُرمى.
     */
    private var seq = 0

    fun changed() {
        seq++
        asking = null
        error = ""
    }

    private fun input(): BulkPriceInput? {
        val app = AppCore.get().app
        val n = value.filter { it.isDigit() }.toLongOrNull()
        if (byAmount) {
            if (n == null || n <= 0 || n > 10_000_000) {
                error = app.getString(R.string.bulk_bad_amount)
                return null
            }
            return BulkPriceInput(
                amount = if (up) n else -n,
                platformSectionId = sectionId.ifBlank { null },
            )
        }
        if (n == null || n <= 0 || (up && n > 100) || (!up && n > 50)) {
            error = app.getString(R.string.bulk_bad_percent)
            return null
        }
        return BulkPriceInput(
            percent = (if (up) n else -n).toDouble(),
            platformSectionId = sectionId.ifBlank { null },
        )
    }

    /** **«تطبيق»** — يعدّ بلا كتابة، ثمّ يفتح نافذةَ التأكيد. */
    fun submit() {
        if (busy) return
        val body = input() ?: return
        val mine = seq
        busy = true
        scope.launch {
            runCatching { run(body.copy(confirm = false)) }
                .onSuccess {
                    if (mine != seq) return@onSuccess
                    if (it.count == 0) {
                        error = AppCore.get().app.getString(R.string.bulk_none)
                    } else {
                        askCount = it.count
                        asking = body
                    }
                }
                .onFailure { if (mine == seq) error = errText(it) }
            busy = false
        }
    }

    fun dismiss() {
        asking = null
    }

    /** **«تأكيد»** — بالقيمة التي عُدَّ عليها، لا بما كُتب بعدها. */
    fun confirm() {
        val body = asking ?: return
        if (busy) return
        asking = null
        busy = true
        scope.launch {
            runCatching { run(body.copy(confirm = true)) }
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

// ══════════════════════════════════════════════════════════════════════
// **قطعُ شاشات الأدوات** — رأسٌ بسهم رجوع، ومربّعُ اختيار، وزرٌّ بأيقونة.
// ══════════════════════════════════════════════════════════════════════

/** **ارتفاعُ مربّعَي «زيادة / تخفيض».** */
private val ChoiceBoxHeight = 64.dp

/** **ارتفاعُ زرَّي «نسبة / مبلغ».** */
private val SegmentHeight = 48.dp

/** **ارتفاعُ بطاقة الخيار الجاهز** — كلُّها واحد. */
private val PresetCardHeight = 76.dp

/** **دائرةُ الأيقونة في المربّع.** */
private val ChoiceIcon = 32.dp

/**
 * **رأسُ شاشة أداة** — سهمُ رجوعٍ واضحٌ (٤٨) ثمّ العنوان. **ولا سطرَ شرح**
 * (ملاحظةُ المالك ٢٠٢٦-١٠-٢٩: «ما في داعي للشرح»).
 */
@Composable
internal fun ToolHeader(title: String, onBack: () -> Unit) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = onBack, modifier = Modifier.size(48.dp)) {
            Icon(
                painter = painterResource(R.drawable.ic_arrow_back),
                contentDescription = stringResource(R.string.act_back),
                tint = Rahal.colors.ink,
            )
        }
        Spacer(Modifier.width(Rahal.space.xs))
        Text(
            title,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleLarge,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
    Spacer(Modifier.height(Rahal.space.md))
}

/** **زرّا «حفظ/تطبيق» و«إلغاء»** — متساويان، وكلٌّ بأيقونته. */
@Composable
internal fun ToolActions(
    primary: String,
    primaryEnabled: Boolean,
    onPrimary: () -> Unit,
    cancelEnabled: Boolean,
    onCancel: () -> Unit,
) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        RahalButton(onClick = onPrimary, enabled = primaryEnabled, modifier = Modifier.weight(1f)) {
            IconText(R.drawable.ic_check_mark, primary)
        }
        RahalOutlineButton(onClick = onCancel, enabled = cancelEnabled, modifier = Modifier.weight(1f)) {
            IconText(R.drawable.ic_close, stringResource(R.string.act_cancel))
        }
    }
}

/** **رسالةُ الخطأ** تحت الحقول. */
@Composable
internal fun ToolError(text: String) {
    if (text.isEmpty()) return
    Spacer(Modifier.height(Rahal.space.sm))
    Text(text, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
}

/** **زرّا الأدوات** تحت زرّ المشروبات (المندوب). */
@Composable
fun MenuToolsButtons(presets: OptionPresetsState, bulk: BulkPriceState) {
    Spacer(Modifier.height(6.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
        RahalOutlineButton(onClick = { presets.show() }, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.presets_title), maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
        RahalOutlineButton(onClick = { bulk.show() }, modifier = Modifier.weight(1f)) {
            Text(stringResource(R.string.bulk_title), maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«خيارات الأصناف»** (تصميمُ ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * بطاقاتٌ متساويةٌ في قائمة: الاسمُ ورقاقةُ طريقة السعر، وتحتهما معاينةُ
 * القيم، **ومربّعُ اختيارٍ واضح** — والبطاقةُ كلُّها تُضغط.
 */
@Composable
fun OptionPresetsEditor(state: OptionPresetsState) {
    Screen {
        ToolHeader(stringResource(R.string.presets_title)) { state.close() }
        if (state.loading && !state.loaded) {
            Text(stringResource(R.string.presets_loading), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            state.all.forEach { p ->
                PresetCard(p, on = p.id in state.picked, enabled = !state.busy) { state.toggle(p.id) }
            }
        }
        ToolError(state.error)
        Spacer(Modifier.height(Rahal.space.lg))
        ToolActions(
            primary = stringResource(R.string.act_save),
            primaryEnabled = state.canSave,
            onPrimary = { state.submit() },
            cancelEnabled = !state.busy,
            onCancel = { state.close() },
        )
        if (state.custom && state.loaded) {
            Spacer(Modifier.height(Rahal.space.sm))
            RahalTextButton(
                onClick = { state.resetToCategory() },
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) { IconText(R.drawable.ic_history, stringResource(R.string.presets_reset)) }
        }
    }
}

@Composable
private fun PresetCard(p: OptionPreset, on: Boolean, enabled: Boolean, onToggle: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .height(PresetCardHeight)
            .clip(Rahal.shape.md)
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.06f) else Rahal.colors.canvas)
            .border(
                if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                if (on) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .selectable(selected = on, enabled = enabled, role = Role.Checkbox, onClick = onToggle)
            .padding(start = Rahal.space.md, end = Rahal.space.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    p.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                Spacer(Modifier.width(Rahal.space.sm))
                when (p.priceMode) {
                    PRICE_MODE_NONE -> TagChip(stringResource(R.string.mod_chip_none), Rahal.colors.inkMuted)
                    PRICE_MODE_WEIGHT -> TagChip(stringResource(R.string.mod_chip_weight), Rahal.colors.accentInk)
                    else -> TagChip(stringResource(R.string.mod_chip_priced), Rahal.colors.brand)
                }
            }
            val opts = p.options.joinToString(" · ") { it.name }
            if (opts.isNotBlank()) {
                Spacer(Modifier.height(2.dp))
                Text(
                    opts,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        Checkbox(
            checked = on,
            onCheckedChange = null,
            enabled = enabled,
            colors = CheckboxDefaults.colors(checkedColor = Rahal.colors.brand),
            modifier = Modifier.size(TouchTarget),
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«تغيير الأسعار»** (تصميمُ ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * مربّعان متساويان «زيادة / تخفيض»، ثمّ «نسبة ٪ / مبلغ ثابت ل.س»، ثمّ
 * القيمة، ثمّ «على:»، ثمّ «تطبيق» ونافذةُ تأكيدٍ واحدة.
 */
@Composable
fun BulkPriceEditor(state: BulkPriceState, sections: List<PlatformSectionRef>) {
    Screen {
        ToolHeader(stringResource(R.string.bulk_title)) { state.close() }
        // **والمدخلاتُ تُقفل والطلبُ ماشٍ** — فلا تتبدّل القيمةُ تحت عدٍّ لم يصل.
        val free = !state.busy
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            ChoiceBox(R.drawable.ic_plus, stringResource(R.string.bulk_up), Rahal.colors.success, state.up, free, Modifier.weight(1f)) {
                state.up = true; state.changed()
            }
            ChoiceBox(R.drawable.ic_minus, stringResource(R.string.bulk_down), Rahal.colors.danger, !state.up, free, Modifier.weight(1f)) {
                state.up = false; state.changed()
            }
        }
        Spacer(Modifier.height(Rahal.space.md))
        Row(
            Modifier
                .fillMaxWidth()
                .height(SegmentHeight)
                .clip(Rahal.shape.md)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
        ) {
            listOf(false to R.string.bulk_mode_percent, true to R.string.bulk_mode_amount).forEach { (amount, label) ->
                val on = state.byAmount == amount
                Box(
                    Modifier
                        .weight(1f)
                        .height(SegmentHeight)
                        .background(if (on) Rahal.colors.brand else Color.Transparent)
                        .selectable(selected = on, enabled = free, role = Role.RadioButton) {
                            if (state.byAmount != amount) {
                                state.byAmount = amount
                                state.value = ""
                                state.changed()
                            }
                        },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        stringResource(label),
                        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                    )
                }
            }
        }
        Spacer(Modifier.height(Rahal.space.md))
        OutlinedTextField(
            value = state.value,
            onValueChange = { v ->
                state.value = v.filter { it in '0'..'9' }.take(if (state.byAmount) 8 else 3)
                state.changed()
            },
            label = { Text(stringResource(if (state.byAmount) R.string.bulk_amount else R.string.bulk_percent)) },
            suffix = { Text(if (state.byAmount) stringResource(R.string.item_currency_syp) else "٪") },
            singleLine = true,
            enabled = free,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Rahal.space.md))
        Text(stringResource(R.string.bulk_scope), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelLarge)
        RadioRow(state.sectionId.isEmpty(), free, stringResource(R.string.bulk_all)) {
            state.sectionId = ""; state.changed()
        }
        sections.forEach { s ->
            RadioRow(state.sectionId == s.id, free, s.name) {
                state.sectionId = s.id; state.changed()
            }
        }
        ToolError(state.error)
        Spacer(Modifier.height(Rahal.space.lg))
        ToolActions(
            primary = stringResource(R.string.bulk_go),
            primaryEnabled = free && state.value.isNotEmpty(),
            onPrimary = { state.submit() },
            cancelEnabled = free,
            onCancel = { state.close() },
        )
    }
    if (state.asking != null) {
        AlertDialog(
            onDismissRequest = { state.dismiss() },
            title = { Text(stringResource(R.string.bulk_title), fontWeight = FontWeight.Bold) },
            text = {
                Text(
                    stringResource(R.string.bulk_ask_count, state.askCount.toString()),
                    style = MaterialTheme.typography.bodyLarge,
                )
            },
            confirmButton = {
                RahalButton(onClick = { state.confirm() }) {
                    IconText(R.drawable.ic_check_mark, stringResource(R.string.act_confirm))
                }
            },
            dismissButton = {
                RahalOutlineButton(onClick = { state.dismiss() }) {
                    IconText(R.drawable.ic_close, stringResource(R.string.act_cancel))
                }
            },
        )
    }
}

/** **مربّعُ «زيادة» أو «تخفيض»** — أيقونةٌ في دائرةٍ واسمٌ، والمختارُ بحدّ المنصّة. */
@Composable
private fun ChoiceBox(
    @DrawableRes icon: Int,
    label: String,
    tint: Color,
    on: Boolean,
    enabled: Boolean,
    modifier: Modifier,
    onPick: () -> Unit,
) {
    Row(
        modifier
            .height(ChoiceBoxHeight)
            .clip(Rahal.shape.md)
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                if (on) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .selectable(selected = on, enabled = enabled, role = Role.RadioButton, onClick = onPick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.Center,
    ) {
        Box(
            Modifier
                .size(ChoiceIcon)
                .clip(CircleShape)
                .background(tint.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(20.dp))
        }
        Spacer(Modifier.width(Rahal.space.sm))
        Text(
            label,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
            color = if (on) Rahal.colors.brand else Rahal.colors.ink,
        )
    }
}

/** **سطرُ اختيارٍ يُضغط كلُّه** — لا الدائرةُ الصغيرةُ وحدَها. */
@Composable
private fun RadioRow(selected: Boolean, enabled: Boolean, label: String, onPick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .height(TouchTarget)
            .selectable(selected = selected, enabled = enabled, role = Role.RadioButton, onClick = onPick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(
            selected = selected,
            onClick = null,
            enabled = enabled,
            colors = RadioButtonDefaults.colors(selectedColor = Rahal.colors.brand),
        )
        Spacer(Modifier.width(Rahal.space.sm))
        Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}
