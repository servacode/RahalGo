package com.rahalgo.design.intro

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandCanvas
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.design.R
import com.rahalgo.design.TaglineStyle

/**
 * ══════════════════════════════════════════════════════════════════════
 * **افتتاح رحّال غو — الشعار مقفول، والحركة حوله**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شرط المالك ٢٠٢٦-٠٨-١١: «اللوغو صورة واحدة مقفولة — لا يُفكَّك ولا
 *  يُعاد رسمه ولا تُبدَّل عناصره».)
 *
 * # ولماذا سقطت المحاولة الأولى
 *
 * **كانت تفكّك الشعار وتحرّك أجزاءه.** وشُوهدت على جهاز حقيقي فلم تُقرأ:
 *
 * - **الشعار كثيف** — أربعة عناصر متداخلة. وبحجم شاشة، كل عنصر عشرون
 *   نقطة، **والدرّاجة أصغر من ظفر.** فحركتها لا تُرى أصلا.
 * - **وتفكيك شعار يُقرأ مكسورا** لا «يتكوّن» — العين تعرف الشكل الكامل،
 *   وكل نقص فيه عطب.
 * - **وخمس مراحل في ١٧٠٠ملّي = ٣٤٠ لكل واحدة** — تحت عتبة ما تلتقطه
 *   العين حركةً، فتُقرأ رفّة.
 *
 * # فالحركة صارت حول الشعار لا فيه
 *
 * **الشعار يظهر كاملا ويبقى كاملا.** والحركة الظاهرة **موجة تخرج من
 * علامة الموقع وتملأ الشاشة** — كما تنبض نقطة على خريطة.
 *
 * **وهي معنى العلامة نفسه**: علامة موقع تنبض تقول «حُدّد موقعك». ليست
 * زخرفة فوق الشعار — **هي شغل الشعار.**
 *
 * **ومقاسها الشاشة كلّها** — لا عشرون نقطة. فتُرى ولو من بعيد.
 *
 * # المدد — ٢٥٠٠ملّي
 *
 * ```
 *    0 →  450   الشعار يظهر كاملا: شفافية ٠→١ وحجم ٩٢٪→١٠٠٪
 *  450 → 1150   موجة أولى من علامة الموقع
 *  850 → 1550   موجة ثانية — متداخلة، فتُقرأ نبضا متّصلا لا نبضتين
 * 1700 → 2100   العبارة تظهر وترتفع ثماني نقاط
 * 2100 → 2500   ثبات
 * ```
 *
 * **وأطول من السابقة بثمانمئة** (طلب المالك: «خلّي الزمن أطول شوي»)،
 * **ومع ذلك تُقرأ أهدأ** لأنها حركتان لا خمس.
 */

/** مدّة الحركة كاملة. */
const val IntroDurationMs = 3000

/** المدّة حين تكون حركات النظام مطفأة — **تُعرض النتيجة ولا يُنتظر.** */
const val IntroReducedMs = 260

// **منحنى الظهور** — سريع ثمّ يهدأ، فيُقرأ «استقرارا» لا «اندفاعا».
private val Settle = CubicBezierEasing(0.16f, 1f, 0.3f, 1f)
private val Smooth = FastOutSlowInEasing

/**
 * موضع علامة الموقع من لوحة الشعار (١٠٢٤) — **مقيس من ملفّ الطبقة.**
 *
 * **ومنه تخرج الموجة** — ولو خرجت من مركز الشعار لبدت دائرة زينة،
 * **لا نبضة موقع.**
 */
private const val PIN_X = 456f / 1024f
private const val PIN_Y = 390f / 1024f

/**
 * نسبة عرض **صندوق** الشعار من عرض الشاشة.
 *
 * **والصندوق أوسع من الرسم**: لوحة الملفّ ١٠٢٤ ومحتواها يشغل ٦١٪ منها
 * عرضا (من ٢١٥ إلى ٨٤٣ — مقيس). **فصندوق بنصف الشاشة يعطي رسما بثلثها.**
 *
 * **وضُبطت بالقياس على الجهاز لا بالحساب**: ٠٫٤٤ أعطت رسما بـ٣٦٪ من
 * عرض الشاشة (مقيس بألوان الشعار وحدها). فرُفعت إلى ٠٫٥٦ ليبلغ نحو ٤٦٪ —
 * وهو ما طلبه المالك بعد أن رآه صغيرا.
 */
private const val LOGO_WIDTH_RATIO = 0.56f

/** كم يعلو الشعار عن مركز الشاشة — ليتّسع تحته للعبارة. */
private const val LOGO_LIFT_RATIO = 0.10f

@Composable
fun BrandIntro(
    modifier: Modifier = Modifier,
    tagline: String,
    reduceMotion: Boolean = false,
    onFinished: () -> Unit = {},
) {
    // **قيمة واحدة لكل ما يتحرّك** — لا مؤقّتات متفرّقة تنجرف كلٌّ منها
    // بمقدار مختلف على جهاز بطيء، **فتتفكّك الحركة ولا يُعرف السبب.**
    val t = remember { Animatable(0f) }

    LaunchedEffect(reduceMotion) {
        if (reduceMotion) {
            t.snapTo(1f)
            kotlinx.coroutines.delay(IntroReducedMs.toLong())
        } else {
            t.animateTo(1f, tween(IntroDurationMs, easing = LinearEasing))
        }
        onFinished()
    }

    val p = t.value
    val appear = stage(p, 0, 400)
    // **موجات متتالية لا متداخلة** (طلب المالك ٢٠٢٦-٠٨-١١: «موجة صغيرة
    // تكبر وتطلع الثانية بعدها والثالثة بعدها»). **وستّمئة لكل واحدة**
    // — والمتداخلة السريعة تُقرأ اضطرابا لا نبضا.
    val ripple1 = stage(p, 400, 1000)
    val ripple2 = stage(p, 1000, 1600)
    val ripple3 = stage(p, 1600, 2200)
    val text = stage(p, 2300, 2700)

    val logo = painterResource(R.drawable.intro_logo)

    BoxWithConstraints(
        modifier = modifier
            .fillMaxSize()
            .background(BrandCanvas),
        contentAlignment = Alignment.Center,
    ) {
        val logoSize = maxWidth * LOGO_WIDTH_RATIO
        val logoPxWidth = with(androidx.compose.ui.platform.LocalDensity.current) { logoSize.toPx() }

        // ══════════════════════════════════════════════════════════════
        // **الموجات — خلف الشعار وبمقاس الشاشة**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا تُرسم داخل صندوق الشعار**: كانت ستُقصّ عند حافّته فتبدو
        // دائرة صغيرة. **ومقاسها الشاشة كلّها هو ما يجعلها تُرى.**
        Canvas(Modifier.fillMaxSize()) {
            // موضع العلامة على الشاشة — مركزُ الشاشة، ثمّ رفعُ الشعار،
            // ثمّ إزاحةُ العلامة داخل لوحته.
            val origin = Offset(
                x = size.width / 2f + (PIN_X - 0.5f) * logoPxWidth,
                y = size.height / 2f - logoPxWidth * LOGO_LIFT_RATIO +
                    (PIN_Y - 0.5f) * logoPxWidth,
            )
            // ══════════════════════════════════════════════════════
            // **ومداها ثلاثة أرباع العرض لا قطر الشاشة**
            // ══════════════════════════════════════════════════════
            //
            // **كان قطر الشاشة (٢٦٣٢ بكسلا) فلم تُرَ الموجة**: عند منتصف
            // عمرها يبلغ نصف قطرها ١٣٨٠ — **أي ضعف نصف عرض الشاشة**،
            // فتخرج حوافّها عن الإطار ولا يبقى في الشاشة إلّا قوسان
            // باهتان في أعلاها وأسفلها.
            //
            // **وثلاثة أرباع العرض تُبقيها داخل الإطار أكثر عمرها**، ثمّ
            // تخرج وهي تتلاشى — وهو ما يُقرأ موجةً تمضي.
            val maxR = size.width * 0.75f
            val startR = logoPxWidth * 0.18f

            drawRipple(ripple1, origin, startR, maxR)
            drawRipple(ripple2, origin, startR, maxR)
            drawRipple(ripple3, origin, startR, maxR)
        }

        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
            modifier = Modifier.offset(y = -(logoSize * LOGO_LIFT_RATIO)),
        ) {
            Image(
                painter = logo,
                contentDescription = null,
                modifier = Modifier
                    .size(logoSize)
                    .alpha(appear)
                    // **الشعار كما هو — لا يُمسّ إلّا حجمُه وشفافيّتُه.**
                    //
                    // ══════════════════════════════════════════════
                    // **وينبض مع كل موجة — نبضة تُرى**
                    // ══════════════════════════════════════════════
                    //
                    // (طلب المالك: «النبض مو واضح — لازم النبض يخلّي
                    //  اللوغو يكبر بكل نبضة».)
                    //
                    // **كانت ١٫٥٪ فلم تُرَ**: على شعار عرضه أربعمئة بكسل
                    // تعني ستّة بكسلات — **تحت عتبة ما تلتقطه العين.**
                    //
                    // **وسبعة بالمئة تُرى ولا تُقرأ ارتدادا**: ثمانية
                    // وعشرون بكسلا تظهر وتعود في ستّمئة ملّي.
                    .scale(
                        0.92f + 0.08f * Settle.transform(appear) +
                            0.07f * (breath(ripple1) + breath(ripple2) + breath(ripple3))
                    ),
            )
            Spacer(Modifier.height(26.dp))
            Text(
                text = tagline,
                style = TaglineStyle,
                color = InkMuted,
                modifier = Modifier
                    .alpha(Smooth.transform(text))
                    // **ترتفع ثماني نقاط وهي تظهر** — حركة تقول «جاءت من
                    // مكان»، لا «ومضت».
                    .offset(y = (8f * (1f - Smooth.transform(text))).dp),
            )
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
//  أدوات صغيرة
// ══════════════════════════════════════════════════════════════════════

/** نسبة تقدّم مرحلة من زمن الحركة الكلّيّ — من ٠ إلى ١. */
private fun stage(progress: Float, fromMs: Int, toMs: Int): Float {
    val now = progress * IntroDurationMs
    if (now <= fromMs) return 0f
    if (now >= toMs) return 1f
    return (now - fromMs) / (toMs - fromMs)
}

/**
 * موجة واحدة — **تتّسع وتخفت وتنحف.**
 *
 * **والثلاثة معا لا الاتّساع وحده**: دائرة تتّسع بسُمك ثابت تُقرأ حلقة
 * تكبر، **وخفوتها مع نحولها هو ما يجعلها تُقرأ موجة تتلاشى في الفضاء.**
 */
private fun DrawScope.drawRipple(
    progress: Float,
    origin: Offset,
    startRadius: Float,
    maxRadius: Float,
) {
    if (progress <= 0f || progress >= 1f) return
    // **وتتّسع بانتظام لا متسارعة** — الموجة المتسارعة تمرّ قبل أن تُتابَع
    // بالعين. (طلب المالك: «بطيئة مشان تكون واضحة».)
    val eased = progress
    // ══════════════════════════════════════════════════════════════════
    // **وتظهر بسرعة ثمّ تخفت ببطء**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان الخفوت تربيعيّا فلم تُرَ الموجة أصلا**: عند منتصف عمرها كانت
    // شفافيّتها ٧٪، وعند ثلاثة أرباعه ٢٪. **فقُيست على الجهاز فلم تظهر.**
    //
    // **فصارت تبلغ ذروتها في أوّل خُمس عمرها** ثمّ تنحدر خطّيّا — فتبقى
    // مرئيّة أكثر عمرها، **وهو المطلوب: حركة تُرى لا حركة موجودة.**
    val fade = if (progress < 0.18f) progress / 0.18f
    else 1f - (progress - 0.18f) / 0.82f
    drawCircle(
        color = BrandTeal,
        radius = startRadius + (maxRadius - startRadius) * eased,
        center = origin,
        alpha = 0.55f * fade,
        style = Stroke(width = 5f + 16f * (1f - progress)),
    )
}

/** نَفَس خفيف يرافق الموجة — يصعد ثمّ يعود. **ولا يتكرّر في الموجة الواحدة.** */
private fun breath(x: Float): Float =
    if (x <= 0f || x >= 1f) 0f else kotlin.math.sin(x * Math.PI).toFloat()
