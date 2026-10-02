package com.rahalgo.driver.orders

import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ خرج من يده — يُقال له لماذا** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **قِيس حيّاً**: ألغى الزبونُ أو العملياتُ طلباً بيد سائق، أو أعادته العملياتُ
 * إلى الطابور — **فتُغلَق الملاحةُ ويقفز التطبيقُ إلى «الطلبات» بلا كلمة.**
 * فيقف في الشارع لا يعرف: أعطبٌ في هاتفه أم طلبٌ أُلغي؟
 *
 * **والمصدرُ سجلُّ الطلب لا الإشعار** (`/driver/orders/{id}/outcome`): الدفعُ قد
 * لا يصل، **وما اختفى من القائمة يُسأل عنه بعينه.** وما فعله هو (سلّم، أعاد)
 * يردّ الخادمُ عنه سبباً فارغاً — **فلا نافذةَ بما فعله للتوّ.**
 */
object Departures {

    /** **ما كان في يده ولم يعد** — بترتيب ما كان. */
    fun departed(previous: Collection<String>, now: List<DriverOrder>): List<String> {
        val here = now.mapTo(HashSet()) { it.id }
        return previous.filter { it !in here }
    }

    /** **جملةُ الرمز** — وفارغٌ لرمزٍ لا نعرفه، فتُقرأ جملةُ الخادم. */
    fun reasonRes(code: String): Int? = when (code) {
        "cancelled_customer" -> R.string.lost_cancelled_customer
        "cancelled_merchant" -> R.string.lost_cancelled_merchant
        "cancelled_ops" -> R.string.lost_cancelled_ops
        "requeued_ops" -> R.string.lost_requeued_ops
        "requeued_system" -> R.string.lost_requeued_system
        "merchant_blocked" -> R.string.lost_merchant_blocked
        "failed_ops" -> R.string.lost_failed_ops
        else -> null
    }
}

/** **طلبٌ لم يعد معه — وسببُه.** */
data class LostTrip(val orderId: String, val number: Long, val reason: String, val message: String)
