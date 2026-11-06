package com.rahalgo.merchant.quickphoto

import com.rahalgo.shared.merchant.NoPhotoItem

/**
 * Fast photo mode queue (owner decision 2026-11-06): the items without a photo in a stable order.
 *
 * - [of] puts items skipped in this session after the untouched ones, and drops items whose photo
 *   is already queued for upload, so coming back to the screen resumes at the next item that still
 *   needs a photo.
 * - [advance] returns the queue one item further after a save or a skip.
 */
class QuickPhotoQueue private constructor(
    private val list: List<NoPhotoItem>,
    private val cursor: Int = 0,
) {

    /** How many items this run covers (the counter's second number). */
    val total: Int get() = list.size

    val current: NoPhotoItem? get() = list.getOrNull(cursor)

    /** 1-based position of the current item (the counter's first number). */
    val position: Int get() = if (list.isEmpty()) 0 else minOf(cursor + 1, list.size)

    val finished: Boolean get() = cursor >= list.size

    /** The id of the item that follows the current one, to start preparing its screen. */
    val nextId: String? get() = list.getOrNull(cursor + 1)?.id

    /** A new queue one item further (immutable, so the screen state notices the change). */
    fun advance(): QuickPhotoQueue = if (cursor < list.size) QuickPhotoQueue(list, cursor + 1) else this

    companion object {
        fun of(
            items: List<NoPhotoItem>,
            skipped: Set<String> = emptySet(),
            queuedForUpload: Set<String> = emptySet(),
        ): QuickPhotoQueue {
            val open = items.filter { it.id !in queuedForUpload }
            val (later, first) = open.partition { it.id in skipped }
            return QuickPhotoQueue(first + later)
        }
    }
}
