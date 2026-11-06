package com.rahalgo.customer.orders

import com.rahalgo.customer.R
import com.rahalgo.shared.customer.OrderSection
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SectionsTest {
    @Test
    fun statusMapsToLabels() {
        assertEquals(R.string.sec_pending, sectionStatusRes("pending"))
        assertEquals(R.string.sec_accepted, sectionStatusRes("accepted"))
        assertEquals(R.string.sec_picked_up, sectionStatusRes("picked_up"))
        assertEquals(R.string.sec_unavailable, sectionStatusRes("unavailable"))
        assertEquals(R.string.tik_progress, sectionStatusRes("something_new"))
    }

    @Test
    fun blockOnlyForTwoOrMoreSections() {
        assertNull(sectionsCount(null))
        assertNull(sectionsCount(emptyList()))
        assertNull(sectionsCount(listOf(OrderSection(seq = 1))))
        assertEquals(2, sectionsCount(listOf(OrderSection(seq = 1), OrderSection(seq = 2))))
    }

    @Test
    fun unavailableIsGreyed() {
        assertTrue(sectionUnavailable(OrderSection(seq = 1, status = "unavailable")))
        assertFalse(sectionUnavailable(OrderSection(seq = 1, status = "accepted")))
    }
}
