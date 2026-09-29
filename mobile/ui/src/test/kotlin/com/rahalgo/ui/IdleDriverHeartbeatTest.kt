package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **السائقُ الواقفُ يبقى مرئيّاً** — `GAP-LOC-IDLE-01`.
 *
 * # ما قِيس على الجهاز
 *
 * **SM-A525F، ٢٠٢٦-٠٩-٢٩، والتطبيقُ في الواجهة على شاشة الرحلة**: الجهازُ
 * ساكنٌ على الطاولة، **فلم يُرسَل موضعٌ واحد.** وعمرُ الموضع على الخادم نما
 * بلا انقطاع: **٢٥٠ ⇐ ٣١٦ ⇐ ٣٣٦ ⇐ ٣٧٠ ⇐ ٤٠٥ ⇐ ٤٣٩ ثانية.**
 *
 * **وطلبٌ خاصٌّ (#1225) بقي في الطابور دقائقَ ولم يُعرَض عليه** — وهو مؤهَّلٌ
 * بكلّ شرطٍ آخر: في الورديّة، حالتُه نشطة، طلبٌ واحدٌ من سقفِ اثنين، ولا
 * عرضَ حيٌّ عنده. **واستعلامُ شبكة الأمان نفسُه ردّ `NONE`** — **والحداثةُ
 * وحدَها هي المانع.**
 *
 * # الجذر
 *
 * `LocationService.request` تضع `setMinUpdateDistanceMeters(MIN_MOVE_M)`
 * و`MIN_MOVE_M = 20`، **والتعليقُ صريحٌ: «ومن وقف لا يُرسل».**
 * **ومحرّكُ التوزيع يشترط موضعاً أحدثَ من `drivers.location_fresh_sec`**
 * (٣٠٠ث افتراضاً).
 *
 * **فالسائقُ المتوقّفُ ينتظر عملاً يصير شائخاً بعد خمس دقائق، فلا يُعرض عليه
 * شيءٌ أبداً حتّى يتحرّك عشرين متراً.**
 *
 * **والعطبُ يضرب أسوأَ ما يمكن**: المنشغلُ يتحرّك فيبقى مرئيّاً، **والفارغُ
 * الواقفُ يختفي** — وهو بعينه من يُراد إعطاؤه طلباً.
 *
 * (حكمُ المالك ٢٠٢٦-٠٩-٢٩: «لا يجوز ألّا يُعرض أيُّ طلبٍ على سائقٍ لأنّه
 *  متوقّف، هذا غلطٌ كبير».)
 *
 * # والإصلاحُ نبضةٌ لا إلغاءُ مرشِّح
 *
 * **مرشِّحُ العشرين متراً يبقى للقراءات الجديدة** — فلا يُستنزَف الجهازُ
 * بضجيج قمرٍ صناعيّ. **وتُضاف نبضةٌ تُعيد إرسالَ آخر موضعٍ معلومٍ** حين يطول
 * الصمتُ، **فتبقى الحداثةُ صادقةً**: الخادمُ يحتاج «أعرف أين هو الآن»، لا
 * «تحرّك».
 *
 * **ودورتُها أقصرُ من حدّ الحداثة بهامشٍ** — نبضةٌ تساويه تصل متأخّرةً ثانيةً
 * فتسقط.
 */
class IdleDriverHeartbeatTest {

    private fun mobileRoot(): File {
        var d: File = File("").absoluteFile
        while (d.parentFile != null && !File(d, "settings.gradle.kts").exists()) {
            d = d.parentFile!!
        }
        return d
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val svc =
        "app-driver/src/main/kotlin/com/rahalgo/driver/location/LocationService.kt"

    /** **IDLE-01 · ثمّة نبضةٌ تُبقي الواقفَ حديثاً.** */
    @Test
    fun idleDriverStillReportsPosition() {
        val src = read(svc)
        assertTrue(
            "**لا نبضةَ في خدمة الموقع** — **فالواقفُ يشيخ فلا يُعرض عليه شيء** " +
                "(قِيس على الجهاز ٢٠٢٦-٠٩-٢٩: العمرُ نما ٢٥٠⇒٤٣٩ والتطبيقُ مفتوح).",
            src.contains("HEARTBEAT_SEC"),
        )
        assertTrue(
            "**النبضةُ لا تُعيد إرسالَ آخر موضعٍ معلوم** — وإعلانُ ثابتٍ بلا " +
                "مسارٍ يستعمله زينةٌ لا إصلاح.",
            src.contains("heartbeat"),
        )
    }

    /**
     * **IDLE-02 · ومرشِّحُ الحركة يبقى — النبضةُ تُكمله لا تُلغيه.**
     *
     * **ومن حذف المرشِّحَ ليحلّ المشكلةَ أغرق الجهازَ**: كلُّ ضجيجِ قمرٍ
     * صناعيٍّ يصير نداءَ شبكة.
     */
    @Test
    fun movementFilterStays() {
        val src = read(svc)
        assertTrue(
            "**حُذف مرشِّحُ الحركة** — النبضةُ تُكمله لا تُلغيه",
            src.contains("setMinUpdateDistanceMeters(MIN_MOVE_M)"),
        )
    }

    /**
     * **IDLE-03 · ودورةُ النبضة أقصرُ من حدّ الحداثة على الخادم.**
     *
     * **والحدُّ `drivers.location_fresh_sec` افتراضُه ٣٠٠** — فنبضةٌ كلَّ
     * ٣٠٠ تصل متأخّرةً فتسقط. **والهامشُ شرطٌ لا زينة.**
     */
    @Test
    fun heartbeatIsWellUnderServerFreshness() {
        val src = read(svc)
        val m = Regex("HEARTBEAT_SEC\\s*=\\s*(\\d+)").find(src)
        assertTrue("**لا قيمةَ لـ`HEARTBEAT_SEC`**", m != null)
        val sec = m!!.groupValues[1].toLong()
        assertTrue(
            "**دورةُ النبضة $sec ث** — **والخادمُ يُشيخ عند ٣٠٠**، فلا هامشَ. " +
                "يُطلب ١٥٠ أو أقلّ.",
            sec in 1..150,
        )
    }
}
