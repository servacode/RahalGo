package com.rahalgo.customer

import android.Manifest
import android.location.Location
import android.os.Looper
import androidx.test.core.app.ActivityScenario
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.rule.GrantPermissionRule
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.Tasks
import com.rahalgo.map.RgLocation
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Rule
import org.junit.Test
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/**
 * **على جهازٍ بلا خدمات غوغل** (بلاغُ المالك ٢٠٢٦-١١-١٨: هواوي) — يُشغَّل والمحاكي قد عُطّلت فيه
 * `com.google.android.gms`، **وموقعُه مضبوطٌ بـ`adb emu geo fix`** على الرقّة (٣٩٫٠١٠٠، ٣٥٫٩٥٠٠).
 * البابُ (`RgLocation`) يجب أن يردّ الموقعَ من أندرويد نفسِه: قراءةً واحدةً، ثمّ قراءاتٍ متتابعة.
 */
class NoGmsLocationTest {
    @get:Rule
    val perms: GrantPermissionRule = GrantPermissionRule.grant(
        Manifest.permission.ACCESS_FINE_LOCATION,
        Manifest.permission.ACCESS_COARSE_LOCATION,
    )

    private val ctx = InstrumentationRegistry.getInstrumentation().targetContext

    /**
     * **والتطبيقُ ظاهرٌ على الشاشة** — أندرويد لا يعطي الموقعَ لتطبيقٍ في الخلفيّة (كما في الاستعمال:
     * شاشةُ العنوان مفتوحة، والمشاركةُ خدمةٌ ظاهرة).
     */
    @Test
    fun gateReadsLocationWithoutGms() {
        ActivityScenario.launch(MainActivity::class.java).use { gateReads() }
    }

    private fun gateReads() {
        if (RgLocation.hasGms(ctx)) return
        assertFalse("خدماتُ غوغل ما زالت شغّالة على المحاكي", RgLocation.hasGms(ctx))
        val client = RgLocation.client(ctx)

        val one: Location? = Tasks.await(client.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null), 40, TimeUnit.SECONDS)
        assertNotNull("ما رجعت قراءةٌ واحدة", one)
        assertEquals(35.95, one!!.latitude, 0.001)
        assertEquals(39.01, one.longitude, 0.001)

        val got = CountDownLatch(2)
        val cb = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                if (result.lastLocation != null) got.countDown()
            }
        }
        val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, 1_000L).setMinUpdateIntervalMillis(500L).build()
        InstrumentationRegistry.getInstrumentation().runOnMainSync {
            client.requestLocationUpdates(req, cb, Looper.getMainLooper())
        }
        val ok = got.await(40, TimeUnit.SECONDS)
        InstrumentationRegistry.getInstrumentation().runOnMainSync { client.removeLocationUpdates(cb) }
        assert(ok) { "ما وصلت قراءاتٌ متتابعة" }
    }

    /** **والطريقُ القديمُ نفسُه على هذا الجهاز** — نداءُ غوغل مباشرةً: لا يردّ موقعاً. هذا عطبُ الزبون. */
    @Test
    fun oldFusedPathGetsNothingWithoutGms() {
        ActivityScenario.launch(MainActivity::class.java).use {
            val old = com.google.android.gms.location.LocationServices.getFusedLocationProviderClient(ctx)
            val got = runCatching {
                Tasks.await(old.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null), 25, TimeUnit.SECONDS)
            }
            android.util.Log.i("RahalGo/location", "الطريقُ القديم: " + got)
            assert(got.getOrNull() == null) { "الطريقُ القديمُ ردّ موقعاً بلا غوغل" }
        }
    }

    /** **وعلى جهازٍ فيه غوغل** (يُشغَّل بعد إعادة تفعيلها) — البابُ يمرّ بغوغل نفسِه ويردّ الموقع. */
    @Test
    fun withGmsGateUsesFused() {
        if (!RgLocation.hasGms(ctx)) return
        ActivityScenario.launch(MainActivity::class.java).use {
            val one = Tasks.await(RgLocation.client(ctx).getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null), 40, TimeUnit.SECONDS)
            assertNotNull("ما رجعت قراءةٌ بغوغل", one)
            assertEquals(35.95, one!!.latitude, 0.001)
        }
    }
}
