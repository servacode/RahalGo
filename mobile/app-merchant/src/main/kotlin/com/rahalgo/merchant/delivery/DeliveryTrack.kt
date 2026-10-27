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
import kotlinx.coroutines.flow.drop
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
//
// **وأوّلُها موافقةُ المنصّة** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — التوصيلةُ تمرّ بالمكتب.
private fun stepOf(status: String): Int = when (status) {
    "pending" -> 0
    "accepted", "dispatching" -> 1
    "assigned" -> 2
    "at_pickup" -> 3
    "picked_up", "on_the_way", "at_dropoff" -> 4
    "delivered" -> LAST_STEP
    else -> -1
}

private const val LAST_STEP = 5

private val openStatuses = setOf(
    "pending", "accepted", "dispatching", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff",
)

@Composable
fun DeliveryTrackScreen(vm: DeliveryTrackViewModel, id: String, onClose: () -> Unit) {
    BackHandler { onClose() }
    // **والوصلةُ تُجدّد المتابعةَ فوراً** (٢٠٢٦-١٠-٠٦) — كان القرعُ كلَّ عشر ثوانٍ وحدَه، **فإسنادُ السائق
    // واستلامُه يصلان متأخّرين.** والقرعُ يبقى احتياطاً.
    LaunchedEffect(id) {
        com.rahalgo.ui.Refresh.tick.drop(1).collect { vm.load(id) }
    }
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
            // **والتحميلُ بالقطعة المشتركة** (تدقيقُ ٢٠٢٦-١٠-٢٧).
            if (vm.error.isEmpty()) com.rahalgo.ui.LoadState(loading = true, error = "")
            return@Screen
        }

        // ══════════════════════════════════════════════════════════════
        // **«إحضار»: كودُ الاستلام لك** (قرارُ المالك ٢٠٢٦-١٠-١٦) — تعطيه للسائق لما يوصلك الغرض.
        // ══════════════════════════════════════════════════════════════
        d.deliveryCode?.takeIf { it.isNotBlank() && d.deliveryMode == "fetch" }?.let { code ->
            Spacer(Modifier.height(12.dp))
            Card(tone = Rahal.colors.success) {
                Text(stringResource(R.string.md_code_title), fontWeight = FontWeight.Bold)
                Spacer(Modifier.height(4.dp))
                Text(
                    code,
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineLarge,
                )
                Text(stringResource(R.string.md_code_hint), color = Rahal.colors.inkMuted)
            }
        }

        val at = stepOf(d.status)
        Spacer(Modifier.height(12.dp))
        if (at < 0) {
            // **انتهت بلا تسليم** — تُقال بحالها وسببها، لا خطواتٌ ناقصة.
            Note(
                stringResource(
                    when (d.status) {
                        "failed" -> R.string.os_failed
                        // **ورفضُ المنصّة يُقال بسببه** — لا «أُلغي» مبهمة.
                        "rejected" -> R.string.md_platform_rejected
                        else -> R.string.os_cancelled
                    },
                ) + (d.cancelReason?.takeIf { it.isNotBlank() }?.let { " — $it" } ?: ""),
                Rahal.colors.danger,
            )
        } else {
            // **والانتظارُ يُرى يجري** — دوائرُ تنبض حول الموتور (طلبُ المالك).
            if (d.status == "dispatching") {
                SearchingBanner()
                Spacer(Modifier.height(12.dp))
            }
            Card {
                // **وفي «إحضار» إلى مكان الاستلام ثمّ إليك** (قرارُ المالك ٢٠٢٦-١٠-١٦) — لا «إلى المستلم».
                val steps = DeliveryTexts.trackSteps(d.deliveryMode == "fetch")
                steps.forEachIndexed { i, label ->
                    val done = i < at || (i == at && at == LAST_STEP)
                    val now = i == at && at < LAST_STEP
                    Row(
                        Modifier.fillMaxWidth().padding(vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        // **والخطوةُ الحاليّةُ تنبض** — والمنجَزةُ خضراء.
                        if (now) {
                            PulseDot(if (at == 0) Rahal.colors.accent else Rahal.colors.brand, 14.dp)
                        } else {
                            Box(
                                Modifier
                                    .size(14.dp)
                                    .clip(CircleShape)
                                    .background(
                                        if (done) Rahal.colors.success
                                        else Rahal.colors.inkMuted.copy(alpha = 0.25f),
                                    ),
                            )
                        }
                        Spacer(Modifier.width(10.dp))
                        Text(
                            stringResource(label),
                            fontWeight = if (now) FontWeight.Bold else FontWeight.Normal,
                            color = if (done || now) Rahal.colors.ink else Rahal.colors.inkMuted,
                        )
                        Spacer(Modifier.weight(1f))
                        val time = when (i) {
                            0 -> d.createdAt
                            4 -> d.pickedUpAt
                            LAST_STEP -> d.deliveredAt
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
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.md_track_details),
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                StatusPill(d.status)
            }
            Hairline()
            // **«إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — العنوانُ مكانُ الاستلام والوجهةُ متجرُك.
            if (d.deliveryMode == "fetch") {
                InfoLine(com.rahalgo.ui.R.drawable.ic_moto, stringResource(R.string.md_row_fetch), strong = true)
            }
            InfoLine(com.rahalgo.ui.R.drawable.ic_user, d.recipientName, strong = true)
            InfoLine(com.rahalgo.ui.R.drawable.ic_phone, d.recipientPhone)
            InfoLine(com.rahalgo.ui.R.drawable.ic_home, d.addressText)
            if (!d.dropoffKnown) {
                InfoLine(com.rahalgo.ui.R.drawable.ic_pin, stringResource(R.string.md_row_no_point), muted = true)
            }
            if (d.parcelNote.isNotBlank()) {
                InfoLine(com.rahalgo.ui.R.drawable.ic_orders, d.parcelNote)
            }
            InfoLine(
                com.rahalgo.ui.R.drawable.ic_wallet,
                stringResource(DeliveryTexts.feeRow(d.feePayer, d.deliveryMode == "fetch"), com.rahalgo.ui.grouped(d.fee)),
            )
            // **وما يقبضه السائقُ من المستلِم لك** — خارجَ حساب المنصّة (٢٠٢٦-١٠-١٥). **وفي «إحضار»
            // يدفعه بمكان الاستلام ويأخذه منك** (٢٠٢٦-١٠-١٦).
            d.collectAmount?.takeIf { it > 0 }?.let {
                InfoLine(
                    com.rahalgo.ui.R.drawable.ic_wallet,
                    stringResource(DeliveryTexts.collectRow(d.deliveryMode == "fetch"), DeliveryTexts.amount(d, it)),
                )
            }
        }
        // **والمبلغُ الحقيقيُّ الذي كتبه السائقُ وحالُ موافقتك** (٢٠٢٦-١٠-١٦).
        if (d.deliveryMode == "fetch") {
            DeliveryTexts.amountRow(d.fetchAmountStatus)?.let { row ->
                Spacer(Modifier.height(10.dp))
                Note(
                    stringResource(row, DeliveryTexts.amount(d, d.fetchAmount ?: 0L)),
                    when (d.fetchAmountStatus) {
                        "approved" -> Rahal.colors.success
                        "rejected" -> Rahal.colors.danger
                        else -> Rahal.colors.accent
                    },
                )
            }
        }
        if (d.status in cancellable) {
            Spacer(Modifier.height(14.dp))
            CancelDeliveryButton(busy = vm.busy) { vm.cancel(id) }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **الساعةُ وحدَها** «14:05» — **بتوقيت سوريا لا كما وصلت** (تدقيقُ ٢٠٢٦-١٠-٢٧).
 *
 * **وكانت تُقصّ من النصّ كما هو** — والخادمُ يرسل بغرينتش أحياناً، **فتظهر
 * ناقصةً ثلاثَ ساعات.** **والقصُّ يبقى لِما لا يُحلَّل.**
 */
private fun clock(iso: String): String {
    com.rahalgo.ui.syriaTime(iso)?.let {
        return it.format(java.time.format.DateTimeFormatter.ofPattern("HH:mm", java.util.Locale.US))
    }
    val t = iso.substringAfter('T', "")
    return if (t.length >= 5) t.substring(0, 5) else ""
}
