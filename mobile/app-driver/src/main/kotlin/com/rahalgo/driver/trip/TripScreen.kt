package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرحلة — خريطة وشريط وبطاقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفة المالك ٢٠٢٦-٠٨-١٢، `docs/DRIVER-APP-PLAN.md` §٣ب.)
 *
 * **ثلاث طبقات**: شريط خطّة السير فوق كلّ شيء · الخريطة · بطاقة سفليّة
 * فيها التفاصيل والأزرار.
 *
 * # ولماذا الشريط فوق
 *
 * **السائق ينظر إلى الشاشة ثانيةً واحدةً وهو واقف** — والشريط يقول أين
 * هو من الرحلة بلا قراءة: **قبلت ← إلى المتجر ← وصلت ← استلمت ← إلى
 * الزبون ← وصلت ← سلّمت.**
 *
 * # والوجهة تتبدّل بنفسها
 *
 * **عند «استلمت الطلب» تصير الوجهة الزبون** — لا يضغط شيئا. (مواصفة
 * المالك: «هون بشكل تلقائي الرحلة تتحوّل إلى الزبون».)
 */
@Composable
fun TripScreen(state: TripState, actions: TripActions) {
    val order = state.order
    if (order == null) {
        NoTrip(onOrders = actions.toOrders)
        return
    }

    Box(Modifier.fillMaxSize()) {
        TripMap(
            driver = state.driver,
            // **وقبل الاستلام تُعرض النقطتان** — بعده تُطفأ نقطة المتجر:
            // **انتهى شأنه منها**، وخريطة فيها ما لم يعد يلزم تشوّش.
            pickup = if (state.step >= TripStep.PICKED_UP) null else state.pickup,
            dropoff = state.dropoff,
            modifier = Modifier.fillMaxSize(),
        )

        StepStrip(
            step = state.step,
            modifier = Modifier.align(Alignment.TopCenter).statusBarsPadding(),
        )

        TripCard(
            order = order,
            state = state,
            actions = actions,
            modifier = Modifier.align(Alignment.BottomCenter),
        )
    }

    if (state.failReasons != null) {
        FailDialog(
            reasons = state.failReasons,
            onPick = actions.fail,
            onDismiss = actions.dismissFail,
        )
    }
}

/**
 * **أسباب التعذّر — تُختار ولا تُكتب.**
 *
 * **والسبب يقرّر من يتحمّل**: «المتجر مغلق» ذنب متجر يستوجب تعويض
 * السائق، **و«تأخّرت» ذنبه هو.** ونصّ حرّ لا يُعدّ ولا يُقاس.
 */
@Composable
private fun FailDialog(
    reasons: List<FailReasonItem>,
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.detail_fail_title)) },
        text = {
            Column {
                if (reasons.isEmpty()) {
                    Text(stringResource(R.string.detail_no_reasons), color = InkMuted)
                }
                for (r in reasons) {
                    TextButton(onClick = { onPick(r.code) }, modifier = Modifier.fillMaxWidth()) {
                        Text(reasonLabel(r.code), modifier = Modifier.fillMaxWidth())
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_cancel)) }
        },
    )
}

/** **ورمز بلا ترجمة يُعرض كما هو** — ليُعرف ويُضاف، لا ليُبتلع. */
@Composable
private fun reasonLabel(code: String): String = when (code) {
    "customer_absent" -> stringResource(R.string.reason_customer_absent)
    "customer_refused" -> stringResource(R.string.reason_customer_refused)
    "customer_unreachable" -> stringResource(R.string.reason_customer_unreachable)
    "address_wrong" -> stringResource(R.string.reason_address_wrong)
    "driver_late" -> stringResource(R.string.reason_driver_late)
    "merchant_closed" -> stringResource(R.string.reason_merchant_closed)
    "merchant_refused" -> stringResource(R.string.reason_merchant_refused)
    "merchant_not_ready" -> stringResource(R.string.reason_merchant_not_ready)
    "order_unknown" -> stringResource(R.string.reason_order_unknown)
    else -> code
}

/** المسافة بالمتر أو بالكيلومتر — **لا «1400 م».** */
private fun distanceText(meters: Double): String {
    val m = meters.toLong()
    return if (m < 1000) "$m م" else "%.1f كم".format(m / 1000.0)
}

/**
 * **الوقت المتوقّع** — من المسافة وسرعة السائق.
 *
 * **وفارغ إن لم تُضبط السرعة**: **رقمٌ مبنيٌّ على صفر يقول «الآن»**،
 * ووعدٌ كاذبٌ للزبون أسوأ من لا وعد.
 *
 * **ودقيقة على الأقلّ** — «٠ دقيقة» لا تُقال لمن لم يصل بعد.
 */
private fun eta(meters: Double, avgSpeedKmh: Long): String {
    if (avgSpeedKmh <= 0 || meters < 0) return ""
    val minutes = ((meters / 1000.0) / avgSpeedKmh * 60).toLong().coerceAtLeast(1)
    return " · ~$minutes د"
}

/**
 * **شريط خطّة السير.**
 *
 * **والخطوة الحاليّة وحدَها ملوّنة** — وما مضى باهت وما بقي أبهت.
 * **ومن لوّن الكلّ** جعل السائق يبحث عن موضعه في سبعة متشابهة.
 */
@Composable
private fun StepStrip(step: TripStep, modifier: Modifier = Modifier) {
    Row(
        modifier
            .fillMaxWidth()
            .background(Color.White.copy(alpha = 0.94f))
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        for (s in TripStep.entries) {
            val done = s.ordinal < step.ordinal
            val now = s == step
            Text(
                text = stringResource(s.label),
                color = when {
                    now -> BrandTeal
                    done -> InkMuted
                    else -> InkMuted.copy(alpha = 0.45f)
                },
                fontWeight = if (now) FontWeight.Bold else FontWeight.Normal,
                style = MaterialTheme.typography.bodySmall,
            )
            if (s != TripStep.entries.last()) {
                Text(
                    text = " ← ",
                    color = InkMuted.copy(alpha = 0.4f),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

/**
 * **البطاقة السفليّة.**
 *
 * **وما يقرّر به السائق أوّلا**: إلى أين يذهب الآن، وكم يقبض.
 */
@Composable
private fun TripCard(
    order: DriverOrder,
    state: TripState,
    actions: TripActions,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(topStart = 20.dp, topEnd = 20.dp))
            .background(Color.White)
            .padding(20.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                // **والعنوان يقول الوجهة الحاليّة** — لا اسم الطلب:
                // **من قرأ «طيف» وهو في طريقه للزبون** قرأ ما مضى.
                text = if (state.step >= TripStep.PICKED_UP) {
                    order.customerName.ifBlank { stringResource(R.string.detail_customer) }
                } else {
                    order.merchantName
                },
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            Text("#${order.number}", color = InkMuted)
        }

        Spacer(Modifier.height(4.dp))
        Text(
            text = if (state.step >= TripStep.PICKED_UP) order.addressText else "",
            color = InkMuted,
        )

        // ══════════════════════════════════════════════════════════════
        // **كم بقي — مسافةً ووقتا**
        // ══════════════════════════════════════════════════════════════
        //
        // (مواصفة المالك ٢٠٢٦-٠٨-١٢: «يجب أن يكون واضحا ما المسافة …
        //  وكم الوقت المتوقّع حسب سرعة السائق».)
        //
        // **والسرعة من المحرّك لا من الشيفرة** (`drivers.avg_speed_kmh`)
        // — تُضبط للمدينة من لوحة الإدارة: **دراجةٌ في الرقّة غيرُ سيّارة
        // في مدينةٍ أخرى.**
        val meters = state.remainingM
        if (meters >= 0) {
            Spacer(Modifier.height(8.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(stringResource(R.string.trip_remaining), color = InkMuted)
                Text(
                    text = distanceText(meters) + eta(meters, state.avgSpeedKmh),
                    fontWeight = FontWeight.Bold,
                )
            }
        }

        Spacer(Modifier.height(10.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(stringResource(R.string.order_cash_due), color = InkMuted)
            Text(
                text = if (order.cashDue > 0) {
                    money(order.cashDue)
                } else {
                    stringResource(R.string.order_prepaid)
                },
                fontWeight = FontWeight.Bold,
                color = if (order.cashDue > 0) BrandTeal else InkMuted,
            )
        }

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Text(state.error, color = BrandOrange)
        }

        Spacer(Modifier.height(14.dp))
        val next = nextAction(order.status)
        if (next != null) {
            Button(
                onClick = { actions.step(next.status) },
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.busy) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(stringResource(next.label))
                }
            }
        }

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
            // **والملاحة تُسلَّم لتطبيق الخرائط** — لا تُبنى هنا:
            // **السائق يعرف تطبيقه ويثق بصوته.**
            TextButton(onClick = actions.navigate, enabled = !state.busy) {
                Text(stringResource(R.string.trip_navigate), color = BrandTeal)
            }
            // **والتعذّر حيث يقبله المحرّك وحدَه** — عند المتجر أو عند
            // باب الزبون. **وزرّ يظهر دائما** يُضغط في غير موضعه فيُردّ
            // برفضٍ لا يفهمه صاحبه.
            if (order.status == "at_pickup" || order.status == "at_dropoff") {
                TextButton(onClick = actions.askFail, enabled = !state.busy) {
                    Text(stringResource(R.string.detail_failed), color = BrandOrange)
                }
            }
        }
    }
}

@Composable
private fun NoTrip(onOrders: () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.trip_none),
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = stringResource(R.string.trip_none_hint),
            color = InkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(18.dp))
        Button(onClick = onOrders) { Text(stringResource(R.string.nav_orders)) }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خطوات الرحلة السبع — وستّة أحوال في المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والفرق مقصود**: «قبلت الطلب» و«في الطريق إلى المتجر» **حالٌ واحد في
 * المحرّك** (`assigned`) — لكنّهما لحظتان مختلفتان عند السائق: **قَبِل،
 * ثمّ مشى.**
 *
 * # وخطأٌ وقع هنا
 *
 * **كُتب الشريطُ يقرأ خطوتَه من ترتيب القائمة والزرُّ يقرأ التالية منها**
 * — فظهر الشريطُ يقول «قبلت الطلب» والزرُّ يقول «استلمت البضاعة»:
 * **قفزةُ خطوةٍ كاملة.** (قيس على الجهاز ٢٠٢٦-٠٨-١٢.)
 *
 * **فصار كلٌّ منهما يُشتقّ من حال الطلب في المحرّك وحدَه** — لا من
 * الآخر.
 */
enum class TripStep(val label: Int) {
    ACCEPTED(R.string.trip_s_accepted),
    TO_PICKUP(R.string.trip_s_to_pickup),
    AT_PICKUP(R.string.trip_s_at_pickup),
    PICKED_UP(R.string.trip_s_picked_up),
    TO_CUSTOMER(R.string.trip_s_to_customer),
    AT_CUSTOMER(R.string.trip_s_at_customer),
    DELIVERED(R.string.trip_s_delivered),
    ;

    companion object {
        /**
         * **يقرأ الخطوة من حال الطلب.**
         *
         * **و`assigned` تُقرأ «في الطريق إلى المتجر»** لا «قبلت»: **القبول
         * لحظةٌ مضت**، وما يفعله الآن هو المشي.
         */
        fun of(status: String): TripStep = when (status) {
            "assigned" -> TO_PICKUP
            "at_pickup" -> AT_PICKUP
            "picked_up" -> PICKED_UP
            "on_the_way" -> TO_CUSTOMER
            "at_dropoff" -> AT_CUSTOMER
            "delivered" -> DELIVERED
            else -> ACCEPTED
        }
    }
}

/** الفعل التالي — **باسم الحال في المحرّك ونصِّ الزرّ.** */
data class NextAction(val status: String, val label: Int)

/**
 * **ما يفعله الآن** — يُشتقّ من حال الطلب لا من موضعه في الشريط.
 *
 * **والمحرّك يرفض أيّ قفزة** (`orders/statuses.go`) — فلو تأخّرت الشاشة
 * عن حاله الحقيقيّ ردّ خطأً ولم يقع شيء.
 */
fun nextAction(status: String): NextAction? = when (status) {
    "assigned" -> NextAction("at_pickup", R.string.step_at_pickup)
    "at_pickup" -> NextAction("picked_up", R.string.step_picked_up)
    "picked_up" -> NextAction("on_the_way", R.string.step_on_the_way)
    "on_the_way" -> NextAction("at_dropoff", R.string.step_at_dropoff)
    "at_dropoff" -> NextAction("delivered", R.string.step_delivered)
    else -> null
}

/** ما تعرضه الرحلة — **ولا تملكه هي.** */
data class TripState(
    val order: DriverOrder? = null,
    /** ما بقي من الطريق بالمتر — **وسالبٌ يعني لا يُعرف.** */
    val remainingM: Double = -1.0,
    /** سرعة السائق الوسطى من المحرّك — **وصفر يعني لا تُحسب مدّة.** */
    val avgSpeedKmh: Long = 0,
    val failReasons: List<FailReasonItem>? = null,
    val step: TripStep = TripStep.ACCEPTED,
    val driver: LatLng? = null,
    val pickup: LatLng? = null,
    val dropoff: LatLng? = null,
    val busy: Boolean = false,
    val error: String = "",
)

data class TripActions(
    val step: (String) -> Unit,
    val askFail: () -> Unit,
    val fail: (String) -> Unit,
    val dismissFail: () -> Unit,
    val navigate: () -> Unit,
    val toOrders: () -> Unit,
)
