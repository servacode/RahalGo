package com.rahalgo.driver.ui

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما بقي من مهلة العرض — عدّادٌ واحدٌ حيثما يُعرض طلب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يقبل بشكلٍ سريع مع عدّاد وقت».)
 *
 * # ولماذا مركزيّ
 *
 * **كان في شاشة الطلبات وحدَها** — ثمّ لزم لافتةَ «طلبٌ على طريقك»
 * في الرحلة. **وعرضان لمهلةٍ واحدةٍ يفترقان**: يُصلَح تنسيقُ الدقائق
 * في أحدهما، **ويبقى الآخرُ يعدّ بالثواني وحدَها.**
 *
 * # وما يقوله اللون
 *
 * **عشرُ ثوانٍ فأقلّ تصير برتقاليّة** — ومن يقود يقرأ اللونَ قبل
 * الرقم. **والمهلةُ التي تنقضي بلا إنذارٍ تُقرأ عطباً**: ضغط فوجد
 * الطلبَ ذهب.
 */
@Composable
fun Countdown(
    expiresAt: String,
    onExpired: () -> Unit,
    onColor: Color = Color.Unspecified,
) {
    val end = remember(expiresAt) {
        runCatching { java.time.Instant.parse(expiresAt).toEpochMilli() }.getOrNull()
    } ?: return

    var left by remember(expiresAt) { mutableStateOf(end - System.currentTimeMillis()) }
    LaunchedEffect(expiresAt) {
        while (left > 0) {
            kotlinx.coroutines.delay(1000)
            left = end - System.currentTimeMillis()
        }
        onExpired()
    }

    if (left <= 0) {
        Text(
            text = stringResource(R.string.offer_expired),
            color = if (onColor == Color.Unspecified) Rahal.colors.inkMuted else onColor,
        )
        return
    }

    val seconds = (left / 1000).toInt()
    // **واللونُ على الأرض الملوّنة يُمرَّر** — أرضٌ سماويّةٌ تحت نصٍّ
    // سماويٍّ لا تُقرأ.
    val tint = when {
        onColor != Color.Unspecified && seconds > 10 -> onColor
        seconds <= 10 -> Rahal.colors.accent
        else -> Rahal.colors.brand
    }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(R.drawable.ic_time),
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = stringResource(R.string.offer_left, "%d:%02d".format(seconds / 60, seconds % 60)),
            color = tint,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}
