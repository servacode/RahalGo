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
import androidx.compose.foundation.layout.size
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
import androidx.compose.ui.graphics.Color
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
import com.rahalgo.ui.amountInput
import com.rahalgo.ui.amountValue
import com.rahalgo.ui.apiError
import com.rahalgo.ui.isDecided
import com.rahalgo.ui.isUsd
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

    /**
     * **«توصيل» أو «إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — `send` افتراضاً. **والوضعُ يغيّر
     * الأجرة** (الإحضارُ ذهابٌ وإياب) **وأسماءَ الحقول**: المستلِمُ في التوصيل ومكانُ الاستلام
     * في الإحضار.
     */
    // **لا وضعَ مختاراً افتراضاً** (ملاحظةُ المالك ٢٠٢٦-١٠-١٦) — يختار «توصيل» أو «إحضار» صراحةً.
    var mode by mutableStateOf("")
        private set

    /**
     * **«بدك السائق يقبض مبلغ من المستلم؟»** (قرارُ المالك ٢٠٢٦-١٠-١٥) — عرضٌ للسائق وحدَه:
     * **يدفعه لك من جيبه ويأخذه من المستلِم، ولا يدخل حسابَ المنصّة.**
     */
    var collectOn by mutableStateOf(false)
        private set
    var collectAmount by mutableStateOf("")

    /**
     * **ليرة أو دولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — `SYP` افتراضاً. **والدولارُ بمنزلتين**
     * (١٢٫٥)، **والخادمُ يحفظه بالسنتات.**
     */
    var collectCurrency by mutableStateOf("SYP")
        private set

    fun pickCollectCurrency(c: String) {
        if (c == collectCurrency) return
        collectCurrency = c
        // **والمبلغُ يُمسح مع العملة** — «٢٥٠٠٠» ليرة لا تصير ٢٥٠٠٠$ بلمسة.
        collectAmount = ""
        touched()
    }

    fun onCollectAmount(v: String) {
        collectAmount = amountInput(v, collectCurrency)
        touched()
    }

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
        // **ورابطٌ يُكتب بعد وصول موقع المستلِم يحلّ محلّه** — فلا تبقى «✓ وصل» فوق غيره.
        if (ask == AskState.Received && link.contains("http")) ask = AskState.Idle
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

    /**
     * **المنصّةُ خارجَ دوامها أو موقوفة** (قرارُ المالك ٢٠٢٦-١٠-٢٩: «مادام المنصة بالدوام
     * معناها في سائقين، إذا كانت خارج الدوام ما يقدر يرسل») — **يُقال متى تفتح ولا زرَّ
     * «أعد المحاولة»**: الضغطُ لا يفتحها.
     */
    var quoteClosed by mutableStateOf(false)
        private set

    /** **موعدُ الفتح بالمللي** — بعده يُعاد سؤالُ الأجرة مع النبض، لا قبله (لا طرقَ على ٥٠٣). */
    private var closedUntil: Long? = null

    // ══════════════════════════════════════════════════════════════════
    // **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩)
    // ══════════════════════════════════════════════════════════════════
    //
    // يصل المستلِمَ بالواتساب رابطُ صفحة «أرسل موقعي»، **ويُسأل الخادمُ كلَّ أربع ثوانٍ**
    // حتّى تصل النقطة فتملأ النموذجَ كما يملؤه الرابط. **والرابطُ الملصوقُ باقٍ احتياطاً.**
    var ask by mutableStateOf(AskState.Idle)
        private set
    var askReq by mutableStateOf<com.rahalgo.shared.merchant.LocationRequest?>(null)
        private set
    var askError by mutableStateOf("")
        private set
    var askAccuracy by mutableStateOf<Double?>(null)
        private set
    private var askJob: kotlinx.coroutines.Job? = null

    fun askLocation() {
        if (storeId.isEmpty() || ask == AskState.Sending) return
        askError = ""
        if (phone.isBlank()) {
            askError = getApplication<Application>().getString(R.string.md_ask_need_phone)
            return
        }
        askJob?.cancel()
        ask = AskState.Sending
        askJob = viewModelScope.launch {
            val req = try {
                api.requestLocation(storeId, phone.trim(), mode.ifEmpty { "send" })
            } catch (e: Exception) {
                ask = AskState.Idle
                askError = apiError(
                    getApplication(), e,
                    mapOf(
                        "location_request_too_soon" to R.string.md_ask_too_soon,
                        "location_request_limit" to R.string.md_ask_limit,
                    ),
                )
                return@launch
            }
            askReq = req
            askAccuracy = null
            ask = AskState.Waiting
            while (true) {
                kotlinx.coroutines.delay(4000)
                // **وتعثّرُ سؤالٍ واحدٍ لا يوقف الانتظار** — يُسأل بعد أربع ثوانٍ.
                val st = runCatching { api.locationRequest(req.id) }.getOrNull() ?: continue
                val lat = st.lat
                val lng = st.lng
                if (st.status == "received" && lat != null && lng != null) {
                    linkJob?.cancel()
                    link = ""
                    linkState = LinkState.Idle
                    setPoint(lat, lng, "")
                    askAccuracy = st.accuracyM
                    ask = AskState.Received
                    break
                }
                if (st.status == "expired") {
                    ask = AskState.Expired
                    break
                }
            }
        }
    }

    private fun resetAsk() {
        askJob?.cancel()
        ask = AskState.Idle
        askReq = null
        askError = ""
        askAccuracy = null
    }

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
        // **والنبضُ العامُّ يُحدّث القائمةَ لا الأجرة** (٢٠٢٦-١٠-١٣) — كان يعيد سؤالَ
        // `delivery-quote` مع كلّ نبضةٍ والمنصّةُ مسكّرة (٥٠٣ مقصود)، **فيطرق بابَها بلا توقّف.**
        // **والأجرةُ تُسأل عند الفتح وتغيّر النقطة و«أعد المحاولة» وحدَها.**
        viewModelScope.launch {
            Refresh.tick.drop(1).collect {
                // **ومنصّةٌ أُغلقت ثمّ حان موعدُ فتحها** — يُعاد سؤالُ الأجرة فيزول المنع.
                val until = closedUntil
                load(withQuote = quoteClosed && until != null && System.currentTimeMillis() >= until)
            }
        }
    }

    fun load(withQuote: Boolean = true) {
        viewModelScope.launch {
            try {
                val mine = SelectedStore.resolve(stores.stores().stores)
                val changed = mine?.id.orEmpty() != storeId
                storeId = mine?.id.orEmpty()
                if (storeId.isEmpty()) return@launch
                // **والأجرةُ تُقال قبل النقطة** — من منطقة المتجر ما لم تُحدَّد.
                // **وفرعٌ تبدّل يُسأل له** ولو كانت نبضة.
                if (withQuote || changed) {
                    val p = point
                    fetchQuote(p?.first, p?.second)
                }
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

    fun pickMode(m: String) {
        if (m == mode) return
        mode = m
        touched()
        val p = point
        fetchQuote(p?.first, p?.second)
    }

    fun toggleCollect(on: Boolean) {
        collectOn = on
        if (!on) {
            collectAmount = ""
            collectCurrency = "SYP"
        }
        touched()
    }

    /** **المبلغُ رقمٌ موجبٌ أو لا شيء** — والخادمُ يردّ الصفر. **وبالوحدة الكبرى** (١٢٫٥ لا ١٢٥٠). */
    val collectValue: Double?
        get() = if (collectOn) amountValue(amountInput(collectAmount, collectCurrency)) else null

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
        quoteClosed = false
        closedUntil = null
        if (storeId.isEmpty()) return
        viewModelScope.launch {
            try {
                val q = api.quote(storeId, lat, lng, mode.ifEmpty { "send" })
                quote = q
                // **والمحفظةُ لا تكفي ⇒ لا يبقى «من محفظتي» مختاراً** — لا دين (٢٠٢٦-١٠-١٥).
                if (payer == "merchant" && !q.merchantCanPay) payer = ""
            } catch (e: Exception) {
                val closed = (e as? com.rahalgo.shared.net.ApiClient.ApiException)?.body
                quoteClosed = closed?.code == "platform_closed_now" || closed?.code == "temporarily_unavailable"
                closedUntil = closed?.details?.get("next_available_at")?.let {
                    runCatching { java.time.OffsetDateTime.parse(it.trim()).toInstant().toEpochMilli() }.getOrNull()
                }
                quoteError = if (closed != null && closed.code == "platform_closed_now") {
                    // **«تفتح المنصة الساعة …»** — وبلا موعدٍ جملةٌ ثابتة.
                    val back = com.rahalgo.ui.backAtText(closed.details["next_available_at"])
                    if (back != null) getApplication<Application>().getString(R.string.md_closed_at, back)
                    else getApplication<Application>().getString(R.string.md_closed_plain)
                } else {
                    // **وخارجُ التغطية يُقال عند النقطة** — لا بعد ملء كلّ شيء.
                    apiError(getApplication(), e)
                }
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **«إحضار»: موافقتُك على مبلغ السائق** (قرارُ المالك ٢٠٢٦-١٠-١٦) — `FetchAmountDialog`
    // ══════════════════════════════════════════════════════════════════
    var amountBusy by mutableStateOf(false)
        private set
    var amountError by mutableStateOf("")
        private set

    /** **يوافق أو يرفض بنسخة المبلغ الذي رآه** — ونسخةٌ تبدّلت تُردّ فتُقرأ القائمةُ من جديد. */
    fun decideAmount(d: Delivery, approve: Boolean) {
        if (amountBusy) return
        amountBusy = true
        amountError = ""
        viewModelScope.launch {
            try {
                api.decideFetchAmount(d.id, approve, d.fetchAmountVersion)
                Refresh.bump()
            } catch (e: Exception) {
                amountError = apiError(getApplication(), e)
            } finally {
                amountBusy = false
                load(withQuote = false)
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
                        deliveryMode = mode,
                        collectAmount = collectValue,
                        amountCurrency = collectCurrency,
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
                mode = ""
                collectOn = false
                collectAmount = ""
                collectCurrency = "SYP"
                link = ""
                linkState = LinkState.Idle
                point = null
                pointLabel = ""
                resetAsk()
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
fun DeliveryScreen(vm: DeliveryViewModel) {
    var tracking by rememberSaveable { mutableStateOf<String?>(null) }
    // **ضغطَ «اطلب سائقاً» والنموذجُ ناقص** — فيُعلَّم الناقصُ بالأحمر (تدقيقُ ٢٠٢٦-١٠-٢٧).
    var tried by rememberSaveable { mutableStateOf(false) }
    tracking?.let { id ->
        val tvm: DeliveryTrackViewModel = viewModel(key = "track-$id")
        DeliveryTrackScreen(tvm, id) { tracking = null; vm.load() }
        return
    }
    Screen {
        ScreenTitle(stringResource(R.string.md_title), stringResource(R.string.md_hint))

        // ══════════════════════════════════════════════════════════════
        // **توصيلةٌ جارية — شريطٌ فوق النموذج بزرّ «تابعها»** (التصميمُ المعتمد ٢٠٢٦-١٠-٢٨)
        // ══════════════════════════════════════════════════════════════
        vm.list?.firstOrNull { toneOf(it.status) == Tone.Waiting || toneOf(it.status) == Tone.Moving }?.let { live ->
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.brand.copy(alpha = 0.10f))
                    .clickable { tracking = live.id }
                    .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_moto),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                )
                Spacer(Modifier.width(Rahal.space.sm))
                Text(
                    stringResource(R.string.md_active, live.number, deliveryStatusText(live.status)),
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.bodyMedium,
                    color = Rahal.colors.ink,
                )
                RahalTextButton(onClick = { tracking = live.id }) {
                    Text(stringResource(R.string.md_follow), fontWeight = FontWeight.Bold)
                }
            }
            Spacer(Modifier.height(Rahal.space.xs))
        }

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
                        stringResource(if (vm.quoteClosed) R.string.md_closed_title else R.string.md_blocked_title),
                        color = Rahal.colors.danger,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(vm.quoteError, color = Rahal.colors.ink, style = MaterialTheme.typography.bodySmall)
                }
                // **وخارجَ الدوام لا «أعد المحاولة»** (٢٠٢٦-١٠-٢٩) — الضغطُ لا يفتح المنصّة،
                // **والنبضُ العامُّ يُعيد السؤالَ بعد موعد الفتح** (`closedUntil`).
                if (!vm.quoteClosed) {
                    RahalTextButton(onClick = { vm.load() }) {
                        Text(stringResource(com.rahalgo.ui.R.string.act_retry))
                    }
                }
            }
        }

        // ── «توصيل» أو «إحضار» — زرّان كبيران، ولا مختارَ افتراضاً (ملاحظةُ المالك ٢٠٢٦-١٠-١٦):
        // «لازم يكونن زرين واضحين لازم يختار وحدة منهن». **والنموذجُ لا يظهر قبل الاختيار.**
        val fetch = vm.mode == "fetch"
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            listOf(
                Triple("send", R.string.md_mode_send, R.string.md_mode_send_sub),
                Triple("fetch", R.string.md_mode_fetch, R.string.md_mode_fetch_sub),
            ).forEach { (key, label, sub) ->
                val on = vm.mode == key
                Column(
                    Modifier
                        .weight(1f)
                        .clip(Rahal.shape.md)
                        .background(if (on) Rahal.colors.brand.copy(alpha = 0.10f) else Rahal.colors.canvas)
                        .border(
                            if (on) 2.dp else Rahal.stroke.hair,
                            if (on) Rahal.colors.brand else Rahal.colors.line,
                            Rahal.shape.md,
                        )
                        .clickable(enabled = !vm.busy) { vm.pickMode(key) }
                        .padding(horizontal = 12.dp, vertical = 16.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(
                        stringResource(label),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleLarge,
                        color = if (on) Rahal.colors.brand else Rahal.colors.ink,
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(
                        stringResource(sub),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                    )
                }
            }
        }
        if (vm.mode.isEmpty()) {
            Spacer(Modifier.height(16.dp))
            Text(
                stringResource(R.string.md_mode_pick),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.fillMaxWidth(),
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            )
            // **و«توصيلاتي» تبقى ظاهرةً قبل الاختيار** — لا تختفي مع النموذج.
            RecentDeliveries(vm) { tracking = it }
            Spacer(Modifier.height(24.dp))
            return@Screen
        }

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.name,
            onValueChange = { vm.name = it; vm.touched() },
            label = { Text(stringResource(if (fetch) R.string.md_name_fetch else R.string.md_name)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_user) },
            isError = tried && vm.name.isBlank(),
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
            label = if (fetch) R.string.md_phone_fetch else R.string.md_phone,
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.address,
            onValueChange = { vm.address = it; vm.touched() },
            label = { Text(stringResource(if (fetch) R.string.md_address_fetch else R.string.md_address)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_home) },
            isError = tried && vm.address.isBlank(),
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )

        // ── موقعُ المستلِم: «اطلب الموقع من المستلم» أوّلاً (٢٠٢٦-١٠-٢٩)، والرابطُ احتياط ──
        Spacer(Modifier.height(10.dp))
        AskLocationCard(vm, fetch)
        Spacer(Modifier.height(8.dp))
        val clipboard = androidx.compose.ui.platform.LocalClipboardManager.current
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                shape = Rahal.shape.md,
                value = vm.link,
                onValueChange = { vm.onLink(it) },
                label = { Text(stringResource(if (fetch) R.string.md_link_fetch else R.string.md_link)) },
                placeholder = { Text(stringResource(if (fetch) R.string.md_link_hint_fetch else R.string.md_link_hint)) },
                leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_pin) },
                singleLine = true,
                enabled = !vm.busy,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(Rahal.space.sm))
            // **«لصق» بلمسة** — يقرأ الحافظةَ ويمرّرها لقارئ الرابط نفسِه.
            RahalOutlineButton(
                onClick = { clipboard.getText()?.text?.trim()?.takeIf { it.isNotEmpty() }?.let { vm.onLink(it) } },
                enabled = !vm.busy,
            ) { Text(stringResource(R.string.md_paste), fontWeight = FontWeight.Bold) }
        }
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
                stringResource(if (fetch) R.string.md_link_not_found_fetch else R.string.md_link_not_found),
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
            LinkState.Idle -> Text(
                // **وفي الإحضار الموقعُ موقعُ مكان الاستلام لا الزبون** (٢٠٢٦-١٠-١٦).
                stringResource(if (fetch) R.string.md_link_how_fetch else R.string.md_link_how),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // **ولا تحديدَ يدويّاً على الخريطة** (طلبُ المالك ٢٠٢٦-١٠-٠٩: «صفحة نظيفة»)
        // — الموقعُ من رابطٍ يشاركه الزبون، والعنوانُ المكتوبُ يكمّله.

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.parcel,
            onValueChange = { vm.parcel = it.take(200); vm.touched() },
            // **«يجيب» لا «يوصّل» في الإحضار** — السائقُ يأتي بالغرض إلى المتجر.
            label = { Text(stringResource(if (fetch) R.string.md_parcel_fetch else R.string.md_parcel)) },
            placeholder = { Text(stringResource(if (fetch) R.string.md_parcel_hint_fetch else R.string.md_parcel_hint)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_orders) },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.driverNote,
            onValueChange = { vm.driverNote = it.take(200); vm.touched() },
            label = { Text(stringResource(R.string.md_driver_note)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_chat) },
            enabled = !vm.busy,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        // ── «بدك السائق يقبض مبلغ من المستلم؟» — عرضٌ لا حساب (٢٠٢٦-١٠-١٥) ──
        // **وفي «إحضار» يدفعه بمكان الاستلام ويأخذه منك** (٢٠٢٦-١٠-١٦) — `DeliveryTexts`.
        Spacer(Modifier.height(10.dp))
        Row(
            Modifier
                .fillMaxWidth()
                .clickable(enabled = !vm.busy) { vm.toggleCollect(!vm.collectOn) },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            androidx.compose.material3.Checkbox(
                checked = vm.collectOn,
                onCheckedChange = { vm.toggleCollect(it) },
                enabled = !vm.busy,
            )
            Column(Modifier.weight(1f)) {
                // **وفي الإحضار المستلِمُ هو المتجرُ نفسُه** — والسائقُ يدفع هناك لا يقبض.
                Text(
                    stringResource(DeliveryTexts.collectQuestion(fetch)),
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.ink,
                )
                // **بلا سطر شرح** (ملاحظةُ المالك ٢٠٢٦-١٠-١٦: «العبارة شيلها ماتلزم اصلا»).
            }
        }
        if (vm.collectOn) {
            // **ليرة أو دولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — **ومنذ ٢٠٢٦-١٠-٢٩ دائرتان بجانب
            // خانة المبلغ** («ل.س» و«$»)، والمختارةُ مملوءةٌ بلون العلامة.
            val usd = isUsd(vm.collectCurrency)
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                shape = Rahal.shape.md,
                value = vm.collectAmount,
                onValueChange = { v -> vm.onCollectAmount(v) },
                isError = tried && vm.collectValue == null,
                label = {
                    Text(
                        stringResource(
                            DeliveryTexts.collectAmountLabel(fetch),
                            stringResource(DeliveryTexts.currencyUnit(vm.collectCurrency)),
                        ),
                    )
                },
                leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_wallet) },
                singleLine = true,
                enabled = !vm.busy,
                keyboardOptions = KeyboardOptions(
                    keyboardType = if (usd) androidx.compose.ui.text.input.KeyboardType.Decimal
                    else androidx.compose.ui.text.input.KeyboardType.Number,
                ),
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(Rahal.space.sm))
            CurrencyDots(selected = vm.collectCurrency, enabled = !vm.busy) { vm.pickCollectCurrency(it) }
            }
        }

        // ── من يدفع — بطاقاتٌ واضحةٌ إجباريّة (طلبُ المالك ٢٠٢٦-١٠-٠٩) ──
        Spacer(Modifier.height(14.dp))
        Text(
            stringResource(R.string.md_who_pays_q),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
            color = if (tried && vm.payer.isEmpty()) Rahal.colors.danger else Color.Unspecified,
        )
        Spacer(Modifier.height(8.dp))
        listOf(
            Triple("merchant", R.string.md_pay_me, R.string.md_pay_me_sub),
            Triple(
                "merchant_cash",
                R.string.md_pay_me_cash,
                if (fetch) R.string.md_pay_me_cash_sub_fetch else R.string.md_pay_me_cash_sub,
            ),
            Triple(
                "recipient",
                if (fetch) R.string.md_pay_recipient_fetch else R.string.md_pay_recipient,
                if (fetch) R.string.md_pay_recipient_sub_fetch else R.string.md_pay_recipient_sub,
            ),
        ).forEach { (key, title, sub) ->
            val on = vm.payer == key
            // **ولا دين** (قرارُ المالك ٢٠٢٦-١٠-١٥): المتاحُ لا يكفي ⇒ «من محفظتي» معطَّل.
            val walletShort = key == "merchant" && vm.quote?.merchantCanPay == false
            val usable = !vm.busy && !walletShort
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
                    .clickable(enabled = usable) { vm.pickPayer(key) }
                    .padding(horizontal = 14.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                androidx.compose.material3.RadioButton(selected = on, onClick = { vm.pickPayer(key) }, enabled = usable)
                Spacer(Modifier.width(8.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(title),
                        fontWeight = FontWeight.Bold,
                        color = if (walletShort) Rahal.colors.inkMuted else Rahal.colors.ink,
                    )
                    Text(
                        stringResource(if (walletShort) R.string.md_wallet_short else sub),
                        color = if (walletShort) Rahal.colors.danger else Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
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
                    stringResource(R.string.md_fee, com.rahalgo.ui.grouped(q.fee)),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    when (vm.payer) {
                        "merchant" -> stringResource(R.string.md_fee_from_wallet, com.rahalgo.ui.grouped(q.availableBalance))
                        "merchant_cash" -> stringResource(if (fetch) R.string.md_fee_me_cash_fetch else R.string.md_fee_me_cash)
                        "recipient" -> stringResource(if (fetch) R.string.md_fee_cash_fetch else R.string.md_fee_cash)
                        else -> stringResource(R.string.md_fee_choose)
                    },
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                )
                // **ومن أين الأجرة** (٢٠٢٦-١٠-١٥): بلا نقطةٍ الأساسُ وحدَه، ومعها المسافة.
                if (q.feeBasis == "no_point") {
                    Text(
                        stringResource(if (fetch) R.string.md_fee_no_point_fetch else R.string.md_fee_no_point),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                } else if (q.feeBasis == "distance" && q.distanceM > 0) {
                    Text(
                        stringResource(R.string.md_fee_distance, "%.1f".format(q.distanceM / 1000.0)),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
            // **ولا يكفيه أن يدفع من محفظته — يُقال قبل الضغط** لا ٤٠٢ بعده.
            if (vm.payer == "merchant" && !q.merchantCanPay) {
                Spacer(Modifier.height(6.dp))
                Note(stringResource(R.string.md_cannot_pay), Rahal.colors.danger)
            }
        }

        // ── ما ينقص · والخطأ · والزرّ ─────────────────────────────────
        val missing = buildList {
            // **وما ينقص بأسماء حقوله كما تُرى** — في الإحضار «الشخص بمكان الاستلام» لا «المستلم».
            if (vm.name.isBlank()) add(stringResource(if (fetch) R.string.md_name_fetch else R.string.md_name))
            if (vm.phone.isBlank()) add(stringResource(if (fetch) R.string.md_phone_fetch else R.string.md_phone))
            if (vm.address.isBlank()) add(stringResource(if (fetch) R.string.md_address_fetch else R.string.md_address))
            if (vm.payer.isEmpty()) add(stringResource(R.string.md_who_pays))
            if (vm.collectOn && vm.collectValue == null) add(stringResource(DeliveryTexts.collectMissing(fetch)))
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
        // ══════════════════════════════════════════════════════════════
        // **وما ينقص يُقال فوق الزرّ** (تدقيقُ ٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **كان الزرُّ يصير رماديّاً بلا كلمة** — والقائمةُ محسوبةٌ ولا تُعرض، فيعبّي
        // التاجرُ الاستمارةَ الطويلة ويجد الزرَّ واقفاً ولا يعرف ليش. **فالناقصُ سطرٌ
        // فوقه، والضغطُ والنموذجُ ناقصٌ يعلّم خاناته بالأحمر** — ولا يُرسل شيئاً.
        if (missing.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Text(
                stringResource(R.string.md_missing, missing.joinToString("، ")),
                color = if (tried) Rahal.colors.danger else Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
                fontWeight = if (tried) FontWeight.Bold else FontWeight.Normal,
            )
        }
        Spacer(Modifier.height(12.dp))
        RahalButton(
            onClick = {
                if (missing.isNotEmpty()) {
                    tried = true
                } else {
                    tried = false
                    vm.send()
                }
            },
            enabled = !vm.busy && !blocked && vm.storeId.isNotEmpty(),
            // **زرٌّ كبيرٌ بعرض الشاشة** (التصميمُ المعتمد ٢٠٢٦-١٠-٢٨).
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
        ) {
            Text(
                stringResource(if (vm.busy) R.string.md_sending else R.string.md_send),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
        }
        RecentDeliveries(vm) { tracking = it }
        Spacer(Modifier.height(24.dp))
    }
}

/** **آخرُ التوصيلات** — تحت النموذج، **وقبل اختيار النوع أيضاً.** */
@Composable
private fun RecentDeliveries(vm: DeliveryViewModel, onOpen: (String) -> Unit) {
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
        // **والفارغُ بالقطعة المشتركة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كبقيّة الشاشات، ويقول ماذا يفعل.
        rows != null && rows.isEmpty() ->
            com.rahalgo.ui.Empty(stringResource(R.string.md_none), hint = stringResource(R.string.md_none_hint))
        rows != null -> rows.take(5).forEach { d ->
            DeliveryRow(d, cancelling = vm.cancelling != null, onOpen = { onOpen(d.id) }) { vm.cancel(d.id) }
        }
    }
}

/**
 * **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — زرٌّ وحالُه تحته:
 * يُرسَل · ينتظر (ومشاركةُ الرابط إن لم تصل الرسالة) · **✓ وصل بدقّته** · انتهى.
 */
@Composable
private fun AskLocationCard(vm: DeliveryViewModel, fetch: Boolean) {
    val uri = androidx.compose.ui.platform.LocalUriHandler.current
    val waiting = vm.ask == AskState.Sending || vm.ask == AskState.Waiting
    RahalOutlineButton(
        onClick = { vm.askLocation() },
        enabled = !vm.busy && !waiting,
        modifier = Modifier
            .fillMaxWidth()
            .height(52.dp),
    ) {
        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_pin), contentDescription = null, tint = Rahal.colors.brand)
        Spacer(Modifier.width(Rahal.space.sm))
        Text(
            stringResource(
                when {
                    vm.ask == AskState.Sending -> R.string.md_ask_sending
                    fetch -> R.string.md_ask_loc_fetch
                    else -> R.string.md_ask_loc
                },
            ),
            fontWeight = FontWeight.Bold,
        )
    }
    val req = vm.askReq
    when {
        vm.ask == AskState.Waiting && req != null -> {
            Spacer(Modifier.height(6.dp))
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.canvas)
                    .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                    .padding(12.dp),
            ) {
                Text(
                    stringResource(if (req.whatsappSent) R.string.md_ask_waiting else R.string.md_ask_not_sent),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                // **ولم تصل الرسالة** (البوتُ غيرُ مقترن) — يشارك الرابطَ بنفسه بلمسة.
                if (!req.whatsappSent && req.waUrl.isNotEmpty()) {
                    RahalTextButton(onClick = { runCatching { uri.openUri(req.waUrl) } }) {
                        Text(stringResource(R.string.md_ask_share), fontWeight = FontWeight.Bold)
                    }
                }
            }
        }
        vm.ask == AskState.Received -> {
            Spacer(Modifier.height(6.dp))
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.success.copy(alpha = 0.10f))
                    .padding(12.dp),
            ) {
                Text(
                    stringResource(if (fetch) R.string.md_ask_received_fetch else R.string.md_ask_received),
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                )
                vm.askAccuracy?.let {
                    Text(
                        stringResource(R.string.md_ask_accuracy, com.rahalgo.ui.grouped(it.toLong())),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
        vm.ask == AskState.Expired -> Text(
            stringResource(R.string.md_ask_expired),
            color = Rahal.colors.danger,
            style = MaterialTheme.typography.bodySmall,
        )
    }
    if (vm.askError.isNotEmpty()) {
        Text(vm.askError, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
    }
}

/** **دائرتا العملة بجانب خانة المبلغ** (٢٠٢٦-١٠-٢٩) — المختارةُ مملوءةٌ بلون العلامة. */
@Composable
private fun CurrencyDots(selected: String, enabled: Boolean, onPick: (String) -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs)) {
        listOf("SYP" to R.string.md_cur_syp, "USD" to R.string.md_cur_usd).forEach { (code, label) ->
            val on = isUsd(selected) == isUsd(code)
            androidx.compose.foundation.layout.Box(
                Modifier
                    .size(48.dp)
                    .clip(androidx.compose.foundation.shape.CircleShape)
                    .background(if (on) Rahal.colors.brand else Rahal.colors.canvas)
                    .border(
                        Rahal.stroke.hair,
                        if (on) Rahal.colors.brand else Rahal.colors.line,
                        androidx.compose.foundation.shape.CircleShape,
                    )
                    .clickable(enabled = enabled) { onPick(code) },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    stringResource(label),
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
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
            Column(Modifier.weight(1f)) {
                Text(
                    stringResource(R.string.md_track_title, d.number.toString()),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall,
                )
                // **ويومُها وساعتُها** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت بلا تاريخ.
                if (d.createdAt.isNotBlank()) {
                    Text(
                        com.rahalgo.merchant.Dates.stamp(androidx.compose.ui.platform.LocalContext.current, d.createdAt),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
            StatusPill(d.status)
        }
        Hairline()
        if (d.status == "dispatching") {
            SearchingBanner(compact = true)
            Spacer(Modifier.height(8.dp))
        }
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
        d.collectAmount?.takeIf { it > 0 }?.let {
            InfoLine(
                com.rahalgo.ui.R.drawable.ic_wallet,
                stringResource(DeliveryTexts.collectRow(d.deliveryMode == "fetch"), DeliveryTexts.amount(d, it)),
            )
        }
        // **و«اضغط لمتابعتها» لِما يجري وحدَه** — المنتهيةُ لا تُتابَع (تدقيقُ ٢٠٢٦-١٠-٢٧).
        if (toneOf(d.status) == Tone.Waiting || toneOf(d.status) == Tone.Moving) {
            Spacer(Modifier.height(6.dp))
            Text(
                stringResource(R.string.md_track_open),
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodySmall,
            )
        }
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
            // **والتحميلُ والفارغُ بالقطع المشتركة** (تدقيقُ ٢٠٢٦-١٠-٢٧).
            rows == null && vm.error.isEmpty() ->
                com.rahalgo.ui.LoadState(loading = true, error = "")
            rows != null && rows.isEmpty() ->
                com.rahalgo.ui.Empty(stringResource(R.string.md_none), hint = stringResource(R.string.md_none_hint))
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

/** **حالُ «اطلب الموقع من المستلم»** — لا طلب · يُرسَل · ينتظر · وصل · انتهى. */
enum class AskState { Idle, Sending, Waiting, Received, Expired }
