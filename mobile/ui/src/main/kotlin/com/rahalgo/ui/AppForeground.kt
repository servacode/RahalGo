package com.rahalgo.ui

import android.app.Activity
import android.app.Application
import android.os.Bundle

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إشارةُ ظهورِ التطبيق — على مستوى العمليّة لا نشاطٍ بعينه** (Obs 3.1)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **لماذا**: إشعارُ الأمانِ «سُجّل الدخول من جهازٍ آخر» يُعرَض نظاميّاً **في
 * الخلفيّة وحدَها** — أمّا في المقدّمة فالرسالةُ داخلَ التطبيق تكفي، ولا يُكرَّر
 * صندوقٌ نظاميّ. **فيلزم إشارةُ ظهورٍ موثوقة.**
 *
 * **ولماذا هذا لا `ProcessLifecycleOwner`**: `onMessageReceived` يجيء على خيطِ
 * FCM في الخلفيّة، **و`ProcessLifecycleOwner.get()` واجهةُ الخيط الرئيس** —
 * فقراءتُها من غيره غيرُ آمنة. **وعدّادٌ `@Volatile`** يُقرأ من أيّ خيط.
 *
 * **وعدُّ النشاطاتِ المبدوءةِ لا المستأنَفة**: البدء/الإيقاف (`onStart/onStop`)
 * حدُّ «مرئيّ»، وهو المقصودُ بـ«ظاهر»؛ الاستئنافُ/الإيقافُ المؤقّت أضيق. **ولا
 * نشاطٌ بعينه** — أيُّ نشاطٍ للتطبيق يرفع العدّاد.
 */
object AppForeground {

    @Volatile
    private var started = 0

    /** **أظاهرٌ التطبيقُ الآن؟** — يُقرأ بأمانٍ من أيّ خيط. */
    val isForeground: Boolean get() = started > 0

    /** يُركَّب مرّةً في `Application.onCreate`. */
    fun install(app: Application) {
        app.registerActivityLifecycleCallbacks(object : Application.ActivityLifecycleCallbacks {
            override fun onActivityStarted(activity: Activity) {
                started++
            }

            override fun onActivityStopped(activity: Activity) {
                if (started > 0) started--
            }

            override fun onActivityCreated(activity: Activity, savedInstanceState: Bundle?) {}
            override fun onActivityResumed(activity: Activity) {}
            override fun onActivityPaused(activity: Activity) {}
            override fun onActivitySaveInstanceState(activity: Activity, outState: Bundle) {}
            override fun onActivityDestroyed(activity: Activity) {}
        })
    }
}
