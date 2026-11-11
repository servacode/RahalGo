package com.rahalgo.rep.board

import android.app.Application
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.shared.rep.RepMe
import com.rahalgo.ui.ActionRow
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Bar
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.IconAction
import com.rahalgo.ui.IconBox
import com.rahalgo.ui.ListRow
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.RahalDivider
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionHeader
import com.rahalgo.ui.StatTile
import com.rahalgo.ui.StatusChip
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.apiError
import com.rahalgo.ui.money
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ المندوب — الشهريُّ أوّلاً ثمّ التراكميّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا الشهريُّ يسبق
 *
 * **رقمٌ إجماليٌّ وحدَه لا يقول أيَعمل هذا الشهرَ أم يعيش على ما مضى** —
 * **ومن جمع مئةَ متجرٍ في سنةٍ ولم يُسجّل واحداً هذا الشهرَ يرى «١٠٠»
 * فيطمئنّ وهو لا يعمل.**
 *
 * **وما يُسأل عنه يوميّاً يسبق ما يُسأل عنه مرّة** (وهو ترتيبُ الويب).
 *
 * # والهدفُ بشريطٍ لا برقم
 *
 * **«٣ من ٥» رقمان يُقرآن** — **والشريطُ يُقرأ بلمحةٍ قبل أن يُقرأ
 * الرقم.** ومن يفتح لوحتَه صباحاً يريد جواباً في ثانية.
 *
 * # وما ينتظر البتّ يُقال هنا
 *
 * **ومن سجّل ثلاثةً ولم يُقبلوا يظنّ أنّ عملَه ضاع** — **والرقمُ يقول
 * إنّه محفوظٌ ينتظر.**
 */
@Composable
fun BoardScreen(
    vm: BoardViewModel,
    /** **بطاقةُ الرصيد تفتح المحفظة** (٢٠٢٦-١٠-٢٧). */
    onOpenWallet: () -> Unit = {},
    /** **وزرُّ «وثّق حسابك» يفتح «حسابي»** (٢٠٢٦-١٠-٢٧). */
    onOpenAccount: () -> Unit = {},
    /** **بطاقةُ «بحاجة معلومات» تفتح «عملائي»** (٢٠٢٦-١٠-٢٨). */
    onOpenClients: () -> Unit = {},
    /** **بدّهن متابعة** (٢٠٢٦-١١-١١) — متاجرُه بأسباب المحرّك، وضغطةُ السطر تفتح العميل. */
    followUp: List<RepMerchant> = emptyList(),
    onOpenClient: (String) -> Unit = {},
    /** **«عرض الكل»** يفتح «عملائي» مفلترةً بـ«بدّه متابعة». */
    onShowAllFollowUp: () -> Unit = {},
    /** **اختصارُ «المكافآت»** (٢٠٢٦-١١-١١) — يفتح «أهدافي والمكافآت». */
    onOpenGoals: () -> Unit = {},
) {
    val me = vm.me
    if (me == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }
    val context = androidx.compose.ui.platform.LocalContext.current

    // **وسحبٌ للتحديث كما في تطبيق المتجر** (٢٠٢٦-١٠-٢٧).
    Refreshable(refreshing = false, onRefresh = { vm.load(force = true) }) {
    Screen {
        // **وبلا تلميحٍ تحت العنوان** — (طلبُ المالك ٢٠٢٦-٠٨-٣١: «أرقامك وهدف الشهر احذفها،
        // غيرُ ضروريّة»). **وتلميحٌ يعيد ما تحته لا يضيف.**
        ScreenTitle(stringResource(R.string.act_my_board))

        // ══════════════════════════════════════════════════════════════
        // **١ · هدفُ الشهر — البطاقةُ الأولى، والبسطُ متاجرُ فُتحت لا طلباتٌ سُلّمت**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٣١: «الهدفُ الشهريّ هو عددُ العملاء المسجَّلين».) **والمحرّكُ يرسل
        // `month_merchants` و`month_delivered` معاً — فالشاشةُ تختار، ومن اختار الخطأ لا يُخطئه بناء.**
        //
        // **وخطٌّ طويلٌ بمراحله الثلاث** (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «لازم يطول خط الهدف بحيث يعرف أنّه
        // حقّق أوّل هدف، وشو باقي لثاني هدف وثالث هدف»). **والعدّادُ شهريٌّ** بتوقيت دمشق.
        //
        // **ولا هدفَ يُخترع** (٢٠٢٦-١٠-٢٧): المراحلُ من المحرّك، ثمّ الهدفُ الشهريُّ إن كان، ثمّ لا شيء.
        val done = me.monthMerchants.toLong()
        val steps = vm.levels.ifEmpty {
            if (me.monthlyTarget > 0) {
                listOf(com.rahalgo.shared.model.TargetLevel(n = 1, target = me.monthlyTarget.toLong()))
            } else {
                emptyList()
            }
        }
        val top = steps.lastOrNull()?.target ?: 0L
        val allDone = top > 0 && done >= top
        RahalCard(padding = Rahal.space.lg) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                IconBox(com.rahalgo.ui.R.drawable.ic_star, Rahal.colors.brand)
                Spacer(Modifier.size(Rahal.space.md))
                Column(Modifier.weight(1f)) {
                    Text(stringResource(R.string.act_month_target), style = Rahal.type.sectionTitle, color = Rahal.colors.ink)
                    val next = steps.firstOrNull { done < it.target }
                    Text(
                        text = when {
                            steps.isEmpty() -> stringResource(R.string.bd_no_target)
                            allDone -> stringResource(R.string.bd_all_done)
                            next != null -> stringResource(R.string.bd_next, (next.target - done).toString(), next.n.toString())
                            else -> ""
                        },
                        color = if (allDone) Rahal.colors.success else Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                    )
                }
                if (steps.isNotEmpty()) {
                    // **«٣ / ٣٥» معزولةً يساريّاً** — فلا ينقلب ترتيبُ الرقمين داخل سطرٍ عربيّ.
                    Text(
                        text = "⁦$done / $top⁩",
                        color = if (allDone) Rahal.colors.success else Rahal.colors.brand,
                        style = Rahal.type.headline,
                    )
                }
            }
            if (steps.isNotEmpty()) {
                Spacer(Modifier.height(Rahal.space.lg))
                // **خطٌّ واحدٌ بقطعٍ على قدر المراحل** — قطعةُ كلِّ مرحلةٍ بطولِ ما بينها وبين سابقتها.
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs)) {
                    var from = 0L
                    steps.forEach { l ->
                        val span = (l.target - from).coerceAtLeast(1)
                        val filled = ((done - from).toFloat() / span).coerceIn(0f, 1f)
                        Box(Modifier.weight(span.toFloat())) {
                            Bar(ratio = filled, color = if (done >= l.target) Rahal.colors.success else Rahal.colors.brand)
                        }
                        from = l.target
                    }
                }
                Spacer(Modifier.height(Rahal.space.md))
                Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    steps.forEach { l -> LevelRow(l, done) }
                }
            }
            // **وما ينتظر شروطَه يُقال** (قرارُ المالك ٢٠٢٦-١٠-١٣) — وإلّا ظنّ أنّ متجراً سجّله ضاع من عدّاده.
            if (me.goalPending > 0) {
                Spacer(Modifier.height(Rahal.space.md))
                StatusChip(goalPendingText(me.goalPending.toInt()), StatusTone.Warning, icon = com.rahalgo.ui.R.drawable.ic_time)
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **٢ · ما ينتظره** — «بحاجة معلومات» عليه هو فتُضغط، و«بانتظار الموافقة» للعلم.
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يُعرضان صفراً**: رقمٌ صفريٌّ بجانب «بانتظار الموافقة» يُقرأ رفضا.
        if (me.needsInfoLeads > 0 || me.pendingLeads > 0) {
            Spacer(Modifier.height(Rahal.space.md))
            Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                if (me.needsInfoLeads > 0) {
                    // (٢٠٢٦-١٠-٢٨) **بطاقةٌ برتقاليّةٌ تُضغط فتفتح «عملائي»** حيث الطلبُ وملاحظتُه.
                    RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Warning, onClick = onOpenClients) {
                        ListRow(
                            title = stringResource(R.string.bd_needs_info),
                            icon = com.rahalgo.ui.R.drawable.ic_warning,
                            iconTint = Rahal.colors.accent,
                            titleColor = Rahal.colors.accentInk,
                            trailing = { StatusChip(me.needsInfoLeads.toString(), StatusTone.Warning) },
                            chevron = true,
                        )
                    }
                }
                if (me.pendingLeads > 0) {
                    RahalCard(variant = CardVariant.Outlined) {
                        ListRow(
                            title = stringResource(R.string.bd_pending),
                            icon = com.rahalgo.ui.R.drawable.ic_time,
                            trailing = { StatusChip(me.pendingLeads.toString(), StatusTone.Info) },
                        )
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **٣ · عمولاتي — والرصيدُ فيها يُفتح به المحفظة** (٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **والرصيدُ ليس من جنس الأرقام**: تلك حصيلةُ عمله، **وهذا ما في يده الآن** — فله بطاقتُه.
        SectionHeader(stringResource(R.string.bd_my_commissions))
        RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Info, padding = Rahal.space.lg) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(stringResource(R.string.bd_balance), color = Rahal.colors.inkMuted, style = Rahal.type.caption)
                    Text(money(me.balance), color = Rahal.colors.brand, style = Rahal.type.priceLarge)
                }
                IconBox(com.rahalgo.ui.R.drawable.ic_wallet, Rahal.colors.brand)
            }
            Spacer(Modifier.height(Rahal.space.md))
            ActionRow {
                StatTile(stringResource(R.string.bd_month_commissions), money(me.monthCommissions), valueColor = Rahal.colors.brand)
                StatTile(stringResource(R.string.bd_commissions), money(me.totalCommissions))
            }
            Spacer(Modifier.height(Rahal.space.md))
            RahalButton(onClick = onOpenWallet, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.bd_open_wallet))
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **٤ · بدّهن متابعة اليوم** (قرارُ المالك ٢٠٢٦-١١-١١) + اختصارُ المكافآت
        // ══════════════════════════════════════════════════════════════
        //
        // **كانت الاختصاراتُ الأربعةُ تكرّر الشريطَ السفليَّ وبطاقةَ المحفظة** — فحلّ محلَّها ما
        // يُفعل فعلاً: متاجرُه التي تنتظر كلمةً منه، **والمحرّكُ هو من يحسب السبب.**
        if (followUp.isNotEmpty()) {
            SectionHeader(
                title = stringResource(R.string.fu_title),
                action = stringResource(R.string.fu_all, followUp.size.toString()),
                onAction = onShowAllFollowUp,
            )
            RahalCard {
                followUp.take(FOLLOW_UP_SHOWN).forEachIndexed { idx, m ->
                    if (idx > 0) RahalDivider()
                    FollowUpRow(m, onClick = { onOpenClient(m.id) })
                }
            }
        }
        SectionHeader(stringResource(R.string.bd_quick))
        RahalCard(onClick = onOpenGoals) {
            ListRow(
                title = stringResource(com.rahalgo.ui.R.string.menu_rewards),
                icon = com.rahalgo.ui.R.drawable.ic_star,
                chevron = true,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **٥ · هذا الشهر ثمّ منذ بدأت** — مربّعاتٌ لا أسطر (طلبُ المالك ٢٠٢٦-٠٨-٣١).
        // ══════════════════════════════════════════════════════════════
        //
        // **والشهريُّ يسبق**: رقمٌ إجماليٌّ وحدَه لا يقول أيَعمل هذا الشهرَ أم يعيش على ما مضى.
        SectionHeader(stringResource(R.string.bd_this_month))
        ActionRow {
            StatTile(stringResource(R.string.bd_month_clients), me.monthMerchants.toString(), icon = com.rahalgo.ui.R.drawable.ic_store)
            StatTile(stringResource(R.string.bd_month_delivered), me.monthDelivered.toString(), icon = com.rahalgo.ui.R.drawable.ic_check)
        }
        SectionHeader(stringResource(R.string.bd_all_time))
        ActionRow {
            StatTile(stringResource(R.string.bd_clients), me.merchants.toString(), icon = com.rahalgo.ui.R.drawable.ic_store)
            StatTile(stringResource(R.string.bd_delivered), me.deliveredOrders.toString(), icon = com.rahalgo.ui.R.drawable.ic_check)
        }

        // ══════════════════════════════════════════════════════════════
        // **٦ · ورمزُه مقفلٌ حتّى يوثّق واتساب**
        // ══════════════════════════════════════════════════════════════
        //
        // **والإدارةُ تنسب إليه برمزه المتجرَ الذي تفتحه من لوحتها** — ومن لا قناةَ تواصلٍ موثّقةً
        // له لا يُتحقّق أنّه هو.
        SectionHeader(stringResource(R.string.bd_code))
        RahalCard {
            if (me.whatsappVerified && !me.inviteCode.isNullOrEmpty()) {
                // **والرمزُ يُنسخ بضغطة** (٢٠٢٦-١٠-٢٧) — يُرسله لمكتبٍ أو لصاحب متجر.
                val code = me.inviteCode.orEmpty()
                val copied = stringResource(R.string.bd_copied)
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Text(text = "⁦$code⁩", style = Rahal.type.headline, color = Rahal.colors.ink, modifier = Modifier.weight(1f))
                    IconAction(
                        icon = com.rahalgo.ui.R.drawable.ic_copy,
                        description = stringResource(R.string.bd_copy),
                        label = stringResource(R.string.bd_copy),
                    ) {
                        val cm = context.getSystemService(android.content.ClipboardManager::class.java)
                        cm?.setPrimaryClip(android.content.ClipData.newPlainText("code", code))
                        com.rahalgo.ui.Flash.ok(copied)
                    }
                }
            } else {
                // **و«غيرُ موثّق» تنبيهٌ برتقاليٌّ بزرّه لا خطأٌ أحمر** (٢٠٢٦-١٠-٢٧) — الأحمرُ يُقرأ عطباً.
                Row(verticalAlignment = Alignment.CenterVertically) {
                    IconBox(com.rahalgo.ui.R.drawable.ic_lock, Rahal.colors.accent)
                    Spacer(Modifier.size(Rahal.space.md))
                    Text(
                        text = stringResource(R.string.bd_code_locked),
                        color = Rahal.colors.accentInk,
                        style = Rahal.type.body,
                        modifier = Modifier.weight(1f),
                    )
                }
                Spacer(Modifier.height(Rahal.space.md))
                RahalOutlineButton(onClick = onOpenAccount, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.bd_verify))
                }
            }
        }
    }
    }
}

/** **أقصى ما يُعرض في «بدّهن متابعة»** — والباقي خلف «عرض الكل». */
private const val FOLLOW_UP_SHOWN = 5

/** **سطرُ متجرٍ يحتاج متابعةً** — اسمُه وأسبابُه بشاراتٍ، وضغطتُه تفتح تفصيلَه. */
@OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
@Composable
private fun FollowUpRow(m: RepMerchant, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Rahal.space.xs)) {
            Text(
                text = m.name,
                style = Rahal.type.bodyStrong,
                color = Rahal.colors.ink,
                maxLines = 1,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
            )
            androidx.compose.foundation.layout.FlowRow(
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.xs),
            ) {
                m.followUp.forEach { reason ->
                    StatusChip(
                        stringResource(
                            when (reason) {
                                "no_items" -> R.string.fu_no_items
                                "conditions" -> R.string.fu_conditions
                                else -> R.string.fu_no_orders
                            },
                        ),
                        StatusTone.Warning,
                    )
                }
            }
        }
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_chevron_end),
            contentDescription = null,
            tint = Rahal.colors.inkMuted,
            modifier = Modifier.size(Rahal.icon.sm),
        )
    }
}

/** **سطرُ مرحلة** — علامةٌ (تمّت أو لا) ثمّ «المرحلة ١ · الهدف ١٠» ثمّ الباقي أو المكافأة. */
@Composable
private fun LevelRow(l: com.rahalgo.shared.model.TargetLevel, done: Long) {
    val reached = done >= l.target
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painterResource(if (reached) com.rahalgo.ui.R.drawable.ic_check else com.rahalgo.ui.R.drawable.ic_star_outline),
            contentDescription = null,
            tint = if (reached) Rahal.colors.success else Rahal.colors.inkMuted,
            modifier = Modifier.size(Rahal.icon.sm),
        )
        Spacer(Modifier.size(Rahal.space.sm))
        Text(
            // **«الهدف ٥» لا «٥ عملاء»** — فلا تُقرأ «1 عملاء» (٢٠٢٦-١٠-٢٧).
            stringResource(R.string.bd_level, l.n.toString(), l.target.toString()),
            color = if (reached) Rahal.colors.success else Rahal.colors.ink,
            style = if (reached) Rahal.type.bodyStrong else Rahal.type.body,
            modifier = Modifier.weight(1f),
        )
        Text(
            if (reached) {
                // **والمكافأةُ بفواصلها كسائر المبالغ** — «+50,000 ل.س».
                if (l.reward > 0) stringResource(R.string.bd_level_done_reward, money(l.reward)) else stringResource(R.string.bd_level_done)
            } else {
                stringResource(R.string.bd_target_left, (l.target - done).toString())
            },
            color = if (reached) Rahal.colors.success else Rahal.colors.inkMuted,
            style = Rahal.type.caption,
        )
    }
}

/**
 * **العددُ يوافق المعدود** (مراجعةُ المالك ٢٠٢٦-١٠-٣٠: «6 متجر») — متجرٌ واحد ·
 * متجران · ٣–١٠ متاجر · ١١–٩٩ متجراً · والمئةُ وأخواتُها متجر.
 */
@Composable
private fun goalPendingText(n: Int): String {
    val r = n % 100
    return when {
        n == 1 -> stringResource(R.string.bd_goal_pending_one)
        n == 2 -> stringResource(R.string.bd_goal_pending_two)
        r in 3..10 -> stringResource(R.string.bd_goal_pending_few, n.toString())
        r in 11..99 -> stringResource(R.string.bd_goal_pending_many, n.toString())
        else -> stringResource(R.string.bd_goal_pending_other, n.toString())
    }
}

class BoardViewModel(app: Application) : AndroidViewModel(app) {

    private val api = RepApi(AppCore.get().api)

    var me by mutableStateOf<RepMe?>(null)
        private set

    /** **مراحلُ هدف الشهر الثلاث** — عددُ كلٍّ ومكافأتُها (`/rep/incentives`). */
    var levels by mutableStateOf<List<com.rahalgo.shared.model.TargetLevel>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    init {
        load()
        // **وما يتبدّل يُقرأ** — عميلٌ يُقبل فيتحرّك الهدف.
        viewModelScope.launch { Refresh.tick.drop(1).collect { load(force = true) } }
    }

    fun load(force: Boolean = false) {
        if (!force && me != null) return
        busy = true
        viewModelScope.launch {
            try {
                me = api.me()
                me?.let { com.rahalgo.ui.menu.ItemLimits.nameMax = it.itemNameMax }
                // **والمراحلُ أفضلُ جهد** — بلاها تبقى البطاقةُ على هدفٍ واحد.
                levels = runCatching { api.incentives().levels }
                    .getOrDefault(levels)
                    .filter { it.target > 0 }
                    .sortedBy { it.target }
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}
