package com.rahalgo.driver.rating

import android.app.Application
import android.util.Log
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateGreen
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.shared.model.ComplaintBrief
import com.rahalgo.shared.model.Reputation
import com.rahalgo.shared.model.Review
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تقييماتُه — نجومُه ومن أعطاها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ننتقل إلى التقييم، نبني الصفحة أيضاً…
 *  وعند النقر عليه يفتح صفحة التقييم كما هي بالويب، فيعرف ما هي
 *  التقييمات التي حصل عليها ومن أين».)
 *
 * # ورقمٌ بلا تفصيلٍ يُقلق ولا يُعلّم
 *
 * **كان في الشريط رقمٌ وحدَه**: «٤٫٢». ومن نزل تقييمُه **لا يعرف أيَّ
 * بابٍ كان ولا ماذا قيل** — فيسأل عمّا فعل ولا جواب. **وتقييمٌ يُحاسَب
 * عليه ولا يُفسَّر يُقرأ حكماً على شخصه** لا ملاحظةً على خدمة.
 *
 * # والاتّجاهُ قبل التفصيل
 *
 * **متوسّطُ الكلّ يتحرّك ببطء**: من كان على ٤٫٨ في مئة تقييمٍ لا يُغيّره
 * أسبوعٌ سيّئ. **ومتوسّطُ الشهر يقولها فورا** — فيُعرَض الاثنان،
 * والفرقُ بينهما هو الخبر.
 *
 * # والشكاوى معها — بلا أسماء
 *
 * **الشكوى خصومةٌ تُحقَّق**، وكشفُ صاحبها يفتح باباً لمن يريد أن يردّ
 * عليه. **والتقييمُ ثناءٌ أو ملاحظةٌ على خدمةٍ وقف فيها أمامه** —
 * فيُنسَب. (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
 */
class RatingViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf<Reputation?>(null)
        private set
    var error by mutableStateOf("")
        private set

    private val backend = Backend.of(getApplication())

    init {
        load()
        // **وتسمع نبضةَ التحديث** — تقييمٌ جديدٌ يصل وهو ينظر.
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        viewModelScope.launch {
            try {
                state = backend.me.reputation()
                error = ""
            } catch (e: Exception) {
                Log.e("RahalGo/تقييم", "فشل نداء السمعة", e)
                error = getApplication<Application>().getString(R.string.err_internal)
            }
        }
    }
}

@Composable
fun RatingScreen(vm: RatingViewModel) {
    val rep = vm.state
    if (rep == null) {
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            if (vm.error.isEmpty()) {
                CircularProgressIndicator()
            } else {
                Text(vm.error, color = StateRed, textAlign = TextAlign.Center)
            }
        }
        return
    }

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        Summary(rep)
        Spacer(Modifier.height(20.dp))
        HorizontalDivider()
        Spacer(Modifier.height(16.dp))

        Text(stringResource(R.string.rate_reviews), style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(10.dp))
        if (rep.reviews.isEmpty()) {
            Text(stringResource(R.string.rate_no_reviews), color = InkMuted)
        }
        rep.reviews.forEach { ReviewRow(it) }

        if (rep.complaints.isNotEmpty()) {
            Spacer(Modifier.height(20.dp))
            HorizontalDivider()
            Spacer(Modifier.height(16.dp))
            Text(
                stringResource(R.string.rate_complaints),
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.height(10.dp))
            rep.complaints.forEach { ComplaintRow(it) }
        }
        Spacer(Modifier.height(32.dp))
    }
}

@Composable
private fun Summary(rep: Reputation) {
    // **ومن لا يُقيَّم دورُه لا تُعرض له بطاقةٌ صفريّة** — «٠٫٠ من ٥»
    // يُقرأ حكماً عليه، ولم يفعل شيئا.
    if (!rep.rated || rep.rating.count == 0) {
        Text(stringResource(R.string.rate_none_yet), color = InkMuted)
        return
    }

    // ══════════════════════════════════════════════════════════════════
    // **ثلاثةُ مربّعاتٍ — كما في الويب حرفيّا**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «أنت ما كتبتَ مربّع عدد التقييمات
    //  ومتوسّط التقييم وتقييمٌ مستقرٌّ أو ممتازٌ أو جيّدٌ أو منخفض —
    //  لازم في حالاتٌ تلقائيّة».)
    //
    // **والأسماءُ من الويب نفسِها** — «متوسط التقييم» و«عدد التقييمات»
    // و«الاتجاه»، **وثلاثُ حالاتٍ يحسبها المحرّك** بمقارنة متوسّط
    // الثلاثين يوماً بمتوسّط الكلّ: في تحسّن · مستقرّ · في انخفاض.
    //
    // **والحسابُ في المحرّك لا هنا**: شاشتان تحسبان الاتّجاه بنفسيهما
    // **تختلفان يومَ يتبدّل حدُّ المقارنة** — فيقرأ في هاتفه «مستقرّ»
    // وفي لوحته «في انخفاض».
    val trend = when (rep.rating.trend) {
        "up" -> R.string.rate_trend_up
        "down" -> R.string.rate_trend_down
        else -> R.string.rate_trend_flat
    }
    val trendColor = when (rep.rating.trend) {
        "up" -> StateGreen
        "down" -> StateRed
        else -> InkMuted
    }

    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        StatBox(
            label = stringResource(R.string.rate_avg),
            value = "%.1f".format(java.util.Locale.US, rep.rating.avg),
            color = BrandOrange,
            modifier = Modifier.weight(1f),
        )
        StatBox(
            label = stringResource(R.string.rate_total),
            value = rep.rating.count.toString(),
            modifier = Modifier.weight(1f),
        )
        StatBox(
            label = stringResource(R.string.rate_trend),
            value = stringResource(trend),
            color = trendColor,
            modifier = Modifier.weight(1f),
        )
    }
    Spacer(Modifier.height(10.dp))
    Text(
        stringResource(R.string.rate_hint),
        color = InkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
}

/**
 * **مربّعُ رقمٍ باسمه** — نظيرُ `StatCard` في الويب.
 *
 * **والاسمُ فوق الرقم لا تحته**: العينُ تمسح صفّاً من المربّعات
 * **فتقرأ الأرقامَ على استقامةٍ واحدة**، والاسمُ يُقرأ حين يُسأل عنه.
 *
 * **والاتّجاهُ كلمةٌ لا رقم** — فيُصغَّر خطُّه ليقع في العرض نفسِه بلا
 * أن يُقصّ.
 */
@Composable
private fun StatBox(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    color: androidx.compose.ui.graphics.Color = com.rahalgo.design.InkDeep,
) {
    Column(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(InkMuted.copy(alpha = 0.07f))
            .padding(horizontal = 10.dp, vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = label,
            color = InkMuted,
            style = MaterialTheme.typography.bodySmall,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = value,
            color = color,
            style = if (value.length > 4) {
                MaterialTheme.typography.titleSmall
            } else {
                MaterialTheme.typography.headlineSmall
            },
            textAlign = TextAlign.Center,
            maxLines = 1,
        )
    }
}

@Composable
private fun ReviewRow(r: Review) {
    Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            // **والنجومُ تُرسم لا تُكتب** — «٣ من ٥» تُقرأ، **وثلاثُ
            // نجومٍ تُرى.**
            repeat(5) { i ->
                Icon(
                    painter = painterResource(R.drawable.ic_star),
                    contentDescription = null,
                    tint = if (i < r.stars) BrandOrange else InkMuted.copy(alpha = 0.30f),
                    modifier = Modifier.size(15.dp),
                )
            }
            Spacer(Modifier.size(8.dp))
            Text(
                text = "#" + r.orderNumber,
                color = InkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // ══════════════════════════════════════════════════════════════
        // **ولا تعليقَ يُعرض — نجومٌ وحدَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «التقييم نلغي منه الردّ أو التعليق،
        //  خلص مجرّد تقييمٍ وفقط».)
        //
        // **والنجومُ حكمٌ لا خصومة**: كلمةٌ قاسيةٌ يقرؤها السائقُ في
        // هاتفه وهو يقود **لا يستطيع الردَّ عليها** — فتشغله ولا
        // تُصلح شيئا.
        //
        // **والتعليقُ يبقى في المحرّك** لمن يحقّق في شكوى — إنّما لا
        // يُعرض هنا.
        Spacer(Modifier.height(4.dp))
        // **ومن أعطاها وأين** — (قرارُ المالك: «فيعرف ما هي التقييمات
        // التي حصل عليها ومن أين»).
        Text(
            text = listOf(r.customerName, r.merchantName).filter { it.isNotEmpty() }
                .joinToString(" · "),
            color = InkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(8.dp))
        HorizontalDivider()
    }
}

@Composable
private fun ComplaintRow(c: ComplaintBrief) {
    Column(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text("#" + c.number, style = MaterialTheme.typography.titleSmall)
            Text(c.status, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Text(c.subject, style = MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
    }
}
