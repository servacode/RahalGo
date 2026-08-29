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

    // ══════════════════════════════════════════════════════════════════
    // **والخريطةُ تعيش أطولَ من الشاشة التي تعرضها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «عندما عدتُ يوجد ثقلٌ أيضاً، والخطُّ
    //  والحركةُ تصبح أبطأ قليلاً لتستعيد نفسها».)
    //
    // **و`remember` تموت بخروج الشاشة من التركيب** — فكلُّ تبديلِ
    // تبويبٍ كان **يهدم `MapView` ويبنيها**: نمطٌ يُحمَّل من جديد،
    // وبلاطاتٌ تُجلب من أوّلها، وطبقاتٌ تُركَّب. **وذاك ثمنُ ثانيةٍ
    // كاملةٍ يراها السائقُ تلعثماً.**
    //
    // **وهي أثقلُ شيءٍ في التطبيق** — فتُحفظ للنشاط لا للشاشة.
    //
    // **ولا تُسرَّب**: تُهدم مع النشاط نفسِه (`onDispose` أدناه حين
    // يُهدم فعلاً لا حين تُبدَّل شاشة).
    val surface = remember(context) {
        MapSurfaceCache.of(context) {
            ensureMapLibre(context)
            MapSurface(MapView(context), MapOverlayRegistry()).also { made ->
            // ══════════════════════════════════════════════════════════
            // **ولا شعارَ ولا زرَّ إسنادٍ ولا بوصلةٍ فوق الخريطة**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-٢٤: «افعل الرخصة بشكلٍ صحيح بدون
            //  أن يكون واضحاً على خريطتنا — من أجل أن تبقى ميزاتُ
            //  خريطتنا مخفيّة».)
            //
            // # والرخصةُ تُؤدَّى ولا تُلغى
            //
            // **رخصةُ MapLibre لا تُلزم بشعار** — هي BSD، **والشعارُ
            // اختيارٌ يضعه المحرّك.**
            //
            // **ورخصةُ OpenStreetMap تُلزم بذكر المساهمين** (ODbL)
            // **وبياناتُ خريطتنا كلُّها منها** — فنُقل الذكرُ إلى
            // «حسابي» (`MapCredit`): مؤدّىً كاملاً وغيرُ مزاحمٍ للطريق.
            // **ومن حذفه من الموضعين خالف رخصة.**
            //
            // # وهنا لا في `MapCanvas`
            //
            // **وشاشةُ الرحلة تُنشئ خريطتَها بمسارٍ آخر** (`TripMap`)
            // — **فإطفاءٌ في مسارٍ واحدٍ لا يشمل الثاني**، وقِيس ذلك
            // بلقطةٍ من جهاز المالك ٢٠٢٦-٠٨-٢٤: الشعارُ ظاهرٌ والإطفاءُ
            // مكتوب. **وهنا حيث تُولد `MapView` فيشمل كلَّ خريطة.**
            //
            // **والبوصلةُ تُطفأ كذلك**: خريطةُ الملاحة تدور مع السائق
            // **فتظهر البوصلةُ دائماً**، وله زرُّ «ردّني إلى موقعي»
            // يفعل ما تفعله وأكثر.
                made.view.getMapAsync { map ->
                    map.uiSettings.isLogoEnabled = false
                    map.uiSettings.isAttributionEnabled = false
                    map.uiSettings.isCompassEnabled = false
                }
            }
        }
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
            // **ولا تُهدم الخريطةُ بخروج الشاشة** — انظر
            // `MapSurfaceCache`: **تُهدم بموت النشاط وحدَه.**
            // **والطبقاتُ تُنظَّف** فلا تبقى طبقةُ شاشةٍ على خريطةِ
            // شاشةٍ أخرى.
            surface.overlays.clear()
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مخزنُ الخرائط — واحدةٌ لكلّ نشاط**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤.)
 *
 * **و`MapView` أثقلُ شيءٍ في التطبيق**: نمطٌ ومصادرُ وطبقاتٌ وسطحُ
 * رسمٍ أصليّ. **وبناؤها عند كلّ تبديلِ تبويبٍ ثمنُ ثانيةٍ يراها
 * السائقُ تلعثماً.**
 *
 * **ومفتاحُه النشاطُ لا التطبيق** — **وخريطةٌ تبقى بعد موت نشاطها
 * تحمل سياقاً ميّتاً**، وهو تسرّبٌ لا يظهر إلّا في جهازٍ يسخن.
 *
 * **ويُنظَّف عند الهدم** — `release`.
 */
object MapSurfaceCache {

    private val held = HashMap<Int, MapSurface>()

    fun of(context: android.content.Context, make: () -> MapSurface): MapSurface {
        val activity = activityOf(context) ?: return make()
        val key = System.identityHashCode(activity)
        held[key]?.let { return it }
        val made = make()
        held[key] = made
        // **ويُهدم مع نشاطه** — فلا يبقى سطحٌ بلا نافذة.
        activity.application.registerActivityLifecycleCallbacks(
            object : android.app.Application.ActivityLifecycleCallbacks {
                override fun onActivityDestroyed(a: android.app.Activity) {
                    if (a !== activity || a.isChangingConfigurations) return
                    activity.application.unregisterActivityLifecycleCallbacks(this)
                    held.remove(key)?.let { dead ->
                        dead.overlays.clear()
                        dead.handle?.destroy()
                        dead.handle = null
                    }
                }

                override fun onActivityCreated(a: android.app.Activity, b: android.os.Bundle?) = Unit
                override fun onActivityStarted(a: android.app.Activity) = Unit
                override fun onActivityResumed(a: android.app.Activity) = Unit
                override fun onActivityPaused(a: android.app.Activity) = Unit
                override fun onActivityStopped(a: android.app.Activity) = Unit
                override fun onActivitySaveInstanceState(a: android.app.Activity, b: android.os.Bundle) = Unit
            },
        )
        return made
    }

    private fun activityOf(context: android.content.Context): android.app.Activity? {
        var c: android.content.Context? = context
        while (c is android.content.ContextWrapper) {
            if (c is android.app.Activity) return c
            c = c.baseContext
        }
        return null
    }
}
