package com.rahalgo.rep.clients

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.waMeDigits

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رقمُ صاحب المتجر يُضغط — اتّصالٌ أو واتساب** (٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المندوبُ شغلُه أن يلاحق صاحبَ المتجر ليُفعّله** — **وكان الرقمُ نصّاً
 * يُنسخ باليد.** فزرّان بوزنٍ واحد: «اتصال» (`tel:`) و«واتساب» (`wa.me`).
 *
 * **والكلمتان والأيقونتان من `:ui`** — هما نفسُهما في صفحة «تواصل معنا»،
 * **وتحويلُ الرقم لـwa.me من الدالّة المشتركة** (`waMeDigits`) فلا يُكتب
 * مرّتين.
 */
@Composable
fun ContactButtons(phone: String, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val wa = waMeDigits(phone)
    Row(modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        RahalOutlineButton(onClick = { dial(context, phone) }, modifier = Modifier.weight(1f)) {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_phone),
                contentDescription = null,
                modifier = Modifier.size(18.dp),
            )
            Spacer(Modifier.size(6.dp))
            Text(stringResource(com.rahalgo.ui.R.string.contact_call))
        }
        if (wa.isNotEmpty()) {
            RahalOutlineButton(onClick = { whatsapp(context, wa) }, modifier = Modifier.weight(1f)) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_whatsapp),
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(stringResource(com.rahalgo.ui.R.string.contact_whatsapp))
            }
        }
    }
}

/** **أيقونةُ اتّصالٍ صغيرة** — لبطاقة طلبٍ معلَّق، حيث لا مكانَ لزرّين. */
@Composable
fun CallIcon(phone: String) {
    val context = LocalContext.current
    IconButton(onClick = { dial(context, phone) }, modifier = Modifier.size(36.dp)) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_phone),
            contentDescription = stringResource(com.rahalgo.ui.R.string.contact_call),
            tint = Rahal.colors.brand,
            modifier = Modifier.size(20.dp),
        )
    }
}

/** **يفتح لوحةَ الاتّصال بالرقم** — ولا يتّصل وحدَه: لا إذنَ `CALL_PHONE`. */
private fun dial(context: Context, phone: String) {
    val p = phone.trim()
    if (p.isEmpty()) return
    runCatching { context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$p"))) }
}

private fun whatsapp(context: Context, digits: String) {
    runCatching {
        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse("https://wa.me/$digits")))
    }
}

/**
 * **تاريخٌ يقرؤه الناس** — «27 تشرين الأول 2026» لا «2026-10-27».
 *
 * **وأسماءُ الشهور من لغة سوريا في النظام** (`ar-SY`: كانون وشباط
 * وتشرين) — **والأرقامُ لاتينيّةٌ كسائر أرقام التطبيق** (`DecimalStyle.STANDARD`).
 * **وما لا يُفهم يُعرض كما جاء مقصوصاً** — ولا يُخترع.
 */
fun longDate(iso: String): String {
    val at = com.rahalgo.ui.syriaTime(iso)?.toLocalDate()
        ?: runCatching { java.time.LocalDate.parse(iso.take(10)) }.getOrNull()
        ?: return iso.take(10)
    val fmt = java.time.format.DateTimeFormatter
        .ofPattern("d MMMM yyyy", java.util.Locale("ar", "SY"))
        .withDecimalStyle(java.time.format.DecimalStyle.STANDARD)
    return runCatching { at.format(fmt) }.getOrDefault(iso.take(10))
}
