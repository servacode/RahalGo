package com.rahalgo.design

import androidx.compose.runtime.Immutable
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.LineHeightStyle
import androidx.compose.ui.unit.sp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلّمُ الخطّ — أدوارٌ بأسماء، لا أحجامٌ بأرقام** (نظامُ التصميم الموحَّد ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مبدأُ المالك: «التصميم لازم يكون موحّد بكلّ التطبيقات، مشان إذا قرّرت يوماً طوّر التصميم
 *  يتطوّر معي بسهولة».)
 *
 * **كلُّ نصٍّ في التطبيقات الأربعة يأخذ دوراً من هنا** — `Rahal.type.body`، `Rahal.type.price`.
 * **ومن أراد أن يكبّر الخطَّ في المنصّة كلِّها غيّر رقماً واحداً في هذا الملفّ.**
 *
 * # القاعدتان
 *
 * - **المعلومةُ المهمّة لا تنزل تحت ١٣** (`caption`) — اسمٌ، سعرٌ، حالُ طلب، عنوان.
 * - **١٢ للبيانات الثانويّة وحدَها** (`meta` و`badge`) — وقتٌ، رقمُ طلب، وسمٌ صغير.
 *
 * # وارتفاعُ السطر عربيّ
 *
 * **الحرفُ العربيُّ أطولُ من اللاتينيّ** بنقاطه وحركاته وذيوله، **وسطرٌ بنسبة ١٫٢ يقصّ الياءَ
 * والنقاطَ أو يلصق السطرين.** فالنسبةُ هنا قرابةَ ١٫٥، **والسطرُ يتوسّط حرفَه** (`Center`) بلا قصّ.
 *
 * # وأدوارُ Material تشير إلى هذه نفسِها
 *
 * `MaterialTheme.typography.bodyMedium` **و`Rahal.type.body` نمطٌ واحد** — فالشاشاتُ القديمةُ
 * تتغيّر مع الجديدة يومَ يتغيّر رقم. (الخريطةُ في `Theme.kt`.)
 */
@Immutable
data class RahalType(
    /** **أكبرُ رقمٍ في الشاشة** — رصيدُ المحفظة، رمزُ التسليم، شاشةُ الترحيب. ٢٨ */
    val display: TextStyle = role(28, 40, FontWeight.Bold),
    /** **عنوانٌ بارز** — رقمٌ كبيرٌ في لوح، عنوانُ نافذةٍ مهمّة. ٢٤ */
    val headline: TextStyle = role(24, 34, FontWeight.Bold),
    /** **عنوانُ الشاشة** — رأسُ كلّ صفحة (`ScreenTitle`/`PageHeader`). ٢٢ */
    val screenTitle: TextStyle = role(22, 32, FontWeight.Bold),
    /** **عنوانُ قسمٍ داخل الشاشة** (`SectionHeader`). ١٧ */
    val sectionTitle: TextStyle = role(17, 26, FontWeight.Bold),
    /** **عنوانُ بطاقةٍ أو سطر** — اسمُ متجر، اسمُ صنف (`ListRow`). ١٥ */
    val cardTitle: TextStyle = role(15, 22, FontWeight.Bold),
    /** **نصٌّ طويلٌ مريح** — شرحٌ في صفحة، نصُّ رسالة. ١٦ */
    val bodyLarge: TextStyle = role(16, 26, FontWeight.Normal),
    /** **النصُّ الأساسيّ.** ١٥ */
    val body: TextStyle = role(15, 24, FontWeight.Normal),
    /** **نصٌّ أساسيٌّ مؤكَّد** — قيمةٌ بجانب اسمها. ١٥ */
    val bodyStrong: TextStyle = role(15, 24, FontWeight.Bold),
    /** **نصٌّ ثانويٌّ مهمّ** — العنوانُ تحت الاسم، شرحُ حقل. **أصغرُ ما تُكتب به معلومةٌ مهمّة.** ١٣ */
    val caption: TextStyle = role(13, 20, FontWeight.Normal),
    /** **بياناتٌ ثانويّة** — وقتٌ، تاريخٌ، رقمُ طلب. **ولا تُكتب بها معلومةٌ يُتّخذ عليها قرار.** ١٢ */
    val meta: TextStyle = role(12, 18, FontWeight.Normal),
    /** **وسمُ حقلٍ أو تبويب** — «رقم الهاتف» فوق حقله. ١٣ */
    val label: TextStyle = role(13, 18, FontWeight.Medium),
    /** **السعر** — في البطاقة والسلّة والفاتورة (`MoneyText`/`DualPrice`). ١٦ */
    val price: TextStyle = role(16, 24, FontWeight.Bold),
    /** **السعرُ الكبير** — الإجماليّ، المبلغُ المطلوب قبضُه. ٢٢ */
    val priceLarge: TextStyle = role(22, 30, FontWeight.Bold),
    /** **الشارةُ والرقاقة** (`StatusChip`/`Badge`). ١٢ */
    val badge: TextStyle = role(12, 16, FontWeight.Bold),
    /** **نصُّ الزرّ.** ١٥ */
    val button: TextStyle = role(15, 22, FontWeight.Bold),
) {
    companion object {
        /**
         * **أصغرُ ما ينكمش إليه نصٌّ يتصاغر ليسع** (`TextAutoSize`) — اسمُ صنفٍ طويلٍ في خانةٍ ضيّقة.
         * **ولا يُستعمل حجماً لنصٍّ ثابت.**
         */
        val AutoSizeFloor = 10.sp
    }
}

/** **السطرُ يتوسّط حرفَه ولا يُقصّ** — النقاطُ والحركاتُ فوق وتحت تبقى كاملة. */
private val ArabicLineHeight = LineHeightStyle(
    alignment = LineHeightStyle.Alignment.Center,
    trim = LineHeightStyle.Trim.None,
)

private fun role(size: Int, line: Int, weight: FontWeight) = TextStyle(
    fontFamily = Tajawal,
    fontWeight = weight,
    fontSize = size.sp,
    lineHeight = line.sp,
    letterSpacing = 0.sp,
    lineHeightStyle = ArabicLineHeight,
)

/** **ثوابتُ الخطّ** — لا تتبدّل بالسمة. */
val RahalTypeTokens = RahalType()
