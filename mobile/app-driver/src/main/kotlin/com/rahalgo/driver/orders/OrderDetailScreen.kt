package com.rahalgo.driver.orders

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطلب في يده — وزرّ واحد يمشّيه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء السادسة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # لماذا زرّ واحد كبير لا قائمة حالات
 *
 * **السائق يقرأ الشاشة بيد واحدة وهو على الدرّاجة** — والسلسلة واحدة لا
 * تتفرّع: **وصلت للمتجر ← استلمت ← بالطريق ← وصلت للزبون ← سلّمت.**
 *
 * **ومن عرض عليه كلّ الحالات ليختار** جعله يقرأ خمسة ويقرّر، **وأخطأ
 * واحد فقفز خطوة** — والمحرّك يرفض القفزة، فيقف في الشارع لا يعرف لماذا.
 *
 * **والزرّ يقول ما فعله لا ما هو فيه**: «استلمت البضاعة» فعلٌ وقع، لا
 * حالٌ قائم.
 *
 * # والتالي يقرّره المحرّك لا هذه الشاشة
 *
 * **الأسماء هنا من آلة الحالات** (`orders/statuses.go`) — **والمحرّك
 * يرفض أيّ قفزة.** فلو تأخّرت الشاشة عن حاله الحقيقيّ ردّ خطأ ولم يقع
 * شيء.
 */
@Composable
fun OrderDetailScreen(state: DetailState, actions: DetailActions) {
    val order = state.order

    Column(
        Modifier
            .fillMaxSize()
            .statusBarsPadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp),
    ) {
        Spacer(Modifier.height(12.dp))
        RahalTextButton(onClick = actions.back) { Text(stringResource(R.string.detail_back)) }

        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                // **واسمٌ فارغٌ لا يُعرض** — الطلبُ الخاصُّ بلا متجر،
                // **فكان العنوانُ يخرج خالياً** ويبقى الرقمُ وحدَه في
                // زاويةٍ: **شاشةٌ تُفتح بلا عنوانٍ تُقرأ نصفَ محمَّلة.**
                text = order.merchantName.ifBlank { stringResource(R.string.card_custom) },
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
            )
            Text("#${order.number}", color = Rahal.colors.inkMuted)
        }

        Spacer(Modifier.height(6.dp))
        Text(statusLabel(order.status), color = Rahal.colors.brand, fontWeight = FontWeight.Bold)

        // **وما طلبه الزبونُ بلفظه** — أوّلُ ما يُقرأ في الخاصّ.
        if (order.customRequest.isNotEmpty()) {
            Spacer(Modifier.height(14.dp))
            Field(stringResource(R.string.card_custom_what), order.customRequest)
        }

        Spacer(Modifier.height(18.dp))
        Field(stringResource(R.string.detail_customer), order.customerName)
        Field(stringResource(R.string.detail_address), order.addressText)
        if (order.itemsCount > 0) {
            Field(stringResource(R.string.detail_items), order.itemsCount.toString())
        }
        Field(stringResource(R.string.detail_total), money(order.total))
        // **والنقد آخر ما يُقرأ وأبرزه** — هو ما يُخطئ فيه: من نسي أنّ
        // الطلب مدفوع سلفا **قبض مرّتين**، ومن ظنّه مدفوعا مشى بلا نقد.
        Field(
            label = stringResource(R.string.order_cash_due),
            value = if (order.cashDue > 0) {
                money(order.cashDue)
            } else {
                stringResource(R.string.order_prepaid)
            },
            strong = true,
        )

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(14.dp))
            Text(state.error, color = Rahal.colors.accent)
        }

        Spacer(Modifier.height(22.dp))
        val next = nextStep(order.status, order.kind == "custom")
        if (next != null) {
            RahalButton(
                onClick = { actions.step(next.to) },
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

        // **والتعذّر لا يُعرض إلّا حيث يقبله المحرّك** — عند المتجر أو
        // عند باب الزبون. **وزرّ يظهر دائما** يُضغط في غير موضعه فيُردّ
        // برفض لا يفهمه صاحبه.
        if (order.status == "at_pickup" || order.status == "at_dropoff") {
            Spacer(Modifier.height(8.dp))
            RahalTextButton(
                onClick = actions.askFail,
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.detail_failed), color = Rahal.colors.accent)
            }
        }

        // **وإعادة الطلب قبل أن يستلم البضاعة فقط** — بعدها هي في يده،
        // **والبضاعة لا تُعاد بضغطة زرّ.**
        if (order.status == "assigned" || order.status == "at_pickup") {
            RahalTextButton(
                onClick = actions.release,
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.detail_release), color = Rahal.colors.inkMuted)
            }
        }
        Spacer(Modifier.height(28.dp))
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
                    Text(stringResource(R.string.detail_no_reasons), color = Rahal.colors.inkMuted)
                }
                for (r in reasons) {
                    RahalTextButton(
                        onClick = { onPick(r.code) },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(reasonLabel(r.code), modifier = Modifier.fillMaxWidth())
                    }
                }
            }
        },
        confirmButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_cancel)) }
        },
    )
}

@Composable
private fun Field(label: String, value: String, strong: Boolean = false) {
    Row(
        Modifier
            .fillMaxWidth()
            .padding(vertical = 7.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, color = Rahal.colors.inkMuted)
        Text(
            text = value,
            fontWeight = if (strong) FontWeight.Bold else FontWeight.Normal,
            color = if (strong) Rahal.colors.brand else MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** الخطوة التالية في السلسلة — **وفارغ يعني لا خطوة له.** */
private fun nextStep(status: String, custom: Boolean = false): Step? = when (status) {
    // **ولا «وصلتُ إلى المتجر» في الخاصّ** — انظر `TripScreen.nextAction`.
    "assigned" ->
        if (custom) Step("picked_up", R.string.step_bought)
        else Step("at_pickup", R.string.step_at_pickup)
    "at_pickup" -> Step("picked_up", R.string.step_picked_up)
    "picked_up" -> Step("on_the_way", R.string.step_on_the_way)
    "on_the_way" -> Step("at_dropoff", R.string.step_at_dropoff)
    "at_dropoff" -> Step("delivered", R.string.step_delivered)
    else -> null
}

private data class Step(val to: String, val label: Int)

@Composable
private fun statusLabel(status: String): String = when (status) {
    "assigned" -> stringResource(R.string.status_assigned)
    "at_pickup" -> stringResource(R.string.status_at_pickup)
    "picked_up" -> stringResource(R.string.status_picked_up)
    "on_the_way" -> stringResource(R.string.status_on_the_way)
    "at_dropoff" -> stringResource(R.string.status_at_dropoff)
    "delivered" -> stringResource(R.string.status_delivered)
    else -> status
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

/** ما تعرضه الشاشة — **ولا تملكه هي.** */
data class DetailState(
    val order: DriverOrder,
    val busy: Boolean = false,
    val error: String = "",
    /** أسباب التعذّر المعروضة الآن — **وفارغ يعني لا سؤال مفتوح.** */
    val failReasons: List<FailReasonItem>? = null,
)

data class DetailActions(
    val step: (String) -> Unit,
    val askFail: () -> Unit,
    val fail: (String) -> Unit,
    val dismissFail: () -> Unit,
    val release: () -> Unit,
    val back: () -> Unit,
)
