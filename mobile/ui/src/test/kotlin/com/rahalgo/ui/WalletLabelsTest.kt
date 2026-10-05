package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كلُّ نوعِ حركةٍ يكتبه المحرّكُ في محفظة مستخدمٍ له اسمٌ عربيّ** —
 * يُقرأ من `backend/internal/fininv/kinds.go` لا من قائمةٍ منسوخة.
 */
class WalletLabelsTest {

    private fun repoRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "backend/internal/fininv/kinds.go").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ المستودع من " + File("").absolutePath)
    }

    /** **أنواعُ الخزينة وحدَها** — لا تقع في محفظة سائقٍ أو متجرٍ أو مندوبٍ أو زبون. */
    private val treasuryOnly = setOf("treasury_withdrawal")

    @Test
    fun `كل نوع في المحرك له اسم عربي`() {
        val src = File(repoRoot(), "backend/internal/fininv/kinds.go").readText()
        val kinds = Regex("""Kind:\s*"([a-z_]+)"""").findAll(src).map { it.groupValues[1] }.toSet()
        assertTrue("لم يُقرأ أيُّ نوع", kinds.size > 10)
        val missing = kinds.filter { it !in treasuryOnly && WalletLabels.kindRes(it) == null }
        assertEquals("أنواعٌ تُعرض بمفاتيحها الإنكليزيّة", emptyList<String>(), missing)
    }

    @Test
    fun `حالات السحب الست لكل منها اسم`() {
        assertNotNull(WalletLabels.kindRes("payout_reversal"))
        assertEquals(R.string.wal_st_processing, WalletLabels.payoutStatusRes("processing"))
        assertEquals(R.string.wal_st_failed, WalletLabels.payoutStatusRes("failed"))
        assertEquals(R.string.wal_st_reversed, WalletLabels.payoutStatusRes("reversed"))
        assertEquals(R.string.wal_st_paid, WalletLabels.payoutStatusRes("paid"))
        assertEquals(R.string.wal_st_rejected, WalletLabels.payoutStatusRes("rejected"))
        assertEquals(R.string.wal_st_pending, WalletLabels.payoutStatusRes("pending"))
        assertTrue(WalletLabels.payoutFailed("reversed"))
        assertTrue(WalletLabels.payoutFailed("failed"))
        assertTrue(WalletLabels.payoutOk("processing"))
    }
}
