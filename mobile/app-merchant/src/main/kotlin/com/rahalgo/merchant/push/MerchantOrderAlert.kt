package com.rahalgo.merchant.push

import android.content.Context
import android.media.AudioAttributes
import android.media.RingtoneManager
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جرسُ الطلب الجديد — إنذارٌ يتكرّر ما دام الطلبُ يحتاج قراراً** (B2)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٩-٢٦: «تنبيهٌ واحدٌ لا يكفي — إنذارٌ صوتيٌّ قويٌّ
 *  متكرّرٌ ما دام الطلبُ قابلاً للفعل، يتوقّف فورَ القبول أو الرفض أو
 *  انقضاء المهلة أو معالجته من مكانٍ آخر».)
 *
 * # لماذا إنذارٌ داخليٌّ فوق الإشعار
 *
 * **الإشعارُ الفوريُّ** (`RahalPushService`، قناةُ `rahalgo_urgent`
 * العالية) **يرنّ مرّةً** — وصاحبُ المطعم يداه في العجين، **فرنّةٌ واحدةٌ
 * تمرّ ويبرد الطلب.** **فما دام التطبيقُ مفتوحاً وثمّة طلبٌ ينتظر قراراً،
 * يُعاد الإنذارُ كلَّ ثوانٍ** — نبضةٌ وصوتٌ قصير، لا صفّارةٌ متّصلة.
 *
 * # ويتوقّف فورَ زوال السبب — لا رنينَ يتيم
 *
 * **الحقيقةُ حقيقةُ الخادم لا ذاكرتُنا**: يُنادى `sync(hasActionable)` من
 * قائمةِ الطلبات الحيّة (تُنعشها النبضةُ والدورة، B4). **فإن قُبل الطلبُ أو
 * رُفض أو انقضت مهلتُه أو عُولج من جهازٍ آخر، خرج من «pending» في أوّل
 * إنعاش، فيتوقّف الجرس.** **ولا يُبعَث طلبٌ عولج**: الإقلاعُ يقرأ القائمةَ،
 * فإن لا طلبَ ينتظر فلا جرس.
 *
 * # وموافقٌ لأندرويد بلا حيلة
 *
 * **لا `full-screen intent` مقيَّدةٌ في أندرويد ١٤** — الإنذارُ الداخليُّ
 * صوتٌ واهتزازٌ ما دام التطبيقُ في المقدّمة، **والإشعارُ العالي** يتكفّل
 * بالخلفيّة والقفل حيث يسمح النظام. **ويُصمَت في الخلفيّة** (يُنادى
 * `stop` عند التواري) فلا يرنّ في جيبٍ مغلق بلا شاشة.
 */
object MerchantOrderAlert {

    private const val TAG = "RahalGo/جرس"

    /** **فترةُ إعادة الإنذار** — نبضةٌ كلَّ خمسِ ثوانٍ لا صفّارةٌ متّصلة. */
    private const val REPEAT_MS = 5_000L

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private var job: Job? = null

    /**
     * **يوائم الجرسَ مع الحقيقة** — يبدأ إن كان ثمّة طلبٌ يحتاج قراراً ولم
     * يكن يرنّ، ويتوقّف إن لم يعد. **آمنٌ للنداء المتكرّر**: لا يبدأ رنيناً
     * ثانياً فوق قائم.
     */
    fun sync(context: Context, hasActionable: Boolean) {
        if (hasActionable) start(context.applicationContext) else stop()
    }

    private fun start(app: Context) {
        if (job?.isActive == true) return
        job = scope.launch {
            while (isActive) {
                runCatching { beep(app) }.onFailure { Log.w(TAG, "تعذّر الصوت", it) }
                runCatching { buzz(app) }.onFailure { Log.w(TAG, "تعذّر الاهتزاز", it) }
                delay(REPEAT_MS)
            }
        }
    }

    /** **يُسكِت فوراً** — عند القبول/الرفض/زوال الطلب أو تواري التطبيق. */
    fun stop() {
        job?.cancel()
        job = null
    }

    /** **رنّةٌ قصيرةٌ من نغمة النظام** — لا ملفَّ نحمله ولا إذنَ نطلبه. */
    private fun beep(app: Context) {
        val uri = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION)
            ?: RingtoneManager.getDefaultUri(RingtoneManager.TYPE_ALARM)
            ?: return
        val ring = RingtoneManager.getRingtone(app, uri) ?: return
        // **وبنيّةِ إنذارٍ** — كي يُسمَع فوق وضع «الصامت مع الإنذارات».
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            ring.audioAttributes = AudioAttributes.Builder()
                .setUsage(AudioAttributes.USAGE_ALARM)
                .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                .build()
        }
        ring.play()
    }

    /** **اهتزازٌ قصير** — يلزمه إذنُ `VIBRATE` في البيان. */
    private fun buzz(app: Context) {
        val vibrator: Vibrator? = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            val mgr = app.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as? VibratorManager
            mgr?.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            app.getSystemService(Context.VIBRATOR_SERVICE) as? Vibrator
        }
        if (vibrator == null || !vibrator.hasVibrator()) return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            vibrator.vibrate(VibrationEffect.createOneShot(600, VibrationEffect.DEFAULT_AMPLITUDE))
        } else {
            @Suppress("DEPRECATION")
            vibrator.vibrate(600)
        }
    }
}
