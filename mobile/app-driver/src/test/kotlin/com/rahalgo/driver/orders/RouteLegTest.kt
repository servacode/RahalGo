package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * **موقعُ المستلِم وصل والسائقُ في الطريق إليه** (طلبُ المالك ٢٠٢٦-١١-٠٦: «ياخد السائق ع
 * موقعه بمرحلة الذهاب إلى الزبون»).
 *
 * **ساقُ الطريق تتبدّل بتبدّل النقطة** — وإلّا رأى `routeKeep` الساقَ نفسَها فأبقى الطريقَ
 * المحفوظَ إلى النقطة القديمة: الدبّوسُ في المكان الجديد والخطُّ والإرشادُ إلى القديم.
 */
class RouteLegTest {

    private fun onTheWay(lat: Double, lng: Double, known: Boolean = true) =
        DriverOrder(id = "o1", status = "on_the_way", lat = lat, lng = lng, dropoffKnown = known)

    @Test
    fun pointReplacedAfterPickupIsANewLeg() {
        assertNotEquals(
            routeLegOf(onTheWay(35.95, 39.01), null),
            routeLegOf(onTheWay(35.952, 39.011), null),
        )
    }

    @Test
    fun pointArrivedAfterPickupIsANewLeg() {
        // **بلا نقطة**: المكتوبُ موقعُ المتجر — ثمّ وصل الموقع.
        assertNotEquals(
            routeLegOf(onTheWay(35.94, 39.00, known = false), null),
            routeLegOf(onTheWay(35.95, 39.01), null),
        )
    }

    @Test
    fun samePointSameLeg() {
        // **ولا طلبَ طريقٍ جديدٍ كلَّ تحديث** — النقطةُ نفسُها ساقٌ واحدة.
        assertEquals(
            routeLegOf(onTheWay(35.95, 39.01), null),
            routeLegOf(onTheWay(35.95, 39.01).copy(status = "at_dropoff"), null),
        )
    }

    @Test
    fun beforePickupThePointDoesNotMatter() {
        // **قبل الاستلام الطريقُ إلى المتجر** — نقطةُ المستلِم لا تبدّله.
        val a = DriverOrder(id = "o1", status = "assigned", lat = 35.94, lng = 39.0, dropoffKnown = false)
        assertEquals(routeLegOf(a, null), routeLegOf(a.copy(lat = 35.95, dropoffKnown = true), null))
    }
}
