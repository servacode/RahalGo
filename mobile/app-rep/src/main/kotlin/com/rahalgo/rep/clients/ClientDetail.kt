package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
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
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.money
import com.rahalgo.ui.RahalOutlineButton

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
fun ClientDetail(vm: ClientsViewModel, onOpenMenu: (String, String) -> Unit) {
    val d = vm.detail
    if (d == null) {
        LoadState(vm.busy, vm.error) { vm.openDetail(vm.openId) }
        return
    }
    val context = LocalContext.current
    val head = d.merchant
    val sum = d.summary

    Screen {
        // ══════════════════════════════════════════════════════════════
        // **ترويسةُ العميل — من هو وحالُه ورقمُ صاحبه**
        // ══════════════════════════════════════════════════════════════
        //
        // **ورقمُ صاحب المتجر يُعرض** — **والمندوبُ يلاحقه ليُفعّله**،
        // ومن لا يجد رقمَه يبحث عنه في هاتفه أو يتّصل بالمكتب.
        Card {
            Row(verticalAlignment = Alignment.CenterVertically) {
                RemoteImage(
                    url = Backend.of(context).media(head.logoThumbUrl),
                    name = head.name,
                    modifier = Modifier
                        .size(52.dp)
                        .clip(Rahal.shape.sm),
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
            Spacer(Modifier.height(8.dp))
            Chip(
                stringResource(
                    if (head.status == "active") R.string.cl_working else R.string.cl_stopped,
                ),
                if (head.status == "active") Rahal.colors.success else Rahal.colors.inkMuted,
            )
            head.ownerPhone?.takeIf { it.isNotEmpty() }?.let {
                Spacer(Modifier.height(8.dp))
                KeyValue(stringResource(R.string.cd_owner_phone), it)
            }
            if (head.joinedAt.isNotEmpty()) {
                KeyValue(stringResource(R.string.cd_joined), head.joinedAt.take(10))
            }

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
            Spacer(Modifier.height(10.dp))
            RahalOutlineButton(
                onClick = { onOpenMenu(vm.openId, head.name) },
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.mn_items)) }
        }

        // ══════════════════════════════════════════════════════════════
        // **وأربعةُ أرقامٍ تُقرأ بنظرة**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        Card {
            KeyValue(stringResource(R.string.cd_sum_orders), sum.orders.toString())
            KeyValue(stringResource(R.string.cd_sum_delivered), sum.delivered.toString())
            KeyValue(
                stringResource(R.string.cd_sum_cancelled),
                sum.cancelled.toString(),
                valueColor = if (sum.cancelled > 0) Rahal.colors.danger else Color_Unspecified,
            )
            Spacer(Modifier.height(6.dp))
            HorizontalDivider()
            Spacer(Modifier.height(6.dp))
            KeyValue(stringResource(R.string.cd_sales), money(sum.deliveredSales))
            KeyValue(
                stringResource(R.string.cd_earnings),
                money(sum.myEarnings),
                valueColor = Rahal.colors.brand,
            )
        }

        Spacer(Modifier.height(16.dp))
        SectionTitle(stringResource(R.string.cd_orders))
        if (d.orders.isEmpty()) {
            Empty(stringResource(R.string.cd_no_orders))
            return@Screen
        }
        d.orders.forEach { OrderRow(it) }

        // **والقاعدةُ تُقال مرّةً تحت الجدول** — **ومن رأى رقماً مشطوباً
        // بلا شرحٍ ظنّ أنّ حقَّه ضاع.**
        Spacer(Modifier.height(12.dp))
        Text(
            text = stringResource(R.string.cd_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(24.dp))
    }
}

private val Color_Unspecified = androidx.compose.ui.graphics.Color.Unspecified

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
            Text(
                text = "#" + o.number,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
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
        KeyValue(stringResource(R.string.cd_col_commission), money(o.platformCommission))

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
