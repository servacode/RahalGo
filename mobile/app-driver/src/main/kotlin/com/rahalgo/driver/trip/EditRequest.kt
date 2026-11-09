package com.rahalgo.driver.trip

import com.rahalgo.shared.driver.DriverApi
import com.rahalgo.shared.model.DriverEditRequest
import com.rahalgo.shared.model.DriverOrder

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«طلب تعديل» بالطلب العادي** (قرار المالك ٢٠٢٦-١١-١٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «كلشي لازم يمشي على المبالغ الحقيقية». الكابتن **ما بيغيّر المصاري بإيده**: بيختار لكل سطر
 * «ناقص» (المتجر ما عطاه، ولو جزء) أو «الزبون لغاه»، وبيضيف «انزاد صنف» باسمه وكمّيته وسعره —
 * **والمكتب بيوافق أو بيرفض**، والحسبة الحقيقية بالمحرّك (`orders/order_edit_request.go`).
 *
 * والمجموع هون **تقريبي** (المجموع ناقص المشال زائد المضاف) — الخصم والتوصيل المجاني بيحسبهم
 * المحرّك، **والرقم الأكيد بيرجع بالرد** (`new_total`).
 */
enum class EditLineAction(val wire: String) { MISSING("missing"), CANCELLED("cancelled") }

data class EditLineDraft(val action: EditLineAction, val remove: Int)

data class EditAddDraft(val name: String, val qty: Int, val price: Long)

data class EditDraft(
    /** معرّف السطر ← فعله وكم قطعة بتنشال. */
    val lines: Map<String, EditLineDraft> = emptyMap(),
    val adds: List<EditAddDraft> = emptyList(),
    val note: String = "",
)

/** **من قبول الطلب لحد التسليم** — والطلب العادي بس. */
private val editRequestStages = setOf("assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")

fun canRequestEdit(order: DriverOrder): Boolean = order.kind == "standard" && order.status in editRequestStages

/** **السطور اللي بتنعدّل** — بلا المرتجع (أثره انقيّد). */
fun editableLines(order: DriverOrder) = order.items.filter { !it.returned && it.id.isNotBlank() }

/** **فرق المجموع** — ناقص المشال بسعره، زائد المضاف. */
fun editDraftDelta(order: DriverOrder, draft: EditDraft): Long {
    var delta = 0L
    for (it in editableLines(order)) {
        val d = draft.lines[it.id] ?: continue
        delta -= (it.unitPrice ?: 0L) * d.remove.coerceIn(0, it.qty)
    }
    for (a in draft.adds) delta += a.price * a.qty
    return delta
}

/** **المجموع التقريبي بعد التعديل** — وما بينزل تحت الصفر. */
fun editDraftTotal(order: DriverOrder, draft: EditDraft): Long = (order.total + editDraftDelta(order, draft)).coerceAtLeast(0)

/**
 * **أيُبعت؟** — في شي يتعدّل، وكل سطر بين ١ وكمّيته، وكل مضاف باسم وكمّية وسعر،
 * **وما بيضل الطلب بلا ولا قطعة** (المحرّك بيرفضه — «أنهِ الطلب»).
 */
fun editDraftValid(order: DriverOrder, draft: EditDraft): Boolean {
    if (draft.lines.isEmpty() && draft.adds.isEmpty()) return false
    val lines = editableLines(order)
    var left = 0
    for (it in lines) {
        val d = draft.lines[it.id]
        if (d != null && (d.remove < 1 || d.remove > it.qty)) return false
        left += it.qty - (d?.remove ?: 0)
    }
    if (draft.lines.keys.any { id -> lines.none { it.id == id } }) return false
    if (draft.adds.any { it.name.isBlank() || it.qty < 1 || it.price < 1 }) return false
    return left + draft.adds.sumOf { it.qty } > 0
}

/** **الجسم للمحرّك.** */
fun EditDraft.toBody(): DriverApi.EditRequestBody = DriverApi.EditRequestBody(
    lines = lines.map { (id, d) -> DriverApi.EditLineBody(itemId = id, action = d.action.wire, qty = d.remove) },
    adds = adds.map { DriverApi.EditAddBody(name = it.name.trim(), qty = it.qty, price = it.price) },
    note = note.trim(),
)

/** **حالة الطلب بالبطاقة** — معلّق، أو رفضه المكتب، أو صار قديم؛ وغيره ما بيبيّن. */
enum class EditRequestBadge { NONE, PENDING, REJECTED, STALE }

fun editRequestBadge(order: DriverOrder, req: DriverEditRequest?): EditRequestBadge {
    if (req == null || !canRequestEdit(order)) return EditRequestBadge.NONE
    return when (req.status) {
        "pending" -> EditRequestBadge.PENDING
        "rejected" -> EditRequestBadge.REJECTED
        "stale" -> EditRequestBadge.STALE
        else -> EditRequestBadge.NONE
    }
}

/**
 * **طلب الواتساب: تصحيح الثمن بعد الاستلام** (قرار المالك ٢٠٢٦-١١-١٠) — من «استلمت» لحد الباب،
 * زرّ ثانوي بيفتح نافذة الثمن نفسها. **وقبل الاستلام بابه خطوة «اكتب الثمن» القائمة.**
 */
fun whatsAppAmountEditable(order: DriverOrder): Boolean =
    isWhatsApp(order) && order.kind == "merchant_delivery" &&
        order.status in setOf("picked_up", "on_the_way", "at_dropoff")
