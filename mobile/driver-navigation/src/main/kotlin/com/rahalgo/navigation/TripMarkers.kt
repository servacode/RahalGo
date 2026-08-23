package com.rahalgo.navigation

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import androidx.core.content.ContextCompat
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

    /** **خاصّيّةُ دوران الأيقونة** — تُكتب في النقطة وتُقرأ في الطبقة. */
    private const val PROP_ROTATE = "rotate"

    private val DRIVER = MapRoutePalette.DRIVER_PIN
    private val PICKUP = MapRoutePalette.PICKUP_PIN
    private val DROPOFF = MapRoutePalette.DROPOFF_PIN

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
        // ══════════════════════════════════════════════════════════════
        // **وأيقوناتُ التطبيق تُمرَّر لا تُستورَد**
        // ══════════════════════════════════════════════════════════════
        //
        // (المرحلة ٠: قِيس أنّ `ic_moto` و`ic_pin` في `app-driver`
        //  **تختلفان عن نسختيهما في `:ui`** — فالاستيرادُ من `:ui`
        //  كان سيغيّر المظهر، **والنقلُ يجب ألّا يغيّر شيئا.**)
        icons: MarkerIcons,
        // ══════════════════════════════════════════════════════════════
        // **ودورانُ الدرّاجة — المرحلة ١**
        // ══════════════════════════════════════════════════════════════
        //
        // **وفارغٌ يعني «لا تُدِرها»**: أوّلُ رحلةٍ قبل أن يُعرف
        // اتّجاه، **وأيقونةٌ تُصفَّر إلى الشمال تكذب على صاحبها.**
        //
        // **ويُمرَّر ولا يُحسب هنا** — الحسابُ في `BearingTracker`،
        // **وهذه ترسم ولا تقرّر.**
        driverBearingDeg: Float? = null,
    ) {
        images(context, style, icons)
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
        points(style, driver, pickup, dropoff, driverBearingDeg)
    }

    /** **الصورُ تُسجَّل مرّةً في الأسلوب** — ثمّ تُنادى بأسمائها. */
    private fun images(context: Context, style: Style, icons: MarkerIcons) {
        if (style.getImage(IMG_DRIVER) != null) return
        // **وعلامةُ السائق سهمٌ لا قرصٌ فيه درّاجة** — (طلبُ المالك
        // ٢٠٢٦-٠٨-٢٣: «بدل أيقونة الدرّاجة نضع مثل سهم قوقل ماب ليكون
        // واضح… شكل الدراجة بشع لأنه مسطّح»).
        //
        // **والسببُ هندسيٌّ لا ذوقيّ**: القرصُ لا يقول اتّجاهاً —
        // **يدور فلا يُرى دورانُه.** والسهمُ شكلُه هو معناه: **رأسُه
        // يقول إلى أين يتّجه في لمحة.**
        style.addImage(IMG_DRIVER, arrow(context, 46))
        style.addImage(IMG_PICKUP, badge(context, icons.pickup, PICKUP, 40))
        style.addImage(IMG_DROPOFF, badge(context, icons.dropoff, DROPOFF, 40))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **سهمُ الاتّجاه — علامةُ السائق**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-٢٣.)
     *
     * **ثلاثيٌّ ذو رأسٍ حادٍّ وذيلٍ مقعّر** — كسهم الملاحة في كلّ تطبيق
     * ملاحةٍ يعرفه الناس. **والشكلُ المألوفُ يُقرأ بلا تعلّم.**
     *
     * **والحافّةُ البيضاءُ ليست زينة**: السهمُ يقع على شارعٍ قد يكون
     * بلونه، **وحدٌّ أبيضُ يفصله عن كلّ أرض.**
     *
     * **ويُرسم متّجهاً إلى الأعلى** — والدورانُ تتولّاه الطبقةُ
     * (`iconRotate`) من اتّجاه السائق المنعَّم.
     */
    private fun arrow(context: Context, sizeDp: Int): Bitmap {
        val px = (sizeDp * context.resources.displayMetrics.density).toInt()
        val bitmap = Bitmap.createBitmap(px, px, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        val paint = Paint(Paint.ANTI_ALIAS_FLAG)

        val w = px.toFloat()
        val path = android.graphics.Path().apply {
            moveTo(w * 0.50f, w * 0.06f)          // الرأسُ إلى الأعلى
            lineTo(w * 0.90f, w * 0.92f)          // الجناحُ الأيمن
            lineTo(w * 0.50f, w * 0.70f)          // الذيلُ المقعّر
            lineTo(w * 0.10f, w * 0.92f)          // الجناحُ الأيسر
            close()
        }

        // **الحدُّ أوّلاً ثمّ الملء** — فيبقى الأبيضُ إطاراً حولَه.
        paint.style = Paint.Style.STROKE
        paint.strokeWidth = w * 0.10f
        paint.strokeJoin = Paint.Join.ROUND
        paint.color = android.graphics.Color.WHITE
        canvas.drawPath(path, paint)

        paint.style = Paint.Style.FILL
        paint.color = android.graphics.Color.parseColor(DRIVER)
        canvas.drawPath(path, paint)
        return bitmap
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

    private fun points(
        style: Style,
        driver: LatLng?,
        pickup: LatLng?,
        dropoff: LatLng?,
        driverBearingDeg: Float?,
    ) {
        val features = buildList {
            pickup?.let { add(feature(it, IMG_PICKUP)) }
            dropoff?.let { add(feature(it, IMG_DROPOFF)) }
            // **وأنت آخرُ ما يُرسم** — فلا تُغطّى بعلامةٍ فوقك.
            driver?.let { add(feature(it, IMG_DRIVER, driverBearingDeg)) }
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
                // ══════════════════════════════════════════════════════
                // **والدورانُ يُقرأ من خاصّيّة النقطة**
                // ══════════════════════════════════════════════════════
                //
                // **وطبقةٌ واحدةٌ تحمل الثلاثةَ** — والمتجرُ والزبونُ
                // لا يدوران، فتُكتب لهما صفراً. **وطبقةٌ ثانيةٌ
                // للدرّاجة وحدَها تعني مصدرين يفترقان.**
                //
                // **و`iconRotationAlignment` خريطةٌ لا شاشة**: مع
                // الخريطة تدور الدرّاجةُ مع الطريق، **ومع الشاشة تبقى
                // ثابتةً والخريطةُ تدور تحتها** — وهو عكسُ المراد في
                // الملاحة.
                PropertyFactory.iconRotate(
                    org.maplibre.android.style.expressions.Expression.get(PROP_ROTATE),
                ),
                PropertyFactory.iconRotationAlignment(
                    org.maplibre.android.style.layers.Property.ICON_ROTATION_ALIGNMENT_MAP,
                ),
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

    private fun feature(at: LatLng, kind: String, rotateDeg: Float? = null): Feature =
        Feature.fromGeometry(Point.fromLngLat(at.longitude, at.latitude)).also {
            it.addStringProperty("kind", kind)
            // **والصفرُ لمن لا يدور** — لا حذفُ الخاصّيّة: **تعبيرٌ
            // يقرأ خاصّيّةً غائبةً يردّ فراغاً فتختفي الأيقونةُ
            // كلُّها** في بعض نسخ المحرّك.
            it.addNumberProperty(PROP_ROTATE, rotateDeg ?: 0f)
        }
}

/**
 * **أيقوناتُ العلامات — يملكها التطبيقُ لا الوحدة.**
 *
 * **ونسخةُ `app-driver` من `ic_moto` تختلف عن نسخة `:ui`** — قِيس في
 * المرحلة ٠. **فمن استوردها من الوحدة غيّر المظهرَ وهو ينقل.**
 */
data class MarkerIcons(
    /**
     * **لم تعد تُرسَم** — علامةُ السائق صارت سهماً يُرسم بالشيفرة
     * (٢٠٢٦-٠٨-٢٣، طلبُ المالك). **وتبقى في العقد** كي لا يُكسر كلُّ
     * من يبنيها، **وتُحذف حين تُنظَّف مواضعُ البناء كلُّها.**
     */
    @Deprecated("علامةُ السائق سهمٌ يُرسم — انظر arrow()")
    @androidx.annotation.DrawableRes val driver: Int,
    @androidx.annotation.DrawableRes val pickup: Int,
    @androidx.annotation.DrawableRes val dropoff: Int,
)
