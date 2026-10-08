package com.rahalgo.merchant

import com.rahalgo.merchant.noStoreMsg
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
    const val OFFERS = "Offers"
    const val HISTORY = "History"
    /** **سجلُّ التوصيلات** — كلُّ «لدي توصيلة» أرسلها المتجر (الخطوة ١٨). */
    const val DELIVERIES = "Deliveries"
    const val WARNINGS = "Warnings"
    const val REPORTS = "Reports"
    const val SALES = "Sales"
    /** **شرحُ التطبيق بالفيديو** — يفتح يوتيوب لا شاشة (طلبُ المالك ٢٠٢٦-١٠-٠٨). */
    const val TUTORIAL = "Tutorial"
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
    // ══════════════════════════════════════════════════════════════════
    // **وعروضُه يبنيها بنفسه** (`OF`، ٢٠٢٦-٠٩-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وكانت عند الإدارة وحدَها** — **فيتّصل ليُنزَل له خصمٌ على صنفٍ
    // اليومَ وحدَه.**
    DrawerItem(
        MerchantItems.OFFERS,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_offers,
        com.rahalgo.ui.R.drawable.ic_offer,
    ),
    DrawerItem(
        MerchantItems.HISTORY,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_history,
        com.rahalgo.ui.R.drawable.ic_history,
    ),
    // **وسجلُّ التوصيلات بجانب سجلّ الطلبات** — (نصُّ المالك ٢٠٢٦-١٠-٠١:
    // «بالقائمة الجانبيّة سجلّ التوصيلات… كلّ عمليّات التوصيل التي قمنا بها».)
    DrawerItem(
        MerchantItems.DELIVERIES,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_deliveries,
        com.rahalgo.ui.R.drawable.ic_moto,
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
    DrawerItem(
        MerchantItems.TUTORIAL,
        com.rahalgo.ui.R.string.menu_platform,
        R.string.menu_tutorial,
        com.rahalgo.ui.R.drawable.ic_play,
    ),
)

/**
 * **يفتح فيديو الشرح** — الرابطُ من المحرّك (`/public/platform`) فيُبدَّل من اللوحة
 * بلا نسخة، **وإن تعذّر سؤالُه فُتح الأصل.**
 */
suspend fun openTutorial(context: android.content.Context) {
    val url = runCatching { com.rahalgo.ui.AppCore.get().auth.platform().merchantTutorialUrl }
        .getOrNull()?.takeIf { it.isNotBlank() } ?: TUTORIAL_URL_DEFAULT
    runCatching {
        context.startActivity(
            android.content.Intent(android.content.Intent.ACTION_VIEW, android.net.Uri.parse(url))
                .addFlags(android.content.Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }
}

private const val TUTORIAL_URL_DEFAULT = "https://youtu.be/cIQ_dXUjZ_g"

/**
 * **«لا متجرَ مرتبطٌ بحسابك»** — من المعجم لا من الشيفرة.
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «راجع كلَّ النصوص… لا أريد كلماتٍ عامّة».)
 *
 * **وكانت مكتوبةً في خمسة ملفّاتٍ بيدٍ** — **وخمسُ نسخٍ من جملةٍ واحدةٍ
 * تتباعد**: تُصحَّح في واحدةٍ وتبقى في أربع.
 */
fun noStoreMsg(): String =
    com.rahalgo.ui.AppCore.get().app.getString(R.string.no_store)
