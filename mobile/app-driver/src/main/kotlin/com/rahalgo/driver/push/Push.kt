package com.rahalgo.driver.push

import android.content.Context
import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.rahalgo.driver.data.Backend
import kotlinx.coroutines.tasks.await

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تسجيل الجهاز — عند كلّ إقلاع لا مرّة واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والتوكن يتبدّل بلا سبب ظاهر**: تنصيبٌ جديد · مسحُ بيانات · قرارٌ من
 * غوغل. **ومن سجّله يوم الدخول وحدَه** بقي المحرّك يرسل إلى جهازٍ لم يعد
 * يسمع — **والسائق ينتظر طلبا لا يرنّ.**
 *
 * **ويُسجَّل بعد الدخول لا قبله**: النقطة تحتاج توكن حساب، **ومن سجّل
 * قبله ربط الجهاز بلا أحد.**
 */
object Push {

    suspend fun register(context: Context) {
        try {
            val token = FirebaseMessaging.getInstance().token.await()
            Backend.of(context).devices.register(token, version(context))
            Log.i(TAG, "سُجّل الجهاز")
        } catch (e: Exception) {
            // **وفشلُه لا يمنع الدخول** — الطلبات تصل بالبثّ الحيّ ما دام
            // التطبيق مفتوحا، **ويُعاد التسجيل في الإقلاع التالي.**
            Log.w(TAG, "تعذّر تسجيل الجهاز", e)
        }
    }

    /**
     * **يُلغى عند الخروج** — **وإلّا وصلت طلباتُ حسابٍ خرج إلى جهازه**،
     * أو وصلت إشعاراتُ سائقٍ إلى جوّالٍ سلّمه لغيره.
     */
    suspend fun unregister(context: Context) {
        try {
            val token = FirebaseMessaging.getInstance().token.await()
            Backend.of(context).devices.unregister(token)
        } catch (e: Exception) {
            Log.w(TAG, "تعذّر إلغاء تسجيل الجهاز", e)
        }
    }

    private fun version(context: Context): String =
        runCatching {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName.orEmpty()
        }.getOrDefault("")

    private const val TAG = "RahalGo/push"
}
