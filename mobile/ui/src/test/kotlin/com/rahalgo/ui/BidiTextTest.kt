package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * عزلُ الأسماء اللاتينيّة بأرقامها (ملاحظةُ المالك ٢٠٢٦-١١-١١): «QA-Money-Model2» تظهر «2QA-Money-Model».
 * **المقطعُ اللاتينيُّ كاملاً يُعزل لا رقمُه وحدَه.**
 */
class BidiTextTest {
    private val fsi = "\u2068"
    private val pdi = "\u2069"

    @Test
    fun `الاسم اللاتيني برقمه يُعزل كاملاً`() {
        val out = BidiText.isolateNumbers("انضمّ QA-Money-Model2 إلى العائلة")
        assertEquals("انضمّ ${fsi}QA-Money-Model2$pdi إلى العائلة", out)
    }

    @Test
    fun `اسم لاتيني بكلمات ورقم يُعزل واحداً`() {
        val out = BidiText.isolateNumbers("طلب من Cafe Roma 2 جاهز")
        assertEquals("طلب من ${fsi}Cafe Roma 2$pdi جاهز", out)
    }

    @Test
    fun `لاتيني بلا رقم لا يُمسّ`() {
        assertEquals("طلب من Cafe Roma جاهز", BidiText.isolateNumbers("طلب من Cafe Roma جاهز"))
    }

    @Test
    fun `الأرقام والمبالغ العاديّة تبقى معزولة كما كانت`() {
        assertEquals(
            "طلب ${fsi}#1201$pdi – ${fsi}+100 ل.س$pdi",
            BidiText.isolateNumbers("طلب #1201 – +100 ل.س"),
        )
    }

    @Test
    fun `النص الفارغ يبقى فارغاً`() {
        assertEquals("", BidiText.isolateNumbers(""))
    }
}
