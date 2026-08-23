package com.rahalgo.navigation

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطبقةُ الثانية — رحلةٌ تُعاد على مسارٍ حقيقيّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (منظومةُ اختبار تطبيق السائق، بأمر المالك ٢٠٢٦-٠٨-٢٣؛ وقاعدتُه
 *  ٢٠٢٦-٠٨-٢٠: «نستطيع اختبارَ مئات التغييرات في الملاحة على رحلاتٍ
 *  حقيقيّةٍ مسجَّلةٍ دون إعادة القيادة كلَّ مرّة».)
 *
 * # والمسارُ حقيقيٌّ لا مصنوع
 *
 * **`raqqa-route.json` جاء من محرّك الإنتاج نفسِه** (٢٠٢٦-٠٨-٢٣):
 * أربعةُ آلافٍ ومئتان وتسعةٌ وتسعون متراً · مئةٌ ورأسان · اثنتا عشرةَ
 * مناورة.
 *
 * **ومسارٌ مصنوعٌ بيدٍ لا يشبه الشوارع**: زواياه قائمةٌ وأضلاعُه
 * متساوية، **فيمرّ عليه كلُّ شيءٍ ولا يمرّ عليه شارعُ الرقّة.**
 *
 * # وما تمسكه هذه الطبقةُ ولا تمسكه الأولى
 *
 * **الأولى تعدّ النداءات** — وهذه **تقود المسارَ من أوّله إلى آخره**:
 * الإسقاطُ، والتقدّمُ، وكشفُ الخروج، والوصول. **وهي الطبقةُ التي
 * تُغني عن الشارع في كلّ تعديلٍ إلّا الأخير.**
 *
 * # ولماذا لا تُقرأ رفيدةٌ مسجَّلة
 *
 * **`TraceRecorder` محصورٌ في بناء التطوير** (أمرُ المالك: «ليس
 * تسجيلَ GPS دائماً في نسخة الإنتاج») — **والمثبَّتُ على جهازه كان
 * إصداراً**، فما سجّل شيئاً. **فلا رفيدةَ حقيقيّةً عندنا بعد.**
 *
 * **والحركةُ تُصنع من المسار الحقيقيّ** بـ`ReplayDrive` — وهي دون
 * الرفيدة في الصدق **وفوقها في التكرار**: لا ضجيجَ ولا انقطاع.
 * **ويومَ تصل رفيدةٌ من الشارع تُضاف بجانبها لا مكانَها.**
 */
class ReplayRouteTest {

    /**
     * **وتُقرأ الرفيدةُ بلا مكتبة.**
     *
     * **و`org.json` مبتورٌ في اختبارات الـJVM** — يردّ `Stub!` ولا
     * يُقرأ منه شيء. **ومكتبةُ JSON تُضاف لرفيدةٍ من مئة سطرٍ حِملٌ
     * على الوحدة كلِّها.**
     */
    private fun route(): NavRoute {
        val f = File("src/test/resources/raqqa-route.txt")
        assertTrue("لم تُوجد الرفيدة: ${f.absolutePath}", f.exists())
        val pts = ArrayList<GeoPoint>()
        val raw = ArrayList<Triple<String, String, Double>>()
        var mode = ""
        for (line in f.readLines()) {
            val l = line.trim()
            if (l.isEmpty() || l.startsWith("#")) continue
            if (l.startsWith("@")) { mode = l; continue }
            val c = l.split(" ")
            when (mode) {
                "@geometry" -> pts.add(GeoPoint(c[0].toDouble(), c[1].toDouble()))
                // **ويُقرأ سطرُ المناورة من آخره** — **ونوعُها قد يحمل
                // فراغاً**: `exit rotary` و`new name` كما ردّهما المحرّكُ
                // في هذه الرفيدة بالذات. **والفصلُ بالمسافة من الأوّل
                // يعطي أربعةَ أجزاءٍ لا ثلاثة.**
                "@steps" -> raw.add(
                    Triple(
                        c.dropLast(2).joinToString(" "),
                        c[c.size - 2],
                        c.last().toDouble(),
                    ),
                )
            }
        }
        // **والتراكميّاتُ تُبنى من الهندسة نفسِها** — **وتراكميّةٌ من
        // مصدرٍ آخرَ لا تطابق الرؤوسَ فيشير التقدّمُ إلى موضعٍ ليس
        // عليه** (وهي علّةُ المرحلة ٢).
        val cum = DoubleArray(pts.size)
        for (i in 1 until pts.size) {
            cum[i] = cum[i - 1] + meters(pts[i - 1], pts[i])
        }
        val mans = ArrayList<NavManeuver>()
        var at = 0.0
        for ((type, mod, dist) in raw) {
            mans.add(
                NavManeuver(
                    kind = kindOf(type, mod),
                    modifier = mod.takeIf { it != "-" },
                    atDistanceM = at,
                ),
            )
            at += dist
        }
        return NavRoute(geometry = pts, cumulativeM = cum, maneuvers = mans)
    }

    /**
     * **وأنواعُ المحرّك تُترجَم إلى أنواعنا** — كما في `NavRouteMapper`.
     *
     * **وتعدادُ مناوراتٍ عندنا لا كلماتِ OSRM** (المرحلة ٢): **ومن
     * مرّرها ربط تطبيقَ السائق بمحرّكٍ بعينه.**
     */
    private fun kindOf(type: String, mod: String): String = when {
        type == "depart" -> ManeuverKinds.DEPART
        type == "arrive" -> ManeuverKinds.ARRIVE
        type == "rotary" || type == "roundabout" -> ManeuverKinds.ROUNDABOUT
        type.startsWith("exit ") -> ManeuverKinds.EXIT_ROUNDABOUT
        type == "turn" && mod == "right" -> ManeuverKinds.TURN_RIGHT
        type == "turn" && mod == "left" -> ManeuverKinds.TURN_LEFT
        type == "continue" && mod == "uturn" -> ManeuverKinds.U_TURN
        else -> ManeuverKinds.STRAIGHT
    }

    private fun meters(a: GeoPoint, b: GeoPoint): Double {
        val r = 6_371_000.0
        val dLat = Math.toRadians(b.lat - a.lat)
        val dLng = Math.toRadians(b.lng - a.lng)
        val h = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(Math.toRadians(a.lat)) * Math.cos(Math.toRadians(b.lat)) *
            Math.sin(dLng / 2) * Math.sin(dLng / 2)
        return 2 * r * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h))
    }

    private fun drive(r: NavRoute, target: GeoPoint? = null): List<NavState> {
        val e = NavEngine()
        e.setRoute(r)
        if (target != null) e.arrivalTarget = target
        return ReplayDrive.fixes(r.geometry).map { e.onFix(it) }
    }

    // ══════════════════════════════════════════════════════════════════
    // **١ · الرفيدةُ صالحةٌ للملاحة**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `the fixture is a usable route`() {
        val r = route()
        assertTrue("المسارُ غيرُ صالحٍ للملاحة", r.usable)
        assertEquals(102, r.geometry.size)
        assertEquals(12, r.maneuvers.size)
        // **وطولُه المحسوبُ يقارب ما قاله المحرّك** — **وفرقٌ كبيرٌ
        // بينهما يعني أنّ الهندسةَ ليست هي التي قِيست عليها المسافة.**
        assertEquals(4299.0, r.totalM, 60.0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · القيادةُ من أوّله إلى آخره**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **والتقدّمُ لا يتراجع.**
     *
     * **وهي العلّةُ التي أُصلحت في المرحلة ٢**: مسارٌ يمرّ قربَ نفسِه
     * يجعل «أقربَ نقطة» تقفز من ٢٥٪ إلى ٧٠٪ **والسائقُ لم يتحرّك.**
     * **وهذا المسارُ فيه دورانٌ للخلف** (`uturn`) — فهو موضعُ الخطر
     * بعينه.
     */
    @Test
    fun `progress never goes backwards`() {
        var last = -1.0
        for (s in drive(route())) {
            val p = s.progress?.progressM ?: continue
            assertTrue("تراجع التقدّم: $last ← $p", p >= last - 0.5)
            last = p
        }
        assertTrue("لم يتقدّم شيءٌ أصلاً", last > 4000.0)
    }

    /**
     * **ولا يُعلَن خروجٌ عن مسارٍ يُقاد عليه حرفيّاً.**
     *
     * **وإعلانُ خروجٍ كاذبٍ يُعيد الحسابَ بلا انقطاع** — فيُغرق
     * المحرّكَ ويستنزف حزمةَ السائق.
     */
    @Test
    fun `driving the route itself never reports off-route`() {
        for (s in drive(route())) {
            assertFalse("أُعلن خروجٌ ونحن على المسار", s.isOffRoute)
        }
    }

    /** **ولا سيرٌ عكسَ الاتّجاه** — والمشيُ على الهندسة بترتيبها. */
    @Test
    fun `driving forward is never read as wrong-way`() {
        for (s in drive(route())) {
            assertFalse("قيل عكسُ الاتّجاه ونحن معه", s.isWrongWay)
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · والوصولُ يُعلَن عند الوجهة لا قبلها**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `arrival is declared only at the end`() {
        val r = route()
        val states = drive(r, target = r.geometry.last())
        assertTrue("لم يُعلَن وصولٌ أصلاً", states.last().arrivedAtTarget)
        // **ولا يُعلَن في النصف الأوّل** — **ووصولٌ مبكّرٌ يُقفل الرحلةَ
        // والسائقُ في الطريق.**
        val half = states.size / 2
        for (i in 0 until half) {
            assertFalse("وصولٌ مبكّرٌ عند القراءة $i", states[i].arrivedAtTarget)
        }
    }

    /**
     * **والخروجُ عن المسار يُكشَف حين يقع فعلاً.**
     *
     * **وكاشفٌ لا يكشف أسوأُ من كاشفٍ يكذب** — السائقُ يمضي في شارعٍ
     * خطأٍ ولا شيء يقول له.
     */
    @Test
    fun `a real detour is detected`() {
        val r = route()
        val e = NavEngine()
        e.setRoute(r)
        val onRoute = ReplayDrive.fixes(r.geometry)
        // **ويُقاد نصفُه ثمّ ينحرف** — انحرافاً عموديّاً على المسار.
        onRoute.take(onRoute.size / 2).forEach { e.onFix(it) }

        // ══════════════════════════════════════════════════════════════
        // **والانحرافُ سيرٌ لا قفز**
        // ══════════════════════════════════════════════════════════════
        //
        // **وقفزةُ مئتي مترٍ في ثانيةٍ ترفضها المصفاةُ** (`NavPipeline`
        // يعدّها `rejectedTeleport`) — **والمرفوضةُ ليست شهادة**، فلا
        // يُحسم خروجٌ أبداً.
        //
        // **فيمشي سبعةً وسبعين متراً في كلّ ثانية**… لا: **سبعةً
        // ونصفاً** — سرعةُ الإعادة نفسُها (٨٫٣ م/ث)، **فيبلغ المئتين
        // في نحو خمسٍ وعشرين قراءة.**
        //
        // **وثمانيةُ أمتارٍ في الثانية هي ما يقطعه من انحرف في شارعٍ
        // جانبيّ** — وذاك ما نريد كشفَه، لا اختطافاً.
        var off = false
        var t = onRoute[onRoute.size / 2].atMs
        val from = r.geometry[r.geometry.size / 2]
        // **ودرجةُ عرضٍ ≈ ١١١ كم** — فسبعةٌ ونصفٌ منها ٠٫٠٠٠٠٦٨.
        val stepDeg = 0.000068
        for (i in 1..30) {
            t += 1000
            val s = e.onFix(
                NavFix(
                    lat = from.lat + stepDeg * i,
                    lng = from.lng,
                    accuracyM = 5f,
                    speedMps = 7.5f,
                    // **وشمالاً** — والانحرافُ عن مسارٍ شرقيٍّ غربيّ.
                    bearingDeg = 0f,
                    atMs = t,
                ),
            )
            if (s.isOffRoute) off = true
        }
        assertTrue("لم يُكشف خروجٌ بعد انحرافِ مئتي مترٍ سيراً", off)
    }
}
