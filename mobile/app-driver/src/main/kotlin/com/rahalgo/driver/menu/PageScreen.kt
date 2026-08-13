package com.rahalgo.driver.menu

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.Card
import com.rahalgo.driver.ui.Empty
import com.rahalgo.driver.ui.LoadState
import com.rahalgo.driver.ui.Screen
import com.rahalgo.driver.ui.ScreenTitle
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
fun PageScreen(vm: SectionsViewModel, item: MenuItem) {
    val contact = vm.contact
    if (contact == null) {
        LoadState(vm.busy, vm.error) { vm.open(item, force = true) }
        return
    }

    val text = when (item) {
        MenuItem.Help -> contact.driverHelpText
        MenuItem.About -> contact.aboutText
        MenuItem.Terms -> contact.termsText
        MenuItem.Privacy -> contact.privacyText
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
        ScreenTitle(stringResource(item.label), stringResource(pageHint(item)))
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
private fun pageHint(item: MenuItem): Int = when (item) {
    MenuItem.Help -> R.string.page_help_hint
    MenuItem.About -> R.string.page_about_hint
    MenuItem.Terms -> R.string.page_terms_hint
    MenuItem.Privacy -> R.string.page_privacy_hint
    else -> R.string.page_help_hint
}
