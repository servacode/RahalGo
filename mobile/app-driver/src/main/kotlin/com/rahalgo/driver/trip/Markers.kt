package com.rahalgo.driver.trip

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import androidx.core.content.ContextCompat
import com.rahalgo.driver.R
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.Style
import org.maplibre.android.style.layers.CircleLayer
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.layers.SymbolLayer
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point

/**
 * ══════════════════════════════════════════════════════════════════════
 * **علاماتُ الرحلة — أنت والمتجر والزبون**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا صورةٌ لا دائرةٌ ملوّنة
 *
 * (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة: درّاجةٌ في هالةٍ ومتجرٌ في علامة.)
 *
 * **وثلاثُ دوائرَ متشابهةٍ تختلف ألوانها** تُقرأ بالتذكّر: أيُّ لونٍ
 * كان المتجر؟ **والشكلُ يُقرأ بلا تذكّر** — درّاجةٌ هي أنت، ومتجرٌ هو
 * المتجر.
 *
 * # وهالةٌ تحت الدرّاجة
 *
 * **موضعُك يجب أن يُلمح لا يُبحث عنه**: الخريطةُ تتحرّك تحتك،
 * **وعلامةٌ بحجم العلامة الأخرى** تضيع بجانبها.
 *
 * # ولا اسمَ مكتوبا على الخريطة
 *
 * **الكتابةُ على خريطة MapLibre تحتاج خادمَ حروف** (`glyphs`) — ولا
 * خادمَ لنا، **والاسمُ مكتوبٌ في لوح الطور وفي البطاقة** فلا يضيع.
 *
 * # والخطّ مسار شوارع لا مستقيم
 *
 * **كان مستقيما يمرّ فوق البيوت** — ومحرّك المسارات (`internal/routing`)
 * صار يعطي خطّ الشوارع ومسافته ومدّته. **والمستقيم يبقى احتياطا** إن
 * لم يردّ المحرّك: خطّ تقريبيّ خير من لا خطّ.
 *
 * **والملاحة الحقيقيّة تبقى لتطبيق الخرائط** بضغطة زرّ — الصوت
 * والانعطافات ليست عندنا.
 */
object Markers {

    private const val SRC_POINTS = "trip-points"
    private const val SRC_LINE = "trip-line"
    private const val SRC_ME = "trip-me"

    private const val DRIVER = "#1E88E5"
    private const val PICKUP = "#FE9501"
    private const val DROPOFF = "#02678F"

    private const val IMG_DRIVER = "img-driver"
    private const val IMG_PICKUP = "img-pickup"
    private const val IMG_DROPOFF = "img-dropoff"

    /**
     * ══════════════════════════════════════════════════════════════════
     * **تُبنى مرّةً ثمّ تُحدَّث — ولا تُبنى مرّتين**
     * ══════════════════════════════════════════════════════════════════
     *
     * **و`addSource` بمعرّفٍ موجودٍ تُسقط التطبيق** — لا تردّ خطأً
     * يُعالَج: `CannotAddSourceException`.
     *
     * **وكانت لا تقع لأنّ الأسلوب كان يُعاد بناؤه مع كلّ رسم**: أسلوبٌ
     * جديدٌ لا مصادرَ فيه، **فالإضافة أوّلُ إضافة دائما.** فلمّا صار
     * الأسلوب يُحمَّل مرّةً (وهو الصواب) **صار الرسمُ الثاني يسقط** —
     * وأوّلُ ما يُوقعه ضغطةُ زرٍّ على الخريطة. (أمسكه المالك
     * ٢٠٢٦-٠٨-١٢: «لا يعملان ويغلقان التطبيق عند الضغط عليهما».)
     *
     * **فتُقرأ المصادر أوّلا**: موجودةٌ تُبدَّل حمولتُها، **وغيرُ
     * موجودةٍ تُنشأ بطبقتها.**
     */
    fun draw(
        context: Context,
        style: Style,
        driver: LatLng?,
        pickup: LatLng?,
        dropoff: LatLng?,
        route: List<LatLng> = emptyList(),
    ) {
        images(context, style)
        // **والخطّ أوّلا ثمّ العلامات** — الترتيب هو ترتيب الرسم:
        // **من رسم الخطّ فوقها** شطبها بخطّ يمرّ في وسطها.
        //
        // ══════════════════════════════════════════════════════════════
        // **والخطُّ مسارُ شوارعَ إن وُجد**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «لا تحترم الشوارع، ترسم خطّاً مستقيما
        //  من فوق البيوت وكأنّ المكان فارغ».)
        //
        // **والمستقيمُ يبقى احتياطا**: محرّكُ المسارات قد ينام، **وخريطةٌ
        // بلا خطٍّ لا تقول إلى أين** — وخطٌّ تقريبيٌّ خيرٌ من لا خطّ.
        line(style, route.ifEmpty { listOfNotNull(driver, pickup, dropoff) })
        me(style, driver)
        points(style, driver, pickup, dropoff)
    }

    /** **الصورُ تُسجَّل مرّةً في الأسلوب** — ثمّ تُنادى بأسمائها. */
    private fun images(context: Context, style: Style) {
        if (style.getImage(IMG_DRIVER) != null) return
        style.addImage(IMG_DRIVER, badge(context, R.drawable.ic_moto, DRIVER, 46))
        style.addImage(IMG_PICKUP, badge(context, R.drawable.ic_store, PICKUP, 40))
        style.addImage(IMG_DROPOFF, badge(context, R.drawable.ic_pin, DROPOFF, 40))
    }

    /**
     * **قرصٌ ملوّنٌ فيه أيقونةٌ بيضاءُ وحافّةٌ بيضاء.**
     *
     * **والحافّةُ ليست زينة**: العلامةُ تقع على بلاطةٍ قد تكون بلونها،
     * **وحدٌّ أبيضُ يفصلها عن كلّ أرض.**
     */
    private fun badge(context: Context, res: Int, color: String, sizeDp: Int): Bitmap {
        val px = (sizeDp * context.resources.displayMetrics.density).toInt()
        val bitmap = Bitmap.createBitmap(px, px, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        val paint = Paint(Paint.ANTI_ALIAS_FLAG)
        val radius = px / 2f

        paint.color = android.graphics.Color.WHITE
        canvas.drawCircle(radius, radius, radius, paint)
        paint.color = android.graphics.Color.parseColor(color)
        canvas.drawCircle(radius, radius, radius - px * 0.08f, paint)

        val icon = ContextCompat.getDrawable(context, res) ?: return bitmap
        val pad = (px * 0.26f).toInt()
        icon.setTint(android.graphics.Color.WHITE)
        icon.setBounds(pad, pad, px - pad, px - pad)
        icon.draw(canvas)
        return bitmap
    }

    /**
     * **هالةُ موضعك** — دائرةٌ شفّافةٌ تحت الدرّاجة.
     *
     * **وهي طبقةٌ وحدَها لتقع تحت العلامات كلِّها** — ولو رُسمت معها
     * لغطّت ما جاورها.
     */
    private fun me(style: Style, driver: LatLng?) {
        val collection = FeatureCollection.fromFeatures(
            listOfNotNull(driver?.let { feature(it, IMG_DRIVER) }),
        )
        val existing = style.getSourceAs<GeoJsonSource>(SRC_ME)
        if (existing != null) {
            existing.setGeoJson(collection)
            return
        }
        style.addSource(GeoJsonSource(SRC_ME, collection))
        style.addLayer(
            CircleLayer("trip-me-layer", SRC_ME).withProperties(
                PropertyFactory.circleRadius(26f),
                PropertyFactory.circleColor(DRIVER),
                PropertyFactory.circleOpacity(0.22f),
            ),
        )
    }

    private fun points(style: Style, driver: LatLng?, pickup: LatLng?, dropoff: LatLng?) {
        val features = buildList {
            pickup?.let { add(feature(it, IMG_PICKUP)) }
            dropoff?.let { add(feature(it, IMG_DROPOFF)) }
            // **وأنت آخرُ ما يُرسم** — فلا تُغطّى بعلامةٍ فوقك.
            driver?.let { add(feature(it, IMG_DRIVER)) }
        }
        val collection = FeatureCollection.fromFeatures(features)
        val existing = style.getSourceAs<GeoJsonSource>(SRC_POINTS)
        if (existing != null) {
            existing.setGeoJson(collection)
            return
        }
        style.addSource(GeoJsonSource(SRC_POINTS, collection))
        style.addLayer(
            SymbolLayer("trip-points-layer", SRC_POINTS).withProperties(
                // **والصورةُ تُقرأ من خاصّيّة النقطة** — لا ثلاثُ طبقاتٍ
                // متشابهة.
                PropertyFactory.iconImage("{kind}"),
                // **ولا تُخفى إن تزاحمت**: العلامتان تتلاصقان حين يقترب
                // من المتجر، **وعلامةٌ تختفي لأنّها ضاقت** تُقرأ عطبا.
                PropertyFactory.iconAllowOverlap(true),
                PropertyFactory.iconIgnorePlacement(true),
            ),
        )
    }

    private fun line(style: Style, points: List<LatLng>) {
        // **وخطٌّ من نقطةٍ واحدةٍ لا يُرسم** — وإن كان مرسوما يُفرَّغ:
        // **خطٌّ إلى وجهةٍ لم تعد تُعرض** يبقى ممدودا إلى لا شيء.
        val coords = points.map { Point.fromLngLat(it.longitude, it.latitude) }
        val geometry = if (coords.size < 2) {
            FeatureCollection.fromFeatures(emptyList())
        } else {
            FeatureCollection.fromFeature(
                Feature.fromGeometry(LineString.fromLngLats(coords)),
            )
        }
        val existing = style.getSourceAs<GeoJsonSource>(SRC_LINE)
        if (existing != null) {
            existing.setGeoJson(geometry)
            return
        }
        if (coords.size < 2) return
        style.addSource(GeoJsonSource(SRC_LINE, geometry))
        // **وطبقتان: هالةٌ عريضةٌ وقلبٌ ضيّق** — خطٌّ رفيعٌ وحدَه يضيع
        // في الشوارع، **وعريضٌ صلبٌ يطمس ما تحته.**
        style.addLayer(
            LineLayer("trip-line-glow", SRC_LINE).withProperties(
                PropertyFactory.lineColor(DRIVER),
                PropertyFactory.lineWidth(14f),
                PropertyFactory.lineOpacity(0.20f),
                PropertyFactory.lineCap("round"),
                PropertyFactory.lineJoin("round"),
            ),
        )
        style.addLayer(
            LineLayer("trip-line-layer", SRC_LINE).withProperties(
                PropertyFactory.lineColor(DRIVER),
                PropertyFactory.lineWidth(5f),
                PropertyFactory.lineOpacity(0.95f),
                PropertyFactory.lineCap("round"),
                PropertyFactory.lineJoin("round"),
            ),
        )
    }

    private fun feature(at: LatLng, kind: String): Feature =
        Feature.fromGeometry(Point.fromLngLat(at.longitude, at.latitude)).also {
            it.addStringProperty("kind", kind)
        }
}
