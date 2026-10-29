package com.rahalgo.merchant.store

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CornerSize
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
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
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
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متجري — أقوى وأبسط** (ملاحظةُ المالك ٦، ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **من فوق إلى تحت**: رأسٌ بالشعار الكبير والاسم · **شريطُ الحال ملوّناً بعرض
 * الشاشة** (أخضر مفتوح · رماديّ مغلق · برتقاليّ إغلاقٌ طارئ · أحمر موقوف) وزرُّه
 * تحته · **أربعةُ أزرارٍ سريعة** (أضف صنفاً · غير المتوفر · تغيير الأسعار · اطلب
 * توصيلة) · **أرقامُ اليوم بأيقونات** · **ثمّ الإعداداتُ في مجموعتين**: «المتجر»
 * و«الدوام والمال»، كلُّ سطرٍ بأيقونةٍ ملوّنةٍ وقيمته ونقطةٍ حمراءَ إن نقص.
 *
 * **والحالُ من الخادم لا من التطبيق** (B5) — ولا مفتاحَ «مفتوح» يدويّ: الحالُ
 * تتبع ساعاتِ الدوام، **والإغلاقُ الطارئُ وحدَه بيده.**
 */
@Composable
fun StoreScreen(
    vm: StoreViewModel,
    unavailable: Int = 0,
    onUnavailable: () -> Unit = {},
    onOffers: () -> Unit = {},
    onPrices: () -> Unit = {},
    onDelivery: () -> Unit = {},
    onTodayOrders: () -> Unit = {},
    stores: List<Store> = emptyList(),
    onSwitch: (String) -> Unit = {},
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
        StorePageScreen(vm, p, suspended) { vm.page = null }
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
                stores = stores,
                onSwitch = onSwitch,
            )
            Column(
                Modifier.padding(Rahal.space.lg),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.lg),
            ) {
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

                // ── (أ) شريطُ الحال ──
                StatusBand(
                    store = store,
                    suspended = suspended,
                    saving = vm.saving,
                    onEmergency = { confirming = true },
                    onReopen = { vm.setOpen(true) },
                )

                // ── (ب) الأزرارُ السريعة — والموقوفُ لا يملكها (B6) ──
                if (!suspended) {
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .height(IntrinsicSize.Min),
                        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                    ) {
                        // **«أضف صنفاً» تبويبٌ تحت — فمكانُه هنا للعروض** (تجربةُ المالك ٢٠٢٦-١٠-٢٩).
                        QuickAction(R.drawable.ic_percent, stringResource(R.string.home_quick_offers), Rahal.colors.brand, Modifier.weight(1f), onClick = onOffers)
                        QuickAction(
                            com.rahalgo.ui.R.drawable.ic_eye_off,
                            stringResource(R.string.home_quick_unavailable),
                            Rahal.colors.brand,
                            Modifier.weight(1f),
                            badge = unavailable,
                            onClick = onUnavailable,
                        )
                        // **تغييرُ الأسعار = أوراقٌ نقديّة في كلّ مكان، والنسبةُ للعروض وحدَها** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
                        QuickAction(com.rahalgo.ui.R.drawable.ic_banknotes, stringResource(R.string.home_quick_prices), Rahal.colors.brand, Modifier.weight(1f), onClick = onPrices)
                        QuickAction(com.rahalgo.ui.R.drawable.ic_moto, stringResource(R.string.home_quick_delivery), Rahal.colors.brand, Modifier.weight(1f), onClick = onDelivery)
                    }
                }

                // ── (ج) أرقامُ اليوم — للقراءة ──
                vm.report?.let { r ->
                    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                        GroupTitle(stringResource(R.string.home_today))
                        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                            Tile(
                                icon = com.rahalgo.ui.R.drawable.ic_orders,
                                tint = Rahal.colors.brand,
                                value = r.summary.orders.toString(),
                                label = stringResource(R.string.reports_today_orders),
                                sub = stringResource(R.string.home_cancelled_sub, r.summary.cancelled.toString()),
                                // **ويُضغط فيُريه طلباتِه** (قرارُ المالك ٢٠٢٦-١٠-٢٩): كان يرى «1» ولا يعرف ما يجهّز.
                                modifier = Modifier.weight(1f).clip(Rahal.shape.md).clickable(onClick = onTodayOrders),
                            )
                            Tile(
                                icon = R.drawable.ic_check,
                                tint = Rahal.colors.brand,
                                value = r.summary.delivered.toString(),
                                label = stringResource(R.string.home_delivered_today),
                                modifier = Modifier.weight(1f),
                            )
                        }
                        // **والمستحقُّ مبيعاتُه ناقصَ العمولة** (قرارُ المالك ٢٠٢٦-٠٨-١٠).
                        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                            Tile(
                                icon = com.rahalgo.ui.R.drawable.ic_chart,
                                tint = Rahal.colors.brand,
                                value = com.rahalgo.ui.money(r.summary.sales),
                                label = stringResource(R.string.home_sales_today),
                                modifier = Modifier.weight(1f),
                            )
                            Tile(
                                icon = com.rahalgo.ui.R.drawable.ic_wallet,
                                tint = Rahal.colors.brand,
                                value = com.rahalgo.ui.money(r.summary.due),
                                label = stringResource(R.string.reports_due_today),
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }
                }

                // ── (د) الإعداداتُ في مجموعتين ──
                val notSet = stringResource(R.string.home_val_not_set)
                SettingsGroup(stringResource(R.string.home_group_store)) {
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
                        value = if (pointSet) store.addressText else notSet,
                        warn = !pointSet,
                    ) { vm.page = StorePage.Address }
                    SettingsRow(
                        icon = R.drawable.ic_image,
                        label = stringResource(R.string.home_row_logo),
                        value = stringResource(
                            if (store.logoThumbUrl.isNullOrBlank()) R.string.home_val_logo_none else R.string.home_val_logo_set,
                        ),
                        enabled = !suspended && !vm.saving,
                        last = true,
                    ) { pickLogo() }
                }
                SettingsGroup(stringResource(R.string.home_group_hours_money)) {
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_time,
                        label = stringResource(R.string.home_row_hours),
                        value = hoursValue(store, vm.hours) ?: notSet,
                        warn = !store.hasHours,
                    ) { vm.page = StorePage.Hours }
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
                        icon = R.drawable.ic_return,
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
                        label = stringResource(R.string.home_row_rate_short),
                        value = store.exchangeRate?.let { com.rahalgo.ui.menu.usdText(it) } ?: notSet,
                    ) { vm.page = StorePage.Rate }
                    SettingsRow(
                        icon = com.rahalgo.ui.R.drawable.ic_work,
                        label = stringResource(R.string.home_row_prep),
                        value = "${store.prepMinutes} ${stringResource(R.string.store_prep_unit)}",
                        last = true,
                    ) { vm.page = StorePage.Prep }
                }
                Spacer(Modifier.height(Rahal.space.xl))
            }
        }
    }
}

/**
 * **رأسُ «متجري»** — لونُ العلامة، **وشعارٌ أكبر** يُضغط فيُغيَّر، والاسمُ وزرُّ
 * تعديله. (والتصنيفُ والتقييمُ لا يردّهما `/merchant/stores` اليوم — فلا يُعرضان.)
 */
@Composable
private fun HomeHeader(
    store: Store,
    suspended: Boolean,
    saving: Boolean,
    onLogo: () -> Unit,
    onRename: () -> Unit,
    stores: List<Store> = emptyList(),
    onSwitch: (String) -> Unit = {},
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg.copy(topStart = CornerSize(0), topEnd = CornerSize(0)))
            .background(Rahal.colors.brand)
            .padding(Rahal.space.lg),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        com.rahalgo.ui.RemoteImage(
            url = com.rahalgo.ui.AppCore.get().media(store.logoThumbUrl),
            name = store.name.ifBlank { "?" },
            modifier = Modifier
                .size(80.dp)
                .clip(Rahal.shape.lg)
                .background(Rahal.colors.canvas)
                .border(Rahal.stroke.focus, Rahal.colors.canvas, Rahal.shape.lg)
                .clickable(enabled = !suspended && !saving, onClick = onLogo),
        )
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                store.name,
                color = Rahal.colors.onBrand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.headlineSmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            // **ومبدّلُ الفرع هنا لا شريطاً فوقه** (تجربةُ المالك ٢٠٢٦-١٠-٢٩) — كان الاسمُ يُقرأ مرّتين.
            if (stores.size > 1) {
                var open by remember { mutableStateOf(false) }
                androidx.compose.foundation.layout.Box {
                    Text(
                        stringResource(R.string.store_switch),
                        color = Rahal.colors.onBrand,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelLarge,
                        modifier = Modifier.clickable { open = true }.padding(vertical = Rahal.space.xs),
                    )
                    androidx.compose.material3.DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
                        stores.forEach { s ->
                            androidx.compose.material3.DropdownMenuItem(
                                text = { Text(s.name, fontWeight = if (s.id == store.id) FontWeight.Bold else FontWeight.Normal) },
                                onClick = { open = false; if (s.id != store.id) onSwitch(s.id) },
                            )
                        }
                    }
                }
            }
            // **ولا عنوانَ ولا «تغيير الشعار» في الرأس** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — الشعارُ
            // نفسُه يُضغط، وسطرا «الشعار» و«العنوان» في الإعدادات تحت.
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
}

/**
 * **شريطُ الحال — أرضيّةٌ فاتحةٌ من لون الحال** (B5، ومراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
 *
 * أخضرُ «مفتوح الآن» · رماديٌّ «مغلق — يفتح الساعة …» · برتقاليٌّ «مغلق مؤقتاً»
 * (طارئ) · أحمرُ «موقوف». **والزرُّ آخرَ الشريط محدَّداً صغيراً**: «إغلاق طارئ» يُسأل قبل التنفيذ،
 * و«إعادة الفتح» في الطارئ بلا سؤال.
 */
@Composable
private fun StatusBand(
    store: Store,
    suspended: Boolean,
    saving: Boolean,
    onEmergency: () -> Unit,
    onReopen: () -> Unit,
) {
    val nextAt = com.rahalgo.ui.backAtText(store.nextOpen)
    val label = when {
        suspended -> stringResource(R.string.store_state_suspended)
        store.emergencyClosed -> stringResource(R.string.store_state_temp_closed)
        !store.openNow -> nextAt?.let { stringResource(R.string.home_closed_until, it) } ?: stringResource(R.string.home_closed)
        else -> stringResource(R.string.home_open_now)
    }
    val sub = when {
        suspended -> stringResource(R.string.home_suspended_sub)
        store.emergencyClosed -> stringResource(R.string.home_temp_closed_sub)
        !store.openNow -> stringResource(R.string.store_state_closed_hours)
        else -> stringResource(R.string.home_by_hours)
    }
    val color = when {
        suspended -> Rahal.colors.danger
        store.emergencyClosed -> Rahal.colors.accent
        !store.openNow -> Rahal.colors.inkMuted
        else -> Rahal.colors.success
    }
    // **أرضيّةٌ فاتحةٌ من لون الحال وكتابةٌ غامقة** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — الأبيضُ على
    // البرتقاليّ لا يُقرأ في الشمس. **و«إغلاق طارئ» زرٌّ محدَّدٌ صغيرٌ آخرَ الشريط** لا البطل.
    val ink = when {
        suspended -> Rahal.colors.danger
        store.emergencyClosed -> Rahal.colors.accentInk
        !store.openNow -> Rahal.colors.ink
        else -> Rahal.colors.success
    }
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(color.copy(alpha = 0.12f))
            .border(Rahal.stroke.hair, color.copy(alpha = 0.35f), Rahal.shape.lg)
            .padding(Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(44.dp)
                .clip(Rahal.shape.pill)
                .background(color.copy(alpha = 0.18f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(
                    when {
                        suspended -> com.rahalgo.ui.R.drawable.ic_lock
                        store.emergencyClosed || !store.openNow -> com.rahalgo.ui.R.drawable.ic_power
                        else -> com.rahalgo.ui.R.drawable.ic_store
                    },
                ),
                contentDescription = null,
                tint = ink,
                modifier = Modifier.size(24.dp),
            )
        }
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(label, color = ink, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleLarge)
            Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        }
        if (!suspended) {
            Spacer(Modifier.width(Rahal.space.sm))
            val reopen = store.emergencyClosed
            com.rahalgo.ui.RahalOutlineButton(
                onClick = if (reopen) onReopen else onEmergency,
                enabled = !saving,
                tone = if (reopen) com.rahalgo.ui.Tone.Success else com.rahalgo.ui.Tone.Danger,
            ) {
                Text(
                    stringResource(if (reopen) R.string.home_reopen else R.string.store_emergency),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelLarge,
                )
            }
        }
    }
}

/** **زرٌّ سريع** — أيقونةٌ في دائرةٍ ملوّنة واسمٌ تحتها، **وعدّادٌ أحمرُ إن وُجد.** */
@Composable
private fun QuickAction(
    icon: Int,
    label: String,
    tint: Color,
    modifier: Modifier = Modifier,
    badge: Int = 0,
    onClick: () -> Unit,
) {
    Box(modifier.fillMaxHeight()) {
        Column(
            Modifier
                .fillMaxSize()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.canvas)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                .clickable(role = Role.Button, onClick = onClick)
                .padding(vertical = Rahal.space.md, horizontal = Rahal.space.xs),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier
                    .size(44.dp)
                    .clip(Rahal.shape.pill)
                    .background(tint.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(24.dp))
            }
            Spacer(Modifier.height(Rahal.space.sm))
            Text(
                label,
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.labelMedium,
                textAlign = TextAlign.Center,
                maxLines = 2,
            )
        }
        if (badge > 0) {
            Text(
                badge.toString(),
                color = Rahal.colors.onPanel,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.labelSmall,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .offset(x = Rahal.space.xs * -1, y = Rahal.space.xs)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.danger)
                    .padding(horizontal = Rahal.space.sm, vertical = 2.dp),
            )
        }
    }
}

@Composable
private fun GroupTitle(title: String) {
    Text(
        title,
        color = Rahal.colors.inkMuted,
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.titleSmall,
        modifier = Modifier.padding(horizontal = Rahal.space.xs),
    )
}

/** **رقمٌ من أرقام اليوم** — أيقونةٌ ملوّنةٌ ثمّ الرقمُ كبيراً ثمّ اسمُه. */
@Composable
private fun Tile(
    icon: Int,
    tint: Color,
    value: String,
    label: String,
    modifier: Modifier = Modifier,
    sub: String = "",
) {
    Row(
        modifier
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(36.dp)
                .clip(Rahal.shape.sm)
                .background(tint.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(20.dp))
        }
        Spacer(Modifier.width(Rahal.space.sm))
        Column(Modifier.weight(1f)) {
            Text(
                value,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleLarge,
                color = Rahal.colors.ink,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (sub.isNotBlank()) {
                Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.labelSmall, maxLines = 1)
            }
        }
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
    tint: Color = Rahal.colors.brand,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 60.dp)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(36.dp)
                .clip(Rahal.shape.sm)
                .background(tint.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(20.dp))
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
        // **ونقطةٌ حمراءُ لما ينقص** (ملاحظةُ المالك ٦د).
        if (warn) {
            Spacer(Modifier.width(Rahal.space.sm))
            Box(
                Modifier
                    .size(10.dp)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.danger),
            )
        }
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
    // **الرأسُ المشترك نفسُه** (`com.rahalgo.ui.PageHeader`) — لا نسخةٌ هنا ونسخةٌ في العروض.
    com.rahalgo.ui.PageHeader(title = title, onBack = onBack)
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
                    StorePage.Rate -> R.string.home_row_rate_short
                    StorePage.Prep -> R.string.home_row_prep
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
            StorePage.Address -> AddressEditor(vm)
            StorePage.Settlement -> Card {
                // **ويبدّلها بنفسه** (قرارُ المالك ٢٠٢٦-١٠-٠١) — **ويُسأل قبل التبديل**:
                // للطلبات الجديدة وحدَها، والمستحقُّ النقديُّ القائمُ يبقى حتّى يُصرف.
                var askMethod by rememberSaveable { mutableStateOf<String?>(null) }
                val pickMethod: (String) -> Unit = { m ->
                    if (!store.settlementConfirmed || store.settlementMethod != m) askMethod = m
                }
                // **البطاقتان نفساهما اللتان في المعالج** (ملاحظةُ المالك ٥).
                Text(stringResource(R.string.q_settlement), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
                Spacer(Modifier.height(Rahal.space.sm))
                IconChoicePair(
                    first = IconChoice(
                        icon = R.drawable.ic_banknotes,
                        title = stringResource(R.string.wz_pay_cash),
                        sub = stringResource(R.string.wz_settlement_cash_sub),
                        selected = store.settlementConfirmed && store.settlementMethod == "cash",
                    ) { pickMethod("cash") },
                    second = IconChoice(
                        icon = com.rahalgo.ui.R.drawable.ic_wallet,
                        title = stringResource(R.string.wz_pay_wallet),
                        sub = stringResource(R.string.wz_settlement_wallet_sub),
                        selected = store.settlementConfirmed && store.settlementMethod == "wallet",
                    ) { pickMethod("wallet") },
                    enabled = !vm.saving && !suspended,
                )
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
                Text(stringResource(R.string.wz_returns_hint), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
                Spacer(Modifier.height(Rahal.space.sm))
                IconChoicePair(
                    first = IconChoice(
                        icon = R.drawable.ic_return,
                        title = stringResource(R.string.wz_returns_yes),
                        sub = stringResource(R.string.wz_returns_yes_sub),
                        selected = store.returnsConfirmed && store.acceptsReturns,
                    ) { vm.setAcceptsReturns(true) },
                    second = IconChoice(
                        icon = R.drawable.ic_block,
                        title = stringResource(R.string.wz_returns_no),
                        sub = stringResource(R.string.wz_returns_no_sub),
                        selected = store.returnsConfirmed && !store.acceptsReturns,
                    ) { vm.setAcceptsReturns(false) },
                    enabled = !vm.saving && !suspended,
                )
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
