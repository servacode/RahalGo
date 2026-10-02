package com.rahalgo.driver.location

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **إقلاعُ الهاتف وسط الورديّة** (٢٠٢٦-١٠-٠٢) — كانت خدمةُ الموقع تموت والخادمُ
 * يراه «على الدوام»، ولا شيءَ يعيدها حتّى يفتح التطبيق بيده.
 */
class BootResumeTest {

    @Test
    fun `nothing for a driver who is signed out or off shift`() {
        assertEquals(BootResume.Act.NOTHING, BootResume.decide(false, true, true, true))
        assertEquals(BootResume.Act.NOTHING, BootResume.decide(true, false, true, true))
    }

    @Test
    fun `the service resumes only with always-on location — otherwise he is asked to open the app`() {
        assertEquals(BootResume.Act.START, BootResume.decide(true, true, true, true))
        assertEquals(BootResume.Act.ASK, BootResume.decide(true, true, true, false))
        assertEquals(BootResume.Act.ASK, BootResume.decide(true, true, false, false))
    }

    @Test
    fun `the receiver is declared with its permission`() {
        val manifest = File("src/main/AndroidManifest.xml").readText()
        assertTrue(manifest.contains("android.permission.RECEIVE_BOOT_COMPLETED"))
        assertTrue(manifest.contains(".location.BootReceiver"))
        assertTrue(manifest.contains("android.intent.action.BOOT_COMPLETED"))
    }
}
