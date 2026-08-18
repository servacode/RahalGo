package com.rahalgo.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الانتظار — موجاتُ العلامة لا دوّارةُ النظام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «شاشةُ التحميل بشكلٍ احترافيٍّ يناسب التطبيقَ
 *
 *	والمشروع، بحيث تكون شاشةُ جاري التحميل بأسلوبٍ مميّزٍ وجميلٍ
 *	ومركزيّ».)
 *
 * # ولماذا موجات
 *
 * **الدوّارةُ الافتراضيّةُ تقول «أندرويد ينتظر» لا «رحّال غو ينتظر»** —
 * وهي نفسُها في كلّ تطبيقٍ على الجهاز.
 *
 * **والموجاتُ لغةُ افتتاحنا نفسُها** (`BrandIntro`، وأقرّها المالكُ
 * ٢٠٢٦-٠٨-١١: «موجةٌ صغيرةٌ تكبر وتطلع الثانيةُ بعدها والثالثةُ
 * بعدها») — **فيرى الانتظارَ امتداداً لما رآه أوّلَ مرّة** لا شيئاً
 * غريباً عنه.
 *
 * # وثلاثٌ متتاليةٌ لا متداخلة
 *
 * **المتداخلةُ السريعةُ تُقرأ اضطراباً لا نبضاً** — وهي ملاحظةُ المالك
 * نفسُها على الافتتاح.
 *
 * # ولا تُبنى في كلّ شاشة
 *
 * **قطعةٌ واحدةٌ في العدّة المشتركة** — و`LoadState` تناديها، **فمن
 * غيّر شكلَ الانتظار غيّره في المنصّة كلِّها.**
 */
@Composable
fun RahalLoader(size: Dp = 64.dp, modifier: Modifier = Modifier) {
    val t = rememberInfiniteTransition(label = "loader")
    val p by t.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            // **وألفٌ وثمانُ مئةٍ لدورةٍ كاملة** — أسرعُ منها يُقرأ
            // عصبيّاً، **وأبطأُ يُقرأ توقّفا.**
            animation = tween(1_800, easing = LinearEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "wave",
    )

    val brand = Rahal.colors.brand
    val accent = Rahal.colors.accent

    Box(modifier.size(size), contentAlignment = Alignment.Center) {
        Canvas(Modifier.fillMaxSize()) {
            val c = Offset(this.size.width / 2f, this.size.height / 2f)
            val maxR = this.size.minDimension / 2f

            // **ثلاثُ موجاتٍ يفصل بينها ثلثُ الدورة** — فتخرج واحدةً
            // بعد واحدة.
            for (i in 0 until 3) {
                val phase = (p + i / 3f) % 1f
                // **وتبدأ من قلب العلامة لا من حافّتها** — فتُقرأ
                // صادرةً عنها.
                val r = maxR * (0.30f + 0.70f * phase)
                drawCircle(
                    color = brand,
                    radius = r,
                    center = c,
                    alpha = (1f - phase) * 0.55f,
                    style = Stroke(width = maxR * 0.06f),
                )
            }

            // **والقلبُ ثابتٌ لا ينبض** — نقطةٌ تتحرّك وموجاتٌ حولها
            // **تُقرأ شيئين يتنازعان العين.**
            drawCircle(color = accent, radius = maxR * 0.22f, center = c)
        }
    }
}

/**
 * **شاشةُ انتظارٍ كاملة** — للحظة التي لا شيءَ فيها بعد.
 *
 * **ونصٌّ تحتها اختياريّ**: «جارٍ التحميل» تحت كلّ انتظارٍ حشوٌ
 * يُقرأ ولا يُفيد — **ويُكتب حين يطول الانتظارُ لسببٍ يُقال.**
 */
@Composable
fun LoadingScreen(text: String = "", modifier: Modifier = Modifier) {
    Column(
        modifier
            .fillMaxSize()
            .padding(ScreenPad),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        RahalLoader()
        if (text.isNotEmpty()) {
            Spacer(Modifier.height(16.dp))
            Text(
                text = text,
                color = Rahal.colors.inkMuted,
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}
