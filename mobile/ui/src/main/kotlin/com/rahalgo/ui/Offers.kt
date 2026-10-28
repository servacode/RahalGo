package com.rahalgo.ui

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.graphics.Color
import androidx.compose.material3.Icon
import androidx.compose.material3.HorizontalDivider
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ العرض كما تُقرأ — ومدّتُه كما تُكتب** (`MO`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا رمزٌ آليٌّ على شاشة
 *
 * **و`scheduled` نصٌّ لمهندسٍ لا لصاحب مطعم** — **ومن قرأه ظنّ
 * التطبيقَ معطوباً.** (وهي قاعدةُ `CartChanges` نفسُها.)
 *
 * # ورمزٌ لا نعرفه لا يُخترَع له نصّ
 *
 * **ومحرّكٌ أحدثُ من الحزمة قد يردّ حالاً جديدة** — **فيُقال إنّها
 * غيرُ معروفةٍ ولا يُدَّعى علمٌ بها**، **ولا تُعرَض على أنّها «سارٍ».**
 *
 * # والمدّةُ تُكتب ساعاتٍ وأيّاماً وتُرسَل نهايةً
 *
 * **ولا تُحفظ «يومان» في الجهاز** — **والنهايةُ هي الحقيقةُ في
 * المحرّك** (`ends_at`): **ومدّةٌ محفوظةٌ ثانيةً تفترق عنها يومَ
 * يُعدَّل أحدُهما.**
 */
object OfferStatus {

    // **وهي نصُّ المحرّك حرفاً** (`offers.StatusAt`).
    const val SCHEDULED = "scheduled"
    const val ACTIVE = "active"
    const val STOPPED = "stopped"
    const val EXPIRED = "expired"

    /** **الحالُ نصّاً عربيّاً** — **ولا يُطبَع رمزٌ آليّ.** */
    fun text(ctx: Context, status: String): String = ctx.getString(
        when (status) {
            SCHEDULED -> R.string.offer_status_scheduled
            ACTIVE -> R.string.offer_status_active
            STOPPED -> R.string.offer_status_stopped
            EXPIRED -> R.string.offer_status_expired
            else -> R.string.offer_status_unknown
        },
    )

    /**
     * **أيُعرَض زرُّ الإيقاف؟**
     *
     * **ولا يُعرَض على منتهٍ ولا على مُنزَل** — **وزرٌّ يُضغط ولا يقع
     * شيءٌ يُقرأ معطوباً**: **والمنتهي انتهى وحدَه.**
     */
    fun canStop(status: String): Boolean = status == ACTIVE || status == SCHEDULED

    /**
     * **أيُنقص السعرَ الآن؟**
     *
     * **والسارِي وحدَه** — **والمجدولُ وعدٌ لم يحلّ بعد.**
     */
    fun discounting(status: String): Boolean = status == ACTIVE
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مدّةُ العرض — تُكتب ساعاتٍ وتُرسَل لحظةَ نهاية** (`MO-02`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وصاحبُ المطعم يقول «يومين»** — **ولا يقول «حتّى الخميس ١١:٥٩».**
 *
 * **ولا تُحسب صلاحيّةٌ هنا** — **الحسابُ ترجمةٌ فقط**: **من ساعاتٍ إلى
 * لحظة.** **والحكمُ في الخادم** (`offers.StatusAt`, `LiveCond`).
 */
object OfferDuration {

    /** **مددٌ تُضغط ضغطةً** — **ولا يُكتب تاريخٌ بالأصابع.** */
    val PRESET_HOURS = listOf(1, 3, 6, 12, 24, 48, 72, 168)

    /** **وأدناها ساعةٌ** — **وعرضٌ لدقيقةٍ لا يراه أحد.** */
    const val MIN_HOURS = 1

    /** **وأقصاها شهر** — **وما بعدَه يُعاد إنشاؤه بقرارٍ جديد.** */
    const val MAX_HOURS = 24 * 30

    /**
     * valid **أمدّةٌ تقع؟**
     *
     * **وتُفحص في الشاشة لتُقال في الحال** — **ولا تُغني عن الخادم**:
     * **الخادمُ يردّ `bad_offer_window` وهو الحَكَم** (`ErrBadWindow`).
     */
    fun valid(hours: Int): Boolean = hours >= MIN_HOURS && hours <= MAX_HOURS

    /**
     * endsAtMillis **لحظةُ النهاية من مدّةٍ بالساعات.**
     *
     * **و`now` يُمرَّر ليُقاس** — **ولا تُقرأ الساعةُ داخل الدالّة.**
     */
    fun endsAtMillis(now: Long, hours: Int): Long = now + hours.toLong() * 3_600_000L

    /**
     * **نصُّ المدّة** — **«٣ ساعات» و«يومان» لا «٤٨ ساعة».**
     *
     * **ومن قرأ «١٦٨ ساعة» حسبها في رأسه.**
     */
    fun label(ctx: Context, hours: Int): String = when {
        hours % 24 == 0 && hours >= 24 ->
            ctx.resources.getQuantityString(R.plurals.offer_days, hours / 24, hours / 24)
        else -> ctx.resources.getQuantityString(R.plurals.offer_hours, hours, hours)
    }
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ عرضٍ — يقرؤها المتجرُ والمندوب** (`MO-01`، `RO-03`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ونسختان تفترقان يوماً** — **فتقول إحداهما «ساري» وتطبع الأخرى
 * رمزاً آليّا.** **والرسمُ واحدٌ لأنّ المقروءَ واحد.**
 *
 * **ولا تعرف نداءً ولا نطاقاً** — **تعرض ما يُمرَّر وتنادي ما يُعطى**:
 * **والفرقُ بين البابين نطاقٌ لا رسم.**
 *
 * **والسعران يجيئان محسوبين من المحرّك** — **ولا يُضربان هنا.**
 */
@Composable
fun OfferCard(
    itemName: String,
    status: String,
    priceBefore: Long,
    priceAfter: Long,
    percent: Int?,
    /**
     * **أو خصمٌ بمبلغٍ ثابت** — بديلُ النسبة لا رفيقُها
     * (قرارُ المالك ٢٠٢٦-٠٩-٣٠). **وأحدُهما فارغٌ دائماً.**
     */
    amount: Long? = null,
    stopping: Boolean,
    onStop: () -> Unit,
    /** **ينتهي متى** — يُكتب في سطر المدّة. وفارغٌ: لا يُقال. */
    endsAt: String? = null,
    /** **يبدأ متى** (٢٠٢٦-١٠-٢٩) — «من … إلى …». وفارغٌ: «حتّى …» وحدَها. */
    startsAt: String? = null,
) {
    val ctx = LocalContext.current
    // ══════════════════════════════════════════════════════════════════
    //  **بطاقةٌ احترافيّة** (طلبُ المالك ٢٠٢٦-١٠-٢٩: «إعادة تصميم كامل»)
    // ══════════════════════════════════════════════════════════════════
    //
    // **شارةُ النوع** (٪ أو ل.س) ثمّ الاسمُ والخصم، **ورقاقةُ الحال بلونها**؛
    // ثمّ السعرُ الجديدُ كبيراً والقديمُ مشطوباً، ثمّ سطرُ المدّة،
    // ثمّ زرُّ الإيقاف حيث يُفيد. **بطاقةٌ بيضاءُ بحدٍّ رفيع** كبطاقات «متجري».
    val live = OfferStatus.discounting(status)
    val tint = if (live) Rahal.colors.brand else Rahal.colors.inkMuted
    val saved = (priceBefore - priceAfter).coerceAtLeast(0)
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(Rahal.space.md),
    ) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier
                    .size(OfferBadgeSize)
                    .clip(Rahal.shape.sm)
                    .background(tint.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    stringResource(if (amount != null && percent == null) R.string.ow_unit_syp else R.string.ow_unit_percent),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = tint,
                )
            }
            Spacer(Modifier.width(Rahal.space.sm))
            Column(Modifier.weight(1f)) {
                // **ولا قصّ** — الاختصارُ يقصّ نقاطَ ما تحت السطر وأوّلَ حرف (٢٠٢٦-١٠-١٤).
                Text(
                    text = itemName,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                    color = Rahal.colors.ink,
                    maxLines = 2,
                    overflow = TextOverflow.Visible,
                )
                val disc = when {
                    percent != null -> stringResource(R.string.ow_disc_percent, percent.toString())
                    amount != null -> stringResource(R.string.ow_disc_amount, money(amount))
                    else -> ""
                }
                if (disc.isNotEmpty()) {
                    Text(disc, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Bold, color = tint)
                }
            }
            Spacer(Modifier.width(Rahal.space.sm))
            // **والحالُ بلفظها لا برمزها** — ولونُها يُفرز الصفَّ بالعين.
            val tone = statusColor(status)
            Text(
                text = OfferStatus.text(ctx, status),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Bold,
                color = tone,
                maxLines = 1,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .background(tone.copy(alpha = 0.12f))
                    .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
            )
        }

        Spacer(Modifier.height(Rahal.space.md))
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Bottom) {
            // **والسعرُ الجديدُ هو الرقم** — والقديمُ صغيرٌ مشطوب.
            Text(
                text = money(priceAfter),
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = if (live) Rahal.colors.success else Rahal.colors.ink,
            )
            Spacer(Modifier.width(Rahal.space.sm))
            Text(
                text = money(priceBefore),
                style = MaterialTheme.typography.bodySmall,
                color = Rahal.colors.inkMuted,
                textDecoration = TextDecoration.LineThrough,
                modifier = Modifier.weight(1f),
            )
            // **والموقوفُ والمنتهي لا «يوفّران» شيئاً** (`OFFER-EXP`).
            if (saved > 0 && (live || status == OfferStatus.SCHEDULED)) {
                Text(
                    text = stringResource(R.string.offer_saves, money(saved)),
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = if (live) Rahal.colors.success else Rahal.colors.inkMuted,
                )
            }
        }

        // **والمدّةُ من … إلى …** — بتوقيت دمشق كما أرسلها المحرّك.
        val from = startsAt?.let { offerTime(it) }
        val to = endsAt?.let { offerTime(it) }
        val period = when {
            from != null && to != null -> stringResource(R.string.ow_period_from_to, from, to)
            to != null -> stringResource(R.string.ow_period_until, to)
            else -> null
        }
        if (period != null) {
            Spacer(Modifier.height(Rahal.space.sm))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    painterResource(R.drawable.ic_time),
                    contentDescription = null,
                    tint = Rahal.colors.inkMuted,
                    modifier = Modifier.size(OfferIconSize),
                )
                Spacer(Modifier.width(Rahal.space.xs))
                Text(period, style = MaterialTheme.typography.bodySmall, color = Rahal.colors.inkMuted)
            }
        }

        // **وزرُّ الإيقاف حيث يُفيد** — **ولا يُعرَض على منتهٍ ولا موقوف.**
        if (OfferStatus.canStop(status)) {
            Spacer(Modifier.height(Rahal.space.md))
            HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
            Spacer(Modifier.height(Rahal.space.sm))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                RahalOutlineButton(onClick = onStop, enabled = !stopping, tone = Tone.Danger) {
                    Icon(
                        painterResource(R.drawable.ic_power),
                        contentDescription = null,
                        modifier = Modifier.size(OfferIconSize),
                    )
                    Spacer(Modifier.width(Rahal.space.xs))
                    Text(stringResource(R.string.ow_stop_do))
                }
            }
        }
    }
}

/** **لونُ الحال** — فعّالٌ أخضر، مجدولٌ برتقاليّ، موقوفٌ أحمر، ومنتهٍ رماديّ. */
@Composable
private fun statusColor(status: String): Color = when (status) {
    OfferStatus.ACTIVE -> Rahal.colors.success
    OfferStatus.SCHEDULED -> Rahal.colors.accent
    OfferStatus.STOPPED -> Rahal.colors.danger
    else -> Rahal.colors.inkMuted
}

/** **شارةُ نوع الخصم** في رأس البطاقة. */
private val OfferBadgeSize = 44.dp

/** **أيقونةُ سطرٍ أو زرّ** داخل البطاقة. */
private val OfferIconSize = 18.dp

/** **لحظةٌ تُقرأ** — يومٌ وتاريخٌ وساعةٌ بتوقيت دمشق. */
internal fun offerTime(iso: String): String? {
    val t = runCatching { java.time.Instant.parse(iso) }.getOrNull() ?: return null
    val fmt = java.time.format.DateTimeFormatter.ofPattern("EEEE d MMMM، h:mm a", java.util.Locale("ar"))
        .withZone(java.time.ZoneId.of("Asia/Damascus"))
    return fmt.format(t)
}

/**
 * **«ينتهي …» بوقتٍ مطلق** — يومٌ وساعةٌ بتوقيت دمشق. **ولا يُقرأ زمنُ الجهاز**
 * (`AB-05`): الحالُ من المحرّك، **وهذا وقتٌ يُقرأ كما أرسله.**
 */
internal fun remainingLabel(ctx: android.content.Context, endsAt: String): String? {
    val end = runCatching { java.time.Instant.parse(endsAt) }.getOrNull() ?: return null
    val fmt = java.time.format.DateTimeFormatter.ofPattern("EEEE h:mm a", java.util.Locale("ar"))
        .withZone(java.time.ZoneId.of("Asia/Damascus"))
    return ctx.getString(R.string.offer_ends_at, fmt.format(end))
}
