package com.rahalgo.customer.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.rahalgo.customer.MainActivity
import com.rahalgo.customer.R
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يصل والتطبيقُ مغلق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (سؤالُ المالك ٢٠٢٦-٠٨-١٩: «المفروض يصل إشعارٌ لو التطبيق مغلق صحيح؟».
 *  وكان الجوابُ حينها: **لا** — والشيفرةُ في السائق وحدَه.)
 *
 * # ولماذا لا يكفي البثُّ الحيّ
 *
 * **المقبسُ يعمل والتطبيقُ مفتوحٌ وحدَه.** ومن أغلقه لا يعرف أنّ سائقَه
 * يسأله «أيّ طابق؟» — **فيقف السائقُ على الباب ينتظر جواباً لا يأتي.**
 *
 * # وقناتان لا واحدة
 *
 * **رسالةُ السائق تُقاطِع، وخبرُ عرضٍ ينتظر.** **وقناةٌ واحدةٌ لهما
 * تجعل صاحبَها يُطفئ الصوتَ كلَّه** حين تكثر العروض — فيفوته ما يهمّ.
 *
 * # والقناةُ لا يتغيّر إلحاحُها بعد إنشائها
 *
 * **أندرويد يثبّتها عند أوّل إنشاء** — ومن أراد تغييرَه بعدها فليُنشئ
 * قناةً بمعرّفٍ جديد. **فتُضبط صحيحةً من أوّل مرّة.**
 */
class PushService : FirebaseMessagingService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * **والتوكنُ يتبدّل بلا سببٍ ظاهر** — تنصيبٌ جديدٌ أو مسحُ بيانات أو
     * قرارٌ من غوغل. **ومن سجّله مرّةً عند الدخول** بقي المحرّكُ يرسل
     * إلى جهازٍ لم يعد يسمع.
     */
    override fun onNewToken(token: String) {
        scope.launch {
            runCatching { AppCore.get().devices.register(token) }
                .onFailure { Log.w(TAG, "تعذّر تسجيلُ التوكن", it) }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val kind = data["kind"].orEmpty()
        val title = message.notification?.title ?: data["title"].orEmpty()
        val body = message.notification?.body ?: data["body"].orEmpty()
        Log.i(TAG, "إشعار: $kind — $title")
        // **وخبرُ الطلب يُقاطِع** — هو ما ينتظره صاحبُه، **وما عداه
        // يُقرأ حين يفتح التطبيق.**
        show(title, body, urgent = kind == KIND_ORDER)
    }

    private fun show(title: String, body: String, urgent: Boolean) {
        if (title.isBlank() && body.isBlank()) return
        val manager = getSystemService(NotificationManager::class.java)
        val channel = if (urgent) CH_ORDER else CH_NEWS
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    channel,
                    getString(if (urgent) R.string.push_ch_order else R.string.push_ch_news),
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
            Intent(this, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        val note = NotificationCompat.Builder(this, channel)
            .setSmallIcon(R.drawable.ic_orders)
            .setContentTitle(title.ifBlank { getString(R.string.app_name) })
            .setContentText(body)
            // **والنصُّ الطويلُ يُقرأ كلُّه** — **ورسالةُ سائقٍ مقصوصةٌ
            // بنقطتين تُفتح لتُقرأ**، وقد يكون فيها ما يُجاب فورا.
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .setContentIntent(open)
            .setPriority(
                if (urgent) NotificationCompat.PRIORITY_HIGH else NotificationCompat.PRIORITY_DEFAULT,
            )
            .build()

        // **ومعرّفٌ ثابتٌ لكلّ نوع** — **وإشعاراتٌ تتراكم في الشريط
        // تُقرأ ازدحاماً فتُمسح كلُّها**، والسجلُّ في التطبيق.
        manager.notify(if (urgent) ID_ORDER else ID_NEWS, note)
    }

    companion object {
        private const val TAG = "RahalGo/push"

        /** **نوعُ خبرِ الطلب كما يرسله المحرّك** (`notifications.KindOrder`). */
        const val KIND_ORDER = "order"
        private const val CH_ORDER = "rahalgo_order"
        private const val CH_NEWS = "rahalgo_news"
        private const val ID_ORDER = 3001
        private const val ID_NEWS = 3002
    }
}
