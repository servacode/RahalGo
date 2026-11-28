package com.rahalgo.customer.live

import com.rahalgo.shared.customer.PartyView

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قراراتُ «الطرف الآخر يشارك موقعَه من تطبيقنا»** (قرارُ المالك ٢٠٢٦-١١-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **دوالُّ صافيةٌ بلا جهاز** تُقاس بالاختبار: من يُستأنف، وأيُّ لافتةٍ تُعرض، ومتى تقف الخدمة.
 */
object PartyPolicy {

    /** **مفتاحُ «أوقفها بيده» لهذا الطلب** — مفصولٌ عن مفاتيح صاحب الطلب في `LiveStopped`. */
    fun stopKey(orderId: String): String = "party:$orderId"

    /** **أأمانةٌ هي؟** — وغيرُها «طلب» في اللافتة والإشعار. */
    fun isAmanah(kind: String): Boolean = kind == "amanah"

    /** **أيُّ طلبٍ تُعرض لافتتُه** — مفتوحٌ ولم يجب عنه بعد. */
    fun bannerTarget(orders: List<PartyView>): PartyView? =
        orders.firstOrNull { it.open && it.answer.isEmpty() }

    /** **بطاقةُ السؤال** — لا تُفتح لمن أجاب ولا لدورٍ انتهى. */
    fun promptTarget(orders: List<PartyView>, promptId: String?): PartyView? =
        if (promptId == null) null
        else orders.firstOrNull { it.orderId == promptId && it.open && it.answer.isEmpty() }

    /**
     * **أيُّ طلبٍ تُستأنف مشاركتُه الآن بلا لمسة** — قال «نعم» ودورُه قائم، **ولم يوقفها بيده، ولا مشاركةَ قائمة
     * له، والإذنُ ممنوحٌ** (لا يُسأل تلقائيّاً).
     */
    fun resumeTarget(
        orders: List<PartyView>,
        stopped: List<String>,
        activePartyId: String?,
        permissionGranted: Boolean,
    ): PartyView? {
        if (!permissionGranted) return null
        return orders.firstOrNull {
            it.open && it.answer == "yes" && stopKey(it.orderId) !in stopped && it.orderId != activePartyId
        }
    }

    /** **أتقف الخدمةُ على هذا الرمز؟** — الطلبُ انتهى أو ليس لي أو سُحبت الموافقة. */
    fun shouldStop(code: String?): Boolean =
        code == "live_location_inactive" || code == "not_found" || code == "party_not_consented"
}
