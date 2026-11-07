package com.rahalgo.customer.shop

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Item
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **The items of an open store page, page by page** (owner 2026-11-07: «ما يصير يكون بطيء»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * A mall of thousands of items made the store page one big answer; on a weak link it was slow
 * or failed. Now the page reads its header and section counts (`ShopViewModel.loadStore`), then
 * this reads the picked section 40 items at a time, and the next 40 when the grid nears its end.
 * A search inside the store asks the server (350 ms after the last keystroke) across sections.
 *
 * Cart and prices are untouched: the items are the same `Item` the market shows.
 */
class StoreList(
    private val scope: CoroutineScope,
    private val api: CustomerApi,
    private val errText: (Throwable) -> String,
) {
    private var storeId = ""

    /** **The section shown** — its id. */
    var section by mutableStateOf("")
        private set

    /** **Search inside the store** — empty: the picked section. */
    var query by mutableStateOf("")
        private set

    var items by mutableStateOf<List<Item>>(emptyList())
        private set

    var hasMore by mutableStateOf(false)
        private set

    /** **The first page of a new section / search is on its way.** */
    var loading by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    private var nextAfter = ""
    private var job: Job? = null
    private var moreJob: Job? = null
    private var searchJob: Job? = null

    /** **A store page opened** — show its first section. */
    fun start(id: String, firstSection: String) {
        storeId = id
        section = firstSection
        query = ""
        searchJob?.cancel()
        reload()
    }

    fun clear() {
        job?.cancel()
        moreJob?.cancel()
        searchJob?.cancel()
        storeId = ""
        section = ""
        query = ""
        items = emptyList()
        hasMore = false
        loading = false
        error = ""
    }

    fun pick(sectionId: String) {
        if (sectionId == section && query.isBlank()) return
        section = sectionId
        query = ""
        searchJob?.cancel()
        reload()
    }

    fun search(text: String) {
        val t = text.take(60)
        if (t == query) return
        query = t
        searchJob?.cancel()
        searchJob = scope.launch {
            delay(SEARCH_DEBOUNCE_MS)
            reload()
        }
    }

    private suspend fun fetch(after: String) = api.storeItems(
        storeId,
        section = if (query.isBlank()) section else "",
        query = query,
        after = after,
        limit = PAGE_SIZE,
    )

    fun reload() {
        if (storeId.isEmpty()) return
        job?.cancel()
        moreJob?.cancel()
        items = emptyList()
        hasMore = false
        error = ""
        loading = true
        val id = storeId
        job = scope.launch {
            runCatching { fetch("") }
                .onSuccess { page ->
                    if (id != storeId) return@onSuccess
                    items = page.items
                    nextAfter = page.nextAfter
                    hasMore = page.nextAfter.isNotEmpty()
                }
                .onFailure { if (it !is kotlinx.coroutines.CancellationException && id == storeId) error = errText(it) }
            loading = false
        }
    }

    /** **The grid neared its end** — the next page, one at a time. */
    fun loadMore() {
        if (!hasMore || nextAfter.isEmpty() || moreJob?.isActive == true || job?.isActive == true) return
        val after = nextAfter
        val id = storeId
        moreJob = scope.launch {
            runCatching { fetch(after) }
                .onSuccess { page ->
                    if (id != storeId || after != nextAfter) return@onSuccess
                    val have = items.mapTo(HashSet()) { it.id }
                    items = items + page.items.filter { it.id !in have }
                    nextAfter = page.nextAfter
                    hasMore = page.nextAfter.isNotEmpty()
                }
                .onFailure { if (it !is kotlinx.coroutines.CancellationException && id == storeId) error = errText(it) }
        }
    }

    private companion object {
        const val PAGE_SIZE = 40
        const val SEARCH_DEBOUNCE_MS = 350L
    }
}
