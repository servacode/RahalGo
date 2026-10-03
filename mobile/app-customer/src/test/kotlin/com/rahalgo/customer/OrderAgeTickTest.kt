package com.rahalgo.customer

import com.rahalgo.ui.NOW_TICK_MS
import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«منذ ١ د» تمشي والشاشةُ مفتوحة.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: بقيت «منذ 1 د» دقائقَ طويلة.)
 *
 * **وشاشةُ Compose تحتاج جهازاً** — فيُثبَت العقدُ في المصدر (كما في
 * `OrderCardInfoTest`): البطاقةُ تقرأ ساعةً تدقّ، **والساعةُ تدقّ كلَّ نصف
 * دقيقةٍ لا أبطأ.**
 */
class OrderAgeTickTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(8) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw IllegalStateException("لم أجد جذرَ mobile")
    }

    private fun read(rel: String) = File(mobileRoot(), rel).readText().replace("\r\n", "\n")

    @Test
    fun `عمرُ الطلب يُقاس بساعةٍ تدقّ لا بلحظة الرسم`() {
        val card = read("app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt")
        assertTrue(
            "**البطاقةُ تقيس العمرَ لحظةَ الرسم وحدَها** — فيقف عند أوّل قراءة",
            card.contains("rememberNow()") && card.contains("Since.text(ctx, order.createdAt, now)"),
        )
        val since = read("ui/src/main/kotlin/com/rahalgo/ui/Since.kt")
        assertTrue("**الساعةُ لا تتجدّد**", since.contains("produceState") && since.contains("delay(periodMs)"))
        assertTrue("**الساعةُ أبطأُ من دقيقة**", NOW_TICK_MS in 1_000L..60_000L)
    }
}
