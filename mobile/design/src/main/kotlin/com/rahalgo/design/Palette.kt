package com.rahalgo.design

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ الألوان — أسماءٌ تُنادى، لا أرقامٌ تُكتب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ بتطبيق الثيم الفاتح والغامق بشكلٍ كاملٍ
 *  للتطبيق، بحيث يكون التصميمان متوافقين واحترافيّين… مع لمسة سحرٍ
 *  وإبداعٍ وأناقةٍ بالتصميم والألوان».)
 *
 * # ما قِيس قبل الكتابة
 *
 * **سبعةُ رماديّاتٍ فاتحةٍ مكتوبةٍ بأرقامها في خمس شاشات**: `#EFF4F6`
 * و`#EEF1F3` و`#F5F7F8` و`#EFF2F4` و`#F0F3F5` — **خمسةُ ألوانٍ لا يفرّق
 * بينها بصر**، كلٌّ اخترعته شاشةٌ لأنّه لم يكن لها اسمٌ تناديه.
 *
 * **وذلك ما يكسر السمةَ الغامقة**: لونٌ يُكتب في شاشةٍ **موضعٌ يُنسى يومَ
 * تُبدَّل الأرض**، فتبقى بقعةٌ بيضاءُ في شاشةٍ سوداء.
 *
 * # والفاتحةُ هي هي — لم تتبدّل
 *
 * **ما كان معروضاً على المالك ووافق عليه يبقى كما هو** — هذه اللوحةُ
 * تسمّي القائمَ لا تخترع غيرَه. **وخمسةُ الرماديّات صارت اسماً واحداً**،
 * وفرقُها بضعُ درجاتٍ لا تُرى.
 *
 * # والغامقةُ مشتقّةٌ لا مقلوبة
 *
 * **وقلبُ الألوان يُخرج شاشةً رماديّةً ميّتة**: أرضُها من كحليّ العلامة
 * نفسِه (`#07283A`) — **وهو اللونُ الذي يقع عليه لوحُ الرحلة فوق
 * الخريطة منذ اليوم الأوّل**، فالسمةُ الغامقةُ امتدادُ ما في التطبيق لا
 * وافدٌ عليه.
 *
 * **والسماويُّ يُرفع في الغامقة**: `#02678F` على أرضٍ داكنةٍ **يذوب
 * فيها** — فيُفتَّح إلى `#4FB8E8`. وهي القاعدةُ نفسُها التي جعلت أرضَ
 * شاشة الافتتاح بيضاءَ لا كحليّة.
 *
 * **والبرتقاليُّ يبقى** — يقع على الداكن والفاتح سواء، **وهو ما يجعل
 * السمتين تُقرآن منصّةً واحدة.**
 */
@Immutable
data class RahalPalette(
    /** **أرضُ الصفحة** — ما تحت كلّ شيء. */
    val canvas: Color,
    /** **أرضُ البطاقة** — تعزل ما فيها عمّا حولَه. */
    val surface: Color,
    /** **أرضٌ أخفض** — حقلُ إدخالٍ أو رقاقةٌ داخل بطاقة. */
    val field: Color,
    /** **الحدّ** — خطٌّ يفصل ولا يُرى إلّا إن طُلب. */
    val line: Color,
    /** **النصُّ الأوّل.** */
    val ink: Color,
    /** **النصُّ الثاني** — شرحٌ وتاريخٌ ووحدة. */
    val inkMuted: Color,
    /** **سماويُّ العلامة** — الفعلُ الأوّل وما هو مختار. */
    val brand: Color,
    /** **ما يُكتب على السماويّ.** */
    val onBrand: Color,
    /** **برتقاليُّ العلامة** — النقدُ والتنبيه. */
    val accent: Color,
    /** **تمام** — مدفوعٌ ومُسلَّمٌ ومكافأة. */
    val success: Color,
    /** **خطأ** — اقبضْ، وتعذّر، وعقوبة. */
    val danger: Color,
    /** **أرضُ تنبيهٍ خفيفة** — تحت التحذير لا خلفه. */
    val warnTint: Color,
    /** **لوحُ الرحلة فوق الخريطة** — داكنٌ ليُقرأ على كلّ بلاطة. */
    val panel: Color,
    /** **فقاعةُ الطرف الآخر** في الحديث. */
    val bubble: Color,
    /** **أهي الغامقة** — لِما لا يُقاس بلون: أيقونةُ شريط النظام. */
    val dark: Boolean,
)

/**
 * **الفاتحة — قِيست من الشاشات القائمة.**
 *
 * **ولا لونَ جديدٌ فيها**: `brand` و`accent` من ملفّات الشعار،
 * و`inkMuted` كما كان، **والرماديّاتُ الخمسُ صارت `surface` و`field`.**
 */
val LightPalette = RahalPalette(
    canvas = Color(0xFFFFFFFF),
    surface = Color(0xFFF5F7F8),
    field = Color(0xFFEEF1F3),
    line = Color(0xFFE3E8EB),
    // **والنصُّ الأوّل كحليُّ العلامة** — وهو ما كان (`onSurface`)، **لا
    // أسودَ**: أسودُ على أبيضَ يقطع، والكحليُّ يُقرأ ساعةً بلا إجهاد.
    ink = Color(0xFF02678F),
    inkMuted = Color(0xFF5A6B75),
    brand = Color(0xFF02678F),
    onBrand = Color(0xFFFFFFFF),
    accent = Color(0xFFFE9501),
    success = Color(0xFF1E9E5A),
    danger = Color(0xFFD64545),
    warnTint = Color(0xFFFFF4E5),
    panel = Color(0xFF07283A),
    bubble = Color(0xFFF0F3F5),
    dark = false,
)

/**
 * **الغامقة — من كحليّ العلامة لا من الأسود.**
 *
 * **وأسودُ خالصٌ يجعل الشاشةَ حفرة**: الحوافُّ تختفي فلا يُعرف أين تنتهي
 * البطاقةُ وأين تبدأ الأرض. **وثلاثُ درجاتٍ متقاربةٍ من الكحليّ** تبني
 * عمقاً يُرى بلا خطوط.
 *
 * **والأخضرُ والأحمرُ يُرفعان أيضا**: `#1E9E5A` على الداكن يُقرأ رماديّاً
 * غامقا، **ومعنى اللون يضيع.**
 */
val DarkPalette = RahalPalette(
    canvas = Color(0xFF07283A),
    surface = Color(0xFF0E3346),
    field = Color(0xFF154054),
    line = Color(0xFF1E5068),
    ink = Color(0xFFE9F2F7),
    inkMuted = Color(0xFF9DB4C0),
    brand = Color(0xFF4FB8E8),
    onBrand = Color(0xFF04222F),
    accent = Color(0xFFFFA726),
    success = Color(0xFF3ECB86),
    danger = Color(0xFFFF7070),
    // **والتنبيهُ لا يُضاء بالأصفر الفاتح على الداكن** — يذوب البياضُ
    // فيه، **فيُبنى من البرتقاليّ مطفأً.**
    warnTint = Color(0xFF3A2A12),
    panel = Color(0xFF061F2D),
    bubble = Color(0xFF17394B),
    dark = true,
)

/**
 * **اللوحةُ النافذة** — تُحقن من `RahalGoTheme`.
 *
 * **وساكنةٌ لا متبدّلة** (`staticCompositionLocalOf`): تبديلُ السمة
 * يُعيد بناءَ الشجرة كلِّها مرّةً واحدة، **وهو أرخصُ من تتبّعِ كلّ قارئٍ
 * للون في كلّ إطار.**
 */
val LocalPalette = staticCompositionLocalOf { LightPalette }

/** **بابُ الألوان في الشاشات** — `Rahal.colors.inkMuted`. */
object Rahal {
    val colors: RahalPalette
        @Composable @ReadOnlyComposable get() = LocalPalette.current
}
