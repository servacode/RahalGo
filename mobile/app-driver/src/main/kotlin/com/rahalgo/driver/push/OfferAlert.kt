package com.rahalgo.driver.push

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.media.AudioAttributes
import android.net.Uri
import androidx.core.app.NotificationCompat
import com.rahalgo.driver.MainActivity
import com.rahalgo.driver.R
import com.rahalgo.ui.Engagement
import com.rahalgo.ui.push.RahalPushService

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تنبيهُ العرض والتطبيقُ بالخلفيّة — يرنّ حتّى يُفتح أو ينقضي العرض**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تجربةُ المالك ٢٠٢٦-١١-١٢: فتح السائقُ التطبيقَ فوجد العرضَ «0:03».)
 *
 * **والعدّادُ صادقٌ** — يُحسب من لحظة الانتهاء التي يرسلها الخادم، لا من ظهور
 * البطاقة. **والذي ضاع هو الوقتُ قبل أن ينتبه**: كان الإشعارُ رنّةً واحدةً قصيرة
 * على القناة العاجلة، فمن لم يسمعها أوّلَ مرّةٍ لم يعلم.
 *
 * **فصار للعرض قناتُه**: رنّةٌ تتكرّر (`FLAG_INSISTENT`) وإشعارٌ بملء الشاشة
 * والهاتفُ مقفل، **ويسكت وحدَه عند انقضاء العرض** (`expires_at` في الحمولة)
 * **أو حين يُفتح التطبيق** (`cancelAll` عند كلّ عودة).
 *
 * **والتطبيقُ ظاهرٌ لا يحتاجه** — البطاقةُ أمامه، فيبقى الإشعارُ العاجلُ المعتاد.
 */
object OfferAlert {

    /**
     * **قناةٌ جديدةٌ لا تعديلُ القديمة** — أندرويد لا يغيّر صوتَ قناةٍ أُنشئت.
     *
     * (ملاحظةُ المالك ٢٠٢٦-١١-١٤: «نغمةُ الطلبات صارت مثل نغمة الاتصال، مزعجة».) كانت القناةُ
     * `rahalgo_offer_ring` على **نغمة اتّصال الهاتف** — فصار للعرض **رنّةُ رحّال غو القصيرة**
     * (`res/raw/order_chime.ogg`: ثلاثُ نقراتِ جرسٍ في نحو ثانية، ثمّ سكوتٌ حتّى الثالثة)،
     * **وتكرارُها كلَّ ثلاث ثوانٍ** يأتي من `FLAG_INSISTENT` نفسِه. **والقديمةُ تُحذف** فلا تبقى في الإعدادات.
     */
    const val CHANNEL = "rahalgo_offer_chime"
    private const val OLD_CHANNEL = "rahalgo_offer_ring"
    private const val TAG = "rahalgo_offer"
    private const val FALLBACK_MS = 60_000L
    private const val MAX_MS = 300_000L

    /** **عرضٌ ينقضي؟** — نوعُ الصفّ في المحرّك `order_offer`. */
    fun isOffer(data: Map<String, String>): Boolean = data["kind"] == PushService.KIND_OFFER

    /**
     * **كم يرنّ بالملّي ثانية** — ما بقي من العرض. **وحمولةٌ قديمةٌ بلا أجلٍ** تأخذ
     * دقيقة (افتراضُ المهلة)، **ولا يزيد على خمس دقائق** (أقصى المهلة). وعرضٌ مات = صفر.
     */
    fun ringMs(expiresAt: String?, nowMs: Long): Long {
        val end = expiresAt?.let { runCatching { java.time.Instant.parse(it).toEpochMilli() }.getOrNull() }
            ?: return FALLBACK_MS
        return (end - nowMs).coerceIn(0L, MAX_MS)
    }

    /** **يرسم التنبيه** — ويعيد `false` إن مات العرضُ فلا شيء يُرسم. */
    fun show(context: Context, data: Map<String, String>, title: String, body: String, icon: Int): Boolean {
        val ms = ringMs(data["expires_at"], System.currentTimeMillis())
        if (ms <= 0L) return false
        val manager = context.getSystemService(NotificationManager::class.java) ?: return false
        ensureChannel(context, manager)

        val dest = Engagement.route(data["entity"], data["entity_id"])
        val intent = Intent(context, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_NEW_TASK)
            .putExtra(RahalPushService.EXTRA_DEST_TYPE, dest.type)
            .putExtra(RahalPushService.EXTRA_DEST_ID, dest.id)
        val open = PendingIntent.getActivity(
            context,
            dest.id.hashCode(),
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
        val note = NotificationCompat.Builder(context, CHANNEL)
            .setSmallIcon(icon)
            .setContentTitle(title.ifBlank { context.getString(R.string.push_ch_offer) })
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setAutoCancel(true)
            .setContentIntent(open)
            // **وبملء الشاشة والهاتفُ مقفل** — وإن لم يُسمح به عُرض فوق الشاشة كالعادة.
            .setFullScreenIntent(open, true)
            .setTimeoutAfter(ms)
            .build()
        // **ورنّةٌ تتكرّر حتّى يُلمس** — لا رنّةٌ واحدةٌ تفوته.
        note.flags = note.flags or Notification.FLAG_INSISTENT
        manager.notify(TAG, data["entity_id"].orEmpty().hashCode(), note)
        return true
    }

    /** **يُسكت كلَّ تنبيه عرض** — يُنادى حين يُفتح التطبيق: البطاقةُ أمامه. */
    fun cancelAll(context: Context) {
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        runCatching {
            manager.activeNotifications.filter { it.tag == TAG }.forEach { manager.cancel(TAG, it.id) }
        }
    }

    private fun ensureChannel(context: Context, manager: NotificationManager) {
        runCatching { manager.deleteNotificationChannel(OLD_CHANNEL) }
        if (manager.getNotificationChannel(CHANNEL) != null) return
        val sound = Uri.parse("android.resource://${context.packageName}/${R.raw.order_chime}")
        manager.createNotificationChannel(
            NotificationChannel(
                CHANNEL,
                context.getString(R.string.push_ch_offer_ring),
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                setSound(
                    sound,
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build(),
                )
                enableVibration(true)
                vibrationPattern = longArrayOf(0, 600, 400, 600, 400, 600)
                lockscreenVisibility = Notification.VISIBILITY_PUBLIC
            },
        )
    }
}
