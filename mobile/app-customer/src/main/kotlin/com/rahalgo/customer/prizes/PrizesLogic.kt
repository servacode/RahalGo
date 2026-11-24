package com.rahalgo.customer.prizes

import com.rahalgo.shared.customer.PrizeItem
import com.rahalgo.shared.customer.PrizesMe
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.temporal.ChronoUnit
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائزُ رحّال غو — القراراتُ الخالصة** (قرارُ المالك ٢٠٢٦-١١-٢٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كلُّ خمسِ دعواتٍ في الشهر فرصةٌ في سحبٍ تلقائيّ أوّلَ كلّ شهرٍ ١٢:٠٠ بتوقيت دمشق.
 * **والحسابُ الحقيقيّ (الفرص، الرصيد، الخصم) عند المحرّك** — وهنا ما تحتاجه الشاشةُ
 * لتعرض: **متى تُعرض لافتةُ «انتظرونا»، وكيف تُقسم الأصنافُ على الرصيد، وما الفرق.**
 *
 * **دالّاتٌ خالصة** — تُختبر بلا شاشة.
 */
object PrizesLogic {

    private val damascus: ZoneId = ZoneId.of("Asia/Damascus")

    /**
     * **أتُعرض اللافتةُ بدل الشاشة؟** — حين يُطفأ المفتاحُ، أو يأتي يومُ الافتتاح بعد اليوم.
     * **والتاريخان `yyyy-MM-dd` فالمقارنةُ نصّيّةٌ صحيحة.** ويومُ الافتتاح نفسُه مفتوح.
     */
    fun isTeaser(enabled: Boolean, launchOn: String, today: String): Boolean =
        !enabled || (launchOn.isNotBlank() && today < launchOn)

    /** **اليومُ بتوقيت دمشق** `yyyy-MM-dd`. */
    fun today(): String = java.time.LocalDate.now(damascus).toString()

    /**
     * **الأصنافُ على قدّ الجائزة وما فوقها.**
     *
     * [within] ما سعرُه ≤ الرصيد، و[over] ما فوقه (يُدفع فرقُه). **ورصيدٌ صفرٌ يعني عرضاً
     * فقط** ([showcase]): كلُّ شيءٍ في [over] ولا إضافة.
     */
    data class Split(val within: List<PrizeItem>, val over: List<PrizeItem>, val showcase: Boolean)

    fun split(items: List<PrizeItem>, balance: Long): Split {
        val sorted = items.sortedBy { it.price }
        if (balance <= 0) return Split(emptyList(), sorted, showcase = true)
        val (a, b) = sorted.partition { it.price <= balance }
        return Split(a, b, showcase = false)
    }

    /** **ما يدفعه الزبون زيادةً** — صفرٌ إن غطّاه الرصيد. */
    fun difference(price: Long, balance: Long): Long = maxOf(0L, price - maxOf(0L, balance))

    /** **تقدّمُه نحو الفرصة القادمة** — [current] من [perEntry]. */
    data class Progress(val current: Int, val perEntry: Int, val toNext: Int)

    fun progress(me: PrizesMe): Progress {
        val per = if (me.invitesPerEntry > 0) me.invitesPerEntry else 5
        val cur = maxOf(0, me.monthInvites) % per
        // **والمحرّكُ يقول ما تبقّى** — فإن غاب حُسب.
        val next = if (me.invitesToNext > 0) me.invitesToNext else per - cur
        return Progress(cur, per, next)
    }

    /** **شارةُ زرّ الهديّة «٣/٥»** — فارغةٌ قبل الافتتاح. */
    fun badge(me: PrizesMe?, teaser: Boolean): String? {
        if (me == null || teaser) return null
        val p = progress(me)
        return "${p.current}/${p.perEntry}"
    }

    /** **أتنبض الأيقونةُ؟** — حين يتبدّل العدّادُ (لا عند أوّل قراءةٍ ولا عند ثباته). */
    fun shouldPulse(previous: String?, now: String?): Boolean = now != null && previous != null && previous != now

    /** **يومُ الافتتاح كما يُكتب** — `2026-12-01` ← `1/12/2026`. وما لا يُقرأ يعود كما هو. */
    fun launchLabel(launchOn: String): String {
        val parts = launchOn.split("-")
        if (parts.size != 3) return launchOn
        val y = parts[0].toIntOrNull() ?: return launchOn
        val m = parts[1].toIntOrNull() ?: return launchOn
        val d = parts[2].toIntOrNull() ?: return launchOn
        return "$d/$m/$y"
    }

    /** **موعدُ السحب بتوقيت دمشق** — `1/12/2026 12:00`؛ وفارغٌ إن لم يُقرأ. */
    fun drawLabel(drawAt: String): String = runCatching {
        OffsetDateTime.parse(drawAt)
            .atZoneSameInstant(damascus)
            .format(DateTimeFormatter.ofPattern("d/M/yyyy HH:mm"))
    }.getOrDefault("")

    /** **يومُ انتهاء الرصيد** — `30/11/2026`؛ وفارغٌ إن لم يُقرأ. */
    fun expiryLabel(expiresAt: String?): String = runCatching {
        OffsetDateTime.parse(expiresAt.orEmpty())
            .atZoneSameInstant(damascus)
            .format(DateTimeFormatter.ofPattern("d/M/yyyy"))
    }.getOrDefault("")

    /** **سلّمُ الجوائز بمراتبه** — (1 ← مبلغ) (2 ← مبلغ) … */
    fun ladder(amounts: List<Long>): List<Pair<Int, Long>> = amounts.mapIndexed { i, a -> (i + 1) to a }

    /** **كم يوم باقي ليومٍ `yyyy-MM-dd`** بتوقيت دمشق — صفرٌ اليومَ وما فات، و`null` إن لم يُقرأ. */
    fun daysUntilDate(iso: String): Long? = runCatching {
        ChronoUnit.DAYS.between(LocalDate.now(damascus), LocalDate.parse(iso)).coerceAtLeast(0)
    }.getOrNull()

    /** **كم يوم باقي لموعدٍ كامل** (السحب) — بيومه في دمشق. */
    fun daysUntilInstant(at: String): Long? = runCatching {
        val day = OffsetDateTime.parse(at).atZoneSameInstant(damascus).toLocalDate()
        ChronoUnit.DAYS.between(LocalDate.now(damascus), day).coerceAtLeast(0)
    }.getOrNull()
}
