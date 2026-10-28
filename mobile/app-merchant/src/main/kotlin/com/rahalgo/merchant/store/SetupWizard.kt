package com.rahalgo.merchant.store

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Tone

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الترحيبُ ومعالجُ الإعداد الأوّل** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يظهر ملءَ الشاشة متى نقص المتجرَ إعدادٌ إلزاميّ** — ولا تُفتح التبويباتُ
 * حتّى يكتمل. **وسؤالٌ واحدٌ في كلّ شاشة**، بشريط تقدّمٍ و«السابق/التالي».
 *
 * **والخطواتُ هي ما ينقص وحدَه** (`StoreViewModel.missingRequired`، وهو فحصُ
 * «جهّز متجرك — أُنجز N من 5» نفسُه) — **ومتجرٌ كاملٌ لا يراه أبداً.** والشعارُ
 * اختياريٌّ بزرّ «لاحقاً».
 *
 * **و«التالي» لا يتقدّم حتّى يردّ الخادمُ بالحفظ** — فلا يُقال «جاهز» على
 * إعدادٍ لم يصل. **وكلُّ حفظٍ بنداء محرّره القائم** — لا بابَ جديداً.
 */
@Composable
fun SetupWizard(
    vm: StoreViewModel,
    onPickPoint: () -> Unit,
    onAddItem: () -> Unit,
    onFinish: () -> Unit,
) {
    val steps = vm.wizardSteps ?: return
    val idx = vm.wizardIndex
    // **والرجوعُ خطوةٌ إلى الوراء لا خروجٌ من المعالج.**
    BackHandler(enabled = idx >= 0) { vm.wizardIndex = idx - 1 }
    Box(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.canvas),
    ) {
        when {
            idx < 0 -> Welcome(steps) { vm.wizardIndex = 0 }
            idx >= steps.size -> Ready(onAddItem = onAddItem, onFinish = onFinish)
            else -> StepScreen(vm, steps, idx, onPickPoint)
        }
    }
}

@Composable
private fun stepLabel(s: SetupStep): String = stringResource(
    when (s) {
        SetupStep.Logo -> R.string.wz_step_logo
        SetupStep.Sections -> R.string.wz_step_sections
        SetupStep.Address -> R.string.wz_step_address
        SetupStep.Hours -> R.string.wz_step_hours
        SetupStep.Settlement -> R.string.wz_step_settlement
        SetupStep.Returns -> R.string.wz_step_returns
    },
)

// ── الترحيب ─────────────────────────────────────────────────────────────

@Composable
private fun Welcome(steps: List<SetupStep>, onStart: () -> Unit) {
    Column(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(bottomStart = Rahal.space.xxl, bottomEnd = Rahal.space.xxl))
                .background(Rahal.colors.brand)
                .statusBarsPadding()
                .padding(start = Rahal.space.xl, end = Rahal.space.xl, top = Rahal.space.xxl, bottom = Rahal.space.xxl),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            Box(
                Modifier
                    .size(64.dp)
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.accent),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_store),
                    contentDescription = null,
                    tint = Rahal.colors.onLight,
                    modifier = Modifier.size(34.dp),
                )
            }
            Text(
                stringResource(R.string.wz_welcome_title),
                color = Rahal.colors.onBrand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.headlineMedium,
            )
            Text(
                stringResource(R.string.wz_welcome_body),
                color = Rahal.colors.onBrand.copy(alpha = 0.85f),
                style = MaterialTheme.typography.bodyLarge,
            )
        }
        Column(
            Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Rahal.space.xl, vertical = Rahal.space.xl),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.lg),
        ) {
            Text(
                stringResource(R.string.wz_welcome_count, steps.size),
                color = Rahal.colors.inkMuted,
                fontWeight = FontWeight.Bold,
            )
            steps.forEachIndexed { i, s ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        Modifier
                            .size(30.dp)
                            .clip(Rahal.shape.pill)
                            .background(Rahal.colors.brand.copy(alpha = 0.10f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            (i + 1).toString(),
                            color = Rahal.colors.brand,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelLarge,
                        )
                    }
                    Spacer(Modifier.width(Rahal.space.md))
                    Text(stepLabel(s), modifier = Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge)
                    if (s == SetupStep.Logo) {
                        Text(
                            stringResource(R.string.wz_optional),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
            }
        }
        RahalButton(
            onClick = onStart,
            tone = Tone.Accent,
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(horizontal = Rahal.space.xl, vertical = Rahal.space.lg)
                .height(56.dp),
        ) {
            Text(stringResource(R.string.wz_start), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
        }
    }
}

// ── خطوةٌ واحدة ─────────────────────────────────────────────────────────

@Composable
private fun StepScreen(vm: StoreViewModel, steps: List<SetupStep>, idx: Int, onPickPoint: () -> Unit) {
    val step = steps[idx]
    val advance = { vm.wizardIndex = idx + 1 }
    // **ما يفعله «التالي» وأيقبله الآن** — تردّه الخطوةُ نفسُها (`Column` مضمَّنٌ
    // فيُقرأ في المرور نفسِه، بلا حالٍ يُكتب أثناء الرسم).
    var next = StepNext(false) {}

    Column(Modifier.fillMaxSize()) {
        // ── الرأس: «إعداد متجرك» · «الخطوة N من M» · شريطُ التقدّم ──
        Column(
            Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(start = Rahal.space.lg, end = Rahal.space.lg, top = Rahal.space.lg, bottom = Rahal.space.sm),
        ) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.wz_header),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    stringResource(R.string.wz_progress, idx + 1, steps.size),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            Spacer(Modifier.height(Rahal.space.md))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                steps.indices.forEach { i ->
                    Box(
                        Modifier
                            .weight(1f)
                            .height(6.dp)
                            .clip(Rahal.shape.pill)
                            .background(if (i <= idx) Rahal.colors.brand else Rahal.colors.line),
                    )
                }
            }
        }
        // ── السؤال ──
        Column(
            Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            next = when (step) {
                SetupStep.Logo -> LogoStep(vm, advance)
                SetupStep.Sections -> SectionsStep(vm, advance)
                SetupStep.Address -> AddressStep(vm, onPickPoint, advance)
                SetupStep.Hours -> HoursStep(vm, advance)
                SetupStep.Settlement -> SettlementStep(vm, advance)
                SetupStep.Returns -> ReturnsStep(vm, advance)
            }
        }
        // ── «السابق» و«التالي» ──
        Row(
            Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        ) {
            RahalOutlineButton(
                onClick = { vm.wizardIndex = idx - 1 },
                modifier = Modifier
                    .width(104.dp)
                    .height(54.dp),
            ) { Text(stringResource(R.string.wz_prev), fontWeight = FontWeight.Bold) }
            val go = next
            RahalButton(
                onClick = { go.act() },
                enabled = go.ok && !vm.saving,
                modifier = Modifier
                    .weight(1f)
                    .height(54.dp),
            ) {
                Text(
                    stringResource(go.label ?: R.string.wz_next),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
            }
        }
    }
}

/** **«التالي» كما تريده الخطوة** — أيُقبل، وبأيّ نصّ، وماذا يفعل. */
private class StepNext(val ok: Boolean, val label: Int? = null, val act: () -> Unit)

@Composable
private fun StepTitle(title: Int, hint: Int) {
    Text(stringResource(title), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.headlineSmall)
    Text(stringResource(hint), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
}

// ── ١ · الشعار — اختياريّ ──

@Composable
private fun ColumnScope.LogoStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(true, null, advance)
    val pick = com.rahalgo.ui.rememberImagePicker(crop = true) { bytes -> vm.setLogo(bytes) }
    val has = !store.logoThumbUrl.isNullOrBlank()
    StepTitle(R.string.wz_logo_q, R.string.wz_logo_hint)
    Spacer(Modifier.height(Rahal.space.sm))
    com.rahalgo.ui.RemoteImage(
        url = com.rahalgo.ui.AppCore.get().media(store.logoThumbUrl),
        name = store.name.ifBlank { "?" },
        modifier = Modifier
            .align(Alignment.CenterHorizontally)
            .size(128.dp)
            .clip(Rahal.shape.lg)
            .border(Rahal.stroke.focus, Rahal.colors.line, Rahal.shape.lg)
            .clickable(enabled = !vm.saving) { pick() },
    )
    RahalOutlineButton(
        onClick = { pick() },
        enabled = !vm.saving,
        modifier = Modifier.align(Alignment.CenterHorizontally),
    ) { Text(stringResource(if (has) R.string.wz_logo_change else R.string.wz_logo_pick)) }
    // **وبلا شعارٍ يصير «التالي» «لاحقاً»** — يتخطّاه ولا يُحبس عنده.
    return StepNext(true, if (has) null else R.string.wz_later, advance)
}

// ── ٢ · أقسامُ السوق ──

@Composable
private fun SectionsStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val current = vm.mySections.map { it.id }.toSet()
    var picked by rememberSaveable { mutableStateOf(current.toList()) }
    val next = StepNext(picked.isNotEmpty()) {
        if (picked.toSet() == current) advance() else vm.saveSections(picked) { advance() }
    }
    StepTitle(R.string.wz_sections_q, R.string.wz_sections_hint)
    if (vm.allSections.isEmpty()) {
        Text(stringResource(R.string.wz_sections_none), color = Rahal.colors.danger)
        return next
    }
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        vm.allSections.forEach { sec ->
            val on = sec.id in picked
            Text(
                sec.name,
                color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
                style = MaterialTheme.typography.bodyLarge,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .background(if (on) Rahal.colors.brand else Rahal.colors.canvas)
                    .border(Rahal.stroke.hair, if (on) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.pill)
                    .clickable { picked = if (on) picked - sec.id else picked + sec.id }
                    .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
            )
        }
    }
    return next
}

// ── ٣ · العنوانُ والدبّوس ──

@Composable
private fun AddressStep(
    vm: StoreViewModel,
    onPickPoint: () -> Unit,
    advance: () -> Unit,
): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    val text = vm.wizardAddress ?: store.addressText
    val lat = vm.pickedLat ?: store.lat
    val lng = vm.pickedLng ?: store.lng
    val pointSet = lat != null && lng != null
    val next = StepNext(text.isNotBlank() && pointSet) {
        if (text.trim() == store.addressText && vm.pickedLat == null && store.lat != null) {
            advance()
        } else {
            vm.saveAddress(text, vm.pickedLat, vm.pickedLng) { vm.wizardAddress = null; advance() }
        }
    }
    StepTitle(R.string.wz_address_q, R.string.wz_address_hint)
    OutlinedTextField(
        value = text,
        onValueChange = { vm.wizardAddress = it },
        label = { Text(stringResource(R.string.act_address)) },
        placeholder = { Text(stringResource(R.string.store_address_hint)) },
        shape = Rahal.shape.md,
        minLines = 2,
        modifier = Modifier.fillMaxWidth(),
    )
    ChoiceRowAction(
        title = stringResource(R.string.store_point),
        status = stringResource(if (pointSet) R.string.store_point_set else R.string.store_point_none),
        ok = pointSet,
        action = stringResource(if (pointSet) R.string.store_point_change else R.string.store_point_pick),
        onAction = onPickPoint,
    )
    return next
}

@Composable
private fun ChoiceRowAction(title: String, status: String, ok: Boolean, action: String, onAction: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .clickable(onClick = onAction)
            .padding(Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_pin), contentDescription = null, tint = Rahal.colors.brand)
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(title, fontWeight = FontWeight.Bold)
            Text(
                status,
                color = if (ok) Rahal.colors.success else Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Text(action, color = Rahal.colors.brand, fontWeight = FontWeight.Bold)
    }
}

// ── ٤ · الدوام — يُختار صراحةً ──

@Composable
private fun HoursStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    val sorted = vm.hours.sortedBy { it.dayOfWeek }
    if (sorted.isEmpty()) {
        StepTitle(R.string.wz_hours_q, R.string.wz_hours_hint)
        LoadState(vm.loading, vm.error) { vm.load() }
        return StepNext(false) {}
    }
    val firstOpen = sorted.firstOrNull { !it.closed }
    // **ولا اختيارَ افتراضيّاً لمن لم يضبط دوامَه** — يختار هو (قرارُ المالك).
    var mode by rememberSaveable {
        mutableStateOf(
            when {
                !store.hasHours -> ""
                sorted.all { it.closed || it.isOpen24h } -> "24"
                else -> "custom"
            },
        )
    }
    var allOpen by rememberSaveable { mutableStateOf(firstOpen?.takeIf { !it.isOpen24h }?.openTime?.take(5) ?: "09:00") }
    var allClose by rememberSaveable { mutableStateOf(firstOpen?.takeIf { !it.isOpen24h }?.closeTime?.take(5) ?: "23:00") }
    var draft by remember(sorted) {
        mutableStateOf(
            if (store.hasHours) sorted
            else sorted.map { it.copy(closed = false, open24h = false, openTime = allOpen, closeTime = allClose) },
        )
    }
    var showDays by rememberSaveable { mutableStateOf(false) }
    var nightCheck by remember { mutableStateOf<List<DayHours>?>(null) }

    val days: List<DayHours> = if (mode == "24") sorted.map { it.with24h(true) } else draft
    val next = StepNext(mode.isNotEmpty()) {
        if (days.any { opensAtNight(it) }) nightCheck = days else vm.saveHours(days) { advance() }
    }
    nightCheck?.let { d ->
        val first = d.first { opensAtNight(it) }
        AlertDialog(
            onDismissRequest = { nightCheck = null },
            title = { Text(stringResource(R.string.hours_night_title)) },
            text = { Text(stringResource(R.string.hours_night_text, clock12(first.openTime), clock12(first.closeTime))) },
            confirmButton = {
                RahalButton(onClick = { nightCheck = null; vm.saveHours(d) { advance() } }) {
                    Text(stringResource(R.string.hours_night_yes))
                }
            },
            dismissButton = {
                RahalOutlineButton(onClick = { nightCheck = null }) { Text(stringResource(R.string.hours_night_fix)) }
            },
        )
    }

    StepTitle(R.string.wz_hours_q, R.string.wz_hours_hint)
    ChoiceCard(
        title = stringResource(R.string.hours_open_24h),
        sub = stringResource(R.string.wz_hours_24_sub),
        selected = mode == "24",
    ) { mode = "24" }
    ChoiceCard(
        title = stringResource(R.string.wz_hours_custom),
        sub = stringResource(R.string.wz_hours_custom_sub),
        selected = mode == "custom",
    ) { mode = "custom" }
    if (mode == "custom") {
        Row(
            Modifier
                .fillMaxWidth()
                .padding(vertical = Rahal.space.xs),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(stringResource(R.string.wz_hours_all), fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
            TimeField(allOpen) { v ->
                allOpen = v
                draft = draft.map { it.copy(closed = false, open24h = false, openTime = v, closeTime = allClose) }
            }
            Text(
                " ${stringResource(R.string.hours_to)} ",
                color = Rahal.colors.inkMuted,
            )
            TimeField(allClose) { v ->
                allClose = v
                draft = draft.map { it.copy(closed = false, open24h = false, openTime = allOpen, closeTime = v) }
            }
        }
        HorizontalDivider(color = Rahal.colors.line)
        RahalTextButton(onClick = { showDays = !showDays }) {
            Text(stringResource(if (showDays) R.string.wz_hours_hide_days else R.string.wz_hours_edit_day), fontWeight = FontWeight.Bold)
        }
        if (showDays) {
            draft.forEachIndexed { i, d ->
                if (i > 0) HorizontalDivider(color = Rahal.colors.line)
                HoursRow(d = d) { changed ->
                    draft = draft.map { if (it.dayOfWeek == changed.dayOfWeek) changed else it }
                }
            }
        }
    }
    return next
}

// ── ٥ · طريقةُ استلام الأموال ──

@Composable
private fun SettlementStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    var m by rememberSaveable { mutableStateOf(if (store.settlementConfirmed) store.settlementMethod else "") }
    // **والمتجرُ يبدّلها بنفسه** (قرارُ المالك ٢٠٢٦-١٠-٠١، `POST .../settlement-method`) —
    // **فيُختار هنا اختياراً لا عرضاً.**
    val chosen = m
    StepTitle(R.string.wz_settlement_q, R.string.wz_settlement_hint)
    ChoiceCard(
        title = stringResource(R.string.store_settlement_cash),
        sub = stringResource(R.string.wz_settlement_cash_sub),
        selected = m == "cash",
    ) { m = "cash" }
    ChoiceCard(
        title = stringResource(R.string.store_settlement_wallet),
        sub = stringResource(R.string.wz_settlement_wallet_sub),
        selected = m == "wallet",
    ) { m = "wallet" }
    return StepNext(chosen.isNotEmpty()) { vm.setSettlementMethod(chosen) { advance() } }
}

// ── ٦ · ترجيعُ الغرض ──

@Composable
private fun ReturnsStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    var pick by rememberSaveable {
        mutableStateOf(if (store.returnsConfirmed) (if (store.acceptsReturns) "yes" else "no") else "")
    }
    val chosen = pick
    StepTitle(R.string.wz_step_returns, R.string.wz_returns_hint)
    ChoiceCard(
        title = stringResource(R.string.yes),
        sub = stringResource(R.string.wz_returns_yes_sub),
        selected = pick == "yes",
    ) { pick = "yes" }
    ChoiceCard(
        title = stringResource(R.string.no),
        sub = stringResource(R.string.wz_returns_no_sub),
        selected = pick == "no",
    ) { pick = "no" }
    return StepNext(chosen.isNotEmpty()) { vm.setAcceptsReturns(chosen == "yes") { advance() } }
}

// ── «متجرك جاهز» ──

@Composable
private fun Ready(onAddItem: () -> Unit, onFinish: () -> Unit) {
    Column(
        Modifier
            .fillMaxSize()
            .statusBarsPadding()
            .navigationBarsPadding()
            .padding(Rahal.space.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            Modifier
                .size(88.dp)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.success.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_store),
                contentDescription = null,
                tint = Rahal.colors.success,
                modifier = Modifier.size(44.dp),
            )
        }
        Spacer(Modifier.height(Rahal.space.xl))
        Text(
            stringResource(R.string.wz_ready_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.headlineMedium,
            color = Rahal.colors.success,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        Text(
            stringResource(R.string.wz_ready_body),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyLarge,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.xxl))
        RahalButton(
            onClick = onAddItem,
            tone = Tone.Accent,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
        ) {
            Text(stringResource(R.string.wz_add_first), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
        }
        Spacer(Modifier.height(Rahal.space.sm))
        RahalTextButton(onClick = onFinish) {
            Text(stringResource(R.string.wz_go_store), fontWeight = FontWeight.Bold)
        }
    }
}
