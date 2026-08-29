package com.rahalgo.rep.link

import android.content.Intent
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.rep.board.BoardViewModel
import com.rahalgo.ui.Card
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رابطُ دعوة المتاجر — ومن سجّل به يُنسب إليه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا مقفلٌ حتّى يوثّق واتساب
 *
 * **المتجرُ الذي يُسجّل برمزه يُنسب إليه وتجري عمولتُه** — **ومن لا
 * قناةَ تواصلٍ موثّقةً له لا يُتحقّق أنّه هو.**
 *
 * # ولا يُبنى الرابطُ في الجهاز
 *
 * **عنوانُ الموقع يتغيّر من نشرٍ إلى نشر** — **ورابطٌ يُركَّب في هاتفٍ
 * يحمل عنوانَ من ركّبه.** فيُبنى هنا من رمزٍ يعطيه المحرّك **وعنوانٍ
 * واحدٍ مكتوبٍ في `Backend`** لا في كلّ شاشة.
 *
 * # ووجهتُه `‎/join` لا `‎/signup`
 *
 * **`join` للمتاجر بكود المندوب، و`signup` للزبائن** — **وشيءٌ اسمُه
 * «ref» في مكانين يُخلط بينهما**، فيقع صاحبُ المتجر على نموذج زبون.
 */
@Composable
fun LinkScreen(vm: BoardViewModel) {
    val me = vm.me
    if (me == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }
    val context = LocalContext.current
    val code = me.inviteCode.orEmpty()
    val open = me.whatsappVerified && code.isNotEmpty()
    val link = if (open) SITE + "/join?ref=" + code else ""

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_link),
            stringResource(R.string.soon_link),
        )

        if (!open) {
            Spacer(Modifier.height(12.dp))
            Note(stringResource(R.string.bd_code_locked), Rahal.colors.danger)
            return@Screen
        }

        Spacer(Modifier.height(12.dp))
        Card {
            Text(
                stringResource(R.string.bd_code),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = code,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.headlineSmall,
            )
            Spacer(Modifier.height(10.dp))
            KeyValue(stringResource(R.string.lk_link), link)
        }

        Spacer(Modifier.height(14.dp))
        RahalButton(
            onClick = {
                // **ويُشارَك بما يعرفه هاتفُه** — واتساب أو رسالة:
                // **ونسخُ رابطٍ إلى الحافظة يُوجب عليه أن يفتح تطبيقاً
                // ويلصق**، وأكثرُهم لا يفعل. **والمندوبُ يقف أمام صاحب
                // المتجر.**
                runCatching {
                    context.startActivity(
                        Intent.createChooser(
                            Intent(Intent.ACTION_SEND).apply {
                                type = "text/plain"
                                putExtra(
                                    Intent.EXTRA_TEXT,
                                    context.getString(R.string.lk_share_text, link),
                                )
                            },
                            null,
                        ),
                    )
                }
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.act_share_link)) }

        Spacer(Modifier.height(8.dp))
        RahalOutlineButton(
            onClick = {
                val cb = context.getSystemService(android.content.ClipboardManager::class.java)
                cb?.setPrimaryClip(android.content.ClipData.newPlainText("rahalgo", link))
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.lk_copy)) }

        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.lk_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **عنوانُ موقع الزبائن** — **ومكتوبٌ مرّةً لا في كلّ شاشة.**
 *
 * **ويومَ يُبدَّل النطاقُ يُبدَّل هنا وحدَه.**
 */
private const val SITE = "https://rahalgo.com"
