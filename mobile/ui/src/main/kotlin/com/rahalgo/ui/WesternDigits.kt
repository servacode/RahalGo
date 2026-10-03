package com.rahalgo.ui

import android.content.Context
import android.content.res.Configuration
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأرقامُ بالأجنبيّة (0123456789) في كلّ مكانٍ — مركزيّاً** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **قِيس على جهازه**: «الطلب #١٣٦٧ لم يعد معك» — `%1$d` في نصٍّ عربيٍّ يُنسَّق بلغة الجهاز،
 * **فتخرج أرقامٌ هنديّةٌ في سطرٍ وأجنبيّةٌ في جاره** (`money()` يكتبها بيده).
 *
 * **والعلاجُ في اللغة لا في كلّ نصّ**: العربيّةُ نفسُها بنظام أرقامٍ لاتينيّ
 * (`ar-u-nu-latn`) — **فكلُّ `getString` و`format` و`stringResource` يكتب 0123456789**
 * بلا أن يُمسّ نصٌّ واحد، **ونصٌّ جديدٌ غداً يرثها بلا أن يعرف.**
 *
 * **ويُركَّب في ثلاثة مواضع** في كلّ تطبيق: `Application` (اللغةُ الافتراضيّةُ لـ`format`)
 * و`attachBaseContext` للنشاط والخدمة (موارده هو). **وإضافةٌ لا تبديل** — تطبيقٌ لم
 * يركّبها يبقى كما كان.
 */
object WesternDigits {

    /** **اللغةُ نفسُها بأرقامٍ لاتينيّة** — `ar` ⇒ `ar-u-nu-latn`. */
    fun locale(base: Locale): Locale =
        Locale.Builder().setLocale(base).setUnicodeLocaleKeyword("nu", "latn").build()

    /** **سياقٌ موارده بأرقامٍ لاتينيّة** — لـ`attachBaseContext`. */
    fun wrap(base: Context): Context {
        val config = Configuration(base.resources.configuration)
        config.setLocale(locale(config.locales[0]))
        return base.createConfigurationContext(config)
    }

    /** **اللغةُ الافتراضيّةُ للعمليّة** — يقرؤها `"%d".format(...)` وما شابهه. */
    fun applyDefault() {
        Locale.setDefault(locale(Locale.getDefault()))
    }
}
