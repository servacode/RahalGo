package com.rahalgo.merchant.orders

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.MerchantOrder
import com.rahalgo.shared.merchant.OrderLine
import com.rahalgo.ui.Card
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ الطلبات — زرٌّ واحدٌ في كلّ لحظة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣.)
 *
 * # ولماذا زرٌّ واحدٌ لا ثلاثة
 *
 * **صاحبُ المتجر يداه في العمل** — ينظر إلى الشاشة ثانيتين. **وثلاثةُ
 * أزرارٍ معاً تعني قراءةً واختياراً**، وزرٌّ واحدٌ يقول الخطوةَ التالية
 * يعني ضغطةً بلا تفكير.
 *
 * **فالزرُّ يتبع الحال**: بانتظارٌ ← «اقبل» · مقبولٌ ← «بدأت التحضير»
 * · قيدُ التحضير ← «جاهز للاستلام».
 *
 * # والاعتذارُ ثانويٌّ لا مساوٍ
 *
 * **زرّان متساويان يجعلان الرفضَ خياراً سهلاً** — وهو مخالفةٌ تُحسب
 * عليه وتضرّ الزبون. **فيبقى نصّاً جانبيّاً**: من أراده وجده، ولا
 * يُغري به.
 *
 * # ولا اسمَ زبونٍ هنا
 *
 * **السائقُ بينهما** — والمتجرُ يصنع ولا يوصّل، **ورقمُ زبونٍ في يده
 * بابُ اتّصالٍ مباشرٍ يوماً ما يُخرج المنصّةَ من بينهما.**
 */
@Composable
fun OrdersScreen(vm: OrdersViewModel) {
    if (vm.loading || (vm.error.isNotEmpty() && vm.orders.isEmpty())) {
        Screen {
            ScreenTitle(stringResource(R.string.orders_title), stringResource(R.string.orders_hint))
            LoadState(vm.loading, vm.error) { vm.refresh() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.refresh() }) {
        Screen {
            ScreenTitle(stringResource(R.string.orders_title), stringResource(R.string.orders_hint))

            if (vm.orders.isEmpty()) {
                Empty(stringResource(R.string.orders_empty))
                return@Screen
            }

            vm.orders.forEach { order ->
                Spacer(Modifier.height(10.dp))
                OrderCard(
                    order = order,
                    busy = vm.busy(order.id),
                    onAccept = { vm.accept(order.id) },
                    onStart = { vm.startPreparing(order.id) },
                    onReady = { vm.markReady(order.id) },
                    onReject = { vm.reject(order.id, "") },
                )
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
private fun OrderCard(
    order: MerchantOrder,
    busy: Boolean,
    onAccept: () -> Unit,
    onStart: () -> Unit,
    onReady: () -> Unit,
    onReject: () -> Unit,
) {
    Card {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                "#" + order.number,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.weight(1f))
            // **ولا عدّادَ مهلةٍ في هذا الباب** — المحرّكُ لا يرسله مع
            // الطلب، **وعدٌّ تحسبه الشاشةُ من وقت الإنشاء يكذب**: ساعةُ
            // الجهاز تفترق عن ساعة الخادم بدقائق. **فيُقرأ منقضياً وهو
            // حيّ، فيتردّد صاحبُ المتجر عن طلبٍ ما زال أمامه وقت.**
        }

        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.order_items),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.labelSmall,
        )
        order.items.forEach { Line(it) }

        // **وملاحظةُ الزبون تُقرأ قبل الصنع لا بعده** — «بلا بصل»
        // تُقرأ بعد أن يُصنع تعني إعادةَ صنعه.
        if (order.notes.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.order_note),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
            )
            Text(order.notes, style = MaterialTheme.typography.bodyMedium)
        }

        Spacer(Modifier.height(10.dp))
        HorizontalDivider()
        Spacer(Modifier.height(10.dp))

        // ══════════════════════════════════════════════════════════════
        // **ولا مالَ في بطاقة الطلب — بقرارِ المحرّك لا بسهوٍ منّا**
        // ══════════════════════════════════════════════════════════════
        //
        // `merchant_privacy.go` يصفّر المبالغَ كلَّها قبل أن ترحل:
        // **«المال — يراه في محفظته وتقاريره».**
        //
        // **وكتبتُ الشاشةَ أوّلاً تعرض «مستحقك» فظهرت «٠ ل.س»** — قِيس
        // على الجهاز ٢٠٢٦-٠٨-٢٣. **وصفرٌ في موضع مالٍ أسوأُ من فراغ**:
        // يُقرأ عطباً في المنصّة أو بخساً في حقّه، **والحقيقةُ أنّه في
        // محفظته كاملاً.**
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.End,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            NextAction(order.status, busy, onAccept, onStart, onReady)
        }

        // **والاعتذارُ نصٌّ جانبيٌّ لا زرٌّ مساوٍ** — انظر أعلاه.
        if (order.status == "pending") {
            Spacer(Modifier.height(4.dp))
            RahalTextButton(onClick = onReject, enabled = !busy) {
                Text(stringResource(R.string.order_reject), color = Rahal.colors.inkMuted)
            }
        }
    }
}

/** **الزرُّ يتبع الحال** — واحدٌ في كلّ لحظة. */
@Composable
private fun NextAction(
    status: String,
    busy: Boolean,
    onAccept: () -> Unit,
    onStart: () -> Unit,
    onReady: () -> Unit,
) {
    when (status) {
        "pending" -> RahalButton(onClick = onAccept, enabled = !busy) {
            Text(stringResource(R.string.order_accept))
        }

        "accepted" -> RahalButton(onClick = onStart, enabled = !busy) {
            Text(stringResource(R.string.order_start))
        }

        "preparing" -> RahalButton(onClick = onReady, enabled = !busy) {
            Text(stringResource(R.string.order_ready))
        }

        // **وما بعد الجاهزيّة ليس بيده** — **وزرٌّ لا يفعل شيئاً أسوأُ
        // من غيابه**: يضغطه فلا يتغيّر شيءٌ فيظنّ التطبيقَ عطبان.
        else -> Text(
            stringResource(R.string.order_ready_done),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

/** **سطرُ صنفٍ بخياراته** — والخياراتُ جزءٌ ممّا يُصنع لا زينة. */
@Composable
private fun Line(line: OrderLine) {
    Column(Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "${line.qty}×",
                fontWeight = FontWeight.Bold,
                color = Rahal.colors.brand,
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(Modifier.size(6.dp))
            Text(line.name, style = MaterialTheme.typography.bodyMedium)
        }
        // ══════════════════════════════════════════════════════════════
        // **والخياراتُ تُقرأ أو يُصنع الطلبُ ناقصاً**
        // ══════════════════════════════════════════════════════════════
        //
        // «كبيرة» و«بلا بصل» و«جبنة إضافيّة» — **من لم يرها صنع صغيرةً
        // بالبصل بلا جبنة**، فيُردّ الطلبُ ويُحسب عليه.
        if (line.options.isNotEmpty()) {
            Text(
                line.options.joinToString("، ") { it.name },
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        if (line.note.isNotBlank()) {
            Text(
                line.note,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(4.dp))
    }
}
