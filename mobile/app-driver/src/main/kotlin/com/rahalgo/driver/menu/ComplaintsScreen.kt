package com.rahalgo.driver.menu

import androidx.compose.foundation.layout.Arrangement
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.Card
import com.rahalgo.driver.ui.Chip
import com.rahalgo.driver.ui.Empty
import com.rahalgo.driver.ui.LoadState
import com.rahalgo.driver.ui.Screen
import com.rahalgo.driver.ui.ScreenTitle
import com.rahalgo.driver.ui.whenText
import com.rahalgo.shared.model.ComplaintBrief

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشكاوى والبلاغات — ما رُفع عليّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # لماذا يراها
 *
 * **كانت تصل غرفةَ العمليات وحدَها.** ومن اشتُكي عليه ولا يعلم **لا
 * يُصلح شيئاً** — يُخصم منه يوماً فيُفاجأ، **ويظنّ الظلمَ حيث كان خبر.**
 *
 * **والشكوى تُقرأ لا تُبلَّغ فحسب**: «تأخّر ساعةً» و«لم يردّ على الهاتف»
 * سببان مختلفان، **وأحدُهما يُصلَح بترتيب اليوم والآخر بشحن الهاتف.**
 *
 * # ولا اسمَ لصاحبها
 *
 * **تلك خصومةٌ تُحقَّق، وكشفُ اسمِه يفتح باباً للردّ عليه** — والذي
 * يخصُّ السائقَ ما وقع لا من قاله.
 *
 * # وبلاغاتُه هو ليست هنا
 *
 * **بلاغُ السائق على متجرٍ يُفتح باسم زبون الطلب** (`support/driver_report.go`)
 * — **فلا بابَ في المحرّك يردّ ما رفعه هو.** وهي ثغرةٌ مقيسة، مكتوبةٌ في
 * تقرير الجرد، **ولا تُخترع لها شاشةٌ تعرض فراغا.**
 */
@Composable
fun ComplaintsScreen(vm: SectionsViewModel) {
    val rep = vm.reputation
    if (rep == null) {
        LoadState(vm.busy, vm.error) { vm.open(MenuItem.Tickets, force = true) }
        return
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_tickets),
            stringResource(R.string.tik_hint),
        )
        if (rep.complaints.isEmpty()) {
            // **وفراغُها خبرٌ سارّ** — يُقال بلونه: لا شكوى عليك.
            Spacer(Modifier.height(20.dp))
            Text(
                stringResource(R.string.tik_none),
                color = Rahal.colors.success,
                fontWeight = FontWeight.Bold,
            )
            Empty(stringResource(R.string.tik_none_hint))
            return@Screen
        }
        rep.complaints.forEach { ComplaintCard(it) }
    }
}

@Composable
private fun ComplaintCard(c: ComplaintBrief) {
    Spacer(Modifier.height(10.dp))
    Card(tone = if (c.status == "open") Rahal.colors.accent else Rahal.colors.inkMuted) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "#" + c.number.toString(),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall,
            )
            Chip(text = ticketStatus(c.status), color = statusColor(c.status))
        }
        Spacer(Modifier.height(6.dp))
        Text(c.subject, style = MaterialTheme.typography.bodyLarge)
        Spacer(Modifier.height(4.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            c.orderNumber?.let {
                Text(
                    text = stringResource(R.string.tik_on_order, it.toString()),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            Text(
                text = whenText(c.createdAt),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/** **حالُ الشكوى بعربيّة** — والمجهولُ يُعرض برمزه ليُبلَّغ عنه. */
@Composable
private fun ticketStatus(status: String): String = when (status) {
    "open" -> stringResource(R.string.tik_open)
    "in_progress" -> stringResource(R.string.tik_progress)
    "resolved" -> stringResource(R.string.tik_resolved)
    else -> status
}

/** **ولا حالَ رابعةً** — المحرّكُ يكتب ثلاثاً (`support`)، والمعجمُ
 *  يسمّي ثلاثاً. **ورابعةٌ تُخترع هنا اسمٌ ميّتٌ يوهم أنّ الحالةَ
 *  مغطّاة.** */
@Composable
private fun statusColor(status: String) = when (status) {
    "resolved" -> Rahal.colors.success
    else -> Rahal.colors.accent
}
