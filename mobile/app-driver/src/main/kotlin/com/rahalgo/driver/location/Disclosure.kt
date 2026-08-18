package com.rahalgo.driver.location

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الإفصاحُ الصريح عن الموقع — قبل أن يسأل النظام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نبدأ بالإغلاق واحدةً تلو الأخرى» —
 *  والخامسة: شرطا غوغل.)
 *
 * # ولماذا شاشةٌ قائمةٌ بذاتها
 *
 * **غوغل تشترط على كلّ تطبيقٍ يطلب الموقعَ في الخلفيّة إفصاحاً ظاهراً
 * داخل التطبيق** — لا سطراً في سياسة الخصوصيّة ولا في الشروط: **يجب أن
 * يُعرض وحدَه، وأن يُقبل بفعلٍ صريح، وأن يسبق نافذةَ النظام.**
 *
 * **والتطبيقُ الذي يطلبه بلا إفصاحٍ يُرفض في المراجعة** — وهو أكثرُ ما
 * تُرَدّ به تطبيقاتُ التوصيل.
 *
 * # وأربعةُ أشياءَ يجب أن تُقال
 *
 * **١ · ما يُجمع** — موقعُك أنت.
 * **٢ · متى** — أثناء الورديّة، **وحتّى والشاشةُ مطفأةٌ والتطبيقُ
 * مغلق.** وهذه أهمُّها وأكثرُها إسقاطاً للتطبيقات: **إن لم تُقل صراحةً
 * لم يُقبل الإفصاح.**
 * **٣ · لماذا** — ليُسنَد إليه أقربُ طلب، وليعرف الزبونُ أين طلبُه.
 * **٤ · ومتى يقف** — بإغلاق الورديّة. **وهو ما يجعل الإفصاحَ صادقاً**:
 * وعدٌ بحدٍّ، لا طلبٌ مفتوح.
 *
 * # وله أن يرفض
 *
 * **زرُّ الرفضِ بجانب القبول** — وإفصاحٌ بلا مخرجٍ ليس إفصاحاً.
 * **ويعمل التطبيقُ بلاه**، وتقف الطلباتُ وحدَها: يُقال ذلك ولا يُخفى.
 */
@Composable
fun LocationDisclosure(onAgree: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                text = stringResource(R.string.disc_title),
                fontWeight = FontWeight.Bold,
            )
        },
        text = {
            Column {
                Text(stringResource(R.string.disc_what))
                Spacer(Modifier.height(10.dp))
                // **والخلفيّةُ تُقال بأوضح ما يمكن** — «والشاشةُ مطفأة».
                Text(
                    text = stringResource(R.string.disc_background),
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.accent,
                )
                Spacer(Modifier.height(10.dp))
                Text(stringResource(R.string.disc_why), color = Rahal.colors.inkMuted)
                Spacer(Modifier.height(10.dp))
                Text(stringResource(R.string.disc_stop), color = Rahal.colors.inkMuted)
            }
        },
        confirmButton = {
            RahalButton(onClick = onAgree) { Text(stringResource(R.string.disc_agree)) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.disc_deny)) }
        },
    )
}

/**
 * **سطرٌ في «حسابي» يقول ما يُجمع** — ولمن أراد أن يقرأ لا لمن يُسأل.
 *
 * **والإفصاحُ يُعرض مرّةً عند الطلب** — ومن أراد مراجعتَه بعد شهرٍ لا
 * يجد إليه سبيلاً، **فيبقى في مكانٍ ثابتٍ يُفتح متى شاء.**
 */
@Composable
fun LocationNotice(modifier: Modifier = Modifier) {
    Column(
        modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                text = stringResource(R.string.disc_title),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(R.string.disc_what) + " " +
                stringResource(R.string.disc_background) + " " +
                stringResource(R.string.disc_why) + " " +
                stringResource(R.string.disc_stop),
            style = MaterialTheme.typography.bodySmall,
            color = Rahal.colors.inkMuted,
        )
    }
}
