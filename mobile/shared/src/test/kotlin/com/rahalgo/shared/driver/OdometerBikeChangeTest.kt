package com.rahalgo.shared.driver

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«بدّلت الدرّاجة / العدّاد»** (قرارُ المالك ٢٠٢٦-١١-٠٦) — متى يُعرض الزرّ، وماذا يُقال بعد
 * رفض «الرقم أقلّ»، وأنّ تفصيلَ الخطأ نصوصٌ يقرؤها التطبيق (القديمُ والجديد).
 */
class OdometerBikeChangeTest {

    @Test
    fun `الزرّ قبل أيّ رفض — لمن له رقم سابق وحدَه`() {
        assertTrue(OdometerFlow.bikeChangeOffered(1250.0, false, null, emptyMap()))
        assertFalse("أوّل قراءة", OdometerFlow.bikeChangeOffered(null, false, null, emptyMap()))
        assertFalse("اختاره", OdometerFlow.bikeChangeOffered(1250.0, true, null, emptyMap()))
    }

    @Test
    fun `بعد رفض الأقلّ — حسب المحرّك`() {
        val left = mapOf("last_km" to "1250", "bike_change_left" to "true")
        val used = mapOf("last_km" to "1250", "bike_change_left" to "false")
        assertTrue(OdometerFlow.bikeChangeOffered(1250.0, false, "odometer_lower", left))
        assertFalse(OdometerFlow.bikeChangeOffered(1250.0, false, "odometer_lower", used))
        // **محرّكٌ قبل الميزة لا يرسل العلامة** — فلا زرّ يقود إلى رفضٍ ثانٍ.
        assertFalse(OdometerFlow.bikeChangeOffered(1250.0, false, "odometer_lower", mapOf("last_km" to "1250")))
    }

    @Test
    fun `ما يُقال بعد رفض الأقلّ`() {
        assertEquals(OdometerFlow.LowerHint.LowerBike, OdometerFlow.lowerHint(false, mapOf("bike_change_left" to "true")))
        assertEquals(OdometerFlow.LowerHint.BikeUsed, OdometerFlow.lowerHint(false, mapOf("bike_change_left" to "false")))
        assertEquals(OdometerFlow.LowerHint.BikeUsed, OdometerFlow.lowerHint(true, emptyMap()))
        assertEquals(OdometerFlow.LowerHint.Lower, OdometerFlow.lowerHint(false, emptyMap()))
    }

    @Test
    fun `تفصيل الخطأ نصوص — يُقرأ بلا سقوط`() {
        val json = Json { ignoreUnknownKeys = true; explicitNulls = false }
        val body = json.decodeFromString<com.rahalgo.shared.model.ApiErrorBody>(
            """{"code":"odometer_lower","message_key":"errors.odometer_lower",""" +
                """"details":{"last_km":"1250","bike_change_left":"true"}}""",
        )
        assertEquals("true", body.details["bike_change_left"])
    }

    @Test
    fun `ردّ القراءة يحمل علامة التبديل`() {
        val json = Json { ignoreUnknownKeys = true; explicitNulls = false }
        val r = json.decodeFromString<OdometerReading>("""{"id":"r1","km":400,"check":"ok","bike_change":true}""")
        assertTrue(r.bikeChange)
    }
}
