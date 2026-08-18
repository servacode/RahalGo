package com.rahalgo.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import kotlinx.coroutines.delay

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رسالةٌ واحدةٌ تطفو فوق الشاشة — لكلّ نجاحٍ وكلّ فشل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٨: «مشان نخلص من قصّة الرسائل بشكلٍ كامل نخلّيه
 *  رسالةً منبثقةً تطلع ع شاشة بكلّ التطبيق، الخطأ والصح — وهيك ما يلزم
 *  نحدّد وين مكان كلّ رسالة… ما نلاحقها وين تروح ووين تجي».)
 *
 * # ما سبقها ولماذا سقط
 *
 * **أوّلاً كانت رسالةً واحدةً في أعلى الشاشة** — فمن ضغط زرّاً في أسفل
 * شاشةٍ تمرّر لم يرها، **فظنّ أنّ الضغطةَ لم تقع.**
 *
 * **ثمّ وُسمت بموضعها** فتُعرض عند الفعل — **وصار كلُّ زرٍّ يحتاج موضعاً
 * يُقرَّر له بيد**، ووقع ما يقع في كلّ قرارٍ يدويّ: **رسالةُ حذف الحساب
 * وُضعت بعد `return` فلم تُرسم أصلاً** (٢٠٢٦-٠٨-١٨)، **ورسالةُ الاسم
 * ظهرت تحت حقل الرقم.**
 *
 * **وثالثةٌ تُنهي البابَ**: الرسالةُ لا مكانَ لها في التخطيط — **تطفو
 * فوقه كلِّه**، فلا يُسأل عن موضعها في شاشةٍ جديدة.
 *
 * # ولماذا في `AppFrame`
 *
 * **يلفّ التطبيقاتِ الثلاثة** — فتُركَّب مرّةً وتعمل في كلّ شاشة.
 * **وثلاثةُ تركيباتٍ تُنسى في واحد.**
 *
 * # وحاملٌ ساكنٌ لا حالُ نموذج
 *
 * **النموذجُ يُنشأ ويُتلف مع شاشته** — ومن أرسل رسالةً ثمّ انتقل تُلفت
 * معه. **والحاملُ الساكنُ يبقى**، فتصل الرسالةُ وإن بدّل صاحبُها
 * الشاشةَ في أثناء النداء.
 *
 * # والرقمُ يتصاعد
 *
 * **ولا رايةٌ تُرفع وتُخفَض**: من أخطأ مرّتين بالخطأ نفسِه لا تظهر له
 * الثانية — **النصُّ لم يتغيّر فلا يُلتقط تغيّر.** **ورقمٌ يتصاعد
 * يُلتقط كلَّ مرّة.**
 */
object Flash {

    /** **ما يُعرض الآن** — وفارغٌ يعني لا شيء. */
    var current by mutableStateOf<Msg?>(null)
        private set

    private var seq = 0L

    data class Msg(val text: String, val ok: Boolean, val id: Long)

    /** **نجاحٌ** — أخضر. */
    fun ok(text: String) = show(text, true)

    /** **فشلٌ** — أحمر. */
    fun fail(text: String) = show(text, false)

    private fun show(text: String, ok: Boolean) {
        if (text.isBlank()) return
        seq += 1
        current = Msg(text, ok, seq)
    }

    fun clear() {
        current = null
    }
}

/**
 * **يرسم الرسالةَ فوق كلّ شيء** — يُركَّب في `AppFrame` وحدَه.
 *
 * **وفي الأعلى لا في الأسفل**: أسفلُ الشاشة شريطُ التبويبات في
 * التطبيقات الثلاثة، **ورسالةٌ تغطّيه تُخفي ما يريد أن يضغطه بعدها.**
 *
 * **وتنصرف وحدَها** — أربعُ ثوانٍ تكفي لقراءة سطر، **وتُمسح بلمسةٍ
 * لمن قرأ.**
 */
@Composable
fun FlashHost() {
    val msg = Flash.current
    // **والمهلةُ تتبع الرقمَ لا النصّ** — رسالتان بالنصّ نفسِه تبدآن
    // مهلتين، **ولو تتبّعت النصَّ لَورثت الثانيةُ ما بقي من الأولى.**
    LaunchedEffect(msg?.id) {
        if (msg == null) return@LaunchedEffect
        delay(4_000)
        if (Flash.current?.id == msg.id) Flash.clear()
    }

    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.TopCenter) {
        AnimatedVisibility(
            visible = msg != null,
            enter = fadeIn() + slideInVertically { -it },
            exit = fadeOut() + slideOutVertically { -it },
        ) {
            val m = msg ?: return@AnimatedVisibility
            Text(
                text = m.text,
                color = Rahal.colors.onBrand,
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier
                    .padding(horizontal = 16.dp, vertical = 12.dp)
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(14.dp))
                    .background(if (m.ok) Rahal.colors.success else Rahal.colors.danger)
                    .clickable { Flash.clear() }
                    .padding(horizontal = 16.dp, vertical = 14.dp),
            )
        }
    }
}
