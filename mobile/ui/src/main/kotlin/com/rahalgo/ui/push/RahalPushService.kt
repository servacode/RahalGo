package com.rahalgo.ui.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.PushChannels
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قاعدةُ الإشعارات — تُكتب مرّةً وترثها التطبيقاتُ الأربعة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «الأشياءُ المركزيّةُ المشتركةُ بين التطبيقات
 *  لا تنساها».)
 *
 * # ولماذا وُجدت
 *
 * **كانت في الزبون والسائق نسختان متطابقتان ٨٥٪** — مئةٌ وتسعةٌ
 * وعشرون سطراً ومئةٌ واثنان وأربعون. **وتطبيقُ المتجر لم يكن فيه شيءٌ
 * أصلاً، فكان المتجرُ لا يعلم بطلبٍ إلّا إن كان التطبيقُ مفتوحاً** —
 * **وذاك أخطرُ عطبٍ فيه: متجرٌ لا يُنبَّه يخسر الطلب.**
 *
 * **ونسختان تتباعدان صامتتين**: يُصلَح عطبٌ في إحداهما ويبقى في الأخرى.
 *
 * # وما يبقى للتطبيق
 *
 * **خمسةٌ لا سادسَ لها** — وما عداها واحدٌ للجميع:
 *
 *   `home()`               الشاشةُ التي تُفتح عند الضغط
 *   `icon()`               أيقونةُ الإشعار
 *   `appName()`            يُعرض حين يصل إشعارٌ بلا عنوان
 *   `isUrgent(kind)`       أيُّ نوعٍ يُوقظ الجهاز
 *   `urgent/newsChannel()` اسمُ القناة في إعدادات النظام
 *
 * # وتسجيلُ التوكن هنا لا هناك
 *
 * **ورمزُ الجهاز يُسجَّل في المحرّك عند كلّ تجديد** — ومن نسيه في
 * تطبيقٍ صار ذلك التطبيقُ لا يستقبل شيئاً، **ولا يظهر العطبُ إلّا حين
 * يشكو صاحبُه أنّ الطلباتِ لا تصله.** ولهذا `final`.
 */
abstract class RahalPushService : FirebaseMessagingService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /** **الشاشةُ التي تُفتح عند ضغط الإشعار.** */
    protected abstract fun home(): Class<*>

    /** **أيقونةُ شريط الإشعارات** — أحاديّةُ اللون كما يشترط أندرويد. */
    protected abstract fun icon(): Int

    /** **اسمُ التطبيق** — يُعرض حين يصل إشعارٌ بلا عنوان. */
    protected abstract fun appName(): String

    /** **اسمُ القناة العاجلة** كما يراه المستخدمُ في إعدادات النظام. */
    protected abstract fun urgentChannelName(): String

    /** **اسمُ القناة العاديّة.** */
    protected abstract fun newsChannelName(): String

    /**
     * **أيُّ نوعٍ يستحقّ إيقاظَ الجهاز.**
     *
     * **ويختلف بين تطبيق**: عرضٌ للسائق · طلبٌ للزبون · طلبٌ جديدٌ
     * للمتجر. **والافتراضُ لا شيء** — فمن لم يقرّر لا يوقظ أحداً.
     */
    protected open fun isUrgent(kind: String): Boolean = false

    final override fun onNewToken(token: String) {
        scope.launch {
            runCatching { AppCore.get().devices.register(token) }
                .onFailure { Log.w(TAG, "تعذّر تسجيلُ التوكن", it) }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val kind = data["kind"].orEmpty()
        // **والحقلُ المخصَّصُ يسبق حقلَ فايربيس** — الأوّلُ يصل والتطبيقُ
        // مغلقٌ أيضاً، **والثاني يبتلعه النظامُ فلا يبلغ الشيفرة.**
        val title = data["title"].orEmpty().ifBlank { message.notification?.title.orEmpty() }
        val body = data["body"].orEmpty().ifBlank { message.notification?.body.orEmpty() }
        Log.i(TAG, "إشعار: $kind — $title")
        show(title, body, isUrgent(kind))
    }

    protected fun show(title: String, body: String, urgent: Boolean) {
        // **ولا إشعارَ فارغ** — صندوقٌ بلا نصٍّ يُقلق ولا يُفيد.
        if (title.isBlank() && body.isBlank()) return
        val manager = getSystemService(NotificationManager::class.java)
        val channel = if (urgent) PushChannels.URGENT else PushChannels.DEFAULT
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    channel,
                    if (urgent) urgentChannelName() else newsChannelName(),
                    if (urgent) {
                        NotificationManager.IMPORTANCE_HIGH
                    } else {
                        NotificationManager.IMPORTANCE_DEFAULT
                    },
                ).apply { if (urgent) enableVibration(true) },
            )
        }
        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, home()).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
        val note = NotificationCompat.Builder(this, channel)
            .setSmallIcon(icon())
            .setContentTitle(title.ifBlank { appName() })
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .setContentIntent(open)
            .setPriority(
                if (urgent) NotificationCompat.PRIORITY_HIGH else NotificationCompat.PRIORITY_DEFAULT,
            )
            .build()
        // **ومعرّفانِ لا واحد**: العاجلُ لا يطمس الخبر، والخبرُ لا يطمس
        // العاجل.
        manager.notify(if (urgent) ID_URGENT else ID_NEWS, note)
    }

    companion object {
        private const val TAG = "RahalGo/push"
        private const val ID_URGENT = 3001
        private const val ID_NEWS = 3002
    }
}
