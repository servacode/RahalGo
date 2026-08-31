package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشكاوى والبلاغات — شاشةٌ واحدةٌ لكلّ من يشتكي ويُشتكى عليه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «الشكاوى والبلاغات بالتطبيقات الثلاثة
 * مختلفة · أفضلُ شكلٍ هو شكلُ الشكاوى في المتجر · **وأساساً يجب أن يكون
 * بشكلٍ مركزيّ** · وليس كلُّ تطبيقٍ يأخذ شكلاً مختلفاً».)
 *
 * # وثلاثةُ أشكالٍ كانت لشيءٍ واحد
 *
 * **المتجر**: تبويبان — واحدةٌ تُعرض في وقت.
 * **السائق**: قسمان فوق بعضهما في تمريرةٍ واحدة.
 * **الزبون**: قائمةٌ واحدة.
 *
 * **والقسمان فوق بعضهما يجعلانه يمرّر ليعرف أيَّهما يقرأ** — **وخلطُ
 * «عليّ» بـ«رفعتُها» يقلب المعنى**: يقرأ شكوًى عليه على أنّها شكواه
 * فيطمئنّ، أو العكسَ فيفزع.
 *
 * # ولماذا تبويبان لا قائمةٌ بعمود
 *
 * **الفصلُ يجب أن يسبق القراءة لا يرافقها.** ومن ضغط تبويباً **عرف قبل
 * أن يقرأ سطراً أيَّ الوجهين ينظر إليه**، ولا يعتمد على لونٍ أو موضعٍ
 * في التمرير.
 *
 * # والفراغُ يُمدح ولا يُترك صامتاً
 *
 * **«لا شكاوى على متجرك — أحسنت»** غيرُ قائمةٍ فارغة. **ومن رأى فراغاً
 * بلا كلمةٍ ظنّ الشاشةَ لم تُحمَّل**، فيعيد ويعيد.
 *
 * # وما تملكه الشاشةُ وما لا تملكه
 *
 * **لا تعرف الشاشةُ أسبابَ البلاغ** — رموزُها تختلف بالدور: المتجرُ
 * يُبلّغ عن سائق، والسائقُ عن متجرٍ وزبون. **فيُسلَّم إليها العنوانُ
 * مترجَماً**، ولو ترجمتْ لَاحتاجت أن تعرف الأدوارَ كلَّها.
 *
 * **والحالُ وحدَها تُترجَم هنا** (`ticketStatusText`) — **لأنّها واحدةٌ
 * للجميع**: المحرّكُ يكتب `open` و`in_progress` و`resolved` لا غير.
 */

/**
 * **صفُّ شكوًى — بأقلّ ما يكفي لعرضه.**
 *
 * **ولا يحمل نموذجَ تطبيقٍ بعينه**: `MyReport` للمتجر والسائق،
 * و`Ticket` للزبون، **ولو أخذت الشاشةُ أحدَها لَما استعملها الآخر.**
 */
data class TicketRow(
    /** **رقمُه كما يُعرَف** — «#١٢» أو معرّفٌ نصّيّ. */
    val key: String,
    /** **العنوان مترجَماً** — سببُ البلاغ أو موضوعُ الشكوى. */
    val title: String,
    /** **رمزُ الحال من المحرّك** — يُترجَم هنا لا عند المنادي. */
    val status: String,
    /** رقمُ الطلب إن كانت على طلب. */
    val orderNumber: String = "",
    /** **ما فُصل به** — ومن أبلغ ولم يُقَل له ما وقع يظنّ بلاغَه أُهمل. */
    val resolution: String = "",
    /** **يومُها مكتوباً** — والمنادي يصوغه، فصياغةُ التاريخ عنده. */
    val date: String = "",
)

/**
 * **الشاشة — بتبويبين أو بقائمةٍ واحدة.**
 *
 * **ومن لا يُشتكى عليه لا يُعرض له تبويب**: الزبونُ يشتكي ولا يُشتكى
 * عليه في شاشته، **وتبويبٌ فارغٌ أبداً يُعلّم صاحبَه ألّا ينظر.**
 * فيُمرَّر `againstMe = null` فتصير قائمةً واحدة.
 */
@Composable
fun TicketsScreen(
    title: String,
    hint: String,
    mine: List<TicketRow>,
    mineEmpty: String,
    againstMe: List<TicketRow>? = null,
    againstMeEmpty: String = "",
) {
    // **والافتتاحُ على «عليّ»** — ما رُفع عليه أعجلُ ممّا رفعه:
    // **الأوّلُ قد يُنذَر به والثاني ينتظر جواباً.**
    var showAgainst by rememberSaveable { mutableStateOf(againstMe != null) }

    Screen {
        ScreenTitle(title, hint)
        if (againstMe != null) {
            SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                SegmentedButton(
                    selected = showAgainst,
                    onClick = { showAgainst = true },
                    shape = SegmentedButtonDefaults.itemShape(index = 0, count = 2),
                ) { Text(stringResource(R.string.tik_on_me)) }
                SegmentedButton(
                    selected = !showAgainst,
                    onClick = { showAgainst = false },
                    shape = SegmentedButtonDefaults.itemShape(index = 1, count = 2),
                ) { Text(stringResource(R.string.tik_mine)) }
            }
            Spacer(Modifier.height(12.dp))
        }

        val list = if (againstMe != null && showAgainst) againstMe else mine
        if (list.isEmpty()) {
            Empty(if (againstMe != null && showAgainst) againstMeEmpty else mineEmpty)
            return@Screen
        }
        Card {
            list.forEachIndexed { i, row ->
                if (i > 0) HorizontalDivider()
                TicketLine(row)
            }
        }
    }
}

@Composable
private fun TicketLine(row: TicketRow) {
    Column(Modifier.fillMaxWidth()) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(row.key, style = MaterialTheme.typography.labelMedium, color = Rahal.colors.inkMuted)
            Chip(ticketStatusText(row.status), ticketStatusColor(row.status))
        }
        Spacer(Modifier.height(4.dp))
        Text(row.title, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
        if (row.orderNumber.isNotBlank()) {
            Text(
                stringResource(R.string.tik_on_order, row.orderNumber),
                style = MaterialTheme.typography.bodySmall,
                color = Rahal.colors.inkMuted,
            )
        }
        // **والحلُّ يُقرأ** — ومن أبلغ ولم يُقَل له ما وقع يظنّ بلاغَه أُهمل.
        if (row.resolution.isNotBlank()) {
            Spacer(Modifier.height(4.dp))
            Note(row.resolution, Rahal.colors.inkMuted)
        }
        if (row.date.isNotBlank()) {
            Text(row.date, style = MaterialTheme.typography.labelSmall, color = Rahal.colors.inkMuted)
        }
        Spacer(Modifier.height(8.dp))
    }
}
