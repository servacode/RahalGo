package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رمزُ المحرّك ومفتاحُ رسالته يبلغان المستخدمَ بعربيّة** — `XG-45`
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمُفسِّرُ دالّةٌ صافية** — **فيُقاس بلا جهازٍ ولا محاكاة**،
 * ويبقى دليلُ الأجهزة لما لا يُقاس إلّا عليها.
 */
class ApiErrorsTest {
    /** **A1 · رمزٌ معروفٌ ⇒ نصُّه هو** — لا العامّة. */
    @Test
    fun knownCodeResolvesToItsOwnString() {
        assertEquals(R.string.err_auth_unavailable, resolveErrorRes("auth_unavailable"))
        assertEquals(R.string.err_out_of_zone, resolveErrorRes("out_of_zone"))
        assertEquals(R.string.err_payout_below_min, resolveErrorRes("payout_below_min"))
    }

    /** **A2 · مجهولٌ ⇒ رسالةٌ عامّةٌ لا رمزٌ خام.** */
    @Test
    fun unknownCodeFallsBackToGeneric() {
        assertEquals(R.string.err_internal, resolveErrorRes("future_unknown_code"))
        assertEquals(
            R.string.err_internal,
            resolveErrorRes("future_unknown_code", "errors.future_unknown_key"),
        )
    }

    /**
     * **A3 · «قيد المعالجة» يُقال صريحاً لا «تعذّر الاتصال»** — `CUST-DEF-002`.
     *
     * **وكان `in_progress` بلا خانةٍ فيقع على `err_internal`** («تعذر
     * الاتصال — حاول بعد قليل») — **رسالةُ فشلٍ تُغري بإعادةٍ تُنشئ طلباً
     * ثانياً والأوّلُ ما زال يُعالَج.** وأخوه `idempotency_reclaimed` كان
     * صحيحاً؛ **فليكونا سواءً على رسالة الانتظار.**
     */
    @Test
    fun inProgressResolvesToWaitNotConnectionFailure() {
        assertEquals(R.string.err_in_progress, resolveErrorRes("in_progress"))
        assertEquals(R.string.err_in_progress, resolveErrorRes("idempotency_reclaimed"))
        // **ولا يبقى «تعذّر الاتصال» جواباً لِـ«قيد المعالجة».**
        assertNotEquals(R.string.err_internal, resolveErrorRes("in_progress"))
        // **ومفتاحُ الرسالة `errors.in_progress` كذلك يحلّ إلى الانتظار.**
        assertEquals(R.string.err_in_progress, resolveErrorRes("", "errors.in_progress"))
    }

    /**
     * **A3c · «مفتاحٌ لجسمين» يُقال صريحاً لا «تعذّر الاتصال»** — `CAF-02`.
     *
     * **`idempotency_key_reused` رمزٌ جديدٌ** — عُدّلت السلّةُ ومحاولةٌ
     * سابقةٌ لم تُحسَم. **لو سقط على `err_internal` لظنّ صاحبُه عطبَ
     * اتصالٍ فأعاد** — والصوابُ أن يُوجَّه إلى «طلباتي».
     */
    @Test
    fun keyReusedResolvesToItsOwnMessage() {
        assertEquals(R.string.err_key_reused, resolveErrorRes("idempotency_key_reused"))
        assertNotEquals(R.string.err_internal, resolveErrorRes("idempotency_key_reused"))
    }

    /**
     * **A3b · رموزُ `CAF-18` تُقال بنصّها لا «تعذّر الاتصال»** — `CUST-20-019`.
     *
     * **`not_found`/`comms_closed`/`comms_no_driver` كانت تسقط على
     * `err_internal`** («تعذر الاتصال — حاول بعد قليل») — **رسالةٌ تقول
     * «اتصال» لعطبٍ ليس اتصالاً تُرسل صاحبَها يفحص شبكتَه بلا داعٍ.**
     */
    @Test
    fun caf18CodesResolveToTheirOwnMeaning() {
        assertEquals(R.string.err_not_found, resolveErrorRes("not_found"))
        assertEquals(R.string.err_comms_closed, resolveErrorRes("comms_closed"))
        assertEquals(R.string.err_comms_no_driver, resolveErrorRes("comms_no_driver"))
        // **ولا يبقى «تعذّر الاتصال» جواباً لأيٍّ منها.**
        assertNotEquals(R.string.err_internal, resolveErrorRes("not_found"))
        assertNotEquals(R.string.err_internal, resolveErrorRes("comms_closed"))
        assertNotEquals(R.string.err_internal, resolveErrorRes("comms_no_driver"))
        // **ومفاتيحُ الرسالة `errors.*` تحلّ كذلك.**
        assertEquals(R.string.err_comms_closed, resolveErrorRes("", "errors.comms_closed"))
        assertEquals(R.string.err_comms_no_driver, resolveErrorRes("", "errors.comms_no_driver"))
    }

    /** **A3 · رمزٌ غائبٌ ومفتاحُ رسالةٍ حاضر ⇒ يُقرأ المفتاح.** */
    @Test
    fun messageKeyIsConsumedWhenCodeIsMissing() {
        assertEquals(
            R.string.err_auth_unavailable,
            resolveErrorRes("", "errors.auth_unavailable"),
        )
        assertEquals(R.string.err_out_of_zone, resolveErrorRes("", "errors.out_of_zone"))
    }

    /** **وغيابُ الاثنين ⇒ العامّة.** */
    @Test
    fun emptyCodeAndKeyFallsBackToGeneric() {
        assertEquals(R.string.err_internal, resolveErrorRes("", ""))
    }

    /** **A6 · واستثناءُ الشاشة يعلو** — `validation` في المحفظة غيرُها. */
    @Test
    fun screenOverrideWins() {
        val extra = mapOf("validation" to R.string.err_invalid_amount)
        assertEquals(R.string.err_invalid_amount, resolveErrorRes("validation", "", extra))
        assertEquals(R.string.err_validation, resolveErrorRes("validation"))
    }

    /**
     * **A4 · و`503 auth_unavailable` لا تطرد صاحبَها** — `R16`.
     *
     * **قال المحرّكُ «تعذّر التحقّق» لا «رمزُك مُبطَل»** — **ومن طُرد
     * بها طُلب منه دخولٌ جديدٌ لن ينفعه.**
     */
    @Test
    fun authUnavailableDoesNotClearSession() {
        assertFalse(sessionRejected(503, "auth_unavailable"))
        assertFalse(sessionRejected(503, "internal"))
        // **والطردُ يبقى حيث كان** — رمزٌ مرفوضٌ أو تجديدٌ مُبطَل.
        assertTrue(sessionRejected(401, "unauthorized"))
        assertTrue(sessionRejected(400, "invalid_refresh"))
    }

    /** **A7 · ولا نصَّ داخليٌّ يبلغ الشاشة** — الجوابُ موردُ نصٍّ لا نصٌّ. */
    @Test
    fun resolverNeverReturnsRawServerText() {
        // **وأيُّ رمزٍ مهما بدا داخليّاً يعود موردَ نصٍّ معروفا.**
        for (raw in listOf(
            "pq: relation \"users\" does not exist",
            "redis: connection refused",
            "SQLSTATE 42P01",
        )) {
            assertEquals(R.string.err_internal, resolveErrorRes(raw))
        }
    }

    /** **وموردُ العامّة ليس هو موردَ المعروف** — وإلّا مرّ الفحصُ فارغا. */
    @Test
    fun genericDiffersFromKnown() {
        assertNotEquals(R.string.err_internal, R.string.err_auth_unavailable)
    }
}
