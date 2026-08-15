package com.rahalgo.ui

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ الشكوى — اسمُها ولونُها في موضعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والحالَ يكتبها المحرّك** (`support`) — **واسمُها في شاشتين يفترق
 * يومَ يُصحَّح أحدُهما**، فيقرأ السائقُ «قيد المعالجة» ويقرأ الزبونُ
 * «مفتوحة» لشكوًى واحدة.
 *
 * **ولا حالَ تُخترع هنا**: المحرّكُ يكتب ثلاثاً، **ورابعةٌ تُخترع اسمٌ
 * ميّتٌ يوهم أنّ الحالةَ مغطّاة.**
 */
@Composable
fun ticketStatusText(status: String): String = when (status) {
    "open" -> stringResource(R.string.tik_open)
    "in_progress" -> stringResource(R.string.tik_progress)
    "resolved" -> stringResource(R.string.tik_resolved)
    else -> status
}

@Composable
fun ticketStatusColor(status: String): Color = when (status) {
    "resolved" -> Rahal.colors.success
    else -> Rahal.colors.accent
}
