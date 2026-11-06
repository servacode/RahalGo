package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.PickupStop

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ من عدّة متاجر — محطّةٌ لكلّ متجر** (٢٠٢٦-١١-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ دالّةٍ هنا صافية** — و`stops` الغائبةُ تعني طلبَ المتجر الواحد: لا شيءَ يتغيّر.
 */

/** **أطلبٌ متعدّد المحطّات؟** */
fun isMultiStop(o: DriverOrder): Boolean = !o.stops.isNullOrEmpty()

/** **المحطّةُ حيّة؟** — المنشولةُ تُعرض مشطوبةً ولا يُذهب إليها. */
fun stopLive(s: PickupStop): Boolean = s.status != "removed"

/** **انتهى أمرُها عندي؟** — استلمتُها أنا أو انشالت. وما استلمه سائقٌ سابقٌ يبقى عليّ. */
fun stopDone(s: PickupStop): Boolean = !stopLive(s) || s.mine

/**
 * **المحطّةُ التي يُذهب إليها الآن** — `nextStop` من الخادم، وإلّا أوّلُ حيّةٍ لم أستلمها بالترتيب.
 * وفارغةٌ: لا محطّةَ بقيت (أو الطلبُ ليس متعدّداً).
 */
fun currentStop(o: DriverOrder): PickupStop? {
    val all = o.stops ?: return null
    val named = all.firstOrNull { it.merchantId == o.nextStop && stopLive(it) && !it.mine }
    return named ?: all.sortedBy { it.seq }.firstOrNull { !stopDone(it) }
}

/**
 * **الفعلُ على محطّةٍ** — «وصلت» حتّى تُسجَّل الوصولة، ثمّ «استلمت». **والمحرّكُ يرفض
 * «استلمت» قبل «وصلت» لتلك المحطّة** (`stop_not_arrived`) — فالترتيبُ هنا هو ترتيبُه.
 */
fun stopAction(s: PickupStop): NextAction? = when {
    stopDone(s) -> null
    s.arrivedAt == null -> NextAction("at_pickup", com.rahalgo.driver.R.string.step_at_pickup)
    else -> NextAction("picked_up", com.rahalgo.driver.R.string.step_picked_up)
}

/**
 * **الفعلُ الأساسيّ في طلبٍ متعدّد المحطّات** — قبل الاستلام الأخير هو فعلُ المحطّة الحاليّة،
 * و«في الطريق للزبون» بعد أن يصير الطلبُ `picked_up`. **وغيرُ المتعدّد: `null` — فيبقى `nextAction`.**
 */
fun multiStopNext(o: DriverOrder): NextAction? {
    if (!isMultiStop(o)) return null
    if (o.status != "assigned" && o.status != "at_pickup") return null
    return currentStop(o)?.let(::stopAction)
}
