package com.rahalgo.driver.home

import android.content.Intent
import android.net.Uri
import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.Rahal
import com.rahalgo.driver.Bidi
import com.rahalgo.driver.R
import com.rahalgo.shared.model.BusyHours
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.ui.PagesViewModel
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.RahalDialog
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.dist
import com.rahalgo.ui.grouped
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما أضافه المالك على لوحة السائق ٢٠٢٦-١١-١١** — ملخّصُ الدوام · أفضلُ الأوقات · تسليمُ المصاري
 * ══════════════════════════════════════════════════════════════════════
 *
 * **عرضٌ فقط**: لا منطقَ مالٍ هنا ولا حركةَ مال. الأرقامُ كما يردّها `driver/me`.
 */

/** **ما يقوله ملخّصُ نهاية الدوام** — أرقامُ يومه كما وصلت بعد السكّر. */
data class ShiftSummary(
    val delivered: Int,
    val earned: Long,
    /** **مسافةُ طلبات اليوم بالمتر** — من القيادة الفعليّة كما يحسبها المحرّك؛ وصفرٌ يعني «لم تُحسب». */
    val driveM: Long,
    /** **ما معه للمكتب الآن.** */
    val cashHeld: Long,
) {
    companion object {
        fun of(me: DriverMe) = ShiftSummary(
            delivered = me.todayDelivered,
            earned = me.todayEarned,
            driveM = me.todayDriveM,
            cashHeld = me.cashHeld,
        )
    }
}

@Composable
fun ShiftSummaryDialog(summary: ShiftSummary, onClose: () -> Unit) {
    RahalDialog(
        onDismissRequest = onClose,
        title = { Text(stringResource(R.string.shift_summary_title), fontWeight = FontWeight.Bold) },
        text = {
            Column {
                SummaryRow(stringResource(R.string.shift_summary_delivered), grouped(summary.delivered.toLong()))
                SummaryRow(stringResource(R.string.shift_summary_earned), Bidi.isolate(money(summary.earned)))
                // **وصفرُ المسافة شرطة** — كما في بطاقة «اليوم»: لا «٠ م».
                SummaryRow(
                    stringResource(R.string.shift_summary_km),
                    if (summary.driveM > 0) dist(summary.driveM.toDouble()) else "—",
                )
                SummaryRow(stringResource(R.string.shift_summary_cash), Bidi.isolate(money(summary.cashHeld)))
                if (summary.cashHeld > 0) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Text(
                        text = stringResource(R.string.shift_summary_cash_hint),
                        color = Rahal.colors.accentInk,
                        style = Rahal.type.caption,
                    )
                }
            }
        },
        confirmButton = {
            RahalButton(onClick = onClose) { Text(stringResource(R.string.shift_summary_done)) }
        },
    )
}

@Composable
private fun SummaryRow(label: String, value: String) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = Rahal.space.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, color = Rahal.colors.inkMuted, style = Rahal.type.body, modifier = Modifier.weight(1f))
        Text(value, color = Rahal.colors.ink, style = Rahal.type.bodyStrong)
    }
}

// ══════════════════════════════════════════════════════════════════════
// أفضل أوقات الشغل
// ══════════════════════════════════════════════════════════════════════

/** **نافذةٌ من ساعاتٍ متتالية** — `from` شاملةٌ و`to` حصريّة (٢٤ ساعة). */
data class BusyWindow(val from: Int, val to: Int)

object BusyWindows {
    /** **تدمج الساعاتِ المتجاورة** (١٣ و١٤ ⇒ ١٣–١٥) وترتّبها زمنيّاً. */
    fun of(hours: List<Int>): List<BusyWindow> {
        val sorted = hours.filter { it in 0..23 }.distinct().sorted()
        val out = mutableListOf<BusyWindow>()
        for (h in sorted) {
            val last = out.lastOrNull()
            if (last != null && last.to == h) out[out.size - 1] = last.copy(to = h + 1) else out += BusyWindow(h, h + 1)
        }
        return out
    }

    /** **الساعةُ على ١٢** (٠ ⇒ ١٢، ١٣ ⇒ ١، ٢٤ ⇒ ١٢). */
    fun hour12(h: Int): Int {
        val x = h % 12
        return if (x == 0) 12 else x
    }

    /** **جزءُ اليوم** من ساعة البداية. */
    @StringRes
    fun period(h: Int): Int = when (h) {
        in 5..11 -> R.string.busy_period_morning
        in 12..15 -> R.string.busy_period_noon
        in 16..17 -> R.string.busy_period_afternoon
        in 18..23 -> R.string.busy_period_evening
        else -> R.string.busy_period_night
    }
}

@Composable
fun BusyHoursCard(busy: BusyHours) {
    if (busy.top.isEmpty()) return
    val windows = BusyWindows.of(busy.top.map { it.hour })
    val periods = windows.map { stringResource(BusyWindows.period(it.from)) }
    val text = windows.mapIndexed { i, w ->
        "${BusyWindows.hour12(w.from)}–${BusyWindows.hour12(w.to)} " + periods[i]
    }.joinToString("، ")
    RahalCard(padding = Rahal.space.lg) {
        Text(stringResource(R.string.busy_hours_title), style = Rahal.type.sectionTitle, color = Rahal.colors.ink)
        Spacer(Modifier.height(Rahal.space.sm))
        Text(
            text = stringResource(R.string.busy_hours_best, text),
            style = Rahal.type.bodyStrong,
            color = Rahal.colors.brand,
        )
        Spacer(Modifier.height(Rahal.space.xs))
        Text(
            text = stringResource(R.string.busy_hours_note, busy.days.toString()),
            style = Rahal.type.caption,
            color = Rahal.colors.inkMuted,
        )
    }
}

// ══════════════════════════════════════════════════════════════════════
// سلّم المصاري للمكتب
// ══════════════════════════════════════════════════════════════════════

/**
 * **زرُّ «سلّم المصاري للمكتب»** — يظهر لمن معه نقد، ويفتح ورقةً بالمبلغ وطريقة التسليم.
 *
 * **لا حركةَ مالٍ منه**: التسليمُ يسجّله موظّفُ المكتب (`officecash`). **وعنوانُ المكتب ورقمُه من إعدادات المنصّة**
 * (`platform.address` · `platform.support_phone`) — نفسُها صفحةُ «تواصل معنا». ولا ساعاتِ دوامٍ في الإعدادات اليوم
 * فلا تُعرض.
 */
@Composable
fun HandInButton(held: Long, modifier: Modifier = Modifier) {
    if (held <= 0) return
    var open by remember { mutableStateOf(false) }
    RahalButton(onClick = { open = true }, modifier = modifier.fillMaxWidth(), compact = true) {
        Text(stringResource(R.string.handin_button))
    }
    if (open) HandInDialog(held) { open = false }
}

@Composable
private fun HandInDialog(held: Long, onClose: () -> Unit) {
    val context = LocalContext.current
    val pages: PagesViewModel = viewModel()
    LaunchedEffect(Unit) { if (pages.platform == null) pages.load() }
    val platform = pages.platform
    val phone = platform?.supportPhone.orEmpty().trim()
    val address = platform?.address.orEmpty().trim()
    RahalDialog(
        onDismissRequest = onClose,
        title = { Text(stringResource(R.string.handin_title), fontWeight = FontWeight.Bold) },
        text = {
            Column {
                Text(stringResource(R.string.handin_amount), color = Rahal.colors.inkMuted, style = Rahal.type.label)
                Text(Bidi.isolate(money(held)), color = Rahal.colors.accentInk, style = Rahal.type.display)
                Spacer(Modifier.height(Rahal.space.md))
                Text(stringResource(R.string.handin_how), style = Rahal.type.body)
                if (address.isNotEmpty()) {
                    Spacer(Modifier.height(Rahal.space.md))
                    Text(stringResource(R.string.handin_address, address), style = Rahal.type.bodyStrong)
                }
                if (phone.isNotEmpty()) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    Text(stringResource(R.string.handin_phone, Bidi.isolate(phone)), style = Rahal.type.bodyStrong)
                }
                if (platform != null && phone.isEmpty() && address.isEmpty()) {
                    Spacer(Modifier.height(Rahal.space.md))
                    Text(stringResource(R.string.handin_none), color = Rahal.colors.inkMuted, style = Rahal.type.caption)
                }
            }
        },
        confirmButton = {
            if (phone.isNotEmpty()) {
                RahalButton(onClick = {
                    context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
                }) { Text(stringResource(R.string.handin_call)) }
            } else {
                RahalButton(onClick = onClose) { Text(stringResource(R.string.handin_close)) }
            }
        },
        dismissButton = {
            if (phone.isNotEmpty()) {
                RahalOutlineButton(onClick = onClose) { Text(stringResource(R.string.handin_close)) }
            }
        },
    )
}
