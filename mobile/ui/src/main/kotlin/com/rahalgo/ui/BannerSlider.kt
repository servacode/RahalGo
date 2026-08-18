package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import kotlinx.coroutines.delay

/**
 * **لافتةٌ في السلايدر** — ما يكفي لرسمها وفتحها.
 *
 * **ولا يُمرَّر نموذجُ الشبكة** — العدّةُ المشتركةُ لا تعرف `CustomerApi`،
 * **وقطعةٌ ترتبط بعقدِ نداءٍ تُجرّ إلى كلّ ما يعرفه.**
 */
data class BannerSlide(
    val id: String,
    val title: String,
    val imageUrl: String?,
    /** **وجهةُ الضغطة** — وفارغُها لافتةٌ تُرى ولا تُفتح. */
    val target: String = "",
)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلايدرُ اللافتات — لا شريطٌ يُسحب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «سلايدر صفحة التسوّق مثل سلايدر الصفحة
 *
 *	الرئيسيّة، ولكن يجب أن تُطبَّق على الجوّال — على التطبيق».)
 *
 * # ولماذا سلايدرٌ لا شريط
 *
 * **كان في الويب شريطاً أفقيّاً يُسحب، ومن لا يسحب لا يرى إلّا الأولى**
 * — فالثانيةُ والثالثةُ تُنشَران ولا يراهما أحد. (قرارُ المالك
 * ٢٠٢٦-٠٨-٠٥، وصار سلايدراً هناك.) **والتطبيقُ بقي بلا لافتاتٍ
 * أصلاً** — يستقبلها من المحرّك ولا يرسمها.
 *
 * # والمهلةُ من لوحة الإدارة
 *
 * **`shop.banner_auto` و`shop.banner_seconds`** — تصلان مع الصفحة
 * (`/public/home`). **ورقمٌ مكتوبٌ هنا يجعل المالكَ يطلب تعديلَ
 * الشيفرةِ ليُبطئ لافتة.**
 *
 * # ونسبةُ ١٦:٥ كما في الويب
 *
 * **ونسختان بنسبتين تجعلان اللافتةَ الواحدةَ تُقصّ في جهازٍ وتكتمل في
 * آخر** — والمالكُ يرفع صورةً واحدةً للاثنين.
 *
 * **والنسبةُ محجوزةٌ قبل وصول الصورة** — فلا تقفز الشاشةُ حين تصل.
 *
 * # ولا تدور وهو يقرأ
 *
 * **الدورانُ يتوقّف لحظةَ يلمسها** — `isScrollInProgress`: **لافتةٌ
 * تنزلق من تحت إصبعه تُقرأ عطبا.**
 */
@Composable
fun BannerSlider(
    items: List<BannerSlide>,
    modifier: Modifier = Modifier,
    auto: Boolean = true,
    everyMs: Int = 5_000,
    onOpen: (BannerSlide) -> Unit = {},
) {
    if (items.isEmpty()) return

    val pager = rememberPagerState { items.size }

    // **ولا مؤقّتَ للافتةٍ واحدة** — لا شيءَ ينتقل إليه، **ومؤقّتٌ يعمل
    // بلا أثرٍ يستهلك ولا يُرى.**
    if (auto && items.size > 1 && everyMs > 0) {
        LaunchedEffect(pager.currentPage, pager.isScrollInProgress, everyMs, items.size) {
            if (pager.isScrollInProgress) return@LaunchedEffect
            delay(everyMs.toLong())
            pager.animateScrollToPage((pager.currentPage + 1) % items.size)
        }
    }

    Column(modifier.fillMaxWidth()) {
        HorizontalPager(
            state = pager,
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 5f)
                .clip(Rahal.shape.md),
        ) { page ->
            val b = items[page]
            Box(
                Modifier
                    .fillMaxSize()
                    .then(
                        if (b.target.isNotEmpty()) {
                            Modifier.clickable { onOpen(b) }
                        } else {
                            Modifier
                        },
                    ),
            ) {
                RemoteImage(
                    url = b.imageUrl,
                    name = b.title,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }

        // **ونقاطٌ تقول كم بقي** — **وسلايدرٌ بلا نقاطٍ يُظنّ صورةً
        // واحدةً تتبدّل بلا سبب.**
        //
        // **ولا تظهر للواحدة** — نقطةٌ وحيدةٌ تقول «لا شيءَ غيرها»
        // فتُقرأ زخرفة.
        if (items.size > 1) {
            Spacer(Modifier.height(8.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                for (i in items.indices) {
                    val on = i == pager.currentPage
                    Box(
                        Modifier
                            .padding(horizontal = 3.dp)
                            .size(width = if (on) 18.dp else 6.dp, height = 6.dp)
                            .clip(Rahal.shape.pill)
                            .background(
                                if (on) {
                                    Rahal.colors.brand
                                } else {
                                    Rahal.colors.inkMuted.copy(alpha = 0.30f)
                                },
                            ),
                    )
                }
            }
        }
    }
}
