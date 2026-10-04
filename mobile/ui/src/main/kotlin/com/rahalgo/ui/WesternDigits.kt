package com.rahalgo.ui

import android.content.Context
import android.content.res.Configuration
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأرقامُ لاتينيّةٌ (0123456789) في كلّ شاشة — من مكانٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الأرقام كلُّها بالأجنبيّة في كلّ مكان، مركزيّاً،
 *  وفاصلُ الآلاف «,» لا «٬».»)
 *
 * # من أين تأتي «٣» على جهازٍ عربيّ
 *
 * **كلُّ نصٍّ فيه `%d` يُنسَّق بلغة الجهاز** — `getString(R.string.x, n)` و
 * `pluralStringResource` و`String.format`: **فيخرج «٣ أصناف» و«محادثة الطلب
 * #٥١١»** وبجانبها «12,500 ل.س» من `money` المكتوبة باليد.
 *
 * # والعلاجُ في السياق لا في كلّ شاشة
 *
 * **`ar-u-nu-latn` عربيّةٌ بأرقامٍ لاتينيّة** — تبقى الكلماتُ والاتّجاهُ
 * عربيّةً **ويتبدّل نظامُ الأرقام وحدَه**. **ويُلفّ به سياقُ النشاط مرّةً**
 * (`attachBaseContext`) **ولغةُ العمليّة** (`Locale.setDefault`) — فلا تُمسّ شاشة.
 */
object WesternDigits {

    /** **اللغةُ نفسُها بأرقامٍ لاتينيّة** — `ar` تصير `ar-u-nu-latn`. */
    fun locale(base: Locale): Locale =
        Locale.Builder().setLocale(base).setUnicodeLocaleKeyword("nu", "latn").build()

    /**
     * **سياقُ النشاط بأرقامٍ لاتينيّة** — يُنادى من `attachBaseContext`.
     *
     * **وفي القاعدة لا في نافذة** — فلا يقع ما وقع في نافذة الوقت
     * (`BadTokenException`): السياقُ يُبنى قبل أن تُعلَّق عليه نافذة.
     */
    fun wrap(base: Context): Context {
        val config = Configuration(base.resources.configuration)
        config.setLocale(locale(config.locales[0]))
        return base.createConfigurationContext(config)
    }

    /** **لغةُ العمليّة كلِّها** — لِما يُنسَّق بلا سياق (`String.format`). */
    fun applyDefault() {
        Locale.setDefault(locale(Locale.getDefault()))
    }

    /**
     * **نصٌّ لم نصنعه** (خادمٌ أو حافظة) — أرقامُه الهنديّةُ والفارسيّةُ لاتينيّة،
     * و«٬» «,» و«٫» «.».
     */
    fun text(s: String): String {
        if (s.none { it in '\u0660'..'\u0669' || it in '\u06F0'..'\u06F9' || it == '\u066C' || it == '\u066B' }) return s
        val b = StringBuilder(s.length)
        for (c in s) {
            b.append(
                when (c) {
                    in '\u0660'..'\u0669' -> '0' + (c - '\u0660')
                    in '\u06F0'..'\u06F9' -> '0' + (c - '\u06F0')
                    '\u066C' -> ','
                    '\u066B' -> '.'
                    else -> c
                },
            )
        }
        return b.toString()
    }
}
