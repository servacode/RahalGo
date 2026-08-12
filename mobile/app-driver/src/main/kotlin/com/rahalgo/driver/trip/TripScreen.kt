package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandCanvas
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkDeep
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

    // ══════════════════════════════════════════════════════════════════
    // **ثلاثة أزرارٍ على الخريطة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة.)
    //
    //	ردَّني    ←  الكاميرا تعود إلى موضعي بعد أن قلّبتُ الخريطة بيدي
    //	اتبعني   ←  تلاحقني وأنا أسير، فلا أمسّها كلَّ دقيقة
    //	الملاحة  ←  أخرج إلى تطبيق الملاحة — الطريق والصوت ليسا عندنا
    //
    // **والملاحقة مُطفأةٌ حتّى تُطلب**: الفتحةُ الأولى تُظهر النقاط
    // كلَّها — **من رأى نفسَه ولم ير وجهتَه** لا يعرف أيّ جهةٍ يمضي.
    var follow by rememberSaveable { mutableStateOf(false) }
    var recenter by rememberSaveable { mutableIntStateOf(0) }

    Box(Modifier.fillMaxSize()) {
        TripMap(
            driver = state.driver,
            // **وقبل الاستلام تُعرض النقطتان** — بعده تُطفأ نقطة المتجر:
            // **انتهى شأنه منها**، وخريطة فيها ما لم يعد يلزم تشوّش.
            pickup = if (state.step >= TripStep.PICKED_UP) null else state.pickup,
            dropoff = state.dropoff,
            follow = follow,
            recenter = recenter,
            modifier = Modifier.fillMaxSize(),
        )

        // ══════════════════════════════════════════════════════════════
        // **ولا شريطَ خطواتٍ فوق الخريطة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «بالرحلة الشريط العلويّ تبع الرحلة
        //  ألغه».)
        //
        // **وسبعُ خطواتٍ تُقرأ في كلّ نظرة** وهو لا يحتاج منها إلّا
        // واحدة: **ما الذي أفعله الآن** — وهي مكتوبةٌ في الزرّ أسفل
        // الشاشة بلفظها. **والباقي تاريخٌ ومستقبلٌ يزاحمان الخريطة.**
        Column(
            Modifier.align(Alignment.TopCenter).statusBarsPadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            // ══════════════════════════════════════════════════════════
            // **لوحُ الطور — إلى أين وكم بقي**
            // ══════════════════════════════════════════════════════════
            //
            // (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة: لوحٌ داكنٌ فوق الخريطة
            //  يقول «في الطريق إلى المتجر» وتحته المسافة والدقائق.)
            //
            // **والرحلة طوران**: إلى المتجر ثمّ — بعد الاستلام — إلى
            // الزبون. **ومن لا يعرف في أيّهما هو** يقرأ المسافة ولا
            // يعرف إلى أين هي.
            //
            // **وهو فوق الخريطة لا تحتها**: عينُه على الطريق، **وما
            // يُقرأ في نظرةٍ خاطفةٍ يكون في أعلى الشاشة** حيث لا يحجبه
            // إبهامٌ ولا يُطلب منه أن ينزل بعينه إلى أسفلها.
            PhaseRibbon(state)

            // ══════════════════════════════════════════════════════════
            // **ومن يحمل أكثر من طلب يرى محطّاته**
            // ══════════════════════════════════════════════════════════
            //
            // (البند العاشر في قائمة المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **والمعروض هو «التالي»** — والباقي يُضغط فيصير هو التالي:
            // **من حمل ثلاثة ولا يعرف أيّها أوّلا** يقرّر بالحدس، ويقف
            // في الشارع يقلّب.
            if (state.stops.size > 1) {
                StopsRow(stops = state.stops, current = order.id, onPick = actions.pickStop)
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **طلبٌ على طريقك — وأنت ماشٍ**
        // ══════════════════════════════════════════════════════════════
        //
        // (البند الحادي عشر في قائمة المالك ٢٠٢٦-٠٨-١٢.)
        //
        // **والمحرّك يحسبه أصلا** (`orders/sameroute.go`): المتجران خلال
        // ٨٠٠ متر والزبونان خلال ٢٠٠٠ — **فما يصل الطابور وأنت في رحلة
        // هو على طريقك فعلا.**
        //
        // **ولافتةٌ لا شاشة**: يقرؤها بطرف عينه وهو يقود، **ويأخذها أو
        // يتركها — وهو حرّ.**
        if (state.onRouteOffer != null) {
            OnRouteBanner(
                offer = state.onRouteOffer,
                busy = state.busy,
                onTake = { actions.takeOffer(state.onRouteOffer.id) },
                onDismiss = actions.dismissOffer,
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .statusBarsPadding()
                    .padding(top = 96.dp),
            )
        }

        Column(Modifier.align(Alignment.BottomCenter)) {
            MapButtons(
                follow = follow,
                onRecenter = { recenter++ },
                onFollow = { follow = !follow },
                onNavigate = actions.navigate,
            )
            TripCard(order = order, state = state, actions = actions)
        }
    }

    if (state.agreeOpen) {
        AgreeDialog(onConfirm = actions.agree, onDismiss = actions.dismissAgree)
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
 * ══════════════════════════════════════════════════════════════════════
 * **الاتّفاق على الطلب الخاصّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثمن البضاعة اختياريّ** — قد تكون أمانةً لا ثمن لها، **فيُترك فارغا
 * ويُقرأ صفرا.** **ومن ألزم برقم في كلّ طلب** جعل السائق يكتب ما ليس
 * صحيحا ليمضي.
 *
 * **وأجرة التوصيل لا تُترك**: هي حقّه، **وطلبٌ بلا أجرة اتّفاقٌ ناقص.**
 */
@Composable
private fun AgreeDialog(onConfirm: (Long, Long) -> Unit, onDismiss: () -> Unit) {
    var goods by remember { mutableStateOf("") }
    var fee by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.agree_title)) },
        text = {
            Column {
                Text(stringResource(R.string.agree_hint), color = InkMuted)
                Spacer(Modifier.height(10.dp))
                OutlinedTextField(
                    value = goods,
                    onValueChange = { goods = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_goods)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = fee,
                    onValueChange = { fee = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_fee)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = { onConfirm(goods.toLongOrNull() ?: 0L, fee.toLongOrNull() ?: 0L) },
                enabled = fee.isNotBlank(),
            ) {
                Text(stringResource(R.string.agree_confirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_cancel)) }
        },
    )
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

/**
 * **أزرارُ الخريطة الثلاثة** — فوق البطاقة وفي جهة الإبهام.
 *
 * **ولا تُترك عائمةً في وسط الخريطة**: يدُه على المقود، **وما يُضغط وهو
 * واقفٌ على إشارةٍ يكون في مرمى إبهامه** لا في منتصف الشاشة.
 *
 * **والملاحة برتقاليّةٌ مسمّاة**: هي الوحيدةُ التي تُخرجه من التطبيق،
 * **وخروجٌ لا يُنتظر** — فلا تشبه أختيها اللتين تحرّكان كاميرا.
 */
@Composable
private fun MapButtons(
    follow: Boolean,
    onRecenter: () -> Unit,
    onFollow: () -> Unit,
    onNavigate: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.End,
        verticalAlignment = Alignment.Bottom,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            MapButton(R.drawable.ic_my_location, R.string.map_recenter, onRecenter)
            Spacer(Modifier.height(10.dp))
            // **والملاحقةُ تُضاء حين تعمل** — زرٌّ يفعل شيئا مستمرّا
            // **ولا يقول إنّه يعمل** يُضغط مرّتين فيُطفأ وهو يُظنّ مشتعلا.
            MapButton(
                icon = R.drawable.ic_navigation,
                label = R.string.map_follow,
                onClick = onFollow,
                on = follow,
            )
            Spacer(Modifier.height(10.dp))
            MapButton(
                icon = R.drawable.ic_arrow_send,
                label = R.string.map_navigate,
                onClick = onNavigate,
                accent = true,
            )
        }
    }
}

/** **قرصٌ واحد** — أيقونةٌ في دائرةٍ ترتفع عن الخريطة بظلّها. */
@Composable
private fun MapButton(
    icon: Int,
    label: Int,
    onClick: () -> Unit,
    on: Boolean = false,
    accent: Boolean = false,
) {
    val ground = when {
        accent -> BrandOrange
        on -> BrandTeal
        else -> BrandCanvas
    }
    Box(
        Modifier
            .size(52.dp)
            .shadow(6.dp, CircleShape)
            .clip(CircleShape)
            .background(ground)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = stringResource(label),
            tint = if (accent || on) Color.White else InkDeep,
            modifier = Modifier.size(24.dp),
        )
    }
}

/**
 * **لوحُ الطور** — إلى أين يمشي الآن، وكم بقي مسافةً ووقتا.
 *
 * **وهو الجواب عن سؤالٍ واحد**: ما الذي أفعله الآن؟ — ولذلك سطرٌ واحدٌ
 * كبيرٌ فوق، **ورقمان صغيران تحته.**
 *
 * **والوقتُ يُحسب من سرعةٍ في المخزن** (`drivers.avg_speed_kmh`) لا من
 * رقمٍ في الشيفرة: **درّاجةٌ في الرقّة غيرُ سيّارةٍ في مدينةٍ أخرى.**
 */
@Composable
private fun PhaseRibbon(state: TripState) {
    val meters = state.remainingM
    Column(
        Modifier
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .clip(RoundedCornerShape(18.dp))
            .background(InkDeep)
            .padding(horizontal = 20.dp, vertical = 10.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(phaseLabel(state.step)),
            color = Color.White,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        if (meters >= 0) {
            Spacer(Modifier.height(4.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Chip(R.drawable.ic_pin, distanceText(meters))
                val minutes = eta(meters, state.avgSpeedKmh)
                if (minutes.isNotEmpty()) {
                    Spacer(Modifier.size(14.dp))
                    Chip(R.drawable.ic_time, minutes.substringAfter("~"))
                }
            }
        }
    }
}

/** **أيقونةٌ ورقم** — على الأرض الداكنة. */
@Composable
private fun Chip(icon: Int, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = BrandOrange,
            modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = text,
            color = Color.White,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/**
 * **اسمُ الطور** — والرحلةُ طوران لا سبعة.
 *
 * **وما قبل الاستلام كلُّه «إلى المتجر»**، وما بعده كلُّه «إلى الزبون»
 * — **والوقوفُ عند أحدهما طورٌ ثالثٌ قصير** يُقال لأنّ الفعل التالي
 * يختلف.
 */
private fun phaseLabel(step: TripStep): Int = when (step) {
    TripStep.AT_PICKUP -> R.string.trip_p_at_store
    TripStep.PICKED_UP, TripStep.TO_CUSTOMER -> R.string.trip_p_to_customer
    TripStep.AT_CUSTOMER -> R.string.trip_p_at_customer
    TripStep.DELIVERED -> R.string.trip_p_delivered
    else -> R.string.trip_p_to_store
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
 * **لافتة «طلب على طريقك».**
 *
 * **ولا تحجب الخريطة** — سطران وزرّان، **ومن ملأ الشاشة بعرضٍ وسائقُه
 * يقود** أجبره على قرارٍ في غير وقته.
 */
@Composable
private fun OnRouteBanner(
    offer: DriverOrder,
    busy: Boolean,
    onTake: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(BrandTeal)
            .padding(14.dp),
    ) {
        Text(
            text = stringResource(R.string.trip_on_route),
            color = Color.White,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = offer.merchantName + " · " + money(offer.cashDue),
            color = Color.White.copy(alpha = 0.9f),
        )
        Spacer(Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Button(onClick = onTake, enabled = !busy) {
                Text(stringResource(R.string.order_accept))
            }
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.trip_leave_offer), color = Color.White)
            }
        }
    }
}

/**
 * **محطّاته حين يحمل أكثر من طلب.**
 *
 * **والحاليّة معلّمة** — وما عداها يُضغط فينتقل إليه.
 */
@Composable
private fun StopsRow(stops: List<Stop>, current: String, onPick: (String) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(Color.White.copy(alpha = 0.94f))
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 12.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        for (stop in stops) {
            val now = stop.id == current
            Text(
                text = "#" + stop.number,
                color = if (now) Color.White else InkMuted,
                fontWeight = if (now) FontWeight.Bold else FontWeight.Normal,
                modifier = Modifier
                    .clip(RoundedCornerShape(10.dp))
                    .background(if (now) BrandTeal else Color(0xFFEFF2F4))
                    .clickable { onPick(stop.id) }
                    .padding(horizontal = 12.dp, vertical = 6.dp),
            )
        }
    }
}

/** محطّة في قائمة من يحمل أكثر من طلب. */
data class Stop(val id: String, val number: Long)

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

        // **وما بقي صعد إلى لوح الطور** — ولا يُكتب هنا ثانية:
        // **رقمان لشيءٍ واحدٍ في شاشةٍ واحدة** يُقرأ أحدهما شيئا آخر.

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

        // **واقتراحٌ لا فعل** — الزرّ نفسه تحته، **وهو من يضغطه.**
        if (state.nearDestination) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(
                    if (order.status == "assigned") R.string.trip_near_pickup
                    else R.string.trip_near_dropoff,
                ),
                color = BrandTeal,
                fontWeight = FontWeight.Bold,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        }

        Spacer(Modifier.height(14.dp))
        val next = nextAction(order.status)
        if (next != null) {
            Button(
                // **والتسليم يمرّ بالصورة إن طلبها المحرّك** — وإلّا
                // ردّ «يلزم إثبات» بعد أن ظنّ صاحبه أنّه أنهى.
                onClick = {
                    if (next.status == "delivered" && state.requirePhoto) {
                        actions.capture()
                    } else {
                        actions.step(next.status)
                    }
                },
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
            // **والملاحة صعدت إلى الخريطة** — قرصٌ برتقاليٌّ بجانب
            // «ردّني» و«تابعني». **وزرّان يفعلان الشيءَ نفسَه في شاشةٍ
            // واحدة** يجعلان صاحبَهما يسأل: أيّهما؟ وهو يقود.
            //
            // **وحديث الزبون من هنا** — لا رقم هاتف في الطرفين.
            TextButton(onClick = actions.chat, enabled = !state.busy) {
                Text(stringResource(R.string.trip_chat), color = BrandTeal)
            }
            // **والاتّفاق للطلب الخاصّ وحدَه** — العاديّ سعرُه معروف
            // سلفا، **وزرٌّ يظهر فيه يسأل عمّا لا يُسأل عنه.**
            if (order.kind == "custom") {
                TextButton(onClick = actions.askAgree, enabled = !state.busy) {
                    Text(stringResource(R.string.agree_button), color = BrandOrange)
                }
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

        // **وإعادة الطلب قبل أن يستلم البضاعة فقط** — بعدها هي في يده،
        // **والبضاعة لا تُعاد بضغطة زرّ.**
        if (order.status == "assigned" || order.status == "at_pickup") {
            TextButton(
                onClick = actions.release,
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.detail_release), color = InkMuted)
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
    /** **هل هو على بُعد خطوات من وجهته؟** — يُقترح ولا يُنفَّذ. */
    val nearDestination: Boolean = false,
    /** أتلزم صورة تسليم؟ — **يقرّره المحرّك** (`drivers.require_delivery_photo`). */
    val requirePhoto: Boolean = false,
    /** أنافذة الاتّفاق مفتوحة؟ — **للطلب الخاصّ وحدَه.** */
    val agreeOpen: Boolean = false,
    /** محطّاته كلّها — **وواحدةٌ منها هي المعروضة.** */
    val stops: List<Stop> = emptyList(),
    /** عرضٌ نزل وهو في رحلة — **وفارغ يعني لا عرض.** */
    val onRouteOffer: DriverOrder? = null,
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
    /** يفتح الكاميرا لصورة التسليم. */
    val capture: () -> Unit,
    val release: () -> Unit,
    val chat: () -> Unit,
    val askAgree: () -> Unit,
    val agree: (Long, Long) -> Unit,
    val dismissAgree: () -> Unit,
    val pickStop: (String) -> Unit,
    val takeOffer: (String) -> Unit,
    val dismissOffer: () -> Unit,
    val askFail: () -> Unit,
    val fail: (String) -> Unit,
    val dismissFail: () -> Unit,
    val navigate: () -> Unit,
    val toOrders: () -> Unit,
)
