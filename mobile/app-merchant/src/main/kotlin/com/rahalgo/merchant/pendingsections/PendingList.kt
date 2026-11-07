package com.rahalgo.merchant.pendingsections

import com.rahalgo.shared.merchant.PendingSyncItem

/**
 * قائمة «أصناف بلا قسم» (قرار المالك ٢٠٢٦-١١-٠٧): منطق صرف بلا واجهة ولا شبكة.
 *
 * - الصفحات تتراكم بالترتيب، والصنف المكرّر (صفحتان تتقاطعان) يُحسب مرة وحدة.
 * - التحديد والقسم المختار لكل صنف يبقيان مع الصنف، ويزولان لما ينتقل.
 * - القسم الفعلي للصنف: اللي اختاره صاحب المتجر، وإلا المقترح من اسمه، وإلا فاضي.
 *
 * كل عملية ترجّع نسخة جديدة (immutable) لتلتقط الشاشة التغيير.
 */
class PendingList private constructor(
    val items: List<PendingSyncItem>,
    /** كم صنفاً ينتظر في المحرّك (مع البحث الحالي). */
    val total: Int,
    /** مفتاح الصفحة الجاية، وفارغ يعني وصلنا للآخر. */
    val next: String,
    val selected: Set<String>,
    private val choices: Map<String, String>,
) {
    val hasMore: Boolean get() = next.isNotEmpty()

    val selectedCount: Int get() = selected.size

    /** هل كل ما في الشاشة الآن محدّد؟ */
    val allSelected: Boolean get() = items.isNotEmpty() && items.all { it.id in selected }

    /** القسم اللي سينتقل إليه الصنف إذا ضغط «انقل»: اختياره، وإلا المقترح، وإلا فاضي. */
    fun sectionFor(item: PendingSyncItem): String =
        choices[item.id] ?: item.suggestedSectionId.orEmpty()

    /** صفحة جديدة بعد الحالية. */
    fun withPage(page: List<PendingSyncItem>, total: Int, next: String): PendingList {
        val have = items.mapTo(HashSet()) { it.id }
        return PendingList(items + page.filter { it.id !in have }, total, next, selected, choices)
    }

    fun choose(itemId: String, sectionId: String): PendingList =
        PendingList(items, total, next, selected, choices + (itemId to sectionId))

    fun toggle(itemId: String): PendingList =
        PendingList(items, total, next, if (itemId in selected) selected - itemId else selected + itemId, choices)

    /** حدّد كل الظاهر، وإن كان كله محدّداً أصلاً أزل التحديد. */
    fun toggleAll(): PendingList =
        PendingList(items, total, next, if (allSelected) emptySet() else items.mapTo(HashSet()) { it.id }, choices)

    fun clearSelection(): PendingList = PendingList(items, total, next, emptySet(), choices)

    /** بعد نقل ناجح: الأصناف تطلع من القائمة، ويصغر العدد بقدر ما انتقل فعلاً. */
    fun without(ids: Set<String>, moved: Int = ids.size): PendingList = PendingList(
        items.filter { it.id !in ids },
        (total - moved).coerceAtLeast(0),
        next,
        selected - ids,
        choices - ids,
    )

    companion object {
        fun empty(): PendingList = PendingList(emptyList(), 0, "", emptySet(), emptyMap())

        fun first(page: List<PendingSyncItem>, total: Int, next: String): PendingList =
            empty().withPage(page, total, next)
    }
}
