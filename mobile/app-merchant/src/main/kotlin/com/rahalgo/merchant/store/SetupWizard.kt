package com.rahalgo.merchant.store

import com.rahalgo.ui.hours.HoursPlan
import com.rahalgo.ui.hours.HoursPlanner
import com.rahalgo.ui.hours.IconChoice
import com.rahalgo.ui.hours.IconChoicePair
import com.rahalgo.ui.hours.NightHoursDialog
import com.rahalgo.ui.hours.opensAtNight
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.graphicsLayer
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
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
            idx >= steps.size -> Ready(onFinish = onFinish)
            else -> StepScreen(vm, steps, idx)
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
                .clip(Rahal.shape.lg.copy(topStart = androidx.compose.foundation.shape.CornerSize(0), topEnd = androidx.compose.foundation.shape.CornerSize(0)))
                .background(Rahal.colors.brand)
                .statusBarsPadding()
                .padding(start = Rahal.space.xl, end = Rahal.space.xl, top = Rahal.space.xxl, bottom = Rahal.space.xxl),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            // **شعارُ رحّال غو لا أيقونةُ متجر** (ملاحظةُ المالك ١) — الرسميُّ نفسُه
            // (`intro_logo` المولَّدُ من `mobile/brand/logo.png`) على أرضٍ بيضاء.
            Box(
                Modifier
                    .size(72.dp)
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.canvas),
                contentAlignment = Alignment.Center,
            ) {
                androidx.compose.foundation.Image(
                    painterResource(com.rahalgo.design.R.drawable.intro_logo),
                    contentDescription = null,
                    modifier = Modifier.size(60.dp),
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
private fun StepScreen(
    vm: StoreViewModel,
    steps: List<SetupStep>,
    idx: Int,
) {
    val step = steps[idx]
    val advance = { vm.wizardIndex = idx + 1 }
    // **ما يفعله «التالي» وأيقبله الآن** — تردّه الخطوةُ نفسُها (`Column` مضمَّنٌ
    // فيُقرأ في المرور نفسِه، بلا حالٍ يُكتب أثناء الرسم).
    var next = StepNext(false) {}

    // **كلُّ سؤالٍ في بطاقةٍ بيضاءَ فوق أرضٍ رماديّةٍ فاتحة** (ملاحظةُ المالك ٢٠٢٦-١٠-٢٨) —
    // **وشريطُ التقدّم و«السابق/التالي» خارجها.**
    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.surface),
    ) {
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
        ) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .shadow(6.dp, Rahal.shape.lg)
                    .clip(Rahal.shape.lg)
                    .background(Rahal.colors.canvas)
                    .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg)
                    .padding(Rahal.space.lg),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                next = when (step) {
                    SetupStep.Logo -> LogoStep(vm, advance)
                    SetupStep.Sections -> SectionsStep(vm, advance)
                    SetupStep.Address -> AddressStep(vm, advance)
                    SetupStep.Hours -> HoursStep(vm, advance)
                    SetupStep.Settlement -> SettlementStep(vm, advance)
                    SetupStep.Returns -> ReturnsStep(vm, advance)
                }
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
private fun StepTitle(title: Int, hint: Int? = null) {
    Text(stringResource(title), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.headlineSmall)
    // **ولا سطرَ شرحٍ إلّا للقرارين الماليّين** — الاستلامُ والإرجاع (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
    if (hint != null) {
        Text(stringResource(hint), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
    }
}

// ── ١ · الشعار — اختياريّ ──

@Composable
private fun ColumnScope.LogoStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(true, null, advance)
    val pick = com.rahalgo.ui.rememberImagePicker(crop = true) { bytes -> vm.setLogo(bytes) }
    val has = !store.logoThumbUrl.isNullOrBlank()
    StepTitle(R.string.wz_logo_q)
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

// ── ٢ · أقسامُ السوق — شبكةٌ مرتّبة (ملاحظةُ المالك ٢) ──

@Composable
private fun SectionsStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val current = vm.mySections.map { it.id }.toSet()
    var picked by rememberSaveable { mutableStateOf(current.toList()) }
    val next = StepNext(picked.isNotEmpty()) {
        if (picked.toSet() == current) advance() else vm.saveSections(picked) { advance() }
    }
    StepTitle(R.string.wz_sections_q)
    if (vm.allSections.isEmpty()) {
        Text(stringResource(R.string.wz_sections_none), color = Rahal.colors.danger)
        return next
    }
    SectionGrid(all = vm.allSections, picked = picked.toSet()) { id ->
        picked = if (id in picked) picked - id else picked + id
    }
    return next
}

// ── ٣ · العنوانُ والدبّوس — بخريطةٍ ظاهرة (ملاحظةُ المالك ٣) ──

@Composable
private fun AddressStep(
    vm: StoreViewModel,
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
    StepTitle(R.string.wz_address_q)
    OutlinedTextField(
        value = text,
        onValueChange = { vm.wizardAddress = it },
        label = { Text(stringResource(R.string.act_address)) },
        placeholder = { Text(stringResource(R.string.store_address_hint)) },
        shape = Rahal.shape.md,
        minLines = 2,
        modifier = Modifier.fillMaxWidth(),
    )
    StoreMapCard(vm, enabled = !vm.saving)
    return next
}

// ── ٤ · الدوام — يُختار صراحةً، والجمعةُ تُسأل (ملاحظتا المالك ٤ و٧) ──

@Composable
private fun HoursStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    val sorted = vm.hours.sortedBy { it.dayOfWeek }
    if (sorted.isEmpty()) {
        StepTitle(R.string.wz_hours_q)
        LoadState(vm.loading, vm.error) { vm.load() }
        return StepNext(false) {}
    }
    // **ولا اختيارَ افتراضيّاً لمن لم يضبط دوامَه** — يختار هو (قرارُ المالك).
    val plan = remember(sorted, store.hasHours) { HoursPlan(sorted, store.hasHours) }
    var nightCheck by remember { mutableStateOf<List<DayHours>?>(null) }
    val next = StepNext(plan.chosen) {
        val days = plan.days()
        if (days.any { opensAtNight(it) }) nightCheck = days else vm.saveHours(days) { advance() }
    }
    nightCheck?.let { d ->
        NightHoursDialog(d, onDismiss = { nightCheck = null }) { nightCheck = null; vm.saveHours(d) { advance() } }
    }
    StepTitle(R.string.wz_hours_q)
    HoursPlanner(plan)
    return next
}

// ── ٥ · طريقةُ استلام الأموال — بطاقتان بأيقونة (ملاحظةُ المالك ٥) ──

@Composable
private fun SettlementStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    var m by rememberSaveable { mutableStateOf(if (store.settlementConfirmed) store.settlementMethod else "") }
    // **والمتجرُ يبدّلها بنفسه** (قرارُ المالك ٢٠٢٦-١٠-٠١) — **فيُختار هنا اختياراً لا عرضاً.**
    val chosen = m
    StepTitle(R.string.wz_settlement_q, R.string.wz_settlement_hint)
    Spacer(Modifier.height(Rahal.space.sm))
    IconChoicePair(
        first = IconChoice(
            icon = R.drawable.ic_banknotes,
            title = stringResource(R.string.wz_pay_cash),
            sub = "",
            selected = m == "cash",
        ) { m = "cash" },
        second = IconChoice(
            icon = com.rahalgo.ui.R.drawable.ic_wallet,
            title = stringResource(R.string.wz_pay_wallet),
            sub = "",
            selected = m == "wallet",
        ) { m = "wallet" },
    )
    return StepNext(chosen.isNotEmpty()) { vm.setSettlementMethod(chosen) { advance() } }
}

// ── ٦ · ترجيعُ الغرض — بالبطاقتين نفسيهما ──

@Composable
private fun ReturnsStep(vm: StoreViewModel, advance: () -> Unit): StepNext {
    val store = vm.store ?: return StepNext(false) {}
    var pick by rememberSaveable {
        mutableStateOf(if (store.returnsConfirmed) (if (store.acceptsReturns) "yes" else "no") else "")
    }
    val chosen = pick
    StepTitle(R.string.wz_step_returns, R.string.wz_returns_hint)
    Spacer(Modifier.height(Rahal.space.sm))
    IconChoicePair(
        first = IconChoice(
            icon = R.drawable.ic_return,
            title = stringResource(R.string.wz_returns_yes),
            sub = "",
            selected = pick == "yes",
        ) { pick = "yes" },
        second = IconChoice(
            icon = R.drawable.ic_block,
            title = stringResource(R.string.wz_returns_no),
            sub = "",
            selected = pick == "no",
        ) { pick = "no" },
    )
    return StepNext(chosen.isNotEmpty()) { vm.setAcceptsReturns(chosen == "yes") { advance() } }
}

// ── «متجرك جاهز» ──

/**
 * **علامةُ صحٍّ كبيرةٌ تظهر بحركةٍ ثمّ يُفتح «متجري» وحدَه** (ملاحظةُ المالك ٢٠٢٦-١٠-٢٨) —
 * **بلا أزرار**: لا قرارَ بقي ليُتّخذ هنا.
 */
@Composable
private fun Ready(onFinish: () -> Unit) {
    val grow = remember { Animatable(0.4f) }
    val fade = remember { Animatable(0f) }
    val done by rememberUpdatedState(onFinish)
    LaunchedEffect(Unit) {
        launch { fade.animateTo(1f, tween(durationMillis = 350)) }
        grow.animateTo(1f, spring(dampingRatio = Spring.DampingRatioMediumBouncy, stiffness = Spring.StiffnessLow))
    }
    LaunchedEffect(Unit) {
        delay(READY_MS)
        done()
    }
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
                .size(128.dp)
                .graphicsLayer {
                    scaleX = grow.value
                    scaleY = grow.value
                    alpha = fade.value
                }
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.success),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(R.drawable.ic_check),
                contentDescription = null,
                tint = Rahal.colors.onBrand,
                modifier = Modifier.size(72.dp),
            )
        }
        Spacer(Modifier.height(Rahal.space.xl))
        Text(
            stringResource(R.string.wz_ready_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.headlineMedium,
            color = Rahal.colors.success,
            textAlign = TextAlign.Center,
            modifier = Modifier.graphicsLayer { alpha = fade.value },
        )
    }
}

/** **كم تبقى شاشةُ «جاهز» قبل أن يُفتح «متجري».** */
private const val READY_MS = 2_500L
