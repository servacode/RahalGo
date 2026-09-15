package com.rahalgo.ui

import android.content.Context

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خبرُ التفاعل ووجهتُه — نصٌّ واحدٌ تقرؤه التطبيقات** (`AN`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ هنا لا في تطبيق
 *
 * **وأربعةُ تطبيقاتٍ تستقبل الدفعَ من قاعدةٍ واحدة** (`RahalPushService`)
 * — **ووجهةٌ تُفكّ في كلّ واحدٍ منها تفترق يوماً**: **فيفتح أحدُها
 * عرضاً ويفتح الآخرُ بيتَه بلا سبب.**
 *
 * # ولا وجهةَ لا نعرفها
 *
 * **ومحرّكٌ أحدثُ من الحزمة قد يرسل وجهةً جديدة** — **فتُفتَح الشاشةُ
 * الأولى ولا يُدَّعى علمٌ بما لا يُعرَف** (`AN-03`): **ولا يُنفَّذ
 * مقصدٌ نظاميٌّ من نصٍّ يجيء من الشبكة.**
 *
 * **ورابطٌ حرٌّ لا يُقبَل أصلاً** — **والمحرّكُ يردّه عند إنشاء
 * الحملة** (`campaigns.ValidDest`)، **وهذا حارسُه الثاني في الجيب.**
 */
object Engagement {

    /** **نوعُ خبر التفاعل** — **نصُّ المحرّك حرفاً** (`campaigns.KindPromo`). */
    const val KIND = "promo"

    /** **وخبرُ الطلب معاملةٌ** — **ولا يُخلط به.** */
    const val KIND_ORDER = "order"

    /**
     * **الوجهاتُ التي يعرف التطبيقُ كيف يفتحها.**
     *
     * **وهي نصُّ المحرّك حرفاً** (`campaigns.DestOffer` وأخواتُها).
     */
    const val DEST_OFFER = "offer"
    const val DEST_MERCHANT = "merchant"
    const val DEST_HOME = "home"

    /**
     * **الوجهةُ بعد التنقية** — **نوعٌ ومعرّف.**
     *
     * **و`Home` ليست فشلاً** — **هي الجوابُ الصحيحُ لِما لا وجهةَ له
     * ولِما لا يُعرَف.**
     */
    data class Dest(val type: String, val id: String) {
        val isHome: Boolean get() = type == DEST_HOME
    }

    val HOME = Dest(DEST_HOME, "")

    /**
     * route **يقرأ الوجهةَ من الخبر ويردّ ما يُفتَح فعلاً.**
     *
     * **ومعرّفٌ فارغٌ لوجهةٍ تحتاجه يسقط إلى البيت** — **وشاشةُ عرضٍ
     * بلا عرضٍ شاشةٌ فارغةٌ تُقرأ عطباً.**
     */
    fun route(entity: String?, entityId: String?): Dest {
        val type = entity?.trim().orEmpty()
        val id = entityId?.trim().orEmpty()
        return when (type) {
            DEST_OFFER, DEST_MERCHANT -> if (id.isEmpty()) HOME else Dest(type, id)
            else -> HOME
        }
    }

    /**
     * **لفظُ الصنف في الصندوق** — **ولا يُطبَع رمزٌ آليّ** (`AN-07`).
     *
     * **و`promo` نصٌّ لمهندسٍ لا لزبون.**
     */
    fun label(ctx: Context, kind: String): String = ctx.getString(
        when (kind) {
            KIND -> R.string.notice_kind_promo
            KIND_ORDER -> R.string.notice_kind_order
            "wallet" -> R.string.notice_kind_wallet
            "account" -> R.string.notice_kind_account
            else -> R.string.notice_kind_other
        },
    )
}
