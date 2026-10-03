package com.rahalgo.shared.model

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حدودُ النصوص الحرّة — مرآةُ المحرّك** (`backend/internal/server/text_limits.go`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: `POST /orders` قبل عنواناً من عشرين ألف حرف.)
 *
 * **والمنعُ في المحرّك** (`text_too_long` بـ٤٠٠) — **وهذه الأرقامُ تمنع
 * الحقلَ أن يقبل ما سيُرفض**، فلا يكتب صاحبُه صفحةً ثمّ يُقال له «اختصر».
 *
 * **ورقمٌ يتبدّل هناك يتبدّل هنا** — بالحروف لا بالبايتات.
 */
object TextLimits {
    const val ADDRESS_TEXT = 400
    const val ORDER_NOTES = 500
    const val ITEM_NOTE = 200
    const val CUSTOM_REQUEST = 1000
    const val COMPLAINT_NOTE = 1000
    const val TICKET_REPLY = 1000
    const val ADDRESS_PART = 150
    /** **الطابقُ رقمٌ قصير.** */
    const val ADDRESS_FLOOR = 20
    /** **حدُّ الحديث** — `comms.MaxBody` في المحرّك. */
    const val CHAT_BODY = 500

    /** **يقصّ ما زاد عند الكتابة** — بالحروف (`Char`) كما يعدّها المحرّك تقريباً. */
    fun fit(text: String, max: Int): String =
        if (text.codePointCount(0, text.length) <= max) text
        else text.substring(0, text.offsetByCodePoints(0, max))
}
