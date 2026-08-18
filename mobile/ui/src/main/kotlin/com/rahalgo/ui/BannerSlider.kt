package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
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
import androidx.compose.ui.draw.shadow
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
 * # ونسبةُ ٢:١ — قِيست ولم تُخمَّن
 *
 * **كانت ١٦:٥ نقلاً عن الويب، ولافتاتُ المالك ١٦٠٠×٨٠٠ أي ٢:١** —
 * قِيست على الخادم ٢٠٢٦-٠٨-١٨. **فكان الإطارُ الأعرضُ يقصّ نحو ثلث
 * ارتفاع صورته** من فوق ومن تحت، **وهو لا يعرف لماذا خرجت ناقصة.**
 *
 * **ونسبةٌ تُنقَل عن الويب بلا قياسٍ تُقصّ ما رفعه** — والصوابُ أن
 * تتبع الصورةَ لا الصفحةَ الأخرى.
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
        // ══════════════════════════════════════════════════════════════
        // **والارتفاعُ يُحسب لا يُترك للإطار**
        // ══════════════════════════════════════════════════════════════
        //
        // **الحافّةُ تأخذ من العرض** (`SIDE`) و`aspectRatio` على السلايدر
        // كلِّه يقيس العرضَ الكامل — **فتخرج الصفحةُ أقصرَ ممّا أُريد.**
        // فيُقاس عرضُ الصفحة أوّلاً ثمّ يُشتقّ الارتفاع.
        BoxWithConstraints(Modifier.fillMaxWidth()) {
            val pageWidth = maxWidth - SIDE * 2 - GAP
            HorizontalPager(
                state = pager,
                // ══════════════════════════════════════════════════════
                // **وحافّةُ التالية تُرى**
                // ══════════════════════════════════════════════════════
                //
                // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لو جعلتَ السلايدر أكبرَ
                //  قليلاً على التطبيق، لكن أفخم برأيي».)
                //
                // **ولافتةٌ تملأ العرضَ لا تقول إنّ خلفَها ثانية** —
                // فتُقرأ صورةً تتبدّل بلا سبب. **وطرفُ التالية هو ما
                // يقول «اسحب».**
                contentPadding = PaddingValues(horizontal = SIDE),
                pageSpacing = GAP,
                modifier = Modifier
                    .fillMaxWidth()
                    // **٢:١ لا ١٦:٥** — وهي نسبةُ ما يرفعه المالكُ
                    // فعلاً: قِيس ٢٠٢٦-٠٨-١٨ أنّ لافتاته ١٦٠٠×٨٠٠،
                    // **والإطارُ الأعرضُ كان يقصّ ثلثَ ارتفاعها.**
                    .height(pageWidth / 2f),
            ) { page ->
                val b = items[page]
                Box(
                    Modifier
                        .fillMaxSize()
                        // **وظلٌّ خفيفٌ يرفعها عن الورقة** — **ولافتةٌ
                        // ملتصقةٌ بالأرض تُقرأ جزءاً منها لا بطاقةً
                        // فوقها.**
                        .shadow(6.dp, Rahal.shape.lg)
                        .clip(Rahal.shape.lg)
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

/** **ما تُفسحه اللافتةُ يميناً ويساراً** — وفيه تُرى حافّةُ التالية. */
private val SIDE = 22.dp

/** **الفاصلُ بين لافتتين** — **وملتصقتان تُقرآن صورةً واحدةً مقطوعة.** */
private val GAP = 10.dp
