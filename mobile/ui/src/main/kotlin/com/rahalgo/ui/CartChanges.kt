package com.rahalgo.ui

import android.content.Context
import com.rahalgo.shared.model.CartChange

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما تبدّل — نصٌّ واحدٌ تقرؤه السلّةُ والدفع** (`CA`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ هنا لا في شاشة
 *
 * **والسلّةُ تعرضه والدفعُ يعرضه** — **ونسختان تفترقان يوماً**:
 * **فتقول السلّةُ «تغيّر السعر» ويقول الدفعُ «حدث خطأ»**، **أو تعرض
 * إحداهما مجموعاً والأخرى غيرَه.**
 *
 * # ولا يُطبَع رمزٌ آليٌّ على شاشة
 *
 * **و`product_price_changed` نصٌّ لمهندسٍ لا لزبون** — **ومن قرأه
 * ظنّ التطبيقَ معطوباً.**
 *
 * # ورمزٌ لا نعرفه لا يُخترَع له نصّ
 *
 * **ومحرّكٌ أحدثُ من الحزمة قد يرسل نوعاً جديداً** — **فيُقال إنّ
 * شيئاً تبدّل ويُطلَب أن يُراجَع**، **ولا يُدّعى علمٌ بما لا يُعرَف.**
 */
object CartChanges {

    // أنواعُ التبدّل — **وهي نصُّ المحرّك حرفاً** (`orders/changes.go`).
    const val PRODUCT_REMOVED = "product_removed"
    const val PRODUCT_UNAVAILABLE = "product_unavailable"
    const val QUANTITY_INVALID = "quantity_invalid"
    const val PRICE_CHANGED = "product_price_changed"
    const val FEE_CHANGED = "delivery_fee_changed"
    const val PROMO_CHANGED = "promo_or_discount_changed"

    /**
     * text **التبدّلُ نصّاً عربيّاً.**
     *
     * **والاسمُ يُذكر ليُعرَف السطر** — **و«تبدّل سعرُ منتج» لا تقول
     * أيَّ منتج.**
     */
    fun text(ctx: Context, c: CartChange): String {
        val name = c.name.ifBlank { ctx.getString(R.string.cc_unnamed_item) }
        return when (c.type) {
            PRODUCT_REMOVED -> ctx.getString(R.string.cc_product_removed, name)
            PRODUCT_UNAVAILABLE -> ctx.getString(R.string.cc_product_unavailable, name)
            QUANTITY_INVALID -> ctx.getString(R.string.cc_quantity_invalid, name)
            PRICE_CHANGED -> ctx.getString(
                R.string.cc_price_changed, name, money(c.oldValue), money(c.newValue),
            )
            FEE_CHANGED -> ctx.getString(
                R.string.cc_fee_changed, money(c.oldValue), money(c.newValue),
            )
            PROMO_CHANGED ->
                // **وسقوطُه كلَّه يُقال بلفظه** — **و«تغيّر من ٣٠٠٠
                // إلى ٠» تُقرأ بصعوبة.**
                if (c.newValue == 0L) {
                    ctx.getString(R.string.cc_promo_gone)
                } else {
                    ctx.getString(
                        R.string.cc_promo_changed, money(c.oldValue), money(c.newValue),
                    )
                }
            // **ونوعٌ لا نعرفه يُقال عامّاً ولا يُطبَع رمزُه.**
            else -> ctx.getString(R.string.cc_review_title)
        }
    }

    /**
     * **أيوجب هذا التبدّلُ مراجعةً قبل الإرسال؟**
     *
     * **وكلُّ تبدّلٍ يوجبها** — **مالٌ كان أو صنفاً**: **ومن أُرسل
     * طلبُه بسعرٍ لم يره أو بلا صنفٍ طلبه لم يحصل على ما وافق عليه.**
     */
    fun needsReview(changes: List<CartChange>): Boolean = changes.isNotEmpty()

    /**
     * ══════════════════════════════════════════════════════════════════
     * **بصمةُ ما رآه — لتُعرَف الحالُ التي وافق عليها**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وليست حاجزَ أمان** — **والمحرّكُ يُعيد الحسابَ عند الإنشاء
     * ويردّ ما شاخ.** **وإنّما تعرف الشاشةُ: أهذه الحالُ التي
     * راجعها أم غيرُها؟**
     *
     * **وتُبنى ممّا رآه بعينه** — **مجموعٌ وأجورٌ وإجماليٌّ وأسطرٌ
     * بأعدادها**: **فلو تبدّل رقمٌ منها بطلت الموافقة.**
     *
     * **ولا رمزَ من الخادم لأجل الشاشة** — **ولا هجرة**: **حاجةُ
     * عرضٍ لا تُغيّر عقداً.**
     */
    fun fingerprint(
        subtotal: Long,
        deliveryFee: Long,
        total: Long,
        discount: Long,
        lines: List<Pair<String, Int>>,
        point: String,
        available: Boolean,
    ): String = buildString {
        append(point).append('|')
        append(subtotal).append('|').append(deliveryFee).append('|')
        append(total).append('|').append(discount).append('|')
        append(available).append('|')
        // **والترتيبُ يُثبَّت** — **وسطران يتبادلان موضعَهما ليسا
        // تبدّلاً في السلّة.**
        lines.sortedBy { it.first }.forEach {
            append(it.first).append(':').append(it.second).append(',')
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موافقةٌ على حالٍ بعينها — لا على ما يأتي** (`CA-11`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومن وافق على مجموعٍ ثمّ تبدّل قبل أن يضغط لم يوافق على الجديد** —
 * **فموافقةٌ تبقى صالحةً لِما لم يُعرَض بعدُ ليست موافقة.**
 */
class ReviewGate {

    /** **بصمةُ الحال التي وافق عليها** — وفارغٌ يعني لم يوافق. */
    var acknowledged: String = ""
        private set

    /** **يوافق على ما بين يديه الآن.** */
    fun accept(fingerprint: String) {
        acknowledged = fingerprint
    }

    /** **تُنسى الموافقةُ** — عند تبديل العنوان أو تفريغ السلّة. */
    fun reset() {
        acknowledged = ""
    }

    /**
     * **أيُسمَح بالإرسال؟**
     *
     * **ولا تبدّلَ ⇒ يُرسَل بلا مراجعة** — **وإنذارٌ بلا سببٍ يُعلَّم
     * عليه فيُقرأ كلُّ إنذارٍ بعده ضجيجا** (`CA-12`).
     *
     * **وتبدّلٌ ⇒ لا يُرسَل حتّى يوافق على هذه الحال بعينها.**
     */
    fun canSubmit(changes: List<CartChange>, fingerprint: String): Boolean =
        if (!CartChanges.needsReview(changes)) true else acknowledged == fingerprint
}
