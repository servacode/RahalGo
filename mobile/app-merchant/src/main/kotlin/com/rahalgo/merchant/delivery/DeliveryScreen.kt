package com.rahalgo.merchant.delivery

import android.app.Application
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.merchant.SelectedStore
import com.rahalgo.shared.merchant.DeliveriesApi
import com.rahalgo.shared.merchant.Delivery
import com.rahalgo.shared.merchant.DeliveryQuote
import com.rahalgo.shared.merchant.DeliveryRecipient
import com.rahalgo.shared.merchant.RecipientSuggest
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.NewDelivery
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Attempt
import com.rahalgo.ui.BottomActionBar
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.SectionHeader
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.Note
import com.rahalgo.ui.PhoneField
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
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
            // **والإلغاءُ لا يُقرأ «رابطاً بلا موقع»** — نداءٌ أُلغي لأنّ غيرَه كُتب لا يكتب
            // «ما قرينا الرابط» فوق نقطةٍ صحيحة (مراجعةُ ٢٠٢٦-١١-٠٦).
            val r = try {
                api.resolveLocation(storeId, link)
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                null
            }
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

    // ══════════════════════════════════════════════════════════════════
    // **مستلموه السابقون يُقترحون وقتَ الكتابة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // «بمجرد ما يكتب اسم الزبون يتعبى باقي المعلومات بشكل تلقائي عنوان موقع رقم وكلشي».
    // **حرفان في الاسم أو الرقم ⇒ قائمةٌ تحت الحقل** (`RecipientSuggest`)، واللمسةُ تملأ
    // الاسمَ والرقمَ والعنوانَ والنقطةَ إن عُرفت — **ويبقى كلُّ شيءٍ قابلاً للتعديل.**
    // **وتعثّرُ الاقتراح صامت**: النموذجُ يعمل كما كان بلاه.

    /** **الحقلُ الذي تظهر تحته القائمة** — `name` · `phone` · وفارغٌ لا قائمة. */
    var suggestFor by mutableStateOf("")
        private set
    var suggestions by mutableStateOf<List<DeliveryRecipient>>(emptyList())
        private set

    /** **النقطةُ من آخر توصيلةٍ لمستلِمٍ اختير** — تُقال تحت العنوان وتُزال بلمسة. */
    var pointFromSaved by mutableStateOf(false)
        private set
    private var suggestJob: kotlinx.coroutines.Job? = null

    /** **الحقلُ الذي ينتظر اقتراحَه** — فيُلغى إن خرج منه قبل أن يصل. */
    private var suggestPending = ""

    /** **المستلِمُ المختار كما مُلئ** — وبه يُعرف أنّ الرقمَ أو العنوانَ تبدّل بعده. */
    private var picked: RecipientSuggest.Fill? = null

    /**
     * **نقطةُ مستلِمٍ سابقٍ لا تبقى لغيره** (مراجعةُ ٢٠٢٦-١١-٠٦) — من بدّل الرقمَ أو كتب في
     * العنوان بعد الاختيار فهو شخصٌ آخر أو مكانٌ آخر، **ونقطةٌ قديمةٌ تُرسل السائقَ إلى غير
     * مكانه.** فتسقط ويعود بلا نقطةٍ كما اليوم، **وتُعاد الأجرة** (`clearPoint`).
     */
    private fun dropStaleSavedPoint() {
        val p = picked ?: return
        if (pointFromSaved && !RecipientSuggest.keepsSavedPoint(p, phone, address)) clearPoint()
    }

    /** **يكتب في العنوان.** */
    fun onAddress(v: String) {
        address = v
        touched()
        dropStaleSavedPoint()
    }

    /** **يكتب في الاسم** — والقائمةُ تتبع ما كُتب. */
    fun onName(v: String) {
        name = v
        touched()
        suggest("name", v)
    }

    /**
     * **يكتب في الرقم** — والرقمُ أدقّ (قرارُ المالك). **والأرقامُ الهنديّةُ لاتينيّة**
     * (`WesternDigits.text`) — كان الخادمُ يردّ «٠٩٣٣…» رقماً غيرَ صالح، ومثلُه الموقع.
     */
    fun onPhone(v: String) {
        phone = com.rahalgo.ui.WesternDigits.text(v)
        touched()
        dropStaleSavedPoint()
        suggest("phone", phone)
    }

    /** **خرج من الحقل** — فتُطوى قائمتُه (إلّا إن انتقل إلى حقلٍ آخرَ له قائمته). */
    fun leaveField(field: String) {
        if (suggestFor == field || suggestPending == field) closeSuggestions()
    }

    private fun suggest(field: String, text: String) {
        suggestJob?.cancel()
        if (storeId.isEmpty() || !RecipientSuggest.shouldSearch(text)) {
            closeSuggestions()
            return
        }
        suggestPending = field
        suggestJob = viewModelScope.launch {
            kotlinx.coroutines.delay(RecipientSuggest.DEBOUNCE_MS)
            // **وتعثّرُه لا يُقال** — قائمةٌ لا تظهر خيرٌ من خطأٍ فوق نموذجٍ يعمل. **والإلغاءُ
            // ليس تعثّراً**: نداءٌ أُلغي لأنّ حرفاً جديداً كُتب لا يكتب قائمةً فارغة (وميض).
            val got = try {
                api.recipients(storeId, text).recipients
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                emptyList()
            }
            suggestPending = ""
            suggestFor = if (got.isEmpty()) "" else field
            suggestions = got
        }
    }

    fun closeSuggestions() {
        suggestJob?.cancel()
        suggestPending = ""
        suggestFor = ""
        suggestions = emptyList()
    }

    /** **اللمسةُ تملأ كلَّ شيء** — والنقطةُ إن عُرفت، وإلّا بلا نقطةٍ كما اليوم. */
    fun pickRecipient(r: DeliveryRecipient) {
        val f = RecipientSuggest.fill(r)
        closeSuggestions()
        name = f.name
        phone = f.phone
        address = f.address
        picked = f
        // **وطلبُ موقعٍ أُرسل لرقمٍ آخر يُنسى** (مراجعةُ ٢٠٢٦-١١-٠٦) — فلا يملأ موقعٌ يصل متأخّراً
        // مستلِماً غيرَ الذي طُلب منه. **وما وصل منه لغيره يسقط** — هو مكانُ شخصٍ آخر.
        if (ask != AskState.Idle && !RecipientSuggest.samePhone(askPhone, f.phone)) {
            if (ask == AskState.Received && !pointFromSaved) clearPoint()
            resetAsk()
        }
        val p = f.point
        if (p != null) {
            // **ونقطتُه تحلّ محلّ رابطٍ أو طلبِ موقعٍ سابق** — لا تبقى «✓ وصل» فوق غيرها.
            linkJob?.cancel()
            link = ""
            linkState = LinkState.Idle
            resetAsk()
            setPoint(p.first, p.second, "")
            pointFromSaved = true
        } else if (pointFromSaved) {
            // **ونقطةُ مستلِمٍ سابقٍ لا تبقى لمستلِمٍ بلا نقطة.**
            clearPoint()
        }
        touched()
    }

    /** **«شيلو من القائمة»** — يختفي الآن، ويرجع وحدَه إن بعث له من جديد. */
    fun hideRecipient(r: DeliveryRecipient) {
        suggestions = suggestions.filterNot { it.phone == r.phone }
        if (suggestions.isEmpty()) suggestFor = ""
        val id = storeId
        viewModelScope.launch { runCatching { api.hideRecipient(id, r.phone) } }
    }

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
                askPhone = phone.trim()
                api.requestLocation(storeId, askPhone, mode.ifEmpty { "send" })
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

    /** **الرقمُ الذي أُرسل له طلبُ الموقع.** */
    private var askPhone = ""

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
        pointFromSaved = false
        point = lat to lng
        pointLabel = label.ifEmpty { String.format(java.util.Locale.US, "%.5f، %.5f", lat, lng) }
        touched()
        fetchQuote(lat, lng)
    }

    /** **النقطةُ اختياريّة** — ومن أزالها عادت الأجرةُ لمنطقة المتجر. */
    fun clearPoint() {
        pointFromLink = false
        pointFromSaved = false
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
                pointFromSaved = false
                picked = null
                closeSuggestions()
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
    // **ضغطَ «اطلب كابتن» والنموذجُ ناقص** — فيُعلَّم الناقصُ بالأحمر (تدقيقُ ٢٠٢٦-١٠-٢٧).
    // **والزرُّ معطَّلٌ حتّى يكتمل، فلمسةُ سطرِ «ناقص» هي التي تعلّم** (التصميمُ المعتمد ٢٠٢٦-١١-٢٧).
    var tried by rememberSaveable { mutableStateOf(false) }
    tracking?.let { id ->
        val tvm: DeliveryTrackViewModel = viewModel(key = "track-$id")
        DeliveryTrackScreen(tvm, id) { tracking = null; vm.load() }
        return
    }
    val fetch = vm.mode == "fetch"
    val q = vm.quote
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
    Column(Modifier.fillMaxSize()) {
    Screen(Modifier.weight(1f)) {
        ScreenTitle(stringResource(R.string.md_title))

        // ══════════════════════════════════════════════════════════════
        // **توصيلةٌ جارية — بطاقةٌ بارزةٌ فوق النموذج بزرّ «تابعها»** (التصميمُ المعتمد ٢٠٢٦-١١-٢٧)
        // ══════════════════════════════════════════════════════════════
        vm.list?.firstOrNull { toneOf(it.status) == Tone.Waiting || toneOf(it.status) == Tone.Moving }?.let { live ->
            LiveBanner(live) { tracking = live.id }
            Spacer(Modifier.height(Rahal.space.sm))
        }

        // **وما يمنع الإرسالَ يُقال أوّلَ الشاشة** — المنصّةُ خارجَ دوامها أو لا
        // كابتنَ قريب (قرارُ المالك ٢٠٢٦-١٠-٠١) — **لا بعد ملء النموذج كلِّه.**
        if (vm.quoteError.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.sm))
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Danger) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_moto),
                        contentDescription = null,
                        tint = Rahal.colors.danger,
                    )
                    Spacer(Modifier.width(Rahal.space.sm))
                    Column(Modifier.weight(1f)) {
                        Text(
                            stringResource(if (vm.quoteClosed) R.string.md_closed_title else R.string.md_blocked_title),
                            color = Rahal.colors.danger,
                            style = Rahal.type.cardTitle,
                        )
                        Text(vm.quoteError, color = Rahal.colors.ink, style = Rahal.type.caption)
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
        }

        // ── «توصيل» أو «إحضار» — خياران كبيران في مفتاحٍ واحد، ولا مختارَ افتراضاً (ملاحظةُ المالك
        // ٢٠٢٦-١٠-١٦): «لازم يكونن زرين واضحين لازم يختار وحدة منهن». **والنموذجُ لا يظهر قبل الاختيار.**
        Spacer(Modifier.height(Rahal.space.sm))
        ModeSwitch(vm)
        if (vm.mode.isEmpty()) {
            // **و«توصيلاتي» تبقى ظاهرةً قبل الاختيار** — لا تختفي مع النموذج.
            RecentDeliveries(vm) { tracking = it }
            Spacer(Modifier.height(Rahal.space.xl))
            return@Screen
        }
        // **وسطرٌ يشرح ما اختاره** — الكابتنُ يأخذ منك أو يحضر إليك.
        Spacer(Modifier.height(Rahal.space.sm))
        Text(
            stringResource(if (fetch) R.string.md_mode_fetch_sub else R.string.md_mode_send_sub),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.caption,
            modifier = Modifier.padding(horizontal = Rahal.space.xs),
        )

        // ══════════════════════════════════════════════════════════════
        // ١ · المستلِم (أو الشخص في مكان الاستلام)
        // ══════════════════════════════════════════════════════════════
        FormSection(
            1,
            stringResource(if (fetch) R.string.md_sec_recipient_fetch else R.string.md_sec_recipient),
            danger = tried && (vm.name.isBlank() || vm.phone.isBlank()),
        )
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.name,
            onValueChange = { vm.onName(it) },
            label = { Text(stringResource(if (fetch) R.string.md_name_fetch else R.string.md_name)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_user) },
            isError = tried && vm.name.isBlank(),
            singleLine = true,
            enabled = !vm.busy,
            modifier = Modifier
                .fillMaxWidth()
                .onFocusChanged { if (!it.isFocused) vm.leaveField("name") },
        )
        Spacer(Modifier.height(Rahal.space.sm))
        // **والرقمُ بحقل الهاتف المشترك** — بأيقونته ومثاله.
        PhoneField(
            value = vm.phone,
            onChange = { vm.onPhone(it) },
            enabled = !vm.busy,
            label = if (fetch) R.string.md_phone_fetch else R.string.md_phone,
            modifier = Modifier.onFocusChanged { if (!it.isFocused) vm.leaveField("phone") },
        )
        // **مستلموه السابقون بطاقاتٌ تُمرَّر أفقيّاً** (قرارُ المالك ٢٠٢٦-١١-٠٥، والتصميمُ المعتمد
        // ٢٠٢٦-١١-٢٧) — تظهر بعد حرفين من الاسم أو الرقم، **وتحت الحقلين كي لا تزيح ما يكتب فيه**،
        // وتُطوى حين يخرج منه.
        if (vm.suggestFor.isNotEmpty()) RecipientSuggestions(vm)

        // ══════════════════════════════════════════════════════════════
        // ٢ · العنوان والموقع
        // ══════════════════════════════════════════════════════════════
        FormSection(
            2,
            stringResource(if (fetch) R.string.md_address_fetch else R.string.md_address),
            danger = tried && vm.address.isBlank(),
        )
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.address,
            onValueChange = { vm.onAddress(it) },
            label = { Text(stringResource(if (fetch) R.string.md_address_fetch else R.string.md_address)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_home) },
            isError = tried && vm.address.isBlank(),
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        // **ونقطةُ آخر توصيلةٍ لمن اختير تُقال** — وتُزال بلمسة فتعود الأجرةُ بلا نقطة.
        if (vm.pointFromSaved) {
            Spacer(Modifier.height(Rahal.space.xs))
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success, padding = Rahal.space.sm) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(if (fetch) R.string.md_saved_point_fetch else R.string.md_saved_point),
                        color = Rahal.colors.success,
                        style = Rahal.type.label,
                        modifier = Modifier.weight(1f),
                    )
                    RahalTextButton(onClick = { vm.clearPoint() }, enabled = !vm.busy) {
                        Text(stringResource(R.string.md_saved_point_remove))
                    }
                }
            }
        }

        // ── موقعُ المستلِم: «اطلب الموقع» وجنبه «لصق رابط» بصفٍّ واحد (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) ──
        // **وحقلُ الرابط مخفيٌّ حتّى يُطلب** — ولا سطرَ شرح («مافي داعي للشرح»).
        var linkOpen by rememberSaveable { mutableStateOf(false) }
        val showLink = linkOpen || vm.link.isNotEmpty()
        Spacer(Modifier.height(Rahal.space.sm))
        AskLocationCard(vm, fetch, linkOpen = showLink) { linkOpen = !linkOpen }
        if (showLink) {
            Spacer(Modifier.height(Rahal.space.sm))
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
                ) { Text(stringResource(R.string.md_paste)) }
            }
        }
        when (vm.linkState) {
            LinkState.Checking -> Text(
                stringResource(R.string.md_link_checking),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
            LinkState.Found -> Text(
                stringResource(R.string.md_link_found),
                color = Rahal.colors.success,
                style = Rahal.type.caption,
            )
            LinkState.NotFound -> Text(
                stringResource(if (fetch) R.string.md_link_not_found_fetch else R.string.md_link_not_found),
                color = Rahal.colors.danger,
                style = Rahal.type.caption,
            )
            LinkState.Idle -> Unit
        }
        // **وسطرُ حالِ الموقع تحت الزرّين** — بلا نقطةٍ تُحسب الأجرةُ الأساسيّةُ وحدَها (وهذا ما كان
        // يُقال في بطاقة الأجرة)، وإن وصلت النقطةُ قالت بطاقاتُها فوق ذلك.
        if (vm.point == null && vm.ask != AskState.Waiting && vm.ask != AskState.Sending && vm.linkState == LinkState.Idle) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(if (fetch) R.string.md_fee_no_point_fetch else R.string.md_fee_no_point),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
        // **ولا تحديدَ يدويّاً على الخريطة** (طلبُ المالك ٢٠٢٦-١٠-٠٩: «صفحة نظيفة»)
        // — الموقعُ من رابطٍ يشاركه الزبون، والعنوانُ المكتوبُ يكمّله.

        // ══════════════════════════════════════════════════════════════
        // ٣ · الغرض
        // ══════════════════════════════════════════════════════════════
        // **«يحضر» لا «يوصّل» في الإحضار** — الكابتنُ يأتي بالغرض إلى المتجر.
        FormSection(3, stringResource(if (fetch) R.string.md_parcel_fetch else R.string.md_parcel))
        OutlinedTextField(
            shape = Rahal.shape.md,
            value = vm.parcel,
            onValueChange = { vm.parcel = it.take(200); vm.touched() },
            label = { Text(stringResource(if (fetch) R.string.md_parcel_fetch else R.string.md_parcel)) },
            placeholder = { Text(stringResource(if (fetch) R.string.md_parcel_hint_fetch else R.string.md_parcel_hint)) },
            leadingIcon = { FieldIcon(com.rahalgo.ui.R.drawable.ic_orders) },
            enabled = !vm.busy,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Rahal.space.sm))
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

        // ══════════════════════════════════════════════════════════════
        // ٤ · من يدفع — ثلاثُ بطاقاتٍ كبيرةٍ إجباريّة (طلبُ المالك ٢٠٢٦-١٠-٠٩) بسطرِ شرحٍ لكلٍّ منها
        // ══════════════════════════════════════════════════════════════
        FormSection(4, stringResource(R.string.md_who_pays), danger = tried && vm.payer.isEmpty())
        listOf(
            "merchant" to R.string.md_pay_me,
            "merchant_cash" to R.string.md_pay_me_cash,
            "recipient" to if (fetch) R.string.md_pay_recipient_fetch else R.string.md_pay_recipient,
        ).forEach { (key, title) ->
            val on = vm.payer == key
            // **ولا دين** (قرارُ المالك ٢٠٢٦-١٠-١٥): المتاحُ لا يكفي ⇒ «من محفظتي» معطَّل.
            val walletShort = key == "merchant" && vm.quote?.merchantCanPay == false
            val usable = !vm.busy && !walletShort
            RahalCard(
                modifier = Modifier.padding(vertical = Rahal.space.xs),
                variant = if (on) CardVariant.Tinted else CardVariant.Outlined,
                tone = if (on) StatusTone.Info else null,
                onClick = if (usable) ({ vm.pickPayer(key) }) else null,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    androidx.compose.material3.RadioButton(selected = on, onClick = { vm.pickPayer(key) }, enabled = usable)
                    Spacer(Modifier.width(Rahal.space.sm))
                    Column(Modifier.weight(1f)) {
                        Text(
                            stringResource(title),
                            style = Rahal.type.cardTitle,
                            color = if (walletShort) Rahal.colors.inkMuted else Rahal.colors.ink,
                        )
                        // **والمحفظةُ لا تكفي — حالٌ يُقال**، وإلّا فسطرٌ يقول متى تُدفع.
                        if (walletShort) {
                            Text(
                                stringResource(R.string.md_wallet_short),
                                color = Rahal.colors.danger,
                                style = Rahal.type.caption,
                            )
                        } else {
                            Text(
                                when (key) {
                                    "merchant" -> q?.let { stringResource(R.string.md_fee_from_wallet, com.rahalgo.ui.grouped(it.availableBalance)) }
                                        ?: stringResource(R.string.md_pay_sub_wallet)
                                    "merchant_cash" -> stringResource(if (fetch) R.string.md_fee_me_cash_fetch else R.string.md_fee_me_cash)
                                    else -> stringResource(if (fetch) R.string.md_fee_cash_fetch else R.string.md_fee_cash)
                                },
                                color = Rahal.colors.inkMuted,
                                style = Rahal.type.caption,
                            )
                        }
                    }
                }
            }
        }

        // ── الأجرةُ قبل الإرسال ───────────────────────────────────────
        if (vm.quoteError.isEmpty() && q != null) {
            Spacer(Modifier.height(Rahal.space.sm))
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Info) {
                // **والأجرةُ المُعفاةُ: الأصلُ مشطوباً وبجانبه «مجاناً» ومصدرُه** (٢٠٢٦-١١-٢٧).
                if (q.freeWaived > 0) {
                    Text(
                        stringResource(R.string.md_fee, com.rahalgo.ui.grouped(q.fee + q.freeWaived)),
                        style = Rahal.type.body,
                        color = Rahal.colors.inkMuted,
                        textDecoration = androidx.compose.ui.text.style.TextDecoration.LineThrough,
                    )
                    Text(
                        if (q.fee > 0) stringResource(R.string.md_fee, com.rahalgo.ui.grouped(q.fee))
                        else com.rahalgo.ui.freeSourceText(q.freeSource),
                        style = Rahal.type.priceLarge,
                        color = Rahal.colors.success,
                    )
                    if (q.fee > 0) {
                        Text(com.rahalgo.ui.freeSourceText(q.freeSource), style = Rahal.type.caption, color = Rahal.colors.success)
                    }
                } else Text(
                    stringResource(R.string.md_fee, com.rahalgo.ui.grouped(q.fee)),
                    style = Rahal.type.priceLarge,
                    color = Rahal.colors.brand,
                )
                Text(
                    when (vm.payer) {
                        "merchant" -> stringResource(R.string.md_fee_from_wallet, com.rahalgo.ui.grouped(q.availableBalance))
                        "merchant_cash" -> stringResource(if (fetch) R.string.md_fee_me_cash_fetch else R.string.md_fee_me_cash)
                        "recipient" -> stringResource(if (fetch) R.string.md_fee_cash_fetch else R.string.md_fee_cash)
                        else -> stringResource(R.string.md_fee_choose)
                    },
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.body,
                )
                // **ومن أين الأجرة** (٢٠٢٦-١٠-١٥): بلا نقطةٍ الأساسُ وحدَه (يُقال تحت زرّي الموقع)، ومعها المسافة.
                if (q.feeBasis == "distance" && q.distanceM > 0) {
                    Text(
                        stringResource(R.string.md_fee_distance, String.format(java.util.Locale.US, "%.1f", q.distanceM / 1000.0)),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                    )
                }
            }
            // **ولا يكفيه أن يدفع من محفظته — يُقال قبل الضغط** لا ٤٠٢ بعده.
            if (vm.payer == "merchant" && !q.merchantCanPay) {
                Spacer(Modifier.height(Rahal.space.xs))
                Note(stringResource(R.string.md_cannot_pay), Rahal.colors.danger)
            }
        }

        // ══════════════════════════════════════════════════════════════
        // «هل تريد أن يقبض الكابتن مبلغاً؟» — مفتاحٌ يكشف الحقل (٢٠٢٦-١٠-١٥)، **عرضٌ لا حساب**.
        // **وفي «إحضار» يدفعه بمكان الاستلام ويأخذه منك** (٢٠٢٦-١٠-١٦) — `DeliveryTexts`.
        // ══════════════════════════════════════════════════════════════
        FormSection(null, stringResource(DeliveryTexts.collectQuestion(fetch)), optional = true)
        RahalCard(variant = CardVariant.Outlined) {
            Row(
                Modifier
                    .fillMaxWidth()
                    .height(Rahal.size.touch)
                    .clickable(enabled = !vm.busy) { vm.toggleCollect(!vm.collectOn) },
                verticalAlignment = Alignment.CenterVertically,
            ) {
                androidx.compose.material3.Switch(
                    checked = vm.collectOn,
                    onCheckedChange = { vm.toggleCollect(it) },
                    enabled = !vm.busy,
                )
                Spacer(Modifier.width(Rahal.space.md))
                Text(
                    stringResource(if (vm.collectOn) R.string.md_collect_on else R.string.md_collect_off),
                    style = Rahal.type.body,
                    color = Rahal.colors.ink,
                )
            }
            if (vm.collectOn) {
                // **ليرة أو دولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — **ومنذ ٢٠٢٦-١٠-٢٩ دائرتان بجانب
                // خانة المبلغ** («ل.س» و«$»)، والمختارةُ مملوءةٌ بلون العلامة.
                val usd = isUsd(vm.collectCurrency)
                Spacer(Modifier.height(Rahal.space.sm))
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
        }

        RecentDeliveries(vm) { tracking = it }
    }
    // ══════════════════════════════════════════════════════════════
    // **وما ينقص يُقال فوق الزرّ** (تدقيقُ ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════
    //
    // **كان الزرُّ يصير رماديّاً بلا كلمة** — والقائمةُ محسوبةٌ ولا تُعرض، فيعبّي
    // التاجرُ الاستمارةَ الطويلة ويجد الزرَّ واقفاً ولا يعرف ليش. **فالناقصُ سطرٌ
    // فوقه بأرضٍ دافئة، ولمسُه يعلّم خاناته بالأحمر**، **فإذا اكتمل صار «جاهز» وانفتح الزرّ.**
    //
    // **والزرُّ ثابتٌ أسفلَ الشاشة** (إعادةُ التصميم ٢٠٢٦-١١-١١) — لا يضيع في آخر نموذجٍ طويل.
    if (vm.mode.isNotEmpty()) {
        BottomActionBar {
            if (vm.sent) Note(stringResource(R.string.md_sent), Rahal.colors.success)
            if (vm.error.isNotEmpty()) Note(vm.error, Rahal.colors.danger)
            if (missing.isNotEmpty()) {
                Text(
                    stringResource(R.string.md_missing, missing.joinToString("، ")),
                    color = if (tried) Rahal.colors.danger else Rahal.colors.accentInk,
                    style = Rahal.type.caption,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.sm)
                        .background(Rahal.colors.warnTint)
                        .clickable { tried = true }
                        .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
                )
            } else {
                Text(
                    if (q != null) stringResource(R.string.md_ready_fee, com.rahalgo.ui.grouped(q.fee))
                    else stringResource(R.string.md_ready),
                    color = Rahal.colors.success,
                    style = Rahal.type.caption,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.sm)
                        .background(Rahal.colors.success.copy(alpha = 0.10f))
                        .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
                )
            }
            RahalButton(
                onClick = {
                    tried = false
                    vm.send()
                },
                enabled = !vm.busy && !blocked && vm.storeId.isNotEmpty() && missing.isEmpty(),
                // **زرٌّ كبيرٌ بعرض الشاشة** (التصميمُ المعتمد ٢٠٢٦-١٠-٢٨).
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    stringResource(if (vm.busy) R.string.md_sending else R.string.md_send),
                    style = Rahal.type.button,
                )
            }
        }
    }
    }
}

/** **التوصيلةُ الجارية** — بطاقةٌ بلون العلامة بزرّ «تابعها»، **ولمسُها كلُّها يفتح المتابعة.** */
@Composable
private fun LiveBanner(live: Delivery, onOpen: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.brand)
            .clickable(role = Role.Button, onClick = onOpen)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        PulseDot(Rahal.colors.accent)
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.md_active_title, live.number),
                color = Rahal.colors.onBrand,
                style = Rahal.type.bodyStrong,
            )
            Text(
                deliveryStatusText(live.status, live.deliveryMode == "fetch"),
                color = Rahal.colors.onBrand.copy(alpha = 0.85f),
                style = Rahal.type.caption,
            )
        }
        Spacer(Modifier.width(Rahal.space.sm))
        androidx.compose.foundation.layout.Box(
            Modifier
                .height(Rahal.size.iconBox)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.canvas)
                .padding(horizontal = Rahal.space.lg),
            contentAlignment = Alignment.Center,
        ) {
            Text(stringResource(R.string.md_follow), color = Rahal.colors.brand, style = Rahal.type.bodyStrong)
        }
    }
}

/** **«توصيل» أو «إحضار»** — مفتاحٌ كبيرٌ بخيارين، المختارُ أبيضُ بنصٍّ بلون العلامة، **ولا مختارَ قبل اللمس.** */
@Composable
private fun ModeSwitch(vm: DeliveryViewModel) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.field)
            .padding(Rahal.space.xs),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
    ) {
        listOf(
            "send" to R.string.md_mode_send,
            "fetch" to R.string.md_mode_fetch,
        ).forEach { (key, label) ->
            val on = vm.mode == key
            androidx.compose.foundation.layout.Box(
                Modifier
                    .weight(1f)
                    .height(Rahal.size.touch)
                    .clip(Rahal.shape.sm)
                    .background(if (on) Rahal.colors.canvas else Color.Transparent)
                    .then(if (on) Modifier.border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.sm) else Modifier)
                    .clickable(enabled = !vm.busy, role = Role.RadioButton) { vm.pickMode(key) },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    stringResource(label),
                    style = Rahal.type.sectionTitle,
                    color = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
                    textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                )
            }
        }
    }
}

/** **آخرُ التوصيلات** — تحت النموذج، **وقبل اختيار النوع أيضاً.** */
@Composable
private fun RecentDeliveries(vm: DeliveryViewModel, onOpen: (String) -> Unit) {
    // ── آخرُ التوصيلات — والسجلُّ كلُّه في القائمة الجانبيّة ────────
    Spacer(Modifier.height(Rahal.space.lg))
    SectionHeader(stringResource(R.string.md_mine))
    if (vm.listError.isNotEmpty()) {
        Note(vm.listError, Rahal.colors.danger)
        RahalTextButton(onClick = { vm.load() }) {
            Text(stringResource(com.rahalgo.ui.R.string.act_retry))
        }
    }
    val rows = vm.list
    when {
        rows == null && vm.listError.isEmpty() ->
            Text(stringResource(R.string.md_loading), color = Rahal.colors.inkMuted, style = Rahal.type.body)
        // **والفارغُ بالقطعة المشتركة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كبقيّة الشاشات، ويقول ماذا يفعل.
        rows != null && rows.isEmpty() ->
            com.rahalgo.ui.Empty(stringResource(R.string.md_none))
        rows != null -> DeliveryGroups(rows.take(5), cancelling = vm.cancelling != null, onOpen = onOpen) { vm.cancel(it) }
    }
}

/** **«جارية الآن» ثمّ «المنتهية»** — بطاقةٌ لكلّ توصيلة (التصميمُ المعتمد ٢٠٢٦-١١-٢٧). */
@Composable
fun DeliveryGroups(rows: List<Delivery>, cancelling: Boolean, onOpen: (String) -> Unit, onCancel: (String) -> Unit) {
    val (live, ended) = rows.partition { toneOf(it.status) == Tone.Waiting || toneOf(it.status) == Tone.Moving }
    if (live.isNotEmpty()) {
        Spacer(Modifier.height(Rahal.space.sm))
        Text(stringResource(R.string.md_group_live, live.size), color = Rahal.colors.inkMuted, style = Rahal.type.label)
        live.forEach { d -> DeliveryRow(d, cancelling = cancelling, onOpen = { onOpen(d.id) }) { onCancel(d.id) } }
    }
    if (ended.isNotEmpty()) {
        Spacer(Modifier.height(Rahal.space.md))
        Text(stringResource(R.string.md_group_done), color = Rahal.colors.inkMuted, style = Rahal.type.label)
        ended.forEach { d -> DeliveryRow(d, cancelling = cancelling, onOpen = { onOpen(d.id) }) { onCancel(d.id) } }
    }
}

/**
 * **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — زرٌّ وحالُه تحته:
 * يُرسَل · ينتظر (ومشاركةُ الرابط إن لم تصل الرسالة) · **✓ وصل بدقّته** · انتهى.
 */
@Composable
private fun AskLocationCard(vm: DeliveryViewModel, fetch: Boolean, linkOpen: Boolean, onLink: () -> Unit) {
    val uri = androidx.compose.ui.platform.LocalUriHandler.current
    val waiting = vm.ask == AskState.Sending || vm.ask == AskState.Waiting
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        // **«اطلب الموقع» الزرُّ الأساسيّ** (التصميمُ المعتمد ٢٠٢٦-١١-٢٧) — مملوءٌ، و«لصق رابط» بجانبه محدَّد.
        RahalButton(
            onClick = { vm.askLocation() },
            enabled = !vm.busy && !waiting,
            modifier = Modifier
                .weight(1.4f)
                .defaultMinSize(minHeight = Rahal.size.touch),
        ) {
            Icon(painterResource(com.rahalgo.ui.R.drawable.ic_pin), contentDescription = null, tint = Rahal.colors.onBrand)
            Spacer(Modifier.width(Rahal.space.sm))
            Text(
                stringResource(
                    when {
                        vm.ask == AskState.Sending -> R.string.md_ask_sending
                        fetch -> R.string.md_ask_loc_fetch
                        else -> R.string.md_ask_loc
                    },
                ),
            )
        }
        Spacer(Modifier.width(Rahal.space.sm))
        // **«لصق رابط» زرٌّ باسمه بجانب الزرّ** (قرارُ المالك ٢٠٢٦-١١-١١: «الأيقونةُ بلا اسم») — يفتح
        // حقلَ الرابط وقتَ الحاجة، **ومختارُه يبقى بحدٍّ ثخين** كما كان.
        RahalOutlineButton(
            onClick = onLink,
            enabled = !vm.busy,
            compact = true,
            modifier = Modifier
                .weight(1f)
                .defaultMinSize(minHeight = Rahal.size.touch),
        ) {
            Icon(painterResource(com.rahalgo.ui.R.drawable.ic_copy), contentDescription = null, tint = Rahal.colors.brand)
            Spacer(Modifier.width(Rahal.space.xs))
            Text(
                stringResource(R.string.md_paste_link),
                fontWeight = if (linkOpen) androidx.compose.ui.text.font.FontWeight.Bold else androidx.compose.ui.text.font.FontWeight.Normal,
            )
        }
    }
    val req = vm.askReq
    when {
        vm.ask == AskState.Waiting && req != null -> {
            Spacer(Modifier.height(Rahal.space.xs))
            RahalCard(variant = CardVariant.Outlined) {
                Text(
                    stringResource(if (req.whatsappSent) R.string.md_ask_waiting else R.string.md_ask_not_sent),
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                )
                // **ولم تصل الرسالة** (البوتُ غيرُ مقترن) — يشارك الرابطَ بنفسه بلمسة.
                if (!req.whatsappSent && req.waUrl.isNotEmpty()) {
                    RahalTextButton(onClick = { runCatching { uri.openUri(req.waUrl) } }) {
                        Text(stringResource(R.string.md_ask_share))
                    }
                }
                // **«انسخ الرابط» ظاهرٌ باسمه ومعه تأكيد** (قرارُ المالك ٢٠٢٦-١١-١١، كالويب).
                if (req.link.isNotEmpty()) {
                    val clipboard = androidx.compose.ui.platform.LocalClipboardManager.current
                    val copied = stringResource(R.string.md_link_copied)
                    RahalTextButton(onClick = {
                        clipboard.setText(androidx.compose.ui.text.AnnotatedString(req.link))
                        com.rahalgo.ui.Flash.ok(copied)
                    }) {
                        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_copy), contentDescription = null, tint = Rahal.colors.brand)
                        Spacer(Modifier.width(Rahal.space.xs))
                        Text(stringResource(R.string.md_copy_link))
                    }
                }
            }
        }
        vm.ask == AskState.Received -> {
            Spacer(Modifier.height(Rahal.space.xs))
            RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success) {
                Text(
                    stringResource(if (fetch) R.string.md_ask_received_fetch else R.string.md_ask_received),
                    color = Rahal.colors.success,
                    style = Rahal.type.cardTitle,
                )
                vm.askAccuracy?.let {
                    Text(
                        stringResource(R.string.md_ask_accuracy, com.rahalgo.ui.grouped(it.toLong())),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                    )
                }
            }
        }
        vm.ask == AskState.Expired -> Text(
            stringResource(R.string.md_ask_expired),
            color = Rahal.colors.danger,
            style = Rahal.type.caption,
        )
    }
    if (vm.askError.isNotEmpty()) {
        Text(vm.askError, color = Rahal.colors.danger, style = Rahal.type.caption)
    }
}

/**
 * **مستلموه السابقون بطاقاتٌ تُمرَّر أفقيّاً** (قرارُ المالك ٢٠٢٦-١١-٠٥، والتصميمُ المعتمد ٢٠٢٦-١١-٢٧)
 * — الاسمُ ثمّ الرقم، **وعلامةُ موقعٍ إن عُرفت نقطتُه**، و✕ «شيلو من القائمة». واللمسةُ تملأ النموذج.
 */
@Composable
private fun RecipientSuggestions(vm: DeliveryViewModel) {
    Spacer(Modifier.height(Rahal.space.sm))
    Text(
        stringResource(R.string.md_saved_title),
        color = Rahal.colors.inkMuted,
        style = Rahal.type.label,
        modifier = Modifier.padding(horizontal = Rahal.space.xs),
    )
    Spacer(Modifier.height(Rahal.space.xs))
    Row(
        Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        vm.suggestions.forEach { r ->
            RahalCard(
                // **عرضٌ ثابتٌ لبطاقة الصفّ الأفقيّ** — فلا تتمدّد في مساحةٍ غير محدودة.
                modifier = Modifier.width(220.dp),
                variant = CardVariant.Outlined,
                padding = Rahal.space.sm,
                onClick = if (!vm.busy) ({ vm.pickRecipient(r) }) else null,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(
                            r.name,
                            color = Rahal.colors.ink,
                            style = Rahal.type.cardTitle,
                            maxLines = 1,
                            overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                        )
                        Text(
                            com.rahalgo.ui.phoneText(RecipientSuggest.localPhone(r.phone)),
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.caption,
                            maxLines = 1,
                        )
                        if (r.lat != null && r.lng != null) {
                            Text(
                                "● " + stringResource(R.string.md_saved_has_point),
                                color = Rahal.colors.success,
                                style = Rahal.type.meta,
                            )
                        }
                    }
                    androidx.compose.foundation.layout.Box(
                        Modifier
                            .size(Rahal.size.iconBox)
                            .clip(androidx.compose.foundation.shape.CircleShape)
                            .clickable(enabled = !vm.busy) { vm.hideRecipient(r) },
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            painterResource(com.rahalgo.ui.R.drawable.ic_close),
                            contentDescription = stringResource(R.string.md_saved_hide),
                            tint = Rahal.colors.inkMuted,
                            modifier = Modifier.size(Rahal.icon.xs),
                        )
                    }
                }
            }
        }
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
                    .size(Rahal.size.action)
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
                    style = Rahal.type.bodyStrong,
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
 * **بطاقةُ توصيلة** — في «آخر التوصيلات» وفي السجلّ، نسخةٌ واحدة (التصميمُ المعتمد ٢٠٢٦-١١-٢٧).
 *
 * رأسٌ برقمها وشارةِ حالها الملوّنة · المستلِمُ وعنوانُه · الغرضُ والأجرةُ ووقتُها · **وشريطٌ جانبيٌّ
 * للجارية، ووسمُ «إحضار»** · **وتنبض «جاري البحث عن كابتن…» ما دامت تنتظر** · **وزرُّ إلغاءٍ أحمرُ
 * يسأل قبل أن يُلغي** (طلبُ المالك ٢٠٢٦-١٠-٠١) — **واللمسةُ تفتح التفاصيل والإلغاءَ والمتابعة.**
 */
@Composable
fun DeliveryRow(d: Delivery, cancelling: Boolean, onOpen: () -> Unit = {}, onCancel: () -> Unit) {
    Spacer(Modifier.height(Rahal.space.sm))
    val live = toneOf(d.status) == Tone.Waiting || toneOf(d.status) == Tone.Moving
    val fetch = d.deliveryMode == "fetch"
    // **سطران مطويّان، واللمسةُ تفتح التفاصيل** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
    var open by rememberSaveable(d.id) { mutableStateOf(false) }
    androidx.compose.foundation.layout.Box(
        Modifier
            .fillMaxWidth()
            // **والملغاةُ والمرفوضةُ أخفتُ قليلاً** — انتهت ولا شيءَ يُنتظر منها.
            .alpha(if (d.status in setOf("cancelled", "rejected", "failed")) 0.7f else 1f),
    ) {
        RahalCard(
            variant = CardVariant.Outlined,
            tone = if (live) StatusTone.Info else null,
            onClick = { open = !open },
        ) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.md_track_title, d.number.toString()),
                    style = Rahal.type.cardTitle,
                    color = Rahal.colors.ink,
                    modifier = Modifier.weight(1f),
                )
                // **رسائلُ الكابتن التي لم تُقرأ** (قرارُ المالك ٢٠٢٦-١١-٢٦) — شارةٌ قبل الحال.
                if (d.chatUnread > 0) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_chat),
                        contentDescription = stringResource(R.string.md_chat),
                        tint = Rahal.colors.brand,
                        modifier = Modifier.size(Rahal.icon.sm),
                    )
                    androidx.compose.material3.Badge { Text(if (d.chatUnread > 99) "99+" else "${d.chatUnread}") }
                    Spacer(Modifier.width(Rahal.space.sm))
                }
                StatusPill(d.status, fetch)
            }
            Spacer(Modifier.height(Rahal.space.xs))
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                if (fetch) {
                    com.rahalgo.ui.Badge(stringResource(R.string.md_row_fetch_tag), StatusTone.Warning)
                    Spacer(Modifier.width(Rahal.space.sm))
                }
                Text(
                    d.recipientName,
                    color = Rahal.colors.ink,
                    style = Rahal.type.bodyStrong,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
            }
            Text(
                d.addressText,
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
                maxLines = 1,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(Rahal.space.sm))
            if (d.parcelNote.isNotBlank()) {
                MetaTag(d.parcelNote)
                Spacer(Modifier.height(Rahal.space.xs))
            }
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f, fill = false)) {
                    MetaTag(stringResource(DeliveryTexts.feeRow(d.feePayer, fetch), com.rahalgo.ui.grouped(d.fee)))
                }
                // **ويومُها وساعتُها** (تدقيقُ ٢٠٢٦-١٠-٢٧).
                if (d.createdAt.isNotBlank()) {
                    Spacer(Modifier.width(Rahal.space.sm))
                    Text(
                        com.rahalgo.merchant.Dates.stamp(androidx.compose.ui.platform.LocalContext.current, d.createdAt),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.meta,
                        maxLines = 1,
                    )
                }
            }
            if (open) {
                Hairline()
                if (d.status == "dispatching") {
                    SearchingBanner(compact = true)
                    Spacer(Modifier.height(Rahal.space.sm))
                }
                InfoLine(com.rahalgo.ui.R.drawable.ic_phone, com.rahalgo.ui.phoneText(d.recipientPhone))
                if (!d.dropoffKnown) {
                    InfoLine(com.rahalgo.ui.R.drawable.ic_pin, stringResource(R.string.md_row_no_point), muted = true)
                }
                d.collectAmount?.takeIf { it > 0 }?.let {
                    InfoLine(
                        com.rahalgo.ui.R.drawable.ic_wallet,
                        stringResource(DeliveryTexts.collectRow(fetch), DeliveryTexts.amount(d, it)),
                    )
                }
                // **و«تابعها» لِما يجري وحدَه** — المنتهيةُ لا تُتابَع.
                if (live) {
                    RahalTextButton(onClick = onOpen) {
                        Text(stringResource(R.string.md_follow))
                    }
                }
                if (d.status in cancellable) {
                    Spacer(Modifier.height(Rahal.space.sm))
                    CancelDeliveryButton(busy = cancelling, onConfirm = onCancel)
                }
            }
        }
        // **شريطٌ جانبيٌّ بلون العلامة للجارية** — يُرى من بعيد أنّها تجري.
        if (live) {
            Box(
                Modifier
                    .align(Alignment.CenterStart)
                    .fillMaxHeight(0.6f)
                    .width(Rahal.space.xs)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.brand),
            )
        }
    }
}

/** **وسمٌ صغيرٌ** على أرضٍ هادئة — الغرضُ وملخّصُ الأجرة. */
@Composable
private fun MetaTag(text: String) {
    Text(
        text,
        color = Rahal.colors.inkMuted,
        style = Rahal.type.meta,
        maxLines = 1,
        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
        modifier = Modifier
            .clip(Rahal.shape.sm)
            .background(Rahal.colors.surface)
            .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xxs),
    )
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
        // **والعنوانُ في رأس الصفحة** (`OverlayPage`) — بسهم رجوع.
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.sm))
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
                com.rahalgo.ui.Empty(stringResource(R.string.md_none))
            rows != null -> DeliveryGroups(rows, cancelling = vm.cancelling != null, onOpen = { tracking = it }) { vm.cancel(it) }
        }
        if (vm.more) {
            Spacer(Modifier.height(Rahal.space.sm))
            RahalOutlineButton(
                onClick = { vm.next() },
                enabled = !vm.busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.md_more)) }
        }
        Spacer(Modifier.height(Rahal.space.xl))
    }
}

/** **حالُ رابط الموقع** — فارغ · يُقرأ · وُجد · لا موقعَ فيه. */
enum class LinkState { Idle, Checking, Found, NotFound }

/** **حالُ «اطلب الموقع من المستلم»** — لا طلب · يُرسَل · ينتظر · وصل · انتهى. */
enum class AskState { Idle, Sending, Waiting, Received, Expired }
