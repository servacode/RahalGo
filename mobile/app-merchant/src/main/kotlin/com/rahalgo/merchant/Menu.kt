package com.rahalgo.merchant

import com.rahalgo.ui.DrawerItem

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أبوابُ درج المتجر — وما يُفتح مرّةً في الأسبوع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «سجلُّ الطلبات قسمٌ منفصلٌ كما بالويب ويكون
 *  داخل القائمة الجانبيّة»، ثمّ: «الإنذارات احذفها من هنا وستكون بقسمٍ
 *  مستقلٍّ بالقائمة الجانبيّة · وبالقائمة الجانبيّة يجب أن يكون قسمُ
 *  الشكاوي والبلاغات · وقسمُ التقارير أيضاً».)
 *
 * # ولماذا الدرجُ لا الشريطُ السفليّ
 *
 * **الشريطُ لما يُفتح كلَّ يوم** — والطلباتُ الجارية عشرين مرّةً في
 * اليوم، والأصنافُ حين ينفد شيء. **وهذه الأربعةُ تُفتح عند حادثة**:
 * زبونٌ يسأل عن طلبٍ مضى · إنذارٌ وصله · سائقٌ تأخّر عليه · آخرُ الشهر.
 *
 * **وخمسةُ تبويباتٍ حدُّ ما يُقرأ بلمحة** — والسادسُ يضيّقها كلَّها.
 *
 * # وترتيبُها بحسب الإلحاح
 *
 * **الإنذارُ أوّلاً** — **وهو الوحيدُ الذي يُحظر المتجرُ إن أُهمل.**
 * ثمّ السجلُّ، ثمّ بلاغاتُه، ثمّ التقارير.
 */
object MerchantItems {
    const val HISTORY = "History"
    const val WARNINGS = "Warnings"
    const val REPORTS = "Reports"
    const val SALES = "Sales"
}

val MERCHANT_ITEMS: List<DrawerItem> = listOf(
    // **ومن أُنذر يرى إنذارَه** — إشعارٌ يمرّ في الشريط يُقرأ مرّةً
    // ويُنسى، **ثمّ يُحظر المتجرُ ولم يعلم أنّ عليه شيئاً.**
    DrawerItem(
        MerchantItems.WARNINGS,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_warnings,
        com.rahalgo.ui.R.drawable.ic_warning,
    ),
    DrawerItem(
        MerchantItems.HISTORY,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_history,
        com.rahalgo.ui.R.drawable.ic_history,
    ),
    // **وبلاغاتُه هو لا شكاوى الزبائن عليه** — انظر `merchant_reports.go`.
    DrawerItem(
        MerchantItems.REPORTS,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_reports,
        com.rahalgo.ui.R.drawable.ic_chat,
    ),
    DrawerItem(
        MerchantItems.SALES,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_sales,
        com.rahalgo.ui.R.drawable.ic_chart,
    ),
)
