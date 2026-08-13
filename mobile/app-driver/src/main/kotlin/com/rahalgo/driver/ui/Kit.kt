package com.rahalgo.driver.ui

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عُدّةُ الشاشات — قطعٌ تُكتب مرّةً وتُستعمل في كلّ قسم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك ٢٠٢٦-٠٨-١٣: «ركّز جيّداً على المركزيّة بكلّ شيء» ·
 *  «كلّ التعديلات التي أطلبها مركزيّة، لا تعدّل شيئاً بالمكان نفسه».)
 *
 * # لماذا وُجدت
 *
 * **كانت كلُّ شاشةٍ تكتب عنوانَها وحشوتَها وفاصلَها بنفسها** — والمحفظةُ
 * تكتب `Notice` خاصّاً بها، والسجلُّ يكتب مثلَه بلونٍ آخر. **وتسعةُ
 * أقسامٍ تُبنى دفعةً واحدةً تعني تسعَ نسخٍ من الشيء نفسه** إن لم تُكتب
 * قطعُها أوّلا.
 *
 * **والسمةُ الغامقةُ قادمة**: كلُّ لونٍ يُكتب في شاشةٍ **موضعٌ يُنسى
 * يومَ تُبدَّل الأرضُ** — فتبقى بقعةٌ بيضاءُ في شاشةٍ سوداء.
 *
 * # وما ليس هنا
 *
 * **لا منطقَ عملٍ ولا نداءَ شبكة** — هذه رسمٌ خالص. **وشاشةٌ تُنادي من
 * داخل قطعةٍ مشتركةٍ تجرّ القطعةَ إلى كلّ ما تعرفه.**
 */

/** حشوةُ الشاشة — **رقمٌ واحدٌ لا يُخمَّن في كلّ ملفّ.** */
val ScreenPad = 16.dp

/**
 * **جسدُ القسم** — عمودٌ يملأ الشاشةَ ويمرّر ما زاد.
 *
 * **والذيلُ فراغٌ بعد آخر سطر**: قائمةٌ تنتهي عند حافّة الشريط السفليّ
 * **تُقرأ مقصوصة**، فيظنّ صاحبُها أنّ تحتها ما لا يصله.
 */
@Composable
fun Screen(modifier: Modifier = Modifier, content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(ScreenPad),
    ) {
        content()
        Spacer(Modifier.height(32.dp))
    }
}

/**
 * **عنوانُ القسم وسطرُ شرحه.**
 *
 * **والشرحُ يقول ما هذه الشاشة بالنسبة إليه** — لا ما اسمُها: الاسمُ
 * فوقه في القائمة التي فتحها منها.
 */
@Composable
fun ScreenTitle(title: String, hint: String = "") {
    Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
    if (hint.isNotEmpty()) {
        Spacer(Modifier.height(4.dp))
        Text(hint, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
    }
    Spacer(Modifier.height(14.dp))
}

/** **عنوانُ مجموعةٍ داخل القسم.** */
@Composable
fun SectionTitle(text: String) {
    Spacer(Modifier.height(18.dp))
    Text(text, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
    Spacer(Modifier.height(8.dp))
}

/**
 * **بطاقةٌ** — أرضٌ خفيفةٌ تعزل ما فيها عمّا حولَه.
 *
 * **ولا ظلَّ ولا حدّ**: أرضٌ بلونٍ باهتٍ تكفي لتقول «هذه وحدة»، **وحدٌّ
 * حول كلّ شيءٍ يُشوّش الشاشة.**
 */
@Composable
fun Card(
    modifier: Modifier = Modifier,
    tone: Color = Rahal.colors.brand,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(
        modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(tone.copy(alpha = 0.06f))
            .padding(14.dp),
        content = content,
    )
}

/**
 * **سطرٌ فيه اسمٌ وقيمة** — الاسمُ باهتٌ والقيمةُ ظاهرة.
 *
 * **والقيمةُ في الطرف الآخر** — فتُقرأ عمودياً حين تتكرّر الأسطر.
 */
@Composable
fun KeyValue(label: String, value: String, valueColor: Color = Color.Unspecified) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
        Text(value, color = valueColor, style = MaterialTheme.typography.bodyMedium)
    }
}

/**
 * **خبرٌ ملوّنٌ في صندوقه** — خطأٌ أحمرُ أو تمامٌ أخضر.
 *
 * **وكان يُكتب في كلّ شاشةٍ من جديد** — بلونٍ ومقاسٍ يختلفان قليلا.
 */
@Composable
fun Note(text: String, color: Color) {
    Text(
        text = text,
        color = color,
        textAlign = TextAlign.Center,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(color.copy(alpha = 0.08f))
            .padding(10.dp),
    )
    Spacer(Modifier.height(10.dp))
}

/**
 * **فراغٌ يقول إنّه فراغ.**
 *
 * **وشاشةٌ بيضاءُ تُقرأ عطبا** — ومن فتح قسماً فلم يجد فيه شيئاً ولا
 * كلمةً ظنّ التطبيقَ لم يحمّل.
 */
@Composable
fun Empty(text: String) {
    Box(
        Modifier.fillMaxWidth().padding(vertical = 40.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(text, color = Rahal.colors.inkMuted, textAlign = TextAlign.Center)
    }
}

/**
 * **حالُ التحميل والفشل — مرّةً لكلّ الأقسام.**
 *
 * **وفشلُ القراءة ليس «لا شيءَ عندك»**: قائمةٌ فارغةٌ تُقرأ خبراً،
 * **والخبرُ كاذب.** فيُقال إنّه تعذّر، **ويُعرض زرٌّ يعيد المحاولة** —
 * وشبكةُ الشارع تنقطع وتعود.
 */
@Composable
fun LoadState(loading: Boolean, error: String, onRetry: (() -> Unit)? = null) {
    Column(
        Modifier.fillMaxWidth().fillMaxHeight(0.6f),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        if (error.isNotEmpty()) {
            Text(error, color = Rahal.colors.danger, textAlign = TextAlign.Center)
            if (onRetry != null) {
                Spacer(Modifier.height(12.dp))
                OutlinedButton(onClick = onRetry) {
                    Text(stringResource(R.string.act_retry))
                }
            }
        } else {
            // **ولا فراغَ بين الفتحة وبدء النداء** — كان يُشترط `loading`،
            // **والرايةُ لا تُرفع إلّا بعد إطارٍ أو إطارين** فيرى فتّاحُ
            // القسم بياضاً ثمّ دوّارة. **وومضةٌ بيضاءُ تُقرأ عطبا.**
            CircularProgressIndicator()
        }
    }
}

/**
 * **شريطُ تقدّمٍ** — نسبةٌ تُرى قبل أن يُقرأ رقم.
 *
 * **ولونُه يقول حالَه**: أخضرُ ما دام بعيداً، **وأحمرُ عند الحافّة.**
 */
@Composable
fun Bar(ratio: Float, color: Color) {
    Box(
        Modifier
            .fillMaxWidth()
            .height(8.dp)
            .clip(RoundedCornerShape(4.dp))
            .background(Rahal.colors.inkMuted.copy(alpha = 0.15f)),
    ) {
        Box(
            Modifier
                .fillMaxWidth(ratio.coerceIn(0f, 1f))
                .fillMaxSize()
                .clip(RoundedCornerShape(4.dp))
                .background(color),
        )
    }
}

/** **رقاقةُ حال** — كلمةٌ في أرضٍ من لونها. */
@Composable
fun Chip(text: String, color: Color) {
    Text(
        text = text,
        color = color,
        style = MaterialTheme.typography.labelMedium,
        fontWeight = FontWeight.Bold,
        modifier = Modifier
            .clip(RoundedCornerShape(8.dp))
            .background(color.copy(alpha = 0.12f))
            .padding(horizontal = 8.dp, vertical = 3.dp),
    )
}

/** **عنوانُ يومٍ فوق حركاته** — «اليوم» أوضحُ من تاريخٍ كامل. */
@Composable
fun DayHead(text: String) {
    Spacer(Modifier.height(14.dp))
    Text(
        text = text,
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.Bold,
    )
    Spacer(Modifier.height(4.dp))
}
