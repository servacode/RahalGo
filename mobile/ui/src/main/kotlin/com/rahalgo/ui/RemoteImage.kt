package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import coil3.compose.AsyncImage
import coil3.compose.AsyncImagePainter
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صورةٌ من المحرّك — بابٌ واحدٌ لكلّ صورةٍ في التطبيقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا مكتبةٌ الآن ولم تكن
 *
 * **كان في التطبيق صورةٌ واحدةٌ صغيرة** (صورةُ الحساب)، **ومكتبةُ تحميلِ
 * صورٍ تبني ذاكرةً وقرصاً وخيوطاً لها وحدَها.** فكُتب قارئٌ بسيط،
 * **وكُتب معه أنّ اليومَ الذي تُعرض فيه صورُ الأصناف تُضاف `coil`.**
 *
 * **واليومُ سوقٌ كاملٌ يُتصفَّح بالصور** — والقارئُ البسيطُ بلا ذاكرةٍ
 * ولا قرص **يُنزّل الصورةَ نفسَها في كلّ تمريرةِ إصبع**: يستنزف حزمةَ
 * من يتصفّح وبطّاريّتَه، **ويومض الفراغُ في كلّ مرّة.**
 *
 * # وما يُعرض حتّى تصل
 *
 * **أرضٌ باهتةٌ بلون العلامة** — لا دائرةُ انتظارٍ تدور في كلّ بطاقة:
 * **عشرُ دوائرَ تدور في شبكةٍ واحدةٍ تُقرأ عطباً لا انتظارا.**
 *
 * # وما يُعرض إن لم تصل
 *
 * **حرفُ اسمها** — لا مربّعٌ رماديّ: **الرماديُّ يُقرأ «عطب»، والحرفُ
 * يُقرأ «لا صورةَ لهذا».**
 *
 * # ولماذا `AsyncImage` لا `SubcomposeAsyncImage`
 *
 * (قِيس ٢٠٢٦-٠٨-١٩ على جهاز المالك: **ربعُ الإطارات تتجاوز مهلتَها**
 *  في تمرير شاشة التسوّق — والمقبولُ دون العشرة.)
 *
 * **والتركيبُ الفرعيُّ يُعيد قياسَ كلّ صورةٍ في تركيبةٍ مستقلّة** —
 * وCoil نفسُها تكتب أنّه أبطأُ وتوصي بتجنّبه في القوائم. **وشبكةٌ فيها
 * عشراتُ البطاقات تدفع الثمنَ عشراتِ المرّات في كلّ تمريرة.**
 *
 * **وكانت خانةُ `loading` فارغةً تماماً** (`loading = {}`) — **فالثمنُ
 * يُدفع مقابلَ لا شيء**: الأرضُ الباهتةُ يرسمها الصندوقُ الحاوي لا
 * الخانة.
 *
 * **والحرفُ البديلُ يبقى** — يُرسم بحالٍ تُقرأ من `onState` بدل خانةٍ
 * تُركَّب فرعيّا.
 */
@Composable
fun RemoteImage(
    url: String?,
    name: String,
    modifier: Modifier = Modifier,
    contentScale: ContentScale = ContentScale.Crop,
) {
    Box(
        modifier.background(Rahal.colors.brand.copy(alpha = 0.10f)),
        contentAlignment = Alignment.Center,
    ) {
        if (url.isNullOrEmpty()) {
            Fallback(name)
            return@Box
        }
        // **والحالُ تُنسى مع تبدّل الرابط** — **وبطاقةٌ يُعاد استعمالُها
        // في شبكةٍ كسولةٍ ترث فشلَ سابقتها** فيظهر الحرفُ فوق صورةٍ
        // سليمة.
        var failed by remember(url) { mutableStateOf(false) }
        AsyncImage(
            model = url,
            contentDescription = null,
            contentScale = contentScale,
            modifier = Modifier.fillMaxSize(),
            onState = { failed = it is AsyncImagePainter.State.Error },
        )
        // **والأرضُ الباهتةُ هي الانتظار** — لا دائرةٌ تدور.
        if (failed) Fallback(name)
    }
}

@Composable
private fun Fallback(name: String) {
    Text(
        text = name.trim().take(1),
        color = Rahal.colors.brand,
        style = MaterialTheme.typography.titleMedium,
    )
}
