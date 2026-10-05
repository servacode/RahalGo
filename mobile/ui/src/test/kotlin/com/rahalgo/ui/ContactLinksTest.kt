package com.rahalgo.ui

import com.rahalgo.shared.model.Platform
import com.rahalgo.shared.model.Social
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **روابطُ قسم التواصل** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — `ContactPage.kt`
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أيقونةٌ تفتح رابطاً خاطئاً أسوأُ من غيابها** — فالروابطُ دوالُّ صافيةٌ
 * تُقاس هنا بلا جهاز.
 */
class ContactLinksTest {

    /** **C1 · الرقمُ المحلّيُّ يصير دوليّاً لـwa.me.** */
    @Test
    fun waMeConvertsLocalSyrianNumbers() {
        assertEquals("963912345678", waMeDigits("0912345678"))
        assertEquals("963912345678", waMeDigits("0912 345 678"))
        assertEquals("963912345678", waMeDigits("+963 912 345 678"))
        assertEquals("963912345678", waMeDigits("00963912345678"))
        assertEquals("963912345678", waMeDigits("912345678"))
        assertEquals("963912345678", waMeDigits("https://wa.me/963912345678"))
        assertEquals("", waMeDigits(""))
        assertEquals("", waMeDigits("  "))
    }

    /** **C2 · الحسابُ الاجتماعيّ: الفارغُ لا أيقونةَ له، والناقصُ يُكمَل.** */
    @Test
    fun socialUrlNormalises() {
        assertNull(socialUrl(""))
        assertNull(socialUrl("   "))
        assertEquals("https://facebook.com/x", socialUrl("https://facebook.com/x"))
        assertEquals("https://facebook.com/x", socialUrl("facebook.com/x"))
        assertEquals("https://t.me/rahal", socialUrl("@rahal", telegram = true))
        assertEquals("tg://resolve?domain=x", socialUrl("tg://resolve?domain=x", telegram = true))
    }

    /** **C3 · الموقعُ: «lat,lng» وحدَه يُعرض خريطة.** */
    @Test
    fun parsesLocation() {
        assertEquals(33.5138 to 36.2765, parseLatLng("33.5138,36.2765"))
        assertEquals(33.5138 to 36.2765, parseLatLng(" 33.5138 , 36.2765 "))
        assertNull(parseLatLng(""))
        assertNull(parseLatLng("33.5"))
        assertNull(parseLatLng("abc,def"))
        assertNull(parseLatLng("95,36"))
    }

    /** **C4 · رابطُ geo بدبّوسٍ واسمٍ مُرمَّز — والاسمُ الفارغُ بلا قوسين.** */
    @Test
    fun geoUriCarriesPinAndLabel() {
        val p = 33.5 to 36.25
        assertEquals("geo:33.5,36.25?q=33.5,36.25(Rahal%20Go)", geoUri(p, "Rahal Go"))
        assertEquals("geo:33.5,36.25?q=33.5,36.25", geoUri(p, " "))
        assertEquals(
            "https://www.google.com/maps/search/?api=1&query=33.5,36.25",
            mapsWebUrl(p),
        )
    }

    /** **C5 · العنوانُ بفاصلٍ واحدٍ ومسافةٍ واحدة.** */
    @Test
    fun addressIsTidied() {
        assertEquals("A، B، C", formatAddress(" A ,B\n\n C  "))
        assertEquals("A، B C", formatAddress("A،B   C"))
        assertEquals("", formatAddress("  \n "))
    }

    /** **C6 · منصّةٌ بلا وسائلَ لا يُعرض لها قسم.** */
    @Test
    fun hasContactOnlyWithSomething() {
        assertFalse(hasContact(Platform()))
        assertFalse(hasContact(Platform(location = "bad")))
        assertTrue(hasContact(Platform(supportPhone = "0912345678")))
        assertTrue(hasContact(Platform(location = "33.5,36.2")))
        assertTrue(hasContact(Platform(social = Social(telegram = "@x"))))
    }

    /**
     * **C7 · قطعةٌ واحدةٌ لصفحتين** — «تواصل معنا» و«من نحن» تناديان
     * `ContactSection` نفسَها، **والرقمُ لا يُطبع نصّاً.**
     */
    @Test
    fun oneSectionUsedTwice() {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) {
            dir = dir.parentFile ?: throw AssertionError("mobile root not found")
        }
        val ui = File(dir, "ui/src/main/kotlin/com/rahalgo/ui")
        val contact = File(ui, "ContactPage.kt").readText()
        val about = File(ui, "PlatformPage.kt").readText()
        assertTrue(contact.contains("ContactSection(p)"))
        assertTrue(about.contains("ContactSection(findUs)"))
        assertFalse(contact.contains("text = p.supportPhone"))
        assertFalse(contact.contains("value = phone"))
    }
}
