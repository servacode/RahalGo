package com.rahalgo.driver.home

import android.app.Application
import com.rahalgo.ui.apiError
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.ui.Refresh
import com.rahalgo.driver.location.LocationPermission
import com.rahalgo.driver.location.LocationService
import com.rahalgo.driver.location.Readiness
import com.rahalgo.shared.driver.OdometerFlow
import com.rahalgo.shared.driver.OdometerInput
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.Notice
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حال اللوحة — والشاشة لا تملكه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ونداء واحد يملأ الشاشة كلّها** (`driver/me`): الوردية والمال واليوم.
 * **ولو قُسّم على ثلاثة نداءات** لظهرت الشاشة أثلاثا، **وكلّ ثلث قد
 * يسقط وحده.**
 */
class HomeViewModel(app: Application) : AndroidViewModel(app) {

    var state by mutableStateOf(HomeState())
        private set

    private val backend = Backend.of(getApplication())

    init {
        refresh()
        // ══════════════════════════════════════════════════════════════
        // **وتسمع نبضةَ التحديث فتُنعش نفسَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٣.)
        //
        // **رصيدُه وتقييمُه وورديّتُه وصورتُه كلُّها هنا** — والشريطُ
        // العلويُّ يقرأ منها. **فكانت تبقى كما فُتحت** حتّى يخرج من
        // التبويب ويعود.
        //
        // **و`drop(1)` تتخطّى القيمةَ الحاليّة** — وإلّا جلبت مرّتين في
        // أوّل فتحة.
        viewModelScope.launch {
            Refresh.tick.drop(1).collect { refresh() }
        }
    }

    /**
     * **قراءةٌ في الخلفية لا تُخفي مفتاحَ العمل** (بلاغُ المالك ٢٠٢٦-١٠-٠٢ بصورة: «بالعمل
     * لا يوجد الزرّ، يختفي أحياناً»). **كانت كلُّ قراءةٍ تضع `busy`** — والمفتاحُ يُستبدل
     * بدائرةٍ لا تُرى على البطاقة؛ **وصارت القراءةُ أكثرَ اليوم** (العودة · قيامُ الوصلة ·
     * كلَّ ١٥ ثانيةً إن انقطعت). **فـ`busy` لفعل السائق وحدَه**، والقراءةُ لها رايتُها.
     */
    private var refreshing = false
    private var busyAt = 0L

    /** **يغلق ملخّصَ نهاية الدوام.** */
    fun closeSummary() {
        state = state.copy(shiftSummary = null)
    }

    fun refresh() {
        if (refreshing || state.busy) return
        refreshing = true
        state = state.copy(error = "")
        viewModelScope.launch {
            state = try {
                val me = backend.driver.me()
                syncService(me)
                refreshUnread()
                // **وهدفُه يُجلب معه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣).
                //
                // **وفشلُه لا يُسقط اللوحة**: `runCatching` يُبقي ما وصل،
                // **ومفتاحُ العمل أهمُّ ما فيها.**
                val goal = runCatching { backend.me.incentives() }.getOrNull() ?: state.goal
                // **وأفضلُ أوقات الشغل أرقامٌ تتبدّل بالساعة** (٢٠٢٦-١١-١١): تُجلب أوّلَ مرّةٍ وكلَّ نصف
                // ساعة، **وفشلُها يُخفي البطاقةَ ولا يُسقط اللوحة.**
                val now = System.currentTimeMillis()
                var busy = state.busyHours
                if (busy == null || now - busyAt > BUSY_REFRESH_MS) {
                    runCatching { backend.driver.busyHours() }.getOrNull()?.let {
                        busy = it
                        busyAt = now
                    }
                }
                state.copy(me = me, goal = goal, readiness = readiness(), busyHours = busy)
            } catch (e: Exception) {
                state.copy(error = describe(e))
            } finally {
                refreshing = false
            }
        }
    }

    /** يُنادى بعد رجوع صاحبه من نافذة الإذن أو من الإعدادات. */
    fun recheckLocation() {
        state = state.copy(readiness = readiness())
        state.me?.let { syncService(it) }
    }

    // ══════════════════════════════════════════════════════════════════
    // **والإذنُ وحدَه لا يُنتج موقعاً** (`DR`، ٢٠٢٦-٠٩-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وكانت تسأل عن الإذن وحدَه** — **فالإذنُ ممنوحٌ وخدمةُ الموقع
    // مطفأةٌ حالٌ تقع كلَّ يوم**، **فيبدو جاهزاً ويفتح ورديّتَه ولا
    // موقعَ يُرسَل.** **ثمّ يُسأل: لماذا لا تصلك طلبات؟**
    //
    // **وتُقرأ الحالُ كلُّها لا رايةٌ واحدة** (`DRF-01`، ٢٠٢٦-٠٩-١٥) —
    // **فالدرجاتُ تُشتقّ منها**: **إذنٌ ناقصٌ يمنع ما يحتاجه وحدَه.**
    private fun readiness(): Readiness.State = Readiness.of(getApplication())

    /**
     * ══════════════════════════════════════════════════════════════════
     * **الخدمة تتبع الوردية — لا زرّا منفصلا**
     * ══════════════════════════════════════════════════════════════════
     *
     * **ومن جعلها زرّا ثانيا** ترك سائقا ورديّته مفتوحة وموقعه مطفأ،
     * **فيظهر للمكتب واقفا وهو يسير**، ولا يصله أقرب طلب.
     *
     * **وحال الوردية يُقرأ من المحرّك لا من الجهاز**: قد يُغلقها المكتب
     * عنه، **فتُطفأ الخدمة عند أوّل قراءة** بلا أن يفعل شيئا.
     */
    private fun syncService(me: DriverMe) {
        val app = getApplication<Application>()
        // **ويُحفظ آخرُ ما عُرف** — فإن أُعيد تشغيلُ الهاتف عادت الخدمةُ أو طُلب
        // فتحُ التطبيق (`BootResume`، ٢٠٢٦-١٠-٠٢).
        // ══════════════════════════════════════════════════════════════
        // **وما دام معه طلبٌ شغّال يرسل كلَّ ثوانٍ** (قرارُ المالك ٢٠٢٦-١٠-١٩)
        // ══════════════════════════════════════════════════════════════
        //
        // **فيتحرّك على خريطة العمليات حيّاً** — وخارجَ الطلب يعود إلى الفترة
        // العاديّة فلا تنفد بطّاريّتُه وهو ينتظر. **والطلبُ يُعرف من المحرّك**
        // (`active_orders`)، ويُعاد قراءتُه مع كلّ نبضةِ تحديث.
        val normal = me.locationPingSec.takeIf { it > 0 } ?: 20L
        val pingSec = if (me.activeOrders > 0) me.locationPingActiveSec.takeIf { it > 0 } ?: normal else normal
        com.rahalgo.driver.location.ShiftMemory.remember(app, me.onShift, pingSec)
        // **ولا يُعاد التشغيلُ إلّا إن تغيّر شيء** — كانت كلُّ نبضةِ تحديثٍ
        // تُعيد رفعَ الخدمة الأماميّة وطلبَ الموقع والفترةُ هي هي.
        if (me.onShift && LocationPermission.granted(app)) {
            if (LocationService.appliedSec != pingSec) LocationService.start(app, pingSec)
        } else {
            // **والإيقافُ يبقى دائماً** — لا يُعيد تشغيلَ شيء، ويلحق خدمةً طُلبت ولم تقم بعد.
            LocationService.stop(app)
        }
    }

    /**
     * **يفتح الوردية أو يغلقها — والجواب من المحرّك لا من الزرّ.**
     *
     * **ولا تُبدَّل الحالة قبل الردّ**: المحرّك قد يرفض الفتح (نقد فوق
     * السقف مثلا)، **ومن بدّل الزرّ متفائلا** أرى صاحبه «متاح» وهو ليس
     * كذلك — فينتظر طلبا لا يأتي، ولا شيء يقول له لماذا.
     */

    fun toggleShift(on: Boolean) {
        if (state.busy || state.odometer != null) return
        // ══════════════════════════════════════════════════════════════
        // **ولا يُعلَن متاحاً وهو لا يبلغه النداء** (`DRF-04`)
        // ══════════════════════════════════════════════════════════════
        //
        // **وقِيس ٢٠٢٦-٠٩-١٥**: **الطلبُ يُكتشَف بدفع `FCM`**،
        // **و`onMessageReceived` لا تفعل غيرَ بناءِ إشعار** — **فإذنُ
        // الإشعار مرفوضٌ ⇒ تصل الرسالةُ ويُسقطها النظامُ صامتاً**،
        // **والجوّالُ في الجيب فلا يرى شيئاً.**
        //
        // **و«متاحٌ» عند المحرّك تعني «أرسلوا إليّ عملاً»** — **ومن
        // أُعلن متاحاً ولا يبلغه النداءُ يُحسَب رافضاً وهو لا يعلم**:
        // **فيُنقص تقييمُه بعملٍ لم يرَه.**
        //
        // **والمنعُ عند الرفع وحدَه** — **والصرفُ يُقبَل على كلّ حال،
        // ولا يُحبَس سائقٌ في ورديّةٍ لأنّ إذناً ناقص.**
        if (on && !state.canGoOnline) {
            val app = getApplication<Application>()
            state = state.copy(
                error = app.getString(
                    if (state.locationOn) R.string.shift_needs_notify
                    else R.string.shift_needs_location,
                ),
            )
            return
        }
        // ══════════════════════════════════════════════════════════════
        // **وشروطُ العمل الجديدة قبل فتح الدوام** (قرارُ المالك ٢٠٢٦-١١-٠٥)
        // ══════════════════════════════════════════════════════════════
        //
        // **تُفتح اللوحةُ بدل نداءٍ يُردّ** — وبعد الموافقة يُفتح الدوامُ من نفسه.
        // **والإغلاقُ لا يُمنع**، وما بيده من طلباتٍ لا يتأثّر.
        if (on && state.me?.termsRequired == true) {
            openTerms(thenShift = true)
            return
        }
        // ══════════════════════════════════════════════════════════════
        // **وعدّادُ الدرّاجة قبل الفتح وقبل السكّر** (قرارُ المالك ٢٠٢٦-١١-٠٥)
        // ══════════════════════════════════════════════════════════════
        //
        // **والمحرّكُ يقول متى** (`odometer_mode`) — والشاشةُ تفتح الورقة. **وقراءةُ بدايةٍ
        // رُفع بعدها الفتحُ** (رقمٌ لم يُوثَّق مثلاً) تُعاد نصفَ ساعة ولا تُطلب صورةٌ ثانية.
        val me = state.me
        val step = OdometerFlow.onToggle(
            on, me?.odometerMode ?: "off", me?.odometerExempt == true, me?.activeOrders ?: 0,
        )
        if (step is OdometerFlow.Sheet) {
            val reuse = pendingStart?.takeIf { on && System.currentTimeMillis() - it.second < PENDING_MS }?.first
            if (reuse != null) {
                sendShift(on, reuse, null)
                return
            }
            state = state.copy(
                error = "",
                odometer = OdometerAsk(
                    step.phase, step.skippable, lastKm = me?.odometerLastKm,
                    bikeOffer = OdometerFlow.bikeChangeOffered(me?.odometerLastKm, false, null, emptyMap()),
                ),
            )
            return
        }
        sendShift(on, null, null)
    }

    /** **قراءةُ بدايةٍ رُفعت ولم يُفتح الدوامُ بعدها** — معرّفُها ووقتُها. */
    private var pendingStart: Pair<String, Long>? = null

    /** **ما بعد سكّر الدوام بالخروج من التطبيق** — يُكمَل حين يُسكَّر. */
    private var afterLogout: (() -> Unit)? = null

    /**
     * **يرفع صورةَ العدّاد ورقمَه ثمّ يفتح الدوام أو يسكّره بمعرّفها.**
     *
     * **والقفزةُ الكبيرة تُسأل** (`odometer_check`) فيعيدها بتأكيد، **والرقمُ الأقلّ مرفوض**
     * (`odometer_lower`) فيصوّر من جديد — والحكمان في المحرّك.
     */
    fun submitOdometer(jpeg: ByteArray?, kmText: String, confirm: Boolean) {
        val ask = state.odometer ?: return
        if (ask.busy) return
        val app = getApplication<Application>()
        if (jpeg == null) {
            state = state.copy(odometer = ask.copy(error = app.getString(R.string.odo_need_photo), confirm = null))
            return
        }
        val km = OdometerInput.normalize(kmText)
        if (km == null) {
            state = state.copy(odometer = ask.copy(error = app.getString(R.string.odo_bad_km), confirm = null))
            return
        }
        state = state.copy(odometer = ask.copy(busy = true, error = "", confirm = null))
        viewModelScope.launch {
            // **ولا موقعَ مع القراءة** — آخرُ نقطةٍ وقت الفتح من الدوام الماضي، والخصوصيّةُ لا تَعِد بها.
            val reading = try {
                backend.driver.sendOdometer(jpeg, km, ask.phase, confirm, bikeChange = ask.bikeChange)
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: ApiClient.ApiException) {
                val d = e.body.details
                state = state.copy(
                    odometer = ask.copy(
                        busy = false,
                        error = when (e.body.code) {
                            "odometer_check" -> ""
                            "odometer_lower" -> when (OdometerFlow.lowerHint(ask.bikeChange, d)) {
                                OdometerFlow.LowerHint.LowerBike ->
                                    app.getString(R.string.odo_lower_bike, d["last_km"].orEmpty())
                                OdometerFlow.LowerHint.BikeUsed -> app.getString(R.string.odo_bike_used)
                                OdometerFlow.LowerHint.Lower ->
                                    app.getString(R.string.odo_lower, d["last_km"].orEmpty())
                            }
                            else -> describe(e)
                        },
                        bikeOffer = if (e.body.code == "odometer_lower") {
                            OdometerFlow.bikeChangeOffered(ask.lastKm, false, e.body.code, d)
                        } else {
                            ask.bikeOffer
                        },
                        // **تبديلٌ رُفض** (استُعمل اليوم) — يرجع رقماً عاديّاً.
                        bikeChange = ask.bikeChange && e.body.code != "odometer_lower",
                        confirm = if (e.body.code == "odometer_check") {
                            app.getString(
                                R.string.odo_check_jump,
                                d["last_km"].orEmpty(), d["km"].orEmpty(), d["diff_km"].orEmpty(),
                            )
                        } else {
                            null
                        },
                    ),
                )
                return@launch
            } catch (e: Exception) {
                state = state.copy(odometer = ask.copy(busy = false, error = describe(e)))
                return@launch
            }
            if (ask.phase == "start") pendingStart = reading.id to System.currentTimeMillis()
            sendShift(ask.phase == "start", reading.id, if (ask.forLogout) "logout" else null)
        }
    }

    /** **«لم أستطع التصوير»** — حين يسمح المحرّك وحدَه (وضعُ «يطلب»). */
    fun skipOdometer() {
        val ask = state.odometer ?: return
        if (!ask.skippable || ask.busy) return
        state = state.copy(odometer = ask.copy(busy = true, error = ""))
        sendShift(ask.phase == "start", null, if (ask.forLogout) "logout" else null)
    }

    /** **رجوعٌ من الورقة** — والخروجُ الذي انتظرها لا يقع. */
    fun dismissOdometer() {
        if (state.odometer?.busy == true) return
        afterLogout = null
        state = state.copy(odometer = null)
    }

    /** **«بدّلت الدرّاجة / العدّاد»** (قرارُ المالك ٢٠٢٦-١١-٠٦) — يسأل أوّلاً. */
    fun askBikeChange() {
        state.odometer?.takeIf { !it.busy }?.let { state = state.copy(odometer = it.copy(bikeAsk = true)) }
    }

    /** **أكّد التبديل** — يصوّر العدّادَ الجديد ويكتب رقمَه، ويُرسل بالعلامة. */
    fun confirmBikeChange() {
        state.odometer?.let {
            state = state.copy(odometer = it.copy(bikeAsk = false, bikeChange = true, bikeOffer = false, error = ""))
        }
    }

    fun cancelBikeChange() {
        state.odometer?.let { state = state.copy(odometer = it.copy(bikeAsk = false)) }
    }

    /** **«أعدّل الرقم»** — يُغلق سؤالَ التأكيد. */
    fun fixOdometer() {
        state.odometer?.let { state = state.copy(odometer = it.copy(confirm = null)) }
    }

    fun odometerCameraDenied() {
        state.odometer?.let {
            state = state.copy(odometer = it.copy(error = getApplication<Application>().getString(R.string.odo_camera_denied)))
        }
    }

    private fun sendShift(on: Boolean, readingId: String?, via: String?) {
        state = state.copy(busy = true, error = "")
        viewModelScope.launch {
            try {
                backend.driver.setShift(on, readingId, via)
                if (on) pendingStart = null
                // **والخريطةُ تُنزَّل مع فتح الورديّة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لتعمل بلا إنترنت.
                if (on) launch { com.rahalgo.driver.trip.MapPackageWorker.ensureForTrip(getApplication()) }
                // **والطابورُ يُقرأ من جديد** — ما نزل قبل الفتح لا يصله دفعٌ.
                com.rahalgo.driver.orders.QueuePulse.bump()
                // **ثمّ يُعاد قراءة الحال كاملا** — فتح الوردية يحرّك
                // أرقاما أخرى، **ومن بدّل الراية وحدها** أبقى بقيّة
                // الشاشة على حال ما قبل الضغطة.
                val me = backend.driver.me()
                syncService(me)
                // **وإعفاءُ البطّاريّة صار خطوةَ تهيئةٍ لا سؤالَ ورديّة**
                // (قرارُ المالك ٢٠٢٦-٠٩-٢٨): يُطلَب في `SetupWizard` مرّةً عند
                // أوّل دخول، **فلا بطاقةَ «اعمل في الخلفية» على الرئيسيّة بعد اليوم.**
                // (خدمةُ الموقع أماميّةٌ فتعمل والتطبيقُ ظاهر؛ والإعفاءُ للتعافي
                // بعد قتل النظام — استرشاديٌّ لا يحجب العمل، انظر `Readiness`.)
                // **وملخّصُ الدوام عند سكّره** (قرارُ المالك ٢٠٢٦-١١-١١) — لا عند الخروج من التطبيق (`logout`)،
                // **فمن يغادر التطبيقَ لا يريد نافذةً تُفتح خلفه.** والأرقامُ من `me` بعد السكّر: يومُ حسابه ما زال قائماً.
                val summary = if (!on && via != "logout") ShiftSummary.of(me) else null
                state = state.copy(
                    me = me,
                    busy = false,
                    readiness = readiness(),
                    odometer = null,
                    shiftSummary = summary ?: state.shiftSummary,
                )
                // **والخروجُ الذي انتظر صورةَ الليل يُكمَل الآن.**
                afterLogout?.let {
                    afterLogout = null
                    it()
                }
                return@launch
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: ApiClient.ApiException) {
                // **والمحرّكُ يقولها إن لم تعرفها الشاشةُ بعد** — نسخةٌ رُفعت وهو على اللوحة.
                // وقراءةُ العدّاد المرفوعة تبقى (`pendingStart`) فلا يصوّر مرّةً ثانية.
                if (on && e.body.code == "driver_terms_required") {
                    state = state.copy(busy = false, odometer = null)
                    openTerms(thenShift = true)
                } else {
                    val msg = describe(e)
                    state = state.odometer?.let { state.copy(busy = false, odometer = it.copy(busy = false, error = msg)) }
                        ?: state.copy(busy = false, error = msg)
                }
            } catch (e: Exception) {
                val msg = describe(e)
                // **والخطأُ في الورقة إن كانت مفتوحة** — يقرؤه حيث ينظر.
                state = state.odometer?.let { state.copy(busy = false, odometer = it.copy(busy = false, error = msg)) }
                    ?: state.copy(busy = false, error = msg)
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **لوحةُ شروط العمل** (قرارُ المالك ٢٠٢٦-١١-٠٥)
    // ══════════════════════════════════════════════════════════════════

    /** **أيُفتح الدوامُ بعد الموافقة؟** — فتحها «ابدأ الدوام» لا الشريط. */
    private var shiftAfterTerms = false

    /** **يجلب الشروطَ ويفتح اللوحة.** */
    fun openTerms(thenShift: Boolean = false) {
        if (state.termsBusy) return
        shiftAfterTerms = shiftAfterTerms || thenShift
        state = state.copy(termsBusy = true, termsError = "")
        viewModelScope.launch {
            state = try {
                state.copy(terms = backend.driver.terms(), termsBusy = false)
            } catch (e: Exception) {
                state.copy(termsBusy = false, error = describe(e))
            }
        }
    }

    fun closeTerms() {
        shiftAfterTerms = false
        state = state.copy(terms = null, termsError = "")
    }

    /**
     * **«قرأت الشروط وموافق»** — بالنسخة المعروضة، **ثمّ يُفتح الدوامُ إن كان طلبه.**
     * **ونسخةٌ رُفعت وهو يقرأ تُعاد قراءتُها** (`terms_version_changed`).
     */
    fun acceptTerms() {
        val shown = state.terms ?: return
        if (state.termsBusy) return
        state = state.copy(termsBusy = true, termsError = "")
        viewModelScope.launch {
            try {
                backend.driver.acceptTerms(shown.version)
                val me = backend.driver.me()
                val resume = shiftAfterTerms
                shiftAfterTerms = false
                state = state.copy(me = me, terms = null, termsBusy = false)
                if (resume && !me.termsRequired && !me.onShift) toggleShift(true)
            } catch (e: ApiClient.ApiException) {
                val fresh = if (e.body.code == "terms_version_changed") {
                    runCatching { backend.driver.terms() }.getOrNull()
                } else {
                    null
                }
                state = state.copy(terms = fresh ?: shown, termsBusy = false, termsError = describe(e))
            } catch (e: Exception) {
                state = state.copy(termsBusy = false, termsError = describe(e))
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الشريط العلويّ — رصيدُه وإشعاراته**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وهنا لا في نموذج ثالث**: يُقرآن مع كلّ تحديث للوحة، **ونموذج
    // رابع لسطرين** يفتح نداءين ويعيش دورة حياة أخرى.
    //
    // ══════════════════════════════════════════════════════════════════
    // **والرصيد من `driver/me` لا من نداءِ محفظةٍ ثانٍ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان يُقرأ بنداءٍ ثانٍ لا يُنادى قطّ**: كُتبت `refreshTop()` ولم
    // تُوصل بـ`refresh()`، **فبقي الشريط يقول «٠ ل.س»** لسائقٍ في
    // محفظته مال. (أمسكه المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **و`driver/me` يحمل الرصيد أصلا** — ومعه التقييم. **فنداءٌ ثانٍ
    // لرقمٍ وصل** فرصةُ خلافٍ بين رقمين لا فائدةَ فيها.
    var unread by mutableStateOf(0)
        private set

    var inbox by mutableStateOf<List<Notice>?>(null)
        private set

    /** يقرأ عدد غير المقروء — **وفشله لا يُسقط اللوحة.** */
    private fun refreshUnread() {
        viewModelScope.launch {
            runCatching { backend.me.inbox(1).unread }.onSuccess { unread = it }
            // **والصندوقُ المفتوحُ يُعاد معها** (٢٠٢٦-١٠-٠٦) — كان الرقمُ يتبدّل والقائمةُ لا.
            if (inbox != null) runCatching { inbox = backend.me.inbox().items }
        }
    }

    fun openInbox() {
        viewModelScope.launch {
            inbox = runCatching { backend.me.inbox().items }.getOrDefault(emptyList())
        }
    }

    /**
     * **يُعلّم الكلَّ مقروءا** — بطلبه هو.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «لازم في تعليم الكلّ كمقروء صحّ».)
     *
     * **وكان يُعلَّم بمجرّد الفتح** — فمن فتح الصندوق ليرى العدد وخرج
     * **ضاع عنه ما لم يقرأه.** والشارةُ تقول «فيه جديد»، **ومن مسحها
     * قبل أن يُقرأ** جعلها لا تعني شيئا.
     *
     * **والمعرّفُ الفارغ يعني الكلَّ** في المحرّك — وهو ما ترسله صفحةُ
     * الويب أيضا.
     */
    fun markAllRead() {
        viewModelScope.launch {
            runCatching { backend.me.markRead() }.onSuccess {
                unread = 0
                inbox = inbox?.map { if (it.read) it else it.copy(read = true) }
            }
        }
    }

    /** **خبرٌ واحدٌ يُقرأ بالضغط عليه.** */
    fun markOneRead(n: com.rahalgo.shared.model.Notice) {
        if (n.read || n.id.isBlank()) return
        viewModelScope.launch {
            runCatching { backend.me.markRead(n.id) }.onSuccess {
                unread = (unread - 1).coerceAtLeast(0)
                inbox = inbox?.map { if (it.id == n.id) it.copy(read = true) else it }
            }
        }
    }

    fun closeInbox() {
        inbox = null
    }

    // ══════════════════════════════════════════════════════════════════
    // **الخروجُ ببوّابة — لا بضغطة** (٢٠٢٦-١٠-٠٢)
    // ══════════════════════════════════════════════════════════════════
    //
    // **قِيس**: كانت `logout()` هنا ميّتةً لا يناديها أحد، **والخروجُ من بوّابة
    // الوحدة** يمسح الجلسةَ وحدَها — **فالورديّةُ تبقى مفتوحةً في الخادم، وخدمةُ
    // الموقع تعمل بلا حساب، ومن في يده طلبٌ يخرج فيبقى الطلبُ يتيماً.**
    //
    // **فالترتيب**: طلبٌ في يده ⇐ يُمنع بكلمةٍ واضحة · وإلّا تُنهى الورديّةُ في
    // الخادم · وتقف الخدمة · **ثمّ** يُمرَّر إلى خروج الوحدة (`proceed`).

    /** **أيُخرَج الآن؟** — فلا يُضغط مرّتين. */
    var loggingOut by mutableStateOf(false)
        private set

    fun logout(proceed: () -> Unit) {
        if (loggingOut) return
        loggingOut = true
        val app = getApplication<Application>()
        viewModelScope.launch {
            try {
                // **والطلباتُ تُسأل من الخادم لا من شاشةٍ قديمة.**
                val mine = try {
                    backend.driver.orders()
                } catch (e: kotlinx.coroutines.CancellationException) {
                    throw e
                } catch (e: Exception) {
                    // **ولا يُخرَج بلا تحقّق** — ورديّةٌ تبقى مفتوحةً بلا صاحب.
                    com.rahalgo.ui.Flash.fail(app.getString(R.string.logout_check_failed, describe(e)))
                    return@launch
                }
                LogoutGate.block(mine)?.let { res ->
                    com.rahalgo.ui.Flash.fail(app.getString(res))
                    return@launch
                }
                // **والخروجُ وهو بدوامه سكّرٌ للدوام** (قرارُ المالك ٢٠٢٦-١١-٠٥) — فتُطلب صورةُ
                // العدّاد كما تُطلب عند المفتاح، **والخروجُ يُكمَل بعدها** (`afterLogout`).
                val me = state.me
                val step = OdometerFlow.onLogout(me?.onShift == true, me?.odometerMode ?: "off", 0)
                if (step is OdometerFlow.Sheet) {
                    afterLogout = { finishLogout(proceed) }
                    state = state.copy(
                        odometer = OdometerAsk(
                            step.phase, step.skippable, forLogout = true, lastKm = me?.odometerLastKm,
                            bikeOffer = OdometerFlow.bikeChangeOffered(me?.odometerLastKm, false, null, emptyMap()),
                        ),
                    )
                    return@launch
                }
                try {
                    backend.driver.setShift(false, via = "logout")
                } catch (e: kotlinx.coroutines.CancellationException) {
                    throw e
                } catch (e: ApiClient.ApiException) {
                    // **وطلبٌ أُسند في اللحظة نفسِها** — يُمنع كما مُنع أعلاه.
                    if (e.body.code == "has_active_orders") {
                        com.rahalgo.ui.Flash.fail(app.getString(R.string.logout_has_order))
                        return@launch
                    }
                    Log.w("RahalGo/logout", "تعذّر إنهاءُ الورديّة — يُكمَل الخروج", e)
                } catch (e: Exception) {
                    Log.w("RahalGo/logout", "تعذّر إنهاءُ الورديّة — يُكمَل الخروج", e)
                }
                finishLogout(proceed)
            } finally {
                loggingOut = false
            }
        }
    }

    /** **ما يبقى من الخروج بعد سكّر الدوام.** */
    private fun finishLogout(proceed: () -> Unit) {
        val app = getApplication<Application>()
        // **والخدمةُ تقف مع الخروج** — إشعارُ ورديّةٍ لحسابٍ خرج يبقى في
        // الشريط وموقعُه يُرسَل، وهو ما لا يقبله أحد.
        LocationService.stop(app)
        // **ولا تبقى رحلةٌ محفوظةٌ لحسابٍ خرج** (٢٠٢٦-١٠-٣٠) — ولا خطواتٌ تنتظر.
        com.rahalgo.driver.orders.TripStore.clear(app.filesDir)
        proceed()
    }

    /**
     * **عطب لا يُعرف لا يُسمّى «لا اتصال»** — القاعدة نفسها في الدخول:
     * الشبكة تُسمّى شبكة وحدها، **وما عداها يُسمّى باسمه ويُكتب في
     * السجلّ.**
     */
    /** **الرمزُ بعربيّة** — من الخريطة المركزيّة (`data/ApiErrors.kt`). */
    private fun describe(e: Exception): String = apiError(getApplication(), e)

    private companion object {
        /** **نصفُ ساعة** بين قراءتين لأفضل الأوقات — الخادمُ نفسُه يحفظها ساعة. */
        const val BUSY_REFRESH_MS = 30 * 60 * 1000L

        /** **قراءةُ البداية تُعاد نصفَ ساعة** — كما يقبلها المحرّك. */
        const val PENDING_MS = 29 * 60 * 1000L
    }
}
