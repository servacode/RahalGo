package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **ورقةُ «أين تريد التوصيل؟» لا تقول «لا عناوين» لأنّ الجلبَ سقط.**
 *
 * رُئي على المحاكي ٢٠٢٦-١٠-٠٥: انقطاعٌ عابرٌ عند الإقلاع، فبقيت الورقةُ
 * فارغةً والخادمُ فيه عنوانان — والزبونُ يُضيف عنوانَه ثانيةً.
 */
class AddressSheetRefetchTest {

    private fun src(name: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            val f = File(dir, "ui/src/main/kotlin/com/rahalgo/ui/$name")
            if (f.exists()) return f.readText()
            val g = File(dir, "src/main/kotlin/com/rahalgo/ui/$name")
            if (g.exists()) return g.readText()
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد $name")
    }

    @Test
    fun `الورقة تعيد الجلب حين تفتح وتقول السقوط`() {
        val address = src("Address.kt")
        assertTrue("الورقةُ لا تُعيد الجلب حين تُفتح", address.contains("LaunchedEffect(Unit) { vm.reloadAddresses() }"))
        assertTrue("الورقةُ لا تفرّق السقوطَ عن الفراغ", address.contains("addresses.isEmpty() && failed"))
        val vm = src("AccountViewModel.kt")
        assertTrue("السقوطُ لا يُحفظ في الحال", vm.contains("addressesFailed = failed"))
        assertTrue("لا دالّةَ لإعادة الجلب", vm.contains("fun reloadAddresses()"))
    }
}
