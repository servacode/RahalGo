package com.rahalgo.driver.location

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.core.content.ContextCompat

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إذن الموقع — ثلاث درجات لا واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * ١ · **تقريبيّ** (حيّ) — لا يكفي لمسافة ولا لتتبّع.
 * ٢ · **دقيق أثناء الاستعمال** — يكفي والتطبيق مفتوح.
 * ٣ · **طوال الوقت** — وهو المطلوب: **الجوّال في الجيب والشاشة مطفأة.**
 *
 * # ولماذا يُطلبان على مرحلتين
 *
 * **أندرويد ١١ فما فوق يرفض الطلبين معا**: من طلب «طوال الوقت» مع
 * الأوّل **رُدّ طلبه كلّه**، ولا يُعرض على صاحبه شيء. **فيُطلب الدقيق
 * أوّلا، ثمّ يُطلب التوسيع.**
 *
 * # و«طوال الوقت» لا تُمنح من نافذة
 *
 * **من أندرويد ١١ يفتح النظام الإعدادات** ولا يعرض نافذة قبول — فيجب أن
 * يُقال للسائق ما الذي يختاره هناك، **وإلّا خرج من الإعدادات بلا أن
 * يغيّر شيئا.**
 */
object LocationPermission {

    /** أدنى ما يُشغّل الخدمة: موقع دقيق أثناء الاستعمال. */
    val FOREGROUND = arrayOf(
        Manifest.permission.ACCESS_FINE_LOCATION,
        Manifest.permission.ACCESS_COARSE_LOCATION,
    )

    /**
     * **الخطوة الثانية — «طوال الوقت».**
     *
     * **وتُطلب كإذن لا بفتح الإعدادات**: النظام حينها يعرض صفحته هو،
     * وفيها **«السماح طوال الوقت» ظاهرا للضغط.**
     *
     * **وفتح «معلومات التطبيق» بدلا منها خطأ** — يقع صاحبه على صفحة
     * فيها التخزين والبطاريّة والإشعارات، **وعليه أن يجد بنفسه:
     * الأذونات ثمّ الموقع ثمّ طوال الوقت.** ثلاث خطوات لا يكملها أحد.
     * (وقع في التجربة ٢٠٢٦-٠٨-١٢.)
     */
    val BACKGROUND = arrayOf(Manifest.permission.ACCESS_BACKGROUND_LOCATION)

    /**
     * **إذن الإشعارات — يُطلب مع الموقع لا وحده.**
     *
     * **ومن أندرويد ١٣ لا يظهر إشعار الخدمة بدونه**: فتعمل الخدمة صامتة،
     * **ولا يرى السائق أنّ ورديّته مفتوحة وأنّ موقعه يُرسل.** (قيس
     * ٢٠٢٦-٠٨-١٢: الخدمة تعمل والإذن مرفوض والإشعار غائب.)
     */
    val NOTIFICATIONS =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            arrayOf(Manifest.permission.POST_NOTIFICATIONS)
        } else {
            emptyArray()
        }

    /** ما يُطلب في الضغطة الأولى: الموقع والإشعار معا. */
    val FIRST_STEP = FOREGROUND + NOTIFICATIONS

    fun granted(context: Context): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) ==
            PackageManager.PERMISSION_GRANTED

    /**
     * **هل يُسمح بالموقع والتطبيق مغلق؟**
     *
     * **ودونه تعمل الخدمة ما دامت الشاشة مضاءة ثمّ تسكت** — فيبدو للمكتب
     * أنّ السائق واقف وهو يسير.
     */
    fun backgroundGranted(context: Context): Boolean =
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q) {
            granted(context)
        } else {
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.ACCESS_BACKGROUND_LOCATION,
            ) == PackageManager.PERMISSION_GRANTED
        }

    /** إذن الإشعارات — **شرط ظهور إشعار الخدمة من أندرويد ١٣.** */
    fun notificationsNeeded(context: Context): Boolean =
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS,
            ) != PackageManager.PERMISSION_GRANTED

    /** يفتح صفحة التطبيق في الإعدادات — **لمن رفض نهائيّا أو لطلب «طوال الوقت».** */
    fun openSettings(context: Context) {
        context.startActivity(
            Intent(
                Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                Uri.fromParts("package", context.packageName, null),
            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }
}
