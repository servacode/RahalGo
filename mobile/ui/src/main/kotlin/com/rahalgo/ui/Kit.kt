package com.rahalgo.ui

import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import kotlinx.coroutines.delay
import androidx.compose.foundation.background
import androidx.compose.foundation.shape.CircleShape
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.unit.sp
import androidx.compose.ui.unit.Dp

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
 * ══════════════════════════════════════════════════════════════════════
 * **شارةُ عدد — دائرةٌ لا بيضة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٥: «المفروض دائريّ أنيق».)
 *
 * **و`CircleShape` على صندوقٍ غيرِ مربّعٍ يرسم قطعاً ناقصا** —
 * والصندوقُ الذي يقيس نفسَه على نصٍّ **يخرج أطولَ من عرضه**: سطرُ
 * النصّ فيه فراغُ الخطّ فوقه وتحته.
 *
 * **فمقاسُها ثابتٌ مربّعٌ والرقمُ في وسطه** — **ودائرةٌ من مربّعٍ
 * دائرةٌ دائما.**
 *
 * **وارتفاعُ السطر يُقصَر على حجم الحرف** — **وإلّا دفع النصُّ نفسَه
 * إلى أسفل المربّع** فبدا الرقمُ غيرَ متوسّط.
 *
 * **وتسعةٌ فما فوق «+٩»** — رقمٌ من ثلاث خاناتٍ يخرج من دائرته.
 *
 * **وواحدةٌ لا اثنتان**: كانت على الجرس نسخةٌ وعلى السلّة نسخة —
 * **ففُصّلت إحداهما ولم تُفصَّل الأخرى.**
 */
@Composable
fun CountBadge(
    count: Int,
    color: Color,
    /** **قطرُها** — والجرسُ أصغرُ من السلّة لأنّ أيقونتَه أصغر. */
    size: Dp = 18.dp,
    modifier: Modifier = Modifier,
) {
    if (count <= 0) return
    val text = if (count > 9) "+9" else count.toString()
    Box(
        modifier
            .size(size)
            .clip(CircleShape)
            .background(color),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            color = Color.White,
            fontSize = size.value.times(0.55f).sp,
            lineHeight = size.value.times(0.55f).sp,
            fontWeight = FontWeight.Bold,
        )
    }
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
            .clip(Rahal.shape.md)
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
            .clip(Rahal.shape.sm)
            .background(color.copy(alpha = 0.08f))
            .padding(10.dp),
    )
    Spacer(Modifier.height(10.dp))
}

/**
 * **فراغٌ يقول إنّه فراغ — وماذا بعد.**
 *
 * **وشاشةٌ بيضاءُ تُقرأ عطبا** — ومن فتح قسماً فلم يجد فيه شيئاً ولا
 * كلمةً ظنّ التطبيقَ لم يحمّل.
 *
 * ══════════════════════════════════════════════════════════════════════
 * **وسطرٌ واحدٌ لا يكفي** (٢٠٢٦-٠٩-١٣، شرطُ المالك)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكلُّ فراغٍ يجيب سؤالين**: **ما وقع؟** و**ما يفعله المرءُ الآن؟**
 *
 * **و«لا توجد طلبات» تجيب الأوّلَ وتترك الثاني** — **فيقف صاحبُها ينظر.**
 *
 * **ولا يُختلَق محتوىً**: **صنفٌ وهميٌّ ليبدو القسمُ عامراً كذبٌ** —
 * **والصدقُ سطرٌ وزرٌّ يعيد المحاولة.**
 *
 * **والتلميحُ والزرُّ اختياريّان** — **فما كان يُنادى بسطرٍ يبقى كما هو**،
 * ولا تُمَسّ مئةُ موضعٍ لأجل حقلٍ أُضيف.
 */
@Composable
fun Empty(
    text: String,
    /** **الخطوةُ التالية** — سطرٌ يقول ما يفعله، وفارغٌ يعني لا خطوة. */
    hint: String = "",
    /** **نصُّ الزرّ** — وفارغٌ يعني لا زرّ. */
    actionLabel: String = "",
    onAction: (() -> Unit)? = null,
) {
    Box(
        Modifier.fillMaxWidth().padding(vertical = 40.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.padding(horizontal = 24.dp),
        ) {
            Text(text, color = Rahal.colors.inkMuted, textAlign = TextAlign.Center)
            if (hint.isNotBlank()) {
                Spacer(Modifier.height(6.dp))
                Text(
                    hint,
                    color = Rahal.colors.inkMuted,
                    textAlign = TextAlign.Center,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            if (actionLabel.isNotBlank() && onAction != null) {
                Spacer(Modifier.height(14.dp))
                // **وزرٌّ ثانويٌّ لا أساسيّ** — **الفراغُ ليس دعوةً ملحّة**،
                // وزرٌّ بلون العلامة وسط شاشةٍ فارغةٍ يُضغط سهواً.
                RahalOutlineButton(onClick = onAction) { Text(actionLabel) }
            }
        }
    }
}

/**
 * **حالُ التحميل والفشل — مرّةً لكلّ الأقسام.**
 *
 * **وفشلُ القراءة ليس «لا شيءَ عندك»**: قائمةٌ فارغةٌ تُقرأ خبراً،
 * **والخبرُ كاذب.** فيُقال إنّه تعذّر، **ويُعرض زرٌّ يعيد المحاولة** —
 * وشبكةُ الشارع تنقطع وتعود.
 */
/**
 * **متى يُقال «تعذّر» بدل الدوران** — **عشرون ثانية.**
 *
 * **ومن القياس لا من الذوق**: **أبطأُ بابٍ على التجهيز ٢٦ مل.ث،
 * وزمنُ الوصول من الجهاز ٢٦٥ مل.ث** — **فهذا أضعافُ أضعافِه.**
 */
private const val STUCK_AFTER_MS = 20_000L

@Composable
fun LoadState(loading: Boolean, error: String, onRetry: (() -> Unit)? = null) {
    // ══════════════════════════════════════════════════════════════════
    // **ودوّارةٌ بلا نهايةٍ عطبٌ لا انتظار** (`PF-01`، ٢٠٢٦-٠٩-١٦)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وكان الشرطُ: إن لم يكن خطأٌ فدُرْ** — **فحالُ «لا تحميلَ ولا
    // خطأ» تدور أبداً**: **نداءٌ ابتُلع، أو ردٌّ فارغٌ، أو رايةٌ لم
    // تُخفَض.** **ومن رآها انتظر ثمّ خرج، ولا شيءَ يقول له أن يعيد.**
    //
    // **ولا يُعاد شرطُ `loading`** — **رفعُه كان إصلاحاً**: **الرايةُ
    // لا تُرفع إلّا بعد إطارٍ أو إطارين، فيرى الفاتحُ بياضاً ثمّ
    // دوّارة.**
    //
    // **فتُحَدُّ بالزمن**: **تدور كما كانت، وإن طال الدورانُ بلا تبدّلٍ
    // قيل «تعذّر» وعُرض بابُ الإعادة.** **والحدُّ من القياس**: **أبطأُ
    // بابٍ على التجهيز ٢٦ مل.ث، وزمنُ الوصول من هنا ٢٦٥ مل.ث** —
    // **فعشرون ثانيةً أضعافُ أضعافِه، ولا تقطع على شبكةٍ بطيئةٍ صادقة.**
    var stuck by remember(loading, error) { mutableStateOf(false) }
    LaunchedEffect(loading, error) {
        stuck = false
        if (error.isEmpty()) {
            delay(STUCK_AFTER_MS)
            stuck = true
        }
    }
    if (stuck && error.isEmpty()) {
        Column(
            Modifier.fillMaxWidth().fillMaxHeight(0.6f),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Text(
                stringResource(R.string.load_stuck),
                color = Rahal.colors.inkMuted,
                textAlign = TextAlign.Center,
            )
            if (onRetry != null) {
                Spacer(Modifier.height(12.dp))
                RahalOutlineButton(onClick = onRetry) {
                    Text(stringResource(R.string.act_retry))
                }
            }
        }
        return
    }
    Column(
        Modifier.fillMaxWidth().fillMaxHeight(0.6f),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        if (error.isNotEmpty()) {
            Text(error, color = Rahal.colors.danger, textAlign = TextAlign.Center)
            if (onRetry != null) {
                Spacer(Modifier.height(12.dp))
                RahalOutlineButton(onClick = onRetry) {
                    Text(stringResource(R.string.act_retry))
                }
            }
        } else {
            // **ولا فراغَ بين الفتحة وبدء النداء** — كان يُشترط `loading`،
            // **والرايةُ لا تُرفع إلّا بعد إطارٍ أو إطارين** فيرى فتّاحُ
            // القسم بياضاً ثمّ دوّارة. **وومضةٌ بيضاءُ تُقرأ عطبا.**
            //
            // **وموجاتُ العلامة لا دوّارةُ النظام** — انظر `RahalLoader`:
            // (طلبُ المالك ٢٠٢٦-٠٨-١٨)، **ودوّارةٌ افتراضيّةٌ تقول
            // «أندرويد ينتظر» لا «رحّال غو ينتظر».**
            RahalLoader()
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
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.inkMuted.copy(alpha = 0.15f)),
    ) {
        Box(
            Modifier
                .fillMaxWidth(ratio.coerceIn(0f, 1f))
                .fillMaxSize()
                .clip(Rahal.shape.pill)
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
            .clip(Rahal.shape.sm)
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مربّعُ رقم — الاسمُ فوقه والرقمُ تحته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(طلبُ المالك ٢٠٢٦-٠٨-٣١:** «طلبات سلّمت اجعلها مربّعاً تحتها عددُ
 * الطلبات · وعمولتك عنه أيضاً مربّعاً تحته مبلغ».)
 *
 * # ولماذا مربّعٌ لا سطرٌ باسمٍ وقيمة
 *
 * **السطرُ يُقرأ نصّاً فيُمرَّر عليه** — والرقمُ فيه بحجم الكلام حوله.
 * **والمربّعُ يُرى قبل أن يُقرأ**: عينٌ تمسح الشاشةَ تلتقط الأرقامَ
 * وحدَها، **ومن أراد تفصيلاً قرأ اسمَه فوقها.**
 *
 * # وهنا لا في كلّ تطبيق
 *
 * **كانت نسختان**: في تقييم السائق، وفي لوحة المتجر. **وثالثةٌ كانت
 * ستُكتب للمندوب** — **وثلاثُ نسخٍ تفترق يومَ يتبدّل لونٌ أو حشوة.**
 *
 * **والرقمُ الطويلُ يصغر خطُّه** — مبلغٌ بستّة أرقامٍ يكسر المربّعَ
 * أو يُقصّ، **ورقمٌ مقصوصٌ أسوأُ من رقمٍ صغير.**
 */
@Composable
fun StatBox(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    color: Color = Rahal.colors.ink,
) {
    Column(
        modifier
            .fillMaxHeight()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.inkMuted.copy(alpha = 0.07f))
            .padding(horizontal = 8.dp, vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        // **والاسمُ سطران دائماً** — (بلاغُ المالك ٢٠٢٦-٠٨-٣١: «مربّعٌ
        // أصغرُ من مربّع، ليسا بنفس الحجم والترتيب»).
        //
        // **واسمٌ يلتفّ سطرين وآخرُ سطراً يجعل مربّعَه أطول** — فيُقرأ
        // الفرقُ معنًى وليس فيه معنى. **فيُحجَز سطران للجميع.**
        Text(
            text = label,
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
            textAlign = TextAlign.Center,
            minLines = 2,
            maxLines = 2,
        )
        Spacer(Modifier.height(6.dp))
        // **والرقمُ بحجمٍ واحدٍ مهما طال** — **وخطّان مختلفان في صفٍّ
        // واحدٍ يقولان إنّ أحدَ الرقمين أهمّ**، وليس كذلك.
        Text(
            text = value,
            color = color,
            style = MaterialTheme.typography.titleMedium,
            textAlign = TextAlign.Center,
            maxLines = 1,
        )
    }
}

/**
 * **صفُّ مربّعاتٍ متساويةٍ حجماً وترتيبا.**
 *
 * **(بلاغُ المالك ٢٠٢٦-٠٨-٣١:** «المربّعاتُ جميلة، ولكنّ المشكلةَ
 * مربّعٌ أصغرُ من مربّع — ليسا بنفس الحجم والترتيب».)
 *
 * **و`IntrinsicSize.Min` تجعل الصفَّ بطول أطولِ مربّعٍ فيه** ثمّ يملؤه
 * الباقي — **بلا رقمٍ ثابتٍ يُكسر يومَ يطول اسم.**
 *
 * **ومن كتب `Row` بيده نسي واحدةً منهما** — فيُعطى الصفُّ جاهزاً.
 */
@Composable
fun StatRow(modifier: Modifier = Modifier, content: @Composable RowScope.() -> Unit) {
    Row(
        modifier.fillMaxWidth().height(IntrinsicSize.Min),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        content = content,
    )
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سطرُ حسبة — اسمٌ يميناً ورقمٌ يساراً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(بلاغُ المالك ٢٠٢٦-٠٨-٣١:** «سجلّ الطلبات والطلبات ما عجبتني
 * الترتيبة بصراحة، مو احترافيّة بالمستوى المطلوب» · «المهمّ يكون الشكلُ
 * احترافيّاً مفهوماً واضحا».)
 *
 * # ولماذا لا مربّعات
 *
 * **المربّعُ لرقمٍ مستقلّ** — طلباتُ اليوم، والملغى، والمبيعات: **ثلاثةُ
 * مقاييسَ لا رابطَ بينها**، والمربّعُ يفصلها فيُقرأ كلٌّ وحدَه.
 *
 * **وهذه حسبةٌ لا مقاييس**: ١٥٠ ناقصَ ١٥ يساوي ١٣٥. **والمربّعاتُ تكسر
 * الطرحَ** فتُقرأ ثلاثةَ أشياءَ، **ومن أراد أن يتحقّق لم يجد الطرحَ
 * أمامه.**
 *
 * **وشكلُ الفاتورة يُقرأ بلا تعليم** — لأنّه شكلُ كلّ فاتورةٍ رآها في
 * حياته: البنودُ فوق، ثمّ الجمعُ، ثمّ الخصمُ بإشارته، **ثمّ خطٌّ
 * والمحصّلةُ تحته.**
 */
@Composable
fun SumLine(
    label: String,
    value: String,
    strong: Boolean = false,
    color: Color = Color.Unspecified,
) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = if (strong) 4.dp else 2.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            color = if (strong) Rahal.colors.ink else Rahal.colors.inkMuted,
            fontWeight = if (strong) FontWeight.Bold else FontWeight.Normal,
            style = if (strong) {
                MaterialTheme.typography.bodyLarge
            } else {
                MaterialTheme.typography.bodyMedium
            },
        )
        Text(
            text = value,
            color = if (color != Color.Unspecified) color else Rahal.colors.ink,
            fontWeight = if (strong) FontWeight.Bold else FontWeight.Medium,
            style = if (strong) {
                MaterialTheme.typography.titleMedium
            } else {
                MaterialTheme.typography.bodyMedium
            },
        )
    }
}
