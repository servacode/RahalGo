package com.rahalgo.customer.chat

import com.rahalgo.customer.chat.LiveChatViewModel.Live
import org.junit.Assert.assertEquals
import org.junit.Test

class LiveChatBadgeTest {
    @Test
    fun `badge is the sum of unread across listed open orders`() {
        val chats = listOf(Live("a", 2, 1181), Live("b", 1, 1182))
        assertEquals(3, LiveChatViewModel.totalUnread(chats))
    }

    @Test
    fun `single order and empty list`() {
        assertEquals(4, LiveChatViewModel.totalUnread(listOf(Live("a", 4, 1))))
        assertEquals(0, LiveChatViewModel.totalUnread(emptyList()))
    }
}
