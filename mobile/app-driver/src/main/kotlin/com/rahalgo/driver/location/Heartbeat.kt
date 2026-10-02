package com.rahalgo.driver.location

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.ui.LastPoint

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نبضةُ الواقف — ولا تُعيد موضعاً ماتت أقمارُه** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت تُعيد آخرَ موضعٍ معلومٍ كلَّ دقيقتين بلا حدّ عمر** — فسائقٌ مات GPS
 * هاتفه قبل ساعةٍ يبقى «حديثاً» عند الخادم، **فيُعرض عليه أقربُ طلبٍ وهو ليس
 * هناك**، ويضيع على الزبون خمسٌ وأربعون ثانيةً في كلّ عرض.
 *
 * **ولا يكفي أن تسكت النبضةُ بعد خمس دقائق**: مرشِّحُ العشرين متراً يُسكت
 * الواقفَ نفسَه — **فيشيخ من ينتظر عملاً وهو بعينه من يُراد إعطاؤه** (حكمُ
 * المالك ٢٠٢٦-٠٩-٢٩). **فحين يشيخ آخرُ موضعٍ يُسأل النظامُ موضعاً جديداً**:
 * إن ردّ أُرسل بوقت التقاطه، **وإن لم يردّ فلا إرسال** — يشيخ عند الخادم
 * فلا يُعرض عليه شيء، **ويُقال له في لوحته إنّ الإشارةَ مفقودة.**
 */
internal object Heartbeat {

    /** **كم صمتٍ قبل النبضة** — دون حدّ الحداثة عند الخادم بهامش. */
    const val EVERY_MS = LocationService.HEARTBEAT_SEC * 1000

    /** **أقصى عمرٍ لموضعٍ يُعاد** — وما زاد يُسأل عنه النظامُ من جديد. */
    const val MAX_RESEND_AGE_MS = 300_000L

    enum class Act {
        /** لا شيء — صمتٌ قصير، أو موضعٌ مزيَّفٌ لا يُغسَل بنبضة. */
        WAIT,

        /** **أعِد آخرَ موضع** — حيٌّ لم يشِخ، والواقفُ ما زال هناك. */
        RESEND,

        /** **اسأل النظامَ موضعاً جديداً** — شاخ ما عندنا أو لا موضعَ أصلاً. */
        REFRESH,
    }

    fun decide(nowMs: Long, lastSentMs: Long, point: LastPoint.Point?): Act {
        if (nowMs - lastSentMs < EVERY_MS) return Act.WAIT
        if (point == null) return Act.REFRESH
        if (point.mocked) return Act.WAIT
        // **وموضعٌ بلا وقتٍ عمرُه مجهول** — لا يُعاد على أنّه حيّ.
        if (point.atMs <= 0L || nowMs - point.atMs > MAX_RESEND_AGE_MS) return Act.REFRESH
        return Act.RESEND
    }
}

/**
 * **أإشارةُ الموقع حيّة؟** — تكتبها خدمةُ الورديّة، وتقرؤها لوحتُه.
 *
 * **ولا تُحفظ**: حالٌ عمرُها دقائق، وتُصفَّر حين تقف الخدمة.
 */
object GpsSignal {
    var lost by mutableStateOf(false)
        private set

    fun alive() {
        lost = false
    }

    fun dead() {
        lost = true
    }
}
