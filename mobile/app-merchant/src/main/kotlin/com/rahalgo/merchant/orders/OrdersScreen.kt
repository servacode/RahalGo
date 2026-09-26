package com.rahalgo.merchant.orders

import com.rahalgo.ui.Since
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.width
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.platform.LocalContext
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
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
import com.rahalgo.ui.SumLine
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.StatBox
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
            ScreenTitle(stringResource(R.string.nav_orders_mine), stringResource(R.string.orders_hint))
            LoadState(vm.loading, vm.error) { vm.refresh() }
        }
        return
    }

    Refreshable(refreshing = false, onRefresh = { vm.refresh() }) {
        Screen {
            ScreenTitle(stringResource(R.string.nav_orders_mine), stringResource(R.string.orders_hint))

            if (vm.orders.isEmpty()) {
                // **ولا طلبَ الآن — وماذا يفعل** (٢٠٢٦-٠٩-١٣).
                //
                // **وصاحبُ المتجر ينظر إلى سطرٍ يقول «لا طلبات»** ولا
                // يعرف أمفتوحٌ متجرُه أم مغلق. **فيُقال له أين يتأكّد.**
                Empty(
                    text = stringResource(R.string.orders_empty),
                    hint = stringResource(R.string.orders_empty_hint),
                )
                return@Screen
            }

            // **وطلبٌ فُتح من إشعاره يُبرَز ثمّ يخفت** (B3) — لا يبقى محاطاً
            // إلى الأبد. **ثمانِ ثوانٍ تكفي ليجده صاحبُ المتجر بنظرة.**
            LaunchedEffect(vm.focusedId) {
                if (vm.focusedId.isNotEmpty()) {
                    kotlinx.coroutines.delay(8_000)
                    vm.clearFocus()
                }
            }

            vm.orders.forEach { order ->
                Spacer(Modifier.height(10.dp))
                OrderCard(
                    order = order,
                    busy = vm.busy(order.id),
                    highlighted = order.id == vm.focusedId,
                    onAccept = { vm.accept(order.id) },
                    onStart = { vm.startPreparing(order.id) },
                    onReady = { vm.markReady(order.id) },
                    onReject = { note -> vm.reject(order.id, note) },
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
    highlighted: Boolean,
    onAccept: () -> Unit,
    onStart: () -> Unit,
    onReady: () -> Unit,
    onReject: (String) -> Unit,
) {
    // **وطلبٌ فُتح من إشعاره محاطٌ بلون العلامة** (B3) — يجده صاحبُ المتجر
    // بنظرةٍ بين طلباتٍ كثيرة.
    val cardModifier = if (highlighted) {
        Modifier.border(2.dp, Rahal.colors.brand, Rahal.shape.md)
    } else {
        Modifier
    }
    Card(modifier = cardModifier) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                "#" + order.number,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )

            // ══════════════════════════════════════════════════════════
            // **وكم مضى عليه — بجانب رقمه**
            // ══════════════════════════════════════════════════════════
            //
            // **(طلبُ المالك ٢٠٢٦-٠٩-٠٢.)** وكانت نصوصُه مكتوبةً في
            // المتجر منذ زمنٍ **ولا يستعملها أحد** — ميزةٌ نُويت ولم
            // تُبنَ.
            //
            // **ومتجرٌ أمامه خمسةُ طلباتٍ لا يعرف أيُّها انتظر أطول**
            // — فيبدأ بأعلاها في القائمة لا بأقدمها، **ويبرد طعامُ
            // من سبق.**
            //
            // **وتُقرأ في الإدارة أيضاً**: متجرٌ تُشيخ طلباتُه على
            // بطاقته يُقاس بها.
            val age = Since.text(LocalContext.current, order.createdAt)
            if (age.isNotEmpty()) {
                Spacer(Modifier.width(8.dp))
                Text(
                    age,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }

            Spacer(Modifier.weight(1f))
            // ══════════════════════════════════════════════════════════
            // **وكم يقبض منه — على البطاقة لا في السجلّ وحده**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «سجلّ الطلبات ما فيه شقد المبلغ
            //  المباع وشقد نسبة العمولة، هيك لازم يكون بشفافية».)
            //
            // **ووُضع في السجلّ ولم يوضع هنا** — فيرى صاحبُ المتجر ما
            // قبضه بعد أن يمضي الطلب، **ولا يراه وهو يقرّر أن يقبله.**
            //
            // **وصافيه لا مجموعه**: المجموعُ يحمل أجرةَ السائق ولا تخصّه،
            // **ورقمٌ أكبرُ ممّا يقبض يُقرأ وعداً لا يُوفى.**
            //
            // **وثلاثةُ أرقامٍ لا رقمٌ واحد** — (طلبُ المالك ٢٠٢٦-٠٨-٣١:
            // «يعرف شو سعر طلبه وأصنافه، ويعرف شو نخصم وشو باقي له»).
            //
            // **ورقمٌ واحدٌ لا يُراجَع**: يقرأ «لك ١٣٥» ولا يعرف من أين
            // جاءت، **فإن شكّ لم يجد ما يطرحه.**
            // **وثلاثةُ مربّعاتٍ لا ثلاثةُ أسطر** — (بلاغُ المالك
            // ٢٠٢٦-٠٨-٣١: «شكلُ المربّع ما عجبني، تنسيقُ الكتابة فيه —
            // يعني لازم تكون واضحةً متناسقةً مفهومة»).
            //
            // **وثلاثةُ أسطرٍ بأحجامٍ وألوانٍ مختلفةٍ تُقرأ ثلاثةَ
            // أشياءَ لا حسبةً واحدة.** **والمربّعاتُ تصفّها بحجمٍ واحدٍ
            // ومحاذاةٍ واحدة**، فيُقرأ الطرحُ بنظرة.
            // **ولا عدّادَ مهلةٍ في هذا الباب** — المحرّكُ لا يرسله مع
            // الطلب، **وعدٌّ تحسبه الشاشةُ من وقت الإنشاء يكذب**: ساعةُ
            // الجهاز تفترق عن ساعة الخادم بدقائق. **فيُقرأ منقضياً وهو
            // حيّ، فيتردّد صاحبُ المتجر عن طلبٍ ما زال أمامه وقت.**
        }

        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.mn_items),
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

        // ══════════════════════════════════════════════════════════════
        // **والحسبةُ بعد الأصناف — كما في كلّ فاتورة**
        // ══════════════════════════════════════════════════════════════
        //
        // **(بلاغُ المالك ٢٠٢٦-٠٨-٣١:** «ما عجبتني الترتيبة، مو
        // احترافيّة» · «المهمّ يكون الشكلُ احترافيّاً مفهوماً واضحا».)
        //
        // **وكانت الأرقامُ فوق الأصناف** — يقرأ «المستحقّ ١٣٥» **ثمّ**
        // يعرف ماذا باع. **والقراءةُ الطبيعيّةُ عكسُها**: ما بِيع، ثمّ
        // كم يُجمع، ثمّ كم يُخصم، ثمّ ما يبقى.
        //
        // **والخصمُ بإشارة ناقص** — فيُرى الطرحُ ولا يُستنتج.
        Spacer(Modifier.height(10.dp))
        HorizontalDivider()
        Spacer(Modifier.height(8.dp))
        SumLine(
            label = stringResource(R.string.ord_sum_lbl),
            value = money(order.subtotal),
        )
        SumLine(
            label = stringResource(R.string.ord_cut_lbl, order.commissionPercent),
            value = "− " + money(order.platformCommission),
        )
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
        Spacer(Modifier.height(2.dp))
        // **والمحصّلةُ خلف الخطّ وبلون العلامة** — هو الرقمُ الذي يعنيه.
        SumLine(
            label = stringResource(R.string.ord_due_lbl),
            value = money(order.merchantNet),
            strong = true,
            color = Rahal.colors.brand,
        )
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
            var showReject by remember { mutableStateOf(false) }
            Spacer(Modifier.height(4.dp))
            RahalTextButton(onClick = { showReject = true }, enabled = !busy) {
                Text(stringResource(R.string.order_reject), color = Rahal.colors.inkMuted)
            }
            if (showReject) {
                RejectReasonDialog(
                    onDismiss = { showReject = false },
                    onPick = { reason ->
                        showReject = false
                        onReject(reason)
                    },
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مُنتقي سببِ الاعتذار — سريعٌ لا حرٌّ** (B7، ٢٠٢٦-٠٩-٢٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك: أسبابٌ جاهزةٌ يختار منها بضغطة — الصنفُ غير متوفّر · ضغطُ
 *  طلبات · تعذّر التجهيز · المتجر على وشك الإغلاق — و«سببٌ آخر» بنصٍّ حرّ.)
 *
 * # ولماذا قائمةٌ لا حقلٌ فارغ
 *
 * **كان الاعتذارُ يرسل نصّاً واحداً محفوظاً** — يصل الزبونَ كلَّ مرّةٍ سواءً،
 * **فلا يعرف لماذا اعتُذر عنه ولا نقيس متجراً يُكثر سبباً بعينه.** وحقلٌ حرٌّ
 * وحدَه بطيءٌ على يدٍ في العجين. **فالجاهزُ ضغطةٌ، والحرُّ لمن أراد.**
 *
 * **والنصُّ العربيُّ نفسُه يُرسَل ويُحفَظ ويصل الزبون** (`cancel_reason`) —
 * لا رمزٌ يُترجَم في موضعين فيفترقان.
 */
@Composable
private fun RejectReasonDialog(
    onDismiss: () -> Unit,
    onPick: (String) -> Unit,
) {
    val canned = listOf(
        stringResource(R.string.order_reject_r_unavailable),
        stringResource(R.string.order_reject_r_busy),
        stringResource(R.string.order_reject_r_cantprep),
        stringResource(R.string.order_reject_r_closing),
    )
    var custom by remember { mutableStateOf(false) }
    var customText by rememberSaveable { mutableStateOf("") }

    androidx.compose.material3.AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.order_reject_title)) },
        text = {
            Column(Modifier.fillMaxWidth()) {
                if (!custom) {
                    canned.forEach { reason ->
                        RahalTextButton(
                            onClick = { onPick(reason) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Row(Modifier.fillMaxWidth()) {
                                Text(reason)
                            }
                        }
                    }
                    // **«سببٌ آخر» يكشف الحقلَ الحرّ** — لمن لا يجد سببَه أعلاه.
                    RahalTextButton(
                        onClick = { custom = true },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Row(Modifier.fillMaxWidth()) {
                            Text(
                                stringResource(R.string.order_reject_r_other),
                                color = Rahal.colors.inkMuted,
                            )
                        }
                    }
                } else {
                    androidx.compose.material3.OutlinedTextField(
                        value = customText,
                        onValueChange = { customText = it },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text(stringResource(R.string.order_reject_custom_hint)) },
                        singleLine = false,
                    )
                }
            }
        },
        confirmButton = {
            if (custom) {
                RahalButton(
                    onClick = { onPick(customText.trim()) },
                    enabled = customText.isNotBlank(),
                ) {
                    Text(stringResource(R.string.order_reject_send))
                }
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) {
                Text(stringResource(com.rahalgo.ui.R.string.act_cancel))
            }
        },
    )
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

        // ══════════════════════════════════════════════════════════════
        // **ولا زرَّ لـ«بدأتُ التحضير»**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٢٩.)
        //
        // **والقبولُ في وضع المتاجر يدخل التحضيرَ بنفسه** — يفعله
        // المحرّك (`autoPreparing`). **فالحالةُ «مقبول» لا تُرى إلّا
        // لحظةً**، وتبقى هنا لوضع المنصّة: **المكتبُ يقبل والمتجرُ يبدأ،
        // وهما شخصان.**
        "accepted" -> RahalButton(onClick = onStart, enabled = !busy) {
            Text(stringResource(R.string.order_start))
        }

        "preparing" -> RahalButton(onClick = onReady, enabled = !busy) {
            Text(stringResource(R.string.order_ready))
        }

        // **وما بعد الجاهزيّة ليس بيده** — **وزرٌّ لا يفعل شيئاً أسوأُ
        // من غيابه**: يضغطه فلا يتغيّر شيءٌ فيظنّ التطبيقَ عطبان.
        // ══════════════════════════════════════════════════════════════
        // **وكلُّ حالةٍ تقول اسمَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٩: «الاعتذار ما ينتظر المتجرُ سائقاً،
        //  سيكون ملغيَّ الطلب».)
        //
        // **وكان الفرعُ الأخير يقول «بانتظار السائق» لكلّ ما ليس من
        // الثلاث** — **فالمرفوضُ والملغيُّ والمسلَّمُ كلُّها تنتظر
        // سائقاً.**
        //
        // **والمتجرُ يجلب طلباتِه كلَّها لا المفتوحةَ وحدَها** — فالحالاتُ
        // الأربعَ عشرةَ تصل هذه الشاشة.
        //
        // **ومن رأى طلباً اعتذر عنه «ينتظر سائقاً» ظنّ اعتذارَه لم
        // يُسجَّل** — فيعتذر ثانيةً، أو ينتظر سائقاً لن يأتي.
        else -> Text(
            statusLabel(status),
            color = when (status) {
                "rejected", "cancelled", "failed" -> Rahal.colors.danger
                "delivered" -> Rahal.colors.success
                else -> Rahal.colors.inkMuted
            },
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


/** **اسمُ الحالة كما يراها صاحبُ المتجر** — لا كما يسمّيها المحرّك. */
@Composable
private fun statusLabel(status: String): String = stringResource(
    when (status) {
        "dispatching" -> R.string.os_dispatching
        "assigned" -> R.string.os_assigned
        "at_pickup" -> R.string.os_at_pickup
        "picked_up" -> R.string.os_picked_up
        "on_the_way" -> R.string.os_on_way
        "at_dropoff" -> R.string.os_at_dropoff
        "delivered" -> com.rahalgo.ui.R.string.ord_st_delivered
        "rejected" -> R.string.os_rejected
        "cancelled" -> R.string.os_cancelled
        "failed" -> R.string.os_failed
        "refunded" -> R.string.os_refunded
        else -> R.string.order_ready_done
    },
)
