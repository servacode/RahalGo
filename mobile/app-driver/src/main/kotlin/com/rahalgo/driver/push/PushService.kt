package com.rahalgo.driver.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.rahalgo.driver.MainActivity
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **استقبال الإشعار — وهو ما يجعل السائق لا ينظر إلى تطبيقه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند الأوّل من قائمة المالك ٢٠٢٦-٠٨-١٢.)
 *
 * # لماذا لا يكفي البثّ الحيّ
 *
 * **البثّ (WS) يعمل والتطبيق مفتوح وحدَه** — وأكثرُ وقت السائق شاشتُه
 * مطفأة والجوّال في جيبه. **فبلا إشعار يفتح تطبيقه كلّ دقيقتين**، والطلب
 * يفوت لمن رآه أوّلا.
 *
 * # وقناتان لا واحدة
 *
 * **الطلب الجديد يرنّ ويهزّ ويعلو فوق كلّ شيء** — وما عداه (تحرّك محفظة،
 * خبر من المكتب) **يجلس في الشريط بلا صوت.** **ومن جعلهما قناةً واحدة**
 * إمّا أزعج صاحبه بكلّ خبر، **أو أسكت الطلب الذي ينتظره.**
 *
 * **وقناة النظام تُنشأ مرّة ولا تتبدّل بعدها**: من أراد تغيير صوتها بعد
 * الإنشاء **لا يقع تغييره** — يجب أن يحذفها المستخدم أو تُنشأ بمعرّفٍ
 * جديد.
 */
class PushService : FirebaseMessagingService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * **التوكن يتبدّل بلا سبب ظاهر** — تنصيبٌ جديد، أو مسحُ بيانات، أو
     * قرارٌ من غوغل. **ومن سجّله مرّة عند الدخول** بقي يرسل إلى جهازٍ
     * لم يعد يسمع.
     */
    override fun onNewToken(token: String) {
        register(token)
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val kind = data["kind"].orEmpty()
        // **والنصُّ من الحمولة أوّلاً** — (٢٠٢٦-٠٨-٢٣: صار المحرّكُ
        // يرسل بياناتٍ خالصةً بلا حقل `notification`، **ليصل التطبيقَ
        // وهو في الخلفيّة فيرسم إشعارَه بقناته وصوته.**)
        //
        // **ويبقى `message.notification` ارتداداً** — فأجهزةٌ لم
        // تُحدَّث بعدُ قد تستقبل رسالةً بالصيغة القديمة.
        val title = data["title"].orEmpty().ifBlank { message.notification?.title.orEmpty() }
        val body = data["body"].orEmpty().ifBlank { message.notification?.body.orEmpty() }
        Log.i(TAG, "إشعار: $kind — $title")

        // **والطلب الجديد يُعامَل معاملةً أخرى** — هو وحدَه ما ينتظره
        // السائق، **وما عداه خبرٌ يُقرأ حين يُفتح التطبيق.**
        val urgent = kind == KIND_OFFER
        show(title, body, urgent)
    }

    private fun show(title: String, body: String, urgent: Boolean) {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    if (urgent) com.rahalgo.ui.PushChannels.URGENT else com.rahalgo.ui.PushChannels.DEFAULT,
                    getString(if (urgent) R.string.push_ch_offer else R.string.push_ch_news),
                    if (urgent) {
                        NotificationManager.IMPORTANCE_HIGH
                    } else {
                        NotificationManager.IMPORTANCE_DEFAULT
                    },
                ).apply {
                    if (urgent) {
                        enableVibration(true)
                        // **ورجّةٌ طويلةٌ لمن على دراجة** — نبضةٌ قصيرة
                        // لا تُحسّ مع صوت الشارع والمحرّك.
                        vibrationPattern = longArrayOf(0, 500, 250, 500)
                    }
                },
            )
        }

        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        val note = NotificationCompat.Builder(this, if (urgent) com.rahalgo.ui.PushChannels.URGENT else com.rahalgo.ui.PushChannels.DEFAULT)
            .setSmallIcon(R.drawable.ic_orders)
            .setContentTitle(title.ifBlank { getString(R.string.push_offer_title) })
            .setContentText(body)
            .setAutoCancel(true)
            .setContentIntent(open)
            .setPriority(
                if (urgent) NotificationCompat.PRIORITY_MAX else NotificationCompat.PRIORITY_DEFAULT,
            )
            .setCategory(
                if (urgent) NotificationCompat.CATEGORY_CALL else NotificationCompat.CATEGORY_MESSAGE,
            )
            .build()

        // **ومعرّفٌ ثابتٌ للعرض** — طلبان يُعرضان معا لا يتراكمان في
        // الشريط: **الثاني يحلّ محلّ الأوّل**، والقائمة في التطبيق.
        manager.notify(if (urgent) ID_OFFER else ID_NEWS, note)
    }

    private fun register(token: String) {
        scope.launch {
            runCatching {
                Backend.of(applicationContext).devices.register(token)
            }.onFailure {
                // **وفشلُ التسجيل لا يُسقط شيئا** — يُعاد عند كلّ إقلاع.
                Log.w(TAG, "تعذّر تسجيل التوكن", it)
            }
        }
    }

    companion object {
        private const val TAG = "RahalGo/push"
        const val KIND_OFFER = "order_offer"
        private const val ID_OFFER = 2001
        private const val ID_NEWS = 2002
    }
}
