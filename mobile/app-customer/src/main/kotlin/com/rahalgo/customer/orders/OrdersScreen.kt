package com.rahalgo.customer.orders

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.ComplaintReason
import com.rahalgo.shared.customer.MyOrder
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import com.rahalgo.ui.ScreenPad
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.OrderChatSheet
import com.rahalgo.ui.OrderChatViewModel

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتي — الجاري وحدَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وما انتهى في «سجلّ الطلبات»** — **ومن بحث عن طلبٍ يجري وسط مئةٍ
 * منتهيةٍ لا يجده.**
 */
@Composable
fun OrdersScreen(vm: OrdersViewModel) {
    OrdersList(
        vm = vm,
        list = vm.open,
        title = stringResource(R.string.nav_orders_mine),
        hint = stringResource(R.string.soon_orders),
        empty = stringResource(R.string.ord_none_open),
        history = false,
    )
}

/** **سجلُّ الطلبات — ما انتهى بفواتيره.** */
@Composable
fun HistoryScreen(vm: OrdersViewModel) {
    OrdersList(
        vm = vm,
        list = vm.history,
        title = stringResource(R.string.menu_history),
        hint = stringResource(R.string.soon_history),
        empty = stringResource(R.string.ord_none_history),
        history = true,
    )
}

@Composable
private fun OrdersList(
    vm: OrdersViewModel,
    list: List<MyOrder>,
    title: String,
    hint: String,
    empty: String,
    history: Boolean,
) {
    // **والنوافذُ تبقى بعد الدوران** — من كتب سببَ شكواه ثمّ أمال هاتفَه
    // لا يعيد كتابتَه. (وقع في تطبيق السائق ٢٠٢٦-٠٨-١٣.)
    var cancelId by rememberSaveable { mutableStateOf<String?>(null) }
    var rateId by rememberSaveable { mutableStateOf<String?>(null) }
    var complainId by rememberSaveable { mutableStateOf<String?>(null) }
    var chatId by rememberSaveable { mutableStateOf<String?>(null) }

    if (list.isEmpty() && vm.error.isNotEmpty()) {
        LoadState(loading = vm.busy, error = vm.error, onRetry = vm::load)
        return
    }

    // **والسحبُ يُنعش هنا أيضاً** — انظر `Refreshable`: **حركةٌ
    // يتعلّمها مرّةً ويتوقّعها في كلّ مكان، وشاشةٌ لا تُنعش تُقرأ
    // عطبا.**
    // **ولوحُ الحديث فوق القائمة** — (شكوى المالك ٢٠٢٦-٠٨-١٨).
    chatId?.let { id ->
        val chatVm: OrderChatViewModel = androidx.lifecycle.viewmodel.compose.viewModel()
        OrderChatSheet(vm = chatVm, orderId = id) { chatId = null }
    }

    // ══════════════════════════════════════════════════════════════════
    // **وقائمةٌ كسولةٌ — لا عمودٌ يبني كلَّ بطاقة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قِيس ٢٠٢٦-٠٨-٢٠ على جهاز المالك: **٩٩٪ من إطارات هذه الشاشة
    //  ضائعة** — والتسوّقُ بصورِه ١٧٪.)
    //
    // **وكانت `Screen` عموداً يُمرَّر** — **يبني كلَّ بطاقةٍ في القائمة
    // ولو كانت خارجَ الشاشة.** ولكلّ طلبٍ جارٍ حركةٌ لا تقف
    // (`Stages.kt`): **عشرُ حركاتٍ تعمل وتسعٌ منها لا تُرى.**
    //
    // **والكسولةُ تبني ما يُرى وحدَه** — فتقف الحركاتُ حين تخرج
    // بطاقاتُها.
    //
    // # ولا تُغيَّر الحشوةُ ولا الترتيب
    //
    // **حشوةُ `Screen` نفسُها** (`ScreenPad`) وذيلُها — **وشاشةٌ تتبدّل
    // هوامشُها مع إصلاحِ أداءٍ تُقرأ تغييرَ تصميم.**
    Refreshable(refreshing = vm.refreshing, onRefresh = vm::refresh) {
    LazyColumn(
        Modifier.fillMaxSize().padding(ScreenPad),
    ) {
        item {
            ScreenTitle(title, hint)

            // **وخطأُ الفعل يبقى ظاهراً** — لا تمحوه إعادةُ القراءة.
            if (vm.actionError.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                Note(vm.actionError, Rahal.colors.danger)
            }
            if (list.isEmpty()) Empty(empty)
        }

        items(list, key = { it.id }) { o ->
            Spacer(Modifier.height(10.dp))
            OrderCard(
                order = o,
                // ══════════════════════════════════════════════════════
                // **والإلغاءُ يُعرض حيث يُقبل — و«بلا مهلة» ليست «لا»**
                // ══════════════════════════════════════════════════════
                //
                // **والمهلةُ من الخادم** (`transitions.go`): **`-1`
                // تعني «بلا مهلة»** — قبل قبول المتجر يُلغي متى شاء،
                // **و`0` وحدَها تعني لا.**
                //
                // **وكان الشرطُ `> 0`** — **فاختفى الزرُّ في الحال
                // الوحيدة التي يملك فيها الإلغاءَ بلا قيد**: طلبٌ
                // بانتظار القبول. (شهده المالك ٢٠٢٦-٠٨-١٥: «ما في زرّ
                // إلغاء بالتطبيق صح؟».)
                onCancel = if (!history && o.cancelSecondsLeft != 0) {
                    { cancelId = o.id }
                } else {
                    null
                },
                onComplain = { complainId = o.id; vm.loadReasons(o.id) },
                // ══════════════════════════════════════════════════════
                // **وحديثُ السائق حيث يوجد سائق**
                // ══════════════════════════════════════════════════════
                //
                // (شكوى المالك ٢٠٢٦-٠٨-١٨: «أيقونةُ الدردشة لم تظهر عند
                //  الزبون».)
                //
                // **ولا يُعرض قبل أن يُسنَد أحد** — **زرُّ حديثٍ بلا
                // مُحدَّثٍ يُضغط فيُفتح فراغ.**
                //
                // **ولا في السجلّ**: طلبٌ انتهى وسائقُه مضى، **وحديثٌ
                // يُفتح فيه لا يقرؤه أحد.**
                // **ولا زرَّ حديثٍ في البطاقة** — (قرارُ المالك
                // ٢٠٢٦-٠٨-١٩: «يجب أن تكون أيقونةً عائمةً مثل
                //  السائق وليس داخلَ الكرت»). انظر `ChatFab`.
                onChat = null,
                // **ولا نجومَ لطلبٍ لم يُسلَّم ولا لطلبٍ قُيّم** — نجومٌ
                // مرّتين تُقرأ أنّ الأولى لم تصل.
                onRate = if (o.status == "delivered" && o.id !in vm.rated) {
                    { rateId = o.id }
                } else {
                    null
                },
            )
        }
        item { Spacer(Modifier.height(32.dp)) }
    }

    cancelId?.let { id ->
        ConfirmCancel(
            onConfirm = { vm.cancel(id); cancelId = null },
            onDismiss = { cancelId = null },
        )
    }
    rateId?.let { id ->
        // **والسائقُ يُقيَّم إن كان للطلب سائق** — ومن لم يوصله أحدٌ
        // لا يُسأل عمّن أوصله.
        val hasDriver = list.any { it.id == id && !it.driverName.isNullOrEmpty() }
        RateDialog(
            hasDriver = hasDriver,
            onConfirm = { stars, driverStars, note ->
                vm.rate(id, stars, driverStars, note)
                rateId = null
            },
            onDismiss = { rateId = null },
        )
    }
    complainId?.let { id ->
        ComplainDialog(
            reasons = vm.reasons,
            onConfirm = { reason, note -> vm.complain(id, reason, note); complainId = null },
            onDismiss = { complainId = null },
        )
    }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تأكيدُ الإلغاء — سؤالٌ واحدٌ لا استجواب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «الإلغاء بدون سبب، فقط تأكيد إلغاء. ما
 *  لنا علاقة بسبب الإلغاء — هو حرٌّ ما نُجبره على كتابة سبب».)
 *
 * **وكان حقلاً إلزاميّاً** — **وهو ضريبةٌ على حقٍّ يملكه**: من أراد
 * الإلغاء كتب حرفاً واحداً ليتخطّاه، **فلا نحن منعناه ولا عرفنا
 * شيئا.**
 *
 * **والتأكيدُ يبقى**: الإلغاءُ لا يُتراجع عنه، **وضغطةٌ واحدةٌ تُنهي
 * طلباً دفع فيه مالاً.**
 */
@Composable
private fun ConfirmCancel(onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.ord_cancel_title)) },
        text = { Text(stringResource(R.string.ord_cancel_ask)) },
        confirmButton = {
            RahalTextButton(onClick = onConfirm) {
                Text(stringResource(R.string.ord_cancel_yes), color = Rahal.colors.danger)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
        },
    )
}

/**
 * **التقييم — تقييمان وكلمة.**
 *
 * **الخدمةُ والسائق** (`orders/ratings.go`): الزبونُ لا يعرف المتجرَ
 * ولا يختاره — **يطلب من «رحّال غو» ونحن نختار من أين نشتري** — لكنّه
 * يعرف من وقف على بابه.
 *
 * **ونجمةٌ واحدةٌ لهما تُظلم أحدَهما**: طعامٌ بارد وسائقٌ سريعٌ يُقرآن
 * رقماً واحداً لا يُصلح شيئا.
 */
@Composable
internal fun RateDialog(
    hasDriver: Boolean,
    onConfirm: (Int, Int?, String) -> Unit,
    onDismiss: () -> Unit,
) {
    var stars by rememberSaveable { mutableIntStateOf(5) }
    var driverStars by rememberSaveable { mutableIntStateOf(5) }
    var note by rememberSaveable { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.ord_rate_title)) },
        text = {
            androidx.compose.foundation.layout.Column {
                Text(
                    text = stringResource(R.string.ord_rate_service),
                    color = Rahal.colors.ink,
                    style = MaterialTheme.typography.bodyMedium,
                )
                StarRow(stars) { stars = it }
                if (hasDriver) {
                    Text(
                        text = stringResource(R.string.ord_rate_driver),
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    StarRow(driverStars) { driverStars = it }
                }
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    label = { Text(stringResource(R.string.ord_rate_note)) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = {
                    onConfirm(stars, driverStars.takeIf { hasDriver }, note.trim())
                },
            ) { Text(stringResource(R.string.ord_send)) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
        },
    )
}

/** **صفُّ نجومٍ واحد** — ونسختان منه تفترقان يومَ يتبدّل شكلُ النجمة. */
@Composable
private fun StarRow(value: Int, onPick: (Int) -> Unit) {
    androidx.compose.foundation.layout.Row {
        (1..5).forEach { n ->
            RahalTextButton(onClick = { onPick(n) }) {
                Text(
                    text = if (n <= value) "★" else "☆",
                    style = MaterialTheme.typography.headlineSmall,
                    color = Rahal.colors.accent,
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشكوى — سببٌ من قائمةٍ لا نصٌّ حرّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «الشكوى لازم أسباب جاهزة وليس أن الزبون
 *  يكتب ماذا يريد».)
 *
 * **ومن السبب يُشتقّ الذنبُ والتعويض**: «نقصٌ في الطلب» فعلُ من عبّأ،
 * **و«لم أستلم» فعلُ من سلّم** — والرمزُ يقولها وحدَه. **ونصٌّ حرٌّ
 * يُقرأ بيدٍ بشريّةٍ ثمّ يُصنَّف بالظنّ.**
 *
 * **والقائمةُ من المحرّك** — تختلف بحال الطلب: **«لم أستلم طلبي» على
 * طلبٍ لم يُسلَّم لغو.**
 *
 * **والتفصيلُ اختياريٌّ إلّا مع «أخرى»** — كما في الويب حرفا:
 * **«أخرى» بلا شرحٍ ورقةٌ بيضاء.**
 */
@Composable
private fun ComplainDialog(
    reasons: List<ComplaintReason>,
    onConfirm: (String, String) -> Unit,
    onDismiss: () -> Unit,
) {
    var pick by rememberSaveable { mutableStateOf("") }
    var note by rememberSaveable { mutableStateOf("") }
    val needNote = pick == "other" && note.isBlank()
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.ord_complain_title)) },
        text = {
            androidx.compose.foundation.layout.Column {
                if (reasons.isEmpty()) {
                    Text(stringResource(R.string.ord_reasons_loading))
                }
                reasons.forEach { r ->
                    val on = r.code == pick
                    RahalTextButton(onClick = { pick = r.code }) {
                        Text(
                            text = (if (on) "● " else "○ ") + reasonLabel(r.code),
                            color = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
                        )
                    }
                }
                // **ولا يُسأل عن تفصيلٍ قبل أن يختار** — حقلُ كتابةٍ
                // فوق قائمةٍ يجعلها تُقرأ زينة.
                if (pick.isNotEmpty()) {
                    OutlinedTextField(
                        value = note,
                        onValueChange = { note = it },
                        label = {
                            Text(
                                stringResource(
                                    if (pick == "other") R.string.ord_complain_what
                                    else R.string.ord_complain_note,
                                ),
                            )
                        },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = { onConfirm(pick, note.trim()) },
                enabled = pick.isNotBlank() && !needNote,
            ) { Text(stringResource(R.string.ord_send)) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
        },
    )
}

/**
 * **اسمُ السبب بالعربيّة** — بألفاظ الويب حرفا (`site.complaint.reasons`).
 *
 * **ولفظان لسببٍ واحدٍ في شاشتين** يجعلان الزبونَ يشكّ أنّهما شيءٌ
 * واحد. **ورمزٌ لا نعرفه يُكتب كما جاء** — سببٌ يُزاد في المحرّك
 * يظهر بلا اسمٍ خيرٌ من أن يختفي.
 */
@Composable
private fun reasonLabel(code: String): String = when (code) {
    "not_received" -> stringResource(R.string.cr_not_received)
    "missing_items" -> stringResource(R.string.cr_missing_items)
    "wrong_items" -> stringResource(R.string.cr_wrong_items)
    "quality" -> stringResource(R.string.cr_quality)
    "late" -> stringResource(R.string.cr_late)
    "driver_conduct" -> stringResource(R.string.rs_driver_conduct)
    "money" -> stringResource(R.string.cr_money)
    "other" -> stringResource(R.string.cr_other)
    else -> code
}
