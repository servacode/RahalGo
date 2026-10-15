package com.rahalgo.ui

import com.rahalgo.shared.net.SessionFailure
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مصنِّفُ نهاية الجلسة — ما يُخرج وما لا يُخرج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-١٠-١٦: جلسةُ المتجر أُزيحت بدخولٍ من جهازٍ ثانٍ،
 *  **فبقي الأوّلُ يعرض «مغلق» والمتجرُ مفتوح** — بلا خروجٍ ولا رسالة.)
 *
 * **والخطآن معاً ممكنان**: مصنِّفٌ ضيّقٌ يترك جلسةً ميّتةً تعرض بياناتٍ
 * قديمة، **ومصنِّفٌ واسعٌ يطرد سائقاً على عطلِ خادمٍ والطلبُ في يده.**
 */
class SessionEndTest {

    /** **SE-01 · رموزُ الموت تُخرج — بأيّ رقمِ حال.** */
    @Test
    fun endingCodesLogOut() {
        assertTrue(SessionFailure.endsSession(401, "session_superseded"))
        assertTrue(SessionFailure.endsSession(401, "invalid_refresh"))
        assertTrue(SessionFailure.endsSession(401, "unauthorized"))
        // **وحسابٌ حُظر أو أُوقف يرفض المحرّكُ تجديدَه بـ٤٠٣** — جلسةٌ لا تُجدَّد.
        assertTrue(SessionFailure.endsSession(403, "user_blocked"))
        assertTrue(SessionFailure.endsSession(403, "user_suspended"))
        assertTrue(SessionFailure.endsSession(403, "not_merchant_account"))
        // **و`invalid_refresh` بأيّ رقمٍ يبقى موتاً** — كما كان في `sessionRejected`.
        assertTrue(SessionFailure.endsSession(400, "invalid_refresh"))
        // **و٤٠١ من المحرّك بعد فشل التجديد موتٌ ولو برمزٍ لا نعرفه.**
        assertTrue(SessionFailure.endsSession(401, "something_new"))
    }

    /** **SE-02 · التعثّرُ لا يُخرج** — الجلسةُ سليمةٌ والمحرّكُ لم يقل غيرَ ذلك. */
    @Test
    fun transientFailuresDoNotLogOut() {
        assertFalse(SessionFailure.endsSession(503, "auth_unavailable"))
        assertFalse(SessionFailure.endsSession(500, "internal"))
        assertFalse(SessionFailure.endsSession(429, "rate_limited"))
        assertFalse(SessionFailure.endsSession(403, "forbidden"))
        assertFalse(SessionFailure.endsSession(403, "password_change_required"))
        assertFalse(SessionFailure.endsSession(426, "update_required"))
        assertFalse(SessionFailure.endsSession(400, "validation"))
        assertFalse(SessionFailure.endsSession(404, "not_found"))
        // **والضيفُ لا جلسةَ له تُنهى** — رمزُ الجهاز لا المحرّك.
        assertFalse(SessionFailure.endsSession(401, "not_signed_in"))
    }

    /** **SE-03 · «جهازٌ آخر» يغلب** — من أيّ الإشارتين ظهر. */
    @Test
    fun supersededReasonWins() {
        assertEquals("session_superseded", SessionFailure.reason("session_superseded", "invalid_refresh"))
        assertEquals("session_superseded", SessionFailure.reason("unauthorized", "session_superseded"))
        assertEquals("invalid_refresh", SessionFailure.reason("unauthorized", "invalid_refresh"))
        assertEquals("unauthorized", SessionFailure.reason("unauthorized", ""))
    }

    /** **SE-04 · والنصُّ نصُّ المالك** — «جهازٌ تاني» بصريحه، وما عداه «انتهت جلستك». */
    @Test
    fun messageFollowsReason() {
        assertEquals(R.string.err_session_superseded, SessionEnd.messageRes("session_superseded"))
        assertEquals(R.string.err_invalid_refresh, SessionEnd.messageRes("invalid_refresh"))
        assertEquals(R.string.err_invalid_refresh, SessionEnd.messageRes("user_blocked"))
        assertEquals(R.string.err_not_driver_account, SessionEnd.messageRes("not_driver_account"))
    }

    /** **SE-05 · ومصنِّفُ الإقلاع هو مصنِّفُ النداء** — لا شرطان يفترقان. */
    @Test
    fun restoreUsesSameClassifier() {
        assertTrue(sessionRejected(403, "user_blocked"))
        assertFalse(sessionRejected(403, "forbidden"))
        assertFalse(sessionRejected(503, "auth_unavailable"))
    }
}
