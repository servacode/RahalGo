package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«موقعي» موصولةٌ فعلاً — في الأربعة** (`MLW`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ فحصُ بنيةٍ لا فحصُ سلوك
 *
 * **و`LocatingTest` يقيس آلةَ الحال** — **وكانت خضراءَ كلَّها والحالُ
 * غيرُ موصولةٍ بزرٍّ واحد**: **صفٌّ مكتوبٌ لا تناديه شاشة.**
 *
 * **وهو ما أُخذ عليّ**: **«موجودٌ» ليست «موصول».**
 *
 * **وقشرةُ كلّ تطبيقٍ في وحدته** — **فلا فحصَ سلوكٍ في `ui` يراها**،
 * **والنصُّ يُقرأ نصّاً.**
 */
class MyLocationWiringTest {

    private companion object {
        /** **مواضعُ «موقعي» الحقيقيّة** — **زرٌّ يضغطه صاحبُه.** */
        val SHELLS = listOf(
            "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt",
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/MainActivity.kt",
            "app-rep/src/main/kotlin/com/rahalgo/rep/MainActivity.kt",
        )
        const val CUSTOMER_HERE =
            "app-customer/src/main/kotlin/com/rahalgo/customer/Here.kt"
        const val ENGINE = "map/src/main/kotlin/com/rahalgo/map/Here.kt"
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

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText()
    }

    /**
     * **MLW-01 · وزرُّ «موقعي» يمرّر حالاً في التطبيقات الثلاثة.**
     *
     * **ومن نسيها في واحدٍ ترك زرّاً ينادي ويمضي** — **وهو العطبُ الذي
     * بدأت منه الدفعة.**
     */
    @Test
    fun `الشاشاتُ تمرّر حالَ التحديد إلى ملتقط النقطة`() {
        for (rel in SHELLS) {
            val src = read(rel)
            assertTrue(
                "**زرُّ «موقعي» بلا حالٍ في** " + rel + " — **ينادي ويمضي**",
                src.contains("locating = locating"),
            )
            assertTrue(
                "**لا حالَ تعيش مع الشاشة في** " + rel,
                src.contains("remember { Locating() }"),
            )
        }
    }

    /**
     * **MLW-02 · والنداءُ نفسُه يحمل الحال.**
     *
     * **ومن مرّرها إلى الرسم ونسيها في النداء رسم زرّاً لا يدور أبداً.**
     */
    @Test
    fun `النداءُ نفسُه يحمل الحال`() {
        for (rel in SHELLS) {
            val src = read(rel)
            assertTrue(
                "**نداءُ الموضع بلا حالٍ في** " + rel,
                Regex("Here\\.refresh\\([^)]*locating").containsMatchIn(src),
            )
        }
    }

    /**
     * **MLW-03 · ولا نسخةَ ثانيةً من قارئ الموضع.**
     *
     * **وكانت ثلاثاً متطابقةً حرفاً بحرف** — **فأُصلحت إحداها وبقي
     * الاثنان كما كانا**، **ولا يعلم أحدٌ أيُّها الحقّ.**
     */
    @Test
    fun `قارئُ الموضع واحدٌ لا نسخٌ في التطبيقات`() {
        val root = mobileRoot()
        val copies = listOf(
            "app-rep/src/main/kotlin/com/rahalgo/rep/Here.kt",
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/Here.kt",
            "app-driver/src/main/kotlin/com/rahalgo/driver/Here.kt",
        )
        for (rel in copies) {
            assertFalse("**نسخةٌ ثانيةٌ عادت**: " + rel, File(root, rel).exists())
        }
        // **وما بقي في تطبيق الزبون لا يقرأ الموضعَ بنفسه** — **نقطةُ
        // الاستكشاف وحدَها.**
        val cust = read(CUSTOMER_HERE)
        assertFalse(
            "**تطبيقُ الزبون عاد يقرأ الموضعَ بنفسه**",
            cust.contains("getCurrentLocation"),
        )
        assertTrue(
            "**لا ينادي المحرّكَ المركزيّ**",
            cust.contains("com.rahalgo.map.Here.refresh"),
        )
    }

    /**
     * **MLW-04 · والزرُّ لا يُقبل ضغطاً ثانياً وهو يعمل.**
     *
     * **ونداءان يتسابقان يصل أقدمُهما آخراً فيُوسَّط به.**
     */
    @Test
    fun `الملتقطُ يمنع الضغطةَ الثانية`() {
        val pick = read("map/src/main/kotlin/com/rahalgo/map/PickPoint.kt")
        assertTrue(
            "**الضغطةُ الثانيةُ تفتح نداءً ثانياً**",
            pick.contains("if (!busy) { vm.wantHere(); onLocate() }"),
        )
        assertTrue("**لا دورانَ يقول إنّه عمل**", pick.contains("CircularProgressIndicator"))
        assertTrue("**السببُ لا يُرسَم**", pick.contains("LocatingText.message"))
    }

    /** **MLW-05 · وثلاثُ درجاتِ دقّةٍ لا واحدة.** */
    @Test
    fun `الحدودُ مسمّاةٌ ثلاثاً`() {
        assertEquals(1000f, Accuracy.CENTER_M, 0f)
        assertEquals(500f, Accuracy.CONFIRM_M, 0f)
        assertEquals(100f, Accuracy.DRIVE_M, 0f)
        assertTrue(Accuracy.DRIVE_M < Accuracy.CONFIRM_M)
        assertTrue(Accuracy.CONFIRM_M < Accuracy.CENTER_M)
    }

    /**
     * **MLW-06 · ولكلّ تعذّرٍ نصُّه وعلاجُه — ستّةٌ لا واحد.**
     *
     * **ومن زاد سبباً سابعاً ونسي نصَّه أسقط هذا** — **ولا يقع على
     * شاشة زبونٍ سطرٌ فارغ.**
     */
    @Test
    fun `لكلّ سببٍ نصٌّ وعلاج`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/Locating.kt")
        val message = src.substringAfter("fun message(").substringBefore("fun action(")
        val action = src.substringAfter("fun action(")
        for (p in Locating.Problem.values()) {
            assertTrue("**سببٌ بلا نصّ**: " + p.name, message.contains(p.name))
            assertTrue("**سببٌ بلا علاج**: " + p.name, action.contains(p.name))
        }
    }

    /**
     * **MLW-07 · وللنداء مهلةٌ ينتهي إليها.**
     *
     * **وزرٌّ يدور إلى الأبد أسوأُ من زرٍّ لا يدور** — **ومن انتظر
     * دقيقةً ظنّ الجهازَ علق.**
     */
    @Test
    fun `للنداء مهلةٌ تنتهي إليها`() {
        val engine = read(ENGINE)
        // **وثابتٌ معرَّفٌ ليس مهلةً تعمل** — **فيُطلَب استعمالُه في
        // تأجيلٍ حقيقيّ** (درسُ الشاهد الثاني: اسمٌ في ملفٍّ لا يكفي).
        assertTrue(
            "**لا تأجيلَ يستعمل المهلة**",
            Regex("""postDelayed\(\{[\s\S]{0,400}?\}, TIMEOUT_MS\)""").containsMatchIn(engine),
        )
        assertTrue(
            "**المهلةُ تُسقط نداءً انتهى**",
            engine.contains("if (state.busy) state.failed(Locating.Problem.TIMEOUT)"),
        )
    }

    /**
     * **MLW-08 · والمحرّكُ يُنتج كلَّ سببٍ يُعرَض.**
     *
     * **وسببٌ لا يُنتجه أحدٌ نصٌّ معروضٌ لا يُبلَغ** — **وهو ما وقع في
     * الدفعة الخامسة**: **نوعُ تبدّلٍ مكتوبٌ في العرض ولا منتِجَ له.**
     *
     * **والإذنُ المرفوضُ نهائيّاً تُنتجه الشاشةُ لا المحرّك** —
     * **لأنّ معرفتَه تحتاج نشاطاً** (`shouldShowRequestPermissionRationale`).
     */
    @Test
    fun `أسبابُ التعذّر يُنتجها المحرّكُ فعلاً`() {
        val engine = read(ENGINE)
        val produced = listOf(
            Locating.Problem.PERMISSION_DENIED,
            Locating.Problem.SERVICE_OFF,
            Locating.Problem.TIMEOUT,
            Locating.Problem.WEAK_ACCURACY,
            Locating.Problem.UNAVAILABLE,
        )
        for (p in produced) {
            // **والاسمُ في تعليقٍ ليس إنتاجاً** — **فيُطلَب النداءُ
            // نفسُه**: `failed(Locating.Problem.X)`.
            assertTrue(
                "**سببٌ معروضٌ لا منتِجَ له**: " + p.name,
                engine.contains("failed(Locating.Problem." + p.name + ")"),
            )
        }
    }

    /**
     * **MLW-09 · والمرحلتان قائمتان.**
     *
     * **وآخرُ موضعٍ معروفٍ يصل في اللحظة** — **ثمّ يُصحَّح بالطازج**:
     * **ومن حذف الأولى أعاد الزرَّ الصامتَ ثلاثَ ثوانٍ.**
     */
    @Test
    fun `المحرّكُ يبدأ بالقديم ثمّ يُصحّح`() {
        val engine = read(ENGINE)
        // **والنداءُ نفسُه يُطلب** — **و«lastLocation» في تعليقٍ أو في
        // اسمٍ آخر ليست مرحلةً تعمل** (كشفه الشاهدُ الثاني).
        assertTrue(
            "**لا مرحلةَ أولى**",
            engine.contains("client.lastLocation.addOnSuccessListener"),
        )
        assertTrue("**لا تصحيحَ بالطازج**", engine.contains("getCurrentLocation"))
        assertTrue("**القديمُ لا يُعلَّم مؤقّتاً**", engine.contains("provisional()"))
    }

    /**
     * **MLW-10 · وخدمةُ الموقع تُسأل قبل أن يُنتظَر قياسٌ لا يجيء.**
     *
     * **والإذنُ ممنوحٌ والخدمةُ مطفأةٌ ⇒ يُنادى المزوّدُ فلا يجيب
     * أبداً** — **فتُقال مهلةٌ والسببُ معروفٌ من أوّله.**
     */
    @Test
    fun `الخدمةُ المطفأةُ تُقال قبل المهلة`() {
        val engine = read(ENGINE)
        val at = engine.indexOf("locationServiceEnabled")
        assertTrue("**لا سؤالَ عن خدمة النظام**", at > 0)
        assertTrue(
            "**سُئلت الخدمةُ بعد أن بدأت المهلة**",
            at < engine.indexOf("TIMEOUT_MS", engine.indexOf("postDelayed") - 200),
        )
    }

    /**
     * **MLW-11 · وحدُّ الدقّة يُختار لكلّ استعمال.**
     *
     * **وتأكيدُ نقطةٍ يُسلَّم إليها طلبٌ لا يقبل ما يقبله توسيطُ خريطة.**
     */
    @Test
    fun `الشاشاتُ تختار حدَّ الدقّة`() {
        val cust = read(CUSTOMER_HERE)
        assertTrue(
            "**الزبونُ بلا حدِّ تأكيد**",
            cust.contains("Accuracy.CONFIRM_M"),
        )
        for (rel in listOf(SHELLS[1], SHELLS[2])) {
            assertTrue(
                "**«موقعي» بلا حدِّ دقّةٍ في** " + rel,
                read(rel).contains("Accuracy.CONFIRM_M"),
            )
        }
    }

    /**
     * **MLW-12 · وما لا تكفي دقّتُه لا يُكتب موضعاً.**
     *
     * **ونقطةٌ بخطإِ كيلومترين تُكتب كأنّها بابُه** — **فيُرسَل إليها
     * سائق.**
     */
    @Test
    fun `الدقّةُ الضعيفةُ لا تُكتب موضعاً`() {
        val fresh = read(ENGINE).substringAfter("getCurrentLocation")
        val weakAt = fresh.indexOf("WEAK_ACCURACY")
        val setAt = fresh.indexOf("LastPoint.set")
        assertTrue("**لا فحصَ دقّةٍ بعد القياس**", weakAt > 0)
        assertTrue("**لا كتابةَ موضع**", setAt > 0)
        assertTrue("**كُتب الموضعُ قبل أن تُفحَص دقّتُه**", weakAt < setAt)
    }

    /**
     * **MLW-13 · ونقطةُ الاستكشاف تبقى استكشافاً.**
     *
     * **وعقدُ الدفعة الخامسة قائم** — **تُخبِر ولا تحكم**: **ولا تصير
     * عنوانَ توصيلٍ إلّا بفعلٍ صريح.**
     */
    @Test
    fun `نقطةُ الاستكشاف لا تصير عنوانَ توصيل`() {
        val cust = read(CUSTOMER_HERE)
        assertTrue("**ذهبت نقطةُ الاستكشاف**", cust.contains("discovery"))
        assertTrue(
            "**تُكتب بلا دقّةٍ** — **ونقطةٌ لا تُعرَف دقّتُها لا يُحكَم بها**",
            cust.contains("Discovery(lat, lng, acc)"),
        )
    }
}
