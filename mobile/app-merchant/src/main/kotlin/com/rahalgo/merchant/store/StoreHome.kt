package com.rahalgo.merchant.store

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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.shared.merchant.Store
import com.rahalgo.ui.Card
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.Tone
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متجري — بيتُ المتجر بالتصميم المعتمد** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **رأسٌ بلون العلامة** فيه الشعارُ والاسمُ، **وتحته بطاقةُ الحال من الخادم**
 * (مفتوح الآن · مغلق حسب الدوام · مغلق مؤقتاً · موقوف) **وزرُّ الإغلاق الطارئ**
 * — ولا مفتاحَ «مفتوح» يدويّ بعد اليوم: **الحالُ تتبع ساعاتِ الدوام.**
 *
 * ثمّ أرقامُ اليوم، وتنبيهُ الأصناف غير المتوفرة، **وقائمةُ «إعدادات المتجر»**:
 * كلُّ سطرٍ يقول قيمتَه الحاليّة ويفتح محرّرَه (`StorePage`).
 *
 * **وكرتُ «جهّز متجرك» خرج** — صار معالجَ الإعداد الأوّل (`SetupWizard`).
 */
@Composable
fun StoreScreen(
    vm: StoreViewModel,
    unavailable: Int = 0,
    onUnavailable: () -> Unit = {},
    onPickPoint: () -> Unit = {},
) {
    val store = vm.store
    if (vm.loading || store == null) {
        Screen {
            ScreenTitle(stringResource(R.string.store_title), "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }
    // **متجرٌ موقوفٌ يُقرأ ولا يُكتب فيه** (B6).
    val suspended = store.status == "suspended"

    vm.page?.let { p ->
        BackHandler { vm.page = null }
        StorePageScreen(vm, p, suspended, onPickPoint) { vm.page = null }
        return
    }

    var renaming by rememberSaveable { mutableStateOf<String?>(null) }
    var confirming by remember { mutableStateOf(false) }
    val pickLogo = com.rahalgo.ui.rememberImagePicker(crop = true) { bytes -> vm.setLogo(bytes) }

    if (confirming) {
        com.rahalgo.ui.ConfirmDialog(
            title = stringResource(R.string.store_emergency_q),
            body = stringResource(R.string.store_emergency_hint),
            confirm = stringResource(R.string.store_emergency_do),
            onConfirm = { vm.emergency(); confirming = false },
            onDismiss = { confirming = false },
        )
    }

    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Column(
            Modifier
                .fillMaxSize()
                .background(Rahal.colors.surface)
                .verticalScroll(rememberScrollState()),
        ) {
            HomeHeader(
                store = store,
                suspended = suspended,
                saving = vm.saving,
                onLogo = { pickLogo() },
                onRename = { renaming = store.name },
                onEmergency = { confirming = true },
                onReopen = { vm.setOpen(true) },
            )
            Column(
                Modifier.padding(Rahal.space.lg),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                // **لافتةُ الإيقاف — فوق كلّ شيء** (B6).
                if (suspended) {
                    Card(tone = Rahal.colors.danger) {
                        Text(
                            stringResource(R.string.store_suspended_banner),
                            color = Rahal.colors.danger,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
                renaming?.let { draft ->
                    Card {
                        OutlinedTextField(
                            value = draft,
                            onValueChange = { renaming = it },
                            label = { Text(stringResource(R.string.mn_store_name)) },
                            singleLine = true,
                            shape = Rahal.shape.md,
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Spacer(Modifier.height(Rahal.space.sm))
                        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                            RahalButton(
                                onClick = { vm.rename(draft); renaming = null },
                                enabled = !vm.saving && draft.isNotBlank(),
                            ) { Text(stringResource(R.string.act_save)) }
                            RahalTextButton(onClick = { renaming = null }) {
                                Text(stringResource(R.string.act_cancel))
                            }
                        }
                    }
                }

                // ── أرقامُ اليوم — للقراءة ──
                vm.report?.let { r ->
                    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                        Tile(
                            value = r.summary.orders.toString(),
                            label = stringResource(R.string.reports_today_orders),
                            sub = stringResource(R.string.home_orders_sub, r.summary.delivered, r.summary.cancelled),
                            modifier = Modifier.weight(1f),
                        )
                        // **والمستحقُّ مبيعاتُه ناقصَ العمولة** (قرارُ المالك ٢٠٢٦-٠٨-١٠) — سطرٌ تحت المبيعات.
                        Tile(
                            value = com.rahalgo.ui.grouped(r.summary.sales),
                            label = stringResource(R.string.home_sales_today),
                            sub = stringResource(R.string.reports_due_today) + " " + money(r.summary.due),
                            modifier = Modifier.weight(1f),
                        )
                    }
                }

                // ── الأصنافُ غيرُ المتوفرة — تفتح «الأصناف» ──
                if (unavailable > 0) {
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.md)
                            .background(Rahal.colors.warnTint)
                            .border(Rahal.stroke.hair, Rahal.colors.accent.copy(alpha = 0.45f), Rahal.shape.md)
                            .clickable(onClick = onUnavailable)
                            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            painterResource(com.rahalgo.ui.R.drawable.ic_warning),
                            contentDescription = null,
                            tint = Rahal.colors.accent,
                        )
                        Spacer(Modifier.width(Rahal.space.md))
                        Column(Modifier.weight(1f)) {
                            Text(
                                stringResource(R.string.home_unavailable, unavailable),
                                fontWeight = FontWeight.Bold,
                                color = Rahal.colors.ink,
                            )
                            Text(
                                stringResource(R.string.home_unavailable_sub),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Chevron(Rahal.colors.accent)
                    }
                }

                // ── إعداداتُ المتجر — كلُّ سطرٍ بقيمته ويفتح محرّرَه ──
                SettingsGroup(stringResource(R.string.store_settings_title)) {
                    val notSet = stringResource(R.string.home_val_not_set)
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_time,
                        label = stringResource(R.string.home_row_hours),
                        value = hoursValue(store, vm.hours) ?: notSet,
                        warn = !store.hasHours,
                    ) { vm.page = StorePage.Hours }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_offer,
                        label = stringResource(R.string.store_sections),
                        value = vm.mySections.joinToString(" · ") { it.name }.ifBlank { notSet },
                        warn = vm.sectionsKnown && vm.mySections.isEmpty(),
                    ) { vm.page = StorePage.Sections }
                    val pointSet = store.lat != null && store.lng != null && store.addressText.isNotBlank()
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_pin,
                        label = stringResource(R.string.home_row_address),
                        value = if (pointSet) stringResource(R.string.home_val_point_set) else notSet,
                        warn = !pointSet,
                    ) { vm.page = StorePage.Address }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_wallet,
                        label = stringResource(R.string.home_row_settlement),
                        value = when {
                            !store.settlementConfirmed -> notSet
                            store.settlementMethod == "wallet" -> stringResource(R.string.home_val_wallet)
                            else -> stringResource(R.string.home_val_cash)
                        },
                        warn = !store.settlementConfirmed,
                    ) { vm.page = StorePage.Settlement }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_history,
                        label = stringResource(R.string.home_row_returns),
                        value = when {
                            !store.returnsConfirmed -> notSet
                            store.acceptsReturns -> stringResource(R.string.home_val_returns_yes)
                            else -> stringResource(R.string.home_val_returns_no)
                        },
                        warn = !store.returnsConfirmed,
                    ) { vm.page = StorePage.Returns }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_chart,
                        label = stringResource(R.string.home_row_rate),
                        value = store.exchangeRate?.let { com.rahalgo.ui.menu.usdText(it) } ?: notSet,
                    ) { vm.page = StorePage.Rate }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_work,
                        label = stringResource(R.string.store_prep),
                        value = "${store.prepMinutes} ${stringResource(R.string.store_prep_unit)}",
                        last = true,
                    ) { vm.page = StorePage.Prep }
                }
                Spacer(Modifier.height(Rahal.space.xl))
            }
        }
    }
}

/** **رأسُ «متجري»** — لونُ العلامة، والشعارُ والاسمُ، وبطاقةُ الحال تحتهما. */
@Composable
private fun HomeHeader(
    store: Store,
    suspended: Boolean,
    saving: Boolean,
    onLogo: () -> Unit,
    onRename: () -> Unit,
    onEmergency: () -> Unit,
    onReopen: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(bottomStart = Rahal.space.xl, bottomEnd = Rahal.space.xl))
            .background(Rahal.colors.brand)
            .padding(start = Rahal.space.lg, end = Rahal.space.lg, top = Rahal.space.lg, bottom = Rahal.space.lg),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            // **وشعارُه يُضغط فيُغيَّر** (طلبُ المالك ٢٠٢٦-١٠-١٨).
            com.rahalgo.ui.RemoteImage(
                url = com.rahalgo.ui.AppCore.get().media(store.logoThumbUrl),
                name = store.name.ifBlank { "?" },
                modifier = Modifier
                    .size(56.dp)
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.canvas)
                    .clickable(enabled = !suspended && !saving, onClick = onLogo),
            )
            Spacer(Modifier.width(Rahal.space.md))
            Column(Modifier.weight(1f)) {
                Text(
                    store.name,
                    color = Rahal.colors.onBrand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (!suspended) {
                    Text(
                        stringResource(
                            if (store.logoThumbUrl.isNullOrBlank()) com.rahalgo.ui.R.string.store_logo_add
                            else R.string.home_change_logo,
                        ),
                        color = Rahal.colors.onBrand.copy(alpha = 0.85f),
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.clickable(enabled = !saving, onClick = onLogo),
                    )
                } else if (store.addressText.isNotBlank()) {
                    Text(
                        store.addressText,
                        color = Rahal.colors.onBrand.copy(alpha = 0.85f),
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            if (!suspended) {
                IconButton(onClick = onRename, modifier = Modifier.size(48.dp)) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_edit),
                        contentDescription = stringResource(R.string.store_rename),
                        tint = Rahal.colors.onBrand,
                    )
                }
            }
        }
        Spacer(Modifier.height(Rahal.space.md))
        StatusCard(store, suspended, saving, onEmergency, onReopen)
    }
}

/**
 * **بطاقةُ الحال — من الخادم لا من التطبيق** (B5).
 *
 * أربعُ حالات: موقوف · مغلق مؤقتاً (طارئ) · مغلق حسب الدوام (بموعد الفتح القادم)
 * · مفتوح الآن. **والزرُّ الوحيدُ «إغلاق طارئ»، ويصير «إعادة الفتح» في الطارئ.**
 */
@Composable
private fun StatusCard(
    store: Store,
    suspended: Boolean,
    saving: Boolean,
    onEmergency: () -> Unit,
    onReopen: () -> Unit,
) {
    val label = when {
        suspended -> stringResource(R.string.store_state_suspended)
        store.emergencyClosed -> stringResource(R.string.store_state_temp_closed)
        !store.openNow -> stringResource(R.string.store_state_closed_hours)
        else -> stringResource(R.string.home_open_now)
    }
    val color = when {
        suspended -> Rahal.colors.danger
        store.emergencyClosed || !store.openNow -> Rahal.colors.inkMuted
        else -> Rahal.colors.success
    }
    val sub = when {
        suspended || store.emergencyClosed -> null
        !store.openNow -> com.rahalgo.ui.backAtText(store.nextOpen)?.let { stringResource(R.string.store_next_open, it) }
        else -> stringResource(R.string.home_by_hours)
    }
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(12.dp)
                .clip(Rahal.shape.pill)
                .background(color),
        )
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(label, color = color, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
            if (sub != null) {
                Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
            }
        }
        if (!suspended) {
            if (store.emergencyClosed) {
                // **والفتحُ بلا سؤال** — كما كان المفتاح.
                RahalOutlineButton(onClick = onReopen, enabled = !saving, tone = Tone.Brand) {
                    Text(stringResource(R.string.home_reopen), fontWeight = FontWeight.Bold)
                }
            } else {
                // **والإغلاقُ يُسأل قبل أن يُنفَّذ** — ضغطةٌ تغلق متجراً وتُبلّغ الإدارة.
                RahalOutlineButton(onClick = onEmergency, enabled = !saving, tone = Tone.Danger) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_power),
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(Modifier.width(Rahal.space.xs))
                    Text(stringResource(R.string.store_emergency), fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
private fun Tile(value: String, label: String, sub: String, modifier: Modifier = Modifier) {
    Column(
        modifier
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(Rahal.space.md),
    ) {
        Text(value, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.headlineSmall, color = Rahal.colors.ink)
        Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.labelSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

/** **قيمةُ الدوام في سطرٍ واحد** — أو فراغٌ إن لم يُضبط بعد. */
@Composable
private fun hoursValue(store: Store, hours: List<DayHours>): String? {
    if (!store.hasHours) return null
    val open = hours.filter { !it.closed }
    if (open.isEmpty()) return null
    if (open.all { it.isOpen24h }) return stringResource(R.string.hours_open_24h)
    val first = open.first()
    return if (open.all { !it.isOpen24h && it.openTime.take(5) == first.openTime.take(5) && it.closeTime.take(5) == first.closeTime.take(5) }) {
        stringResource(R.string.home_val_hours_range, clock12(first.openTime.take(5)), clock12(first.closeTime.take(5)))
    } else {
        stringResource(R.string.home_val_hours_mixed)
    }
}

/** **سهمُ «يفتح»** — سهمُ الرجوع مقلوباً، فيشير إلى الأمام بالعربي. */
@Composable
internal fun Chevron(tint: Color = Rahal.colors.inkMuted) {
    Icon(
        painterResource(com.rahalgo.ui.R.drawable.ic_arrow_back),
        contentDescription = null,
        tint = tint,
        modifier = Modifier
            .size(18.dp)
            .graphicsLayer { rotationZ = 180f },
    )
}

/** **مجموعةُ أسطرٍ بعنوان** — كما في التصميم: عنوانٌ باهتٌ وصندوقٌ أبيضُ بحدّ. */
@Composable
internal fun SettingsGroup(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column {
        Text(
            title,
            color = Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
            modifier = Modifier.padding(horizontal = Rahal.space.xs),
        )
        Spacer(Modifier.height(Rahal.space.sm))
        Column(
            Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.canvas)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
            content = content,
        )
    }
}

/** **سطرُ إعداد** — أيقونةٌ في مربّع، واسمٌ، وقيمتُه الحاليّة، وسهم. */
@Composable
internal fun SettingsRow(
    icon: Int,
    label: String,
    value: String = "",
    warn: Boolean = false,
    last: Boolean = false,
    onClick: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 56.dp)
            .clickable(onClick = onClick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(32.dp)
                .clip(Rahal.shape.sm)
                .background(Rahal.colors.brand.copy(alpha = 0.10f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = Rahal.colors.brand, modifier = Modifier.size(18.dp))
        }
        Spacer(Modifier.width(Rahal.space.md))
        Text(label, color = Rahal.colors.ink, style = MaterialTheme.typography.bodyLarge)
        Spacer(Modifier.width(Rahal.space.sm))
        Text(
            value,
            color = if (warn) Rahal.colors.danger else Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            textAlign = androidx.compose.ui.text.style.TextAlign.End,
            modifier = Modifier.weight(1f),
        )
        Spacer(Modifier.width(Rahal.space.xs))
        Chevron()
    }
    if (!last) HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
}

/**
 * **بطاقةُ اختيارٍ كبيرة** — دائرةٌ وعنوانٌ وسطرُ شرح، والمختارةُ بحدّ العلامة.
 * (التصميمُ المعتمد: خطوةُ الدوام وما يشبهها.)
 */
@Composable
internal fun ChoiceCard(
    title: String,
    sub: String,
    selected: Boolean,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(if (selected) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (selected) Rahal.stroke.focus else Rahal.stroke.hair,
                if (selected) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(
            selected = selected,
            onClick = null,
            enabled = enabled,
            colors = RadioButtonDefaults.colors(selectedColor = Rahal.colors.brand),
        )
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(title, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium, color = Rahal.colors.ink)
            if (sub.isNotBlank()) {
                Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

/** **رأسُ صفحةٍ فرعيّة** — سهمُ رجوعٍ وعنوان. */
@Composable
internal fun PageHeader(title: String, onBack: () -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = onBack, modifier = Modifier.size(48.dp)) {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_arrow_back),
                contentDescription = stringResource(R.string.home_back),
                tint = Rahal.colors.ink,
            )
        }
        Spacer(Modifier.width(Rahal.space.xs))
        Text(title, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleLarge)
    }
    Spacer(Modifier.height(Rahal.space.sm))
}

/**
 * **صفحةُ إعدادٍ واحدة** — تُفتح من سطرها في «متجري»، **وفيها المحرّرُ القائمُ
 * نفسُه** (الدوام · الأقسام · العنوان والدبّوس · المستحقّات · الترجيع · سعرُ
 * الصرف · مدّةُ التحضير) بلا نداءٍ جديد.
 */
@Composable
private fun StorePageScreen(
    vm: StoreViewModel,
    page: StorePage,
    suspended: Boolean,
    onPickPoint: () -> Unit,
    onBack: () -> Unit,
) {
    val store = vm.store ?: return
    Screen {
        PageHeader(
            title = stringResource(
                when (page) {
                    StorePage.Hours -> R.string.home_row_hours
                    StorePage.Sections -> R.string.store_sections
                    StorePage.Address -> R.string.home_row_address
                    StorePage.Settlement -> R.string.home_row_settlement
                    StorePage.Returns -> R.string.home_row_returns
                    StorePage.Rate -> R.string.home_row_rate
                    StorePage.Prep -> R.string.store_prep
                },
            ),
            onBack = onBack,
        )
        when (page) {
            StorePage.Hours -> {
                if (vm.hours.isEmpty()) LoadState(false, vm.error) { vm.load() } else HoursEditor(vm)
            }
            StorePage.Sections -> {
                if (vm.allSections.isEmpty()) LoadState(false, vm.error) { vm.load() } else SectionsEditor(vm)
            }
            StorePage.Address -> AddressEditor(vm, onPickPoint)
            StorePage.Settlement -> Card {
                // **ويبدّلها بنفسه** (قرارُ المالك ٢٠٢٦-١٠-٠١) — **ويُسأل قبل التبديل**:
                // للطلبات الجديدة وحدَها، والمستحقُّ النقديُّ القائمُ يبقى حتّى يُصرف.
                var askMethod by rememberSaveable { mutableStateOf<String?>(null) }
                BigChoice(
                    question = stringResource(R.string.q_settlement),
                    options = listOf(
                        stringResource(R.string.store_settlement_cash) to (store.settlementConfirmed && store.settlementMethod == "cash"),
                        stringResource(R.string.store_settlement_wallet) to (store.settlementConfirmed && store.settlementMethod == "wallet"),
                    ),
                    enabled = !vm.saving && !suspended,
                ) { i ->
                    val m = if (i == 0) "cash" else "wallet"
                    if (!store.settlementConfirmed || store.settlementMethod != m) askMethod = m
                }
                if (store.settlementMethod == "cash" && store.unpaidCashDue > 0) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    KeyValue(
                        label = stringResource(R.string.store_unpaid_cash),
                        value = money(store.unpaidCashDue),
                        valueColor = Rahal.colors.brand,
                    )
                }
                askMethod?.let { m ->
                    com.rahalgo.ui.ConfirmDialog(
                        title = stringResource(R.string.settlement_q),
                        body = stringResource(R.string.settlement_body),
                        confirm = stringResource(R.string.settlement_yes),
                        onConfirm = { vm.setSettlementMethod(m); askMethod = null },
                        onDismiss = { askMethod = null },
                        danger = false,
                    )
                }
            }
            StorePage.Returns -> Card {
                BigChoice(
                    question = stringResource(R.string.q_returns),
                    options = listOf(
                        stringResource(R.string.yes) to (store.returnsConfirmed && store.acceptsReturns),
                        stringResource(R.string.no) to (store.returnsConfirmed && !store.acceptsReturns),
                    ),
                    enabled = !vm.saving && !suspended,
                ) { i -> vm.setAcceptsReturns(i == 0) }
            }
            StorePage.Rate -> RateCard(vm, store, suspended)
            StorePage.Prep -> Card {
                // **ورقمٌ متفائلٌ يكسر الوعدَ لا يُسرّع المطبخ.**
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(R.string.store_prep),
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyLarge,
                    )
                    RahalTextButton(
                        onClick = { vm.setPrepMinutes(store.prepMinutes - 5) },
                        enabled = !vm.saving && !suspended && store.prepMinutes > 5,
                    ) { Text("−", style = MaterialTheme.typography.titleLarge) }
                    Text(
                        "${store.prepMinutes} ${stringResource(R.string.store_prep_unit)}",
                        fontWeight = FontWeight.Bold,
                    )
                    RahalTextButton(
                        onClick = { vm.setPrepMinutes(store.prepMinutes + 5) },
                        enabled = !vm.saving && !suspended,
                    ) { Text("+", style = MaterialTheme.typography.titleLarge) }
                }
            }
        }
    }
}

/**
 * **سعرُ الصرف وعملةُ العرض** (طلبُ المالك ٢٠٢٦-١٠-١٢) — ينحفظ لحاله بعد ثانيتين
 * ونصف من آخر حرف، **ولا يُحفظ دون عشر ليرات**، **وقفزةٌ فوق ١٠٪ تُسأل.**
 */
@Composable
private fun RateCard(vm: StoreViewModel, store: Store, suspended: Boolean) {
    Card {
        var rateText by rememberSaveable(store.id) { mutableStateOf(com.rahalgo.ui.menu.usdText(store.exchangeRate)) }
        LaunchedEffect(rateText) {
            val v = com.rahalgo.ui.menu.parseUsd(rateText)
            if (!suspended && v != null && v >= MIN_RATE && v != store.exchangeRate) {
                kotlinx.coroutines.delay(2500)
                vm.setExchangeRate(rateText)
            }
        }
        Text(stringResource(R.string.store_rate), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.height(Rahal.space.sm))
        OutlinedTextField(
            value = rateText,
            onValueChange = { v -> rateText = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(10) },
            label = { Text(stringResource(R.string.store_rate_field)) },
            singleLine = true,
            enabled = !suspended,
            shape = Rahal.shape.md,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Rahal.space.md))
        BigChoice(
            question = stringResource(R.string.q_display),
            options = listOf(
                stringResource(R.string.store_display_syp) to (store.displayCurrency != "USD"),
                stringResource(R.string.store_display_usd) to (store.displayCurrency == "USD"),
            ),
            enabled = !vm.saving && !suspended,
        ) { i ->
            if (i == 1 && (store.exchangeRate ?: 0.0) <= 0.0) {
                com.rahalgo.ui.Flash.fail(com.rahalgo.ui.AppCore.get().app.getString(R.string.store_display_need_rate))
            } else {
                vm.setDisplayCurrency(if (i == 0) "SYP" else "USD")
            }
        }
    }
    vm.rateJump?.let { j ->
        AlertDialog(
            onDismissRequest = { vm.dismissRateJump() },
            title = { Text(stringResource(R.string.store_rate_jump_title)) },
            text = {
                Column {
                    Text(stringResource(R.string.store_rate_jump_body, j.oldRate, j.newRate))
                    if (j.samples.isNotBlank()) {
                        Spacer(Modifier.height(Rahal.space.sm))
                        Text(stringResource(R.string.store_rate_jump_samples), fontWeight = FontWeight.Bold)
                        Text(j.samples)
                    }
                }
            },
            confirmButton = {
                RahalTextButton(onClick = { vm.setExchangeRate(j.newRate, confirm = true) }) {
                    Text(stringResource(R.string.store_rate_jump_yes))
                }
            },
            dismissButton = {
                RahalTextButton(onClick = { vm.dismissRateJump() }) {
                    Text(stringResource(R.string.store_rate_jump_no))
                }
            },
        )
    }
}
