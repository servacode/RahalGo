package com.rahalgo.navigation

import android.graphics.PointF
import android.graphics.RectF
import org.maplibre.android.maps.MapLibreMap
import org.maplibre.android.maps.Style
import org.maplibre.android.style.expressions.Expression
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.Property
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point as GeoJsonPoint

/**
 * ══════════════════════════════════════════════════════════════════
 * **خطوطُ البدائل — تحتَ الموصى به، وتُلمَس بمعرِّفها**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٧، البند ٢٦؛ وإغلاقُ واجهة ٧، البنود ١٤ و٢٤ إلى ٢٨.)
 *
 * **الترتيبُ المطلوب**:
 *
 *	ZONES  →  ALTERNATIVES  →  PRIMARY ROUTE  →  MARKERS
 *
 * **والبدائلُ تحتَ الموصى به** فلا تخفيه، **وكلُّها تحتَ
 * `rahalgo-route-anchor`** فلا تخفي أسماءَ الشوارع.
 *
 * # **والمعرِّفُ في الخاصّيّة لا في الترتيب** (البند ٢٨)
 *
 * **أمرُ المالك نصّاً**: «ولا تربط الضغط إلى array index أو layer
 * position. routeId فقط».
 *
 * **فترتيبُ القائمة يتبدّل بين جلبتين** — والسائقُ يضغط ما ظنّه
 * الأوّل. **فكلُّ خطٍّ يحمل `route_id` في خصائصه**، ومنه يُعرف.
 *
 * # **والخطُّ ليس هدفَ لمس** (البند ٢٥)
 *
 * **خمسةُ بكسلاتٍ لا تُلمَس بإصبعٍ على مقود.** **فيُسأل المحرّكُ عن
 * مستطيلٍ حول اللمسة** لا عن نقطة — ٤٨dp كما يوصي أندرويد.
 */
object AltRouteLayer {

    const val SOURCE = "rahalgo-alt-routes"
    const val LAYER = "rahalgo-alt-routes-layer"

    /** **مِرساةُ المسار من ٦أ** — والبدائلُ تحتَها. */
    const val ROUTE_ANCHOR = "rahalgo-route-anchor"

    /** **أولويّةُ السجلّ** — بين المناطق والمسار. */
    const val PRIORITY = 150

    /** **خاصّيّةُ الهويّة** — البند ٢٨. */
    const val PROP_ROUTE_ID = "route_id"

    /** **وهل هو المعايَن؟** — فيُرسم بلونه وعرضه. */
    const val PROP_PREVIEW = "preview"

    /**
     * **منطقةُ اللمس** — البند ٢٥.
     *
     * **٤٨dp هي أدنى هدفِ لمسٍ يوصي به أندرويد**، والنصفُ في كلّ
     * اتّجاهٍ حول نقطة الضغط.
     */
    const val TOUCH_TARGET_DP = 48f

    /** **بديلٌ يُرسم** — هندسةٌ ومعرِّف. */
    data class Drawable(val routeId: String, val route: NavRoute)

    /**
     * **يرسم البدائلَ أو يمحوها.**
     *
     * **وقائمةٌ فارغةٌ تمحو** — لا بدائلَ ليست إخفاقاً، **والطبقةُ
     * تختفي بلا أثر** (البند ٢١).
     *
     * @param previewRouteId **المعايَنُ الآن** — يُبرَز ولا يُعتمد.
     */
    fun draw(style: Style, alternatives: List<Drawable>, previewRouteId: String? = null) {
        val features = alternatives
            .filter { it.route.geometry.size >= 2 }
            .map { alt ->
                Feature.fromGeometry(
                    LineString.fromLngLats(
                        alt.route.geometry.map { GeoJsonPoint.fromLngLat(it.lng, it.lat) },
                    ),
                ).apply {
                    addStringProperty(PROP_ROUTE_ID, alt.routeId)
                    addBooleanProperty(PROP_PREVIEW, alt.routeId == previewRouteId)
                }
            }
        val collection = FeatureCollection.fromFeatures(features)

        val existing = style.getSourceAs<GeoJsonSource>(SOURCE)
        if (existing != null) {
            // **والتحديثُ لا الإضافة** — `addSource` بمعرِّفٍ موجودٍ
            // **تُسقط التطبيق** ولا تردّ خطأً (كما في `TripMarkers`).
            existing.setGeoJson(collection)
            return
        }

        style.addSource(GeoJsonSource(SOURCE, collection))

        /**
         * **واللونُ والعرضُ من الخاصّيّة** — لا طبقتان.
         *
         * **فطبقةٌ للمعايَنِ وأخرى للباقي** تعني ترتيباً يُدار بيد،
         * **وتعني مصدرين يتفرّقان.** والتعبيرُ يفصلهما في طبقةٍ
         * واحدة.
         */
        val isPreview = Expression.get(PROP_PREVIEW)
        val layer = LineLayer(LAYER, SOURCE).withProperties(
            PropertyFactory.lineColor(
                Expression.switchCase(
                    isPreview, Expression.color(android.graphics.Color.parseColor(MapRoutePalette.PREVIEW)),
                    Expression.color(android.graphics.Color.parseColor(MapRoutePalette.ALTERNATIVE)),
                ),
            ),
            PropertyFactory.lineWidth(
                Expression.switchCase(
                    isPreview, Expression.literal(MapRoutePalette.PREVIEW_WIDTH),
                    Expression.literal(MapRoutePalette.ALTERNATIVE_WIDTH),
                ),
            ),
            PropertyFactory.lineOpacity(
                Expression.switchCase(
                    isPreview, Expression.literal(MapRoutePalette.PREVIEW_OPACITY),
                    Expression.literal(MapRoutePalette.ALTERNATIVE_OPACITY),
                ),
            ),
            PropertyFactory.lineCap(Property.LINE_CAP_ROUND),
            PropertyFactory.lineJoin(Property.LINE_JOIN_ROUND),
        )

        /**
         * **وتحتَ المِرساة** — فلا تخفي أسماءَ الشوارع.
         *
         * **وإن غابت المِرساةُ فالنمطُ ليس نمطَنا** — تُضاف في أعلى
         * الترتيب ويُترك الأمرُ للفاحص. **ولا اختيارَ عشوائيٌّ لطبقةٍ
         * أخرى.**
         */
        if (style.getLayer(ROUTE_ANCHOR) != null) {
            style.addLayerBelow(layer, ROUTE_ANCHOR)
        } else {
            style.addLayer(layer)
        }
    }

    /** **يمحو الطبقةَ ومصدرَها** — عند إخفاء الخيارات. */
    fun clear(style: Style) {
        style.getLayer(LAYER)?.let { style.removeLayer(LAYER) }
        style.getSourceAs<GeoJsonSource>(SOURCE)?.let { style.removeSource(SOURCE) }
    }

    // ══════════════════════════════════════════════════════════════
    // **اللمس** — البنود ٢٤ إلى ٢٨
    // ══════════════════════════════════════════════════════════════

    /**
     * **معرِّفُ البديل تحتَ اللمسة** — أو `null`.
     *
     * **ولا يستدعي `setRoute`** (البند ٤٦) — يردّ معرِّفاً، **والقرارُ
     * في آلة الحال.**
     *
     * **والمستطيلُ مصفاةٌ أوّليّةٌ لا حكم** (البند ١٦): يُسأل المحرّكُ
     * عمّا يقع فيه، **ثمّ تُقاس المسافةُ الحقيقيّةُ لكلّ مرشَّح.**
     */
    fun routeIdAt(map: MapLibreMap, screenX: Float, screenY: Float, density: Float): String? {
        val halfPx = TOUCH_TARGET_DP * density / 2f
        val rect = RectF(screenX - halfPx, screenY - halfPx, screenX + halfPx, screenY + halfPx)
        val hits = map.queryRenderedFeatures(rect, LAYER)

        val candidates = hits.mapNotNull { feature ->
            val id = feature.getStringProperty(PROP_ROUTE_ID) ?: return@mapNotNull null
            val line = feature.geometry() as? LineString ?: return@mapNotNull null
            val screen = line.coordinates().map { c ->
                val pt = map.projection.toScreenLocation(
                    org.maplibre.android.geometry.LatLng(c.latitude(), c.longitude()),
                )
                TapGeometry.ScreenPoint(pt.x, pt.y)
            }
            id to screen
        }
        return TapGeometry.pick(candidates, screenX, screenY, density)
    }

    /** **ومركزُ اللمسة** — يُبنى مرّةً ويُمرَّر. */
    fun touchPoint(x: Float, y: Float): PointF = PointF(x, y)
}
