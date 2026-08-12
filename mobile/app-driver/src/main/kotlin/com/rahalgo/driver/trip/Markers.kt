package com.rahalgo.driver.trip

import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.Style
import org.maplibre.android.style.layers.CircleLayer
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نقاط الرحلة الثلاث وخطُّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثلاثة ألوان تُقرأ بلا كلام**: أنت · المتجر · الزبون.
 *
 * # ولماذا خطّ مستقيم لا مسار شوارع
 *
 * **المسار الحقيقيّ يحتاج خادم توجيه** (`routing`) — وهو خدمة ثانية
 * تُنصب وتُصان. **والخطّ المستقيم يقول الجهة والبعد**، وهما ما يقرّر
 * بهما السائق. **والملاحة الحقيقيّة تُسلَّم لتطبيق الخرائط** بضغطة زرّ.
 */
object Markers {

    private const val SRC_POINTS = "trip-points"
    private const val SRC_LINE = "trip-line"

    private const val DRIVER = "#1E88E5"
    private const val PICKUP = "#02678F"
    private const val DROPOFF = "#FE9501"

    fun draw(style: Style, driver: LatLng?, pickup: LatLng?, dropoff: LatLng?) {
        // **والخطّ أوّلا ثمّ النقاط** — الترتيب هو ترتيب الرسم:
        // **من رسم الخطّ فوق النقاط** شطبها بخطّ يمرّ في وسطها.
        line(style, listOfNotNull(driver, pickup, dropoff))
        points(style, driver, pickup, dropoff)
    }

    private fun points(style: Style, driver: LatLng?, pickup: LatLng?, dropoff: LatLng?) {
        val features = buildList {
            driver?.let { add(feature(it, "driver")) }
            pickup?.let { add(feature(it, "pickup")) }
            dropoff?.let { add(feature(it, "dropoff")) }
        }
        style.addSource(GeoJsonSource(SRC_POINTS, FeatureCollection.fromFeatures(features)))
        style.addLayer(
            CircleLayer("trip-points-layer", SRC_POINTS).withProperties(
                PropertyFactory.circleRadius(9f),
                PropertyFactory.circleStrokeWidth(3f),
                PropertyFactory.circleStrokeColor("#FFFFFF"),
                // **واللون بحسب نوع النقطة** — يُقرأ من خاصّيّتها لا من
                // ثلاث طبقات متشابهة.
                PropertyFactory.circleColor(
                    org.maplibre.android.style.expressions.Expression.match(
                        org.maplibre.android.style.expressions.Expression.get("kind"),
                        org.maplibre.android.style.expressions.Expression.color(
                            android.graphics.Color.parseColor(DRIVER),
                        ),
                        org.maplibre.android.style.expressions.Expression.stop(
                            "pickup",
                            org.maplibre.android.style.expressions.Expression.color(
                                android.graphics.Color.parseColor(PICKUP),
                            ),
                        ),
                        org.maplibre.android.style.expressions.Expression.stop(
                            "dropoff",
                            org.maplibre.android.style.expressions.Expression.color(
                                android.graphics.Color.parseColor(DROPOFF),
                            ),
                        ),
                    ),
                ),
            ),
        )
    }

    private fun line(style: Style, points: List<LatLng>) {
        if (points.size < 2) return
        val coords = points.map { Point.fromLngLat(it.longitude, it.latitude) }
        style.addSource(
            GeoJsonSource(SRC_LINE, Feature.fromGeometry(LineString.fromLngLats(coords))),
        )
        style.addLayer(
            LineLayer("trip-line-layer", SRC_LINE).withProperties(
                PropertyFactory.lineColor(PICKUP),
                PropertyFactory.lineWidth(4f),
                PropertyFactory.lineOpacity(0.55f),
            ),
        )
    }

    private fun feature(at: LatLng, kind: String): Feature =
        Feature.fromGeometry(Point.fromLngLat(at.longitude, at.latitude)).also {
            it.addStringProperty("kind", kind)
        }
}
