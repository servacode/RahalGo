package com.rahalgo.ui

import android.content.Intent
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.ui.push.RahalPushService

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بمَ فُتح التطبيق؟** — **وجهةٌ تُقرأ مرّةً** (`AN-02`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا حالٌ لا نداءٌ مباشر
 *
 * **والشاشةُ تُبنى بعد النشاط** — **ومن فتح وجهةً في `onCreate` فتحها
 * قبل أن يوجد ما يعرضها.**
 *
 * # وتُستهلَك مرّةً واحدة
 *
 * **ومن أدار جهازَه فأُعيد بناءُ الشاشة وجد نفسَه يُساق إلى العرض
 * ثانيةً** — **ثمّ كلّما أدار.** **فالوجهةُ تُقرأ وتُمحى.**
 */
object Opened {

    /** **ما لم يُقرأ بعد** — **وفارغُه يعني فتحةً عاديّة.** */
    var pending by mutableStateOf(Engagement.HOME)
        private set

    /**
     * **يُقرأ من مقصد النشاط** — **ويُنقّى ثانيةً.**
     *
     * **ولا يُصدَّق ما في المقصد** — **ومقصدٌ يصنعه غيرُنا قد يحمل
     * أيَّ نصّ**: **والتنقيةُ هنا كالتنقية عند الاستقبال.**
     */
    fun from(intent: Intent?) {
        if (intent == null) return
        val dest = Engagement.route(
            intent.getStringExtra(RahalPushService.EXTRA_DEST_TYPE),
            intent.getStringExtra(RahalPushService.EXTRA_DEST_ID),
        )
        if (!dest.isHome) {
            pending = dest
        }
        // **ويُمحى من المقصد نفسِه** — **فإعادةُ بناءِ النشاط لا تُعيد
        // قراءتَه** (وهو ما يفعله `Invited.fromLink` في مرآته).
        intent.removeExtra(RahalPushService.EXTRA_DEST_TYPE)
        intent.removeExtra(RahalPushService.EXTRA_DEST_ID)
    }

    /** **يأخذ الوجهةَ ويُفرغها** — **فلا تُفتَح مرّتين.** */
    fun take(): Engagement.Dest {
        val d = pending
        pending = Engagement.HOME
        return d
    }

    /** **يُصفَّر في الفحص.** */
    fun resetForTest() {
        pending = Engagement.HOME
    }
}
