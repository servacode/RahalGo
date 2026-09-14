package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **النصُّ الصريحُ مفتوحٌ للتصحيح وحدَه** (`P-8`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع
 *
 * **بُني جسرُ `P-8`** — عنوانُ المحرّك يُوجَّه إلى `127.0.0.1:8080` عبر
 * `adb reverse` — **فأُقلعت القطعةُ على الجهاز فردّ النظامُ:**
 *
 *	java.net.UnknownServiceException: CLEARTEXT communication to
 *	127.0.0.1 not permitted by network security policy
 *
 * **وأندرويدُ ٩+ يمنع النصَّ الصريح افتراضاً** — **وكان تطبيقُ السائق
 * وحدَه يملك إعدادَ أمنِ شبكةٍ في `src/debug`** (قرارُ المالك
 * ٢٠٢٦-٠٨-٢٢) **لأنّه وحدَه كان يكلّم مضيفاً حلقيّاً.** **ويومَ صارت
 * الأربعةُ تكلّمه لزمها ما لزمه.**
 *
 * # وما يحرسه هذا الفحص
 *
 * **الأوّل — الفتحُ في `src/debug` لا في `main` ولا في `release`.**
 * **ومن نقل السمةَ إلى البيان الأصليّ فتح النصَّ الصريحَ في قطعةٍ
 * تُرفع إلى الناس** — **وهو ثقبٌ في العزل لا عطبٌ يظهر في شاشة.**
 *
 * **الثاني — والفتحُ محدودٌ بالمضيف الحلقيّ.** **`base-config` يبقى
 * مغلقاً**، **ومن أضاف نطاقاً عامّاً إلى `domain-config` فتح النصَّ
 * الصريحَ إلى الشبكة.**
 *
 * **والثالث — `usesCleartextTraffic="true"` ممنوعٌ في البيان الأصليّ**
 * — **وهو البابُ الأوسعُ**: يفتح كلَّ مضيفٍ بلا استثناء.
 *
 * # ولمَ نصٌّ هنا
 *
 * **والبيانُ نصٌّ** — **ولا نداءَ يُسأل عن موضع ملفٍّ في مجموعةِ
 * مصدر.** **والبيانُ المدموجُ للإصدار يُقاس على القطعة نفسِها** عند
 * بناء مرشَّحي الإصدار، **وهذا يمنع الخطأَ قبل أن يُبنى.**
 */
class CleartextPolicyTest {

    private companion object {
        val APPS = listOf("app-customer", "app-driver", "app-merchant", "app-rep")
    }

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() &&
                File(dir, "app-customer").exists()
            ) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    /** **والأربعةُ تملك إعدادَ التصحيح** — **فجسرُ `P-8` يعمل في الأربعة.** */
    @Test
    fun `الأربعةُ تفتح النصَّ الصريحَ في التصحيح`() {
        val root = mobileRoot()
        for (app in APPS) {
            val manifest = File(root, "$app/src/debug/AndroidManifest.xml")
            assertTrue(
                "**بيانُ تصحيحٍ مفقودٌ في " + app + "** — " +
                    "**فنداءُ التجهيز يسقط برفضٍ أمنيٍّ لا برسالةِ شبكة**",
                manifest.exists(),
            )
            assertTrue(
                "**بيانُ تصحيحِ " + app + " لا يُشير إلى إعداد أمنِ الشبكة**",
                manifest.readText().contains("android:networkSecurityConfig=\"@xml/network_security_config\""),
            )
        }
    }

    /**
     * **والفتحُ محدودٌ بالمضيف الحلقيّ** — **والأساسُ يبقى مغلقاً.**
     */
    @Test
    fun `الفتحُ لا يتجاوز المضيفَ الحلقيّ`() {
        val root = mobileRoot()
        val loopback = setOf("127.0.0.1", "localhost", "::1")
        for (app in APPS) {
            val f = File(root, "$app/src/debug/res/xml/network_security_config.xml")
            assertTrue("**إعدادُ أمنِ الشبكة مفقودٌ في " + app + "**", f.exists())
            val text = f.readText()
            assertTrue(
                "**أساسُ " + app + " لا يُغلق النصَّ الصريح** — فالفتحُ عامٌّ",
                text.contains("<base-config cleartextTrafficPermitted=\"false\" />"),
            )
            val domains = Regex("<domain[^>]*>([^<]+)</domain>")
                .findAll(text).map { it.groupValues[1].trim() }.toList()
            assertTrue("**لا نطاقَ مذكورٌ في " + app + "**", domains.isNotEmpty())
            for (d in domains) {
                assertTrue(
                    "**نطاقٌ غيرُ حلقيٍّ مفتوحٌ للنصّ الصريح في " + app + "**: " + d,
                    d in loopback,
                )
            }
            assertFalse(
                "**`includeSubdomains` يفتح ما تحت النطاق في " + app + "**",
                text.contains("includeSubdomains=\"true\""),
            )
        }
    }

    /**
     * **ولا فتحَ في البيان الأصليّ ولا في مجموعةِ الإصدار.**
     *
     * **وهذا هو الحارسُ الذي يمنع الحادثَ**: **سمةٌ تُنقَل إلى `main`
     * «لتعمل في الحالين» تفتح النصَّ الصريحَ في قطعةِ الناس.**
     */
    @Test
    fun `الإصدارُ لا يفتح النصَّ الصريح`() {
        val root = mobileRoot()
        for (app in APPS) {
            for (set in listOf("main", "release")) {
                val m = File(root, "$app/src/$set/AndroidManifest.xml")
                if (!m.exists()) continue
                val text = m.readText()
                assertFalse(
                    "**`networkSecurityConfig` في `" + set + "` من " + app +
                        "** — **فالإصدارُ يرث فتحَ التصحيح**",
                    text.contains("networkSecurityConfig"),
                )
                assertFalse(
                    "**`usesCleartextTraffic=\"true\"` في `" + set + "` من " + app +
                        "** — **وهو البابُ الأوسع**",
                    text.contains("usesCleartextTraffic=\"true\""),
                )
            }
            // **ولا إعدادَ أمنِ شبكةٍ في موارد `main`/`release`** —
            // **فملفٌّ هناك يُدمَج في الإصدار ولو لم يُشَر إليه اليوم.**
            for (set in listOf("main", "release")) {
                val res = File(root, "$app/src/$set/res/xml/network_security_config.xml")
                assertFalse(
                    "**إعدادُ أمنِ شبكةٍ في موارد `" + set + "` من " + app + "**",
                    res.exists(),
                )
            }
        }
    }
}
