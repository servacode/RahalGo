package com.rahalgo.ui

import android.content.Context
import android.content.Intent
import android.location.LocationManager
import android.os.Build
import android.provider.Settings
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«موقعي» — تُجيب في الحال ثمّ تدقّق** (`ML`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # المسألة
 *
 * **وكان الزرُّ ينادي `Here.refresh` ويمضي** — **لا حالَ ولا جواب.**
 * **فيضغط صاحبُه ولا يقع شيءٌ يُرى**: **ثانيةً، ثمّ ثانيةً** —
 * **وكلُّ ضغطةٍ تفتح نداءَ موضعٍ جديداً.**
 *
 * **وزرٌّ يُضغط ولا يتبدّل شيءٌ يُقرأ معطوباً** — **ومن ظنّه معطوباً
 * ضغطه خمساً.**
 *
 * # ومرحلتان لا واحدة
 *
 * **وآخرُ موضعٍ معروفٍ يصل في اللحظة** — **فيُوسَّط به الخريطةُ
 * فوراً**، **ثمّ يُطلَب قياسٌ طازجٌ فيُصحَّح.**
 *
 * **ولا يُتَّخذ القديمُ حقيقةَ توصيل** — **وعقدُ الدفعة الخامسة قائم**:
 * **نقطةُ الاستكشاف تُخبِر ولا تحكم**، **والتأكيدُ فعلٌ صريح.**
 *
 * # ولمَ هنا لا في كلّ تطبيق
 *
 * **وأربعةُ تطبيقاتٍ تسأل السؤالَ نفسَه** — **ونسخةٌ لكلٍّ تفترق
 * يوماً**: **فيُدقَّق في واحدٍ ويبقى الثلاثةُ كما كانوا.**
 */
class Locating {

    /** **ما يمنع الموضعَ — ولكلٍّ علاجُه.** */
    enum class Problem {
        /** **رُفض الإذن** — ويُطلَب ثانيةً. */
        PERMISSION_DENIED,

        /** **رُفض نهائيّاً** — **ولا نافذةَ بعدها**: الإعدادات. */
        PERMISSION_PERMANENT,

        /** **خدمةُ الموقع مطفأةٌ في النظام** — **والإذنُ لا يُصلحها.** */
        SERVICE_OFF,

        /** **طال الانتظارُ ولم يصل قياس.** */
        TIMEOUT,

        /** **وصل قياسٌ لا تُعرَف دقّتُه أو هي ضعيفةٌ جدّاً.** */
        WEAK_ACCURACY,

        /** **سقط المزوّدُ لسببٍ لا نعرفه** — **ولا يُدَّعى علمٌ به.** */
        UNAVAILABLE,
    }

    /** **حالُ الطلب** — تقرؤها الشاشةُ لترسم الزرّ. */
    enum class Phase {
        /** **ساكن.** */
        IDLE,

        /** **يُطلب الآن** — **والزرُّ يدور ولا يُقبل ضغطاً ثانياً.** */
        ACQUIRING,

        /** **وصل موضعٌ يُعتَمد للتوسيط.** */
        FIXED,

        /** **تعذّر** — والسببُ في `problem`. */
        FAILED,
    }

    var phase by mutableStateOf(Phase.IDLE)
        private set

    var problem by mutableStateOf<Problem?>(null)
        private set

    /** **أهذا موضعٌ قديمٌ عُرض للتوسيط ريثما يصل الطازج؟** */
    var provisional by mutableStateOf(false)
        private set

    /** **أفي الجوّ نداء؟** — **يقرؤها الزرُّ ليُعطَّل** (`ML-04`). */
    val busy: Boolean get() = phase == Phase.ACQUIRING

    /**
     * start **يبدأ طلباً واحداً** — **ويردّ `false` إن كان قائماً.**
     *
     * **وضغطتان سريعتان طلبٌ واحد** — **ولا نداءان للموضع يتسابقان
     * فيصل أقدمُهما آخراً فيُوسَّط به.**
     */
    fun start(): Boolean {
        if (busy) return false
        phase = Phase.ACQUIRING
        problem = null
        provisional = false
        return true
    }

    /**
     * provisional **موضعٌ قديمٌ يُوسَّط به الآن** — **والطلبُ باقٍ.**
     *
     * **ولا يُنهي الطلبَ** — **فالطازجُ لم يصل بعد**، **والزرُّ يبقى
     * يدور حتّى يصل.**
     */
    fun provisional() {
        if (phase != Phase.ACQUIRING) return
        provisional = true
    }

    /** **وصل الطازج.** */
    fun fixed() {
        phase = Phase.FIXED
        problem = null
        provisional = false
    }

    /** **تعذّر — بسببه.** */
    fun failed(why: Problem) {
        phase = Phase.FAILED
        problem = why
        provisional = false
    }

    /** **يُصفَّر عند مغادرة الشاشة.** */
    fun reset() {
        phase = Phase.IDLE
        problem = null
        provisional = false
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أتكفي هذه الدقّةُ لِما نريد؟** (`ML-05`، `ML-11`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وتوسيطُ خريطةٍ يقبل قياساً خشناً** — **ومئةُ مترٍ لا تُرى على
 * مستوى المدينة.**
 *
 * **وتأكيدُ نقطةِ تسليمٍ لا يقبله** — **فمئةُ مترٍ بابٌ آخر**،
 * **وحدُّ المنطقة يُقاس بمئات الأمتار** (كما في نقطة الاستكشاف).
 *
 * **وتتبّعُ السائق أشدُّها** — **والمسافةُ والمسارُ يُبنيان عليه.**
 */
object Accuracy {

    /** **يكفي لتوسيط خريطة.** */
    const val CENTER_M: Float = 1000f

    /** **يكفي لتأكيد نقطةِ توصيلٍ أو التقاط.** */
    const val CONFIRM_M: Float = 500f

    /** **يكفي لتتبّع سائقٍ في رحلة.** */
    const val DRIVE_M: Float = 100f

    /** **أتكفي؟** — **وغيرُ المعروفة لا تكفي لشيء.** */
    fun enough(accuracyM: Float, limit: Float): Boolean =
        accuracyM >= 0f && accuracyM <= limit
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ولكلّ تعذّرٍ نصُّه وعلاجُه** (`MLW-06`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **و«حدث خطأ» جوابٌ لا يُعمَل به** — **وعلاجُ الإذن المرفوض غيرُ
 * علاج الخدمة المطفأة**: **ومن قيل له «حاول ثانية» وخدمتُه مطفأةٌ
 * حاول عشراً.**
 *
 * **وهنا لا في كلّ شاشة** — **وأربعُ نسخٍ تفترق يوماً.**
 */
object LocatingText {

    /** **ما وقع — سطراً واحداً يقرؤه صاحبُه.** */
    fun message(ctx: Context, p: Locating.Problem): String = ctx.getString(
        when (p) {
            Locating.Problem.PERMISSION_DENIED -> R.string.loc_fail_denied
            Locating.Problem.PERMISSION_PERMANENT -> R.string.loc_fail_permanent
            Locating.Problem.SERVICE_OFF -> R.string.loc_fail_service_off
            Locating.Problem.TIMEOUT -> R.string.loc_fail_timeout
            Locating.Problem.WEAK_ACCURACY -> R.string.loc_fail_weak
            Locating.Problem.UNAVAILABLE -> R.string.loc_fail_unavailable
        },
    )

    /** **وما يفعله** — **زرٌّ يقول فعلَه لا «حسناً».** */
    fun action(ctx: Context, p: Locating.Problem): String = ctx.getString(
        when (p) {
            Locating.Problem.PERMISSION_DENIED -> R.string.loc_fail_denied_fix
            Locating.Problem.PERMISSION_PERMANENT -> R.string.loc_fail_permanent_fix
            Locating.Problem.SERVICE_OFF -> R.string.loc_fail_service_off_fix
            Locating.Problem.TIMEOUT -> R.string.loc_fail_timeout_fix
            Locating.Problem.WEAK_ACCURACY -> R.string.loc_fail_weak_fix
            Locating.Problem.UNAVAILABLE -> R.string.loc_fail_unavailable_fix
        },
    )
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أخدمةُ الموقع مُشغَّلةٌ في الجهاز؟** (`MLW-03`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والإذنُ ممنوحٌ والخدمةُ مطفأةٌ حالٌ تقع كلَّ يوم** — **ولا نافذةَ
 * تُرفَض ولا رسالةَ تظهر**: **يُنادى المزوّدُ فلا يجيب أبداً.**
 *
 * **وهنا لأنّ أربعةَ تطبيقاتٍ تسأله** — **وكانت جاهزيّةُ السائق
 * وحدَها تعرفه.**
 *
 * **وعطبُ القراءة لا يُقرأ منعاً** — **المنعُ بعلمٍ لا بجهل.**
 */
fun locationServiceEnabled(context: Context): Boolean {
    val lm = ContextCompat.getSystemService(context, LocationManager::class.java)
        ?: return true
    return runCatching {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            lm.isLocationEnabled
        } else {
            @Suppress("DEPRECATION")
            lm.isProviderEnabled(LocationManager.GPS_PROVIDER) ||
                lm.isProviderEnabled(LocationManager.NETWORK_PROVIDER)
        }
    }.getOrDefault(true)
}

/**
 * **يفتح صفحةَ إعدادات الموقع في النظام.**
 *
 * **ولا تُفتح «معلوماتُ التطبيق» لهذا** — **فالخطأُ ليس في أذوننا بل
 * في مفتاح النظام**، **ومن وقع على صفحة التطبيق بحث فيها عمّا ليس
 * فيها.**
 */
fun openLocationSettings(context: Context) {
    context.startActivity(
        Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
    )
}

/** **يفتح صفحةَ التطبيق في النظام** — **لمن رُفض نهائيّاً.** */
fun openAppSettings(context: Context) {
    context.startActivity(
        Intent(
            Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
            android.net.Uri.fromParts("package", context.packageName, null),
        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
    )
}

/**
 * **العلاجُ المناسبُ للتعذّر** — **يُنادى من زرّ اللافتة.**
 *
 * **ويردّ `true` إن كان العلاجُ محاولةً ثانيةً** — **فينادي صاحبُه
 * التحديدَ من جديد**، **و`false` إن فُتحت صفحةُ نظام.**
 */
fun fixProblem(context: Context, p: Locating.Problem): Boolean = when (p) {
    Locating.Problem.SERVICE_OFF -> {
        openLocationSettings(context)
        false
    }
    Locating.Problem.PERMISSION_PERMANENT -> {
        openAppSettings(context)
        false
    }
    // **والإذنُ المرفوضُ مرّةً يُطلَب ثانيةً** — **والطلبُ عند صاحب
    // الشاشة لأنّ نافذةَ النظام تحتاج نشاطاً.**
    else -> true
}
