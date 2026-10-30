package com.rahalgo.rep.clients

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.Backend
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepOrderLine
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.StatBox
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.money
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable

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
 * يُصدّق.** فهنا كلُّ طلب: رقمُه وتاريخُه وحالُه وقيمتُه وعمولةُ
 * المنصّة عليه ونصيبُه منه.
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
        // **وزرُّ رجوعٍ ظاهر** — كما في صفحة الأصناف (٢٠٢٦-١٠-٢٧)؛ **ومن لا
        // يعرف زرَّ الجهاز لا يجد طريقَ العودة.**
        // **سهمُ رجوعٍ وعنوان** — كما في صفحات المتجر (مراجعةُ المالك ٢٠٢٦-١٠-٣٠).
        com.rahalgo.ui.PageHeader(title = stringResource(R.string.nav_clients), onBack = { vm.closeDetail() })
        Spacer(Modifier.height(Rahal.space.sm))
        // ══════════════════════════════════════════════════════════════
        // **ترويسةُ العميل — من هو وحالُه ورقمُ صاحبه**
        // ══════════════════════════════════════════════════════════════
        //
        // **ورقمُ صاحب المتجر يُعرض** — **والمندوبُ يلاحقه ليُفعّله**،
        // ومن لا يجد رقمَه يبحث عنه في هاتفه أو يتّصل بالمكتب.
        Card {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // **والشعارُ يُضغط فيُبدَّل** (طلبُ المالك ٢٠٢٦-١٠-١٩) — كما عند صاحب المتجر.
                RemoteImage(
                    url = Backend.of(context).media(head.logoThumbUrl),
                    name = head.name,
                    modifier = Modifier
                        .size(52.dp)
                        .clip(Rahal.shape.sm)
                        .clickable(enabled = !vm.busy) { pickLogo() },
                )
                Spacer(Modifier.size(10.dp))
                Column {
                    Text(
                        text = head.name,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                    )
                    if (head.categoryName.isNotEmpty()) {
                        Text(
                            text = head.categoryName,
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
            }
            com.rahalgo.ui.RahalTextButton(onClick = { pickLogo() }, enabled = !vm.busy) {
                Text(
                    stringResource(
                        if (head.logoThumbUrl.isNullOrBlank()) com.rahalgo.ui.R.string.store_logo_add
                        else com.rahalgo.ui.R.string.store_logo_change,
                    ),
                    color = Rahal.colors.brand,
                )
            }
            Spacer(Modifier.height(8.dp))
            Chip(
                stringResource(
                    if (head.status == "active") R.string.cl_working else R.string.cl_stopped,
                ),
                if (head.status == "active") Rahal.colors.success else Rahal.colors.inkMuted,
            )
            head.ownerPhone?.takeIf { it.isNotEmpty() }?.let {
                Spacer(Modifier.height(8.dp))
                KeyValue(stringResource(R.string.cd_owner_phone), ltrPhone(it))
                // **والرقمُ يُضغط** (٢٠٢٦-١٠-٢٧) — اتّصالٌ أو واتساب، **وكان نصّاً
                // يُنسخ باليد.**
                Spacer(Modifier.height(6.dp))
                ContactButtons(it)
                Spacer(Modifier.height(6.dp))
            }
            if (head.joinedAt.isNotEmpty()) {
                // **وتاريخٌ يقرؤه الناس** — «27 تشرين الأول 2026» لا صيغةَ آلة.
                KeyValue(stringResource(R.string.cd_joined), longDate(head.joinedAt))
            }
            // **وما ينقصه ليُحسب — كما في بطاقته** (٢٠٢٦-١٠-٢٧).
            // **ونقصُ الدوام يُضغط فيفتح صفحتَه** (٢٠٢٦-١٠-٣٠).
            row?.let { MissingStrip(missingFor(it), onClick = if (!it.hasHours) ({ vm.openHours() }) else null) }

            // **وساعاتُ دوامه — يضبطها هو** (قرارُ المالك ٢٠٢٦-١٠-٣٠) بالمخطّط نفسِه
            // الذي في تطبيق المتجر (`ClientHours`).
            Spacer(Modifier.height(8.dp))
            HoursRow(vm)

            // ══════════════════════════════════════════════════════════
            // **وبابُ أصنافه — يبنيها نيابةً عنه**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لا يوجد بالتطبيق زرُّ إضافة
            //  منتجات العميل كما اتّفقنا سابقا».)
            //
            // **ومتجرٌ ينضمّ ولا يفتح لوحتَه** — صاحبُه في متجره لا في
            // حاسوب، **وسوقٌ فيه متاجرُ بلا أصنافٍ سوقٌ فارغ.**
            //
            // **وهنا لا في القائمة**: المندوبُ يفتح العميلَ ليعمل عليه،
            // **وزرٌّ في سطر القائمة يُضغط سهواً وهو يمرّر.**
            //
            // **والزرّان نفسُهما في بطاقته** (٢٠٢٦-١٠-٢٧) — «إضافة صنف» و«إنشاء
            // عرض»؛ **وكان هنا «الأصناف» وحدَه، فالشاشتان لا تتطابقان.**
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                RahalOutlineButton(
                    onClick = { onOpenItems(vm.openId, head.name) },
                    modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
                ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_items)) }
                RahalButton(
                    onClick = { onOpenMenu(vm.openId, head.name) },
                    modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
                ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_add_item)) }
                RahalOutlineButton(
                    onClick = { onOpenOffers(vm.openId, head.name) },
                    modifier = Modifier.weight(1f).height(48.dp),
                compact = true,
                ) { com.rahalgo.ui.FitButtonText(stringResource(R.string.mn_make_offer)) }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وأربعةُ أرقامٍ تُقرأ بنظرة**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        // **ومربّعاتٌ لا أسطر** — (طلبُ المالك ٢٠٢٦-٠٨-٣١: «داخل ملفّ
        // المتجر أيضاً: إجمالي الطلبات · تم التسليم · ملغى · مبيعات
        // سلّمت · صافي عمولتي — أيضاً مربّعاتٌ أنيقةٌ مناسبة»).
        //
        // **وصفّان لا صفٌّ من خمسة**: خمسةُ مربّعاتٍ في عرض الهاتف
        // **تضيق فيُقصّ الرقم**، والعددُ فوق والمالُ تحت — **والفصلُ
        // بينهما معنًى**: تلك ما وقع، وهذه ما نلتَه.
        Card {
            StatRow {
                StatBox(
                    label = stringResource(R.string.cd_sum_orders),
                    value = sum.orders.toString(),
                    modifier = Modifier.weight(1f),
                )
                StatBox(
                    label = stringResource(R.string.cd_sum_delivered),
                    value = sum.delivered.toString(),
                    modifier = Modifier.weight(1f),
                )
                StatBox(
                    label = stringResource(R.string.cd_sum_cancelled),
                    value = sum.cancelled.toString(),
                    modifier = Modifier.weight(1f),
                    color = if (sum.cancelled > 0) Rahal.colors.danger else Rahal.colors.ink,
                )
            }
            Spacer(Modifier.height(8.dp))
            StatRow {
                StatBox(
                    label = stringResource(R.string.cd_sales),
                    value = money(sum.deliveredSales),
                    modifier = Modifier.weight(1f),
                )
                StatBox(
                    label = stringResource(R.string.cd_earnings),
                    value = money(sum.myEarnings),
                    modifier = Modifier.weight(1f),
                    color = Rahal.colors.brand,
                )
            }
        }

        Spacer(Modifier.height(16.dp))
        SectionTitle(stringResource(R.string.cd_orders))
        if (d.orders.isEmpty()) {
            Empty(stringResource(R.string.cd_no_orders))
            return@Screen
        }
        d.orders.forEach { OrderRow(it) }

        // **و«عرض المزيد» حين بقي ما لم يُجلب** (٢٠٢٦-١٠-٢٧) — **وكانت
        // القائمةُ تقف عند عشرين** والمربّعاتُ فوقها تعدّ الكلّ.
        if (vm.hasMoreOrders) {
            Spacer(Modifier.height(10.dp))
            RahalOutlineButton(
                onClick = { vm.loadMoreOrders() },
                enabled = !vm.loadingMore,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.cd_more, d.orders.size.toString(), d.total.toString()))
            }
        }

        // **وسطرُ الشرح الطويل تحت الجدول حُذف** (طلبُ المالك ٢٠٢٦-١٠-٣٠).
        Spacer(Modifier.height(24.dp))
    }
    }
}

/**
 * **سطرُ «ساعات الدوام»** — قيمتُه الحاليّة («مفتوح 24 ساعة» · «9:00 ص – 11:00 م» ·
 * «لم يُضبط بعد» بالأحمر)، ويُضغط فيفتح صفحتَها.
 */
@Composable
private fun HoursRow(vm: ClientsViewModel) {
    val set = vm.hoursSet
    val value = com.rahalgo.ui.hours.hoursSummary(set, vm.hours)
        ?: if (set) "" else stringResource(com.rahalgo.ui.R.string.home_val_not_set)
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sm)
            .clickable { vm.openHours() }
            .padding(vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            stringResource(com.rahalgo.ui.R.string.home_row_hours),
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
        Spacer(Modifier.size(8.dp))
        Text(
            value,
            color = if (set) Rahal.colors.inkMuted else Rahal.colors.danger,
            style = MaterialTheme.typography.bodySmall,
            maxLines = 1,
            textAlign = androidx.compose.ui.text.style.TextAlign.End,
            modifier = Modifier.weight(1f),
        )
        Spacer(Modifier.size(6.dp))
        androidx.compose.material3.Icon(
            androidx.compose.ui.res.painterResource(com.rahalgo.ui.R.drawable.ic_arrow_back),
            contentDescription = null,
            tint = Rahal.colors.inkMuted,
            modifier = Modifier
                .size(18.dp)
                .graphicsLayer { rotationZ = 180f },
        )
    }
}

/**
 * **سطرُ طلب** — رقمُه وحالُه وقيمتُه وعمولةُ المنصّة ونصيبُه.
 *
 * **وثلاثُ دِلاءٍ لا أربعَ عشرةَ حالة**: «السائقُ في المتجر» و«جارٍ
 * إسنادُ سائق» تفاصيلُ تشغيلٍ **لا تعني المندوبَ ولا يملك تغييرَها.**
 *
 * **ويعنيه سؤالان**: أتمَّ الطلبُ فقبضتُ عمولتَه؟ أم ضاع؟
 */
@Composable
private fun OrderRow(o: RepOrderLine) {
    val done = o.status == "delivered"
    val lost = o.status in CANCELLED
    Spacer(Modifier.height(8.dp))
    Card {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // **وتاريخُه بجانب رقمه** (٢٠٢٦-١٠-٢٧) — **ومن لا يعرف أمن أمسِ
            // الطلبُ أم من شهرٍ لا يعرف أيسأل عنه.** وبتوقيت دمشق (`whenText`).
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.weight(1f)) {
                Text(
                    text = "#" + o.number,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
                if (o.createdAt.isNotEmpty()) {
                    Spacer(Modifier.size(8.dp))
                    Text(
                        text = com.rahalgo.ui.whenText(o.createdAt),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
            }
            Chip(
                stringResource(
                    when {
                        done -> R.string.ord_st_delivered
                        lost -> R.string.cd_st_cancelled
                        else -> R.string.cd_st_running
                    },
                ),
                when {
                    done -> Rahal.colors.success
                    lost -> Rahal.colors.danger
                    else -> Rahal.colors.accent
                },
            )
        }
        Spacer(Modifier.height(6.dp))
        KeyValue(stringResource(R.string.cd_col_total), money(o.total))
        // **ولا يُعرض هامشُ المنصّة ولا عمولتُها الداخليّة** (RQ-7، ٢٠٢٦-٠٩-٢٦):
        // المندوبُ يرى إجماليَّ الطلب ونصيبَه هو — لا هامشَنا. (حُذف صفُّ عمولة المنصّة.)

        // ══════════════════════════════════════════════════════════════
        // **ونصيبُه — مشطوباً إن ضاع**
        // ══════════════════════════════════════════════════════════════
        //
        // **«لا تُحتسب» بجانبه** — **ورقمٌ مشطوبٌ بلا كلمةٍ يُقرأ خطأً
        // في العرض لا حالاً في الحساب.**
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = stringResource(R.string.cd_col_share),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (lost && o.forfeitedShare > 0) {
                    Text(
                        text = money(o.forfeitedShare),
                        color = Rahal.colors.inkMuted,
                        textDecoration = TextDecoration.LineThrough,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Spacer(Modifier.size(6.dp))
                    Text(
                        text = stringResource(R.string.cd_forfeited),
                        color = Rahal.colors.danger,
                        style = MaterialTheme.typography.labelSmall,
                    )
                } else {
                    Text(
                        text = money(o.myShare),
                        color = if (o.myShare > 0) Rahal.colors.brand else Rahal.colors.inkMuted,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
        }

        // **وسببُ الإلغاء يُقال** — **ومن لم يعرف لماذا ضاع طلبٌ لا
        // يستطيع أن يمنع الثاني.**
        if (lost) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = o.cancelReason.ifEmpty { stringResource(R.string.cd_no_reason) },
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/** **ما يُعدّ ضائعا** — كما في الويب حرفا. */
private val CANCELLED = setOf("rejected", "cancelled", "failed", "refunded")
