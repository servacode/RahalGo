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
