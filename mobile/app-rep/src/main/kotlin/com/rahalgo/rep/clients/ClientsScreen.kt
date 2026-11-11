package com.rahalgo.rep.clients

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.Lead
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.ui.Bar
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.EmptyState
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.RahalDivider
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionHeader
import com.rahalgo.ui.StatusChip
import com.rahalgo.ui.StatusDot
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عملائي — من سجّلهم وما ناله عنهم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «ما في شي اسمه متاجري، اسمه عملائي».)
 *
 * # والمعلَّقون فوق
 *
 * **هم ما ينتظر عملا** — **والمقبولون يعملون وحدَهم.** ومن سجّل عميلاً
 * صباحاً يفتح الشاشةَ ليرى أقُبل، **لا ليقرأ عمولةَ متجرٍ سجّله الشهرَ
 * الماضي.**
 *
 * **والمرفوضُ ليس معلَّقاً** (٢٠٢٦-١٠-٢٧) — له قسمُه المطويّ تحتهم،
 * **وسببُ رفضه يُقال** (`decision_note`).
 *
 * # والتفعيلُ يُقال بعددين لا بكلمة
 *
 * **«قيد التفعيل» وحدَها لا تقول كم بقي** — **ومن لا يعرف كم بقي لا
 * يعرف أيلاحق صاحبَ المتجر أم ينتظر.**
 *
 * **والعمولةُ لا تجري حتّى يبلغ العددَ** (`sales.activation_orders`) —
 * **فرقمٌ ظاهرٌ خيرٌ من وعدٍ مبهم.**
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ClientsScreen(
    vm: ClientsViewModel,
    onOpenMenu: (String, String) -> Unit = { _, _ -> },
    onOpenItems: (String, String) -> Unit = { _, _ -> },
    /**
     * **وبابُ عروضه من بطاقته** — (طلبُ المالك ٢٠٢٦-٠٩-٣٠).
     *
     * **ويُمرَّر المتجرُ باسمه ومعرّفه معاً** — **وحارسُ `RO-01` يقوم
     * على أن يبقى الاسمُ فوق شاشة العروض**، فلا يُفتح بمعرّفٍ وحدَه.
     */
    onOpenOffers: (String, String) -> Unit = { _, _ -> },
    /** **زرُّ القائمة الفارغة** — ينقله إلى تبويب «إضافة عميل». */
    onAddClient: () -> Unit = {},
    /** **«بحاجة معلومات» تُصلَح من بطاقتها** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — تفتح نموذجَ الإضافة مملوءاً. */
    onEditLead: (Lead) -> Unit = {},
) {
    // **والتفصيلُ يغطّي القائمةَ** — **ولا صفحةٌ ثانيةٌ يخرج إليها
    // فيعود فلا يجد موضعَه.**
    if (vm.openId.isNotEmpty()) {
        // **وصفحةُ «ساعات الدوام» فوق التفصيل** (٢٠٢٦-١٠-٣٠) — والرجوعُ منها إليه.
        if (vm.hoursOpen) {
            BackHandler { vm.hoursOpen = false }
            ClientHours(vm)
            return
        }
        if (vm.sectionsOpen) {
            BackHandler { vm.sectionsOpen = false }
            ClientSections(vm)
            return
        }
        BackHandler { vm.closeDetail() }
        ClientDetail(vm, onOpenMenu, onOpenOffers, onOpenItems)
        return
    }

    val list = vm.merchants
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.load() }
        return
    }
    val context = LocalContext.current
    val pending = vm.pendingLeads
    val rejected = vm.rejectedLeads

    // **وسحبٌ للتحديث كما في تطبيق المتجر** (٢٠٢٦-١٠-٢٧) — **ومن سجّل
    // عميلاً صباحاً يسحب ليرى أقُبل** بلا خروجٍ وعودة.
    Refreshable(refreshing = false, onRefresh = { vm.load() }) {
        Screen {
            // **والعددُ بجانب الاسم — لا سطراً تحته ولا عنوانَ قسم** (طلبُ المالك ٢٠٢٦-٠٨-٣١:
            // «حطّ رقمَ عدد العملاء بجانب عملائي»).
            ScreenTitle(stringResource(R.string.nav_clients) + " (" + list.size + ")")

            // ══════════════════════════════════════════════════════════
            // **والمعلَّقون أوّلا** — ما ينتظر عملا، والعددُ يعدّهم وحدَهم.
            // ══════════════════════════════════════════════════════════
            if (pending.isNotEmpty()) {
                SectionHeader(stringResource(R.string.cl_pending, pending.size.toString()))
                Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    pending.forEach { LeadCard(it, onEdit = { onEditLead(it) }) }
                }
            }

            // **والمرفوضون في قسمهم مطويّين** — يُرَون بضغطة، ولا يزحمون.
            if (rejected.isNotEmpty()) {
                SectionHeader(
                    title = stringResource(R.string.cl_rejected_title, rejected.size.toString()),
                    action = stringResource(
                        if (vm.showRejected) R.string.cl_rejected_hide else R.string.cl_rejected_show,
                    ),
                    onAction = { vm.showRejected = !vm.showRejected },
                )
                if (vm.showRejected) {
                    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                        rejected.forEach { LeadCard(it, onEdit = { onEditLead(it) }) }
                    }
                }
            }

            if (list.isEmpty() && vm.leads.isEmpty()) {
                // **ولا متجرَ بعد — وزرٌّ إلى حيث يُسجَّل** (٢٠٢٦-١٠-٢٧).
                EmptyState(
                    title = stringResource(R.string.cl_none),
                    icon = com.rahalgo.ui.R.drawable.ic_store,
                    actionLabel = stringResource(R.string.nav_add_client),
                    onAction = onAddClient,
                )
                return@Screen
            }

            if (list.isNotEmpty()) {
                if (pending.isNotEmpty() || rejected.isNotEmpty()) {
                    SectionHeader(stringResource(R.string.nav_clients))
                }
                // ══════════════════════════════════════════════════════
                // **وبحثٌ وفلترةٌ لمن كثر عملاؤه** (٢٠٢٦-١٠-٢٧)
                // ══════════════════════════════════════════════════════
                //
                // **ولا تُعرض لمن له خمسةٌ أو أقلّ** — قائمةٌ قصيرةٌ تُرى كلُّها.
                if (list.size > 5) {
                    OutlinedTextField(
                        value = vm.query,
                        onValueChange = { vm.query = it },
                        placeholder = { Text(stringResource(R.string.cl_search), style = Rahal.type.body) },
                        leadingIcon = {
                            Icon(
                                painterResource(com.rahalgo.ui.R.drawable.ic_search),
                                contentDescription = null,
                                modifier = Modifier.size(Rahal.icon.sm),
                            )
                        },
                        singleLine = true,
                        textStyle = Rahal.type.body,
                        shape = Rahal.shape.md,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(Rahal.space.sm))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                        FilterOption(com.rahalgo.ui.R.string.wal_all, ClientFilter.All, vm)
                        FilterOption(R.string.cl_f_missing, ClientFilter.Missing, vm)
                        FilterOption(R.string.cl_f_no_items, ClientFilter.NoItems, vm)
                        FilterOption(R.string.cl_stopped, ClientFilter.Stopped, vm)
                    }
                    Spacer(Modifier.height(Rahal.space.xs))
                }
                // **وبلا حقلٍ ظاهرٍ لا بحثَ يُطبَّق** (٢٠٢٦-١٠-٢٨) — والنموذجُ يصفّرهما أيضاً.
                val shown = if (list.size > 5) vm.visible(list) else list
                if (shown.isEmpty()) {
                    EmptyState(
                        title = stringResource(R.string.cl_no_match),
                        icon = com.rahalgo.ui.R.drawable.ic_search,
                    )
                }
                Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    shown.forEach { m ->
                        MerchantCard(
                            m = m,
                            media = { path -> Backend.of(context).media(path) },
                            onOpen = { vm.openDetail(m.id) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun FilterOption(label: Int, f: ClientFilter, vm: ClientsViewModel) {
    FilterChip(
        selected = vm.filter == f,
        onClick = { vm.filter = f },
        label = { Text(stringResource(label), style = Rahal.type.label) },
        shape = Rahal.shape.pill,
    )
}

/**
 * **عميلٌ ينتظر البتّ — أو رُفض.**
 *
 * **ولا عمولةَ فيه ولا طلبات** — **ورقمٌ صفريٌّ بجانب اسمٍ يُقرأ فشلا**، وهو لم يبدأ بعد.
 */
@Composable
private fun LeadCard(lead: Lead, onEdit: () -> Unit) {
    val rejected = lead.status == "rejected"
    val needsInfo = lead.status == "needs_info"
    RahalCard(variant = CardVariant.Outlined) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = lead.storeName,
                style = Rahal.type.cardTitle,
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            StatusChip(
                stringResource(
                    when {
                        rejected -> R.string.cl_rejected
                        needsInfo -> R.string.cl_needs_info
                        else -> R.string.cl_waiting
                    },
                ),
                if (rejected) StatusTone.Danger else StatusTone.Warning,
            )
        }
        // **ورقمُ صاحبه بجانبه زرُّ اتّصال** — هو من يُلاحَق ليُكمل.
        if (lead.ownerName.isNotEmpty() || lead.phone.isNotEmpty()) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(
                        text = listOf(lead.ownerName, ltrPhone(lead.phone), lead.area)
                            .filter { it.isNotEmpty() }
                            .joinToString(" · "),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                    )
                    if (lead.createdAt.isNotEmpty()) {
                        Text(
                            text = stringResource(R.string.cl_lead_since, com.rahalgo.ui.dayText(lead.createdAt)),
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.meta,
                        )
                    }
                }
                if (lead.phone.isNotEmpty()) CallIcon(lead.phone)
            }
        } else if (lead.createdAt.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                text = stringResource(R.string.cl_lead_since, com.rahalgo.ui.dayText(lead.createdAt)),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.meta,
            )
        }
        // **وملاحظةُ الإدارة تُقال** — سببُ الرفض أو المعلومةُ المطلوبة (`decision_note`).
        if (lead.decisionNote.isNotEmpty() && (rejected || needsInfo)) {
            Spacer(Modifier.height(Rahal.space.sm))
            Text(
                text = stringResource(R.string.cl_admin_note, lead.decisionNote),
                color = if (rejected) Rahal.colors.danger else Rahal.colors.accentInk,
                style = Rahal.type.bodyStrong,
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.sm)
                    .background((if (rejected) Rahal.colors.danger else Rahal.colors.accent).copy(alpha = 0.08f))
                    .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
            )
        }
        // **و«بحاجة معلومات» تُصلَح من بطاقتها** (قرارُ المالك ٢٠٢٦-١٠-٢٧).
        if (needsInfo) {
            Spacer(Modifier.height(Rahal.space.md))
            RahalButton(onClick = onEdit, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.cl_fix_lead))
            }
        }
    }
}

/** **ما ينقص المتجرَ ليُحسب في هدفه** — بنداً بنداً، وفارغٌ إن لم ينقصه شيء. */
@Composable
internal fun missingFor(m: RepMerchant): List<String> = buildList {
    if (m.goalState != "pending") return@buildList
    if (!m.hasHours) add(stringResource(R.string.cl_miss_hours))
    if (!m.hasLocation) add(stringResource(R.string.cl_miss_location))
    if (m.photoItems < m.minItems) {
        add(stringResource(R.string.cl_miss_items, (m.minItems - m.photoItems).toString()))
    }
}

/** **شريطُ ما ينقص** — برتقاليٌّ فوق الأزرار، **فهو أهمُّ ما يُفعل في المتجر.** */
@Composable
internal fun MissingStrip(missing: List<String>, onClick: (() -> Unit)? = null) {
    if (missing.isEmpty()) return
    Spacer(Modifier.height(Rahal.space.md))
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sm)
            .background(Rahal.colors.warnTint)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_warning),
            contentDescription = null,
            tint = Rahal.colors.accentInk,
            modifier = Modifier.size(Rahal.icon.xs),
        )
        Spacer(Modifier.size(Rahal.space.sm))
        Text(
            text = stringResource(R.string.cl_goal_missing, missing.joinToString(" · ")),
            color = Rahal.colors.accentInk,
            style = Rahal.type.bodyStrong,
            modifier = Modifier.weight(1f),
        )
    }
}

/**
 * **بطاقةُ عميلٍ في القائمة** (قرارُ المالك ٢٠٢٦-١١-١١).
 *
 * «طلبات سلمت وعمولتك لازم يكون مربعات، وبلا اصناف ومحسوب في هدفك لازم بالاعلى، والاتصال
 * والواتس يكون بالكرت احسن مايفوت ع صفحة العميل ليبعثله ع واتس او يتصل بيه» — فالشاراتُ في
 * رأس البطاقة، وأيقونتا الواتساب والاتصال بجانب الاسم، ورقما «سُلّم» و«عمولتي» مربّعان تحت.
 * **والضغطُ على البطاقة يفتح تفصيلَها كما كان.**
 */
@Composable
private fun MerchantCard(
    m: RepMerchant,
    media: (String?) -> String?,
    onOpen: () -> Unit,
) {
    val active = m.status == "active"
    val context = androidx.compose.ui.platform.LocalContext.current
    RahalCard(onClick = onOpen) {
        // **الشاراتُ أوّلاً** — ما ينقصه ليُحسب، أو أنّه محسوب، و«بلا أصناف».
        val missing = missingFor(m)
        val counted = missing.isEmpty() && m.goalState == "counted"
        if (missing.isNotEmpty() || counted || m.itemsCount == 0) {
            Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                when {
                    missing.isNotEmpty() ->
                        StatusChip(stringResource(R.string.cl_missing_badge, missing.size.toString()), StatusTone.Warning)
                    counted ->
                        StatusChip(stringResource(R.string.cl_goal_counted), StatusTone.Success)
                }
                if (m.itemsCount == 0) {
                    StatusChip(stringResource(R.string.cl_f_no_items), StatusTone.Warning, icon = com.rahalgo.ui.R.drawable.ic_warning)
                }
            }
            Spacer(Modifier.height(Rahal.space.sm))
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(m.logoThumbUrl),
                name = m.name,
                modifier = Modifier
                    .size(Rahal.size.avatarMd)
                    .clip(Rahal.shape.md),
            )
            Spacer(Modifier.size(Rahal.space.md))
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Rahal.space.xxs)) {
                Text(
                    text = m.name,
                    style = Rahal.type.cardTitle,
                    color = Rahal.colors.ink,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                // **اسمُ صاحبه لا رقمُه** (قرارُ المالك ٢٠٢٦-١١-١١).
                m.ownerName?.takeIf { it.isNotBlank() }?.let {
                    Text(
                        text = it,
                        color = Rahal.colors.ink,
                        style = Rahal.type.caption,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    StatusDot(
                        stringResource(if (active) R.string.cl_working else R.string.cl_stopped),
                        if (active) StatusTone.Success else StatusTone.Neutral,
                    )
                    if (m.categoryName.isNotEmpty()) {
                        Text(
                            text = m.categoryName,
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.meta,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
            // **الواتساب والاتصال على البطاقة نفسِها** — لا يفتح التفصيلَ ليراسل صاحبَه.
            val phone = m.ownerPhone?.takeIf { it.isNotBlank() }
            if (phone != null) {
                val wa = com.rahalgo.ui.waMeDigits(phone)
                if (wa.isNotEmpty()) {
                    com.rahalgo.ui.IconAction(
                        icon = com.rahalgo.ui.R.drawable.ic_whatsapp,
                        description = stringResource(com.rahalgo.ui.R.string.contact_whatsapp),
                        tint = Rahal.colors.success,
                    ) { openWhatsapp(context, wa) }
                    // **وفراغٌ بين الزرّين** (ملاحظةُ المالك ٢٠٢٦-١١-١١: «لاصقين ببعض»).
                    Spacer(Modifier.size(Rahal.space.sm))
                }
                com.rahalgo.ui.IconAction(
                    icon = com.rahalgo.ui.R.drawable.ic_phone,
                    description = stringResource(com.rahalgo.ui.R.string.contact_call),
                ) { dialNumber(context, phone) }
            }
        }

        // **والتفعيلُ بعددين لا بكلمة** — كم أنجز من الطلبات التي تُصرف بعدها عمولتُه.
        val needed = m.activationNeeded.toInt()
        if (needed > 1 && m.activationDone < needed) {
            Spacer(Modifier.height(Rahal.space.sm))
            Text(
                text = stringResource(R.string.cl_activation, m.activationDone.toString(), needed.toString()),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
            Spacer(Modifier.height(Rahal.space.xs))
            Bar(ratio = m.activationDone.toFloat() / needed.toFloat(), color = Rahal.colors.accent)
        }

        // **أربعةُ مربّعاتٍ في سطرين** (قرارُ المالك ٢٠٢٦-١١-١١) — «سُلّم» و«عمولتي»، ثمّ «شقد النا نسبة
        // عمولة عنده» و«عدد الاصناف الي منزلها». `StatTile` المشترك.
        Spacer(Modifier.height(Rahal.space.md))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            com.rahalgo.ui.StatTile(
                label = stringResource(R.string.cl_s_orders),
                value = m.deliveredOrders.toString(),
                modifier = Modifier.weight(1f),
            )
            com.rahalgo.ui.StatTile(
                label = stringResource(R.string.cl_s_commission),
                value = money(m.myCommission),
                valueColor = Rahal.colors.brand,
                modifier = Modifier.weight(1f),
            )
        }
        Spacer(Modifier.height(Rahal.space.sm))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            com.rahalgo.ui.StatTile(
                label = stringResource(R.string.cd_commission_tile),
                value = pctLabel(m.storeCommissionPercent),
                modifier = Modifier.weight(1f),
            )
            com.rahalgo.ui.StatTile(
                label = stringResource(R.string.cl_s_items),
                value = m.itemsCount.toString(),
                valueColor = if (m.itemsCount == 0) Rahal.colors.accentInk else Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
        }
    }
}
