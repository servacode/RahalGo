package com.rahalgo.ui

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
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
    stopping: Boolean,
    onStop: () -> Unit,
) {
    val ctx = LocalContext.current
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .padding(12.dp),
    ) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                text = itemName,
                fontWeight = FontWeight.Bold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            // **والحالُ بلفظها لا برمزها** — **و«ساري» وحدَها بلون
            // العلامة**: **فما يُنقص السعرَ الآن يُرى من بعيد.**
            Text(
                text = OfferStatus.text(ctx, status),
                style = MaterialTheme.typography.bodySmall,
                color = if (OfferStatus.discounting(status)) {
                    Rahal.colors.accent
                } else {
                    Rahal.colors.inkMuted
                },
            )
        }
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(
                R.string.offer_price_line, money(priceBefore), money(priceAfter),
            ) + (percent?.let { "  ·  " + it + "٪" } ?: ""),
            style = MaterialTheme.typography.bodySmall,
            color = Rahal.colors.inkMuted,
        )
        // **وزرُّ الإيقاف حيث يُفيد** — **ولا يُعرَض على منتهٍ.**
        if (OfferStatus.canStop(status)) {
            Spacer(Modifier.height(6.dp))
            RahalTextButton(onClick = onStop, enabled = !stopping) {
                Text(stringResource(R.string.offer_stop), color = Rahal.colors.danger)
            }
        }
    }
}
