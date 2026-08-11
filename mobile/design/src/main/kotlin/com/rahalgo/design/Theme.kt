package com.rahalgo.design

import androidx.compose.material3.MaterialTheme
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

/** نمطُ شعار الافتتاح — **متوسّطُ الوزن لا عريض**: العبارةُ تُقرأ ولا تصيح. */
val TaglineStyle = TextStyle(
    fontFamily = Tajawal,
    fontWeight = FontWeight.Medium,
    fontSize = 17.sp,
    letterSpacing = 0.2.sp,
)

@Composable
fun RahalGoTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = RahalGoColors,
        typography = RahalGoTypography,
        content = content,
    )
}
