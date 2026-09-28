package com.rahalgo.driver.replaylab

import android.app.Application
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.ViewModelProvider
import com.rahalgo.driver.trip.TripActions
import com.rahalgo.driver.trip.TripScreen

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرحلة التجريبيّة المواجِهةُ للمالك — على شاشة الرحلة الحقيقيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بناءُ التطوير وحدَه** (`src/debug`، معلَنٌ في بيان التطوير). يعرض
 * `TripScreen` نفسَها التي يراها السائق، مقودةً بمحرّك الإعادة. **شارةُ
 * «تجريبيّ» صغيرةٌ فقط** — وما عداها تجربةُ سائقٍ حقيقيّة. معزولٌ ماليّاً/شبكيّاً.
 */
class ReplayTripActivity : ComponentActivity() {

    internal class Model(app: Application) : AndroidViewModel(app) {
        val c = ReplayTripController(app)
        override fun onCleared() { c.dispose() }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // **تركيبُ النواة** كما تفعل الشاشةُ الرئيسيّة — فـ`TripScreen`
        // (زرُّ الإعادة) يقرأ `AppCore.get()`، ويُطلَق هذا النشاطُ مباشرةً
        // دونها. لا نداءَ شبكةٍ هنا؛ مجرّدُ تركيبِ العميل.
        runCatching { com.rahalgo.driver.data.Backend.of(applicationContext) }
        val model = ViewModelProvider(this)[Model::class.java]
        val c = model.c
        setContent {
            MaterialTheme {
                Box(Modifier.fillMaxSize()) {
                    TripScreen(
                        state = c.tripState(),
                        actions = TripActions(
                            step = { c.onStep(it) },
                            capture = {},
                            release = {},
                            chat = {},
                            askAgree = {},
                            agree = { _, _ -> },
                            dismissAgree = {},
                            pickStop = {},
                            takeOffer = {},
                            dismissOffer = {},
                            askFail = {},
                            fail = {},
                            dismissFail = {},
                            problem = {},
                            emergency = {},
                            retryEmergency = {},
                            dismissEmergency = {},
                            toOrders = { finish() },
                        ),
                        voice = c.voice,
                        navSession = c.session,
                        following = true,
                        // **زرُّ الرحلة التجريبيّة داخل الشاشة يبدأ/يوقف القيادة نفسَها.**
                        onReplay = { fixes -> if (fixes.isEmpty()) c.stop() else c.start() },
                    )
                    DebugBadge(c)
                }
            }
        }
        if (savedInstanceState == null) c.start()
    }
}

/** شارةٌ صغيرةٌ غيرُ حاجبة: وسمُ «تجريبيّ» + سرعةٌ + إيقافٌ مؤقّت. */
@Composable
private fun DebugBadge(c: ReplayTripController) {
    Row(
        Modifier
            .statusBarsPadding()
            .padding(top = 2.dp, start = 8.dp, end = 8.dp)
            .clip(RoundedCornerShape(10.dp))
            .background(Color(0xCC0E1116))
            .padding(horizontal = 8.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text("${c.kmh()} كم/س", color = Color(0xFF7FD1AE), fontWeight = FontWeight.Bold)
        listOf(1, 2, 3).forEach { x ->
            FilterChip(selected = c.speed == x, onClick = { c.changeSpeed(x) }, label = { Text("${x}x") })
        }
        val pausedNow = c.paused
        FilterChip(
            selected = pausedNow,
            onClick = { if (pausedNow) c.resume() else c.pause() },
            label = { Text(if (pausedNow) "استئناف" else "إيقاف") },
        )
    }
}
