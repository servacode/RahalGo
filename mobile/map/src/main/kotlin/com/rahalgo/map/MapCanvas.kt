package com.rahalgo.map

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import com.rahalgo.ui.AppCore
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحُ الخريطة — والأسلوبُ من المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والويبُ يرسم بالأسلوب نفسِه** (`public/map-style.json`) — **فالمدينةُ
 * تبدو واحدةً في الشاشتين**، ولا يرى المندوبُ شارعاً بشكلٍ ويراه المكتبُ
 * بشكلٍ آخر.
 *
 * # ودورةُ حياة `MapView` تُدار بيدها
 *
 * **مكتبةُ الخرائط تكتب على القرص وتفتح خيوطا** — **ومن نسي `onDestroy`
 * تركها تعمل بعد أن تُغلق الشاشة**، فتستنزف البطّاريّةَ ولا يُرى سببُها.
 *
 * # ولا يُقرأ العنوانُ إلّا حين تستقرّ
 *
 * **`onCameraIdle` لا `onCameraMove`** — **ونداءٌ في كلّ إطارٍ يُغرق
 * المزوّد**: تحريكةٌ واحدةٌ بالإصبع تُطلق عشرين نداء.
 */
@Composable
fun MapCanvas(
    start: LatLng,
    onSettle: (LatLng) -> Unit,
    modifier: Modifier = Modifier,
    /**
     * **موضعٌ تقفز إليه** — من نتيجة بحث.
     *
     * **وفارغٌ يعني لا قفزة** — **ولو كانت رايةً منفصلةً لَنُسي إطفاؤها**
     * فتقفز الخريطةُ كلّما أُعيد رسمُها.
     */
    jumpTo: LatLng? = null,
    onJumped: () -> Unit = {},
) {
    val context = LocalContext.current
    val styleUrl = remember { AppCore.get().baseUrl + "/api/v1/public/map-style.json" }

    val view = remember {
        // **والتهيئةُ من `MapHost`** — كانت هنا وفي تطبيق السائق،
        // **ونسختان من تهيئةِ مكتبةٍ أصليّةٍ تفترقان يومَ يتبدّل
        // معاملُها.** (المرحلة ٠.)
        ensureMapLibre(context)
        MapView(context).apply {
            // **و`onCreate` تبقى هنا وحدَها** — `TripMap` لا تناديها،
            // **وتوحيدُهما تغييرُ سلوكٍ** وهو خارجَ المرحلة ٠.
            onCreate(null)
            getMapAsync { map ->
                map.setStyle(Style.Builder().fromUri(styleUrl)) {
                    map.moveCamera(CameraUpdateFactory.newLatLngZoom(start, 15.0))
                    // **وأوّلُ قراءةٍ عند الفتح** — فلا يقف أمام خريطةٍ
                    // بلا اسمٍ ينتظر أن يحرّكها.
                    onSettle(start)
                }
                map.addOnCameraIdleListener { onSettle(map.cameraPosition.target ?: start) }
            }
        }
    }

    // **والقفزةُ تُحرّك الكاميرا ثمّ تُطفأ** — و`onCameraIdle` بعدها
    // يقرأ العنوانَ من نفسه، **فلا نداءَ ثانياً هنا.**
    LaunchedEffect(jumpTo) {
        val to = jumpTo ?: return@LaunchedEffect
        view.getMapAsync { map ->
            map.animateCamera(CameraUpdateFactory.newLatLngZoom(to, 16.0))
        }
        onJumped()
    }

    MapLifecycle(view)

    AndroidView(factory = { view }, modifier = modifier)
}
