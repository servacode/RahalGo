package com.rahalgo.design

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.material3.Typography
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ثيم رحّال غو — مصدر واحد لأربعة تطبيقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`GROUND-RULES.md` §1.2: ممنوع لون أو خط أو قياس مباشر في أي شاشة.)
 *
 * **والقيم من الموقع لا مخترعة**: لونا العلامة مأخوذان من ملفّات الشعار
 * نفسها (ملفّات SVG في `brand/intro`)، والخط هو `Tajawal` — وهو ما يستعمله
 * `theme.css` فعلا.
 *
 * # وتصحيح لتناقض في الوثائق
 *
 * `GROUND-RULES.md` كانت تقول «الخط المعتمد IBM Plex Sans Arabic»،
 * **و`theme.css` يقول `Tajawal` منذ زمن.** ووثيقةٌ تخالف الشيفرةَ أسوأُ
 * من لا وثيقة — فصُحّحت إلى الواقع.
 */

// ══════════════════════════════════════════════════════════════════════
//  ألوان العلامة
// ══════════════════════════════════════════════════════════════════════
//
// **مقروءةٌ من ملفّات الشعار حرفا بحرف** — لا مقاربة ولا تخمين.

/** سماويُّ العلامة — جسمُ الشعار والدرّاجة. */
val BrandTeal = Color(0xFF02678F)

/** برتقاليُّ العلامة — الطريق. */
val BrandOrange = Color(0xFFFE9501)

/**
 * أرضُ شاشة الافتتاح — **بيضاء لا بلون العلامة.**
 *
 * **والموقعُ داكنُ الثيم**، ولو أُخذت أرضُه هنا **لَذاب سماويُّ الشعار
 * فيها** وبقي البرتقاليُّ وحدَه معلَّقا — وهو نصفُ الشكل لا كلُّه.
 *
 * **وهو القرارُ نفسُه المتخذ في أيقونة التطبيق** (`brand/icon.json`)،
 * فتتّسق الأيقونةُ مع أول شاشة يراها المستخدم.
 */
val BrandCanvas = Color(0xFFFFFFFF)

/** نصٌّ ثانويٌّ على الأرض البيضاء — رماديٌّ مائلٌ إلى سماويّ العلامة. */
val InkMuted = Color(0xFF5A6B75)

// ══════════════════════════════════════════════════════════════════════
// **لونان يقولان حال المال بلا قراءة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرار المالك ٢٠٢٦-٠٨-١٢: «إذا الدفع نقدي — أي غير مقبوض — يجب أن يكون
//  المربّع أحمر، وإذا على محفظة معناها مدفوع أخضر».)
//
// **والسائق يقرأ البطاقة في ثانية وهو واقف**: **أحمر يعني اقبض**،
// وأخضر يعني سلّم وامضِ. **ومن خلط بينهما** طالب زبوناً دفع، أو مشى
// بلا نقد.
//
// **وهما غير لوني الهويّة** — الكحليّ والبرتقاليّ يقولان «هذه رحّال غو»،
// **وهذان يقولان حال هذا الطلب.**
val StateRed = Color(0xFFD64545)
val StateGreen = Color(0xFF1E9E5A)

/**
 * **أرضٌ داكنةٌ فوق الخريطة** — كحليُّ العلامة مطفأً.
 *
 * **ولوحُ الرحلة يقع على بلاطات المدينة**: بيضاءَ في الشمس ورماديّةً في
 * الظلّ، **وأرضٌ فاتحةٌ عليها تذوب.** والداكنُ يُقرأ على كلّ بلاطة.
 *
 * **وهو مشتقٌّ من `BrandTeal` لا لونٌ ثالث** — لوحةٌ واحدةٌ في التطبيقات
 * الأربعة، **ولونٌ يُكتب في شاشةٍ يفترق عن أخواته يوما.**
 */
val InkDeep = Color(0xFF07283A)

private val RahalGoColors = lightColorScheme(
    primary = BrandTeal,
    onPrimary = Color.White,
    secondary = BrandOrange,
    onSecondary = Color.White,
    background = BrandCanvas,
    onBackground = BrandTeal,
    surface = BrandCanvas,
    onSurface = BrandTeal,
)

// ══════════════════════════════════════════════════════════════════════
//  الخط
// ══════════════════════════════════════════════════════════════════════

val Tajawal = FontFamily(
    Font(R.font.tajawal_regular, FontWeight.Normal),
    Font(R.font.tajawal_medium, FontWeight.Medium),
    Font(R.font.tajawal_bold, FontWeight.Bold),
)

private val RahalGoTypography = Typography().run {
    // **والعائلةُ تُحقن في كل نمط** — ونمطٌ يُنسى يعود إلى خطّ النظام،
    // **فتظهر شاشةٌ بخطّين** ولا يُعرف السبب.
    copy(
        displayLarge = displayLarge.copy(fontFamily = Tajawal),
        displayMedium = displayMedium.copy(fontFamily = Tajawal),
        displaySmall = displaySmall.copy(fontFamily = Tajawal),
        headlineLarge = headlineLarge.copy(fontFamily = Tajawal),
        headlineMedium = headlineMedium.copy(fontFamily = Tajawal),
        headlineSmall = headlineSmall.copy(fontFamily = Tajawal),
        titleLarge = titleLarge.copy(fontFamily = Tajawal),
        titleMedium = titleMedium.copy(fontFamily = Tajawal),
        titleSmall = titleSmall.copy(fontFamily = Tajawal),
        bodyLarge = bodyLarge.copy(fontFamily = Tajawal),
        bodyMedium = bodyMedium.copy(fontFamily = Tajawal),
        bodySmall = bodySmall.copy(fontFamily = Tajawal),
        labelLarge = labelLarge.copy(fontFamily = Tajawal),
        labelMedium = labelMedium.copy(fontFamily = Tajawal),
        labelSmall = labelSmall.copy(fontFamily = Tajawal),
    )
}

/**
 * نمطُ عبارة الافتتاح.
 *
 * **وكبُرت من ١٧ إلى ٢٢** (طلب المالك ٢٠٢٦-٠٨-١١) — **وبقيت ثانويّةً
 * أمام الشعار**: الشعارُ نحو ٤٦٪ من عرض الشاشة، والعبارةُ سطرٌ تحته.
 *
 * **ووزنُها متوسّطٌ لا عريض**: عبارةٌ عريضةٌ بجانب شعارٍ كبيرٍ تُزاحمه.
 */
val TaglineStyle = TextStyle(
    fontFamily = Tajawal,
    fontWeight = FontWeight.Medium,
    fontSize = 22.sp,
    letterSpacing = 0.2.sp,
)

@Composable
fun RahalGoTheme(content: @Composable () -> Unit) {
    // ══════════════════════════════════════════════════════════════════
    // **الاتّجاه من اليمين — ولو كان الجهاز إنجليزيّا**
    // ══════════════════════════════════════════════════════════════════
    //
    // **أندرويد يأخذ الاتّجاه من لغة الجهاز لا من لغة التطبيق.** وجوّال
    // السائق قد يكون إنجليزيّا، **فيُرسم تطبيقٌ عربيٌّ كلُّه من اليسار**:
    // أوّلُ تبويبٍ في أقصى اليسار، وشريطُ خطّةِ السير يمشي عكسَ القراءة.
    //
    // **وقع وقيس ٢٠٢٦-٠٨-١٢**: «الرحلة» في اليسار وهي أوّلُ الأقسام،
    // والشريطُ من اليسار إلى اليمين — ورآهما المالك.
    //
    // **و`supportsRtl` في البيان لا يكفي**: هو يسمح ولا يفرض. **وهذا
    // تطبيقٌ عربيٌّ وحدَه**، فاتّجاهه من صفته لا من إعدادات جهازٍ لا
    // يملكه.
    CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Rtl) {
        MaterialTheme(
            colorScheme = RahalGoColors,
            typography = RahalGoTypography,
            content = content,
        )
    }
}
