package com.rahalgo.design.intro

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathMeasure
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.VectorPainter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.vectorResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandCanvas
import com.rahalgo.design.InkMuted
import com.rahalgo.design.R
import com.rahalgo.design.TaglineStyle
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **افتتاحُ رحّال غو — موقعٌ ثمّ طريقٌ ثمّ سائقٌ ثمّ وصول**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلب المالك ٢٠٢٦-٠٨-١١، بتفصيلٍ كامل للمراحل والمدد.)
 *
 * **والحركةُ تحكي ما تفعله المنصّة**: تُحدَّد نقطةٌ، فيُرسم طريق، فيمشي
 * سائق، فيصل. **وهذا معنى العلامة لا زخرفة فوقها.**
 *
 * # ولماذا أربعُ طبقاتٍ لا صورةٌ واحدة
 *
 * **الشعارُ ملفٌّ مسطّح**: الحرفُ والدرّاجةُ باللون السماويّ نفسِه بالضبط
 * (٨٣٪ من الشعار)، والطريقُ يمرّ خلفَ الحرف وأمامَه. **فصلُه بالألوان
 * يُخرج الاثنين معا بهالةٍ بيضاءَ حول كلّ قطعة.**
 *
 * فأُعدّت أربعُ طبقاتٍ متجهيّةٍ منفصلة، **كلٌّ منها في لوحة ١٠٢٤ في
 * موضعه من الشعار الكامل** — فتُركَّب فوق بعضها فتعطي الشعارَ حرفيّا.
 *
 * # وخطُّ منتصف الطريق
 *
 * الطريقُ **شكلٌ مملوءٌ لا خطّ**، فلا يُمكن «رسمُه» تدريجيّا كما يُرسم
 * خطّ. **فاستُخرج منه خطُّ منتصفٍ** (`RoutePoints`) يُستعمل مرّتين:
 * قناعا يكشف الطريقَ من أوّله إلى آخره، **ومسارا تمشي عليه الدرّاجة** —
 * فتمشي فوقه لا بجانبه.
 *
 * # والمدد
 *
 * ```
 *    0 →  300   العلامةُ تظهر وتنبض نبضةً واحدة        «تُحدّد النقطة»
 *  250 →  800   الطريقُ يُرسم من أسفلَ إلى أعلى        «يُفتح الطريق»
 *  550 → 1050   الدرّاجةُ تمشي عليه ومعها ميلٌ خفيف     «يتحرّك السائق»
 *  850 → 1350   الحرفُ والسهمُ يتكوّنان                 «يصل»
 * 1300 → 1550   الشعارُ يستقرّ ونبضةٌ أخيرة
 * 1450 → 1700   العبارةُ تظهر
 * ```
 *
 * **والمراحلُ متداخلةٌ عمدا** — لا متتابعة: حركةٌ تنتظر أختَها تُقرأ
 * بطيئةً حتّى لو كانت مدّتُها قصيرة.
 */

/** مدّةُ الحركة كاملةً — **والانتقالُ بعدها من مسؤوليّة المنادي.** */
const val IntroDurationMs = 1700

/** المدّةُ حين تكون حركاتُ النظام مطفأة — **تُعرض النتيجةُ ولا يُنتظر.** */
const val IntroReducedMs = 220

// اللوحةُ التي رُسمت عليها الطبقاتُ الأربع.
private const val CANVAS = 1024f

/** مركزُ علامة الموقع في اللوحة — محورُ تكبيرها ونبضها. */
private val PinPivot = Offset(456f, 390f)

/** موضعُ الدرّاجة في الشعار الساكن — منه تُقاس إزاحتُها. */
private val BikeHome = Offset(358f, 720f)

/**
 * **نسبةُ موضع الدرّاجة من طول المسار — ٢١٪.**
 *
 * **مقيسةٌ لا مقدَّرة**: أقربُ نقطةٍ في خطّ المنتصف إلى `BikeHome` تقع
 * عند ٠٫٢١٠ من طوله البالغ ٧٣٠. **فتصل الدرّاجةُ إلى مكانها من الشعار
 * بالضبط**، ولا تقف بجانبه.
 */
private const val BikeHomeFraction = 0.21f

// **منحنى الحركة** — بدايةٌ سريعةٌ ونهايةٌ هادئة. وهو ما يجعلها تُقرأ
// «مقصودة» لا «مندفعة».
private val Smooth = FastOutSlowInEasing
private val Settle = CubicBezierEasing(0.22f, 1f, 0.36f, 1f)

/**
 * @param reduceMotion تُمرَّر من المنادي بعد قراءة إعدادات النظام.
 * @param onFinished يُنادى مرّةً واحدةً حين تنتهي الحركة.
 */
@Composable
fun BrandIntro(
    modifier: Modifier = Modifier,
    tagline: String,
    reduceMotion: Boolean = false,
    onFinished: () -> Unit = {},
) {
    // **قيمةٌ واحدةٌ لكلّ ما يتحرّك** — لا مؤقّتاتٌ متفرّقة: مؤقّتاتٌ
    // تُطلق معا تنجرف كلٌّ منها بمقدارٍ مختلفٍ على جهازٍ بطيء،
    // **فتتفكّك الحركةُ ولا يُعرف السبب.**
    val t = remember { Animatable(0f) }

    LaunchedEffect(reduceMotion) {
        if (reduceMotion) {
            // **تُعرض النتيجةُ مباشرةً** — ومن أطفأ الحركةَ في نظامه أراد
            // ذلك، **وتطبيقٌ يتجاهله يُقرأ معطوبا لا أنيقا.**
            t.snapTo(1f)
            kotlinx.coroutines.delay(IntroReducedMs.toLong())
        } else {
            t.animateTo(1f, tween(IntroDurationMs, easing = LinearEasing))
        }
        onFinished()
    }

    val p = t.value
    val pin = stage(p, 0, 300)
    val pinPulse = stage(p, 180, 300)
    val route = stage(p, 250, 800)
    val bike = stage(p, 550, 1050)
    val mark = stage(p, 850, 1350)
    val settle = stage(p, 1300, 1550)
    val settlePulse = settle
    val text = stage(p, 1450, 1700)

    val pinPainter = rememberVectorPainter(ImageVector.vectorResource(R.drawable.intro_pin))
    val routePainter = rememberVectorPainter(ImageVector.vectorResource(R.drawable.intro_route))
    val bikePainter = rememberVectorPainter(ImageVector.vectorResource(R.drawable.intro_motorcycle))
    val markPainter = rememberVectorPainter(ImageVector.vectorResource(R.drawable.intro_mark))
    // ══════════════════════════════════════════════════════════════════
    // **وكلُّ ما يُقاس يُبنى مرّةً — لا في كلّ إطار**
    // ══════════════════════════════════════════════════════════════════
    //
    // **قِيس على جهازٍ حقيقيّ: ٤٢٪ من الإطارات متقطّعة** حين كانت
    // `PathMeasure` و`Paint` تُبنيان داخل الرسم. **وستّون إطاراً في
    // الثانية تعني ستّين تخصيصاً** في أثقل لحظةٍ في عمر التطبيق —
    // لحظةِ إقلاعه.
    //
    // **والمسارُ ثابتٌ لا يتغيّر**، وإنّما يتغيّر مقدارُ ما يُقتطع منه.
    val center = remember { routeCenterline() }
    val measure = remember { PathMeasure().apply { setPath(center, false) } }
    val length = remember { measure.length }
    // **وأوعيةٌ تُعاد تعبئتُها** — لا تُخصَّص من جديد.
    val revealPath = remember { Path() }
    val maskPath = remember { Path() }
    val maskPaint = remember {
        android.graphics.Paint().apply {
            style = android.graphics.Paint.Style.STROKE
            strokeWidth = 300f
            strokeCap = android.graphics.Paint.Cap.ROUND
            strokeJoin = android.graphics.Paint.Join.ROUND
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(BrandCanvas),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(210.dp)
                // **استقرارٌ بلا ارتداد** — من ٠٫٩٦ إلى ١، ومنحنىً يهدأ.
                .scale(0.96f + 0.04f * Settle.transform(settle)),
        ) {
            androidx.compose.foundation.Canvas(Modifier.fillMaxSize()) {
                val k = size.minDimension / CANVAS
                scaled(k) {
                    // ══════════════════════════════════════════════════
                    // **وترتيبُ الرسم ترتيبُ الطبقات في الشعار**
                    // ══════════════════════════════════════════════════
                    //
                    // **وهو ترقيمُ ملفّات المصدر نفسُه**: الحرفُ أوّلاً
                    // (خلفَ الكلّ)، ثمّ العلامةُ في جوفه، ثمّ الطريقُ
                    // يمرّ أمامَه، **ثمّ الدرّاجةُ فوق الجميع.**
                    //
                    // **والزمنُ غيرُ الترتيب**: الحرفُ يُرسم أوّلاً ويظهر
                    // آخراً — لا تعارضَ بينهما.

                    // ١ · الحرفُ والسهم — يُكشفان من اليسار إلى اليمين،
                    // **فيكون السهمُ آخرَ ما يظهر** ويُقرأ انطلاقاً.
                    if (mark > 0f) {
                        clipRect(0f, 0f, CANVAS * Smooth.transform(mark), CANVAS) {
                            layer(markPainter, 1f)
                        }
                    }

                    // ٢ · العلامة — ظهورٌ وتكبيرٌ صغيرٌ ثمّ نبضةٌ واحدة.
                    val pinScale = 0.85f + 0.15f * Smooth.transform(pin) +
                        0.04f * pulse(pinPulse) + 0.03f * pulse(settlePulse)
                    layer(pinPainter, alpha = Smooth.transform(pin), scale = pinScale,
                        pivot = PinPivot)

                    // ٣ · الطريقُ يُرسم — قناعٌ يمشي على خطّ المنتصف.
                    if (route > 0f) {
                        revealPath.reset()
                        measure.getSegment(0f, length * Smooth.transform(route), revealPath, true)
                        // **وعرضُ القناع أوسعُ من أعرض موضعٍ في الطريق**
                        // (٢٤١ بكسلاً) — وأضيقُ منه يقصّ حوافَّه.
                        maskPath.reset()
                        maskPaint.getFillPath(revealPath.asAndroidPath(), maskPath.asAndroidPath())
                        clipPath(maskPath) { layer(routePainter, 1f) }
                    }

                    // ٤ · الدرّاجةُ تمشي على الخطّ نفسِه **وتستقرّ في مكانها.**
                    //
                    // **ولا تختفي في النهاية** — هي جزءٌ من الشعار،
                    // **وإخفاؤها يترك فجوةً بيضاءَ في الطريق** لأنّ طبقةَ
                    // الطريق مقصوصةٌ حولها. (وقع وشُوهد في أوّل بناء.)
                    //
                    // **ومسافتُها قصيرة** — إلى موضعها من الشعار (٢١٪ من
                    // المسار) لا إلى آخره: **حركةٌ توحي بالتوصيل، لا
                    // رحلةٌ عبر الشاشة.**
                    if (bike > 0f) {
                        val at = length * BikeHomeFraction * Smooth.transform(bike)
                        val pos = measure.getPosition(at)
                        val tan = measure.getTangent(at)
                        val tilt = Math.toDegrees(
                            kotlin.math.atan2(tan.y.toDouble(), tan.x.toDouble())
                        ).toFloat()
                        // **وميلٌ خفيفٌ جدّاً** — خُمسُ ميل المسار، ويتلاشى
                        // عند الاستقرار. **وميلٌ كاملٌ يقلبها على المنعطف.**
                        val settleOff = 1f - Smooth.transform(bike)
                        translate(pos.x - BikeHome.x, pos.y - BikeHome.y) {
                            rotate(tilt * 0.2f * settleOff, pivot = BikeHome) {
                                layer(bikePainter, 1f)
                            }
                        }
                    }
                }
            }
        }

        androidx.compose.foundation.layout.Spacer(Modifier.height(18.dp))

        Text(
            text = tagline,
            style = TaglineStyle,
            color = InkMuted,
            modifier = Modifier
                .alpha(Smooth.transform(text))
                // **وترتفعُ ثمانيَ نقاطٍ وهي تظهر** — حركةٌ تقول «جاءت
                // من مكان»، لا «ومضت».
                .offsetY((8f * (1f - Smooth.transform(text))).dp),
        )
    }
}

// ══════════════════════════════════════════════════════════════════════
//  أدواتٌ صغيرة
// ══════════════════════════════════════════════════════════════════════

/** نسبةُ تقدّمِ مرحلةٍ من زمن الحركة الكلّيّ — من ٠ إلى ١. */
private fun stage(progress: Float, fromMs: Int, toMs: Int): Float {
    val now = progress * IntroDurationMs
    if (now <= fromMs) return 0f
    if (now >= toMs) return 1f
    return (now - fromMs) / (toMs - fromMs)
}

/** نبضةٌ واحدةٌ ناعمة — تصعد ثمّ تعود. **ولا تتكرّر.** */
private fun pulse(x: Float): Float =
    if (x <= 0f || x >= 1f) 0f else kotlin.math.sin(x * Math.PI).toFloat()

private fun fadeOut(x: Float): Float = (1f - x).coerceIn(0f, 1f)

private fun DrawScope.scaled(k: Float, body: DrawScope.() -> Unit) {
    scale(k, pivot = Offset.Zero) { body() }
}

private fun DrawScope.layer(
    painter: VectorPainter,
    alpha: Float,
    scale: Float = 1f,
    pivot: Offset = Offset(CANVAS / 2f, CANVAS / 2f),
) {
    if (alpha <= 0f) return
    scale(scale, pivot = pivot) {
        with(painter) { draw(Size(CANVAS, CANVAS), alpha = alpha) }
    }
}

private fun DrawScope.clipRect(l: Float, t: Float, r: Float, b: Float, body: DrawScope.() -> Unit) {
    val p = Path().apply { addRect(androidx.compose.ui.geometry.Rect(l, t, r, b)) }
    clipPath(p) { body() }
}

private fun androidx.compose.ui.graphics.Path.asAndroidPath(): android.graphics.Path =
    (this as androidx.compose.ui.graphics.AndroidPath).internalPath

/** خطُّ منتصف الطريق مسارا — **يُبنى مرّةً ويُعاد استعمالُه.** */
private fun routeCenterline(): Path = Path().apply {
    RoutePoints.forEachIndexed { i, (x, y) ->
        if (i == 0) moveTo(x, y) else lineTo(x, y)
    }
}

private fun Modifier.offsetY(dp: androidx.compose.ui.unit.Dp): Modifier =
    this.offset(y = dp)
