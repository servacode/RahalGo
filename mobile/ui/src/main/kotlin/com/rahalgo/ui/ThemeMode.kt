package com.rahalgo.ui

import android.content.Context
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السمةُ التي اختارها — تبقى بعد إغلاق التطبيق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-١٣: «طبّق الثيم الفاتح والغامق بشكلٍ كاملٍ
 *  للتطبيق».)
 *
 * # ثلاثُ حالاتٍ لا اثنتان
 *
 * **«اتبع النظام» ليست الفاتحة**: من جعل جهازَه يتبع الشمسَ أراد ذلك في
 * كلّ تطبيقاته — **وتطبيقٌ يثبت على الفاتحة يبيّض شاشةً في يدِ من يقود
 * ليلاً.**
 *
 * **وأوّلُ لمسةٍ تُخرجه من «النظام» إلى قرارٍ صريح** — ومن اختار مرّةً
 * لا يُبدَّل عليه اختيارُه بغروب الشمس.
 *
 * # ولماذا هنا لا في نموذج
 *
 * **تُقرأ قبل أن تُرسم أوّلُ بكسل** — ونموذجٌ يُنشأ داخل الشجرة يُقرأ
 * بعدها، **فتومض الشاشةُ بيضاءَ ثمّ تسودّ.**
 *
 * **وتفضيلاتٌ عاديّةٌ لا مشفّرة**: هذا اختيارُ شكلٍ لا سرّ، **والمشفّرةُ
 * تُفتح بمفتاحٍ من مخزن المفاتيح** فتؤخّر الإقلاع بلا سبب.
 */
enum class ThemeMode { System, Light, Dark }

object AppTheme {

    private const val FILE = "rahalgo.ui"
    private const val KEY = "theme"

    fun read(context: Context): ThemeMode =
        when (prefs(context).getString(KEY, null)) {
            "light" -> ThemeMode.Light
            "dark" -> ThemeMode.Dark
            else -> ThemeMode.System
        }

    fun write(context: Context, mode: ThemeMode) {
        prefs(context).edit().putString(
            KEY,
            when (mode) {
                ThemeMode.Light -> "light"
                ThemeMode.Dark -> "dark"
                ThemeMode.System -> ""
            },
        ).apply()
    }

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)
}

/**
 * **حالُ السمة في الشجرة** — تُقرأ من القرص مرّةً وتُكتب عند التبديل.
 *
 * **واللمسةُ تقلب المرئيَّ لا الحال**: من كان على «النظام» ونظامُه غامقٌ
 * فلمس **أراد الفاتحة** — لا أن يقفز إلى الغامقة التي هو فيها.
 */
class ThemeState(private val context: Context, initial: ThemeMode) {

    var mode by mutableStateOf(initial)
        private set

    @Composable
    fun isDark(): Boolean = when (mode) {
        ThemeMode.Dark -> true
        ThemeMode.Light -> false
        ThemeMode.System -> isSystemInDarkTheme()
    }

    fun toggle(nowDark: Boolean) {
        mode = if (nowDark) ThemeMode.Light else ThemeMode.Dark
        AppTheme.write(context, mode)
    }
}

@Composable
fun rememberTheme(context: Context): ThemeState =
    remember { ThemeState(context, AppTheme.read(context)) }
