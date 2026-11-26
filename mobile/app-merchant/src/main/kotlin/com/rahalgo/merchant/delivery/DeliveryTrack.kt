package com.rahalgo.merchant.delivery

import android.app.Application
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DeliveriesApi
import com.rahalgo.shared.merchant.Delivery
import com.rahalgo.shared.merchant.hasChat
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.ChatFab
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.Note
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
 * **خمسُ خطواتٍ لا أربعَ عشرةَ حالاً**: بانتظار كابتن · الكابتنُ في الطريق
 * إليك · الكابتنُ عندك · في الطريق إلى المستلِم · تمّ التسليم. **ولا «تحضير»**
 * — مراحلُ الزبون فيها مطبخٌ لا وجودَ له هنا، فلا تُستعار.
 *
 * **وتُقرأ كلَّ عشر ثوانٍ ما دامت جارية** — ويتوقّف القرعُ حين تنتهي.
 *
 * **والشكلُ (التصميمُ المعتمد ٢٠٢٦-١١-٢٧)**: بطاقةُ حالٍ كبيرةٌ بشريط تقدّم · كودُ الاستلام في الأعلى
 * لـ«إحضار» · خطٌّ زمنيٌّ عموديٌّ متّصل · تفاصيلُ بأيقونات · وإلغاءٌ بسطرٍ يقول متى يُمكن.
 * **ولا بطاقةَ كابتن**: الخادمُ لا يرسل للمتجر اسمَه ولا رقمَه (`driver_assigned` وحدَه).
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
    // **ولوحُ الحديث نفسُه الذي عند الزبون** (`OrderChatSheet`) — والمحرّكُ يعرف أنّك المتجر من جلستك.
    var chatOpen by androidx.compose.runtime.saveable.rememberSaveable(id) { mutableStateOf(false) }
    if (chatOpen) {
        val chatVm: com.rahalgo.ui.OrderChatViewModel = androidx.lifecycle.viewmodel.compose.viewModel()
        com.rahalgo.ui.OrderChatSheet(
            vm = chatVm,
            orderId = id,
            // **وما قُرئ يسقط من الشارة** — فتحُ الحديث يَسِمه في المحرّك.
            onLoaded = { vm.load(id) },
        ) { chatOpen = false }
    }
    // **والحديثُ قرصٌ طافٍ فوق الشاشة** (قرارُ المالك ٢٠٢٦-١١-٢٦: «عائم متل الزبون») — لا زرٌّ في آخرها.
    Box(Modifier.fillMaxSize()) {
    Screen {
        ScreenTitle(
            stringResource(R.string.md_track_title, d?.number?.toString() ?: ""),
            stringResource(R.string.md_track_hint),
        )
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.sm))
            Note(vm.error, Rahal.colors.danger)
        }
        if (d == null) {
            // **والتحميلُ بالقطعة المشتركة** (تدقيقُ ٢٠٢٦-١٠-٢٧).
            if (vm.error.isEmpty()) com.rahalgo.ui.LoadState(loading = true, error = "")
            return@Screen
        }

        val fetch = d.deliveryMode == "fetch"
        val at = stepOf(d.status)

        // ══════════════════════════════════════════════════════════════
        // **بطاقةُ الحال الكبيرة** — الحالُ الآن بخطٍّ كبيرٍ وشريطُ تقدّم.
        // **والمنتهيةُ بلا تسليمٍ حمراءُ بسببها** (فشلٌ، رفضُ المنصّة، إلغاء) — لا خطواتٌ ناقصة.
        // ══════════════════════════════════════════════════════════════
        StatusHero(
            label = stringResource(if (fetch) R.string.md_hero_label_fetch else R.string.md_hero_label),
            text = if (d.status == "dispatching") stringResource(R.string.md_step_waiting) else deliveryStatusText(d.status, fetch),
            reason = if (at < 0) d.cancelReason?.takeIf { it.isNotBlank() } else null,
            progress = if (at < 0) null else (at.toFloat() / LAST_STEP).coerceAtLeast(0.06f),
            danger = at < 0,
        )

        // ══════════════════════════════════════════════════════════════
        // **«إحضار»: كودُ الاستلام لك** (قرارُ المالك ٢٠٢٦-١٠-١٦) — تعطيه للكابتن لما يوصلك الغرض.
        // **وفي أعلى الصفحة بخطٍّ كبير.**
        // ══════════════════════════════════════════════════════════════
        d.deliveryCode?.takeIf { it.isNotBlank() && fetch }?.let { code ->
            Spacer(Modifier.height(Rahal.space.md))
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success) {
                Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(stringResource(R.string.md_code_title), style = Rahal.type.cardTitle, color = Rahal.colors.ink)
                    Spacer(Modifier.height(Rahal.space.xs))
                    Text(
                        code,
                        color = Rahal.colors.success,
                        style = Rahal.type.display.copy(fontSize = 40.sp, letterSpacing = 10.sp),
                    )
                    Text(
                        stringResource(R.string.md_code_hint),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                        textAlign = TextAlign.Center,
                    )
                }
            }
        }

        // **والمبلغُ الحقيقيُّ الذي كتبه الكابتنُ وحالُ موافقتك** (٢٠٢٦-١٠-١٦) — وبابُ الموافقة نافذتُها
        // (`FetchAmountDialog`) فوق أيّ شاشة.
        if (fetch) {
            DeliveryTexts.amountRow(d.fetchAmountStatus)?.let { row ->
                Spacer(Modifier.height(Rahal.space.md))
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

        Spacer(Modifier.height(Rahal.space.md))
        if (at >= 0) {
            // **والانتظارُ يُرى يجري** — دوائرُ تنبض حول الموتور (طلبُ المالك).
            if (d.status == "dispatching") {
                SearchingBanner()
                Spacer(Modifier.height(Rahal.space.md))
            }
            // **خطٌّ زمنيٌّ عموديٌّ متّصل** — المنجَزُ أخضرُ بعلامة، والحاليُّ بلون العلامة، والآتي باهت.
            RahalCard(variant = CardVariant.Outlined) {
                // **وفي «إحضار» إلى مكان الاستلام ثمّ إليك** (قرارُ المالك ٢٠٢٦-١٠-١٦) — لا «إلى المستلم».
                val steps = DeliveryTexts.trackSteps(fetch)
                steps.forEachIndexed { i, label ->
                    val done = i < at || (i == at && at == LAST_STEP)
                    val now = i == at && at < LAST_STEP
                    val time = when (i) {
                        0 -> d.createdAt
                        4 -> d.pickedUpAt
                        LAST_STEP -> d.deliveredAt
                        else -> null
                    }
                    val hint = when {
                        now && !fetch && i == 2 -> R.string.md_step_hint_to_you
                        now && fetch && i == 4 -> R.string.md_step_hint_to_you_fetch
                        else -> null
                    }
                    TimelineStep(
                        label = stringResource(label),
                        hint = hint?.let { stringResource(it) },
                        time = if ((done || now) && !time.isNullOrBlank()) clock(time) else "",
                        done = done,
                        now = now,
                        last = i == steps.lastIndex,
                        nowColor = if (at == 0) Rahal.colors.accent else Rahal.colors.brand,
                    )
                }
            }
            Spacer(Modifier.height(Rahal.space.md))
        }

        // ── تفاصيلُها ─────────────────────────────────────────────────
        RahalCard(variant = CardVariant.Outlined) {
            Text(
                stringResource(R.string.md_track_details),
                style = Rahal.type.cardTitle,
                color = Rahal.colors.ink,
            )
            Spacer(Modifier.height(Rahal.space.xs))
            // **«إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — العنوانُ مكانُ الاستلام والوجهةُ متجرُك.
            if (fetch) {
                DetailRow(
                    com.rahalgo.ui.R.drawable.ic_moto,
                    stringResource(R.string.md_lbl_type),
                    stringResource(R.string.md_row_fetch),
                    strong = true,
                )
            }
            DetailRow(
                com.rahalgo.ui.R.drawable.ic_user,
                stringResource(if (fetch) R.string.md_lbl_recipient_fetch else R.string.md_lbl_recipient),
                d.recipientName,
                strong = true,
            )
            DetailRow(
                com.rahalgo.ui.R.drawable.ic_phone,
                stringResource(R.string.md_lbl_phone),
                com.rahalgo.ui.phoneText(d.recipientPhone),
            )
            DetailRow(
                com.rahalgo.ui.R.drawable.ic_home,
                stringResource(if (fetch) R.string.md_lbl_address_fetch else R.string.md_lbl_address),
                d.addressText,
            )
            if (!d.dropoffKnown) {
                DetailRow(com.rahalgo.ui.R.drawable.ic_pin, null, stringResource(R.string.md_row_no_point), muted = true)
            }
            if (d.parcelNote.isNotBlank()) {
                DetailRow(com.rahalgo.ui.R.drawable.ic_orders, stringResource(R.string.md_lbl_parcel), d.parcelNote)
            }
            DetailRow(
                com.rahalgo.ui.R.drawable.ic_wallet,
                null,
                stringResource(DeliveryTexts.feeRow(d.feePayer, fetch), com.rahalgo.ui.grouped(d.fee)),
            )
            // **وما يقبضه الكابتنُ من المستلِم لك** — خارجَ حساب المنصّة (٢٠٢٦-١٠-١٥). **وفي «إحضار»
            // يدفعه بمكان الاستلام ويأخذه منك** (٢٠٢٦-١٠-١٦).
            d.collectAmount?.takeIf { it > 0 }?.let {
                DetailRow(
                    com.rahalgo.ui.R.drawable.ic_banknotes,
                    null,
                    stringResource(DeliveryTexts.collectRow(fetch), DeliveryTexts.amount(d, it)),
                )
            }
        }

        // ── الإلغاء: متاحٌ قبل وصول الكابتن بسطرٍ يقول حدَّه، وبعده سطرٌ يقول لماذا لا ──
        if (d.status in cancellable) {
            Spacer(Modifier.height(Rahal.space.lg))
            CancelDeliveryButton(busy = vm.busy) { vm.cancel(id) }
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(if (fetch) R.string.md_cancel_note_fetch else R.string.md_cancel_note),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        } else if (d.status in openStatuses) {
            Spacer(Modifier.height(Rahal.space.lg))
            Text(
                stringResource(R.string.md_cancel_closed),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        }
        // **ومتّسعٌ تحت آخر زرّ** كي لا يغطّيه القرصُ الطافي.
        Spacer(Modifier.height(if (d.hasChat()) 96.dp else Rahal.space.xl))
    }
    // ══════════════════════════════════════════════════════════════
    // **راسل الكابتن** (قرارُ المالك ٢٠٢٦-١١-٢٦) — من الإسناد حتّى التسليم، ويبقى يُقرأ بعده.
    // **قرصٌ طافٍ بعدّاد الجديد** كما عند الزبون (`ChatFab`).
    // ══════════════════════════════════════════════════════════════
    if (d?.hasChat() == true && !chatOpen) {
        ChatFab(unread = d.chatUnread) { chatOpen = true }
    }
    }
}

/** **بطاقةُ الحال الكبيرة** — الحالُ بخطٍّ كبيرٍ فوق شريط تقدّم، وحمراءُ للمنتهية بلا تسليم. */
@Composable
private fun StatusHero(label: String, text: String, reason: String?, progress: Float?, danger: Boolean) {
    Spacer(Modifier.height(Rahal.space.sm))
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(if (danger) Rahal.colors.danger else Rahal.colors.brand)
            .padding(Rahal.space.lg),
    ) {
        Text(label, color = Rahal.colors.onBrand.copy(alpha = 0.9f), style = Rahal.type.caption)
        Text(text, color = Rahal.colors.onBrand, style = Rahal.type.headline)
        if (!reason.isNullOrBlank()) {
            Text(reason, color = Rahal.colors.onBrand, style = Rahal.type.body)
        }
        if (progress != null) {
            Spacer(Modifier.height(Rahal.space.md))
            Box(
                Modifier
                    .fillMaxWidth()
                    .height(Rahal.space.sm)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.onBrand.copy(alpha = 0.3f)),
            ) {
                Box(
                    Modifier
                        .fillMaxWidth(progress.coerceIn(0f, 1f))
                        .fillMaxHeight()
                        .clip(Rahal.shape.pill)
                        .background(Rahal.colors.onBrand),
                )
            }
        }
    }
}

/** **خطوةٌ في الخطّ الزمنيّ** — دائرةٌ وخطٌّ نازلٌ يصلها بالتالية، والنصُّ ووقتُه بجانبها. */
@Composable
private fun TimelineStep(
    label: String,
    hint: String?,
    time: String,
    done: Boolean,
    now: Boolean,
    last: Boolean,
    nowColor: Color,
) {
    Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min)) {
        Column(Modifier.fillMaxHeight().width(Rahal.icon.md), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier
                    .size(Rahal.icon.md)
                    .clip(CircleShape)
                    .background(
                        when {
                            done -> Rahal.colors.success
                            now -> nowColor.copy(alpha = 0.12f)
                            else -> Rahal.colors.canvas
                        },
                    )
                    .border(
                        Rahal.stroke.focus,
                        when {
                            done -> Rahal.colors.success
                            now -> nowColor
                            else -> Rahal.colors.line
                        },
                        CircleShape,
                    ),
                contentAlignment = Alignment.Center,
            ) {
                if (done) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_check),
                        contentDescription = null,
                        tint = Rahal.colors.onBrand,
                        modifier = Modifier.size(Rahal.icon.xs),
                    )
                } else if (now) {
                    PulseDot(nowColor, Rahal.size.dot)
                }
            }
            if (!last) {
                Box(
                    Modifier
                        .weight(1f)
                        .width(Rahal.stroke.focus)
                        .background(if (done) Rahal.colors.success else Rahal.colors.line),
                )
            }
        }
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f).padding(bottom = if (last) Rahal.space.none else Rahal.space.lg)) {
            Text(
                label,
                style = if (now) Rahal.type.bodyStrong else Rahal.type.body,
                color = if (done || now) Rahal.colors.ink else Rahal.colors.inkMuted,
            )
            if (hint != null) {
                Text(hint, color = Rahal.colors.inkMuted, style = Rahal.type.caption)
            }
        }
        if (time.isNotBlank()) {
            Spacer(Modifier.width(Rahal.space.sm))
            Text(time, color = Rahal.colors.inkMuted, style = Rahal.type.meta)
        }
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
