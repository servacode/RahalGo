package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **سباقُ الحذف — البنود ٨ إلى ١٢**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً** (البند ١١) — خمسُ حالاتٍ بأعيانها:
 *
 *	أ فعّالة  →  حذفُ أ                     →  يُرفض
 *	أ فعّالة  →  انتقالٌ إلى ب  →  حذفُ أ    →  يُرفض
 *	أ فعّالة  →  نجح ب  →  حُرّرت أ          →  يُقبل
 *	أ فعّالة  →  سقط ب  →  أ باقيةٌ محجوزة   →  يُرفض
 *	أ فعّالة  →  نجح الاتّصال  →  حُرّرت أ    →  يُقبل
 */
class MapArchiveLeaseTest {

    private val a = MapArchiveLease.Ref("raqqa", "2026-08-01", File("/tmp/a.pmtiles"))
    private val b = MapArchiveLease.Ref("raqqa", "2026-08-21", File("/tmp/b.pmtiles"))

    private fun leaseOn(ref: MapArchiveLease.Ref) = MapArchiveLease().apply {
        beginSwitch(ref)
        styleLoaded(ref)
    }

    private fun verdict(l: MapArchiveLease, r: MapArchiveLease.Ref) =
        l.deletionVerdict(r.regionId, r.dataVersion)

    // ── ١ · الفعّالةُ لا تُحذف ────────────────────────────────────

    @Test
    fun `حذفُ الفعّالة يُرفض`() {
        val l = leaseOn(a)
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Refused)
        assertEquals(a.archive, l.activeArchive())
    }

    // ── ٢ · وأثناء الانتقال يُنتظر ───────────────────────────────

    @Test
    fun `حذفُ الفعّالة أثناء انتقالٍ جارٍ يُنتظر`() {
        val l = leaseOn(a)
        l.beginSwitch(b)
        val v = verdict(l, a)
        assertTrue("$v", v is MapArchiveLease.Verdict.Wait)
        assertEquals("والفعّالةُ ما زالت أ", a.archive, l.activeArchive())
    }

    @Test
    fun `وحذفُ التي يُنتقَل إليها يُنتظر`() {
        val l = leaseOn(a)
        l.beginSwitch(b)
        assertTrue(verdict(l, b) is MapArchiveLease.Verdict.Wait)
    }

    // ── ٣ · بعد نجاح الجديدة تُحرَّر القديمة ─────────────────────

    @Test
    fun `بعد نجاح الانتقال تُحذف القديمة`() {
        val l = leaseOn(a)
        l.beginSwitch(b)
        l.styleLoaded(b)
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Allowed)
        assertTrue(verdict(l, b) is MapArchiveLease.Verdict.Refused)
        assertEquals(b.archive, l.activeArchive())
    }

    // ── ٤ · وإن سقط الانتقال تبقى القديمةُ محجوزة ────────────────

    @Test
    fun `سقوطُ الانتقال يُبقي القديمةَ محجوزة`() {
        val l = leaseOn(a)
        l.beginSwitch(b)
        l.switchFailed()

        assertTrue("أ ما زالت محميّة", verdict(l, a) is MapArchiveLease.Verdict.Refused)
        assertEquals("وهي الفعّالة", a.archive, l.activeArchive())
        assertTrue("وب صارت قابلةً للحذف", verdict(l, b) is MapArchiveLease.Verdict.Allowed)
    }

    // ── ٥ · والذهابُ إلى الاتّصال يُحرّرها ───────────────────────

    @Test
    fun `نجاحُ الانتقال إلى الاتّصال يُحرّر القديمة`() {
        val l = leaseOn(a)
        l.beginSwitchToOnline()
        // **وأثناء الانتقال ما زالت أ مفتوحةً** — فلا تُحذف.
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Refused)

        l.styleLoaded(null)
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Allowed)
        assertNull("ولا أرشيفَ مفتوح", l.activeArchive())
    }

    @Test
    fun `سقوطُ الانتقال إلى الاتّصال يُبقي القديمة`() {
        val l = leaseOn(a)
        l.beginSwitchToOnline()
        l.switchFailed()
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Refused)
        assertEquals(a.archive, l.activeArchive())
    }

    // ── وحزمةٌ لا علاقةَ لها تُحذف ───────────────────────────────

    @Test
    fun `غيرُ المحجوزة تُحذف دائماً`() {
        val l = leaseOn(a)
        assertTrue(l.deletionVerdict("aleppo", "2026-08-21") is MapArchiveLease.Verdict.Allowed)
    }

    @Test
    fun `ولا حجزَ يعني لا مانع`() {
        val l = MapArchiveLease()
        assertNull(l.activeArchive())
        assertTrue(verdict(l, a) is MapArchiveLease.Verdict.Allowed)
    }

    // ── والحمايةُ تشمل المطلوبَ والمحمَّل ────────────────────────

    @Test
    fun `اللقطةُ تُعلن الاثنين أثناء الانتقال`() {
        val l = leaseOn(a)
        l.beginSwitch(b)
        val snap = l.snapshot()
        assertTrue("انتقالٌ جارٍ", snap.switching)
        assertEquals(setOf("raqqa@2026-08-01", "raqqa@2026-08-21"), snap.pinned)
    }

    @Test
    fun `وبعد الاستقرار واحدةٌ فقط`() {
        val l = leaseOn(a)
        val snap = l.snapshot()
        assertTrue(!snap.switching)
        assertEquals(setOf("raqqa@2026-08-01"), snap.pinned)
    }
}
