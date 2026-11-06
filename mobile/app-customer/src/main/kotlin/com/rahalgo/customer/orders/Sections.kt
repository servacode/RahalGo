package com.rahalgo.customer.orders

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.OrderSection
import com.rahalgo.ui.Chip

/**
 * **نصُّ حالِ القسم** — دالّةٌ صرفةٌ تُختبر بلا واجهة.
 *
 * **وحالٌ لا نعرفها تُقرأ «قيد المعالجة» لا رمزَها** (كباقي الشاشة).
 */
@StringRes
internal fun sectionStatusRes(status: String): Int = when (status) {
    "pending" -> R.string.sec_pending
    "accepted" -> R.string.sec_accepted
    "picked_up" -> R.string.sec_picked_up
    "unavailable" -> R.string.sec_unavailable
    else -> R.string.tik_progress
}

/** **أيُّ القسم لا يُصنع؟** — بنودُه تُعرض مشطوبةً باهتة. */
internal fun sectionUnavailable(section: OrderSection): Boolean = section.status == "unavailable"

/**
 * **عدد المحلّات كما يُعرض** — من الأقسام نفسِها. و`null` يعني لا كتلةَ:
 * الحقلُ غائبٌ (طلبٌ عاديّ) أو قسمٌ واحدٌ لا يُعدّ طلباً متعدّداً.
 */
internal fun sectionsCount(sections: List<OrderSection>?): Int? =
    sections?.size?.takeIf { it >= 2 }

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«طلبك من ٣ محلات» — أقسامُ الطلب المتعدّد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بلا اسمِ متجرٍ ولا معرّفه** (P-1) — «القسم ١» و«القسم ٢» وحالُ كلٍّ منهما.
 * **والحقلُ غائبٌ في الطلب العاديّ فلا تُرسم شيئاً.** والأرقامُ غربيّةٌ كباقي الشاشة.
 */
@Composable
internal fun OrderSectionsBlock(sections: List<OrderSection>?) {
    val n = sectionsCount(sections) ?: return
    Spacer(Modifier.height(10.dp))
    HorizontalDivider()
    Spacer(Modifier.height(8.dp))
    Text(
        text = stringResource(R.string.sec_title, n.toString()),
        color = Rahal.colors.ink,
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.titleSmall,
    )
    sections.orEmpty().sortedBy { it.seq }.forEach { s ->
        Spacer(Modifier.height(8.dp))
        val gone = sectionUnavailable(s)
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.sec_part, s.seq.toString()),
                color = if (gone) Rahal.colors.inkMuted else Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            Chip(
                stringResource(sectionStatusRes(s.status)),
                when (s.status) {
                    "unavailable" -> Rahal.colors.danger
                    "picked_up" -> Rahal.colors.success
                    "accepted" -> Rahal.colors.brand
                    else -> Rahal.colors.accent
                },
            )
        }
        Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            s.items.forEach { it ->
                Text(
                    text = it.name + " × " + it.qty,
                    color = if (gone) Rahal.colors.inkMuted else Rahal.colors.ink,
                    style = MaterialTheme.typography.bodySmall,
                    textDecoration = if (gone) TextDecoration.LineThrough else null,
                )
            }
        }
    }
}
