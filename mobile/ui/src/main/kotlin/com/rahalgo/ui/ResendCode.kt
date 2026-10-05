package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أعد إرسال الرمز — بعد دقيقتين، وفي كلّ موضعٍ ينتظر رمزا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٥.)
 *
 * **وكان من لم يصله رمزُه لا يملك إلّا الرجوع** — يعود إلى الرقم ويطلب
 * من جديد، **أو يظنّ الرمزَ لن يأتي فيترك.** والمواضعُ أربعة: إنشاءُ
 * الحساب، والدخولُ برمز، واستعادةُ كلمة المرور، وتوثيقُ واتساب.
 *
 * # ولماذا دقيقتان لا فورا
 *
 * **زرٌّ يُضغط كلَّ ثانيةٍ يستهلك حصّةَ الرقم في المحرّك** — فيُردّ
 * `rate_limited` على الطلب الذي كان سيصل. **والعدّادُ يقول له: رمزُك
 * في الطريق، انتظر قليلا.**
 *
 * # وما لا يملكه هذا المكوّن
 *
 * **لحظةُ الإرسال تأتي من النموذج** (`sentAt`) — **فلا تبدأ المهلةُ إلّا
 * بنجاح الطلب.** وطلبٌ رُدّ بـ`rate_limited` لا يحرّكها، **فيبقى الزرُّ
 * متاحاً بعد انقضائها** ويُقال سببُ الردّ في رسالة الشاشة كما هو.
 *
 * @param sentAt لحظةُ آخر إرسالٍ ناجح (بالمللي من الحقبة) — وصفرٌ يعني الزرَّ فورا.
 * @param onResend يطلب الرمزَ بالنداء نفسِه — **والشاشةُ تمحو الحقلَ قبله.**
 */
@Composable
fun ResendCodeRow(
    sentAt: Long,
    busy: Boolean,
    onResend: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var now by remember { mutableLongStateOf(System.currentTimeMillis()) }
    // **ويُعاد العدُّ مع كلّ إرسالٍ ناجح** — مفتاحُه اللحظةُ نفسُها.
    LaunchedEffect(sentAt) {
        now = System.currentTimeMillis()
        while (resendSecondsLeft(sentAt, now) > 0) {
            kotlinx.coroutines.delay(1000)
            now = System.currentTimeMillis()
        }
    }
    val left = resendSecondsLeft(sentAt, now)

    Row(
        modifier.fillMaxWidth().heightIn(min = 40.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (left > 0) {
            Icon(
                painter = painterResource(R.drawable.ic_time),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(14.dp),
            )
            Spacer(Modifier.size(6.dp))
            Text(
                text = stringResource(R.string.resend_wait, resendClock(left)),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        } else {
            RahalTextButton(onClick = onResend, enabled = !busy) {
                Icon(
                    painter = painterResource(R.drawable.ic_history),
                    contentDescription = null,
                    modifier = Modifier.size(16.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(stringResource(R.string.resend_code), fontWeight = FontWeight.Medium)
            }
        }
    }
}

/** **مهلةُ إعادة الإرسال** — دقيقتان (قرارُ المالك ٢٠٢٦-١٠-٠٥). */
const val RESEND_WINDOW_MS: Long = 120_000L

/**
 * **كم ثانيةً بقيت قبل أن يُتاح الزرّ** — دالّةٌ صافيةٌ تُقاس بلا جهاز.
 *
 * **وتُقرَّب إلى الأعلى**: «٠:٠٠» والزرُّ لم يظهر بعد يُقرأ عطبا. **وصفرٌ
 * في `sentAt` يعني لم يُرسَل شيء** فالزرُّ متاح، **وساعةٌ رجعت إلى الوراء
 * لا تُطيل المهلةَ فوق دقيقتين.**
 */
fun resendSecondsLeft(sentAt: Long, now: Long, windowMs: Long = RESEND_WINDOW_MS): Int {
    if (sentAt <= 0L) return 0
    val leftMs = (sentAt + windowMs - now).coerceIn(0L, windowMs)
    return ((leftMs + 999L) / 1000L).toInt()
}

/**
 * **الثواني بصيغة `m:ss`** — «2:00» ثمّ «1:59» … «0:01».
 *
 * **وبأرقامٍ غربيّةٍ مهما كانت لغةُ الجهاز** (`Locale.ROOT`) — كسائر
 * أرقام التطبيق (`WesternDigits.kt`).
 */
fun resendClock(seconds: Int): String {
    val s = seconds.coerceAtLeast(0)
    return String.format(java.util.Locale.ROOT, "%d:%02d", s / 60, s % 60)
}
