package com.rahalgo.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import com.rahalgo.map.MapOverlayRegistry
import com.rahalgo.map.MapStyleRepository
import com.rahalgo.map.data.MapRuntime
import com.rahalgo.map.data.MapSourceResolver
import com.rahalgo.map.rememberMapSurface
import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapLibreMap
import org.maplibre.android.maps.Style

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطة الرحلة — نمطٌ متّجهٌ من مستودعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٢ و١٩ و٢٠ و٢١ و٢٢.)
 *
 * # **ما كان وما صار**
 *
 * **كانت تقرأ نمطاً راستراً من حزمة التطبيق** (`asset://map-style.json`)
 * **يجلب بلاطاتِه من `tile.openstreetmap.org`.** وذلك انتهى: **لا مسارَ
 * إنتاجٍ في أندرويد يبلغ راستراً عامّاً** (البند ٢)، **ويحرسه
 * `check-map-style-single.mjs`.**
 *
 * **والنمطُ الآن من `MapStyleRepository`** — هو نفسُه الذي يرسم به لوحُ
 * الالتقاط في تطبيقَي الزبون والمندوب.
 *
 * # **وإعادةُ تحميل النمط تمحو كلَّ ما فوقه**
 *
 * (البند ١٩، وسمّاه المالكُ «نقطةً شديدةَ الأهمّيّة».)
 *
 * **حين تبدَّل المصدرُ** — ذهبت الشبكةُ أو عادت — **تُحمَّل MapLibre
 * نمطاً جديداً وتُسقط كلَّ مصدرٍ وطبقةٍ أضفناها**: خطَّ المسار،
 * ودبّوسَ السائق، ونقطتَي الاستلام والتسليم.
 *
 * **ولا تردّ خطأً.** الشوارعُ تُرسم سليمةً **وفوقها لا شيء** — والسائقُ
 * يقود ولا يرى مسارَه.
 *
 * **فالرسمُ يُسجَّل مُعيداً في `surface.overlays`**، **ويُنادى السجلُّ
 * بعد كلّ `StyleLoaded`** لا مرّةً عند الإنشاء.
 *
 * # **والكاميرا تُحفظ عبر التبديل** (البند ٢١)
 *
 * **تحميلُ نمطٍ لا يُصفّر الكاميرا في MapLibre**، **لكنّ شيفرتَنا كانت
 * تنادي `fitAll` في `onStyleLoaded`.** فلو بُدِّل المصدرُ أثناء الملاحة
 * **لقفزت الكاميرا من أمام السائق إلى منظرٍ عامٍّ للرحلة كلِّها.**
 *
 * **فصار `fitAll` عند أوّل تحميلٍ وحدَه**، **والموضعُ يُلتقط ويُعاد**
 * فيما بعده.
 *
 * # **والملاحةُ لا تُصفَّر مع الخريطة** (البند ٢٠)
 *
 * **الخريطةُ عارضٌ لا حالة.** فـ`NavigationSession` و`RouteProgress`
 * و`VoicePlanner` و`WrongWayDetector` و`RerouteEngine` **كلُّها في
 * نموذج الشاشة**، **ولا يمسّها تحميلُ نمطٍ ولا إعادةُ إنشاء `MapView`.**
 *
 * # والعلامات بطبقات لا بإضافة
 *
 * **إضافة العلامات في MapLibre مكتبة ثانية** — وثلاث نقاط وخطّ لا
 * تستحقّها. **فتُرسم دوائر وخطّ من `GeoJSON`** مباشرة.
 */

/** مركز الرقّة — **يُفتح عليه حتّى تُعرف النقاط.** */
private val RAQQA = LatLng(35.9528, 39.0079)

@Composable
fun TripMap(
    icons: MarkerIcons,
    driver: LatLng?,
    pickup: LatLng?,
    dropoff: LatLng?,
    route: List<LatLng> = emptyList(),
    follow: Boolean = false,
    recenter: Int = 0,
    nav: NavRender? = null,
    /**
     * **بدائلُ تُرسم ولا يُلاحَ عليها** — المرحلة ٧، البند ٢٨.
     *
     * **`render-only data`** — لا تدخل `NavigationSession` ولا
     * `VoicePlanner` ولا الكاشفَين.
     */
    alternatives: List<AltRouteLayer.Drawable> = emptyList(),

    /**
     * **المعايَنُ الآن** — إغلاقُ واجهة ٧، البند ١٤.
     *
     * **يُبرَز بصريّاً ولا يُعتمد.** والمسارُ الفعّالُ يبقى واضحاً
     * **لأنّ السائقَ لم يعتمد البديلَ بعد.**
     */
    previewRouteId: String? = null,

    /**
     * **ضغطةٌ على خطٍّ بديل** — البند ٢٤.
     *
     * **تردّ معرِّفاً ولا تُغيّر ملاحة** — والقرارُ في آلة الحال.
     * **و`null` تعني: ضُغط على غير بديل** (البند ٢٧: إلغاءُ معاينة).
     */
    onRouteTapped: ((String?) -> Unit)? = null,
    online: Boolean = true,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val surface = rememberMapSurface()
    val view = surface.view
    val handled = remember { intArrayOf(-1) }
    val animator = remember { com.rahalgo.map.MarkerAnimator() }
    val shown = remember { doubleArrayOf(Double.NaN, Double.NaN) }
    val shownBearing = remember { floatArrayOf(0f) }
    val navKey = remember { longArrayOf(-1L) }

    /**
     * **أوّلُ تحميلٍ يُؤطّر، وما بعده يُحافظ** — البند ٢١.
     *
     * **ولا `fitAll` تلقائيٌّ بلا سبب**، خصوصاً في الملاحة.
     */
    val framed = remember { booleanArrayOf(false) }

    /** **الحالُ الأخيرُ للرسم** — يقرؤه المُعيدُ بعد تحميل النمط. */
    val latest = remember { arrayOfNulls<TripDraw>(1) }
    latest[0] = TripDraw(driver, pickup, dropoff, route, icons)

    /**
     * **وبدائلُ العرض ومعاينتُها** — تُعاد بعد كلّ تحميل نمط.
     *
     * **البند ٣٢**: «preview highlight restored… بدون تغيير
     * Navigation generation».
     */
    val latestAlts = remember { arrayOfNulls<List<AltRouteLayer.Drawable>>(1) }
    latestAlts[0] = alternatives
    val latestPreview = remember { arrayOfNulls<String>(1) }
    latestPreview[0] = previewRouteId

    /**
     * **والنمطُ الحاليُّ يُمسك حين يُحمَّل.**
     *
     * **ولا يُسأل عنه بـ`getMapAsync`** — تلك غيرُ متزامنة، **فتردّ
     * بعد أن يكون المُعيدُ قد انتهى ولم يرسم شيئاً.**
     */
    val styleRef = remember { arrayOfNulls<Style>(1) }

    /**
     * **الربطُ يُعاد حسابُه حين يتبدّل الاتّصال أو موضعُ السائق.**
     *
     * **ولا يُسأل في كلّ إطار** (البند ٤٢) — المفتاحُ هو `online`
     * وصندوقُ المسار، **وكلاهما يتبدّل نادراً.**
     */
    val routeBbox = remember(route) { bboxOf(route) }
    val binding = remember(online, routeBbox, driver?.latitude, driver?.longitude) {
        MapStyleRepository.bind(
            purpose = if (nav != null) {
                MapSourceResolver.Purpose.NAVIGATION
            } else {
                MapSourceResolver.Purpose.TRIP
            },
            online = online,
            lat = driver?.latitude ?: pickup?.latitude,
            lng = driver?.longitude ?: pickup?.longitude,
            routeBbox = routeBbox,
            navigating = nav != null,
        )
    }

    DisposableEffect(Unit) {
        onDispose { animator.cancel() }
    }

    /**
     * ══════════════════════════════════════════════════════════════
     * **لمسُ خطٍّ بديل** — البنود ٢٤ إلى ٢٧
     * ══════════════════════════════════════════════════════════════
     *
     * **ولا يستدعي `setRoute`** — يردّ معرِّفاً، **والقرارُ في آلة
     * الحال.** فلمسةٌ خاطئةٌ على مِقودٍ لا تبدّل ملاحةً وصوتاً.
     *
     * **ومنطقةُ اللمس ٤٨dp** لا عرضُ الخطّ — البند ٢٥.
     *
     * **و`null` تعني: لا بديلَ تحتها** — فتُلغى المعاينةُ إن كانت
     * (البند ٢٧).
     */
    val density = context.resources.displayMetrics.density
    DisposableEffect(surface, onRouteTapped) {
        val tap = onRouteTapped
        if (tap == null) return@DisposableEffect onDispose { }
        val listener = MapLibreMap.OnMapClickListener { point ->
            var handled = false
            view.getMapAsync { libre ->
                val screen = libre.projection.toScreenLocation(point)
                tap(AltRouteLayer.routeIdAt(libre, screen.x, screen.y, density))
                handled = true
            }
            // **ولا تُبتلع الضغطة** — الخريطةُ تبقى تعمل كما كانت.
            false
        }
        view.getMapAsync { it.addOnMapClickListener(listener) }
        onDispose { view.getMapAsync { it.removeOnMapClickListener(listener) } }
    }

    /**
     * **تسجيلُ مُعيدِ الطبقات** — مرّةً لهذه الشاشة.
     *
     * **بمعرِّفٍ ثابتٍ فلا يتراكم** مع إعادة التركيب.
     */
    DisposableEffect(surface) {
        surface.overlays.register(
            id = "trip-markers",
            priority = MapOverlayRegistry.Priority.ROUTE,
        ) {
            val style = styleRef[0] ?: return@register
            val draw = latest[0] ?: return@register
            Markers.draw(
                context, style, draw.driver, draw.pickup, draw.dropoff, draw.route, draw.icons,
            )
        }
        /**
         * **وطبقةُ البدائل** — بأولويّةٍ بين المناطق والمسار.
         *
         * **فتُرسم تحتَ الموصى به** ولا تخفيه (البند ٢٦).
         */
        surface.overlays.register(
            id = "trip-alternatives",
            priority = AltRouteLayer.PRIORITY,
        ) {
            val style = styleRef[0] ?: return@register
            AltRouteLayer.draw(style, latestAlts[0] ?: emptyList(), latestPreview[0])
        }
        onDispose {
            surface.overlays.unregister("trip-markers")
            surface.overlays.unregister("trip-alternatives")
        }
    }

    /**
     * **تحميلُ النمط** — عند أوّل مرّةٍ وعند كلّ تبديلِ ربط.
     *
     * **والكاميرا تُلتقط قبل التحميل وتُعاد بعده** (البند ٢١).
     */
    LaunchedEffect(binding) {
        val ready = binding as? MapRuntime.Binding.Ready ?: return@LaunchedEffect
        view.getMapAsync { libre ->
            val keep: CameraPosition? = if (framed[0]) libre.cameraPosition else null
            libre.setStyle(Style.Builder().fromJson(ready.bound.json)) { loaded ->
                styleRef[0] = loaded
                /**
                 * **وهنا يصير المطلوبُ محمَّلاً** — إغلاقُ ٦ب
                 * الوظيفيّ، البندان ٩ و١٢.
                 *
                 * **قبل هذا السطر الأرشيفُ محجوزٌ ولم يُفتح**؛
                 * **وبعده هو ما تقرأ منه الخريطة**، **فيُحرَّر
                 * القديمُ ويُكتب `active.json`.**
                 */
                MapStyleRepository.onStyleLoaded(binding)
                // **الطبقاتُ تُعاد أوّلاً** — فلا إطارَ واحدٌ بلا مسار.
                surface.overlays.restoreAll()
                if (keep != null) {
                    libre.moveCamera(CameraUpdateFactory.newCameraPosition(keep))
                } else {
                    framed[0] = true
                    fitAll(libre, driver, pickup, dropoff)
                }
            }
        }
    }

    AndroidView(factory = { view }, modifier = modifier) { map ->
        map.getMapAsync { libre ->
            val style = libre.style ?: return@getMapAsync
            // **البدائلُ أوّلاً** — فتبقى تحتَ الدبابيس.
            AltRouteLayer.draw(style, alternatives, previewRouteId)
            Markers.draw(context, style, driver, pickup, dropoff, route, icons)

            if (recenter != handled[0]) {
                handled[0] = recenter
                if (driver != null) {
                    libre.easeCamera(CameraUpdateFactory.newLatLngZoom(driver, 16.5))
                } else {
                    fitAll(libre, driver, pickup, dropoff)
                }
            } else if (nav != null && nav.targetLat != null && nav.targetLng != null) {
                if (nav.stepId != navKey[0]) {
                    navKey[0] = nav.stepId
                    val fromLat = if (shown[0].isNaN()) nav.targetLat else shown[0]
                    val fromLng = if (shown[1].isNaN()) nav.targetLng else shown[1]
                    val fromBearing = shownBearing[0]
                    val toBearing = nav.bearingDeg ?: fromBearing
                    val ms = if (shown[0].isNaN()) 0L else nav.durationMs
                    animator.animate(
                        fromLat, fromLng, nav.targetLat, nav.targetLng,
                        fromBearing, toBearing, ms,
                    ) { lat, lng, bearing ->
                        shown[0] = lat
                        shown[1] = lng
                        shownBearing[0] = bearing
                        libre.style?.let {
                            Markers.draw(
                                context, it, LatLng(lat, lng), pickup, dropoff, route,
                                icons, bearing,
                            )
                        }
                    }
                    val cam = nav.camera(com.rahalgo.map.CameraPrimitives.bearingOf(libre))
                    com.rahalgo.map.CameraPrimitives.ease(
                        libre, cam.lat, cam.lng, cam.zoom,
                        cam.bearingDeg, cam.tiltDeg, cam.durationMs,
                    )
                }
            } else if (follow && driver != null) {
                libre.easeCamera(CameraUpdateFactory.newLatLngZoom(driver, 17.0))
            }
        }
    }
}

/** **صورةُ ما يُرسم** — يقرؤها المُعيدُ بعد تحميل النمط. */
private data class TripDraw(
    val driver: LatLng?,
    val pickup: LatLng?,
    val dropoff: LatLng?,
    val route: List<LatLng>,
    val icons: MarkerIcons,
)

/**
 * **صندوقُ المسار** — البند ١٧.
 *
 * **يُسأل به: هل تغطّي الحزمةُ المحلّيّةُ المسارَ كلَّه؟** فإن خرج عنه
 * **والاتّصالُ متاحٌ فالأونلاين أولى**، ولا يُدَّعى أنّ الأساسَ كاملٌ
 * خارج المنطقة.
 */
internal fun bboxOf(route: List<LatLng>): List<Double>? {
    if (route.isEmpty()) return null
    var west = route[0].longitude
    var east = route[0].longitude
    var south = route[0].latitude
    var north = route[0].latitude
    for (p in route) {
        if (p.longitude < west) west = p.longitude
        if (p.longitude > east) east = p.longitude
        if (p.latitude < south) south = p.latitude
        if (p.latitude > north) north = p.latitude
    }
    return listOf(west, south, east, north)
}

private fun fitAll(
    libre: MapLibreMap,
    driver: LatLng?,
    pickup: LatLng?,
    dropoff: LatLng?,
) {
    val points = listOfNotNull(driver, pickup, dropoff)
    when {
        points.size >= 2 -> libre.easeCamera(
            CameraUpdateFactory.newLatLngBounds(LatLngBounds.fromLatLngs(points), 120),
        )
        points.size == 1 -> libre.easeCamera(CameraUpdateFactory.newLatLngZoom(points[0], 15.0))
        else -> libre.easeCamera(CameraUpdateFactory.newLatLngZoom(RAQQA, 13.0))
    }
}
