package com.rahalgo.design.intro

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.wrapContentSize
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
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.design.BrandOrange
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
 *  150 →  550   دفعة أمامية ست نقاط ذهابا وإيابا       [مضاف]
 *  180 →  520   خط سرعة خلف الشعار                     [مضاف]
 *  400 → 1000   موجة أولى + نبضة
 * 1000 → 1600   موجة ثانية + نبضة
 * 1600 → 2200   موجة ثالثة + نبضة
 * 2100 → 2400   لمعة تمرّ فوق الشعار                    [مضاف]
 * 2250 → 2900   العبارة تُكتب تدريجيا وتُلوَّن
 * 2900 → 4900   ثبات — **ثانيتان كاملتان** قبل الانتقال
 * ```
 *
 * **والموجات متتالية لا متداخلة** (طلب المالك): موجة تكبر ثمّ التي بعدها.
 * **وتُقرأ أهدأ من خمس مراحل** وإن طالت — لأنها حركة واحدة تتكرّر.
 *
 * **والوقفة الأخيرة ثانيتان** (طلب المالك ٢٠٢٦-٠٨-١١): **لا شيء يتحرّك
 * فيها** — الشعار والعبارة ساكنان، فيُقرأ الأثر ويُحفظ قبل أن تُبدَّل
 * الشاشة.
 */

/** مدّة الحركة كاملة — **بما فيها وقفة النهاية.** */
const val IntroDurationMs = 4900

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
    /** الجزءُ الذي يُلوَّن ببرتقاليّ العلامة — اسمُ المنصّة. */
    brandWord: String = "",
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

    // ══════════════════════════════════════════════════════════════════
    // **طبقاتٌ جديدةٌ تعمل داخل الزمن القائم لا بعده**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلب المالك ٢٠٢٦-٠٨-١١: «أضف فوق الحركة الحالية، لا تستبدلها،
    //  ولا تضاعف المدّة».)
    //
    // **فالدفعةُ الأماميّةُ وخطُّ السرعة يقعان أثناء الظهور** (١٥٠–٥٥٠)،
    // **واللمعةُ بعد آخر موجةٍ وقبل العبارة** (٢١٠٠–٢٤٠٠). **ولم تُزَد
    // المدّةُ إلّا مئتي ملّي** لتتّسع الكتابةُ التدريجيّة.
    val push = stage(p, 150, 550)
    val trail = stage(p, 180, 520)
    val sweep = stage(p, 2100, 2400)
    val text = stage(p, 2250, 2900)

    val logo = painterResource(R.drawable.intro_logo)

    BoxWithConstraints(
        modifier = modifier
            .fillMaxSize()
            // **وأرضُ الافتتاح أرضُ السمة** — وبياضٌ ثابتٌ يومض في وجه
            // من فتح تطبيقَه ليلاً على سمةٍ غامقة، **فأوّلُ ما يراه
            // ومضةٌ تُغمض العين.**
            .background(Rahal.colors.canvas),
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
                    // ══════════════════════════════════════════════
                    // **دفعةٌ أماميّةٌ خفيفة — ست نقاط ذهابا وإيابا**
                    // ══════════════════════════════════════════════
                    //
                    // **باتّجاه السهم في الشعار** (يمينا)، **فتُقرأ
                    // تقدّما لا اهتزازا.** وتقع أثناء الظهور فلا تُطيل
                    // الزمن.
                    .offset(x = (6f * arc(push)).dp)
                    // **وخطُّ سرعةٍ خلفه** — طبقةٌ مستقلّةٌ تحته، **ولا
                    // ضبابَ على الشعار نفسِه**: الشعارُ يبقى حادّا مئةً
                    // بالمئة (شرط المالك).
                    .drawBehind { drawSpeedTrail(trail) }
                    .alpha(appear)
                    // **ولمعةٌ تمرّ فوقه مرّةً واحدة** — قناعٌ يقتصر على
                    // شكل الشعار (`SrcAtop`)، **فلا يُضيء البياضَ حوله**
                    // ولا يمسّ ألوانَه الأصليّة إلّا تفتيحا لحظيّا.
                    .drawWithContent {
                        drawContent()
                        drawSweep(sweep)
                    }
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
            Tagline(text = tagline, brand = brandWord, progress = text)
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

// ══════════════════════════════════════════════════════════════════════
//  الطبقات المضافة
// ══════════════════════════════════════════════════════════════════════

/**
 * **العبارة تُكتب تدريجيّا — كشفٌ لا حرفٌ حرفا.**
 *
 * (طلب المالك ٢٠٢٦-٠٨-١١: «تأثير أنها تُكتب كتابة وليس دفعة واحدة».)
 *
 * # ولماذا كشفٌ لا اقتطاعُ نصّ
 *
 * **العربيّةُ تُشكَّل حروفُها بحسب جيرانها**: «طلبك» مقتطعةً عند حرفين
 * تصير «طل» — **وشكلُ الطاء واللام يتبدّل**، فتُرى الكلمةُ تتراقص وهي
 * «تُكتب». وكلُّ مقطعٍ يُعيد قياسَ السطر فيقفز يمينا ويسارا.
 *
 * **فيُرسم النصُّ كاملا مرّةً واحدة، ويُكشف من يمينه إلى يساره** — وهو
 * اتّجاه الكتابة العربيّة. **فالشكلُ سليمٌ والعرضُ ثابتٌ من أوّل إطار.**
 *
 * # والتلوين
 *
 * (طلب المالك: «تلوين الكتابة بالأزرق ورحّال غو بالبرتقالي».)
 *
 * **واسمُ المنصّة يُمرَّر مفتاحا لا يُقتطع بالحساب** — اقتطاعُ آخرِ
 * كلمتين يكسر مع أوّل تبديلٍ للعبارة ولا شيءَ يُنبّه.
 */
@Composable
private fun Tagline(text: String, brand: String, progress: Float) {
    val eased = Smooth.transform(progress)
    val styled = buildAnnotatedString {
        val at = if (brand.isNotEmpty()) text.lastIndexOf(brand) else -1
        if (at < 0) {
            withStyle(SpanStyle(color = BrandTeal)) { append(text) }
        } else {
            withStyle(SpanStyle(color = BrandTeal)) { append(text.substring(0, at)) }
            withStyle(SpanStyle(color = BrandOrange)) { append(text.substring(at)) }
        }
    }
    Box(
        Modifier
            .wrapContentSize()
            .clipToBounds()
            // **وترتفع ثماني نقاط وهي تظهر** — حركةٌ تقول «جاءت من مكان».
            .offset(y = (8f * (1f - eased)).dp)
            .drawWithContent {
                // **يُكشف من اليمين** — مبدأ الكتابة العربيّة.
                clipRect(left = size.width * (1f - eased)) { this@drawWithContent.drawContent() }
            },
    ) {
        Text(text = styled, style = TaglineStyle)
    }
}

/**
 * **خطُّ سرعةٍ خلف الشعار — طبقةٌ مستقلّةٌ لا ضبابٌ عليه.**
 *
 * (شرط المالك: «لا تطبّق Blur على اللوغو نفسه، يبقى حادّا مئة بالمئة».)
 *
 * **وثلاثةُ خطوطٍ قصيرةٍ إلى يساره** — في اتّجاه معاكسٍ للدفعة، **فتُقرأ
 * أثرا تركه** لا زخرفةً بجانبه. **وتختفي تماما مع استقراره.**
 *
 * **وشفافيّتُها منخفضةٌ جدّا** — على أرضٍ بيضاء يصير كلُّ ما زاد لطخةً
 * تُقرأ عطبا في الرسم لا سرعة.
 */
private fun DrawScope.drawSpeedTrail(progress: Float) {
    if (progress <= 0f || progress >= 1f) return
    val fade = kotlin.math.sin(progress * Math.PI).toFloat()
    val h = size.height
    val w = size.width
    for (i in 0..2) {
        val y = h * (0.46f + i * 0.06f)
        val len = w * (0.22f - i * 0.04f) * fade
        drawLine(
            color = BrandOrange,
            start = androidx.compose.ui.geometry.Offset(-len, y),
            end = androidx.compose.ui.geometry.Offset(w * 0.06f, y),
            strokeWidth = h * 0.012f,
            alpha = 0.22f * fade,
        )
    }
}

/**
 * **لمعةٌ تمرّ فوق الشعار مرّةً واحدة.**
 *
 * **و`SrcAtop` تقصرها على شكل الشعار** — فلا تُضيء البياضَ حوله، **ولا
 * تُبدَّل ألوانُه**: بياضٌ شفّافٌ يمرّ فيفتّح ما تحته لحظةً ثمّ يزول.
 *
 * **وليست وهجا**: الوهجُ يمتدّ خارجَ الشكل ويُقرأ رخيصا.
 */
private fun DrawScope.drawSweep(progress: Float) {
    if (progress <= 0f || progress >= 1f) return
    val w = size.width
    val band = w * 0.45f
    val x = -band + (w + band * 2f) * progress
    drawRect(
        brush = Brush.horizontalGradient(
            0f to Color.Transparent,
            0.5f to Color.White.copy(alpha = 0.55f),
            1f to Color.Transparent,
            startX = x,
            endX = x + band,
        ),
        blendMode = BlendMode.SrcAtop,
    )
}

/** قوسٌ يذهب ويعود — من صفرٍ إلى واحدٍ ثمّ إلى صفر. **بلا ارتداد.** */
private fun arc(x: Float): Float =
    if (x <= 0f || x >= 1f) 0f else kotlin.math.sin(x * Math.PI).toFloat()
