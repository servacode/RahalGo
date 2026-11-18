package com.rahalgo.map

import android.annotation.SuppressLint
import android.content.Context
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.android.gms.location.CurrentLocationRequest
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.tasks.CancellationToken
import com.google.android.gms.tasks.Task
import com.google.android.gms.tasks.TaskCompletionSource
import com.google.android.gms.tasks.Tasks

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مصدرُ الموقع الواحد — بخدمات غوغل أو بلاها** (بلاغُ المالك ٢٠٢٦-١١-١٨: «عندنا مشكلة بهواتف هاوواي»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ قراءةِ موقعٍ في التطبيقات الأربعة كانت من `FusedLocationProviderClient`** — وهو خدمةُ غوغل.
 * **وهواوي وهونر الجديدة بلا خدمات غوغل**، فالنداءُ لا يردّ أبداً: «طال الانتظار ولم يصل موقع»
 * عند الزبون، **وكابتنٌ بهواوي لا يصل موقعُه للّوحة.**
 *
 * **فصار هنا بابٌ واحدٌ بنفس أسماء غوغل** (`requestLocationUpdates` · `removeLocationUpdates` ·
 * `getCurrentLocation` · `lastLocation`):
 *
 *	بخدمات غوغل  ←  `FusedLocationProviderClient` نفسُه — **لا يتغيّر شيءٌ على هذه الجوّالات.**
 *	بلاها        ←  `LocationManager` من أندرويد نفسِه: الأقمارُ (GPS) والشبكةُ معاً.
 *
 * **ومن أراد الموقعَ ينادي `RgLocation.client(context)` لا `LocationServices`** — يحرسه
 * `RgLocationWiringTest`: لا `getFusedLocationProviderClient` خارجَ هذا الملفّ.
 */
object RgLocation {

    @Volatile
    private var gms: Boolean? = null

    /** **أفي الجهاز خدماتُ غوغل صالحة؟** — تُسأل مرّةً لكلّ تشغيل. */
    fun hasGms(context: Context): Boolean {
        gms?.let { return it }
        val ok = runCatching {
            GoogleApiAvailability.getInstance().isGooglePlayServicesAvailable(context.applicationContext) ==
                ConnectionResult.SUCCESS
        }.getOrDefault(false)
        gms = ok
        if (!ok) Log.i(TAG, "لا خدماتِ غوغل — الموقعُ من أندرويد مباشرةً (GPS والشبكة)")
        return ok
    }

    /** **للاختبار وحدَه** — يفرض المسار. */
    internal fun forceGms(value: Boolean?) {
        gms = value
    }

    fun client(context: Context): Client {
        val app = context.applicationContext
        return if (hasGms(app)) FusedClient(LocationServices.getFusedLocationProviderClient(app)) else PlatformClient(app)
    }

    /** **البابُ نفسُه للطريقين** — بأسماء غوغل وأنواعها، فلا يتغيّر المستدعي إلّا بسطر الإنشاء. */
    interface Client {
        fun requestLocationUpdates(request: LocationRequest, callback: LocationCallback, looper: Looper): Task<Void>
        fun removeLocationUpdates(callback: LocationCallback): Task<Void>
        fun getCurrentLocation(priority: Int, token: CancellationToken?): Task<Location?>
        fun getCurrentLocation(request: CurrentLocationRequest, token: CancellationToken?): Task<Location?>
        val lastLocation: Task<Location?>
    }

    @SuppressLint("MissingPermission")
    private class FusedClient(private val c: FusedLocationProviderClient) : Client {
        override fun requestLocationUpdates(request: LocationRequest, callback: LocationCallback, looper: Looper) =
            c.requestLocationUpdates(request, callback, looper)

        override fun removeLocationUpdates(callback: LocationCallback) = c.removeLocationUpdates(callback)

        @Suppress("UNCHECKED_CAST")
        override fun getCurrentLocation(priority: Int, token: CancellationToken?) =
            c.getCurrentLocation(priority, token) as Task<Location?>

        @Suppress("UNCHECKED_CAST")
        override fun getCurrentLocation(request: CurrentLocationRequest, token: CancellationToken?) =
            c.getCurrentLocation(request, token) as Task<Location?>

        @Suppress("UNCHECKED_CAST")
        override val lastLocation: Task<Location?>
            get() = c.lastLocation as Task<Location?>
    }

    /**
     * **أندرويد بلا غوغل** — مستمعٌ على الأقمار وآخرُ على الشبكة لكلّ نداء، **وكلُّ قراءةٍ تصل
     * المستدعيَ كما تصله من غوغل** (`LocationResult`). **والشبكةُ قد لا تكون** (جوّالٌ بلا مزوّد) —
     * فالأقمارُ وحدَها، وفي مكانٍ مكشوف تكفي.
     */
    @SuppressLint("MissingPermission")
    private class PlatformClient(private val context: Context) : Client {
        private val lm: LocationManager? = context.getSystemService(LocationManager::class.java)
        private val listeners = HashMap<LocationCallback, List<LocationListener>>()

        private fun providers(): List<String> {
            val m = lm ?: return emptyList()
            val all = runCatching { m.allProviders }.getOrDefault(emptyList())
            return listOf(LocationManager.GPS_PROVIDER, LocationManager.NETWORK_PROVIDER).filter { it in all }
        }

        override fun requestLocationUpdates(request: LocationRequest, callback: LocationCallback, looper: Looper): Task<Void> {
            removeNow(callback)
            val m = lm ?: return Tasks.forException(IllegalStateException("لا LocationManager"))
            val every = request.minUpdateIntervalMillis.coerceAtLeast(500L)
            val made = ArrayList<LocationListener>()
            for (p in providers()) {
                val l = object : LocationListener {
                    override fun onLocationChanged(location: Location) {
                        callback.onLocationResult(LocationResult.create(listOf(location)))
                    }

                    override fun onProviderEnabled(provider: String) {}
                    override fun onProviderDisabled(provider: String) {}

                    @Deprecated("أندرويد القديم")
                    override fun onStatusChanged(provider: String?, status: Int, extras: Bundle?) {}
                }
                try {
                    m.requestLocationUpdates(p, every, 0f, l, looper)
                    made += l
                } catch (e: SecurityException) {
                    made.forEach { runCatching { m.removeUpdates(it) } }
                    return Tasks.forException(e)
                } catch (e: Exception) {
                    Log.w(TAG, "تعذّر الاستماعُ لـ$p", e)
                }
            }
            if (made.isEmpty()) return Tasks.forException(IllegalStateException("لا مزوّدَ موقع"))
            synchronized(listeners) { listeners[callback] = made }
            return Tasks.forResult(null)
        }

        private fun removeNow(callback: LocationCallback) {
            val old = synchronized(listeners) { listeners.remove(callback) } ?: return
            old.forEach { runCatching { lm?.removeUpdates(it) } }
        }

        override fun removeLocationUpdates(callback: LocationCallback): Task<Void> {
            removeNow(callback)
            return Tasks.forResult(null)
        }

        override fun getCurrentLocation(priority: Int, token: CancellationToken?): Task<Location?> =
            single(token, FRESH_MAX_AGE_MS)

        override fun getCurrentLocation(request: CurrentLocationRequest, token: CancellationToken?): Task<Location?> =
            single(token, request.maxUpdateAgeMillis.coerceAtMost(FRESH_MAX_AGE_MS))

        /**
         * **قراءةٌ واحدة**: آخرُ ما يعرفه الجهاز إن كان طازجاً، **وإلّا أوّلُ قراءةٍ حيّةٍ من أيّ
         * مزوّد**، **وبعد المهلة `null`** — كما تردّ غوغل حين لا موقع.
         */
        private fun single(token: CancellationToken?, maxAgeMs: Long): Task<Location?> {
            val m = lm ?: return Tasks.forResult(null)
            bestLast()?.let { if (ageMs(it) <= maxAgeMs) return Tasks.forResult(it) }
            val src = TaskCompletionSource<Location?>()
            val made = ArrayList<LocationListener>()
            val main = Handler(Looper.getMainLooper())
            fun done(loc: Location?) {
                if (src.trySetResult(loc)) made.forEach { runCatching { m.removeUpdates(it) } }
            }
            for (p in providers()) {
                val l = object : LocationListener {
                    override fun onLocationChanged(location: Location) = done(location)
                    override fun onProviderEnabled(provider: String) {}
                    override fun onProviderDisabled(provider: String) {}

                    @Deprecated("أندرويد القديم")
                    override fun onStatusChanged(provider: String?, status: Int, extras: Bundle?) {}
                }
                try {
                    m.requestLocationUpdates(p, 0L, 0f, l, Looper.getMainLooper())
                    made += l
                } catch (e: SecurityException) {
                    made.forEach { runCatching { m.removeUpdates(it) } }
                    return Tasks.forException(e)
                } catch (e: Exception) {
                    Log.w(TAG, "تعذّرت قراءةُ $p", e)
                }
            }
            if (made.isEmpty()) return Tasks.forResult(null)
            token?.onCanceledRequested { done(null) }
            main.postDelayed({ done(null) }, SINGLE_TIMEOUT_MS)
            return src.task
        }

        private fun bestLast(): Location? {
            val m = lm ?: return null
            return (providers() + LocationManager.PASSIVE_PROVIDER)
                .mapNotNull { runCatching { m.getLastKnownLocation(it) }.getOrNull() }
                .minByOrNull { ageMs(it) }
        }

        override val lastLocation: Task<Location?>
            get() = Tasks.forResult(bestLast())
    }

    private fun ageMs(l: Location): Long = (SystemClock.elapsedRealtimeNanos() - l.elapsedRealtimeNanos) / 1_000_000

    /** **قراءةٌ أحدثُ من هذا تُردّ فوراً** بلا انتظار قمر. */
    private const val FRESH_MAX_AGE_MS = 10_000L

    /** **مهلةُ القراءة الواحدة** — كمهلة غوغل تقريباً؛ وبعدها `null`. */
    private const val SINGLE_TIMEOUT_MS = 30_000L

    private const val TAG = "RahalGo/location"
}
