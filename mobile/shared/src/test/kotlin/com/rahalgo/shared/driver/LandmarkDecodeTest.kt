package com.rahalgo.shared.driver

import com.rahalgo.shared.model.Envelope
import com.rahalgo.shared.net.Ack
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أضف معلماً» — ردُّ الخادم الجديد تفكّه نسخةُ الميدان 0.7.0 والجديدة** (مراجعة ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **0.7.0 تفكّ النجاحَ بـ`Ack`** (`86bccb2f`)، **والجديدةُ بـ`LandmarkResult`**، **والمكرَّرُ
 * ٤٠٩ تفصيلُه `Map<String,String>`** — ورقمٌ فيه يُسقط الفكَّ كلَّه فيقرأ السائقُ «تعذّر إتمام
 * الطلب». **والأجسامُ هنا كما يكتبها الخادم** (`server/map_contrib_handlers.go` و`landmark_quota.go`).
 */
class LandmarkDecodeTest {

    /** **نسخةُ الإعداد كما في `ApiClient` حرفاً.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    private val newSuccess = """
        {"data":{"id":42,"status":"pending","counted":true,"today":3,"quota":5,"day":"2026-11-05",
         "notice":"انحسب ✓ — معالمك اليوم: 3 من 5","point_from_ping":false}}
    """.trimIndent()

    @Test
    fun fieldAppDecodesTheNewSuccess() {
        val env = json.decodeFromString(Envelope.serializer(Ack.serializer()), newSuccess)
        assertTrue("0.7.0 لم تفكّ ردَّ النجاح الجديد", env.data != null && env.error == null)
    }

    @Test
    fun newAppReadsCountAndNotice() {
        val r = json.decodeFromString(Envelope.serializer(LandmarkResult.serializer()), newSuccess).data!!
        assertEquals(3, r.today)
        assertEquals(5, r.quota)
        assertTrue(r.counted)
        assertEquals("انحسب ✓ — معالمك اليوم: 3 من 5", r.notice)
        // **وخادمٌ أقدمُ يردّ `{id, status}` وحدَهما** — لا يسقط الفكّ، والنصُّ يقع على نصّ التطبيق.
        val old = json.decodeFromString(
            Envelope.serializer(LandmarkResult.serializer()),
            """{"data":{"id":7,"status":"pending"}}""",
        ).data!!
        assertEquals(0, old.quota)
        assertEquals("", old.notice)
    }

    @Test
    fun duplicateDetailsAreAllStrings() {
        val body = """
            {"error":{"code":"landmark_exists","message_key":"errors.conflict","details":{
             "notice":"«مسجد النور» موجود من قبل على الخريطة (على بعد 40 م) — ما انحسب. معالمك اليوم: 2 من 5",
             "today":"2","quota":"5","day":"2026-11-05","existing_name":"مسجد النور","distance_m":"40","source":"map"}}}
        """.trimIndent()
        val err = json.decodeFromString(Envelope.serializer(Ack.serializer()), body).error!!
        assertEquals("landmark_exists", err.code)
        assertTrue(err.details.getValue("notice").contains("موجود من قبل"))
        assertEquals("2", err.details["today"])
    }
}
