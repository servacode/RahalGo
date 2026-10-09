package com.rahalgo.merchant.delivery

import android.app.Application
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.merchant.SelectedStore
import com.rahalgo.shared.merchant.DeliveriesApi
import com.rahalgo.shared.merchant.Delivery
import com.rahalgo.shared.merchant.DeliveryQuote
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.NewDelivery
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Attempt
import com.rahalgo.ui.Card
import com.rahalgo.ui.Note
import com.rahalgo.ui.PhoneField
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.SectionTitle
import com.rahalgo.ui.apiError
import com.rahalgo.ui.isDecided
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«لدي توصيلة» — تبويبٌ في الشريط السفليّ** (الخطوة ١٨، ٢٠٢٦-١٠-٠١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٩-٢٩ · وموضعُ الزرّ: «الزرّ يكون بالشريط السفلي».)
 *
 * **زبونٌ اشترى خارجَ رحّال غو والغرضُ جاهز** — يُكتب المستلِمُ وعنوانُه ووصفُ
 * ما يُحمل ومن يدفع، **وتُقال الأجرةُ قبل الإرسال**، ثمّ تمضي إلى السائقين.
 *
 * **وثلاثةُ دافعين** (نصُّ المالك: «لازم في أنا نقدي»): محفظتي · أنا نقداً ·
 * المستلِمُ نقداً. **ونقطةُ التسليم اختياريّة** («يمكن لا يملك عنوانَ المستلِم
 * على الخريطة») — **فالأجرةُ من منطقة المتجر، والسائقُ يتّصل بالمستلِم.**
 *
 * **ودروسُ فحص المندوب مبنيّةٌ فيه من أوّل سطر**: ما ينقص يُسمّى تحت الزرّ ·
 * خطأُ الإرسال فوق الزرّ · «تمّ» يُقال · مفتاحُ منع التكرار يثبت حتّى النجاح ·
 * **والفرعُ المختارُ يُقرأ في كلّ تحميل** (`SelectedStore.resolve`).
 */
class DeliveryViewModel(app: Application) : AndroidViewModel(app) {

    private val core = AppCore.get()
    private val stores = MerchantApi(core.api)
    private val api = DeliveriesApi(core.api)

    var storeId by mutableStateOf("")
        private set

    var name by mutableStateOf("")
    var phone by mutableStateOf("")
    var address by mutableStateOf("")
    var parcel by mutableStateOf("")
    var driverNote by mutableStateOf("")

    /** `merchant` · `merchant_cash` · `recipient` — **ولا افتراض**: يُختار عن قصد. */
    var payer by mutableStateOf("")
        private set

    var point by mutableStateOf<Pair<Double, Double>?>(null)
        private set
    var pointLabel by mutableStateOf("")
        private set

    // ══════════════════════════════════════════════════════════════════
    // **رابطُ موقع المستلِم** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
    // ══════════════════════════════════════════════════════════════════
    //
    // «الكلّ يعطي مشاركة» — يلصق المتجرُ ما شاركه الزبونُ بواتساب، **فيُقرأ
    // نقطةً** بدل البحث عنها في خريطةٍ لا يعرفها. **والخريطةُ احتياط.**
    var link by mutableStateOf("")
        private set
    var linkState by mutableStateOf(LinkState.Idle)
        private set
    private var linkJob: kotlinx.coroutines.Job? = null

    fun onLink(text: String) {
        link = text.take(2000)
        touched()
        linkJob?.cancel()
        if (!link.contains("http")) {
            linkState = LinkState.Idle
            if (pointFromLink) clearPoint()
            return
        }
        linkState = LinkState.Checking
        linkJob = viewModelScope.launch {
            kotlinx.coroutines.delay(400)
            val r = runCatching { api.resolveLocation(storeId, link) }.getOrNull()
            if (r != null && r.found) {
                setPoint(r.lat, r.lng, "")
                pointFromLink = true
                linkState = LinkState.Found
            } else {
                if (pointFromLink) clearPoint()
                linkState = LinkState.NotFound
            }
        }
    }

    private var pointFromLink = false

    var quote by mutableStateOf<DeliveryQuote?>(null)
        private set
    var quoteError by mutableStateOf("")
        private set

    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var sent by mutableStateOf(false)
        private set

    var list by mutableStateOf<List<Delivery>?>(null)
        private set
    var listError by mutableStateOf("")
        private set
    var cancelling by mutableStateOf<String?>(null)
        private set

    init {
        load()
        // **وتبديلُ الفرع يُعيد القراءة** — لا تبقى توصيلاتُ الأوّل.
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        viewModelScope.launch {
            try {
                val mine = SelectedStore.resolve(stores.stores().stores)
                storeId = mine?.id.orEmpty()
                if (storeId.isEmpty()) return@launch
                // **والأجرةُ تُقال قبل النقطة** — من منطقة المتجر ما لم تُحدَّد.
                val p = point
                fetchQuote(p?.first, p?.second)
                list = api.list(storeId).deliveries
                listError = ""
            } catch (e: Exception) {
                listError = apiError(getApplication(), e)
            }
        }
    }

    fun pickPayer(p: String) {
        payer = p
        touched()
    }

    fun setPoint(lat: Double, lng: Double, label: String) {
        pointFromLink = false
        point = lat to lng
        pointLabel = label.ifEmpty { "%.5f، %.5f".format(lat, lng) }
        touched()
        fetchQuote(lat, lng)
    }

    /** **النقطةُ اختياريّة** — ومن أزالها عادت الأجرةُ لمنطقة المتجر. */
    fun clearPoint() {
        pointFromLink = false
        point = null
        pointLabel = ""
        touched()
        fetchQuote(null, null)
    }

    private fun fetchQuote(lat: Double?, lng: Double?) {
        quote = null
        quoteError = ""
        if (storeId.isEmpty()) return
        viewModelScope.launch {
            try {
                quote = api.quote(storeId, lat, lng)
            } catch (e: Exception) {
                // **وخارجُ التغطية يُقال عند النقطة** — لا بعد ملء كلّ شيء.
                quoteError = apiError(getApplication(), e)
            }
        }
    }

    /** **من كتب من جديد لا يرى «تمّ» السابقة ولا خطأها.** */
    fun touched() {
        sent = false
        error = ""
    }

    fun send() {
        if (busy || storeId.isEmpty()) return
        val p = point
        busy = true
        error = ""
        sent = false
        val key = Attempt.key(Attempt.MERCHANT_DELIVERY)
        viewModelScope.launch {
            try {
                api.create(
                    storeId,
                    NewDelivery(
                        recipientName = name.trim(),
                        recipientPhone = phone.trim(),
                        addressText = address.trim(),
                        lat = p?.first,
                        lng = p?.second,
                        parcelNote = parcel.trim(),
                        driverNote = driverNote.trim(),
                        feePayer = payer,
                    ),
                    idempotencyKey = key,
                )
                Attempt.clear(Attempt.MERCHANT_DELIVERY)
                sent = true
                name = ""
                phone = ""
                address = ""
                parcel = ""
                driverNote = ""
                payer = ""
                link = ""
                linkState = LinkState.Idle
                point = null
                pointLabel = ""
                load()
                Refresh.bump()
            } catch (e: Exception) {
                if (isDecided(e)) Attempt.clear(Attempt.MERCHANT_DELIVERY)
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }

    fun cancel(id: String) {
        if (cancelling != null) return
        cancelling = id
        viewModelScope.launch {
            try {
                api.cancel(id)
                load()
                Refresh.bump()
            } catch (e: Exception) {
                listError = apiError(getApplication(), e)
            } finally {
                cancelling = null
            }
        }
    }
}


@Composable
fun DeliveryScreen(vm: DeliveryViewModel, onPickPoint: () -> Unit) {
    var tracking by rememberSaveable { mutableStateOf<String?>(null) }
    tracking?.let { id ->
        val tvm: DeliveryTrackViewModel = viewModel(key = "track-$id")
        DeliveryTrackScreen(tvm, id) { tracking = null; vm.load() }
        return
    }
    Screen {
        ScreenTitle(stringResource(R.string.md_title), stringResource(R.string.md_hint))

        // **وما يمنع الإرسالَ يُقال أوّلَ الشاشة** — المنصّةُ خارجَ دوامها أو لا
        // سائقَ قريب (قرارُ المالك ٢٠٢٦-١٠-٠١) — **لا بعد ملء النموذج كلِّه.**
        if (vm.quoteError.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.danger.copy(alpha = 0.08f))
                    .border(Rahal.stroke.hair, Rahal.colors.danger.copy(alpha = 0.4f), Rahal.shape.md)
                    .padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_moto),
                    contentDescription = null,
                    tint = Rahal.colors.danger,
                )
                Spacer(Modifier.width(10.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.md_blocked_title),
                        color = Rahal.colors.danger,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(vm.quoteError, color = Rahal.colors.ink, style = MaterialTheme.typography.bodySmall)
                }
                RahalTextButton(onClick = { vm.load() }) {
                    Text(stringResource(com.rahalgo.ui.R.string.act_retry))
                }
            }
        }

        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.name,
            onValueChange = { vm.name = it; vm.touched() },
            label = { Text(stringResource(R.string.md_name)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_user) },
            singleLine = true,
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        // **والرقمُ بحقل الهاتف المشترك** — بأيقونته ومثاله.
        PhoneField(
            value = vm.phone,
            onChange = { vm.phone = it; vm.touched() },
            enabled = !vm.busy,
            label = R.string.md_phone,
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.address,
            onValueChange = { vm.address = it; vm.touched() },
            label = { Text(stringResource(R.string.md_address)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_home) },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )

        // ── موقعُ المستلِم: رابطٌ يُلصق، والخريطةُ احتياط (٢٠٢٦-١٠-٠٨) ──
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.link,
            onValueChange = { vm.onLink(it) },
            label = { Text(stringResource(R.string.md_link)) },
            placeholder = { Text(stringResource(R.string.md_link_hint)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_pin) },
            singleLine = true,
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        when (vm.linkState) {
            LinkState.Checking -> Text(
                stringResource(R.string.md_link_checking),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            LinkState.Found -> Text(
                stringResource(R.string.md_link_found),
                color = Rahal.colors.success,
                style = MaterialTheme.typography.bodySmall,
            )
            LinkState.NotFound -> Text(
                stringResource(R.string.md_link_not_found),
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
            LinkState.Idle -> Text(
                stringResource(R.string.md_link_how),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // **ولا تحديدَ يدويّاً على الخريطة** (طلبُ المالك ٢٠٢٦-١٠-٠٩: «صفحة نظيفة»)
        // — الموقعُ من رابطٍ يشاركه الزبون، والعنوانُ المكتوبُ يكمّله.

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.parcel,
            onValueChange = { vm.parcel = it.take(200); vm.touched() },
            label = { Text(stringResource(R.string.md_parcel)) },
            placeholder = { Text(stringResource(R.string.md_parcel_hint)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_orders) },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.driverNote,
            onValueChange = { vm.driverNote = it.take(200); vm.touched() },
            label = { Text(stringResource(R.string.md_driver_note)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_chat) },
            enabled = !vm.busy,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        // ── من يدفع — بطاقاتٌ واضحةٌ إجباريّة (طلبُ المالك ٢٠٢٦-١٠-٠٩) ──
        Spacer(Modifier.height(14.dp))
        Text(stringResource(R.string.md_who_pays_q), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.height(8.dp))
        listOf(
            Triple("merchant", R.string.md_pay_me, R.string.md_pay_me_sub),
            Triple("merchant_cash", R.string.md_pay_me_cash, R.string.md_pay_me_cash_sub),
            Triple("recipient", R.string.md_pay_recipient, R.string.md_pay_recipient_sub),
        ).forEach { (key, title, sub) ->
            val on = vm.payer == key
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
                    .clip(Rahal.shape.md)
                    .background(if (on) Rahal.colors.brand.copy(alpha = 0.10f) else Rahal.colors.canvas)
                    .border(
                        if (on) 2.dp else Rahal.stroke.hair,
                        if (on) Rahal.colors.brand else Rahal.colors.line,
                        Rahal.shape.md,
                    )
                    .clickable(enabled = !vm.busy) { vm.pickPayer(key) }
                    .padding(horizontal = 14.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                androidx.compose.material3.RadioButton(selected = on, onClick = { vm.pickPayer(key) }, enabled = !vm.busy)
                Spacer(Modifier.width(8.dp))
                Column(Modifier.weight(1f)) {
                    Text(stringResource(title), fontWeight = FontWeight.Bold, color = Rahal.colors.ink)
                    Text(stringResource(sub), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
                }
            }
        }

        // ── الأجرةُ قبل الإرسال ───────────────────────────────────────
        val q = vm.quote
        if (vm.quoteError.isEmpty() && q != null) {
            Spacer(Modifier.height(10.dp))
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.success.copy(alpha = 0.10f))
                    .padding(12.dp),
            ) {
                Text(
                    stringResource(R.string.md_fee, q.fee.toString()),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    when (vm.payer) {
                        "merchant" -> stringResource(R.string.md_fee_from_wallet, q.walletBalance.toString())
                        "merchant_cash" -> stringResource(R.string.md_fee_me_cash)
                        "recipient" -> stringResource(R.string.md_fee_cash)
                        else -> stringResource(R.string.md_fee_choose)
                    },
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            // **ولا يكفيه أن يدفع من محفظته — يُقال قبل الضغط** لا ٤٠٢ بعده.
            if (vm.payer == "merchant" && !q.merchantCanPay) {
                Spacer(Modifier.height(6.dp))
                Note(stringResource(R.string.md_cannot_pay), Rahal.colors.danger)
            }
        }

        // ── ما ينقص · والخطأ · والزرّ ─────────────────────────────────
        val missing = buildList {
            if (vm.name.isBlank()) add(stringResource(R.string.md_name))
            if (vm.phone.isBlank()) add(stringResource(R.string.md_phone))
            if (vm.address.isBlank()) add(stringResource(R.string.md_address))
            if (vm.payer.isEmpty()) add(stringResource(R.string.md_who_pays))
        }
        val blocked = vm.quoteError.isNotEmpty() ||
            (vm.payer == "merchant" && q != null && !q.merchantCanPay)

        if (vm.sent) {
            Spacer(Modifier.height(12.dp))
            Note(stringResource(R.string.md_sent), Rahal.colors.success)
        }
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Note(vm.error, Rahal.colors.danger)
        }
        Spacer(Modifier.height(12.dp))
        RahalButton(
            onClick = { vm.send() },
            enabled = !vm.busy && missing.isEmpty() && !blocked && vm.storeId.isNotEmpty(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(if (vm.busy) R.string.md_sending else R.string.md_send))
        }
        // ── آخرُ التوصيلات — والسجلُّ كلُّه في القائمة الجانبيّة ────────
        Spacer(Modifier.height(20.dp))
        SectionTitle(stringResource(R.string.md_mine))
        if (vm.listError.isNotEmpty()) {
            Note(vm.listError, Rahal.colors.danger)
            RahalTextButton(onClick = { vm.load() }) {
                Text(stringResource(com.rahalgo.ui.R.string.act_retry))
            }
        }
        val rows = vm.list
        when {
            rows == null && vm.listError.isEmpty() ->
                Text(stringResource(R.string.md_loading), color = Rahal.colors.inkMuted)
            rows != null && rows.isEmpty() ->
                Text(stringResource(R.string.md_none), color = Rahal.colors.inkMuted)
            rows != null -> rows.take(5).forEach { d ->
                DeliveryRow(d, cancelling = vm.cancelling != null, onOpen = { tracking = d.id }) { vm.cancel(d.id) }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun FieldIcon(icon: Int) {
    Icon(painterResource(icon), contentDescription = null, tint = Rahal.colors.inkMuted)
}

/**
 * **بطاقةُ توصيلة** — في «آخر التوصيلات» وفي السجلّ، نسخةٌ واحدة.
 *
 * رأسٌ برقمها وشارةِ حالها · **«جاري البحث عن سائق…» تنبض ما دامت تنتظر** ·
 * المستلِمُ ورقمُه وعنوانُه والغرضُ والأجرةُ كلٌّ بأيقونته · **وزرُّ إلغاءٍ أحمرُ
 * يسأل قبل أن يُلغي** (طلبُ المالك ٢٠٢٦-١٠-٠١).
 */
@Composable
fun DeliveryRow(d: Delivery, cancelling: Boolean, onOpen: () -> Unit = {}, onCancel: () -> Unit) {
    Spacer(Modifier.height(10.dp))
    val tone = toneColor(toneOf(d.status))
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, tone.copy(alpha = 0.45f), Rahal.shape.md)
            .clickable { onOpen() }
            .padding(14.dp),
    ) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.md_track_title, d.number.toString()),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall,
                modifier = Modifier.weight(1f),
            )
            StatusPill(d.status)
        }
        Hairline()
        if (d.status == "dispatching") {
            SearchingBanner(compact = true)
            Spacer(Modifier.height(8.dp))
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
            stringResource(
                when (d.feePayer) {
                    "recipient" -> R.string.md_row_cash
                    "merchant_cash" -> R.string.md_row_me_cash
                    else -> R.string.md_row_wallet
                },
                d.fee.toString(),
            ),
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.md_track_open),
            color = Rahal.colors.brand,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodySmall,
        )
        if (d.status in cancellable) {
            Spacer(Modifier.height(10.dp))
            CancelDeliveryButton(busy = cancelling, onConfirm = onCancel)
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **سجلُّ التوصيلات — في القائمة الجانبيّة** (نصُّ المالك ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// «بالقائمة الجانبيّة سجلّ التوصيلات، بحيث من خلاله نستطيع الوصولَ إلى كلّ
// عمليّات التوصيل التي قمنا بها» — **فكلُّها صفحةً صفحة** لا آخرُ خمسين.

class DeliveryHistoryViewModel(app: Application) : AndroidViewModel(app) {
    private val core = AppCore.get()
    private val stores = MerchantApi(core.api)
    private val api = DeliveriesApi(core.api)

    var rows by mutableStateOf<List<Delivery>?>(null)
        private set
    var error by mutableStateOf("")
        private set
    var more by mutableStateOf(false)
        private set
    var busy by mutableStateOf(false)
        private set
    var cancelling by mutableStateOf<String?>(null)
        private set
    private var storeId = ""
    private var page = 1

    init {
        load()
        viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }
    }

    fun load() {
        page = 1
        fetch(reset = true)
    }

    fun next() {
        if (busy || !more) return
        page += 1
        fetch(reset = false)
    }

    private fun fetch(reset: Boolean) {
        busy = true
        viewModelScope.launch {
            try {
                if (reset || storeId.isEmpty()) {
                    storeId = SelectedStore.resolve(stores.stores().stores)?.id.orEmpty()
                }
                if (storeId.isEmpty()) return@launch
                val got = api.list(storeId, page).deliveries
                rows = if (reset) got else rows.orEmpty() + got
                more = got.size >= PAGE
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            } finally {
                busy = false
            }
        }
    }

    fun cancel(id: String) {
        if (cancelling != null) return
        cancelling = id
        viewModelScope.launch {
            try {
                api.cancel(id)
                load()
                Refresh.bump()
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            } finally {
                cancelling = null
            }
        }
    }

    private companion object {
        /** **صفحةُ الخادم** (`ListMerchantDeliveries`) — فصفحةٌ ناقصةٌ آخرُها. */
        const val PAGE = 50
    }
}

@Composable
fun DeliveryHistoryScreen(vm: DeliveryHistoryViewModel) {
    var tracking by rememberSaveable { mutableStateOf<String?>(null) }
    tracking?.let { id ->
        val tvm: DeliveryTrackViewModel = viewModel(key = "track-$id")
        DeliveryTrackScreen(tvm, id) { tracking = null; vm.load() }
        return
    }
    Screen {
        ScreenTitle(stringResource(R.string.menu_deliveries), stringResource(R.string.md_history_hint))
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Note(vm.error, Rahal.colors.danger)
            RahalTextButton(onClick = { vm.load() }) {
                Text(stringResource(com.rahalgo.ui.R.string.act_retry))
            }
        }
        val rows = vm.rows
        when {
            rows == null && vm.error.isEmpty() ->
                Text(stringResource(R.string.md_loading), color = Rahal.colors.inkMuted)
            rows != null && rows.isEmpty() ->
                Text(stringResource(R.string.md_none), color = Rahal.colors.inkMuted)
            rows != null -> rows.forEach { d ->
                DeliveryRow(d, cancelling = vm.cancelling != null, onOpen = { tracking = d.id }) { vm.cancel(d.id) }
            }
        }
        if (vm.more) {
            Spacer(Modifier.height(10.dp))
            RahalOutlineButton(
                onClick = { vm.next() },
                enabled = !vm.busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.md_more)) }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/** **حالُ رابط الموقع** — فارغ · يُقرأ · وُجد · لا موقعَ فيه. */
enum class LinkState { Idle, Checking, Found, NotFound }
