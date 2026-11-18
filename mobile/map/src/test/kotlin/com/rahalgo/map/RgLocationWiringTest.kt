package com.rahalgo.map

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * **كلُّ قراءةِ موقعٍ من بابٍ واحد** (بلاغُ المالك ٢٠٢٦-١١-١٨: «عندنا مشكلة بهواتف هاوواي»).
 *
 * **جوّالٌ بلا خدمات غوغل لا يردّ عليه `FusedLocationProviderClient` أبداً** — فمن ناداه مباشرةً
 * أعاد العطبَ في شاشته وحدَها. **فلا `getFusedLocationProviderClient` خارجَ `RgLocation.kt`** في
 * التطبيقات الأربعة ومكتباتها.
 */
class RgLocationWiringTest {
    private val root: File = generateSequence(File("").absoluteFile) { it.parentFile }
        .first { File(it, "settings.gradle.kts").exists() }

    private fun sources(): List<File> = root.walkTopDown()
        .onEnter { it.name != "build" && it.name != ".gradle" && it.name != "test" && it.name != "androidTest" }
        .filter { it.isFile && it.extension == "kt" }
        .toList()

    @Test
    fun noDirectFusedClientOutsideTheGate() {
        val offenders = sources()
            .filter { it.name != "RgLocation.kt" }
            .filter { f -> f.readText().contains("getFusedLocationProviderClient(") }
            .map { it.relativeTo(root).path }
        assertEquals("نداءٌ مباشرٌ لخدمة غوغل للموقع خارج RgLocation", emptyList<String>(), offenders)
    }

    @Test
    fun theFourLocationReadersUseTheGate() {
        val must = listOf(
            "map/src/main/kotlin/com/rahalgo/map/LiveFix.kt",
            "map/src/main/kotlin/com/rahalgo/map/Here.kt",
            "map/src/main/kotlin/com/rahalgo/map/PointMapCard.kt",
            "app-customer/src/main/kotlin/com/rahalgo/customer/live/LiveShareService.kt",
            "app-driver/src/main/kotlin/com/rahalgo/driver/location/LocationService.kt",
            "app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt",
            "driver-navigation/src/main/kotlin/com/rahalgo/navigation/LocationEngine.kt",
        )
        for (p in must) {
            assertTrue("$p لا يقرأ الموقع من RgLocation", File(root, p).readText().contains("RgLocation.client("))
        }
    }
}
