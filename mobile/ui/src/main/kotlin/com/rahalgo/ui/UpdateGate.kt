package com.rahalgo.ui

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ التحديث — لا تُغلق ولا يُرجع منها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «التطبيق يجب أن تظهر شاشةٌ تجبر المستخدمَ
 *  على تحديث التطبيق من غوغل بلاي أو من الرابط الخارجيّ — مو معقول
 *  التطبيق يتحدّث والمستخدم ما عنده خبر بالشي».)
 *
 * # ولماذا لا تُغلق
 *
 * **ونسخةٌ تخلّفت عن المحرّك تنكسر عند صاحبها بصمت**: نداءٌ يردّ حقلاً
 * لا تعرفه، وشاشةٌ تُرسم بلا بيانات. **ورسالةٌ تُغلق بضغطةٍ تُغلق
 * وتُنسى.**
 *
 * **والرجوعُ مُعترَضٌ كذلك** — **ومن ضغط الرجوعَ فوجد نفسَه في التطبيق
 * ظنّ أنّ التحديثَ اختياريّ.**
 *
 * # وبابٌ واحدٌ لا بابان
 *
 * **بلاي أوّلاً** — وهو ما نزّل منه. **فإن لم يكن على الجهاز** (وأجهزةٌ
 * كثيرةٌ في سوريا بلا خدمات غوغل) **فُتح الرابطُ في المتصفّح.**
 */
@Composable
fun UpdateGate(pkg: String, fallbackUrl: String = "https://rahalgo.com/app") {
    val ctx = LocalContext.current

    // **ولا مخرجَ بالرجوع** — انظر أعلاه.
    BackHandler(enabled = true) { }

    Surface(Modifier.fillMaxSize(), color = Rahal.colors.canvas) {
        Column(
            modifier = Modifier.fillMaxSize().padding(horizontal = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Text(
                text = stringResource(R.string.update_title),
                color = Rahal.colors.ink,
                style = MaterialTheme.typography.headlineSmall,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(12.dp))
            Text(
                text = stringResource(R.string.update_body),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(28.dp))
            RahalButton(
                onClick = {
                    // **ومحاولةُ بلاي أوّلاً** — انظر أعلاه.
                    val market = Intent(
                        Intent.ACTION_VIEW,
                        Uri.parse("market://details?id=$pkg"),
                    ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    val ok = runCatching { ctx.startActivity(market) }.isSuccess
                    if (!ok) {
                        runCatching {
                            ctx.startActivity(
                                Intent(Intent.ACTION_VIEW, Uri.parse(fallbackUrl))
                                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                            )
                        }
                    }
                },
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.update_action)) }
        }
    }
}
