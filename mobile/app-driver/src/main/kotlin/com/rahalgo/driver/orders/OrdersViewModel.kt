package com.rahalgo.driver.orders

import android.app.Application
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
import com.rahalgo.driver.trip.TripState
import com.rahalgo.driver.trip.TripStep
import org.maplibre.android.geometry.LatLng
import com.rahalgo.shared.net.ApiClient
import io.ktor.client.plugins.HttpRequestTimeoutException
import java.io.IOException
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

    var state by mutableStateOf(OrdersState())
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
        backend.live.start(
            scope = viewModelScope,
            onState = { up -> Log.i("RahalGo/live", if (up) "الوصلة قامت" else "الوصلة انقطعت") },
            onEvent = { refresh() },
        )
    }

    override fun onCleared() {
        // **ولا تبقى وصلة بلا شاشة** — تستنزف البطارية وتوقظ الجهاز.
        backend.live.stop()
        super.onCleared()
    }

    fun refresh() {
        viewModelScope.launch {
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
    var openId by mutableStateOf<String?>(null)
        private set

    var detail by mutableStateOf(DetailState(order = com.rahalgo.shared.model.DriverOrder()))
        private set

    fun open(orderId: String) {
        openId = orderId
        detail = DetailState(order = state.mine.firstOrNull { it.id == orderId } ?: return)
    }

    fun close() {
        openId = null
    }

    /** يحرّك الطلب خطوة — **ثمّ يعيد قراءة كلّ شيء.** */
    fun step(to: String) {
        val id = openId ?: return
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
        val order = detail.order
        viewModelScope.launch {
            detail = try {
                detail.copy(failReasons = backend.driver.failReasons(order.status))
            } catch (e: Exception) {
                detail.copy(error = describe(e))
            }
        }
    }

    fun dismissFail() {
        detail = detail.copy(failReasons = null)
    }

    fun fail(reason: String) {
        val id = openId ?: return
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
        val id = openId ?: return
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
        return TripState(
            order = order,
            step = TripStep.of(order.status),
            // **وما بقي يتبدّل بتبدّل الوجهة**: قبل الاستلام المسافةُ إلى
            // المتجر، وبعده طولُ المشوار إلى الزبون.
            remainingM = when (order.status) {
                "assigned", "at_pickup" -> order.toPickupM
                else -> order.legM
            },
            // **والسرعة من المحرّك لا من الشيفرة** — تُضبط للمدينة كلّها.
            avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
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
