package com.rahalgo.ui

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.shared.model.parseBlocks

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحاتُ المنصّة — نصٌّ واحدٌ يقرؤه الويبُ والتطبيق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «التطبيقُ والويبُ نفسُ النموذج والتسميات
 *  والشكل والأفعال والأسماء وكلّ شيء».)
 *
 * # لماذا النصُّ من المحرّك لا من `strings.xml`
 *
 * **وثيقةٌ قانونيّةٌ في نسختين تُعدَّل إحداهما وتشيخ الأخرى** — **فمن
 * قرأ الشروطَ في التطبيق وافق على غير ما في الموقع.** وهي أخطرُ صور
 * التكرار: **ليست خطأً في الشاشة، إنّما خلافٌ في عقد.**
 *
 * **ويُبدَّل من اللوحة بلا نشرِ إصدار** — ولو كُتب في التطبيق **لَاحتاج
 * تعديلُ سطرٍ فيه إصداراً في غوغل بلاي** ينتظر مراجعتَه أيّاما.
 *
 * # وشكلٌ واحدٌ لأربع صفحات
 *
 * **التعليماتُ ومن نحن والشروطُ والخصوصيّةُ نصٌّ ذو كتل** — وأربعُ
 * شاشاتٍ لأربعة نصوصٍ أربعُ نسخٍ من الشيء نفسِه.
 */
@Composable
fun PlatformPage(
    vm: PagesViewModel,
    item: DrawerItem,
    /**
     * **دورُ من يقرأ** — ولكلٍّ تعليماتُه.
     *
     * **وثلاثةُ نصوصٍ في المحرّك لا واحد**: السائقُ له ورديّةٌ
     * وصندوق، والزبونُ له سلّةٌ وعنوان، **والمندوبُ له عملاءُ
     * وعمولة.**
     *
     * **ورايةٌ ثنائيّةٌ لثلاثة أدوارٍ تُجبر الثالثَ على أن يكون
     * أحدَهما** — **وقِيس (٢٠٢٦-٠٨-١٤) أنّ المندوبَ كان يقرأ «كيف
     * أبدأ ورديّتي؟».**
     */
    role: HelpRole = HelpRole.Customer,
) {
    val contact = vm.contact
    if (contact == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }

    val text = when (item.key) {
        PlatformPages.HELP -> when (role) {
            HelpRole.Driver -> contact.driverHelpText
            HelpRole.Rep -> contact.repHelpText
            HelpRole.Merchant -> contact.merchantHelpText
            HelpRole.Customer -> contact.helpText
        }
        PlatformPages.ABOUT -> contact.aboutText
        PlatformPages.TERMS -> contact.termsText
        PlatformPages.PRIVACY -> contact.privacyText
        else -> ""
    }
    // **والقوالبُ تُملأ هنا كما تُملأ في الويب** — ولا يُترك «{name}»
    // ظاهراً لقارئ.
    val blocks = parseBlocks(
        text = text,
        name = contact.legalName,
        phone = contact.supportPhone,
        address = contact.address,
    )

    Screen {
        ScreenTitle(stringResource(item.label), stringResource(pageHint(item.key)))
        if (blocks.isEmpty()) {
            Empty(stringResource(R.string.page_empty))
            return@Screen
        }
        blocks.forEach { b ->
            Spacer(Modifier.height(10.dp))
            Card {
                if (b.head.isNotEmpty()) {
                    Text(
                        text = b.head,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Spacer(Modifier.height(6.dp))
                }
                // **وسطرٌ لكلّ معنًى** — فقرةٌ من عشرة أسطرٍ لا تُقرأ.
                b.body.forEachIndexed { i, p ->
                    if (i > 0) Spacer(Modifier.height(6.dp))
                    Text(p, style = MaterialTheme.typography.bodyMedium)
                }
            }
        }
    }
}

/** **سطرُ الشرح تحت العنوان** — يقول ما هذه الصفحة لا ما اسمُها. */
private fun pageHint(key: String): Int = when (key) {
    PlatformPages.ABOUT -> R.string.page_about_hint
    PlatformPages.TERMS -> R.string.page_terms_hint
    PlatformPages.PRIVACY -> R.string.page_privacy_hint
    else -> R.string.page_help_hint
}

/** **أدوارُ التعليمات** — ولكلٍّ نصُّه في المحرّك. */
enum class HelpRole { Customer, Driver, Rep, Merchant }
