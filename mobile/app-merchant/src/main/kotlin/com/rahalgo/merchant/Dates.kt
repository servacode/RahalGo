package com.rahalgo.merchant

import android.content.Context
import com.rahalgo.ui.SYRIA_ZONE
import com.rahalgo.ui.syriaTime
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التاريخُ كما يقرؤه صاحبُ المتجر — لا كما تكتبه الآلة** (تدقيقُ ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان يظهر «2026-10-27»** في التقارير والإنذارات وسجلّ الطلبات — صيغةُ
 * آلة. **فصار «اليوم» و«أمس» ثمّ «الثلاثاء 27 تشرين الأول».**
 *
 * **وأسماءُ الأشهر والأيّام من المعجم لا من الشيفرة** — بالأسماء السوريّة،
 * **والأرقامُ غربيّةٌ كبقيّة التطبيق.**
 */
object Dates {

    private val TIME = DateTimeFormatter.ofPattern("HH:mm", Locale.US)

    /** **اسمُ اليوم بالتقويم** — «الثلاثاء 27 تشرين الأول»، **والسنةُ إن لم تكن سنتَنا.** */
    fun full(ctx: Context, date: LocalDate, today: LocalDate = LocalDate.now(SYRIA_ZONE)): String {
        val months = ctx.resources.getStringArray(R.array.months_sy)
        val month = months.getOrNull(date.monthValue - 1) ?: date.monthValue.toString()
        val base = weekday(ctx, date.dayOfWeek) + " " + date.dayOfMonth + " " + month
        return if (date.year != today.year) base + " " + date.year else base
    }

    /** **«اليوم» · «أمس» · ثمّ التاريخ كاملاً.** */
    fun day(ctx: Context, date: LocalDate, today: LocalDate = LocalDate.now(SYRIA_ZONE)): String =
        when (ChronoUnit.DAYS.between(date, today)) {
            0L -> ctx.getString(com.rahalgo.ui.R.string.chat_today)
            1L -> ctx.getString(com.rahalgo.ui.R.string.chat_yesterday)
            else -> full(ctx, date, today)
        }

    /** **يومُ طابعٍ من الخادم بتوقيت سوريا** — و`null` إن لم يُفهم. */
    fun dateOf(iso: String): LocalDate? =
        syriaTime(iso)?.toLocalDate()
            ?: runCatching { LocalDate.parse(iso.take(10)) }.getOrNull()

    /** **يومُ الطابع نصّاً** — وما لا يُفهم يُعاد كما هو. */
    fun dayOf(ctx: Context, iso: String): String = dateOf(iso)?.let { day(ctx, it) } ?: iso

    /** **«اليوم · 14:05»** — يومُ الطابع وساعتُه. */
    fun stamp(ctx: Context, iso: String): String {
        val at = syriaTime(iso) ?: return dayOf(ctx, iso)
        return day(ctx, at.toLocalDate()) + " · " + at.format(TIME)
    }

    /** **تاريخُ «yyyy-MM-dd»** (مدى التقرير) — بالاسم لا بالأرقام. */
    fun ymd(ctx: Context, ymd: String): String =
        runCatching { full(ctx, LocalDate.parse(ymd)) }.getOrDefault(ymd)

    private fun weekday(ctx: Context, d: DayOfWeek): String = ctx.getString(
        when (d) {
            DayOfWeek.SUNDAY -> R.string.days_sun
            DayOfWeek.MONDAY -> R.string.days_mon
            DayOfWeek.TUESDAY -> R.string.days_tue
            DayOfWeek.WEDNESDAY -> R.string.days_wed
            DayOfWeek.THURSDAY -> R.string.days_thu
            DayOfWeek.FRIDAY -> R.string.days_fri
            DayOfWeek.SATURDAY -> R.string.days_sat
        },
    )
}
