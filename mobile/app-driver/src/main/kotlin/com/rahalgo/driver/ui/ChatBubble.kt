package com.rahalgo.driver.ui

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.shared.model.ChatMessage

/**
 * ══════════════════════════════════════════════════════════════════════
 * **فقاعةُ الرسالة — شكلٌ واحدٌ حيثما تُقرأ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك ٢٠٢٦-٠٨-١٣: «ركّز جيّداً على المركزيّة بكلّ شيء».)
 *
 * **كانت في `trip/ChatSheet.kt` خاصّةً بها** — ثمّ لزمت «دردشاتي
 * السابقة». **ونسخُها يعني حديثاً يُقرأ بشكلين**: أزرقُ في الرحلة
 * ورماديٌّ في السجلّ، **والرسالةُ هي هي.**
 */

/**
 * **فقاعةُ رسالة** — لي في جهةٍ ولغيري في الأخرى.
 *
 * **وعلامةُ القراءة على ما كتبتُه أنا وحدَه** (قرارُ المالك ٢٠٢٦-٠٨-١٢):
 * «قُرئت» على رسالة الآخر خبرٌ عنّي أنا، **وأنا أراه الآن.**
 */
@Composable
fun ChatBubble(message: ChatMessage) {
    val mine = message.mine
    Box(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        contentAlignment = if (mine) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Column(
            Modifier
                .clip(RoundedCornerShape(14.dp))
                .background(if (mine) Rahal.colors.brand else Rahal.colors.bubble)
                .padding(horizontal = 14.dp, vertical = 10.dp),
            horizontalAlignment = Alignment.End,
        ) {
            Text(
                text = message.body,
                color = if (mine) Rahal.colors.onBrand else Rahal.colors.ink,
            )
            Spacer(Modifier.height(2.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = chatTime(message.createdAt),
                    color = if (mine) Rahal.colors.onBrand.copy(alpha = 0.75f) else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
                )
                if (mine) {
                    Spacer(Modifier.size(5.dp))
                    Text(
                        text = if (message.readAt != null) "✓✓" else "✓",
                        color = Rahal.colors.onBrand.copy(
                            alpha = if (message.readAt != null) 1f else 0.55f,
                        ),
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
            }
        }
    }
}

/** **سطرُ اليوم** — يفصل ما كُتب أمس عمّا كُتب اليوم. */
@Composable
fun ChatDayChip(day: String) {
    Box(Modifier.fillMaxWidth().padding(vertical = 8.dp), contentAlignment = Alignment.Center) {
        Text(
            text = day,
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier
                .clip(RoundedCornerShape(10.dp))
                .background(Rahal.colors.bubble)
                .padding(horizontal = 10.dp, vertical = 3.dp),
        )
    }
}

/**
 * **الساعة بتوقيت دمشق** — لا بتوقيت الجهاز.
 *
 * **وسائقٌ ضبط ساعتَه غلطاً أو سافر** يقرأ أوقاتاً لا تطابق ما تقرؤه
 * الإدارة، **وحُجّةٌ بساعتين مختلفتين ليست حجّة.**
 */
fun chatTime(iso: String): String = chatAt(iso)?.format(HOUR).orEmpty()

/** **اليوم** — «اليوم» و«أمس» ثمّ التاريخ. */
@Composable
fun chatDay(iso: String): String {
    val at = chatAt(iso) ?: return ""
    val today = java.time.LocalDate.now(DAMASCUS)
    return when (at.toLocalDate()) {
        today -> stringResource(R.string.chat_today)
        today.minusDays(1) -> stringResource(R.string.chat_yesterday)
        else -> at.format(DAY)
    }
}

private val DAMASCUS: java.time.ZoneId = java.time.ZoneId.of("Asia/Damascus")
private val HOUR = java.time.format.DateTimeFormatter.ofPattern("HH:mm")
private val DAY = java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd")

/** **ونصٌّ لا يُقرأ لا يُسقط الشاشة** — يُترك فارغا. */
private fun chatAt(iso: String): java.time.ZonedDateTime? = runCatching {
    java.time.OffsetDateTime.parse(iso).atZoneSameInstant(DAMASCUS)
}.getOrNull()
