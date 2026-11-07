package com.rahalgo.merchant.pendingsections

import com.rahalgo.shared.merchant.PendingSyncItem
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PendingListTest {
    private fun item(id: String, suggested: String? = null) =
        PendingSyncItem(id = id, name = "item $id", suggestedSectionId = suggested)

    private fun items(vararg ids: String) = ids.map { item(it) }

    @Test
    fun pagesAccumulateInOrderWithoutDuplicates() {
        var l = PendingList.first(items("a", "b"), total = 4, next = "k1")
        assertTrue(l.hasMore)
        l = l.withPage(items("b", "c", "d"), total = 4, next = "")
        assertEquals(listOf("a", "b", "c", "d"), l.items.map { it.id })
        assertFalse(l.hasMore)
    }

    @Test
    fun sectionIsChoiceThenSuggestionThenEmpty() {
        val a = item("a", suggested = "s1")
        val b = item("b")
        var l = PendingList.first(listOf(a, b), 2, "")
        assertEquals("s1", l.sectionFor(a))
        assertEquals("", l.sectionFor(b))
        l = l.choose("a", "s2").choose("b", "s3")
        assertEquals("s2", l.sectionFor(a))
        assertEquals("s3", l.sectionFor(b))
    }

    @Test
    fun selectionTogglesAndSelectAllFlips() {
        var l = PendingList.first(items("a", "b", "c"), 3, "")
        l = l.toggle("a")
        assertEquals(1, l.selectedCount)
        assertFalse(l.allSelected)
        l = l.toggleAll()
        assertEquals(3, l.selectedCount)
        assertTrue(l.allSelected)
        l = l.toggleAll()
        assertEquals(0, l.selectedCount)
        l = l.toggle("b").toggle("b")
        assertEquals(0, l.selectedCount)
    }

    @Test
    fun movedItemsLeaveListSelectionAndChoicesAndShrinkTotal() {
        var l = PendingList.first(items("a", "b", "c"), total = 10, next = "k")
        l = l.toggle("a").toggle("b").choose("a", "s1").choose("c", "s9")
        l = l.without(setOf("a", "b"), moved = 2)
        assertEquals(listOf("c"), l.items.map { it.id })
        assertEquals(0, l.selectedCount)
        assertEquals(8, l.total)
        assertEquals("s9", l.sectionFor(l.items[0]))
        assertTrue(l.hasMore)
    }

    @Test
    fun totalNeverGoesNegative() {
        val l = PendingList.first(items("a"), total = 1, next = "").without(setOf("a"), moved = 5)
        assertEquals(0, l.total)
    }

    @Test
    fun emptyListIsNeverAllSelected() {
        val l = PendingList.empty()
        assertFalse(l.allSelected)
        assertEquals(0, l.toggleAll().selectedCount)
    }
}
