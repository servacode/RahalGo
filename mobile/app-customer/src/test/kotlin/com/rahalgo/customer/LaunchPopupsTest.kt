package com.rahalgo.customer

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/** **نافذةٌ واحدةٌ في الفتحة** — الثانيةُ تنتظر الفتحةَ التالية ولا تُلغى. */
class LaunchPopupsTest {
    @Before
    fun fresh() = LaunchPopups.reset()

    @Test
    fun secondPopupWaitsForNextOpen() {
        assertTrue(LaunchPopups.claim("fix"))
        assertFalse(LaunchPopups.claim("rate"))
        // **وصاحبُ الفتحة يبقى مرسوماً عبر إعادة التركيب.**
        assertTrue(LaunchPopups.claim("fix"))
        LaunchPopups.reset()
        assertTrue(LaunchPopups.claim("rate"))
        assertEquals("rate", LaunchPopups.holderForTest())
    }
}
