package com.rahalgo.ui

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import android.util.Log

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قنواتُ الإشعار — تُنشأ عند الإقلاع، وأسماؤها عقدٌ مع المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قِيس على جهاز المالك ٢٠٢٦-٠٨-٢٣: إشعارٌ يصل **بلا صوتٍ ولا اهتزاز**
 *  على قناة `fcm_fallback_notification_channel`.)
 *
 * # العطبُ الأوّل — اسمان لا يلتقيان
 *
 * **المحرّكُ يرسل `channel_id`** (`push/fcm.go`) والتطبيقُ ينشئ غيرَه:
 *
 *	المحرّك            التطبيقُ (قبل)
 *	rahalgo_urgent     rahalgo_order · rahalgo_offer
 *	rahalgo_default    rahalgo_news
 *
 * **وأندرويد لا يخطئ بصمت في هذه**: يجد قناةً لا وجودَ لها **فيرتدّ
 * إلى قناة فايربيس الاحتياطيّة** — **بلا صوتٍ ولا اهتزاز.**
 *
 * **وزبونٌ ينتظر سائقاً على الباب لا يشعر بشيء.**
 *
 * # العطبُ الثاني — وهو الأخبث
 *
 * **كانت القناةُ تُنشأ داخل `show()`** — أي **بعد أن تصل رسالة.**
 *
 * **ورسالةُ `notification` لا تُسلَّم إلى `onMessageReceived` والتطبيقُ
 * في الخلفيّة** — يرسمها النظامُ بنفسه. **فالقناةُ لا تُنشأ أبداً في
 * الحالة التي نحتاجها**: خلفيّةٌ وشاشةٌ مقفلة، وهي حالُ الزبون
 * الحقيقيّة.
 *
 * **فتُنشأ عند الإقلاع** — قبل أن تصل رسالةٌ واحدة.
 *
 * # وأسماؤها لا تُبدَّل بعد اليوم
 *
 * **وأندرويد لا يعيد ضبطَ قناةٍ أُنشئت** — من بدّل أهمّيّتَها بعد
 * التثبيت **لم يتبدّل شيءٌ على أجهزة من ثبّت قبله**، إلّا أن يُبدَّل
 * المعرّفُ نفسُه. **فيُفكَّر فيها مرّةً وتُثبَّت.**
 */
object PushChannels {

    /** **العاجل** — خبرُ طلبٍ ينتظره صاحبُه: يرنّ ويهتزّ. */
    const val URGENT = "rahalgo_urgent"

    /** **العاديّ** — ما يُقرأ حين يفتح التطبيق. */
    const val DEFAULT = "rahalgo_default"

    /**
     * **تُنشأ مرّةً عند الإقلاع** — و`createNotificationChannel` آمنةٌ
     * للتكرار: **تُنشئ إن لم تكن، وتُهمَل إن كانت.**
     */
    fun ensure(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val m = context.getSystemService(NotificationManager::class.java) ?: return
        runCatching {
            m.createNotificationChannel(
                NotificationChannel(
                    URGENT,
                    context.getString(R.string.push_ch_urgent),
                    NotificationManager.IMPORTANCE_HIGH,
                ).apply {
                    enableVibration(true)
                    // **ويُعرض على شاشة القفل** — **وإشعارٌ يُخفى عنها
                    // يصل ولا يُرى**، وهي الحالُ التي يُنتظر فيها.
                    lockscreenVisibility = android.app.Notification.VISIBILITY_PUBLIC
                },
            )
            m.createNotificationChannel(
                NotificationChannel(
                    DEFAULT,
                    context.getString(R.string.push_ch_default),
                    NotificationManager.IMPORTANCE_DEFAULT,
                ),
            )
        }.onFailure { Log.w("RahalGo/push", "تعذّر إنشاءُ القنوات", it) }
    }
}
