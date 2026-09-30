package com.rahalgo.merchant.delivery

import android.app.Application
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DeliveriesApi
import com.rahalgo.shared.merchant.Delivery
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.Note
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.apiError
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مراقبةُ التوصيلة — ليعرف المتجرُ أين هي** (الخطوة ١٨، ٢٠٢٦-١٠-٠١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (نصُّ المالك: «يجب أن يكون هناك شاشةُ مراقبة الطلب بسجلّ الطلبات ليعرف
 *  المتجرُ حالةَ توصيلته».)
 *
 * **خمسُ خطواتٍ لا أربعَ عشرةَ حالاً**: بانتظار سائق · السائقُ في الطريق
 * إليك · السائقُ عندك · في الطريق إلى المستلِم · تمّ التسليم. **ولا «تحضير»**
 * — مراحلُ الزبون فيها مطبخٌ لا وجودَ له هنا، فلا تُستعار.
 *
 * **وتُقرأ كلَّ عشر ثوانٍ ما دامت جارية** — ويتوقّف القرعُ حين تنتهي.
 */
class DeliveryTrackViewModel(app: Application) : AndroidViewModel(app) {
    private val api = DeliveriesApi(AppCore.get().api)

    var d by mutableStateOf<Delivery?>(null)
        private set
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set

    fun load(id: String) {
        viewModelScope.launch {
            try {
                d = api.get(id)
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
        }
    }

    fun cancel(id: String) {
        if (busy) return
        busy = true
        viewModelScope.launch {
            try {
                d = api.cancel(id)
                Refresh.bump()
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            } finally {
                busy = false
            }
        }
    }
}

/** **الخطوةُ من الحال** — والمنتهيةُ بلا تسليمٍ خارجَ الخطّ (-1). */
private fun stepOf(status: String): Int = when (status) {
    "dispatching" -> 0
    "assigned" -> 1
    "at_pickup" -> 2
    "picked_up", "on_the_way", "at_dropoff" -> 3
    "delivered" -> 4
    else -> -1
}

private val openStatuses = setOf("dispatching", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")

@Composable
fun DeliveryTrackScreen(vm: DeliveryTrackViewModel, id: String, onClose: () -> Unit) {
    BackHandler { onClose() }
    LaunchedEffect(id) {
        vm.load(id)
        // **والقرعُ ما دامت جارية** — ويتوقّف حين تنتهي أو تُغلق الشاشة.
        while (true) {
            delay(10_000)
            val st = vm.d?.status
            if (st != null && st !in openStatuses) break
            vm.load(id)
        }
    }
    val d = vm.d
    Screen {
        ScreenTitle(
            stringResource(R.string.md_track_title, d?.number?.toString() ?: ""),
            stringResource(R.string.md_track_hint),
        )
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Note(vm.error, Rahal.colors.danger)
        }
        if (d == null) {
            if (vm.error.isEmpty()) Text(stringResource(R.string.md_loading), color = Rahal.colors.inkMuted)
            return@Screen
        }

        val at = stepOf(d.status)
        Spacer(Modifier.height(12.dp))
        if (at < 0) {
            // **انتهت بلا تسليم** — تُقال بحالها وسببها، لا خطواتٌ ناقصة.
            Note(
                stringResource(
                    if (d.status == "failed") R.string.os_failed else R.string.os_cancelled,
                ) + (d.cancelReason?.takeIf { it.isNotBlank() }?.let { " — $it" } ?: ""),
                Rahal.colors.danger,
            )
        } else {
            Card {
                val steps = listOf(
                    R.string.md_step_waiting,
                    R.string.md_step_to_you,
                    R.string.md_step_at_you,
                    R.string.md_step_to_recipient,
                    R.string.md_step_delivered,
                )
                steps.forEachIndexed { i, label ->
                    val done = i < at || (i == at && at == 4)
                    val now = i == at && at < 4
                    Row(
                        Modifier.fillMaxWidth().padding(vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            Modifier
                                .size(14.dp)
                                .clip(CircleShape)
                                .background(
                                    when {
                                        done -> Rahal.colors.success
                                        now -> Rahal.colors.brand
                                        else -> Rahal.colors.inkMuted.copy(alpha = 0.25f)
                                    },
                                ),
                        )
                        Spacer(Modifier.width(10.dp))
                        Text(
                            stringResource(label),
                            fontWeight = if (now) FontWeight.Bold else FontWeight.Normal,
                            color = if (done || now) Rahal.colors.ink else Rahal.colors.inkMuted,
                        )
                        Spacer(Modifier.weight(1f))
                        val time = when (i) {
                            0 -> d.createdAt
                            3 -> d.pickedUpAt
                            4 -> d.deliveredAt
                            else -> null
                        }
                        if ((done || now) && !time.isNullOrBlank()) {
                            Text(clock(time), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }

        // ── تفاصيلُها ─────────────────────────────────────────────────
        Spacer(Modifier.height(12.dp))
        Card {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(stringResource(R.string.md_track_to, d.recipientName), fontWeight = FontWeight.Bold)
                Text(d.recipientPhone, color = Rahal.colors.inkMuted)
                Text(d.addressText, color = Rahal.colors.inkMuted)
                if (!d.dropoffKnown) {
                    Text(stringResource(R.string.md_row_no_point), color = Rahal.colors.inkMuted)
                }
                if (d.parcelNote.isNotBlank()) Text(d.parcelNote)
                Text(
                    stringResource(
                        when (d.feePayer) {
                            "recipient" -> R.string.md_row_cash
                            "merchant_cash" -> R.string.md_row_me_cash
                            else -> R.string.md_row_wallet
                        },
                        d.fee.toString(),
                    ),
                )
            }
        }
        if (d.status in setOf("dispatching", "assigned", "at_pickup")) {
            Spacer(Modifier.height(8.dp))
            RahalTextButton(onClick = { vm.cancel(id) }, enabled = !vm.busy) {
                Text(stringResource(R.string.md_cancel))
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/** **الساعةُ وحدَها** «14:05» — من طابع الخادم (`2026-10-01T14:05:…`). */
private fun clock(iso: String): String {
    val t = iso.substringAfter('T', "")
    return if (t.length >= 5) t.substring(0, 5) else ""
}
