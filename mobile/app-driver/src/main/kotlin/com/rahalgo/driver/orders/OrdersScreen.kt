package com.rahalgo.driver.orders

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.grouped
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطلبات — ما عُرض عليه وما في يده**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الخامسة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # ولماذا في شاشة واحدة
 *
 * **السائق يوازن بينهما في اللحظة نفسها**: أيأخذ عرضا جديدا وفي يده
 * اثنان؟ **ومن فصلهما شاشتين** جعله يبدّل ليقرّر.
 *
 * # والعرض أوّلا
 *
 * **العرض يفوت** — يأخذه غيره أو تنقضي مهلته. **وما في يده لا يفوت.**
 * فالذي يزول يُعرض قبل الذي يبقى.
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** —
 * والنداءات في `OrdersViewModel`.
 */
@Composable
fun OrdersScreen(state: OrdersState, actions: OrdersActions) {
    if (state.loading) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator()
        }
        return
    }

    LazyColumn(
        Modifier
            .fillMaxSize()
            // **وتحت شريط الحالة لا خلفه** — الشاشة تُرسم من حافة إلى
            // حافة. **وشريط النظام السفليّ يحسبه `Scaffold` مرّة**،
            // فلو أُضيف هنا `safeDrawing` لحُسب مرّتين وارتفع المحتوى.
            .statusBarsPadding(),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = 20.dp,
            end = 20.dp,
            bottom = 28.dp,
        ),
    ) {
        if (state.error.isNotEmpty()) {
            item {
                Column(Modifier.fillMaxWidth().padding(vertical = 12.dp)) {
                    Text(state.error, color = BrandOrange)
                    TextButton(onClick = actions.refresh) {
                        Text(stringResource(R.string.home_retry))
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **ورديّة مغلقة تُقال هنا لا تُترك فراغا**
        // ══════════════════════════════════════════════════════════════
        //
        // **من ورديّته مغلقة لا يصله عرض أبدا** — فيرى قائمة فارغة
        // **ويظنّ أنّ العمل راكد.** والسبب في يده وهو لا يعرفه.

        item {
            SectionTitle(
                stringResource(R.string.orders_offered),
                state.offers.size,
                Modifier.padding(top = 18.dp),
            )
        }
        if (state.offers.isEmpty()) {
            // **والفراغ لا يُترك فراغا** — يُقال سببه.
            item { WhyNoOrders(state) }
        } else {
            items(state.offers, key = { it.id }) { order ->
                OrderCard(
                    order = order,
                    offer = true,
                    avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
                    busy = state.acceptingId == order.id,
                    // **ولا يُقبل طلبان معا** — الضغطة الثانية أثناء الأولى
                    // تفتح نداءين، **وقد يعود الأوّل بالرفض والثاني بالقبول**
                    // فلا يعرف صاحبه ما الذي وقع.
                    enabled = state.acceptingId == null,
                    onAccept = { actions.accept(order.id) },
                    onOpen = null,
                    actionLabel = R.string.order_agree,
                    // ══════════════════════════════════════════════════
                    // **والرفض في الحالين** — (قرار المالك ٢٠٢٦-٠٨-١٢)
                    // ══════════════════════════════════════════════════
                    //
                    // **ومعناه يختلف**: في «بالدور» ينقل الدور فورا إلى
                    // من بعده، **وفي «للجميع» يخفيه عن شاشته ويبقى
                    // لغيره** — فلا يقرأ في كلّ تحديث طلبا لا يريده.
                    onDecline = { actions.decline(order.id) },
                    // **وانقضاء المهلة يعيد القراءة** — البطاقة لم تعد
                    // له، **ومن أبقاها** جعله يضغطها فيُردّ «سبقك غيرك».
                    onExpired = actions.refresh,
                )
            }
        }

        item {
            SectionTitle(
                stringResource(R.string.orders_mine),
                state.mine.size,
                Modifier.padding(top = 24.dp),
            )
        }
        if (state.mine.isEmpty()) {
            item { Empty(stringResource(R.string.orders_no_mine)) }
        } else {
            items(state.mine, key = { it.id }) { order ->
                OrderCard(
                    order = order,
                    offer = false,
                    avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
                    busy = false,
                    enabled = true,
                    // **وزرّ «ابدأ الرحلة» هو الفعل الوحيد هنا** — لا
                    // «تفاصيل» ولا صفحة سطور: **الرحلة هي التفاصيل.**
                    // (مواصفة المالك ٢٠٢٦-٠٨-١٢.)
                    onAccept = { actions.startTrip(order.id) },
                    onOpen = { actions.startTrip(order.id) },
                    actionLabel = R.string.trip_start,
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لماذا لا تصلني طلبات؟**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهذا أكثر ما يُسأل عنه المكتب** — والسائق يرى قائمة فارغة فيظنّ
 * العمل راكدا، **والسبب في يده وهو لا يعرفه.**
 *
 * **وكلّ الأسباب في `driver/me` اليوم** — لا نداء جديد ولا حقل جديد:
 * وردية · موقع · نقد بلغ سقفه · طلبات بعدد الحدّ.
 *
 * **وتُقرأ بالترتيب**: الأوّل المانع هو السبب، **ولا يُعرض خمسة أسباب
 * معا** فلا يعرف أيّها يعالج.
 */
@Composable
private fun WhyNoOrders(state: OrdersState) {
    val me = state.me
    val reason: Pair<Int, Boolean>? = when {
        me == null -> null
        !me.onShift -> R.string.why_shift_closed to true
        !state.locationOn -> R.string.why_location_off to true
        me.cashLimit > 0 && me.cashHeld >= me.cashLimit -> R.string.why_cash_full to true
        me.maxActiveOrders > 0 && me.activeOrders >= me.maxActiveOrders ->
            R.string.why_too_many to true

        else -> R.string.why_all_good to false
    }

    if (reason == null) {
        Empty(stringResource(R.string.orders_no_offers))
        return
    }

    val (text, blocking) = reason
    Box(
        Modifier
            .fillMaxWidth()
            .padding(top = 12.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(if (blocking) Color(0xFFFFF4E5) else Color(0xFFF5F7F8))
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(text),
            color = if (blocking) BrandOrange else InkMuted,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun SectionTitle(text: String, count: Int, modifier: Modifier = Modifier) {
    Row(modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(text, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
        Spacer(Modifier.size(8.dp))
        Text("(${grouped(count.toLong())})", color = InkMuted)
    }
}

@Composable
private fun Empty(text: String) {
    Text(
        text = text,
        color = InkMuted,
        modifier = Modifier.fillMaxWidth().padding(vertical = 14.dp),
    )
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة الطلب — كما رسمها المالك ٢٠٢٦-٠٨-١٢**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ستّة أسطر يقرؤها في ثانيتين وهو واقف**:
 *
 * ١ · **رأس**: «طلب جديد» · عدّاد المهلة · رقمه.
 * ٢ · **المتجر** باسمه.
 * ٣ · **من أين وإلى أين** بأسماء الأحياء — لا إحداثيات.
 * ٤ · **كم يبعد عنك** و**كم المشوار**.
 * ٥ · **أجرتك** و**كيف يُدفع** و**كم يستغرق**.
 * ٦ · **موافق** و**رفض**.
 *
 * # ولماذا اسم الحيّ لا العنوان الكامل
 *
 * **«حي الروضة» يعرفه السائق في لحظة** — والعنوان الكامل سطران يقرؤهما
 * بعد أن يقبل. **والقرار يحتاج الجهة لا الباب.**
 *
 * # وأجرتك غير إجماليّ الطلب
 *
 * **الأولى ما يكسبه، والثاني ما يقبضه للمتجر** — وكان يرى الثاني وحدَه
 * فيظنّه كسبه.
 */
@Composable
private fun OrderCard(
    order: DriverOrder,
    offer: Boolean,
    busy: Boolean,
    enabled: Boolean,
    avgSpeedKmh: Long,
    onAccept: () -> Unit,
    onOpen: (() -> Unit)?,
    actionLabel: Int,
    onDecline: (() -> Unit)? = null,
    onExpired: () -> Unit = {},
) {
    Column(
        Modifier
            .fillMaxWidth()
            .padding(top = 10.dp)
            .clip(RoundedCornerShape(18.dp))
            .border(1.dp, Color(0xFFE3E8EB), RoundedCornerShape(18.dp))
            .background(Color.White)
            .then(if (onOpen != null) Modifier.clickable(onClick = onOpen) else Modifier)
            .padding(16.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (offer) {
                Chip(R.drawable.ic_bell, stringResource(R.string.card_new_order))
            } else {
                Text(statusText(order.status), color = BrandTeal, fontWeight = FontWeight.Bold)
            }
            Text("#" + order.number, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }

        val expiresAt = order.offerExpiresAt
        if (offer && expiresAt != null) {
            Spacer(Modifier.height(8.dp))
            Countdown(expiresAt, onExpired = onExpired)
        }

        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(R.drawable.ic_store),
                contentDescription = null,
                tint = BrandOrange,
                modifier = Modifier.size(22.dp),
            )
            Spacer(Modifier.size(8.dp))
            Text(
                text = order.merchantName.ifBlank { stringResource(R.string.card_custom) },
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (onOpen != null) {
                Icon(
                    painter = painterResource(R.drawable.ic_chevron),
                    contentDescription = null,
                    tint = InkMuted,
                    modifier = Modifier.size(20.dp),
                )
            }
        }

        Spacer(Modifier.height(10.dp))
        Leg(
            label = stringResource(R.string.card_from_store),
            value = order.pickupArea.ifBlank { order.merchantName },
            tint = BrandTeal,
        )
        Leg(
            label = stringResource(R.string.card_to_customer),
            value = order.dropoffArea.ifBlank { order.addressText },
            tint = BrandOrange,
        )

        if (order.toPickupM >= 0 || order.legM >= 0) {
            Spacer(Modifier.height(12.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                if (order.toPickupM >= 0) {
                    Metric(
                        icon = R.drawable.ic_arrow_send,
                        label = stringResource(R.string.card_to_you),
                        value = distance(order.toPickupM),
                        modifier = Modifier.weight(1f),
                    )
                }
                if (order.legM >= 0) {
                    Metric(
                        icon = R.drawable.ic_arrow_send,
                        label = stringResource(R.string.order_leg),
                        value = distance(order.legM),
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        }

        Spacer(Modifier.height(8.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Metric(
                icon = R.drawable.ic_money,
                label = stringResource(R.string.card_fee),
                value = money(order.deliveryFee),
                strong = true,
                modifier = Modifier.weight(1f),
            )
            Metric(
                icon = R.drawable.ic_money,
                label = stringResource(R.string.card_payment),
                value = stringResource(
                    if (order.cashDue > 0) R.string.card_cash else R.string.card_wallet,
                ),
                modifier = Modifier.weight(1f),
            )
            // **والمدّة تُحسب ولا تُترك فارغة** — إلّا إن لم تُضبط السرعة:
            // **رقم مبنيّ على صفر يقول «الآن»** وهو وعد كاذب.
            val minutes = eta(order.toPickupM, order.legM, avgSpeedKmh)
            if (minutes > 0) {
                Metric(
                    icon = R.drawable.ic_time,
                    label = stringResource(R.string.card_eta),
                    value = stringResource(R.string.card_minutes, minutes),
                    modifier = Modifier.weight(1f),
                )
            }
        }

        if (offer) {
            Spacer(Modifier.height(12.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    painter = painterResource(R.drawable.ic_info),
                    contentDescription = null,
                    tint = InkMuted,
                    modifier = Modifier.size(16.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = stringResource(R.string.card_waiting),
                    color = InkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }

        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Button(
                onClick = onAccept,
                enabled = enabled && !busy,
                modifier = Modifier.weight(if (onDecline != null) 2f else 1f),
            ) {
                if (busy) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Icon(
                        painter = painterResource(R.drawable.ic_check_circle),
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.size(6.dp))
                    Text(stringResource(actionLabel))
                }
            }
            if (onDecline != null) {
                OutlinedButton(
                    onClick = onDecline,
                    enabled = enabled && !busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(stringResource(R.string.order_decline), color = BrandOrange)
                }
            }
        }
    }
}

/**
 * **ما بقي من مهلة العرض** — يُعدّ كلّ ثانية.
 *
 * **ويحمرّ في آخر عشر ثوان** — القرار صار عاجلا، **ولون واحد طوال
 * المهلة لا يقول ذلك.**
 *
 * **وتاريخ لا يُقرأ لا يُعرض** — المحرّك يرسل `ISO-8601`، **ومن سقط
 * تحليله** يُترك السطر فارغا بدل «صفر ثانية» كاذبة.
 */
@Composable
private fun Countdown(expiresAt: String, onExpired: () -> Unit) {
    val end = remember(expiresAt) {
        runCatching { java.time.Instant.parse(expiresAt).toEpochMilli() }.getOrNull()
    } ?: return

    var left by remember(expiresAt) { mutableStateOf(end - System.currentTimeMillis()) }
    LaunchedEffect(expiresAt) {
        while (left > 0) {
            kotlinx.coroutines.delay(1000)
            left = end - System.currentTimeMillis()
        }
        onExpired()
    }

    if (left <= 0) {
        Text(stringResource(R.string.offer_expired), color = InkMuted)
        return
    }

    val seconds = (left / 1000).toInt()
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(R.drawable.ic_time),
            contentDescription = null,
            tint = if (seconds <= 10) BrandOrange else BrandTeal,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = stringResource(R.string.offer_left, "%d:%02d".format(seconds / 60, seconds % 60)),
            color = if (seconds <= 10) BrandOrange else BrandTeal,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

/** **شارة صغيرة** — أيقونة وكلمة. */
@Composable
private fun Chip(icon: Int, text: String) {
    Row(
        Modifier
            .clip(RoundedCornerShape(20.dp))
            .background(BrandTeal)
            .padding(horizontal = 10.dp, vertical = 5.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = text,
            color = Color.White,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Bold,
        )
    }
}

/** **طرف الرحلة** — دبّوس واسم حيّ. */
@Composable
private fun Leg(label: String, value: String, tint: Color) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 3.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.size(8.dp))
        Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.size(8.dp))
        Text(value, fontWeight = FontWeight.Bold, maxLines = 1)
    }
}

/** **رقم بعنوانه** — في صندوق صغير. */
@Composable
private fun Metric(
    icon: Int,
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    strong: Boolean = false,
) {
    Column(
        modifier
            .clip(RoundedCornerShape(12.dp))
            .background(Color(0xFFF5F7F8))
            .padding(vertical = 10.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = if (strong) BrandTeal else InkMuted,
                modifier = Modifier.size(14.dp),
            )
            Spacer(Modifier.size(4.dp))
            Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(2.dp))
        Text(
            text = value,
            fontWeight = FontWeight.Bold,
            color = if (strong) BrandTeal else MaterialTheme.colorScheme.onSurface,
            maxLines = 1,
        )
    }
}

/**
 * **المدّة المتوقّعة للرحلة كلّها** — إليه ثمّ إلى الزبون.
 *
 * **وصفر يعني لا تُعرض**: سرعة غير مضبوطة أو مسافة مجهولة، **ورقم مبنيّ
 * على صفر وعد كاذب.**
 */
private fun eta(toPickupM: Double, legM: Double, avgSpeedKmh: Long): Long {
    if (avgSpeedKmh <= 0) return 0
    val meters = (if (toPickupM > 0) toPickupM else 0.0) + (if (legM > 0) legM else 0.0)
    if (meters <= 0) return 0
    return ((meters / 1000.0) / avgSpeedKmh * 60).toLong().coerceAtLeast(1)
}

/**
 * **المسافة بالمتر أو بالكيلومتر** — لا «1400 م».
 *
 * **والسائق يقدّر بالكيلومتر فوق الألف** — ورقم بأربع خانات يُقرأ مرّتين.
 */
private fun distance(meters: Double): String {
    val m = meters.toLong()
    return if (m < 1000) "$m م" else "${"%.1f".format(m / 1000.0)} كم"
}

/**
 * **حال الطلب بالعربيّة.**
 *
 * **ومجهول يُعرض رمزه** — لا يُبتلع في «قيد التنفيذ»: **حال جديد في
 * المحرّك يظهر هنا فيُعرف ويُترجم**، بدل أن يختفي تحت كلمة عامّة.
 */
@Composable
private fun statusText(status: String): String = when (status) {
    "assigned" -> stringResource(R.string.status_assigned)
    "picked_up" -> stringResource(R.string.status_picked_up)
    "on_the_way" -> stringResource(R.string.status_on_the_way)
    "arrived" -> stringResource(R.string.status_arrived)
    "dispatching" -> stringResource(R.string.status_dispatching)
    else -> status
}

/** ما تعرضه الشاشة — **ولا تملكه هي.** */
data class OrdersState(
    val offers: List<DriverOrder> = emptyList(),
    val mine: List<DriverOrder> = emptyList(),
    /** حال السائق كاملا — **منه تُبنى إجابة «لماذا لا تصلني طلبات».** */
    val me: DriverMe? = null,
    val locationOn: Boolean = true,
    val loading: Boolean = true,
    /** الطلب الذي يُقبل الآن — **وفارغ يعني لا شيء قيد القبول.** */
    val acceptingId: String? = null,
    val error: String = "",
)

data class OrdersActions(
    val accept: (String) -> Unit,
    val decline: (String) -> Unit,
    /** **يفتح رحلته** — يختار الطلب وينتقل إلى الخريطة. */
    val startTrip: (String) -> Unit,
    val refresh: () -> Unit,
)
