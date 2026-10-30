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
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.Lead
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.ui.Bar
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
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
            // ══════════════════════════════════════════════════════════
            // **والعددُ بجانب الاسم — لا سطراً تحته ولا عنوانَ قسم**
            // ══════════════════════════════════════════════════════════
            //
            // **(طلبُ المالك ٢٠٢٦-٠٨-٣١:** «من سجّلتهم وما نلتَه عنهم ما
            // تلزم · **يعملون (٤) أيضاً احذفها** · حطّ رقمَ عدد العملاء
            // بجانب عملائي».)
            ScreenTitle(stringResource(R.string.nav_clients) + " (" + list.size + ")")

            // ══════════════════════════════════════════════════════════
            // **والمعلَّقون أوّلا** — ما ينتظر عملا
            // ══════════════════════════════════════════════════════════
            //
            // **والعددُ يعدّهم وحدَهم** — **وكان يعدّ المرفوضين معهم**، فيرى
            // «ينتظرون (٣)» وواحدٌ منهم رُفض من شهر.
            if (pending.isNotEmpty()) {
                SectionTitle(stringResource(R.string.cl_pending, pending.size.toString()))
                pending.forEach { LeadCard(it, onEdit = { onEditLead(it) }) }
            }

            // **والمرفوضون في قسمهم مطويّين** — يُرَون بضغطة، ولا يزحمون.
            if (rejected.isNotEmpty()) {
                SectionTitle(stringResource(R.string.cl_rejected_title, rejected.size.toString()))
                RahalTextButton(onClick = { vm.showRejected = !vm.showRejected }) {
                    Text(
                        stringResource(
                            if (vm.showRejected) R.string.cl_rejected_hide else R.string.cl_rejected_show,
                        ),
                        color = Rahal.colors.brand,
                    )
                }
                if (vm.showRejected) rejected.forEach { LeadCard(it, onEdit = { onEditLead(it) }) }
            }
            if (pending.isNotEmpty() || rejected.isNotEmpty()) {
                Spacer(Modifier.height(14.dp))
                HorizontalDivider()
            }

            if (list.isEmpty() && vm.leads.isEmpty()) {
                // **ولا متجرَ بعد — وزرٌّ إلى حيث يُسجَّل** (٢٠٢٦-١٠-٢٧).
                // **وكان التلميحُ يقول «زرّ الإضافة في الأعلى»** — والإضافةُ
                // تبويبٌ تحت، **وجملتان تعيدان الشيءَ نفسَه.**
                Empty(
                    text = stringResource(R.string.cl_none),
                    actionLabel = stringResource(R.string.nav_add_client),
                    onAction = onAddClient,
                )
                return@Screen
            }

            if (list.isNotEmpty()) {
                // ══════════════════════════════════════════════════════
                // **وبحثٌ وفلترةٌ لمن كثر عملاؤه** (٢٠٢٦-١٠-٢٧)
                // ══════════════════════════════════════════════════════
                //
                // **ومن له ثلاثون عميلاً يمرّر طويلاً ليبلغ واحدا.** **ولا
                // تُعرض لمن له خمسةٌ أو أقلّ** — قائمةٌ قصيرةٌ تُرى كلُّها.
                if (list.size > 5) {
                    Spacer(Modifier.height(10.dp))
                    OutlinedTextField(
                        value = vm.query,
                        onValueChange = { vm.query = it },
                        placeholder = { Text(stringResource(R.string.cl_search)) },
                        leadingIcon = {
                            Icon(
                                painterResource(com.rahalgo.ui.R.drawable.ic_search),
                                contentDescription = null,
                                modifier = Modifier.size(20.dp),
                            )
                        },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(6.dp))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        FilterOption(com.rahalgo.ui.R.string.wal_all, ClientFilter.All, vm)
                        FilterOption(R.string.cl_f_missing, ClientFilter.Missing, vm)
                        FilterOption(R.string.cl_f_no_items, ClientFilter.NoItems, vm)
                        FilterOption(R.string.cl_stopped, ClientFilter.Stopped, vm)
                    }
                }
                // **وبلا حقلٍ ظاهرٍ لا بحثَ يُطبَّق** (٢٠٢٦-١٠-٢٨) — والنموذجُ يصفّرهما أيضاً.
                val shown = if (list.size > 5) vm.visible(list) else list
                if (shown.isEmpty()) {
                    Empty(stringResource(R.string.cl_no_match))
                }
                shown.forEachIndexed { i, m ->
                    MerchantCard(
                        m = m,
                        // **وكلُّ متجرٍ بلونٍ خفيفٍ يميّزه عن جاره** (قرارُ المالك ٢٠٢٦-١٠-٣٠: «فايتين ببعض»).
                        tone = cardTones()[i % 3],
                        media = { path -> Backend.of(context).media(path) },
                        onOpen = { vm.openDetail(m.id) },
                        onAddItem = { onOpenMenu(m.id, m.name) },
                        onItems = { onOpenItems(m.id, m.name) },
                        onMakeOffer = { onOpenOffers(m.id, m.name) },
                    )
                }
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
private fun FilterOption(label: Int, f: ClientFilter, vm: ClientsViewModel) {
    FilterChip(
        selected = vm.filter == f,
        onClick = { vm.filter = f },
        label = { Text(stringResource(label)) },
    )
}

/**
 * **عميلٌ ينتظر البتّ — أو رُفض.**
 *
 * **ولا عمولةَ فيه ولا طلبات** — **ورقمٌ صفريٌّ بجانب اسمٍ يُقرأ فشلا**،
 * وهو لم يبدأ بعد.
 */
@Composable
private fun LeadCard(lead: Lead, onEdit: () -> Unit) {
    val rejected = lead.status == "rejected"
    val needsInfo = lead.status == "needs_info"
    Spacer(Modifier.height(8.dp))
    Card {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = lead.storeName,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.weight(1f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            // **و«الإدارة بدها معلومات» حالٌ ثالثة** — **وكانت تُقرأ
            // «بانتظار الموافقة» بلونٍ عاديّ**، فلا يعرف أنّ الدورَ عليه.
            Chip(
                stringResource(
                    when {
                        rejected -> R.string.cl_rejected
                        needsInfo -> R.string.cl_needs_info
                        else -> R.string.cl_waiting
                    },
                ),
                if (rejected) Rahal.colors.danger else Rahal.colors.accent,
            )
        }
        if (lead.ownerName.isNotEmpty() || lead.phone.isNotEmpty()) {
            Spacer(Modifier.height(4.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = listOf(lead.ownerName, ltrPhone(lead.phone), lead.area)
                        .filter { it.isNotEmpty() }
                        .joinToString(" · "),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.weight(1f),
                )
                // **واتّصالٌ بضغطة** — يلاحق صاحبَه وهو في الطريق.
                if (lead.phone.isNotEmpty()) CallIcon(lead.phone)
            }
        }
        if (lead.createdAt.isNotEmpty()) {
            Text(
                text = stringResource(R.string.cl_lead_since, com.rahalgo.ui.dayText(lead.createdAt)),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
        }
        // **وسببُ الرفض وما ينقص يُقالان** — **ومن رُفض عميلُه بلا سببٍ يُعيد
        // تسجيلَه بالخطأ نفسِه.** وهي ملاحظةُ الإدارة (`decision_note`) لا ما
        // كتبه المندوبُ نفسُه (`note`) — كانت تعرض الثانيةَ فلا يرى ما طُلب.
        if (lead.decisionNote.isNotEmpty() && (lead.status == "rejected" || needsInfo)) {
            Spacer(Modifier.height(6.dp))
            Text(
                text = stringResource(R.string.cl_admin_note, lead.decisionNote),
                color = if (rejected) Rahal.colors.danger else Rahal.colors.accent,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // **و«بحاجة معلومات» تُصلَح من هنا** (قرارُ المالك ٢٠٢٦-١٠-٢٧).
        if (needsInfo) {
            Spacer(Modifier.height(8.dp))
            RahalButton(onClick = onEdit, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.cl_fix_lead))
            }
        }
    }
}

/**
 * **رقمٌ صغيرٌ بكلمته** — أربعةٌ في صفٍّ واحدٍ من البطاقة.
 *
 * **وليس `StatBox`**: أربعةُ مربّعاتٍ بحشوته في عرض الهاتف تقصّ الرقم،
 * **وهذا أخفُّ** — رقمٌ ثقيلٌ وكلمةٌ صغيرةٌ تحته.
 */
@Composable
private fun MiniStat(label: String, value: String, color: Color, modifier: Modifier) {
    Column(
        modifier
            .clip(Rahal.shape.sm)
            .background(Rahal.colors.inkMuted.copy(alpha = 0.07f))
            .padding(horizontal = 4.dp, vertical = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            value,
            color = color,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            textAlign = TextAlign.Center,
        )
        Text(
            label,
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.labelSmall,
            // **سطران لا قصّ** (مراجعةُ المالك ٢٠٢٦-١٠-٣٠: «طلبات سلمت» كانت تنقطع).
            minLines = 2,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            textAlign = TextAlign.Center,
        )
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
    Spacer(Modifier.height(8.dp))
    Text(
        text = stringResource(R.string.cl_goal_missing, missing.joinToString(" · ")),
        color = Rahal.colors.accent,
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.bodySmall,
        modifier = Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sm)
            .background(Rahal.colors.warnTint)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 10.dp, vertical = 8.dp),
    )
}

/**
 * **عميلٌ مقبولٌ يعمل** — بعمولته وحال تفعيله.
 *
 * # وأخفُّ ممّا كان (٢٠٢٦-١٠-٢٧)
 *
 * **كانت البطاقةُ صورةً واسماً وشارتين وسطرَ نقصٍ وفاصلاً وصفّين من
 * أربعة مربّعاتٍ وزرّين وشريطاً** — **وأهمُّ ما فيها (ما ينقص المتجرَ
 * ليُحسب) ضائعٌ في وسطها.** **فصارت**: الاسمُ وشاراته، ثمّ شريطُ التفعيل
 * إن كان، **ثمّ صفٌّ واحدٌ بأربعة أرقام**، **ثمّ ما ينقص بشريطٍ برتقاليّ
 * فوق الزرّين.** والأرقامُ والزرّان كما هي.
 */
@Composable
private fun MerchantCard(
    m: RepMerchant,
    media: (String?) -> String?,
    onOpen: () -> Unit,
    onAddItem: () -> Unit,
    onMakeOffer: () -> Unit,
    /** **«الأصناف» — قائمةُ المتجر بأدواتها** (٢٠٢٦-١٠-٣٠): «إضافة صنف» صار يفتح صنفاً جديداً مباشرة. */
    onItems: () -> Unit,
    tone: androidx.compose.ui.graphics.Color? = null,
) {
    Spacer(Modifier.height(Rahal.space.md))
    // **وضغطُ البطاقة يفتح تفصيلَه** — (طلبُ المالك
    // ٢٠٢٦-٠٨-١٤: «إذا ضغطتُ على اسم العميل يجب أن أعرف كلَّ
    // التفاصيل التي تخصّني كمندوب»).
    Card(modifier = Modifier.clickable(onClick = onOpen), tone = tone) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(m.logoThumbUrl),
                name = m.name,
                modifier = Modifier
                    .size(46.dp)
                    .clip(Rahal.shape.sm),
            )
            Spacer(Modifier.size(10.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = m.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (m.categoryName.isNotEmpty()) {
                    Text(
                        text = m.categoryName,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }

        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Chip(
                stringResource(
                    if (m.status == "active") R.string.cl_working else R.string.cl_stopped,
                ),
                if (m.status == "active") Rahal.colors.success else Rahal.colors.inkMuted,
            )
            // **أمحسوبٌ في هدفه أم ينتظر** (قرارُ المالك ٢٠٢٦-١٠-١٣) — **والهدفُ يتبع
            // المندوبَ الحاليّ** (قرارُ المالك ٢٠٢٦-١٠-٢٠): المتجرُ المنقولُ إليه يُحسب له،
            // والمنقولُ عنه لا يُحسب لمن فتحه.
            when (m.goalState) {
                "counted" -> Chip(stringResource(R.string.cl_goal_counted), Rahal.colors.success)
                "pending" -> Chip(stringResource(R.string.cl_goal_pending), Rahal.colors.accent)
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وشريطُ التفعيل حين لا يزال دونه — تحت الشارات**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يُعرض لمن فعّل** — شريطٌ ممتلئٌ دائماً يُقرأ زينةً لا خبرا.
        val needed = m.activationNeeded.toInt()
        if (needed > 1 && m.activationDone < needed) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(
                    R.string.cl_activation,
                    m.activationDone.toString(),
                    needed.toString(),
                ),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            Spacer(Modifier.height(4.dp))
            Bar(
                ratio = m.activationDone.toFloat() / needed.toFloat(),
                color = Rahal.colors.accent,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **وأربعةُ أرقامٍ في صفٍّ واحد**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «نضيف مربّعاً لعدد العناصر المعروضة
        //  بكلّ متجر… لنعرف كلّ متجر كم صنف عارض بدون ما نفوت عليه».)
        //
        // **والصفرُ في الأصناف يُصبَغ تنبيهاً لا رماديّاً**: **متجرٌ بلا
        // صنفٍ معروضٍ لا يظهر للزبون أصلاً** — فهو عملٌ ناقصٌ لا رقمٌ
        // يُقرأ، **ومن رآه رماديّاً مرّ عليه.**
        Spacer(Modifier.height(10.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            MiniStat(
                label = stringResource(R.string.cl_s_orders),
                value = m.deliveredOrders.toString(),
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
            // **والمالُ أعرضُ من العدد** — فلا يُقصّ «125,000 ل.س».
            MiniStat(
                label = stringResource(R.string.cl_s_commission),
                value = money(m.myCommission),
                color = Rahal.colors.brand,
                modifier = Modifier.weight(1.6f),
            )
            MiniStat(
                label = stringResource(R.string.cl_s_items),
                value = m.itemsCount.toString(),
                color = if (m.itemsCount == 0) Rahal.colors.accent else Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
            MiniStat(
                label = stringResource(R.string.cl_s_offers),
                value = m.offersCount.toString(),
                color = if (m.offersCount > 0) Rahal.colors.success else Rahal.colors.inkMuted,
                modifier = Modifier.weight(1f),
            )
        }

        // **وما ينقصه بنداً بنداً — فوق الزرّين** — فيعرف المندوبُ ما يُكمله
        // وهو واقفٌ في المتجر.
        MissingStrip(missingFor(m))

        // **واتّصالٌ وواتساب من البطاقة** — بلا فتح العميل (مراجعةُ المالك ٢٠٢٦-١٠-٣٠).
        m.ownerPhone?.takeIf { it.isNotBlank() }?.let {
            Spacer(Modifier.height(8.dp))
            ContactButtons(it)
        }

        // ══════════════════════════════════════════════════════════════
        // **وفعلاه من بطاقته — لا من داخله**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «نضيف نفس الزرّ إضافة صنف أو أنشئ
        //  عرض بنفس البطاقة، هيك نسهّل عالمندوب».)
        //
        // **وهما الزرّان نفسُهما في صفحة الأصناف** — نصّاً وترتيباً،
        // **فلا يتعلّم المندوبُ بابين لفعلٍ واحد.**
        Spacer(Modifier.height(8.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            RahalOutlineButton(
                onClick = onItems,
                modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
            ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_items)) }
            RahalButton(
                onClick = onAddItem,
                modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
            ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_add_item)) }

            RahalOutlineButton(
                onClick = onMakeOffer,
                modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
            ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_make_offer)) }
        }
    }
}

/** **ثلاثةُ ألوانٍ هادئةٍ تتناوب** — بطاقتان متجاورتان لا تتشابهان. */
@Composable
private fun cardTones() = listOf(Rahal.colors.brand, Rahal.colors.accent, Rahal.colors.success)
