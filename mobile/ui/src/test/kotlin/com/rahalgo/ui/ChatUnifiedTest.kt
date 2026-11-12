package com.rahalgo.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * **الدردشةُ واحدةٌ في كلّ التطبيقات** (قرارُ المالك ٢٠٢٦-١١-١٢) — لوحُ الزبون/المتجر ودردشةُ رحلة
 * السائق يرسمان القائمةَ وشريطَ الكتابة من `ChatConversation.kt`، والترتيبُ: صورة · حقل · مايك/إرسال.
 */
class ChatUnifiedTest {

    private fun read(rel: String): String {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        return File(dir, rel).readText().replace("\r\n", "\n")
    }

    @Test
    fun `order chat sheet draws the shared list and composer`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/OrderChat.kt")
        assertTrue(src.contains("ChatMessageList(vm.messages"))
        assertTrue(src.contains("ChatComposer("))
        assertFalse("حقلُ كتابةٍ خاصٌّ باللوح رجع", src.contains("OutlinedTextField("))
    }

    @Test
    fun `composer order is photo then field then mic or send`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/ChatConversation.kt")
        val body = src.substringAfter("fun ChatComposer(")
        val photo = body.indexOf("ChatPhotoButton(")
        val field = body.indexOf("OutlinedTextField(")
        val mic = body.indexOf("ChatMicButton(")
        val send = body.indexOf("ChatSendButton(")
        assertTrue(photo in 0 until field)
        assertTrue(field < mic && mic < send)
        assertTrue("مفتاحُ اللوحة لا يُرسل", body.contains("ImeAction.Send"))
    }

    @Test
    fun `web chat follows the same order`() {
        val src = read("../web/packages/ui/src/OrderChat.tsx")
        val form = src.substringAfter("<form")
        val photo = form.indexOf("aria-label={C.photo}")
        val field = form.indexOf("<textarea")
        val mic = form.indexOf("<IconMic")
        assertTrue("الصورةُ قبل الحقل في الموقع", photo in 0 until field)
        assertTrue("المايكُ بعد الحقل في الموقع", mic > field)
    }
}
