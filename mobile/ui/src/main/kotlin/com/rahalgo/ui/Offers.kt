package com.rahalgo.ui

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
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
    /** **ينتهي متى** (٢٠٢٦-١٠-٠٩) — «ينتهي بعد ٥ ساعات» تحت السعر. وفارغٌ: لا يُقال. */
    endsAt: String? = null,
) {
    val ctx = LocalContext.current
    // ══════════════════════════════════════════════════════════════════
    //  **وبطاقةٌ تُقرأ من بعيد — لا ثلاثةُ أسطرٍ رماديّة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «تصميم عروض المتجر يجب أن تكون أقوى
    //  بصريّاً من العرض الحالي».)
    //
    // **وكانت سطرين بحجمٍ واحدٍ ولونٍ واحد**: اسمُ الصنف، ثمّ
    // `قبل ← بعد · ٢٠٪` **كلُّه `bodySmall` رماديّ.** **فالخصمُ — وهو
    // كلُّ سببِ وجود البطاقة — أصغرُ ما فيها**، والسعرُ الجديدُ لا
    // يُميَّز عن القديم إلّا بسهمٍ بينهما.
    //
    // **وصارت:**
    //   - **شارةُ خصمٍ ملوّنةٌ في رأس البطاقة** — «٢٠٪» أو «−٥٠٠» بلون
    //     العلامة على أرضيّةٍ خفيفة، **تُقرأ قبل أن يُقرأ الاسم.**
    //   - **والسعرُ الجديدُ بحجم العنوان** وبلون النجاح، **والقديمُ
    //     صغيرٌ مشطوبٌ فوقه** — **لا سهمٌ بين رقمين متساويين.**
    //   - **وسطرُ التوفير بالليرة** — **والمتجرُ يفهم «يوفّر ٥٠٠» أسرعَ
    //     من «٢٠٪»**، وهو ما يقوله لزبونه.
    //   - **وحدٌّ يمينيٌّ بلون الحال** — سارٍ بلون العلامة، ومنتهٍ
    //     رماديّ: **فصفٌّ من عشرِ بطاقاتٍ يُفرَز بالعين بلا قراءة.**
    val live = OfferStatus.discounting(status)
    val edge = if (live) Rahal.colors.brand else Rahal.colors.inkMuted
    val saved = (priceBefore - priceAfter).coerceAtLeast(0)
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, edge.copy(alpha = 0.35f), Rahal.shape.md)
            .padding(12.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // **وشارةُ الخصم أوّلَ ما يقع عليه البصر.**
            val badge = when {
                percent != null -> percent.toString() + "٪−"
                amount != null -> "−" + money(amount)
                else -> ""
            }
            if (badge.isNotEmpty()) {
                Text(
                    text = badge,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                    color = if (live) Rahal.colors.brand else Rahal.colors.inkMuted,
                    modifier = Modifier
                        .clip(Rahal.shape.sm)
                        .background(
                            (if (live) Rahal.colors.brand else Rahal.colors.inkMuted)
                                .copy(alpha = 0.12f),
                        )
                        .padding(horizontal = 8.dp, vertical = 3.dp),
                )
                Spacer(Modifier.width(8.dp))
            }
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
                color = if (live) Rahal.colors.accent else Rahal.colors.inkMuted,
            )
        }

        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.Bottom) {
            // **والسعرُ الجديدُ هو الرقم** — بحجم العنوان ولون النجاح.
            Text(
                text = money(priceAfter),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = if (live) Rahal.colors.success else Rahal.colors.ink,
            )
            Spacer(Modifier.width(8.dp))
            // **والقديمُ مشطوبٌ صغيرٌ** — **ورقمان متساويان في الحجم
            // يجعلان القارئَ يقارن بدل أن يرى.**
            Text(
                text = money(priceBefore),
                style = MaterialTheme.typography.bodySmall,
                color = Rahal.colors.inkMuted,
                textDecoration = TextDecoration.LineThrough,
            )
        }

        // **وما يوفّره بالليرة** — **وهو ما يقوله المتجرُ لزبونه.**
        //
        // **والموقوفُ والمنتهي لا «يوفّران» شيئاً** (`OFFER-EXP`، رُئي على
        // الجهاز): «يوفر ٧٥ ل.س» بجانب «موقوف» يُقرأ عرضاً سارياً.
        if (saved > 0 && (live || status == OfferStatus.SCHEDULED)) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = stringResource(R.string.offer_saves, money(saved)),
                style = MaterialTheme.typography.bodySmall,
                color = if (live) Rahal.colors.success else Rahal.colors.inkMuted,
            )
        }
        // **وزرُّ الإيقاف حيث يُفيد** — **ولا يُعرَض على منتهٍ.**
        if (live && endsAt != null) {
            remainingLabel(ctx, endsAt)?.let {
                Spacer(Modifier.height(4.dp))
                Text(it, style = MaterialTheme.typography.bodySmall, color = Rahal.colors.inkMuted)
            }
        }
        if (OfferStatus.canStop(status)) {
            Spacer(Modifier.height(6.dp))
            RahalTextButton(onClick = onStop, enabled = !stopping) {
                Text(stringResource(R.string.offer_stop), color = Rahal.colors.danger)
            }
        }
    }
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
