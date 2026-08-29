package com.rahalgo.driver.trip

import android.app.Activity
import android.app.PictureInPictureParams
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.util.Rational
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نافذةُ الملاحة الطافية — الخريطةُ لا تقف حين يخرج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٤: «لو نقدر نخلّي شاشةَ الخريطة تصغر في حال
 *  طلع من البرنامج — نافذةٌ منبثقةٌ صغيرة… مثلاً نظام غوغل ماب».)
 *
 * # ولماذا هذا لا خدمةٌ في الخلفيّة
 *
 * **وضعُ `Picture-in-Picture` يُبقي الشاشةَ مرئيّةً لا مطويّة** —
 * فالتركيبُ حيٌّ والخريطةُ ترسم والملاحةُ تقرأ الموقعَ والصوتُ يتكلّم.
 * **ونقلُ الملاحة إلى خدمةٍ يعيد بناءَ كلّ شيء** ليصل إلى ما يعطيه
 * سطران.
 *
 * **وأندرويد يمنح تطبيقاً في `PiP` أولويّةَ المرئيّ** — فلا يُخنق موقعُه
 * ولا يُقتل عند ضغط الذاكرة، **وهو ما لا تضمنه خدمةٌ عاديّة.**
 *
 * # ومتى يُدخَل إليه
 *
 * **عند مغادرة السائق وحدَها** (`onUserLeaveHint`) — **لا عند كلّ
 * `onPause`**: مكالمةٌ واردةٌ أو إشعارٌ يسحب الشاشةَ لحظةً، **ومن دخل
 * `PiP` حينها ترك نافذةً صغيرةً على شاشة القفل.**
 *
 * **ولا يُدخَل إلّا والملاحةُ تعمل** — نافذةٌ طافيةٌ لخريطةٍ ساكنةٍ
 * إزعاجٌ لا خدمة.
 */
object FloatingNav {

    /**
     * **أيدعم الجهازُ النافذةَ الطافية؟**
     *
     * **ولا يُفترض** — **وأجهزةٌ كثيرةٌ في سوريا أندرويد ٨ وما دونه**،
     * وبعضُها يُسقط الخاصّيّةَ وإن كان إصدارُه يدعمها.
     */
    fun supported(context: Context): Boolean =
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.O &&
            context.packageManager.hasSystemFeature(PackageManager.FEATURE_PICTURE_IN_PICTURE)

    /**
     * **نسبةُ النافذة** — أعرضُ من الطول قليلاً.
     *
     * **وأندرويد يرفض ما خرج عن ١:٢٫٣٩ و٢٫٣٩:١** — **ويُلقي استثناءً
     * لا يُمسَك فيسقط التطبيق**، فتُختار نسبةٌ آمنةٌ ثابتة.
     */
    private val ASPECT = Rational(3, 4)

    /**
     * **يدخل النافذةَ الطافية** — ويردّ هل دخل.
     *
     * **ولا يُسقط التطبيقَ إن رفض النظام**: بعضُ الأجهزة تمنعها
     * بسياسةٍ من الشركة المصنّعة، **وبعضُها يمنعها والشاشةُ مقسومة.**
     */
    fun enter(activity: Activity): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return false
        if (!supported(activity)) return false
        return runCatching {
            activity.enterPictureInPictureMode(
                PictureInPictureParams.Builder().setAspectRatio(ASPECT).build(),
            )
        }.getOrDefault(false)
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ النافذة — تقرؤها الشاشةُ فتُخفي ما لا يُقرأ في مربّعٍ صغير**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وفي مربّعٍ من بضعة سنتيمترات لا موضعَ لأزرارِ الرحلة ولا للحديث** —
 * **والضغطُ فيها لا يعمل أصلاً**: أندرويد لا يمرّر اللمسَ إلى نافذةٍ
 * طافية.
 *
 * **فتُعرض الخريطةُ والتعليمةُ التاليةُ وحدَهما.**
 */
class FloatingNavState {

    /** **أنحن في النافذة الطافية الآن؟** */
    var inPip by mutableStateOf(false)
        internal set

    /** **أتعمل الملاحة؟** — يكتبها من يعرف، ويقرؤها قرارُ الدخول. */
    var navigating by mutableStateOf(false)
}
