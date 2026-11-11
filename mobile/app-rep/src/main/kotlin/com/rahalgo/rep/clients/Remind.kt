package com.rahalgo.rep.clients

import android.content.Context
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepMerchant
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.waMeDigits

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«ذكّره» — رسالةُ متابعةٍ جاهزةٌ من واتساب المندوب نفسِه** (قرارُ المالك ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **لا بوتّ ولا رسالةَ من الخادم**: يفتح واتسابَ المندوب على رقم صاحب المتجر ونصٌّ جاهزٌ
 * بحسب السبب الأوّل في `follow_up`، **ويضغط هو «إرسال».**
 */
internal fun remindMessage(context: Context, m: RepMerchant, repName: String): String {
    val rep = repName.ifBlank { context.getString(R.string.rem_rep_fallback) }
    return when (m.followUp.firstOrNull()) {
        "no_items" -> context.getString(R.string.rem_no_items, rep, m.name)
        "conditions" -> {
            val missing = buildList {
                if (!m.hasHours) add(context.getString(R.string.cl_miss_hours))
                if (!m.hasLocation) add(context.getString(R.string.cl_miss_location))
                if (m.photoItems < m.minItems) {
                    add(context.getString(R.string.cl_miss_items, (m.minItems - m.photoItems).toString()))
                }
            }.joinToString("، ")
            context.getString(R.string.rem_conditions, rep, m.name, missing)
        }
        else -> context.getString(R.string.rem_no_orders, rep, m.name)
    }
}

/** **أيُعرض «ذكّره»؟** — لمتجرٍ يحتاج متابعةً ولصاحبه رقمُ واتساب. */
internal fun canRemind(m: RepMerchant): Boolean =
    m.followUp.isNotEmpty() && waMeDigits(m.ownerPhone.orEmpty()).isNotEmpty()

/** **زرُّ «ذكّره»** — يفتح الواتساب بالنصّ الجاهز. */
@Composable
internal fun RemindButton(m: RepMerchant, repName: String, modifier: Modifier = Modifier) {
    if (!canRemind(m)) return
    val context = LocalContext.current
    RahalOutlineButton(
        onClick = {
            openWhatsapp(context, waMeDigits(m.ownerPhone.orEmpty()), remindMessage(context, m, repName))
        },
        modifier = modifier,
    ) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_whatsapp),
            contentDescription = null,
            tint = Rahal.colors.success,
            modifier = Modifier.size(Rahal.icon.xs),
        )
        Spacer(Modifier.size(Rahal.space.xs))
        Text(stringResource(R.string.rem_button))
    }
}
