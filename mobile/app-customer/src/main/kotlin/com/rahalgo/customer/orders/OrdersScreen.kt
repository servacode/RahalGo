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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.foundation.layout.size
import com.rahalgo.ui.RahalButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.verticalScroll
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
fun OrdersScreen(
    vm: OrdersViewModel,
    walletBalance: Long = 0,
    /** **«تصفّح السوق» من الفراغ** — ينقل لتبويب التسوّق. */
    onBrowse: () -> Unit = {},
) {
    // ══════════════════════════════════════════════════════════════════
    // **«جارية · منتهية» فوق «طلباتي»** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **من أراد طلبَ الأمس ليقيّمه أو يشتكي عليه فتح «طلباتي» فوجدها فارغة** —
    // والسجلُّ في القائمة الجانبيّة. **فصار حبّتين بالعدد هنا، وبندُ القائمة
    // باقٍ اختصاراً.** **والقائمتان من جلبٍ واحد** (`load`) — لا نداءَ ثانٍ.
    var showHistory by rememberSaveable { mutableStateOf(false) }
    OrdersList(
        vm = vm,
        list = if (showHistory) vm.history else vm.open,
        title = stringResource(R.string.nav_orders_mine),
        // **ولا سطرَ شرحٍ تحت العنوان** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
        hint = "",
        empty = stringResource(if (showHistory) R.string.ord_none_history else R.string.ord_none_open),
        history = showHistory,
        // **والسجلُّ منتهٍ — لا تأكيدَ عرضٍ فيه، فلا حاجةَ لرصيد.**
        walletBalance = if (showHistory) 0 else walletBalance,
        onBrowse = if (showHistory) null else onBrowse,
        tabs = {
            Spacer(Modifier.height(4.dp))
            androidx.compose.foundation.layout.Row(
                horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(8.dp),
            ) {
                TabPill(
                    text = stringResource(R.string.ord_tab_open, vm.open.size.toString()),
                    on = !showHistory,
                ) { showHistory = false }
                TabPill(
                    text = stringResource(
                        R.string.ord_tab_history,
                        vm.history.size.toString() + if (vm.historyHasMore) "+" else "",
                    ),
                    on = showHistory,
                ) { showHistory = true }
            }
            Spacer(Modifier.height(4.dp))
        },
    )
}

/** **حبّةُ اختيار** — المختارةُ بلون العلامة والأخرى بإطار، كفلاتر لوحة الإدارة. */
@Composable
private fun TabPill(text: String, on: Boolean, onClick: () -> Unit) {
    Text(
        text = text,
        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
        fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
        style = MaterialTheme.typography.labelLarge,
        modifier = Modifier
            .clip(Rahal.shape.pill)
            .then(
                if (on) Modifier.background(Rahal.colors.brand)
                else Modifier.border(1.dp, Rahal.colors.line, Rahal.shape.pill),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
    )
}

/**
 * **كم ثانيةً بقيت من مهلة الإلغاء الآن** — تمشي والشاشةُ مفتوحة (تدقيقُ ٢٠٢٦-١٠-٢٧).
 *
 * **وكانت رقمَ آخرِ جلب** فيبقى الزرُّ بعد انقضائها ويُرَدّ الضغط. **و`-1` «بلا مهلة»
 * و`0` «لا» يمرّان كما هما.**
 */
@Composable
private fun cancelLeft(secs: Int, loadedAt: Long): Int {
    if (secs <= 0) return secs
    val left by androidx.compose.runtime.produceState(remain(secs, loadedAt), secs, loadedAt) {
        while (value > 0) {
            kotlinx.coroutines.delay(1000)
            value = remain(secs, loadedAt)
        }
    }
    return left
}

private fun remain(secs: Int, at: Long): Int {
    val gone = if (at <= 0L) 0L else (android.os.SystemClock.elapsedRealtime() - at) / 1000L
    return (secs - gone).toInt().coerceAtLeast(0)
}

/** **سجلُّ الطلبات — ما انتهى بفواتيره.** */
@Composable
fun HistoryScreen(vm: OrdersViewModel) {
    OrdersList(
        vm = vm,
        list = vm.history,
        // **والعنوانُ في رأس الصفحة بسهم رجوع، ولا سطرَ شرح** (٢٠٢٦-١٠-٣٠).
        title = "",
        hint = "",
        empty = stringResource(R.string.ord_none_history),
        history = true,
        // **والسجلُّ منتهٍ — لا تأكيدَ عرضٍ فيه، فلا حاجةَ لرصيد.**
        walletBalance = 0,
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
    walletBalance: Long,
    /** **زرُّ «تصفّح السوق» تحت الفراغ** — و`null` لا زرّ. */
    onBrowse: (() -> Unit)? = null,
    /** **حبّتا «جارية · منتهية»** تحت العنوان — و`null` في شاشة السجلّ وحدَها. */
    tabs: (@Composable () -> Unit)? = null,
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
            tabs?.invoke()

            // **وخطأُ الفعل يبقى ظاهراً** — لا تمحوه إعادةُ القراءة.
            if (vm.actionError.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                Note(vm.actionError, Rahal.colors.danger)
            }
            // **ولا «لا طلبات» قبل أوّل جواب** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت تومض لحظةً
            // فيظنّ من طلب الآن أنّ طلبَه ضاع.
            if (list.isEmpty() && !vm.loadedOnce) {
                androidx.compose.foundation.layout.Box(
                    Modifier.fillMaxWidth().padding(vertical = 40.dp),
                    contentAlignment = androidx.compose.ui.Alignment.Center,
                ) { com.rahalgo.ui.RahalLoader() }
            } else if (list.isEmpty()) {
                Empty(
                    text = empty,
                    actionLabel = if (onBrowse != null) stringResource(R.string.act_browse_market) else "",
                    onAction = onBrowse,
                )
            }
        }

        items(list, key = { it.id }) { o ->
            Spacer(Modifier.height(10.dp))
            val left = if (history) 0 else cancelLeft(o.cancelSecondsLeft, vm.loadedAt)
            OrderCard(
                order = o,
                // **والمهلةُ سطرٌ مفهومٌ جنب الحال** (قرارُ المالك ٢٠٢٦-١٠-٣٠) —
                // «يمكنك إلغاء الطلب خلال 46 ثانية»، ويغيب مع الزرّ عند الصفر.
                cancelHint = if (left > 0) {
                    stringResource(R.string.ord_cancel_within, CancelCountdown.within(left))
                } else {
                    null
                },
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
                onCancel = if (!history && left != 0) {
                    { vm.clearAction(); cancelId = o.id }
                } else {
                    null
                },
                onComplain = { vm.clearAction(); complainId = o.id; vm.loadReasons(o.id) },
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
                    { vm.clearAction(); rateId = o.id }
                } else {
                    null
                },
                // **وتأكيدُ عرض الطلب المخصَّص** (Batch 2c) — في الجاري وحدَه.
                walletBalance = walletBalance,
                onConfirmQuote = if (!history && o.kind == "custom") {
                    { method -> vm.confirmQuote(o, method) }
                } else {
                    null
                },
                // **ورفضُ السعر الجديد بعد الاستلام** (قرارُ المالك ٢٠٢٦-١٠-١٥).
                onRejectQuote = if (!history && o.kind == "custom") {
                    { vm.rejectQuote(o.id) }
                } else {
                    null
                },
                confirming = vm.acting,
            )
        }
        // **مزيدٌ من السجلّ — صفحةً صفحة** (`CAF-14`/`CUST-14-026`).
        // **ولا يظهر إلّا في السجلّ وحين يبقى ما يُجلب.**
        if (history && vm.historyHasMore) {
            item {
                Spacer(Modifier.height(12.dp))
                RahalButton(
                    onClick = { vm.loadMore() },
                    enabled = !vm.loadingMore,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (vm.loadingMore) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text(stringResource(R.string.ord_load_more))
                    }
                }
            }
        }
        item { Spacer(Modifier.height(32.dp)) }
    }

    // ══════════════════════════════════════════════════════════════════
    // **والنافذةُ تُغلق بعد نجاح الفعل لا قبله** (تدقيقُ ٢٠٢٦-١٠-٢٨)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كانت تُغلق فورَ الضغط** — فإن سقط الفعلُ أو رُدّ ضاع بصمت. **والآن تبقى
    // مفتوحةً وزرُّها معطّلٌ حتّى يرجع الجواب، وخطؤه يُكتب فيها.**
    cancelId?.let { id ->
        ConfirmCancel(
            sending = vm.acting,
            error = vm.actionError,
            onConfirm = { vm.cancel(id) { cancelId = null } },
            onDismiss = { cancelId = null; vm.clearAction() },
        )
    }
    rateId?.let { id ->
        // **والسائقُ يُقيَّم إن كان للطلب سائق** — ومن لم يوصله أحدٌ
        // لا يُسأل عمّن أوصله.
        val hasDriver = list.any { it.id == id && !it.driverName.isNullOrEmpty() }
        RateDialog(
            hasDriver = hasDriver,
            sending = vm.acting,
            error = vm.actionError,
            onConfirm = { stars, driverStars ->
                vm.rate(id, stars, driverStars) { rateId = null }
            },
            onDismiss = { rateId = null; vm.clearAction() },
        )
    }
    complainId?.let { id ->
        ComplainDialog(
            reasons = vm.reasons,
            loading = vm.reasonsLoading,
            failed = vm.reasonsFailed,
            sending = vm.acting,
            error = vm.actionError,
            onRetry = { vm.loadReasons(id) },
            onConfirm = { reason, note -> vm.complain(id, reason, note) { complainId = null } },
            onDismiss = { complainId = null; vm.clearAction() },
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
private fun ConfirmCancel(
    sending: Boolean,
    error: String,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.ord_cancel_title)) },
        text = {
            androidx.compose.foundation.layout.Column {
                Text(stringResource(R.string.ord_cancel_ask))
                DialogError(error)
            }
        },
        confirmButton = {
            RahalTextButton(onClick = onConfirm, enabled = !sending) {
                Text(stringResource(R.string.ord_cancel_yes), color = Rahal.colors.danger)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
        },
    )
}

/** **خطأُ الفعل داخل نافذته** — فلا يُخفيه ظهرُها فوق القائمة. */
@Composable
private fun DialogError(error: String) {
    if (error.isEmpty()) return
    Spacer(Modifier.height(8.dp))
    Text(error, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
}

/**
 * **التقييم — تقييمان بلا كلمة.**
 *
 * **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «اتّفقنا سابقاً أنّه تقييمٌ بدون أيّ
 * تعليقٍ بأيّ تطبيق — يعني فقط تقييم».)
 *
 * # وحقلُ الكلمة كان يُجمع ويُدفن
 *
 * **كان «كلمة إن أردت» يُكتب ويُرسل ويُخزَّن** — **ولا شاشةَ في المنصّة
 * تعرضه**: `Review.comment` في النموذج، **وبطاقةُ السائق لا تقرؤه.**
 *
 * **وحقلٌ يُطلب ولا يُقرأ أسوأُ من غيابه**: يكتب الزبونُ شكواه فيه
 * ظانّاً أنّ أحداً سيقرؤها، **ثمّ لا يردّ عليه أحد.** وبابُ الشكوى
 * غيرُه، **فيضيع بلاغُه في حقلٍ ميّت.**
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
    sending: Boolean = false,
    error: String = "",
    onConfirm: (Int, Int?) -> Unit,
    onDismiss: () -> Unit,
) {
    // ══════════════════════════════════════════════════════════════════
    // **والنجومُ تبدأ فارغة** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كانت تبدأ خمساً** فمن ضغط «إرسال» دغري أرسل خمسَ نجومٍ وهو لم يقيّم —
    // **فيتضخّم المتوسّطُ في اللوحة ويختفي الضعيف.** فالزرُّ معطّلٌ حتّى يختار.
    var stars by rememberSaveable { mutableIntStateOf(0) }
    var driverStars by rememberSaveable { mutableIntStateOf(0) }
    val ready = stars > 0 && (!hasDriver || driverStars > 0)
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.ord_rate_title)) },
        text = {
            androidx.compose.foundation.layout.Column {
                Text(
                    text = stringResource(R.string.ord_rate_service),
                    color = Rahal.colors.ink,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Bold,
                )
                StarRow(stars) { stars = it }
                if (hasDriver) {
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = stringResource(R.string.ord_rate_driver),
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    StarRow(driverStars) { driverStars = it }
                }
                DialogError(error)
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = {
                    onConfirm(stars, driverStars.takeIf { hasDriver })
                },
                enabled = ready && !sending,
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
    // **أيقونةُ نجمةٍ حقيقيّة بحجم ٤٠** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — كانت «★/☆» نصّاً
    // يتبدّل شكلُه من جهازٍ لجهاز. **والفارغةُ بلون الحدّ والمختارةُ برتقاليّة.**
    androidx.compose.foundation.layout.Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
        (1..5).forEach { n ->
            val label = stringResource(R.string.rate_star_n, n.toString())
            androidx.compose.foundation.layout.Box(
                Modifier
                    .size(44.dp)
                    .clip(androidx.compose.foundation.shape.CircleShape)
                    .clickable(onClickLabel = label) { onPick(n) },
                contentAlignment = androidx.compose.ui.Alignment.Center,
            ) {
                androidx.compose.material3.Icon(
                    painter = androidx.compose.ui.res.painterResource(com.rahalgo.ui.R.drawable.ic_star),
                    contentDescription = label,
                    tint = if (n <= value) Rahal.colors.accent else Rahal.colors.line,
                    modifier = Modifier.size(40.dp),
                )
            }
        }
    }
    // **وكلمةٌ تحت النجوم تقول ما اختار** — «سيئ، مقبول، منيح، ممتاز».
    Text(
        text = stringResource(
            when (value) {
                1 -> R.string.rate_word_1
                2 -> R.string.rate_word_2
                3 -> R.string.rate_word_3
                4 -> R.string.rate_word_4
                5 -> R.string.rate_word_5
                else -> R.string.rate_pick
            },
        ),
        color = if (value > 0) Rahal.colors.accent else Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
        fontWeight = if (value > 0) FontWeight.Bold else FontWeight.Normal,
    )
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
    loading: Boolean,
    failed: Boolean,
    sending: Boolean,
    error: String,
    onRetry: () -> Unit,
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
            androidx.compose.foundation.layout.Column(
                Modifier.verticalScroll(androidx.compose.foundation.rememberScrollState()),
            ) {
                // **«عم نجيب» أم «ما قدرنا» مع «إعادة»** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت النافذةُ
                // تعلق على «تجلب الأسباب…» إلى الأبد إن سقط النداء.
                if (reasons.isEmpty() && failed && !loading) {
                    Text(stringResource(R.string.ord_reasons_failed), color = Rahal.colors.danger)
                    RahalTextButton(onClick = onRetry) {
                        Text(stringResource(com.rahalgo.ui.R.string.act_retry))
                    }
                } else if (reasons.isEmpty()) {
                    Text(stringResource(R.string.ord_reasons_loading), color = Rahal.colors.inkMuted)
                }
                // **صفوفٌ بزرّ اختيارٍ حقيقيّ** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — كانت «● ○»
                // حروفاً لا يعرفها قارئُ الشاشة اختياراً. **كبطاقتي الدفع في السلّة.**
                reasons.forEach { r ->
                    val on = r.code == pick
                    androidx.compose.foundation.layout.Row(
                        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.sm)
                            .selectable(
                                selected = on,
                                onClick = { pick = r.code },
                                role = androidx.compose.ui.semantics.Role.RadioButton,
                            )
                            .padding(vertical = 2.dp),
                    ) {
                        androidx.compose.material3.RadioButton(selected = on, onClick = null)
                        Text(
                            text = reasonLabel(r.code),
                            color = if (on) Rahal.colors.ink else Rahal.colors.inkMuted,
                            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
                // **ولا يُسأل عن تفصيلٍ قبل أن يختار** — حقلُ كتابةٍ
                // فوق قائمةٍ يجعلها تُقرأ زينة.
                if (pick.isNotEmpty()) {
                    OutlinedTextField(
                        value = note,
                        // **وبحدّ المحرّك** — انظر `TextLimits`.
                        onValueChange = {
                            note = com.rahalgo.shared.model.TextLimits.fit(it, com.rahalgo.shared.model.TextLimits.COMPLAINT_NOTE)
                        },
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
                DialogError(error)
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = { onConfirm(pick, note.trim()) },
                enabled = pick.isNotBlank() && !needNote && !sending,
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
 * واحد. **ورمزٌ لا نعرفه يُقرأ «أخرى» لا رمزَه** (٢٠٢٦-١٠-١٨) — والحارسُ
 * (`RawCodeTest`) يقرأ أسبابَ المحرّك فيكشف سبباً زِيد بلا اسم.
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
    else -> stringResource(R.string.cr_other)
}
