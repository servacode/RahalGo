package com.rahalgo.design

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.foundation.isSystemInDarkTheme
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

/**
 * **طقمُ مادّة مبنيٌّ من اللوحة** — لا مكتوبٌ بجانبها.
 *
 * **وطقمان يُكتبان بأرقامٍ متوازيةٍ يفترقان**: يُصحَّح لونٌ في اللوحة
 * **ويبقى قديماً في الأزرار** — والزرُّ من مادّة.
 */
private fun schemeOf(p: RahalPalette): ColorScheme {
    val base = if (p.dark) darkColorScheme() else lightColorScheme()
    // ══════════════════════════════════════════════════════════════════
    // **وأدوارُ الأسطح تُملأ كلُّها — لا الأربعةُ المشهورة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **قِيس على الجهاز**: أرضُ الصفحة كحليّةٌ (`#07283A`) **والشريطُ
    // السفليُّ أبيضُ فاتح** (`#F2F2F2`) — لأنّ `NavigationBar` في مادّة٣
    // لا يقرأ `surface` **إنّما `surfaceContainer`**، وهو دورٌ لم يُملأ
    // فبقي على افتراض المكتبة.
    //
    // **ودورٌ يُترك للمكتبة يخرج بلونٍ لا يعرفه أحد** — لا من اللوحة ولا
    // من العلامة، **فتقع رقعةٌ غريبةٌ في شاشةٍ مضبوطة.**
    return base.copy(
        primary = p.brand,
        onPrimary = p.onBrand,
        primaryContainer = p.brand,
        onPrimaryContainer = p.onBrand,
        secondary = p.accent,
        onSecondary = p.onBrand,
        secondaryContainer = p.field,
        onSecondaryContainer = p.ink,
        background = p.canvas,
        onBackground = p.ink,
        surface = p.canvas,
        onSurface = p.ink,
        surfaceVariant = p.field,
        onSurfaceVariant = p.inkMuted,
        // **وحاويّاتُ السطح هي أرضُ الأشرطة والقوائم والحوارات.**
        surfaceContainerLowest = p.canvas,
        surfaceContainerLow = p.canvas,
        surfaceContainer = p.canvas,
        surfaceContainerHigh = p.surface,
        surfaceContainerHighest = p.surface,
        // **ولا صبغةَ ارتفاعٍ** — مادّة٣ تُلقي غلالةً من `surfaceTint` على
        // ما ارتفع، **فتُزيح لونَ اللوحة درجاتٍ لا تُقصد.**
        surfaceTint = androidx.compose.ui.graphics.Color.Transparent,
        error = p.danger,
        onError = p.onBrand,
        errorContainer = p.danger.copy(alpha = 0.15f),
        onErrorContainer = p.danger,
        outline = p.line,
        outlineVariant = p.line,
        inverseSurface = p.ink,
        inverseOnSurface = p.canvas,
        scrim = androidx.compose.ui.graphics.Color.Black,
    )
}

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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السمةُ — فاتحةٌ وغامقةٌ وقرارُ صاحبها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-١٣: «طبّق الثيم الفاتح والغامق بشكلٍ كاملٍ
 *  للتطبيق، بحيث يكون التصميمان متوافقين».)
 *
 * **والافتراضُ إعدادُ النظام** — من جعل جهازَه غامقاً أراد ذلك في كلّ
 * تطبيقاته، **وتطبيقٌ يبيّضُ الشاشةَ في يدِ من يقود ليلاً يُغلَق.**
 *
 * **وقرارُه هو يغلبه** — الجرسُ في الشريط يقول ما اختار، ويبقى بعد
 * إغلاق التطبيق.
 *
 * @param dark أغامقةٌ هي؟ **وفارغُه يعني «اسأل النظام».**
 */
@Composable
fun RahalGoTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val palette = if (dark) DarkPalette else LightPalette
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
    CompositionLocalProvider(
        LocalLayoutDirection provides LayoutDirection.Rtl,
        LocalPalette provides palette,
        // ══════════════════════════════════════════════════════════════
        // **ولونُ المحتوى الافتراضيُّ لونُنا لا لونُ Material**
        // ══════════════════════════════════════════════════════════════
        //
        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٥: «في كثير أماكن بالثيم الغامق غلط…
        //  النصّ غامق مو واضح».)
        //
        // # ولماذا وقع
        //
        // **و`Text` بلا لونٍ صريحٍ يقرأ `LocalContentColor`** — ولم
        // نكن نضعه، **فيأخذ افتراضَ Material المشتقَّ من مخطّطه لا من
        // لوحتنا.** فيخرج حبرٌ داكنٌ على أرضٍ داكنة.
        //
        // # ولا يُصلَح شاشةً شاشة
        //
        // **وأصلحتُ اثنَي عشرَ موضعاً في تطبيق الزبون بلونٍ صريح**
        // (٢٠٢٦-٠٨-٢٥) — **وتلك مطاردةٌ لا هندسة**: يبقى السائقُ
        // والمتجرُ والمندوبُ على العطب، **ويعود في كلّ شاشةٍ تُكتب غدا.**
        //
        // **والموضعُ الصحيحُ هنا**: سطرٌ واحدٌ يحكم التطبيقاتِ الأربعة.
        LocalContentColor provides palette.ink,
    ) {
        MaterialTheme(
            colorScheme = schemeOf(palette),
            typography = RahalGoTypography,
            // ══════════════════════════════════════════════════════════
            // **وأشكالُنا تدخل السمةَ — لا تُكتب في كلّ حقل**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «والحقولَ خلّيها بزاويةٍ دائريّة،
            //  أفضل واحترافيٌّ أكثر».)
            //
            // **وحقلُ Material افتراضُه `extraSmall` أي أربعُ نقاط** —
            // زاويةٌ تكاد تكون قائمة. **ومن دوّرها في شاشةٍ نسيها في
            // عشر** — وهو ما كان: حقولُ الدخول مربّعةٌ وحقولُ العنوان
            // مربّعة.
            //
            // **وحقنُها في السمة يبلغ كلَّ ما يرسمه Material**: الحقولُ
            // والقوائمُ المنسدلةُ والنوافذُ والألواحُ — **بلا أن تُلمس
            // شاشةٌ واحدة.**
            shapes = Shapes(
                extraSmall = Rahal.shape.md,
                small = Rahal.shape.md,
                medium = Rahal.shape.md,
                large = Rahal.shape.lg,
                extraLarge = Rahal.shape.lg,
            ),
            content = content,
        )
    }
}
