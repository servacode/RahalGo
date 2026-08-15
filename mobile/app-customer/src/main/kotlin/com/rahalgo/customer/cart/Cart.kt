package com.rahalgo.customer.cart

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.model.Item

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السلّة — في الذاكرة لا على القرص**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا لا تُحفظ
 *
 * **الأسعارُ تتبدّل والمصادرُ تُغلق** — وسلّةٌ من أمسِ تحمل سعرَ أمس،
 * **فيرى رقماً ويُحاسَب بغيره.** والمحرّكُ يُعيد الحسابَ على كلّ حال،
 * **لكنّ المفاجأةَ عند الدفع تكسر الثقة.**
 *
 * **وسلّةٌ تعيش ما دام التطبيقُ مفتوحاً تكفي** — من يطلب يطلب في جلسةٍ
 * واحدة.
 *
 * # ولماذا خارجَ الشاشة
 *
 * **يُضاف من السوق ويُقرأ في السلّة ويُعرض عدُّه في الشريط** — **وثلاثُ
 * شاشاتٍ تقرأ حالاً واحدة**: ولو مُلكت لإحداهنّ لَذهبت بذهابها.
 */
object Cart {

    var lines by mutableStateOf<List<Line>>(emptyList())
        private set

    val count: Int get() = lines.sumOf { it.qty }

    /** **ما يُقرأ قبل التسعيرة** — والمحرّكُ يُعيد الحسابَ عند الإرسال. */
    val subtotal: Long get() = lines.sumOf { it.item.price * it.qty }

    /** **يُزاد صنفٌ أو يُرفع عدّه** — ولا يتكرّر السطرُ نفسُه مرّتين. */
    fun add(item: Item, qty: Int = 1) {
        val at = lines.indexOfFirst { it.item.id == item.id }
        lines = if (at >= 0) {
            lines.toMutableList().also { it[at] = it[at].copy(qty = it[at].qty + qty) }
        } else {
            lines + Line(item, qty)
        }
    }

    /** **يُنقص أو يُحذف** — والصفرُ يعني «ارفعه من السلّة». */
    fun setQty(itemId: String, qty: Int) {
        lines = if (qty <= 0) {
            lines.filter { it.item.id != itemId }
        } else {
            lines.map { if (it.item.id == itemId) it.copy(qty = qty) else it }
        }
    }

    fun clear() {
        lines = emptyList()
    }

    data class Line(val item: Item, val qty: Int)
}
