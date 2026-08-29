package com.rahalgo.ui

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إذنُ الإشعارات — يُطلب حين يصير له معنى**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تدقيقُ الجاهزيّة ٢٠٢٦-٠٨-١٩: قِيس أنّ حزمةَ الزبون بلا الإذن،
 *  وتطبيقُ السائق يحمله — فالزبونُ وحدَه ناقص.)
 *
 * # وبلاه تُبتلع الإشعاراتُ بصمت
 *
 * **من أندرويد ١٣ لا يُعرض إشعارٌ بلا إذن** — يُرسله الخادمُ ويقبله
 * الجهازُ **ولا يظهر.** لا خطأَ ولا سجلّ. **فلا يعرف الزبونُ أنّ
 * طلبَه قُبل ولا أنّ السائقَ وصل**، ويفتح التطبيقَ كلَّ دقيقةٍ يسأل.
 *
 * # ولماذا لا يُطلب عند أوّل فتحة
 *
 * **نافذةٌ تظهر قبل أن يرى التطبيقَ تُرفض** — لا يعرف ما هذا التطبيقُ
 * ولا لماذا يريد أن يزعجه. **والرفضُ في أندرويد لا يُعاد سؤالُه
 * مرّتين**، ثمّ يُقفل البابُ إلى إعدادات النظام.
 *
 * # فيُطلب بعد الدخول
 *
 * **حينها صار له حسابٌ وطلباتٌ تُتابَع** — والإشعارُ يخصّه هو.
 * **ومن دخل يفهم لماذا يُسأل.**
 *
 * # ولا يُسأل من رفض
 *
 * **النظامُ نفسُه يُسقط النداءَ الثاني** بعد رفضين، **ونافذةٌ لا تظهر
 * ولا تُخبر أحداً تُقرأ صمتا.** فيُنادى مرّةً لكلّ حياةِ شاشة، وما
 * بعدها قرارُ صاحبه.
 */
@Composable
fun AskNotifyPermission(enabled: Boolean) {
    // **وما دون أندرويد ١٣ لا إذنَ أصلاً** — الإشعارُ يعمل بلا سؤال.
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return

    val context = LocalContext.current
    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { /* **وقرارُه قرارُه** — لا يُعاد سؤالُه ولا يُمنع من التطبيق. */ }

    LaunchedEffect(enabled) {
        if (!enabled || notifyGranted(context)) return@LaunchedEffect

        // ══════════════════════════════════════════════════════════════
        // **ولا يُسأل مرّتين في جلسةٍ واحدة**
        // ══════════════════════════════════════════════════════════════
        //
        // **`AskStartupPermissions` تسأل عند أوّل فتح** (قرارُ المالك
        // ٢٠٢٦-٠٨-٢٦: «لازم أوّل ما يفتح التطبيق مشان ما ننسى أيّ
        // إذن»). **وهذه كانت تسأل بعدها مباشرةً.**
        //
        // **ورفضتان متتاليتان في أندرويد تقفلان الإذنَ نهائيّاً** — لا
        // نافذةَ بعدهما أبداً، والعلاجُ في إعدادات النظام لا في
        // التطبيق. **فرفضةٌ واحدةٌ من المستخدم كانت تصير قفلاً دائماً.**
        //
        // **والعلامةُ نفسُها** التي تكتبها `AskStartupPermissions` —
        // فمن سُئل هناك لا يُسأل هنا، **ومن لم يُسأل (تطبيقٌ لا يطلب
        // الإشعاراتِ عند الإقلاع) يُسأل هنا كما كان.**
        val prefs = context.getSharedPreferences("rahalgo", Context.MODE_PRIVATE)
        if (prefs.getBoolean("asked_startup_v1", false)) return@LaunchedEffect

        ask.launch(Manifest.permission.POST_NOTIFICATIONS)
    }
}

/** **أمُنح الإذن؟** — وما دون أندرويد ١٣ ممنوحٌ دائما. */
fun notifyGranted(context: Context): Boolean =
    Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
        ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
        PackageManager.PERMISSION_GRANTED
