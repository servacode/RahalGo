package com.rahalgo.merchant.orders

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أيُعرض «جاهز للاستلام»؟** (٢٠٢٦-١٠-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **في وضع المتاجر القبولُ يدخل التحضيرَ ويُرسل للسائقين بنفسه** (قرارُ
 * المالك ٢٠٢٦-٠٨-٢٩) — **فلا تُرى «يُحضَّر» لحظةً**، وكان الزرُّ مربوطاً
 * بها وحدَها فلا يظهر أبداً.
 *
 * **والجاهزيّةُ وسمٌ لا حال**: المحرّكُ يقبلها من القبول حتّى وقوف السائق
 * عند الباب (`readyableStatuses` في `merchant_ops_handlers.go`)، **وتُعلَن
 * مرّةً** (`ready_at`) — فبعدها يُقال «أُعلن جاهزاً» لا زرٌّ ثانٍ.
 */
private val READYABLE = setOf("accepted", "preparing", "dispatching", "assigned", "at_pickup")

/** **الزرُّ يظهر** — حالٌ يقبلها المحرّك ولم تُعلَن الجاهزيّةُ بعد. */
internal fun canMarkReady(status: String, readyAt: String?): Boolean =
    status in READYABLE && readyAt.isNullOrBlank()

/** **أُعلنت الجاهزيّةُ والطلبُ لم يُستلَم بعد** — يُقال ذلك بدل الزرّ. */
internal fun readyAnnounced(status: String, readyAt: String?): Boolean =
    status in READYABLE && !readyAt.isNullOrBlank()

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ لا مستحقَّ فيه** (تدقيقُ ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المحرّكُ يحسب المجموعَ والعمولةَ والمستحقَّ لكلّ طلبٍ مهما كان حالُه** —
 * فطلبٌ اعتُذر عنه أو أُلغي كان يُري «المستحق لك 135» **ولن يقبض منه شيئاً**،
 * فيظنّ أنّ له عند المنصّة مالاً. **فالحسبةُ تُخفى في هذه الأحوال وحدَها.**
 */
private val NO_PAYOUT = setOf("rejected", "cancelled", "failed", "refunded")

/** **الطلبُ انتهى بلا بيع** — فلا عمولةَ ولا مستحقَّ يُعرض. */
internal fun noPayout(status: String): Boolean = status in NO_PAYOUT

/**
 * **مجموعةُ الطلب في شاشة الطلبات** — جديدٌ ينتظر قراراً، ثمّ ما يُحضَّر،
 * ثمّ ما ينتظر السائق، **ثمّ ما انتهى** (لا يصل مع `open_only` إلّا من خادمٍ أقدم).
 */
internal enum class OrderGroup { New, Preparing, Waiting, Closed }

internal fun groupOf(status: String, readyAt: String?): OrderGroup = when {
    status == "pending" -> OrderGroup.New
    status == "delivered" || noPayout(status) -> OrderGroup.Closed
    status == "accepted" || status == "preparing" || canMarkReady(status, readyAt) -> OrderGroup.Preparing
    else -> OrderGroup.Waiting
}
