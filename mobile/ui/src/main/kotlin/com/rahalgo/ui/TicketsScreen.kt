package com.rahalgo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
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
    /** **معرّفُه للفتح** — يُمرَّر إلى `onOpen`، ولا يُعرَض. فارغٌ يعني لا فتح. */
    val id: String = "",
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
    /** **يُفتَح صفٌّ فيُقرأ خيطُه** — فارغٌ يبقي الصفوفَ غيرَ قابلةٍ للفتح. */
    onOpen: ((TicketRow) -> Unit)? = null,
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
            Spacer(Modifier.height(Rahal.space.md))
        }

        val list = if (againstMe != null && showAgainst) againstMe else mine
        if (list.isEmpty()) {
            Empty(if (againstMe != null && showAgainst) againstMeEmpty else mineEmpty)
            return@Screen
        }
        // **كلُّ بلاغٍ بطاقتُه** (مراجعةُ التصميم ٢٠٢٦-١١-٠٩) لا شريحةٌ بفواصل.
        list.forEach { row ->
            RahalCard(padding = Rahal.space.none) { TicketLine(row, onOpen) }
            Spacer(Modifier.height(Rahal.space.md))
        }
    }
}

@Composable
private fun TicketLine(row: TicketRow, onOpen: ((TicketRow) -> Unit)?) {
    // **ويُفتَح إن كان له معرّفٌ ومُستقبِل** — وإلّا بقي سطراً يُقرأ (المتجر/السائق).
    val open = if (onOpen != null && row.id.isNotBlank()) ({ onOpen(row) }) else null
    // **الموضوعُ أوّلاً وحالُه في طرفه** (إعادةُ التصميم ٢٠٢٦-١١-١١) — ثمّ رقمُ البلاغ
    // والطلبُ والتاريخ سطراً خافتاً: **ما يُبحث عنه يُقرأ أوّلا، وما يُرجَع إليه يُقرأ ثانيا.**
    Column(
        Modifier
            .fillMaxWidth()
            .then(if (open != null) Modifier.clickable(onClick = open) else Modifier)
            .padding(Rahal.space.md),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.xs),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            verticalAlignment = Alignment.Top,
        ) {
            Text(
                row.title,
                style = Rahal.type.cardTitle,
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
            Chip(ticketStatusText(row.status), ticketStatusColor(row.status))
        }
        val meta = buildList {
            add(row.key)
            if (row.orderNumber.isNotBlank()) add(stringResource(R.string.tik_on_order, row.orderNumber))
            if (row.date.isNotBlank()) add(row.date)
        }.filter { it.isNotBlank() }.joinToString("  ·  ")
        if (meta.isNotEmpty()) {
            Text(meta, style = Rahal.type.meta, color = Rahal.colors.inkMuted)
        }
        // **والحلُّ يُقرأ** — ومن أبلغ ولم يُقَل له ما وقع يظنّ بلاغَه أُهمل.
        if (row.resolution.isNotBlank()) {
            Spacer(Modifier.height(Rahal.space.xxs))
            Note(row.resolution, Rahal.colors.inkMuted)
        }
    }
}
