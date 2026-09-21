package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تفصيلُ الشكوى وخيطُ ردودها — لا دردشة** (`SUP-013`/`SUP-014`، PRQ-2)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان الزبونُ يفتح شكوى ثمّ لا يرى جوابَ المنصّة**: الردودُ تحت
 * `‎/admin/tickets` وحدَها، **وشاشتُه تقول رقمَها لا حالَها.** فيُعرَض له
 * الخيطُ كاملاً — **ما كتبه هو وما ردّت به المنصّة، مفصولَين بوضوح** —
 * ويردّ ما دامت مفتوحة.
 *
 * **وليست محادثةً حيّة** (قرارُ المالك): لا مؤشّرَ كتابةٍ، ولا حضورٍ، ولا
 * إيصالاتِ قراءة، ولا مرفقات — **خيطُ شكوًى يُقرأ ويُردُّ عليه لا أكثر.**
 *
 * **والتمييزُ من الخادم لا من مقارنةِ معرّفات** (`mine`): **فلا يُكشَف
 * للزبون معرّفُ موظّفٍ ردّ**، ولا يُخطئ في نسبة ردٍّ إلى صاحبه.
 *
 * **وهي عرضٌ صافٍ**: الحالُ والردودُ والمسوّدةُ تُمرَّر إليها، والنداءاتُ
 * تُرفَع بردّ نداء — **فتُقاس بلا جهاز.** وحالُ التحميل والفشلِ يحرسهما
 * `LoadState` عند المنادي كسائر الأقسام.
 */

/**
 * **أتُقبَل الردود؟** — المحلولةُ لا يُردُّ عليها.
 *
 * **والمحرّكُ هو الحاكم** (`support.Reply` يردّ `409 ticket_resolved` على
 * «resolved»)، **والشاشةُ تتبعه**: تُخفي الحقلَ وتقول لماذا بدل حقلٍ باهتٍ
 * يُضغط فيُردّ. **وحالتا «open» و«in_progress» تقبلان.**
 */
fun ticketCanReply(status: String): Boolean = status != "resolved"

/** **رسالةٌ في الخيط** — نصُّها، أهي لصاحب الشكوى، ومتى. */
data class TicketMessage(
    val body: String,
    /** **`mine` من الخادم** — أصاحبُ الشكوى كتبها أم المنصّة. */
    val mine: Boolean,
    /** **وقتُها `ISO`** — يُصاغ هنا بتوقيت دمشق. */
    val at: String,
)

@Composable
fun TicketThreadScreen(
    subject: String,
    status: String,
    createdAt: String,
    messages: List<TicketMessage>,
    /** **لا ردودَ بعد** — فراغٌ يُقال لا يُترك صامتاً. */
    emptyReplies: String,
    /** **أيُقبَل الردُّ؟** — مغلقةٌ لا يُردُّ عليها. */
    canReply: Boolean,
    draft: String,
    onDraftChange: (String) -> Unit,
    onSend: () -> Unit,
    sending: Boolean,
    /** **خطأُ الإرسال** — كـ«محلولةٌ لا يُردُّ عليها»؛ فارغٌ يعني لا خطأ. */
    sendError: String,
    onBack: () -> Unit,
) {
    Screen {
        // **رجوعٌ ظاهرٌ في الشاشة** — لا يُعتمَد على زرّ النظام وحدَه.
        RahalTextButton(onClick = onBack) {
            Text("‹ " + stringResource(R.string.act_back))
        }
        Spacer(Modifier.height(6.dp))

        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                subject,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.widthIn(max = 260.dp),
            )
            Chip(ticketStatusText(status), ticketStatusColor(status))
        }
        val created = whenText(createdAt)
        if (created.isNotBlank()) {
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.tik_created, created),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(14.dp))

        // ── الخيطُ — أو فراغٌ يقول إنّه فراغ ─────────────────────────────
        if (messages.isEmpty()) {
            Empty(emptyReplies)
        } else {
            messages.forEach { m ->
                TicketBubble(m)
                Spacer(Modifier.height(8.dp))
            }
        }

        Spacer(Modifier.height(10.dp))

        // ── الردُّ — إن كانت مفتوحة ──────────────────────────────────────
        if (canReply) {
            if (sendError.isNotBlank()) {
                Note(sendError, Rahal.colors.danger)
            }
            OutlinedTextField(
                value = draft,
                onValueChange = onDraftChange,
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text(stringResource(R.string.tik_reply_hint)) },
                minLines = 2,
                enabled = !sending,
            )
            Spacer(Modifier.height(10.dp))
            RahalButton(
                onClick = onSend,
                modifier = Modifier.fillMaxWidth(),
                enabled = !sending && draft.isNotBlank(),
            ) { Text(stringResource(R.string.tik_send)) }
        } else {
            // **مغلقةٌ — يُقال صريحاً لماذا لا حقلَ للردّ**، لا حقلٌ باهتٌ صامت.
            Note(stringResource(R.string.tik_closed), Rahal.colors.inkMuted)
        }
    }
}

/**
 * **فقاعةُ رسالةٍ في خيط الشكوى** — بلغة التصميم عينِها.
 *
 * **صاحبُها في جهةٍ والمنصّةُ في الأخرى** (`Start`/`End` تتبع الاتجاهَ
 * فتصحّ في العربيّة)، **ولونٌ ولافتةٌ باسم الكاتب** — **فلا يُخلَط ردُّه
 * بردّ المنصّة.** ولا علاماتِ قراءةٍ ولا حضور: خيطُ شكوًى لا محادثة.
 */
@Composable
private fun TicketBubble(m: TicketMessage) {
    Box(
        Modifier.fillMaxWidth(),
        contentAlignment = if (m.mine) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Column(
            Modifier
                .widthIn(max = 300.dp)
                .clip(Rahal.shape.md)
                .background(if (m.mine) Rahal.colors.brand else Rahal.colors.bubble)
                .padding(horizontal = 14.dp, vertical = 10.dp),
        ) {
            Text(
                text = stringResource(if (m.mine) R.string.tik_you else R.string.tik_support),
                color = if (m.mine) Rahal.colors.onBrand.copy(alpha = 0.85f) else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(3.dp))
            Text(
                text = m.body,
                color = if (m.mine) Rahal.colors.onBrand else Rahal.colors.ink,
            )
            val at = chatTime(m.at)
            if (at.isNotBlank()) {
                Spacer(Modifier.height(3.dp))
                Text(
                    text = at,
                    color = if (m.mine) Rahal.colors.onBrand.copy(alpha = 0.75f) else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
                )
            }
        }
    }
}
