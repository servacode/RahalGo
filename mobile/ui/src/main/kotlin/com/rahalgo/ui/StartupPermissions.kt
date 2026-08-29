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
 * **أذوناتُ الإقلاع — تُطلب مرّةً في أوّل فتحةٍ لا عند أوّل عطب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «أذوناتُ التطبيق يجب أن تكون من أوّل لحظة
 *  بعد تنزيل البرنامج. وقتَ أردتُ أحدّد موقعاً على الخريطة بالزرّ لا
 *  يستجيب، ذهبتُ إلى الحساب الشخصيّ ليظهر لي الإذن ثمّ عدتُ».)
 *
 * (وطلبُ التعميم ٢٠٢٦-٠٨-٢٦: «لازم أوّل ما يفتح التطبيق مشان ما ننسى
 *  أيّ إذنٍ يهمّنا، مع تفعيل الإشعارات».)
 *
 * # ولماذا مركزيّةٌ لا في كلّ تطبيق
 *
 * **وكانت في الزبون وحدَه، مكتوبةً في `MainActivity` بين مئتَي سطر** —
 * والسائقُ والمتجرُ والمندوبُ بلا شيء. **فصاحبُ المتجر يفتح تطبيقَه
 * ولا يعلم لماذا لا يصله إشعارُ طلب.**
 *
 * **وزرٌّ لا يستجيب أسوأُ من زرٍّ يسأل**: من ضغط ولم يحدث شيءٌ ظنّ
 * التطبيقَ معطّلاً، **ولم يخطر له أنّ النظام يمنعه.**
 *
 * # ولا تُطلب إلّا ما يعني التطبيق
 *
 * **وتطبيقٌ يسأل عن إذنٍ لا يستعمله يُقرأ فضوليّاً** — فيُرفض ما
 * يحتاجه معه. **فتُمرَّر الأذونُ المطلوبةُ صراحةً.**
 *
 * # ومرّةً واحدةً لا مع كلّ إقلاع
 *
 * **وأندرويد يُغلق البابَ بعد رفضين** — فمن سُئل وأبى لا يُسأل ثانيةً
 * عند كلّ فتحة. **والعلامةُ تُحفظ فلا يُستنزف الرفضُ الثاني في إقلاعٍ
 * لم ينتبه له صاحبُه.**
 *
 * # وواحدةٌ بعد واحدة لا دفعةً
 *
 * **ونافذتان تظهران معاً تُربك** — والثانيةُ تُضغط بلا قراءةٍ لأنّ
 * اليدَ ما زالت على الزرّ. **فتُطلب مجموعةً واحدةً يعرضها النظامُ
 * بالتتابع.**
 */
@Composable
fun AskStartupPermissions(
    /** **ما يعني هذا التطبيق** — انظر أعلاه. */
    permissions: List<String>,
    /** **مفتاحُ العلامة** — يختلف لو أردنا سؤالاً ثانياً بعد ميزةٍ جديدة. */
    key: String = "asked_startup_v1",
) {
    val context = LocalContext.current
    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { /* **وقرارُه قرارُه** — لا يُعاد سؤالُه ولا يُمنع من التطبيق. */ }

    LaunchedEffect(Unit) {
        val want = permissions.filter { !granted(context, it) }
        if (want.isEmpty()) return@LaunchedEffect
        val prefs = context.getSharedPreferences("rahalgo", Context.MODE_PRIVATE)
        if (prefs.getBoolean(key, false)) return@LaunchedEffect
        prefs.edit().putBoolean(key, true).apply()
        ask.launch(want.toTypedArray())
    }
}

/**
 * **الأذونُ التي يحتاجها تطبيقٌ يستقبل إشعاراتٍ ويحدّد موقعا.**
 *
 * **و`POST_NOTIFICATIONS` لا وجودَ لها قبل أندرويد ١٣** — وطلبُها هناك
 * يردّ رفضاً دائماً، **فتُحذف من القائمة لا تُطلب وتُرفض.**
 */
fun startupPermissions(notify: Boolean = true, location: Boolean = true): List<String> =
    buildList {
        if (notify && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            add(Manifest.permission.POST_NOTIFICATIONS)
        }
        if (location) add(Manifest.permission.ACCESS_FINE_LOCATION)
    }

private fun granted(context: Context, p: String): Boolean =
    ContextCompat.checkSelfPermission(context, p) == PackageManager.PERMISSION_GRANTED
