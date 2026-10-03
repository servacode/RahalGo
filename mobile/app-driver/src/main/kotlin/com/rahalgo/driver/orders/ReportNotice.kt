package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«وصل بلاغك للإدارة» يخصّ وصولاً بعينه** (تجربةُ القبول ٢٠٢٦-١٠-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **قِيس مرّتين على المحاكي**: «المتجر مغلق» عند الأوّل ← حوّلته الإدارةُ لمتجرٍ آخر ← وصل
 * الجديد ← **فعاد الطلبُ «وصلت المتجر» كما كان، فطابق المفتاحُ القديم** (`id/status`)
 * وبقي «وصل بلاغك» **وخُبّئ «استلمت الطلب»** عند متجرٍ لم يُبلَّغ عنه قطّ.
 *
 * **فالمفتاحُ يحمل المتجرَ أيضاً**، **وأيُّ قراءةٍ ترى الطلبَ في غير مفتاحه تمحو الخبر** —
 * فرجوعُه إلى الطريق بعد التحويل يمحوه ولو عاد إلى «وصلت المتجر» بعدها.
 */
internal object ReportNotice {

    /** **مفتاحُ الخبر**: الطلبُ ومرحلتُه ومتجرُه. */
    fun key(o: DriverOrder): String = o.id + "/" + o.status + "/" + o.merchantName

    /** **أشاخ الخبرُ؟** — طلبُه خرج من يده، أو رُئي في غير مرحلته أو متجره. */
    fun stale(noticeFor: String, mine: List<DriverOrder>): Boolean {
        if (noticeFor.isEmpty()) return false
        val id = noticeFor.substringBefore('/')
        val o = mine.firstOrNull { it.id == id } ?: return true
        return key(o) != noticeFor
    }
}
