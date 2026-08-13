package com.rahalgo.driver.menu

import android.content.Intent
import com.rahalgo.design.Rahal
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.Card
import com.rahalgo.driver.ui.Empty
import com.rahalgo.driver.ui.LoadState
import com.rahalgo.driver.ui.Screen
import com.rahalgo.driver.ui.ScreenTitle
import com.rahalgo.driver.ui.SectionTitle

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تواصل معنا — ولا رقمَ مكتوبٌ في الشيفرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسم المنصّة بأيّ مكانٍ أبدا» — وكذلك
 *  رقمُها وعنوانُها.)
 *
 * **وما كُتب في شيفرةٍ لا يُبدَّل إلّا بنشر** — فيبقى الرقمُ القديمُ
 * معروضاً شهراً، **ومن اتّصل به لم يجد أحدا.**
 *
 * # وفارغُه يُحذف لا يُعرض
 *
 * **وسطرٌ يقول «الهاتف: —» أسوأُ من غيابه**: يُقرأ عطباً في المنصّة لا
 * حقلاً لم يُملأ بعد. **وأيقونةٌ لا تفتح شيئاً عطبٌ ظاهر.**
 *
 * # ويُضغط فيقع الفعل
 *
 * **رقمٌ يُقرأ ثمّ يُكتب بالإصبع في لوحة الاتّصال عملٌ زائد** — والسائقُ
 * يفتح هذه الشاشةَ وهو في الشارع.
 */
@Composable
fun ContactScreen(vm: SectionsViewModel) {
    val p = vm.platform
    if (p == null) {
        LoadState(vm.busy, vm.error) { vm.open(MenuItem.Contact, force = true) }
        return
    }
    val context = LocalContext.current
    fun open(url: String) {
        runCatching {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        }
    }

    val phone = p.supportPhone
    val wa = p.social.whatsapp.filter(Char::isDigit)
    val empty = phone.isEmpty() && wa.isEmpty() && p.address.isEmpty() &&
        p.social.facebook.isEmpty() && p.social.instagram.isEmpty() && p.social.telegram.isEmpty()

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_contact),
            stringResource(R.string.contact_hint),
        )
        if (empty) {
            Empty(stringResource(R.string.contact_none))
            return@Screen
        }

        if (phone.isNotEmpty()) {
            Way(
                label = stringResource(R.string.contact_phone),
                value = phone,
                onClick = { open("tel:" + phone) },
            )
        }
        if (wa.isNotEmpty()) {
            Way(
                label = stringResource(R.string.contact_whatsapp),
                value = p.social.whatsapp,
                onClick = { open("https://wa.me/" + wa) },
            )
        }
        if (p.address.isNotEmpty()) {
            Way(
                label = stringResource(R.string.contact_address),
                value = p.address,
                // **والعنوانُ يفتح الخريطةَ إن كان له موضع** — وإلّا
                // فنصٌّ يُقرأ ولا يُضغط.
                onClick = if (p.location.isNotEmpty()) {
                    { open("geo:" + p.location + "?q=" + Uri.encode(p.address)) }
                } else {
                    null
                },
            )
        }

        val accounts = listOf(
            R.string.contact_facebook to p.social.facebook,
            R.string.contact_instagram to p.social.instagram,
            R.string.contact_telegram to p.social.telegram,
        ).filter { it.second.isNotEmpty() }

        if (accounts.isNotEmpty()) {
            SectionTitle(stringResource(R.string.contact_follow))
            accounts.forEach { (label, url) ->
                Way(
                    label = stringResource(label),
                    value = url,
                    onClick = { open(url) },
                )
            }
        }
    }
}

/** **بابُ تواصلٍ واحد** — اسمُه وقيمتُه، ويُضغط إن كان له فعل. */
@Composable
private fun Way(label: String, value: String, onClick: (() -> Unit)?) {
    Spacer(Modifier.height(8.dp))
    Card(
        modifier = if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier,
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
            Text(
                text = value,
                color = if (onClick != null) Rahal.colors.brand else MaterialTheme.colorScheme.onSurface,
                fontWeight = FontWeight.Medium,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}
