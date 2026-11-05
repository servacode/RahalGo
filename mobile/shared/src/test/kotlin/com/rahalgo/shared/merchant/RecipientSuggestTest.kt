package com.rahalgo.shared.merchant

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **اقتراحُ المستلِم وقتَ الكتابة** (قرارُ المالك ٢٠٢٦-١١-٠٥) — متى يُسأل الخادم، وما يملأ
 * النموذجَ عند الاختيار، **وردُّ الخادم يُقرأ كما يُرسَل.**
 */
class RecipientSuggestTest {

    /** **نسخةُ إعداد `ApiClient`.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    @Test
    fun searchesFromTwoCharacters() {
        assertFalse(RecipientSuggest.shouldSearch(""))
        assertFalse(RecipientSuggest.shouldSearch("أ"))
        assertFalse(RecipientSuggest.shouldSearch(" ٠ "))
        assertTrue(RecipientSuggest.shouldSearch("أح"))
        assertTrue(RecipientSuggest.shouldSearch("09"))
        assertTrue(RecipientSuggest.shouldSearch("٠٩"))
        assertEquals(300L, RecipientSuggest.DEBOUNCE_MS)
    }

    @Test
    fun fillTakesEverythingAndThePointOnlyWhenKnown() {
        val withPoint = json.decodeFromString<DeliveryRecipients>(
            """{"recipients":[{"name":"أحمد","phone":"+963933111222","address_text":"الثكنة",
               "lat":35.95,"lng":39.01,"last_used_at":"2026-11-05T10:00:00Z","times":3}]}""",
        ).recipients.single()
        val f = RecipientSuggest.fill(withPoint)
        assertEquals("أحمد", f.name)
        assertEquals("0933111222", f.phone)
        assertEquals("الثكنة", f.address)
        assertEquals(35.95 to 39.01, f.point)
        assertEquals(3, withPoint.times)

        // **بلا نقطةٍ لا نقطة** — والخادمُ يرسل `null`.
        val noPoint = json.decodeFromString<DeliveryRecipient>(
            """{"name":"سامر","phone":"+963944000111","address_text":"المشلب","lat":null,"lng":null}""",
        )
        assertNull(RecipientSuggest.fill(noPoint).point)
    }

    @Test
    fun localPhoneOnlyRewritesTheStoredSyrianForm() {
        assertEquals("0933111222", RecipientSuggest.localPhone("+963933111222"))
        assertEquals("0933111222", RecipientSuggest.localPhone("0933111222"))
        assertEquals("+90532123456", RecipientSuggest.localPhone("+90532123456"))
    }

    @Test
    fun samePhoneIgnoresFormat() {
        assertTrue(RecipientSuggest.samePhone("0933111222", "+963933111222"))
        assertTrue(RecipientSuggest.samePhone("٠٩٣٣ ١١١ ٢٢٢", "00963933111222"))
        assertFalse(RecipientSuggest.samePhone("0933111222", "0933111223"))
        assertFalse(RecipientSuggest.samePhone("", ""))
    }

    @Test
    fun savedPointDropsWhenPhoneOrAddressChanges() {
        val picked = RecipientSuggest.Fill("أحمد", "0933111222", "الثكنة", 35.95 to 39.01)
        // **نفسُه بصيغةٍ أخرى — تبقى.**
        assertTrue(RecipientSuggest.keepsSavedPoint(picked, "+963933111222", "الثكنة"))
        // **رقمٌ آخر — شخصٌ آخر — تسقط.**
        assertFalse(RecipientSuggest.keepsSavedPoint(picked, "0933111223", "الثكنة"))
        // **رقمٌ يُمحى ليُكتب غيرُه — تسقط من أوّل حرف.**
        assertFalse(RecipientSuggest.keepsSavedPoint(picked, "093311122", "الثكنة"))
        // **عنوانٌ عُدّل — مكانٌ آخر — تسقط.**
        assertFalse(RecipientSuggest.keepsSavedPoint(picked, "0933111222", "الثكنة، البناية الثانية"))
    }
}
