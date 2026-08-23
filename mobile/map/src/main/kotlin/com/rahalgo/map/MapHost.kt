package com.rahalgo.map

import android.content.Context
import android.os.Bundle
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import org.maplibre.android.MapLibre
import org.maplibre.android.WellKnownTileServer
import org.maplibre.android.maps.MapView

/**
 * ══════════════════════════════════════════════════════════════════
 * **حاملُ الخريطة — دورةُ حياةٍ واحدةٌ لكلّ الشاشات**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٢٨ و٢٩ و٣٠.)
 *
 * # **ما كان قبل هذا**
 *
 * `MapCanvas` **تستدعي `onCreate` داخل `remember`** و`TripMap` **لا
 * تستدعيها أصلاً** (`TD-MAP-ONCREATE`). **و`onSaveInstanceState`
 * و`onLowMemory` لا يصلان أيَّ واحدةٍ منهما** (`TD-MAP-LOWMEM`).
 *
 * **وMapLibre تشترط الدورةَ كاملة.** وما ينقص منها لا يظهر خطأً —
 * **يظهر تسريبَ ذاكرةٍ عند التدوير، وموتَ عمليّةٍ عند ضغط الذاكرة.**
 *
 * # **والدورةُ تُربط بمالك دورة الشاشة لا بـ`DisposableEffect`**
 *
 * **`DisposableEffect(Unit)` تُنادى عند التركيب وتُهدم عند الخروج** —
 * **وهي لا تعرف أنّ التطبيق صُغّر.** فكانت `onPause` تصل حين تُغلق
 * الشاشةُ لا حين يخرج المستخدم، **والخريطةُ تظلّ ترسم في الخلفيّة.**
 */

fun ensureMapLibre(context: Context) {
    MapLibre.getInstance(context.applicationContext, null, WellKnownTileServer.MapLibre)
}

/**
 * **خريطةٌ حيّةٌ بدورتها وسجلِّ طبقاتها.**
 *
 * **والسجلُّ معها لا بجانبها** — فلو صار عامّاً **لاختلطت طبقاتُ
 * شاشتين مفتوحتين.**
 */
class MapSurface internal constructor(
    val view: MapView,
    val overlays: MapOverlayRegistry,
) {
    internal var handle: MapLifecycleBridge.Handle? = null
}

/**
 * **يُنشئ الخريطةَ ويسجّلها في الجسر.**
 *
 * **`remember` بلا مفتاح** — فإعادةُ التركيب لا تُنشئ `MapView` ثانية
 * (البند ٣٠). **وهي أثقلُ شيءٍ في الشاشة**، ولو أُنشئت مع كلّ إعادةٍ
 * **لتسرّبت عشراتٌ منها في دقيقة.**
 */
@Composable
fun rememberMapSurface(): MapSurface {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current

    val surface = remember {
        ensureMapLibre(context)
        MapSurface(MapView(context), MapOverlayRegistry())
    }

    DisposableEffect(surface) {
        val handle = MapLifecycleBridge.register(MapViewHost(surface.view))
        surface.handle = handle
        handle.create()

        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> handle.start()
                Lifecycle.Event.ON_RESUME -> handle.resume()
                Lifecycle.Event.ON_PAUSE -> handle.pause()
                Lifecycle.Event.ON_STOP -> {
                    // **والحالُ يُحفظ قبل الإيقاف** — فالنظامُ قد لا
                    // يعود إلينا، **وموضعُ الكاميرا يضيع.**
                    handle.saveInstanceState()
                    handle.stop()
                }
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)

        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
            surface.overlays.clear()
            handle.destroy()
            surface.handle = null
        }
    }

    return surface
}

/**
 * **الوصلةُ إلى `MapView`** — سطرٌ لكلّ حدث.
 *
 * **ولا منطقَ هنا** — المنطقُ في `MapLifecycleBridge` وهو مُختبَر.
 */
private class MapViewHost(private val view: MapView) : MapLifecycleBridge.Host {
    private val state = Bundle()

    override fun onCreate() = view.onCreate(null)
    override fun onStart() = view.onStart()
    override fun onResume() = view.onResume()
    override fun onPause() = view.onPause()
    override fun onStop() = view.onStop()
    override fun onDestroy() = view.onDestroy()
    override fun onLowMemory() = view.onLowMemory()
    override fun onSaveInstanceState() {
        view.onSaveInstanceState(state)
    }
}

/**
 * **يوصل ضغطَ الذاكرة إلى كلّ خريطةٍ حيّة** — البند ٢٩.
 *
 * **يُنادى من `Application.onLowMemory` و`onTrimMemory`** — والتطبيقُ
 * هو الذي يسمع النظام، **والخرائطُ لا تسمعه.**
 */
fun dispatchMapLowMemory() = MapLifecycleBridge.dispatchLowMemory()

/** **عددُ الخرائط الحيّة** — يُقرأ لكشف التسريب. */
fun liveMapCount(): Int = MapLifecycleBridge.liveCount()

/**
 * **وما بقي للتوافق** — تُستعمل حيث لم يُرحَّل بعد.
 *
 * **والدورةُ فيها كاملةٌ الآن** — فلا شاشةَ بلا `onCreate`.
 */
@Composable
fun rememberMapView(): MapView = rememberMapSurface().view

@Composable
@Deprecated(
    "الدورةُ صارت في rememberMapSurface — ولا تُدار من الشاشة",
    ReplaceWith("rememberMapSurface()"),
)
fun MapLifecycle(view: MapView) {
    // **مقصودٌ ألّا تفعل شيئاً** — الدورةُ تُدار في `rememberMapSurface`.
    // **ولو أدارتها هنا أيضاً لوصل كلُّ حدثٍ مرّتين.**
}
