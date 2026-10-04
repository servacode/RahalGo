package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **السائقُ يرى «تسوية من الإدارة» ولا يرى ملاحظةَ المكتب** (قرارُ المالك
 * ٢٠٢٦-١٠-٠٤، صفحةُ التعويضات، البند ٣).
 *
 * كان كشفُ محفظته يقول «تعويض» ويعرض تحتها ملاحظةَ الموظّف («تعويض نصف الأجرة»).
 * **وتطبيقاتُ المتجر والمندوب والزبون لا تتغيّر** — الوسيطُ للسائق وحدَه.
 */
class DriverCompensationWordingTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-driver").exists()) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    @Test
    fun `تطبيق السائق يطلب اللفظ المحايد`() {
        val main = File(mobileRoot(), "app-driver/src/main/kotlin/com/rahalgo/driver/MainActivity.kt").readText()
        assertTrue(
            "تطبيقُ السائق لا يمرّر neutralCompensation = true",
            main.contains("WalletScreen(vm = walletVm, neutralCompensation = true)"),
        )
    }

    @Test
    fun `الشاشة تحول التعويض الى تسوية بلا ملاحظة`() {
        val screen = File(mobileRoot(), "ui/src/main/kotlin/com/rahalgo/ui/WalletScreen.kt").readText()
        assertTrue(
            "الشاشةُ لا تحوّل «compensation» إلى «adjustment» وتمسح الملاحظة",
            screen.contains("""if (t.kind == "compensation") t.copy(kind = "adjustment", note = "") else t"""),
        )
        val strings = File(mobileRoot(), "ui/src/main/res/values/strings.xml").readText()
        assertTrue(strings.contains("""<string name="wal_k_adjustment">تسوية من الإدارة</string>"""))
    }
}
