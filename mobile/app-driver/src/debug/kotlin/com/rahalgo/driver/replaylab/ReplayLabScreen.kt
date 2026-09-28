package com.rahalgo.driver.replaylab

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.navigation.ManeuverKinds
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.RerouteStatus
import com.rahalgo.navigation.TripMap
import com.rahalgo.navigation.MarkerIcons
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ مختبر الإعادة** — خريطةُ الملاحةِ الحقيقيّةُ + لوحةٌ + أزرارُ QA
 * ══════════════════════════════════════════════════════════════════════
 *
 * **تقرأ حالةَ `NavigationSession` مباشرةً** (`render`/`nav`/`rerouteStatus`)
 * فما يُرى هو ما يراه السائقُ الحقيقيّ. **بناءُ التطوير وحدَه.**
 */
@Composable
internal fun ReplayLabScreen(c: ReplayLabController) {
    val render = c.session.render
    val nav = c.session.nav
    val driver = render?.let { r -> r.targetLat?.let { la -> r.targetLng?.let { lo -> LatLng(la, lo) } } }
    val routeLine = c.session.route?.geometry?.map { LatLng(it.lat, it.lng) } ?: emptyList()
    val toCustomer = c.phase == ReplayLabController.Phase.TO_CUSTOMER || c.phase == ReplayLabController.Phase.ARRIVED
    val merchant = LatLng(c.scenario.merchant.lat, c.scenario.merchant.lng)
    val customer = LatLng(c.scenario.customer.lat, c.scenario.customer.lng)

    Box(Modifier.fillMaxSize()) {
        TripMap(
            icons = MarkerIcons(
                driver = R.drawable.ic_moto,
                pickup = R.drawable.ic_store,
                dropoff = R.drawable.ic_pin,
            ),
            driver = driver,
            pickup = if (toCustomer) null else merchant,
            dropoff = if (toCustomer) customer else null,
            route = routeLine,
            follow = true,
            nav = render,
            modifier = Modifier.fillMaxSize(),
        )

        // ── لوحةٌ علويّة: التوجيه · المسافة · الزمن · الحالة ──
        Column(
            Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(12.dp),
        ) {
            HudCard(
                title = when (c.phase) {
                    ReplayLabController.Phase.IDLE -> "مختبر الرحلة التجريبيّة"
                    ReplayLabController.Phase.TO_MERCHANT -> "الطريق إلى المتجر"
                    ReplayLabController.Phase.TO_CUSTOMER -> "الطريق إلى الزبون"
                    ReplayLabController.Phase.ARRIVED -> "انتهت الرحلة"
                },
                instruction = maneuverText(nav?.currentManeuver),
                distance = distanceText(nav?.remainingM),
                eta = etaText(nav?.remainingSec, nav?.remainingM),
                offRoute = nav?.isOffRoute == true,
                reroute = c.session.rerouteStatus,
            )
        }

        // ── لوحةٌ سفليّة: المشهد + السرعة + الأزرار ──
        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(Color(0xF20E1116))
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            // مُنتقي المشهد (قبل البدء)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                c.scenarios().forEach { s ->
                    FilterChip(
                        selected = c.scenario.id == s.id,
                        onClick = { c.selectScenario(s) },
                        enabled = !c.running,
                        label = { Text(s.title) },
                    )
                }
            }
            // السرعة
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("السرعة:", color = Color.White)
                listOf(1, 2, 5).forEach { x ->
                    FilterChip(
                        selected = c.speed == x,
                        onClick = { c.changeSpeed(x) },
                        label = { Text("${x}x") },
                    )
                }
            }
            // الأزرار
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                if (!c.running) {
                    Button(onClick = { c.start() }, modifier = Modifier.weight(1f)) { Text("ابدأ") }
                } else if (c.paused) {
                    Button(onClick = { c.resume() }, modifier = Modifier.weight(1f)) { Text("استئناف") }
                } else {
                    Button(onClick = { c.pause() }, modifier = Modifier.weight(1f)) { Text("إيقاف مؤقّت") }
                }
                OutlinedButton(onClick = { c.restart() }, modifier = Modifier.weight(1f)) { Text("إعادة") }
                OutlinedButton(onClick = { c.stop() }, modifier = Modifier.weight(1f)) { Text("إيقاف") }
            }
        }
    }
}

@Composable
private fun HudCard(
    title: String,
    instruction: String,
    distance: String,
    eta: String,
    offRoute: Boolean,
    reroute: RerouteStatus,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(Color(0xF20E1116))
            .padding(14.dp),
    ) {
        Text(title, color = Color(0xFF7FD1AE), fontWeight = FontWeight.Bold)
        Spacer(Modifier.width(4.dp))
        Text(instruction, color = Color.White, fontWeight = FontWeight.Bold)
        Row(Modifier.fillMaxWidth().padding(top = 6.dp), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            Text("المسافة: $distance", color = Color(0xFFB6C2CF))
            Text("الوقت: $eta", color = Color(0xFFB6C2CF))
        }
        val status = when {
            reroute == RerouteStatus.REROUTING -> "يُعاد التوجيه…" to Color(0xFFF0B429)
            reroute == RerouteStatus.REROUTE_FAILED -> "تعذّرت إعادةُ التوجيه" to Color(0xFFE5484D)
            offRoute -> "خارج المسار" to Color(0xFFF0B429)
            else -> null
        }
        status?.let { (txt, col) ->
            Text(txt, color = col, fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 6.dp))
        }
    }
}

private fun maneuverText(m: NavManeuver?): String = when (m?.kind) {
    ManeuverKinds.DEPART -> "انطلق"
    ManeuverKinds.TURN_LEFT -> "انعطف يساراً"
    ManeuverKinds.TURN_RIGHT -> "انعطف يميناً"
    ManeuverKinds.SLIGHT_LEFT -> "مِل يساراً"
    ManeuverKinds.SLIGHT_RIGHT -> "مِل يميناً"
    ManeuverKinds.SHARP_LEFT -> "انعطف يساراً بحدّة"
    ManeuverKinds.SHARP_RIGHT -> "انعطف يميناً بحدّة"
    ManeuverKinds.STRAIGHT -> "تابع مستقيماً"
    ManeuverKinds.ROUNDABOUT, ManeuverKinds.EXIT_ROUNDABOUT -> "عند الدوّار"
    ManeuverKinds.ARRIVE -> "وصلت الوجهة"
    null -> "…"
    else -> "تابع المسار"
}

private fun distanceText(m: Double?): String {
    if (m == null || m < 0) return "—"
    return if (m >= 1000) "%.1f كم".format(m / 1000) else "${m.toInt()} م"
}

// **الزمنُ من المحرّك إن توفّر، وإلّا من المسافة** (المسارُ المُصطنَعُ بلا مدد):
// ~٨٫٣ م/ث سرعةُ الإعادة.
private fun etaText(sec: Double?, distM: Double?): String {
    val s = when {
        sec != null && sec > 0 -> sec.toInt()
        distM != null && distM >= 0 -> (distM / 8.3).toInt()
        else -> return "—"
    }
    val min = s / 60
    return if (min >= 1) "$min د ${s % 60} ث" else "$s ثانية"
}
