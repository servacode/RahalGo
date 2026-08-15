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
import coil3.compose.SubcomposeAsyncImage
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
        SubcomposeAsyncImage(
            model = url,
            contentDescription = null,
            contentScale = contentScale,
            modifier = Modifier.fillMaxSize(),
            // **والأرضُ الباهتةُ هي الانتظار** — لا دائرةٌ تدور.
            loading = {},
            error = { Fallback(name) },
        )
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
