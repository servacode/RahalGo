package com.rahalgo.map

import android.annotation.SuppressLint
import android.content.Context
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الموقعُ الحيّ في لاقط النقطة — يتحسّن ما دامت الخريطةُ مفتوحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظةُ المالك ٢٠٢٦-١٠-١٢: «وقت عم حدد الموقع من الخريطة… ماعم يجيب الموقع بشكل
 *  دقيق» · «نخلي الجي بي اس اسرع بتحديد الموقع».)
 *
 * **قِيس على جهازه**: `getCurrentLocation` يردّ **أوّلَ** قراءةٍ ويقف — شبكةٌ ±٢٠م أو
 * GPS بثلاثة أقمارٍ ±٩٥م — **وكانت تُقبل حتّى ٥٠٠ م** (`Accuracy.CONFIRM_M`).
 *
 * **فيُسمع كلَّ ثانيةٍ بأعلى دقّة من لحظة فتح الخريطة** — كما تفعل خرائطُ غوغل:
 * أوّلُ قراءةٍ تقريبيّةٌ تضع الدبّوسَ فوراً، **وكلُّ أدقَّ منها يحلّ محلَّها.**
 * **والأقدمُ من دقيقةٍ لا يُعتمد** (موضعٌ محفوظٌ من مكانٍ آخر).
 */
object LiveFix {

    /** **دقّةُ أفضل قراءةٍ الآن** بالمتر — وفارغةٌ قبل أوّل قراءة. */
    var accuracyM by mutableStateOf<Float?>(null)
        private set

    /** **أفضلُ نقطةٍ الآن.** */
    var point by mutableStateOf<LatLng?>(null)
        private set

    var running by mutableStateOf(false)
        private set

    private var client: FusedLocationProviderClient? = null
    private var callback: LocationCallback? = null
    private var bestAtMs = 0L

    /** **قراءةٌ أقدمُ من هذا لا تُعتمد.** */
    private const val MAX_AGE_MS = 60_000L

    /** **وأفضلُ قراءةٍ تُستبدَل بالأحدث بعد هذا** — من خرج إلى الشارع تغيّر موضعُه. */
    private const val BEST_KEEP_MS = 15_000L

    @SuppressLint("MissingPermission")
    fun start(context: Context) {
        if (running || !Here.granted(context)) return
        val c = LocationServices.getFusedLocationProviderClient(context.applicationContext)
        val request = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, 1_000L)
            .setMinUpdateIntervalMillis(500L)
            .setWaitForAccurateLocation(false)
            .build()
        val cb = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                for (loc in result.locations) {
                    if (!loc.hasAccuracy()) continue
                    val ageMs = (SystemClock.elapsedRealtimeNanos() - loc.elapsedRealtimeNanos) / 1_000_000
                    if (ageMs > MAX_AGE_MS) continue
                    val now = SystemClock.elapsedRealtime()
                    val best = accuracyM
                    if (best == null || loc.accuracy <= best || now - bestAtMs > BEST_KEEP_MS) {
                        accuracyM = loc.accuracy
                        point = LatLng(loc.latitude, loc.longitude)
                        bestAtMs = now
                    }
                }
            }
        }
        runCatching {
            c.requestLocationUpdates(request, cb, Looper.getMainLooper())
            client = c
            callback = cb
            running = true
        }.onFailure { Log.w("RahalGo/livefix", "تعذّر الاستماعُ للموضع", it) }
    }

    fun stop() {
        val c = client
        val cb = callback
        if (c != null && cb != null) runCatching { c.removeLocationUpdates(cb) }
        client = null
        callback = null
        running = false
        accuracyM = null
        point = null
        bestAtMs = 0L
    }
}
