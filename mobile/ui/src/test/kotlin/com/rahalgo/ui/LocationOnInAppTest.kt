package com.rahalgo.ui

import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«الموقع مطفأ» يُحلّ داخل التطبيق** (قرارُ المالك ٢٠٢٦-١٠-٢١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «وقت الزبون يحدد العنوان مايكون مشغل الموقع فيشغل الموقع وهو بالتطبيق — مالازم
 * يسكر التطبيق ويفتحه». **ثلاثةُ أبواب**: نافذةُ غوغل «موافق» من الزرّ، وبثُّ النظام
 * حين يُشغَّل من الشريط العلويّ، والعودةُ من الإعدادات. **ومشاركةُ الموقع المباشر**
 * تُشغّله قبل أن تبدأ.
 *
 * **حراسةُ مصدرٍ** كـ`CustDef006Test` — النافذةُ والبثُّ يحتاجان جهازاً.
 */
class LocationOnInAppTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val enabler = "map/src/main/kotlin/com/rahalgo/map/LocationEnabler.kt"
    private val pick = "map/src/main/kotlin/com/rahalgo/map/PickPoint.kt"
    private val main = "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
    private val trip = "app-driver/src/main/kotlin/com/rahalgo/driver/trip/TripScreen.kt"

    @Test
    fun enablerUsesGoogleDialogAndListensToTheSwitch() {
        val s = read(enabler)
        assertTrue("**لا نافذةَ غوغل** — `ResolvableApiException`", s.contains("ResolvableApiException"))
        assertTrue("**لا بثَّ لمفتاح الموقع**", s.contains("PROVIDERS_CHANGED_ACTION"))
        assertTrue("**ولا رجوعَ للإعدادات إن غابت النافذة**", s.contains("openLocationSettings"))
    }

    @Test
    fun pickPointRetriesWhenLocationTurnsOn() {
        val s = read(pick)
        assertTrue("**شاشةُ العنوان لا تسمع مفتاحَ الموقع**", s.contains("LocationServiceEffect("))
        assertTrue("**زرُّ «شغّل الموقع» يفتح الإعدادات لا النافذة**", s.contains("enableLocation()"))
    }

    @Test
    fun customerShellRefreshesOnSwitchAndResume() {
        val s = read(main)
        assertTrue("**لا سماعَ لمفتاح الموقع في الغلاف**", s.contains("com.rahalgo.map.LocationServiceEffect"))
        assertTrue(
            "**العودةُ من الإعدادات لا تلتقط الموقعَ المشغَّل**",
            s.contains("p == Locating.Problem.SERVICE_OFF && com.rahalgo.ui.locationServiceEnabled(context)"),
        )
    }

    @Test
    fun liveShareTurnsLocationOnFirst() {
        val s = read(main)
        assertTrue("**المشاركةُ تبدأ والموقعُ مطفأ**", s.contains("rememberLocationEnabler") && s.contains("startLive(o.id)"))
    }

    /**
     * **والعنوانُ المضافُ من «أين تريد التوصيل؟» يصير مختاراً** (قرارُ المالك ٢٠٢٦-١٠-٢١) —
     * كان يُحفظ ويبقى القديمُ مختاراً فيذهب الطلبُ إلى غير ما أراد.
     */
    @Test
    fun addressAddedFromSheetBecomesSelected() {
        val addr = read("ui/src/main/kotlin/com/rahalgo/ui/Address.kt")
        val vm = read("ui/src/main/kotlin/com/rahalgo/ui/AccountViewModel.kt")
        val model = read("shared/src/main/kotlin/com/rahalgo/shared/model/Account.kt")
        assertTrue("**لا يُعرف أنّ الجديدَ جاء من اللوحة**", addr.contains("fromSheet = picking && id == null"))
        assertTrue("**المحرّرُ لا يُمرّر الاختيار**", addr.contains("select = selectNew"))
        assertTrue("**النموذجُ لا يُرسل is_default**", model.contains("@SerialName(\"is_default\")"))
        assertTrue("**الطلبُ لا يحمل الاختيار**", vm.contains("isDefault = if (select) true else null"))
    }

    @Test
    fun driverTripUsesTheDialogToo() {
        val s = read(trip)
        assertTrue("**رحلةُ السائق تفتح الإعدادات لا النافذة**", s.contains("enableLocation()"))
    }
}
