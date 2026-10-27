package com.rahalgo.merchant.orders

import com.rahalgo.ui.Since
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.material3.Icon
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.foundation.layout.padding
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
import com.rahalgo.shared.merchant.commissionLabel
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
fun OrdersScreen(
    vm: OrdersViewModel,
    /** **يفتح «سجلّ الطلبات»** — المنتهي هناك لا هنا (تدقيقُ ٢٠٢٦-١٠-٢٧). */
    onHistory: (() -> Unit)? = null,
    /** **يفتح «متجري»** — من القائمة الفارغة: هناك مفتاحُ الفتح. */
    onStore: (() -> Unit)? = null,
) {
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
                    actionLabel = if (onStore != null) stringResource(R.string.orders_go_store) else "",
                    onAction = onStore,
                )
                HistoryLink(onHistory)
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

            // ══════════════════════════════════════════════════════════
            // **والطلباتُ بمجموعاتها — الجديدُ أوّلاً** (تدقيقُ ٢٠٢٦-١٠-٢٧)
            // ══════════════════════════════════════════════════════════
            //
            // **كانت قائمةً واحدةً بترتيب الخادم** — فطلبٌ جديدٌ ينتظر قراراً
            // قد يقع تحت طلبٍ يُحضَّر، **ومهلتُه تمضي وهو لا يُرى.**
            val grouped = vm.orders.groupBy { groupOf(it.status, it.readyAt) }
            OrderGroup.values().forEach { g ->
                val list = grouped[g].orEmpty()
                if (list.isEmpty()) return@forEach
                GroupHead(g, list.size)
                list.forEach { order ->
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
            }
            HistoryLink(onHistory)
            Spacer(Modifier.height(24.dp))
        }
    }
}

/** **عنوانُ مجموعةٍ بعددها** — «جديدة — بتستناك (2)». */
@Composable
private fun GroupHead(g: OrderGroup, count: Int) {
    Spacer(Modifier.height(16.dp))
    Text(
        stringResource(
            when (g) {
                OrderGroup.New -> R.string.orders_group_new
                OrderGroup.Preparing -> R.string.orders_group_preparing
                OrderGroup.Waiting -> R.string.orders_group_waiting
                OrderGroup.Closed -> R.string.orders_group_closed
            },
            count,
        ),
        color = if (g == OrderGroup.New) Rahal.colors.accent else Rahal.colors.inkMuted,
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.Bold,
    )
}

/** **رابطُ السجلّ تحت القائمة** — المنتهي هناك. */
@Composable
private fun HistoryLink(onHistory: (() -> Unit)?) {
    if (onHistory == null) return
    Spacer(Modifier.height(12.dp))
    RahalTextButton(onClick = onHistory, modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.orders_history_link))
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
        // **وفي صندوقٍ ملوّنٍ بأيقونة تنبيه** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت سطراً عادياً بين
        // الأصناف، **و«حساسية» تُقرأ بنظرةٍ أو لا تُقرأ.**
        if (order.notes.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.sm)
                    .background(Rahal.colors.warnTint)
                    .padding(horizontal = 10.dp, vertical = 8.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_warning),
                    contentDescription = null,
                    tint = Rahal.colors.accent,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(8.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.order_note),
                        color = Rahal.colors.accent,
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        order.notes,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
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
        // **وطلبٌ انتهى بلا بيعٍ لا حسبةَ فيه** (تدقيقُ ٢٠٢٦-١٠-٢٧) — انظر `noPayout`.
        if (noPayout(order.status)) {
            Text(
                stringResource(R.string.ord_no_payout),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
        } else {
            SumLine(
                label = stringResource(R.string.ord_sum_lbl),
                value = money(order.subtotal),
            )
            SumLine(
                label = stringResource(R.string.ord_cut_lbl, order.commissionLabel),
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
        }
        Spacer(Modifier.height(10.dp))

        // **والفعلُ التالي زرٌّ كبيرٌ بعرض البطاقة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان صغيراً على
        // الطرف، **وزرُّ الاعتذار تحته أعرضَ منه، فتذهب العينُ إلى الرفض أوّلاً.**
        NextAction(order.status, order.readyAt, busy, onAccept, onStart, onReady)

        // **والاعتذارُ نصٌّ جانبيٌّ لا زرٌّ مساوٍ** — انظر أعلاه.
        if (order.status == "pending") {
            var showReject by rememberSaveable(order.id) { mutableStateOf(false) }
            Spacer(Modifier.height(8.dp))
            // **وزرٌّ أحمرُ واضحٌ بعرض البطاقة** (طلبُ المالك ٢٠٢٦-١٠-٠١: «اعتذر عن
            // الطلب خلّيه زرّاً أحمرَ واضحاً لأنّه رفض») — **ويسأل عن السبب قبل أن يرفض.**
            // **ويبقى محدَّداً تحت زرّ القبول المملوء** — أحمرُ واضحٌ كما أراد المالك،
            // **والقبولُ أكبرُ منه وأثقل** (تدقيقُ ٢٠٢٦-١٠-٢٧).
            com.rahalgo.ui.RahalOutlineButton(
                onClick = { showReject = true },
                enabled = !busy,
                tone = com.rahalgo.ui.Tone.Danger,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.order_reject))
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

/** **الزرُّ الأساسيُّ بعرض البطاقة وارتفاعٍ يسع الإبهام.** */
private val PRIMARY = Modifier.fillMaxWidth().height(52.dp)

/** **الزرُّ يتبع الحال** — واحدٌ في كلّ لحظة. */
@Composable
private fun NextAction(
    status: String,
    readyAt: String?,
    busy: Boolean,
    onAccept: () -> Unit,
    onStart: () -> Unit,
    onReady: () -> Unit,
) {
    // ══════════════════════════════════════════════════════════════════
    // **و«جاهز للاستلام» من القبول حتّى وقوف السائق** (٢٠٢٦-١٠-١٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان الزرُّ في «يُحضَّر» وحدَها** — وفي وضع المتاجر يقفز القبولُ إلى
    // «بانتظار سائق» فلا يظهر أبداً. **والحالُ تبقى مكتوبةً بجانبه.**
    // انظر `Readiness.kt`.
    if (status != "pending" && status != "accepted" && canMarkReady(status, readyAt)) {
        Column(Modifier.fillMaxWidth()) {
            if (status != "preparing") {
                Text(
                    statusLabel(status),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.size(6.dp))
            }
            RahalButton(onClick = onReady, enabled = !busy, modifier = PRIMARY) {
                Text(stringResource(R.string.order_ready), fontWeight = FontWeight.Bold)
            }
        }
        return
    }
    if (status != "accepted" && readyAnnounced(status, readyAt)) {
        Text(
            stringResource(R.string.order_ready_done),
            color = Rahal.colors.success,
            style = MaterialTheme.typography.bodySmall,
        )
        return
    }
    when (status) {
        "pending" -> RahalButton(onClick = onAccept, enabled = !busy, modifier = PRIMARY) {
            Text(
                stringResource(R.string.order_accept),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
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
        // وهما شخصان.** **و«جاهز» بجانبه** — من حضّر سريعاً لا يمرّ بخطوتين.
        "accepted" -> Column(Modifier.fillMaxWidth()) {
            RahalButton(onClick = onStart, enabled = !busy, modifier = PRIMARY) {
                Text(stringResource(R.string.order_start), fontWeight = FontWeight.Bold)
            }
            if (canMarkReady(status, readyAt)) {
                RahalTextButton(onClick = onReady, enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.order_ready))
                }
            }
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
