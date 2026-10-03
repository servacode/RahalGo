package com.rahalgo.ui

import com.rahalgo.shared.model.ChatThread
import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الحديثُ المنتهي لا حقلَ كتابةٍ فيه.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: الرأسُ يقول «انتهى — للقراءة فقط» والحقلُ تحته
 *  يقبل الكتابة ثمّ لا يحدث شيء.)
 */
class ChatClosedTest {

    @Test
    fun `المنتهي لا يُكتب فيه والمفتوحُ يُكتب`() {
        assertFalse(chatCanWrite(ChatThread(open = false)))
        assertTrue(chatCanWrite(ChatThread(open = true)))
        // **وقبل أن يصل الرأسُ لا يُحكم بالانتهاء.**
        assertTrue(chatCanWrite(null))
    }

    /** **والشاشةُ تسأل قبل أن ترسم الحقل** — وإلّا بقي الحقلُ مهما قالت الدالّة. */
    @Test
    fun `الشاشةُ تُخفي الحقلَ في المنتهي`() {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        val src = File(dir, "ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt").readText()
        assertTrue(src.contains("if (chatCanWrite(vm.head))") && src.contains("R.string.chat_ended"))
    }
}
