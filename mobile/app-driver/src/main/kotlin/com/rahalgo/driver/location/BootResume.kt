package com.rahalgo.driver.location

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.rahalgo.driver.MainActivity
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أُعيد تشغيلُ الهاتف والورديّةُ مفتوحة** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **قِيس**: لا مستقبِلَ لإقلاع الجهاز — **فسائقٌ أطفأ هاتفَه وأشعله وسط ورديّته
 * يبقى «على الدوام» عند الخادم وخدمةُ موقعه ميّتة** حتّى يفتح التطبيقَ بيده.
 * فيشيخ موضعُه، **ولا يعلم لماذا لا تصله طلبات.**
 *
 * # وما يُسمح به بعد الإقلاع
 *
 * **أندرويد يسمح بخدمةٍ أماميّةٍ من نوع `location` بعد الإقلاع لمن منح «الموقع
 * طوال الوقت»** — فهي ممّا يُستثنى من منع البدء من الخلفيّة (١٢+)، **ومن دون ذلك
 * الإذن لا تُمنح الخدمةُ إذنَ الموقع وهي مبدوءةٌ من الخلفيّة (١٤+).** فإن لم
 * يكن الإذنُ أو رُدّت الخدمة **يُرسَل إشعارٌ يطلب فتحَ التطبيق** — ولا تُبدأ خدمةٌ
 * بلا موقعٍ تُشغل الشريطَ بإشعارٍ كاذب.
 *
 * **ولا يُسأل الخادم**: لا جلسةَ تُجدَّد في لحظة الإقلاع ولا شبكةَ مضمونة. **فالحالُ
 * من آخرِ ما عرفه التطبيق** (`ShiftMemory`) — وأوّلُ قراءةٍ لـ`/driver/me` تُصحّحه.
 */
object BootResume {

    enum class Act { NOTHING, START, ASK }

    fun decide(signedIn: Boolean, onShift: Boolean, fineGranted: Boolean, backgroundGranted: Boolean): Act = when {
        !signedIn || !onShift -> Act.NOTHING
        fineGranted && backgroundGranted -> Act.START
        else -> Act.ASK
    }

    private const val CHANNEL = "rahalgo_shift_resume"
    private const val NOTE_ID = 1002

    /** **«ورديّتُك مفتوحة — افتح التطبيق»** — حين لا تُبدأ الخدمةُ وحدَها. */
    fun askToOpen(context: Context) {
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    CHANNEL,
                    context.getString(R.string.boot_channel),
                    NotificationManager.IMPORTANCE_HIGH,
                ),
            )
        }
        val open = PendingIntent.getActivity(
            context, 0, Intent(context, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val note = NotificationCompat.Builder(context, CHANNEL)
            .setSmallIcon(R.drawable.ic_orders)
            .setContentTitle(context.getString(R.string.boot_title))
            .setContentText(context.getString(R.string.boot_text))
            .setStyle(NotificationCompat.BigTextStyle().bigText(context.getString(R.string.boot_text)))
            .setContentIntent(open)
            .setAutoCancel(true)
            .build()
        // **وإذنُ الإشعار قد يُرفض** — فلا يسقط المستقبِل.
        runCatching { manager.notify(NOTE_ID, note) }
            .onFailure { Log.w("RahalGo/boot", "تعذّر إشعارُ فتح التطبيق", it) }
    }
}

/**
 * **آخرُ ما عرفه التطبيقُ عن ورديّته** — يُكتب مع كلّ قراءةٍ لـ`/driver/me`،
 * ويُمحى بالخروج. **ولا يُقرأ إلّا بعد الإقلاع.**
 */
object ShiftMemory {
    private const val PREFS = "driver_shift_memory"

    fun remember(context: Context, onShift: Boolean, pingSec: Long) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit()
            .putBoolean("on", onShift).putLong("ping", pingSec).apply()
    }

    fun forget(context: Context) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().clear().apply()
    }

    fun onShift(context: Context): Boolean =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean("on", false)

    fun pingSec(context: Context): Long =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getLong("ping", 20L).takeIf { it > 0 } ?: 20L
}

/** **مستقبِلُ الإقلاع** — يُعيد خدمةَ الموقع أو يطلب فتحَ التطبيق (`BootResume`). */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) return
        val app = context.applicationContext
        val signedIn = runCatching { Backend.of(app).session.refreshToken().isNotEmpty() }.getOrDefault(false)
        when (
            BootResume.decide(
                signedIn = signedIn,
                onShift = ShiftMemory.onShift(app),
                fineGranted = LocationPermission.granted(app),
                backgroundGranted = LocationPermission.backgroundGranted(app),
            )
        ) {
            BootResume.Act.NOTHING -> Unit
            BootResume.Act.START -> try {
                LocationService.start(app, ShiftMemory.pingSec(app), fromBoot = true)
            } catch (e: Exception) {
                // **`ForegroundServiceStartNotAllowedException` أو رفضُ أمن** — يُطلب الفتح.
                Log.w("RahalGo/boot", "رُدّت خدمةُ الموقع بعد الإقلاع", e)
                BootResume.askToOpen(app)
            }
            BootResume.Act.ASK -> BootResume.askToOpen(app)
        }
    }
}
