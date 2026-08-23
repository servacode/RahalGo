package com.rahalgo.navigation

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ الصوت — لا مقطعَ يُطلب وليس في القائمة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا حارسٌ لا اختبارُ سلوك
 *
 * **المقاطعُ تُولَّد من `clips.json` مرّةً وتُشحن** — **ومقطعٌ يطلبه
 * الشيفرةُ ولا يُولَّد يسقط صامتاً إلى المحرّك الآليّ**، فيسمع السائقُ
 * روبوتاً في منعطفٍ وصوتاً بشريّاً في الذي يليه.
 *
 * **وذاك أسوأُ من الروبوت كلِّه**: التبدّلُ يُقرأ عطباً.
 *
 * **فالجدولان يُقارَنان هنا** — والبناءُ يسقط إن افترقا.
 */
class VoiceClipsTest {

    private fun manifest(): Set<String> {
        // **والملفُّ من المستودع لا نسخةٌ في الاختبار** — **ونسخةٌ
        // ثانيةٌ تشيخ وحدَها فيحرس الاختبارُ ما لم يعد قائما.**
        val f = File("voice/clips.json")
        assertTrue("لم تُوجد قائمةُ المقاطع: ${f.absolutePath}", f.exists())
        return Regex("\"([a-z0-9_]+)\"\\s*:")
            .findAll(f.readText())
            .map { it.groupValues[1] }
            .filterNot { it.startsWith("_") }
            .toSet()
    }

    private fun man(kind: String, modifier: String? = null, exit: Int? = null) =
        NavManeuver(kind = kind, modifier = modifier, atDistanceM = 0.0, roundaboutExit = exit)

    private fun cue(
        kind: CueKind,
        m: NavManeuver? = null,
        d: Int? = null,
    ) = VoiceCue(
        id = CueId(generation = 1L, maneuverAtM = 0.0, stage = CueStage.NOW),
        kind = kind,
        stage = CueStage.NOW,
        maneuver = m,
        roundedDistanceM = d,
        priority = 1,
        validUntilProgressM = Double.MAX_VALUE,
        text = "",
    )

    /** **كلُّ ما تطلبه الشيفرةُ موجودٌ فيما يُولَّد.** */
    @Test
    fun `every requested clip exists in the manifest`() {
        val have = manifest()
        val kinds = listOf(
            ManeuverKinds.DEPART, ManeuverKinds.STRAIGHT,
            ManeuverKinds.TURN_RIGHT, ManeuverKinds.TURN_LEFT,
            ManeuverKinds.SLIGHT_RIGHT, ManeuverKinds.SLIGHT_LEFT,
            ManeuverKinds.SHARP_RIGHT, ManeuverKinds.SHARP_LEFT,
            ManeuverKinds.U_TURN, ManeuverKinds.MERGE,
            ManeuverKinds.OFF_RAMP, ManeuverKinds.EXIT_ROUNDABOUT,
            ManeuverKinds.ARRIVE, ManeuverKinds.UNKNOWN,
        )
        val asked = mutableSetOf<String>()
        for (k in kinds) {
            asked += VoiceClips.of(cue(CueKind.MANEUVER, man(k)))
            for (d in listOf(50, 100, 150, 200, 300, 400, 500, 700, 1000)) {
                asked += VoiceClips.of(cue(CueKind.MANEUVER, man(k), d))
            }
        }
        for (mod in listOf("right", "left", null)) {
            asked += VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.FORK, mod)))
        }
        for (e in 1..6) {
            asked += VoiceClips.of(
                cue(CueKind.MANEUVER, man(ManeuverKinds.ROUNDABOUT, exit = e)),
            )
        }
        asked += VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.ROUNDABOUT)))
        for (k in listOf(CueKind.REROUTE, CueKind.WRONG_WAY, CueKind.ROUTE_END, CueKind.ARRIVE)) {
            asked += VoiceClips.of(cue(k))
        }

        val missing = asked - have
        assertTrue("مقاطعُ تُطلب ولا تُولَّد: $missing", missing.isEmpty())
    }

    /** **والمسافةُ تسبق الفعل** — كما في الجملة المكتوبة. */
    @Test
    fun `distance comes before the action`() {
        assertEquals(
            listOf("d_200", "act_right"),
            VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.TURN_RIGHT), 200)),
        )
    }

    /**
     * **ولا تُنطق مناورةٌ بلا مسافتها.**
     *
     * **«انعطف يميناً» بلا «بعد مئتي متر» تُقرأ أمراً فوريّاً** —
     * فينعطف قبل أوانه. **فتُسلَّم كاملةً إلى المحرّك أو لا تُقال.**
     */
    @Test
    fun `an off-table distance drops the whole cue to the engine`() {
        assertTrue(
            VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.TURN_RIGHT), 137)).isEmpty(),
        )
    }

    /** **و«الآن» تلحق الفعلَ ولا تسبقه.** */
    @Test
    fun `now follows the action`() {
        assertEquals(
            listOf("act_left", "now"),
            VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.TURN_LEFT))),
        )
    }

    /** **والوصولُ لا يلحقه «الآن»** — «لقد وصلت الآن» ركيكة. */
    @Test
    fun `arrival takes no now`() {
        assertEquals(
            listOf("act_arrive"),
            VoiceClips.of(cue(CueKind.MANEUVER, man(ManeuverKinds.ARRIVE))),
        )
    }

    /** **ودوّارٌ بسبعة مخارجَ يسقط إلى المحرّك** — ولا يُقال بلا رقم. */
    @Test
    fun `a seventh roundabout exit drops to the engine`() {
        assertTrue(
            VoiceClips.of(
                cue(CueKind.MANEUVER, man(ManeuverKinds.ROUNDABOUT, exit = 7)),
            ).isEmpty(),
        )
    }
}
