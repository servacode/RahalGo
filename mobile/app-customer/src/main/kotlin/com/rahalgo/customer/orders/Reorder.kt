package com.rahalgo.customer.orders

import com.rahalgo.customer.cart.Cart
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.ItemDetail
import com.rahalgo.shared.customer.MyOrder
import com.rahalgo.shared.customer.MyOrderLine

/**
 * **«اطلب مرة تانية»** (قرارُ المالك ٢٠٢٦-١١-١١) — أصنافُ الطلب المُسلَّم تعود إلى السلّة بخياراتها،
 * **من كتالوج اليوم لا من لقطة الأمس**: السعرُ والتوفّرُ يحكمهما الحاضر. **وصنفٌ ما عاد متوفراً
 * (أو اختفى أحدُ خياراته) يُترك ويُقال ذلك**، ولا يُخمَّن له بديل. التسعيرُ يبقى للمحرّك عند فتح السلّة.
 */
internal object Reorder {

    data class Result(val added: Int, val skipped: Int)

    /** سطرٌ جاهزٌ للسلّة، و`lostDrink` إن سقط مشروبُه فقط. */
    internal data class Planned(val item: com.rahalgo.shared.model.Item, val pick: Cart.Pick, val lostDrink: Boolean)

    /** يطابق سطراً قديماً على صنفٍ حاليّ — و`null` إن لم يعد صالحاً. */
    internal fun plan(line: MyOrderLine, drinkLines: List<MyOrderLine>, detail: ItemDetail): Planned? {
        if (!detail.item.available) return null
        val all = detail.modifiers.flatMap { it.options }
        val options = line.options.orEmpty().map { o -> all.firstOrNull { it.id == o.id && !o.id.isNullOrBlank() } }
        if (options.any { it == null || !it.available }) return null
        var lost = false
        val drinks = drinkLines.mapNotNull { d ->
            val opt = all.firstOrNull { it.id == d.drinkOptionId && it.available }
            if (opt == null) {
                lost = true
                null
            } else {
                Cart.DrinkPick(opt, d.qty)
            }
        }
        return Planned(detail.item, Cart.Pick(options.filterNotNull(), line.qty, drinks), lost)
    }

    suspend fun run(api: CustomerApi, order: MyOrder): Result {
        val all = order.items.orEmpty()
        val lines = all.filter { it.drinkFor == null && !it.menuItemId.isNullOrBlank() && it.qty > 0 }
        val details = HashMap<String, ItemDetail?>()
        var added = 0
        var skipped = 0
        for (l in lines) {
            val id = l.menuItemId.orEmpty()
            val detail = if (details.containsKey(id)) {
                details[id]
            } else {
                runCatching { api.itemDetail(id) }.getOrNull().also { details[id] = it }
            }
            val planned = detail?.let { plan(l, all.filter { d -> d.drinkFor == l.id }, it) }
            if (planned == null) {
                skipped++
            } else {
                Cart.add(planned.item, planned.pick.qty, planned.pick.options, planned.pick.drinks)
                added++
                if (planned.lostDrink) skipped++
            }
        }
        return Result(added, skipped)
    }
}
