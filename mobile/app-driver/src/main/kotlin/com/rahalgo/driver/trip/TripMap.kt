package com.rahalgo.driver.trip

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import com.rahalgo.driver.data.Backend
import org.maplibre.android.MapLibre
import org.maplibre.android.WellKnownTileServer
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطة الرحلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٢ من `docs/DRIVER-APP-PLAN.md`.)
 *
 * # لماذا بلاطات نقطيّة لا متجهيّة
 *
 * **الويب يرسم بها** (`web/packages/ui/map.tsx`: `tile.openstreetmap.org`
 * ثمّ بدائل) — **فالمدينة تبدو واحدة في الشاشتين**، ولا يرى السائق شارعا
 * بشكل والمكتب يراه بشكل آخر.
 *
 * **ولا مفتاح ولا فاتورة** — بخلاف خرائط غوغل.
 *
 * # وأسلوب الرسم يُكتب هنا لا يُجلب من خادم
 *
 * **ملفّ الأسلوب في MapLibre يُنزَّل عادةً من مزوّد** — ونحن لا مزوّد
 * لنا، **فيُكتب المصدر والطبقة بأيدينا**: مصدر نقطيّ واحد وطبقة فوقه.
 * **ولا نداء ثالث لخادم أسلوب** قد يسقط فتبقى الشاشة رماديّة.
 *
 * # والعلامات بطبقات لا بإضافة
 *
 * **إضافة العلامات في MapLibre مكتبة ثانية** — وثلاث نقاط وخطّ لا
 * تستحقّها. **فتُرسم دوائر وخطّ من `GeoJSON`** مباشرة.
 */

/** مركز الرقّة — **يُفتح عليه حتّى تُعرف النقاط.** */
private val RAQQA = LatLng(35.9528, 39.0079)

/**
 * **أسلوب الخريطة — في ملفّ لا في نصّ داخل الشيفرة.**
 *
 * **والبلاطات من `openstreetmap.org`** كما في الويب. **وسياستهم تمنع
 * الاستعمال الثقيل** — فيوم يكبر عدد السائقين يُبدَّل العنوان بخادمنا،
 * **وهو سطر واحد في الملفّ.**
 *
 * # ولماذا ملفّ
 *
 * **تنزيل منطقة غير متّصلة يحتاج عنوان أسلوب** (`asset://`) لا نصّا:
 * **المنزِّل يقرأ الأسلوب ليعرف أيّ بلاطات يجلب.** ونصٌّ داخل الشيفرة لا
 * عنوان له.
 *
 * **وأرضيّة تحت البلاطات** — والخريطة سوداء حتّى تصل أوّل بلاطة، **وسوادٌ
 * يملأ الشاشة يُقرأ عطبا** لا انتظارا. (رآه المالك ٢٠٢٦-٠٨-١٢.)
 */
/**
 * ══════════════════════════════════════════════════════════════════════
 * **عنوانان للأسلوب — واحدٌ للعرض وآخرُ للتنزيل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والعارض يقرأ الأصول** (`asset://`) — **فيرسم بلا شبكة أصلا**، وهو
 * ما نريده: خريطة تعمل والإنترنت مقطوع.
 *
 * **والمنزّل يقرأ عنوانا شبكيّا وحدَه**: جُرّب `asset://` فردّ «تعذّر
 * تحليل العنوان»، وجُرّب `file://` فردّ مثلها (قيسا على الجهاز
 * ٢٠٢٦-٠٨-١٢) — **فبقي التنزيل على صفر بلا سبب ظاهر.** فيقرأ أسلوبه من
 * المحرّك.
 *
 * **والبلاطات واحدة في الاثنين** — وهي ما يُخزَّن ويُقرأ، **لا ملفّ
 * الأسلوب.** فما نزّله المنزّل يرسمه العارض.
 */
const val STYLE_ASSET = "asset://map-style.json"

/**
 * خريطة تعرض ثلاث نقاط وخطّ الرحلة.
 *
 * **وتُبنى مرّة واحدة** (`remember`) — ومن أعاد بناء `MapView` مع كلّ
 * تحديث موقع **أعاد تحميل البلاطات كلّها كلّ عشرين ثانية.**
 */
@Composable
fun TripMap(
    driver: LatLng?,
    pickup: LatLng?,
    dropoff: LatLng?,
    /** **أتلاحق الكاميرا صاحبَها؟** — زرّ السير على الخريطة. */
    follow: Boolean = false,
    /**
     * **عدّاد «ردّني إلى موضعي»** — يزيد مع كلّ ضغطة.
     *
     * **ولماذا عدّاد لا دالّة**: `AndroidView` لا تُنادى إلّا حين
     * يتبدّل شيءٌ مُمرَّرٌ إليها، **ورقمٌ يزيد أصدقُ إشارةٍ على ضغطة**
     * من رايةٍ تُرفع وتُنزَّل فتضيع إن ضُغط مرّتين.
     */
    recenter: Int = 0,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val view = remember { createMapView(context) }
    // **وما عولج لا يُعاد** — الدالّة تُنادى مع كلّ رسمٍ جديد.
    val handled = remember { intArrayOf(-1) }

    DisposableEffect(Unit) {
        view.onStart()
        view.onResume()
        onDispose {
            view.onPause()
            view.onStop()
            view.onDestroy()
        }
    }

    AndroidView(factory = { view }, modifier = modifier) { map ->
        map.getMapAsync { libre ->
            // ══════════════════════════════════════════════════════════
            // **والأسلوب يُحمَّل مرّة**
            // ══════════════════════════════════════════════════════════
            //
            // **كان يُعاد ضبطُه مع كلّ نبضة موقع** — و`setStyle` تهدم
            // الطبقات وتبنيها، **فترتجف الخريطة كلَّ عشرين ثانية.**
            val style = libre.style
            if (style == null) {
                libre.setStyle(Style.Builder().fromUri(STYLE_ASSET)) {
                    Markers.draw(context, it, driver, pickup, dropoff)
                    fitAll(libre, driver, pickup, dropoff)
                }
            } else {
                Markers.draw(context, style, driver, pickup, dropoff)
            }

            // **وردُّه إلى موضعه أوّلا** — ضغطةٌ صريحةٌ تسبق كلَّ سلوكٍ
            // تلقائيّ.
            if (recenter != handled[0]) {
                handled[0] = recenter
                if (driver != null) {
                    libre.easeCamera(CameraUpdateFactory.newLatLngZoom(driver, 16.5))
                } else {
                    fitAll(libre, driver, pickup, dropoff)
                }
            } else if (follow && driver != null) {
                // **والملاحقة تُقرّب** — من يسير يريد الشارع الذي تحته
                // لا المدينة كلَّها.
                libre.easeCamera(CameraUpdateFactory.newLatLngZoom(driver, 17.0))
            }
        }
    }
}

/**
 * **تُظهر النقاط كلَّها** — لا تلاحق السائق وحدَه.
 *
 * **ومن رأى نفسَه ولم ير المتجر** لا يعرف أيّ جهةٍ يمضي.
 */
private fun fitAll(
    libre: org.maplibre.android.maps.MapLibreMap,
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تهيئة المكتبة — قبل أيّ استعمال لها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكانت تُهيَّأ عند بناء الخريطة وحدَها** — فحين أُخفي تبويب الرحلة
 * (لا رحلة الآن) **صارت اللوحة تسأل عن المناطق المنزَّلة قبل أن تُهيَّأ
 * المكتبة**، فيسقط التطبيق:
 *
 *     MapLibreConfigurationException
 *
 * **ولم يظهر على المحاكي**: كنتُ أفتح تبويب الرحلة أوّلا فتُهيَّأ.
 * **وظهر على أوّل جهاز حقيقيّ** بلا رحلة (٢٠٢٦-٠٨-١٢).
 *
 * **فتُهيَّأ من مكان واحد يناديه الاثنان** — والمكتبة تتجاهل النداء
 * الثاني.
 *
 * **ولا مفتاح**: المفتاح فارغ لأنّ MapLibre لا يطلب حسابا. **أمّا نوع
 * خادم البلاطات فلا يقبل الفراغ** — مرّرتُه فارغا فسقط التطبيق عند أوّل
 * فتح للخريطة.
 */
fun ensureMapLibre(context: Context) {
    MapLibre.getInstance(context.applicationContext, null, WellKnownTileServer.MapLibre)
}

private fun createMapView(context: Context): MapView {
    ensureMapLibre(context)
    return MapView(context)
}
