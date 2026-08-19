package com.rahalgo.ui

import android.content.Context
import android.util.Log
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.platform.LocalContext
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تسجيلُ الجهاز للدفع — بابٌ واحدٌ للتطبيقات الثلاثة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (سؤالُ المالك ٢٠٢٦-٠٨-١٩: «المفروض يصل إشعارٌ لو التطبيق مغلق صحيح؟».
 *  وكان الجوابُ: لا — الشيفرةُ في السائق وحدَه والمفتاحُ غيرُ مضبوط.)
 *
 * # ولماذا هنا لا في كلّ تطبيق
 *
 * **كانت في `app-driver` وحدَه** — **وهي عائلةُ «بُني مرّةً ونُسي في
 * الباقي»** التي تكرّرت في هذا المستودع ستَّ مرّات: الرسالةُ الطافية
 * وشريطُ الشبكة والافتتاح وجالبُ الصور وإذنُ الإشعارات وقرصُ الحديث.
 *
 * **فتُكتب مرّةً في `:ui`** — وثلاثةُ تركيباتٍ تُنسى في واحد.
 *
 * # ويُسجَّل عند كلّ إقلاعٍ لا مرّةً واحدة
 *
 * **والتوكنُ يتبدّل بلا سببٍ ظاهر**: تنصيبٌ جديدٌ · مسحُ بيانات · قرارٌ
 * من غوغل. **ومن سجّله يومَ الدخول وحدَه** بقي المحرّكُ يرسل إلى جهازٍ
 * لم يعد يسمع — **والزبونُ ينتظر خبراً لا يرنّ.**
 *
 * # وبعد الدخول لا قبله
 *
 * **النقطةُ تحتاج توكنَ حساب** — **ومن سجّل قبله ربط الجهازَ بلا أحد.**
 */
object Push {

    suspend fun register(context: Context) {
        try {
            val token = FirebaseMessaging.getInstance().token.await()
            AppCore.get().devices.register(token, version(context))
            Log.i(TAG, "سُجّل الجهاز")
        } catch (e: Exception) {
            // **وفشلُه لا يمنع الدخول** — الأخبارُ تصل بالبثّ الحيّ ما
            // دام التطبيقُ مفتوحا، **ويُعاد التسجيلُ في الإقلاع التالي.**
            Log.w(TAG, "تعذّر تسجيلُ الجهاز", e)
        }
    }

    /**
     * **يُلغى عند الخروج** — **وإلّا وصلت أخبارُ حسابٍ خرج إلى جهازه**،
     * أو وصلت إشعاراتُ صاحبِه إلى جوّالٍ سلّمه لغيره.
     */
    suspend fun unregister(context: Context) {
        try {
            val token = FirebaseMessaging.getInstance().token.await()
            AppCore.get().devices.unregister(token)
        } catch (e: Exception) {
            Log.w(TAG, "تعذّر إلغاءُ تسجيل الجهاز", e)
        }
    }

    private fun version(context: Context): String =
        runCatching {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName.orEmpty()
        }.getOrDefault("")

    private const val TAG = "RahalGo/push"
}

/**
 * **يُسجَّل ما دام داخلاً** — يُركَّب مرّةً في جذر التطبيق.
 *
 * **ومفتاحُه معرّفُ الحساب لا رايةُ «داخل»** — **ومن بدّل حسابَه على
 * الجهاز نفسِه** تُسجَّل أجهزتُه من جديد، **وإلّا بقيت أخبارُ الأوّل
 * تصل الثاني.**
 */
@Composable
fun RegisterPush(userId: String?) {
    val context = LocalContext.current
    LaunchedEffect(userId) {
        if (userId.isNullOrEmpty()) return@LaunchedEffect
        Push.register(context)
    }
}
