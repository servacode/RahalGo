package com.rahalgo.merchant.pendingsections

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.merchant.AutoSortResult
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.ui.Flash
import com.rahalgo.ui.err
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * شاشة «أصناف بلا قسم» (قرار المالك ٢٠٢٦-١١-٠٧: صاحب المتجر يستعمل الجوّال وحدَه).
 * مرآة بطاقة «توزيع الأقسام» بالويب (`PriceSyncSections.tsx`): توزيع تلقائي، بحث، صفحات من ٥٠،
 * نقل صنف أو محدّدين، و«حطّ الباقي بقسم منوّعات» بعد تأكيد.
 */
class PendingSectionsState(
    private val scope: CoroutineScope,
    private val api: MerchantApi,
    private val storeId: () -> String,
    /** يُنادى عند الإغلاق ليعيد القائمة العدّ. */
    private val onClosed: () -> Unit,
) {
    var open by mutableStateOf(false)
        private set
    var loading by mutableStateOf(false)
        private set
    var loadingMore by mutableStateOf(false)
        private set

    /** سقط الجلب ولا قائمة بيدنا: يُعرض خطأ مع زر إعادة. */
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set
    var list by mutableStateOf(PendingList.empty())
        private set
    var query by mutableStateOf("")
        private set
    var selecting by mutableStateOf(false)
        private set
    var sections by mutableStateOf<List<PlatformSectionRef>>(emptyList())
        private set

    /** سطر نتيجة التوزيع التلقائي (أو النقل)، وفارغ يعني لا سطر. */
    var result by mutableStateOf<Result?>(null)
        private set
    var confirmingRest by mutableStateOf(false)
        private set

    sealed interface Result {
        data class Sorted(val r: AutoSortResult) : Result
        data class Moved(val n: Int) : Result
    }

    private var searchJob: Job? = null
    private var generation = 0

    fun show() {
        open = true
        query = ""
        selecting = false
        result = null
        confirmingRest = false
        list = PendingList.empty()
        reload()
        if (sections.isEmpty()) {
            scope.launch {
                runCatching { api.platformSections().sections }.onSuccess { sections = it }
                    .onFailure { Flash.fail(err(it)) }
            }
        }
    }

    fun close() {
        searchJob?.cancel()
        generation++
        open = false
        onClosed()
    }

    fun onQuery(q: String) {
        query = q
        searchJob?.cancel()
        searchJob = scope.launch {
            delay(SEARCH_DELAY_MS)
            reload()
        }
    }

    fun reload() {
        val gen = ++generation
        loading = true
        error = ""
        scope.launch {
            runCatching { api.priceSyncPending(storeId(), "", query, PAGE) }
                .onSuccess {
                    if (gen != generation) return@onSuccess
                    list = PendingList.first(it.items, it.total, it.next)
                    selecting = selecting && list.items.isNotEmpty()
                }
                .onFailure { if (gen == generation) error = err(it) }
            if (gen == generation) loading = false
        }
    }

    fun loadMore() {
        if (loading || loadingMore || !list.hasMore) return
        val gen = generation
        loadingMore = true
        scope.launch {
            runCatching { api.priceSyncPending(storeId(), list.next, query, PAGE) }
                .onSuccess { if (gen == generation) list = list.withPage(it.items, it.total, it.next) }
                .onFailure { Flash.fail(err(it)) }
            loadingMore = false
        }
    }

    fun autoSort() {
        if (busy) return
        busy = true
        scope.launch {
            runCatching { api.priceSyncAutoSort(storeId()) }
                .onSuccess {
                    result = Result.Sorted(it)
                    selecting = false
                    reload()
                }
                .onFailure { Flash.fail(err(it)) }
            busy = false
        }
    }

    fun choose(itemId: String, sectionId: String) {
        list = list.choose(itemId, sectionId)
    }

    fun startSelecting() {
        selecting = true
    }

    fun stopSelecting() {
        selecting = false
        list = list.clearSelection()
    }

    fun toggle(itemId: String) {
        list = list.toggle(itemId)
    }

    fun toggleAll() {
        list = list.toggleAll()
    }

    /** «انقل» على صنف واحد إلى قسمه المختار (أو المقترح). */
    fun moveOne(itemId: String) {
        val item = list.items.firstOrNull { it.id == itemId } ?: return
        val section = list.sectionFor(item)
        if (section.isEmpty()) return
        move(listOf(itemId), section)
    }

    /** «انقل المحدّد إلى قسم…». */
    fun moveSelected(sectionId: String) {
        val ids = list.selected.toList()
        if (ids.isEmpty() || sectionId.isEmpty()) return
        move(ids, sectionId)
    }

    private fun move(ids: List<String>, sectionId: String) {
        if (busy) return
        busy = true
        scope.launch {
            runCatching { api.priceSyncMove(storeId(), ids, sectionId) }
                .onSuccess {
                    list = list.without(ids.toSet(), it.moved)
                    result = Result.Moved(it.moved)
                    if (list.selected.isEmpty()) selecting = false
                    // الصفحة قصرت وفي بعد؟ كمّل التحميل.
                    if (list.items.size < PAGE && list.hasMore) loadMore()
                }
                .onFailure { Flash.fail(err(it)) }
            busy = false
        }
    }

    fun askRest() {
        confirmingRest = true
    }

    fun dismissRest() {
        confirmingRest = false
    }

    fun moveRest() {
        confirmingRest = false
        if (busy) return
        busy = true
        scope.launch {
            runCatching { api.priceSyncMoveRest(storeId()) }
                .onSuccess {
                    result = Result.Moved(it.moved)
                    selecting = false
                    reload()
                }
                .onFailure { Flash.fail(err(it)) }
            busy = false
        }
    }

    companion object {
        const val PAGE = 50
        const val SEARCH_DELAY_MS = 350L
    }
}
