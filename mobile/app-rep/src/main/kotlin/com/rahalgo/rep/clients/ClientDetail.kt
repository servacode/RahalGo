package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepOrderLine
import com.rahalgo.ui.ActionRow
import com.rahalgo.ui.AvatarWithEdit
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.EmptyState
import com.rahalgo.ui.IconAction
import com.rahalgo.ui.IconTile
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.PageHeader
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.RahalDivider
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.SectionHeader
import com.rahalgo.ui.StatTile
import com.rahalgo.ui.StatusChip
import com.rahalgo.ui.StatusDot
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تفصيلُ عميل — شفافيّةُ العمولة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٤: «إذا ضغطتُ على اسم العميل يجب أن أعرف كلَّ
 *  التفاصيل التي تخصّني كمندوبٍ عن العميل».)
 *
 * # ولماذا كلُّ طلبٍ بسطره
 *
 * **المندوبُ يقبض نسبةً من طلبات متجرٍ جلبه، ولم يكن يرى من أين جاءت** —
 * رقمٌ مجمَّعٌ في لوحته وكفى.
 *
 * **وثقةُ من يعمل بالعمولة تُبنى على أن يُراجع بنفسه لا على أن
 * يُصدّق.** فهنا كلُّ طلب: رقمُه وتاريخُه وحالُه وقيمتُه ونصيبُه منه.
 *
 * # والملغى يبقى في القائمة
 *
 * **لا يُحذف سطرُه**: **إخفاؤه يجعل المجموعَ لا يُطابَق، وهو نقيضُ
 * الشفافيّة.** ونصيبُه يُعرض مشطوباً — **ما كان سيُحتسب لولا الإلغاء.**
 *
 * # ولا تُحسب المجاميعُ هنا
 *
 * **تُقرأ من الدفتر** — **وطلبٌ رُجّع تُسحب عمولتُه بقيدٍ معاكس**،
 * وحسبةٌ في الجهاز لا تعرف ذلك **فتُظهر للمندوب مالاً ليس له.**
 *
 * # والشكلُ من العُدّة المشتركة (٢٠٢٦-١١-١١)
 *
 * `AvatarWithEdit` · `IconAction` · `IconTile` · `StatusDot` · `ActionRow` · `StatTile` — **وكانت
 * نسخاً محلّيّةً في `ClientCard.kt`** فنُقلت إلى `ui` وحُذفت من هنا، وقراراتُ المالك في البطاقة باقية.
 */
@Composable
fun ClientDetail(
    vm: ClientsViewModel,
    onOpenMenu: (String, String) -> Unit,
    onOpenOffers: (String, String) -> Unit = { _, _ -> },
    onOpenItems: (String, String) -> Unit = { _, _ -> },
) {
    val d = vm.detail
    if (d == null) {
        LoadState(vm.busy, vm.error) { vm.openDetail(vm.openId) }
        return
    }
    val context = LocalContext.current
    val pickLogo = com.rahalgo.ui.rememberImagePicker(crop = true) { bytes -> vm.setLogo(bytes) }
    val head = d.merchant
    val sum = d.summary
    // **وبطاقتُه في القائمة تعرف ما ينقصه** — والتفصيلُ لا يحمله، فيُقرأ منها.
    val row = vm.merchants.orEmpty().firstOrNull { it.id == vm.openId }

    // **وسحبٌ للتحديث** (٢٠٢٦-١٠-٢٧) — يرى أنزلت عمولةٌ بلا خروجٍ وعودة.
    Refreshable(refreshing = false, onRefresh = { vm.refreshDetail() }) {
    Screen {
        // **سهمُ رجوعٍ وعنوان** — كما في صفحات المتجر (مراجعةُ المالك ٢٠٢٦-١٠-٣٠).
        PageHeader(title = stringResource(R.string.nav_clients), onBack = { vm.closeDetail() })
        Spacer(Modifier.height(Rahal.space.sm))
        // ══════════════════════════════════════════════════════════════
        // **ترويسةُ العميل — شعارُه واسمُه فوق، واسمُ صاحبه لا رقمُه** (قرارُ المالك ٢٠٢٦-١١-١١)
        // ══════════════════════════════════════════════════════════════
        //
        // **والرقمُ لا يُعرض** — ويبقى خلف أيقونتي الواتساب والاتّصال.
        val ownerPhone = head.ownerPhone?.takeIf { it.isNotBlank() }
        val wa = com.rahalgo.ui.waMeDigits(ownerPhone.orEmpty())
        val active = head.status == "active"
        RahalCard(padding = Rahal.space.lg) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // **والشعارُ عليه كاميرا** تفتح «كاميرا أم معرض» ثمّ القصّ — بدل زرّ «أضف شعار».
                AvatarWithEdit(
                    url = Backend.of(context).media(head.logoThumbUrl),
                    name = head.name,
                    description = stringResource(R.string.cd_logo_camera),
                    enabled = !vm.busy,
                    onPick = { pickLogo() },
                )
                Spacer(Modifier.size(Rahal.space.md))
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Rahal.space.xxs)) {
                    Text(
                        text = head.name,
                        style = Rahal.type.cardTitle,
                        color = Rahal.colors.ink,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    // **اسمُ صاحب المتجر** — الاسمُ هو المطلوب لا رقمُه.
                    head.ownerName?.takeIf { it.isNotBlank() }?.let {
                        Text(
                            text = stringResource(R.string.cd_owner_name, it),
                            color = Rahal.colors.ink,
                            style = Rahal.type.body,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    if (head.categoryName.isNotEmpty()) {
                        Text(text = head.categoryName, color = Rahal.colors.inkMuted, style = Rahal.type.caption)
                    }
                }
            }
            // **وحالُه وتاريخُ انضمامه، وأيقونتا التواصل في الطرف.**
            Spacer(Modifier.height(Rahal.space.md))
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Rahal.space.xxs)) {
                    StatusDot(
                        stringResource(if (active) R.string.cl_working else R.string.cl_stopped),
                        if (active) StatusTone.Success else StatusTone.Neutral,
                    )
                    if (head.joinedAt.isNotEmpty()) {
                        Text(
                            stringResource(R.string.cd_joined_short, longDate(head.joinedAt)),
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.meta,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                if (ownerPhone != null) {
                    if (wa.isNotEmpty()) {
                        IconAction(
                            icon = com.rahalgo.ui.R.drawable.ic_whatsapp,
                            description = stringResource(com.rahalgo.ui.R.string.contact_whatsapp),
                            tint = Rahal.colors.success,
                        ) { openWhatsapp(context, wa) }
                        // **وفراغٌ بين الزرّين** (ملاحظةُ المالك ٢٠٢٦-١١-١١: «لاصقين ببعض»).
                        Spacer(Modifier.size(Rahal.space.sm))
                    }
                    IconAction(
                        icon = com.rahalgo.ui.R.drawable.ic_phone,
                        description = stringResource(com.rahalgo.ui.R.string.contact_call),
                    ) { dialNumber(context, ownerPhone) }
                }
            }
            // **وما ينقصه ليُحسب — شريطٌ بنبرةٍ واضحة** (٢٠٢٦-١٠-٢٧)، ونقصُ الدوام يُضغط فيفتح صفحتَه (٢٠٢٦-١٠-٣٠).
            row?.let { MissingStrip(missingFor(it), onClick = if (!it.hasHours) ({ vm.openHours() }) else null) }

            // ══════════════════════════════════════════════════════════
            // **ثلاثةُ مربّعاتِ وظائف** — دوامٌ · عمولةٌ · أقسام، لكلٍّ أيقونتُه وقيمتُه وضغطتُه.
            // ══════════════════════════════════════════════════════════
            Spacer(Modifier.height(Rahal.space.md))
            RahalDivider()
            Spacer(Modifier.height(Rahal.space.md))
            val storeId = vm.openId
            ActionRow {
                HoursTile(vm)
                // **والحفظُ في الـvm لا في المربّع** — فيصل ردُّه التفصيلَ ولو فتح «ساعات الدوام» أو «الأقسام»
                // أو «الأصناف» قبل أن يصل، وتبقى النسبةُ إن رجع. (قرارُ المالك ٢٠٢٦-١١-٠٤)
                StoreCommissionRow(
                    merchantId = storeId,
                    head = head,
                    saving = vm.commissionSaving,
                    onSave = { percent, done -> vm.saveStoreCommission(storeId, percent, done) },
                )
                IconTile(
                    icon = R.drawable.ic_grid,
                    label = stringResource(R.string.cd_sections_tile),
                    onClick = { vm.openSections() },
                )
            }

            // ══════════════════════════════════════════════════════════
            // **وصفُّ الإجراءات** — «إضافة صنف» هو الأساسيّ (أعرضُ وأقوى)، وبجانبه «الأصناف» و«إنشاء عرض».
            // ══════════════════════════════════════════════════════════
            Spacer(Modifier.height(Rahal.space.md))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RahalOutlineButton(
                    onClick = { onOpenItems(vm.openId, head.name) },
                    modifier = Modifier.weight(1f).height(Rahal.size.action),
                    compact = true,
                ) { ActionLabel(com.rahalgo.ui.R.drawable.ic_cart, stringResource(R.string.mn_items)) }
                RahalButton(
                    onClick = { onOpenMenu(vm.openId, head.name) },
                    modifier = Modifier.weight(1.25f).height(Rahal.size.action),
                    compact = true,
                ) { ActionLabel(com.rahalgo.ui.R.drawable.ic_plus, stringResource(R.string.mn_add_item)) }
                RahalOutlineButton(
                    onClick = { onOpenOffers(vm.openId, head.name) },
                    modifier = Modifier.weight(1f).height(Rahal.size.action),
                    compact = true,
                ) { ActionLabel(com.rahalgo.ui.R.drawable.ic_offer, stringResource(R.string.mn_make_offer)) }
            }
            // **«أضف فرع» لعميله** (قرارُ المالك ٢٠٢٦-١١-١٠) — طلبٌ يوافق عليه المكتب.
            Spacer(Modifier.height(Rahal.space.sm))
            RahalOutlineButton(
                onClick = { vm.branchOpen = true },
                modifier = Modifier.fillMaxWidth().height(Rahal.size.action),
                compact = true,
            ) { ActionLabel(com.rahalgo.ui.R.drawable.ic_store, stringResource(com.rahalgo.ui.R.string.branch_add)) }
        }

        // ══════════════════════════════════════════════════════════════
        // **وأرقامُه بنظرة** — ما وقع فوق، وما نلتَه تحت (طلبُ المالك ٢٠٢٦-٠٨-٣١: «مربّعاتٌ أنيقة»).
        // ══════════════════════════════════════════════════════════════
        //
        // **وصفّان لا صفٌّ من خمسة**: خمسةُ مربّعاتٍ في عرض الهاتف **تضيق فيُقصّ الرقم.**
        SectionHeader(stringResource(R.string.cd_summary_title))
        RahalCard {
            ActionRow {
                StatTile(stringResource(R.string.cd_sum_orders), sum.orders.toString())
                StatTile(
                    stringResource(R.string.cd_sum_delivered),
                    sum.delivered.toString(),
                    valueColor = if (sum.delivered > 0) Rahal.colors.success else Rahal.colors.ink,
                )
                StatTile(
                    stringResource(R.string.cd_sum_cancelled),
                    sum.cancelled.toString(),
                    valueColor = if (sum.cancelled > 0) Rahal.colors.danger else Rahal.colors.ink,
                )
            }
            Spacer(Modifier.height(Rahal.space.sm))
            ActionRow {
                StatTile(stringResource(R.string.cd_sales), money(sum.deliveredSales))
                StatTile(stringResource(R.string.cd_earnings), money(sum.myEarnings), valueColor = Rahal.colors.brand)
            }
        }

        SectionHeader(stringResource(R.string.cd_orders))
        if (d.orders.isEmpty()) {
            EmptyState(
                title = stringResource(R.string.cd_no_orders),
                icon = com.rahalgo.ui.R.drawable.ic_orders,
            )
            return@Screen
        }
        Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            d.orders.forEach { OrderRow(it) }
        }

        // **و«عرض المزيد» حين بقي ما لم يُجلب** (٢٠٢٦-١٠-٢٧) — **وكانت القائمةُ تقف عند عشرين.**
        if (vm.hasMoreOrders) {
            Spacer(Modifier.height(Rahal.space.md))
            RahalOutlineButton(
                onClick = { vm.loadMoreOrders() },
                enabled = !vm.loadingMore,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.cd_more, d.orders.size.toString(), d.total.toString()))
            }
        }
        // **وسطرُ الشرح الطويل تحت الجدول حُذف** (طلبُ المالك ٢٠٢٦-١٠-٣٠).
    }
    }
}

/**
 * **مربّعُ «ساعات الدوام»** — قيمتُه الحاليّة («مفتوح 24 ساعة» · «9:00 ص – 11:00 م» ·
 * «لم يُضبط بعد» بالأحمر)، ويُضغط فيفتح صفحتَها.
 */
@Composable
private fun HoursTile(vm: ClientsViewModel) {
    val set = vm.hoursSet
    val value = com.rahalgo.ui.hours.hoursSummary(set, vm.hours)
        ?: if (set) "" else stringResource(com.rahalgo.ui.R.string.home_val_not_set)
    IconTile(
        icon = com.rahalgo.ui.R.drawable.ic_time,
        label = stringResource(com.rahalgo.ui.R.string.home_row_hours),
        value = value,
        valueColor = if (set) Rahal.colors.inkMuted else Rahal.colors.danger,
        onClick = { vm.openHours() },
    )
}

/** **أيقونةٌ ونصٌّ داخل زرّ إجراء** — أيقونةٌ صغيرةٌ قبل النصّ. */
@Composable
private fun ActionLabel(icon: Int, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) {
        Icon(painterResource(icon), contentDescription = null, modifier = Modifier.size(Rahal.icon.xs))
        Spacer(Modifier.size(Rahal.space.xs))
        com.rahalgo.ui.FitButtonText(text)
    }
}

/**
 * **سطرُ طلب** — رقمُه وتاريخُه وحالُه، ثمّ قيمتُه ونصيبُه.
 *
 * **وثلاثُ دِلاءٍ لا أربعَ عشرةَ حالة**: «السائقُ في المتجر» و«جارٍ إسنادُ سائق» تفاصيلُ تشغيلٍ
 * **لا تعني المندوبَ ولا يملك تغييرَها.** ويعنيه سؤالان: أتمَّ الطلبُ فقبضتُ عمولتَه؟ أم ضاع؟
 */
@Composable
private fun OrderRow(o: RepOrderLine) {
    val done = o.status == "delivered"
    val lost = o.status in CANCELLED
    RahalCard(variant = CardVariant.Outlined) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // **وتاريخُه تحت رقمه** (٢٠٢٦-١٠-٢٧) — بتوقيت دمشق (`whenText`).
            Column(Modifier.weight(1f)) {
                Text(text = "#" + o.number, style = Rahal.type.bodyStrong, color = Rahal.colors.ink)
                if (o.createdAt.isNotEmpty()) {
                    Text(text = com.rahalgo.ui.whenText(o.createdAt), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
                }
            }
            StatusChip(
                stringResource(
                    when {
                        done -> R.string.ord_st_delivered
                        lost -> R.string.cd_st_cancelled
                        else -> R.string.cd_st_running
                    },
                ),
                when {
                    done -> StatusTone.Success
                    lost -> StatusTone.Danger
                    else -> StatusTone.Warning
                },
            )
        }
        Spacer(Modifier.height(Rahal.space.sm))
        RahalDivider()
        Spacer(Modifier.height(Rahal.space.xs))
        KeyValue(stringResource(R.string.cd_col_total), money(o.total))
        // **ولا يُعرض هامشُ المنصّة ولا عمولتُها الداخليّة** (RQ-7، ٢٠٢٦-٠٩-٢٦).

        // **ونصيبُه — مشطوباً إن ضاع، و«لا تُحتسب» بجانبه**: رقمٌ مشطوبٌ بلا كلمةٍ يُقرأ خطأً في العرض.
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(text = stringResource(R.string.cd_col_share), color = Rahal.colors.inkMuted, style = Rahal.type.body)
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (lost && o.forfeitedShare > 0) {
                    Text(
                        text = money(o.forfeitedShare),
                        color = Rahal.colors.inkMuted,
                        textDecoration = TextDecoration.LineThrough,
                        style = Rahal.type.body,
                    )
                    Spacer(Modifier.size(Rahal.space.xs))
                    Text(text = stringResource(R.string.cd_forfeited), color = Rahal.colors.danger, style = Rahal.type.meta)
                } else {
                    Text(
                        text = money(o.myShare),
                        color = if (o.myShare > 0) Rahal.colors.brand else Rahal.colors.inkMuted,
                        style = Rahal.type.bodyStrong,
                    )
                }
            }
        }

        // **وسببُ الإلغاء يُقال** — ومن لم يعرف لماذا ضاع طلبٌ لا يستطيع أن يمنع الثاني.
        if (lost) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                text = o.cancelReason.ifEmpty { stringResource(R.string.cd_no_reason) },
                color = Rahal.colors.danger,
                style = Rahal.type.caption,
            )
        }
    }
}

/** **ما يُعدّ ضائعا** — كما في الويب حرفا. */
private val CANCELLED = setOf("rejected", "cancelled", "failed", "refunded")
