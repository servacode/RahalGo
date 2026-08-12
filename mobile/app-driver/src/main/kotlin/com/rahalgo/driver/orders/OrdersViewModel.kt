package com.rahalgo.driver.orders

import android.app.Application
import android.os.SystemClock
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.location.LastPoint
import com.rahalgo.driver.location.LocationPermission
import com.rahalgo.driver.trip.ChatState
import com.rahalgo.driver.trip.Stop
import com.rahalgo.driver.trip.TripState
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute
import com.rahalgo.driver.trip.TripStep
import org.maplibre.android.geometry.LatLng
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حال الطلبات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثلاثة نداءات في واحد**: المعروض · ما في يده · وحال ورديّته. **وهي
 * تُقرأ معا** لأنّ قائمة عروض فارغة معناها مختلف تماما حسب الوردية:
 * **راكد، أو مغلق على نفسه.**
 */
class OrdersViewModel(app: Application) : AndroidViewModel(app) {

    private companion object {
        /** **حدّ «وصلت»** — بالمتر. */
        const val ARRIVAL_M = 80f

        /** **كم بين نظرةٍ وأخرى** — والوقوفُ يُقاس بالثواني لا بالنبضات. */
        const val ARRIVAL_TICK_MS = 5_000L

        /** **كم يقف حتّى يُعدّ واصلا** — (قرار المالك: بين ٣٠ و٦٠ ثانية). */
        const val ARRIVAL_HOLD_MS = 30_000L
    }


    var state by mutableStateOf(OrdersState())
        private set

    // ══════════════════════════════════════════════════════════════════
    // **وما يقرؤه `init` يُعلَن قبله**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وخصائصُ `by mutableStateOf` تُهيَّأ بترتيب كتابتها**: و`init`
    // ينادي `refresh()` **فيقرأ خاصّيّةً لم تُهيَّأ بعد** — ويسقط
    // التطبيق عند الإقلاع بـ`NullPointerException`، **لا عند الاستعمال
    // فيُعرف سببُه.** (وقع ٢٠٢٦-٠٨-١٢ حين صار `load()` يقرأ `openId`.)
    //
    // **والمترجم لا يمسكها**: النوعُ غيرُ فارغٍ في التوقيع، **والفراغ
    // يقع في الزمن لا في النوع.**

    /**
     * **الطلب المفتوح** — وفارغ يعني القائمة معروضة.
     *
     * **ويُحدَّث من القائمة نفسها** بعد كلّ خطوة — فلا نسختان لطلب
     * واحد تفترقان.
     */
    var openId by mutableStateOf<String?>(null)
        private set

    /** **كم رسالةً تنتظره في حديث طلبه** — وصفرٌ يعني لا شارة. */
    var chatUnread by mutableStateOf(0)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **مسارُ الطرف الحاليّ** — بالشوارع لا بالهواء
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وفارغٌ يعني «لا مسار»**: محرّكُ المسارات قد ينام، **والخريطة
    // ترسم خطَّها المستقيم كما كانت** ولا تقف.
    var route by mutableStateOf<OrderRoute?>(null)
        private set

    private val backend = Backend.of(getApplication())

    init {
        refresh()
        // ══════════════════════════════════════════════════════════════
        // **والقائمة تتحدّث بنفسها**
        // ══════════════════════════════════════════════════════════════
        //
        // (البند الثالث في قائمة المالك ٢٠٢٦-٠٨-١٢: «القائمة تتحدّث
        //  لحالها» — وكان يلزم أن يخرج من التبويب ويعود.)
        //
        // **والإشارة بلا حمولة**: يقول المحرّك «تغيّر شيء»، **وما يراه
        // هذا السائق تقرّره نقطة الطابور** بحسب ورديّته ونمط التوزيع.
        watchArrival()
        backend.live.start(
            scope = viewModelScope,
            onState = { up -> Log.i("RahalGo/live", if (up) "الوصلة قامت" else "الوصلة انقطعت") },
            onEvent = {
                refresh()
                // **والحديثُ المفتوح يُعاد قراءته** — الإشارةُ تقول
                // «تغيّر شيء»، **ومن أعاد القائمة وحدَها** ترك صاحبَه
                // ينظر إلى حديثٍ لا يتحرّك وقد وصلته رسالة.
                if (chat != null) reloadChat()
            },
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **والوصولُ يُعلن نفسَه — لا يُضغط**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «يكون ذكيّاً بمجرّد أن الموقع أصبح عند
    //  المتجر… في حال السائق توقّف بين الـ٣٠ والـ٦٠ ثانية».)
    //
    // **و«وصلت» حقيقةٌ يعرفها الجهازُ أدقَّ من ذاكرة سائقٍ يقود** —
    // وكلُّ ضغطةٍ تُرفع عنه ربحٌ خالص.
    //
    // # ولماذا وقوفٌ لا لمسةُ دائرة
    //
    // **الموضعُ ينطّ**: يمرّ بجانب المتجر وهو ذاهبٌ إلى غيره فيُسجَّل
    // «وصل» ولم يصل. **ووقتُ الوصول يدخل في حساب التأخّر والشكاوى** —
    // فلا يُكتب بقراءةٍ عابرة.
    //
    // **فثلاثون ثانيةً داخل الدائرة** — من وقف عندها وقف حقّا.
    //
    // # ولماذا هذه الخطوة وحدَها
    //
    // **«وصلت» لا مالَ فيها ولا إثبات** — أسوأُ ما يقع أن تتقدّم دقيقة.
    // **و«استلمت» و«سلّمت» فيهما مالٌ وذمّة** — فتبقيان بيده وحدَه.
    //
    // **والزرُّ باقٍ**: سوقٌ مسقوفٌ يُضعف الموضع، **ومن انتظر آليّةً لا
    // تأتي** وقف عند الباب لا يعرف ماذا يفعل.
    private var nearSince: Long = 0L

    private fun watchArrival() {
        viewModelScope.launch {
            while (true) {
                delay(ARRIVAL_TICK_MS)
                val order = state.mine.firstOrNull { it.id == openId }
                    ?: state.mine.firstOrNull()
                val to = when (order?.status) {
                    "assigned" -> "at_pickup"
                    "on_the_way" -> "at_dropoff"
                    else -> null
                }
                if (order == null || to == null || !near(LastPoint.value, order)) {
                    nearSince = 0L
                    continue
                }
                val now = SystemClock.elapsedRealtime()
                if (nearSince == 0L) {
                    nearSince = now
                } else if (now - nearSince >= ARRIVAL_HOLD_MS) {
                    nearSince = 0L
                    Log.i("RahalGo/وصول", "وقوفٌ عند الوجهة — تُعلَن " + to)
                    step(to)
                }
            }
        }
    }

    override fun onCleared() {
        // **ولا تبقى وصلة بلا شاشة** — تستنزف البطارية وتوقظ الجهاز.
        backend.live.stop()
        super.onCleared()
    }

    fun refresh() {
        viewModelScope.launch { load() }
    }

    /**
     * **القراءة نفسها — وتُنتظَر.**
     *
     * **ومن قبِل طلبا يُنقل إلى رحلته فورا**، ولو نُقل قبل أن تصل
     * القائمة **لرأى شاشة رحلة فارغة لحظة** ثمّ امتلأت أمامه.
     */
    private suspend fun load() {
        // **وشارةُ الحديث تُقرأ مع كلّ تحديث** — والوصلةُ الحيّة تنادي
        // التحديث، **فما يصل يُرى في ثانيته لا في فتحةٍ تالية.**
        loadChatBadge()
        loadRoute()
        state = try {
            state.copy(
                offers = backend.driver.queue(),
                mine = backend.driver.orders(),
                // **والحال كاملا لا الوردية وحدها** — شاشة «لماذا
                // لا تصلني طلبات» تُبنى منه.
                me = backend.driver.me(),
                locationOn = LocationPermission.granted(getApplication()),
                loading = false,
                error = "",
            )
        } catch (e: Exception) {
            state.copy(loading = false, error = describe(e))
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **والرحلة تبدأ بنفسها — لا بضغطة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «برنامج ذكيّ، ما في داعي السائق يظلّ
    //  يضغط على الشاشة وألف كبسة — مجرّد ما وافق على طلب تبدأ الرحلة
    //  بشكل تلقائيّ».)
    //
    // **والقبول نفسه هو البدء**: من ضغط «موافق» لا ينتظر شيئا آخر —
    // **وضغطةٌ ثانية تقول ما قالته الأولى** خطوةٌ زائدة في يد رجل يقود.
    //
    // **ورايةٌ لا حالٌ محفوظ**: تُرفع مرّة وتُنزَل حين يُنقل، **ولو
    // بقيت مرفوعة** لأعادته إلى الرحلة كلّما فتح الطلبات.
    var startTrip by mutableStateOf(false)
        private set

    fun tripOpened() {
        startTrip = false
    }

    /**
     * **يأخذ الطلب — ثمّ يعيد قراءة القائمتين.**
     *
     * **ولا يُنقل من قائمة إلى قائمة في الشاشة تفاؤلا**: القبول قد يُرفض
     * (**سبقه غيره** أو بلغ حدّه)، **ومن نقله قبل الجواب** أرى صاحبه
     * طلبا في يده وهو ليس له.
     */
    fun accept(orderId: String) {
        if (state.acceptingId != null) return
        state = state.copy(acceptingId = orderId, error = "")
        viewModelScope.launch {
            try {
                backend.driver.accept(orderId)
            } catch (e: Exception) {
                state = state.copy(acceptingId = null, error = describe(e))
                // **والقائمة تُعاد قراءتها حتّى بعد الرفض**: «سبقك غيره»
                // يعني أنّ البطاقة لم تعد موجودة، **ومن أبقاها** جعله
                // يضغطها ثانية.
                refresh()
                return@launch
            }
            state = state.copy(acceptingId = null)
            // **والقائمة أوّلا ثمّ النقل** — لا شاشةَ رحلةٍ فارغة.
            load()
            openId = orderId
            startTrip = true
        }
    }

    /**
     * **يرفض العرض** — وينتقل الدور فورا إلى من بعده.
     *
     * **والقائمة تُعاد قراءتها بعده**: البطاقة لم تعد له، **ومن أبقاها**
     * جعله يضغطها فيُردّ «ليس عرضك».
     */
    fun decline(orderId: String) {
        if (state.acceptingId != null) return
        state = state.copy(acceptingId = orderId, error = "")
        viewModelScope.launch {
            try {
                backend.driver.decline(orderId)
            } catch (e: Exception) {
                state = state.copy(acceptingId = null, error = describe(e))
                refresh()
                return@launch
            }
            state = state.copy(acceptingId = null)
            refresh()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الطلب المفتوح — في النموذج لا في الشاشة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغ يعني القائمة معروضة.** ولماذا هنا: بعد كلّ خطوة تُعاد
    // قراءة القائمتين، **والطلب المفتوح يُحدَّث من القائمة نفسها** —
    // فلا نسختان لطلب واحد تفترقان.

    var detail by mutableStateOf(DetailState(order = com.rahalgo.shared.model.DriverOrder()))
        private set

    fun open(orderId: String) {
        openId = orderId
        detail = DetailState(order = state.mine.firstOrNull { it.id == orderId } ?: return)
    }

    fun close() {
        openId = null
    }

    // ══════════════════════════════════════════════════════════════════
    // **الطلبُ الذي يعمل عليه — موضعٌ واحدٌ يقرّره**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وشاشةُ الرحلة تعرض `mine.first()` إن لم يُفتح شيء** (`trip()`)،
    // **وكانت الأفعالُ تشترط `openId`** — فتُقرأ البطاقةُ ويُضغط زرُّها
    // **ولا يقع شيء ولا تُقال كلمة.**
    //
    // **ويقع بعد كلّ إقلاق للتطبيق**: `openId` رايةٌ في الذاكرة تُمحى مع
    // العملية، **والطلبُ باقٍ في يده** — فيفتح تطبيقَه ويضغط «وصلت
    // المتجر» فلا يستجيب. (أمسكه المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وأسوأُ ما فيه أنّه صامت**: لا خطأَ ولا دوّارة — **وزرٌّ لا يفعل
    // ولا يقول** يُقرأ تطبيقاً معطوباً.
    private fun currentId(): String? = openId ?: state.mine.firstOrNull()?.id

    /** يحرّك الطلب خطوة — **ثمّ يعيد قراءة كلّ شيء.** */
    fun step(to: String) {
        val id = currentId() ?: return
        if (detail.busy) return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.transition(id, to)
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            // **والتسليم يُغلق الطلب** — فيُعاد إلى القائمة لا إلى شاشة
            // طلب لم يعد له وجود فيها.
            if (to == "delivered") {
                openId = null
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /** يسأل المحرّك أيّ الأسباب تصلح في هذا الحال. */
    fun askFail() {
        val order = state.mine.firstOrNull { it.id == openId }
            ?: state.mine.firstOrNull()
            ?: detail.order
        viewModelScope.launch {
            val reasons = runCatching { backend.driver.failReasons(order.status) }
                .getOrDefault(emptyList())
            // **وقائمةٌ فارغةٌ ليست «لا شيء»** — هي «لا سببَ يُختار هنا»،
            // **وبابُها الطارئ** لا نافذةٌ فارغةٌ تُغلق فورا.
            if (reasons.isEmpty()) {
                emergencyOpen = true
            } else {
                detail = detail.copy(order = order, failReasons = reasons)
            }
        }
    }

    fun dismissFail() {
        detail = detail.copy(failReasons = null)
    }

    fun fail(reason: String) {
        val id = currentId() ?: return
        detail = detail.copy(failReasons = null, busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.transition(id, "failed", reason = reason)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    fun release() {
        val id = currentId() ?: return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.release(id)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **حال الرحلة — مشتقّ لا محفوظ**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والخطوة تُقرأ من حال الطلب في المحرّك** لا من راية في الجهاز:
     * **من حفظها عنده** رأى «استلمت» وقد أُلغي الطلب من المكتب.
     *
     * **والطلب المعروض هو المفتوح** — وإن لم يُفتح شيء فأوّل ما في يده:
     * **من له طلب واحد لا يُطلب منه أن يختاره.**
     */
    fun trip(driver: LastPoint.Point?): TripState {
        val order = state.mine.firstOrNull { it.id == openId } ?: state.mine.firstOrNull()
            ?: return TripState()
        val line = route?.points.orEmpty().mapNotNull {
            if (it.size >= 2) LatLng(it[0], it[1]) else null
        }
        return TripState(
            order = order,
            // **والمسار يُفضَّل على الخطّ المستقيم** — وفارغٌ يعني أنّ
            // المحرّك لم يردّ، **فيُرسم المستقيمُ ولا تبقى الخريطةُ
            // بلا خطّ.**
            routeLine = line,
            routeM = route?.takeIf { it.available }?.distanceM ?: -1.0,
            routeSec = route?.takeIf { it.available }?.durationS ?: -1.0,
            step = TripStep.of(order.status),
            // **وما بقي يتبدّل بتبدّل الوجهة**: قبل الاستلام المسافةُ إلى
            // المتجر، وبعده طولُ المشوار إلى الزبون.
            remainingM = when (order.status) {
                "assigned", "at_pickup" -> order.toPickupM
                else -> order.legM
            },
            // **والسرعة من المحرّك لا من الشيفرة** — تُضبط للمدينة كلّها.
            avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
            requirePhoto = state.me?.requirePhoto ?: false,
            agreeOpen = agreeOpen,
            emergencyOpen = emergencyOpen,
            stops = state.mine.map { Stop(it.id, it.number) },
            // **وأوّل عرضٍ معروضٍ عليه وهو في رحلة** — وما رُفض لا يعود.
            onRouteOffer = state.offers.firstOrNull { it.id !in dismissedOffers },
            // ══════════════════════════════════════════════════════════
            // **يعرف بنفسه أنّك وصلت**
            // ══════════════════════════════════════════════════════════
            //
            // (البند الخامس في قائمة المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **ولا يُحرّك الطلب بنفسه**: قربٌ ليس وصولا — قد يمرّ من
            // الشارع، **وطلبٌ يمشي خطوةً بلا أن يضغطها صاحبه** يُفقده
            // الثقة بالتطبيق كلِّه. **فيُقترح ويُضغط.**
            nearDestination = near(driver, order),
            failReasons = detail.failReasons,
            driver = driver?.let { LatLng(it.lat, it.lng) },
            // **ونقطة المتجر قد تغيب** — متجرٌ قديمٌ بلا دبّوس:
            // **فتُرسم الرحلة بنقطتين** بدل أن تسقط الشاشة.
            pickup = order.navLat?.let { la -> order.navLng?.let { ln -> LatLng(la, ln) } },
            dropoff = LatLng(order.lat, order.lng),
            busy = detail.busy,
            error = detail.error,
        )
    }

    /**
     * **أهو على بُعد خطواتٍ من وجهته؟**
     *
     * **وثمانون مترا لا عشرة**: دقّة القمر في المدينة بين خمسة وعشرين
     * مترا، **ومن ضيّق الحدّ** لم يقترح شيئا على من يقف عند الباب.
     *
     * **والمسافة من دالّة النظام** — لا حساب مثلّثات بأيدينا: **خطأ في
     * سطر منه يزيح الوصول مئات الأمتار.**
     */
    private fun near(driver: LastPoint.Point?, order: DriverOrder): Boolean {
        if (driver == null) return false
        val lat: Double
        val lng: Double
        when (order.status) {
            "assigned" -> {
                lat = order.navLat ?: return false
                lng = order.navLng ?: return false
            }

            "on_the_way" -> {
                lat = order.lat
                lng = order.lng
            }

            else -> return false
        }
        val out = FloatArray(1)
        android.location.Location.distanceBetween(driver.lat, driver.lng, lat, lng, out)
        return out[0] <= ARRIVAL_M
    }

    /**
     * **يرسل صورة التسليم** — ثمّ يحرّك الطلب إلى «سُلّم».
     *
     * **والخطوة بعدها لا قبلها**: المحرّك يرفض «سُلّم» بلا إثبات حين
     * يُفعَّل الإعداد، **ومن حرّك أوّلا** ردّه بخطأ وهو يحمل الصورة.
     */
    fun sendProof(jpeg: ByteArray, point: LastPoint.Point?) {
        val id = currentId() ?: return
        if (detail.busy) return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.sendProof(id, jpeg, point?.lat, point?.lng)
                backend.driver.transition(id, "delivered")
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /** **يعيد الطلب إلى الطابور** — قبل أن يستلم البضاعة. */
    fun releaseCurrent() {
        openId = currentId()
        release()
    }

    // ══════════════════════════════════════════════════════════════════
    // **حديث الطلب**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغ يعني أنّه غير مفتوح** — لا حديث بلا طلب.
    var chat by mutableStateOf<ChatState?>(null)
        private set

    // ══════════════════════════════════════════════════════════════════
    // **شارةُ الحديث — كم ينتظرك فيه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «ولازم يطلع إشعار وعدّاد في حال وصلت
    //  رسالة».)
    //
    // **ورسالةٌ تصل ولا شيء يقولها** تُقرأ بعد ساعة — والزبون ينتظر
    // جوابا عن «الباب الثاني أم الأوّل؟»

    /**
     * **يقرأ مسارَ الطلب الذي في يده** — وفشلُه صامت.
     *
     * **والمحرّك يخزّنه عشرَ دقائق بمفتاح نقطتين مقرّبتين**، فنداءٌ مع
     * كلّ تحديثٍ لا يثقل عليه: **من مشى خطوةً قرأ الجوابَ المخزّن.**
     */
    private suspend fun loadRoute() {
        val id = currentId()
        if (id == null) {
            route = null
            return
        }
        runCatching { backend.driver.route(id) }
            .onSuccess { route = if (it.available) it else null }
    }

    /** **يقرأ ما ينتظره في طلبه الحاليّ** — وفشلُه صامت. */
    private suspend fun loadChatBadge() {
        val id = currentId() ?: return
        runCatching { backend.chat.threads().threads }
            .onSuccess { rows -> chatUnread = rows.firstOrNull { it.orderId == id }?.unread ?: 0 }
    }

    /** **يعيد قراءة الحديث المفتوح** — بلا وميضِ تحميلٍ ولا فقدِ ما كُتب. */
    private fun reloadChat() {
        val id = currentId() ?: return
        viewModelScope.launch {
            runCatching { backend.chat.thread(id) }.onSuccess { th ->
                chat = ChatState(
                    messages = th.messages,
                    peerName = th.peerName,
                    open = th.open,
                )
                // **وما قُرئ لا يبقى في الشارة** — فتحُ الحديث يوسمه.
                chatUnread = 0
            }
        }
    }

    fun openChat() {
        val id = currentId() ?: return
        chat = ChatState(busy = true)
        viewModelScope.launch { loadChat(id) }
    }

    fun closeChat() {
        chat = null
    }

    fun sendMessage(body: String) {
        val id = currentId() ?: return
        if (body.isBlank()) return
        viewModelScope.launch {
            try {
                backend.chat.send(id, body)
            } catch (e: Exception) {
                Log.w("RahalGo/chat", "تعذّر إرسال الرسالة", e)
            }
            loadChat(id)
        }
    }

    private suspend fun loadChat(id: String) {
        chatUnread = 0
        chat = try {
            val thread = backend.chat.thread(id)
            ChatState(
                messages = thread.messages,
                peerName = thread.peerName,
                open = thread.open,
            )
        } catch (e: Exception) {
            Log.w("RahalGo/chat", "تعذّرت قراءة الحديث", e)
            chat?.copy(busy = false) ?: ChatState(busy = false)
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **«لدي مشكلة» — وليس لكلّ حالٍ سببٌ يُختار**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **والمحرّك يقبل أسبابَ تعذّرٍ عند بابين فقط**: عند المتجر وعند
    // باب الزبون — **وهو صواب**: «الزبون غائب» لا تُقال وأنت في الطريق.
    //
    // **وما بينهما ليس بلا مشاكل**: عطلٌ في الدرّاجة، أو إيقافٌ في
    // الطريق. **فيُفتح بابُ الطارئ** — تُنبَّه العمليات ويُقرأ موضعُه.
    var emergencyOpen by mutableStateOf(false)
        private set

    fun dismissEmergency() {
        emergencyOpen = false
    }

    /** **يبلّغ العمليات** — ثمّ يُغلق النافذة ويعيد القراءة. */
    fun emergency(point: LastPoint.Point?, note: String = "") {
        val id = currentId() ?: return
        emergencyOpen = false
        viewModelScope.launch {
            runCatching { backend.driver.emergency(id, point?.lat, point?.lng, note) }
                .onFailure { Log.w("RahalGo/طارئ", "تعذّر البلاغ", it) }
            load()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **ومشكلةُ السائق فعلٌ يختلف بحسب موضعه من الرحلة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    //	قبل الاستلام  ←  يعود الطلبُ للطابور، ويأخذه سائقٌ ثانٍ في دقيقة
    //	بعد الاستلام  ←  البضاعةُ في يده، **فتُنبَّه العملياتُ ولا تُترك**
    //
    // **والسببُ يُكتب في الحالين**: «تعطّلت درّاجتي» تُقرأ في المكتب
    // ويُعرف بها لماذا تأخّر الطلب، **وإعادةٌ بلا سبب تُقرأ تردّدا.**
    fun reportProblem(reason: String, point: LastPoint.Point?) {
        val order = state.mine.firstOrNull { it.id == openId }
            ?: state.mine.firstOrNull()
            ?: return
        detail = detail.copy(failReasons = null)
        if (order.status == "assigned" || order.status == "at_pickup") {
            releaseWith(reason)
        } else {
            emergency(point, reason)
        }
    }

    private fun releaseWith(note: String) {
        val id = currentId() ?: return
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.release(id, note)
                openId = null
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    var agreeOpen by mutableStateOf(false)
        private set

    fun askAgree() {
        agreeOpen = true
    }

    fun dismissAgree() {
        agreeOpen = false
    }

    /** **يوثّق ما اتُّفق عليه** — ثمّ يعيد قراءة الطلب بسعره الجديد. */
    fun agree(goods: Long, fee: Long) {
        val id = currentId() ?: return
        agreeOpen = false
        detail = detail.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.agree(id, goods, fee)
            } catch (e: Exception) {
                detail = detail.copy(busy = false, error = describe(e))
                return@launch
            }
            detail = detail.copy(busy = false)
            reload()
        }
    }

    /**
     * **عروضٌ تركها وهو في رحلة** — فلا تعود لافتتُها.
     *
     * **وفي الذاكرة لا على القرص**: العرض يفوت خلال دقائق، **وقائمةٌ
     * تعيش بعد إقلاعٍ جديدٍ تخفي عرضاً جديدا.**
     */
    private val dismissedOffers = mutableSetOf<String>()

    fun dismissOffer() {
        state.offers.firstOrNull { it.id !in dismissedOffers }?.let { dismissedOffers += it.id }
        // **ولمسةٌ للحالة** — لتُعاد قراءة الشاشة.
        state = state.copy()
    }

    /** يعيد القراءة **ويحدّث الطلب المفتوح من القائمة نفسها.** */
    private fun reload() {
        viewModelScope.launch {
            try {
                val mine = backend.driver.orders()
                state = state.copy(
                    offers = backend.driver.queue(),
                    mine = mine,
                    loading = false,
                    error = "",
                )
                val id = openId
                if (id != null) {
                    val found = mine.firstOrNull { it.id == id }
                    // **وطلب اختفى من القائمة يُغلق** — أُلغي من المكتب
                    // أو فُكّ إسناده، **ومن أبقى شاشته** جعله يضغط خطوة
                    // على طلب لم يعد له.
                    if (found == null) openId = null else detail = detail.copy(order = found)
                }
            } catch (e: Exception) {
                state = state.copy(error = describe(e))
            }
        }
    }

    /**
     * **رفض المحرّك ليس عطبا** — والشاشة تقول سببه كما قاله.
     *
     * (رموز `driver_handlers.go`: `order_taken` · `not_on_shift` ·
     * `cash_limit_reached` · `too_many_active_orders`.)
     */
    private fun describe(e: Exception): String {
        Log.e("RahalGo/orders", "فشل نداء الطلبات", e)
        val app = getApplication<Application>()
        return when {
            e is ApiClient.ApiException -> when (e.body.code) {
                "order_taken" -> app.getString(R.string.err_order_taken)
                "offer_not_yours" -> app.getString(R.string.err_offer_not_yours)
                "not_on_shift" -> app.getString(R.string.err_not_on_shift)
                "cash_limit_reached" -> app.getString(R.string.err_cash_limit)
                "too_many_active_orders" -> app.getString(R.string.err_too_many_active)
                "not_your_order" -> app.getString(R.string.err_not_your_order)
                // **وصورة التسليم لم تُبنَ بعد** — والرسالة تقول ذلك
                // صراحة، **لا «تعذّر»** يبحث صاحبه عن سببه في الشارع.
                "delivery_proof_required" -> app.getString(R.string.err_proof_required)
                "bad_fail_reason" -> app.getString(R.string.err_bad_fail_reason)
                "unauthorized", "invalid_refresh" -> app.getString(R.string.err_invalid_refresh)
                "" -> app.getString(R.string.err_internal)
                else -> e.body.code
            }

            e is IOException || e is HttpRequestTimeoutException ->
                app.getString(R.string.err_network)

            else -> app.getString(R.string.err_unexpected) + " (" + e.javaClass.simpleName + ")"
        }
    }
}
