package com.rahalgo.ui.hours

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مخطّطُ الدوام المشترك — للمتجر وللمندوب معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٣٠: المندوبُ يضبط ساعاتِ دوام متجر عميله **بالشاشة نفسِها**
 * التي في تطبيق المتجر.) فنُقل المخطّطُ من `app-merchant` إلى هنا — **نسخةٌ واحدةٌ
 * لا نسختان** — ويستعمله معالجُ المتجر وصفحةُ «ساعات الدوام» عنده وصفحةُ العميل
 * عند المندوب.
 *
 * - «٢٤ ساعة» أو «ساعات محددة»، سطرٌ لكلّ يوم، وسؤالُ الجمعة صراحةً.
 * - **والأيّامُ تحت «ساعات محددة» مباشرةً** ولا تُخفى إلّا بـ«مفتوح 24 ساعة» (٢٠٢٦-١٠-٢٩).
 */

/** **الجمعة** — كما يرقّمها المحرّك (٠ الأحد … ٦ السبت). */
const val FRIDAY = 5

// ══════════════════════════════════════════════════════════════════════
// ١ · بطاقاتُ الاختيار
// ══════════════════════════════════════════════════════════════════════

/** **علامةُ «مختار»** — دائرةٌ بلون العلامة فيها صحّ، في زاوية المربّع أو البطاقة. */
@Composable
fun CheckBadge(modifier: Modifier = Modifier) {
    Box(
        modifier
            .size(24.dp)
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.brand),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painterResource(R.drawable.ic_check),
            contentDescription = null,
            tint = Rahal.colors.onBrand,
            modifier = Modifier.size(16.dp),
        )
    }
}

/**
 * **بطاقةُ اختيارٍ كبيرة** — دائرةٌ وعنوانٌ وسطرُ شرح، والمختارةُ بحدّ العلامة.
 * (التصميمُ المعتمد: خطوةُ الدوام وما يشبهها.)
 */
@Composable
fun ChoiceCard(
    title: String,
    sub: String,
    selected: Boolean,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(if (selected) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (selected) Rahal.stroke.focus else Rahal.stroke.hair,
                if (selected) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(
            selected = selected,
            onClick = null,
            enabled = enabled,
            colors = RadioButtonDefaults.colors(selectedColor = Rahal.colors.brand),
        )
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(title, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium, color = Rahal.colors.ink)
            if (sub.isNotBlank()) {
                Text(sub, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

/** **خيارٌ واحدٌ في البطاقتين** — أيقونتُه وعنوانُه وسطرُ شرحه. */
class IconChoice(
    val icon: Int,
    val title: String,
    val sub: String,
    val selected: Boolean,
    val onClick: () -> Unit,
)

/**
 * **بطاقتان كبيرتان متجاورتان** — أيقونةٌ في دائرة، وعنوان، وسطرُ شرح؛
 * **والمختارةُ بحدّ العلامة وعلامةِ صحّ.** والبطاقتان بطولٍ واحد.
 */
@Composable
fun IconChoicePair(first: IconChoice, second: IconChoice, enabled: Boolean = true) {
    Row(
        Modifier
            .fillMaxWidth()
            .height(IntrinsicSize.Min),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        IconChoiceCard(first, enabled, Modifier.weight(1f))
        IconChoiceCard(second, enabled, Modifier.weight(1f))
    }
}

@Composable
private fun IconChoiceCard(c: IconChoice, enabled: Boolean, modifier: Modifier) {
    Box(modifier.fillMaxHeight()) {
        Column(
            Modifier
                .fillMaxSize()
                .heightIn(min = 168.dp)
                .clip(Rahal.shape.lg)
                .background(if (c.selected) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
                .border(
                    if (c.selected) Rahal.stroke.focus else Rahal.stroke.hair,
                    if (c.selected) Rahal.colors.brand else Rahal.colors.line,
                    Rahal.shape.lg,
                )
                .clickable(enabled = enabled, role = Role.RadioButton, onClick = c.onClick)
                .padding(horizontal = Rahal.space.md, vertical = Rahal.space.lg),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier
                    .size(56.dp)
                    .clip(Rahal.shape.pill)
                    .background(if (c.selected) Rahal.colors.brand else Rahal.colors.brand.copy(alpha = 0.10f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painterResource(c.icon),
                    contentDescription = null,
                    tint = if (c.selected) Rahal.colors.onBrand else Rahal.colors.brand,
                    modifier = Modifier.size(28.dp),
                )
            }
            Spacer(Modifier.height(Rahal.space.md))
            Text(
                c.title,
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
                textAlign = TextAlign.Center,
            )
            if (c.sub.isNotBlank()) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    c.sub,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    textAlign = TextAlign.Center,
                )
            }
        }
        if (c.selected) {
            CheckBadge(
                Modifier
                    .align(Alignment.TopEnd)
                    .padding(Rahal.space.sm),
            )
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// ٢ · حالُ المخطّط وأدواتُه
// ══════════════════════════════════════════════════════════════════════

/**
 * **حالُ مخطّط الدوام**.
 *
 * - `mode`: «» لم يختر بعد · «24» مفتوحٌ ٢٤ ساعة · «custom» ساعاتٌ محدّدة.
 * - `friday`: «» لم يُجب · «on» دوام · «off» عطلة — **يُسأل صراحةً في الحالين.**
 * - `draft`: الأيّامُ السبعةُ كما يراها — **والجمعةُ منها تتبع `friday`.**
 *
 * **والحفظُ بالأيّام السبعة** كما يبنيها `days()`.
 */
@Stable
class HoursPlan(saved: List<DayHours>, hasHours: Boolean) {
    var mode by mutableStateOf(
        when {
            !hasHours -> ""
            saved.filter { it.dayOfWeek != FRIDAY && !it.closed }.let { o -> o.isNotEmpty() && o.all { it.isOpen24h } } -> "24"
            else -> "custom"
        },
    )
    var friday by mutableStateOf(
        when {
            !hasHours -> ""
            saved.firstOrNull { it.dayOfWeek == FRIDAY }?.closed == true -> "off"
            else -> "on"
        },
    )

    // **ولمن لم يضبط دوامَه بعد: ٩ص–١١م لكلّ يوم** — والجمعةُ حتّى يجيب.
    var draft by mutableStateOf(
        if (hasHours) {
            saved
        } else {
            saved.map { it.copy(closed = false, open24h = false, openTime = "09:00", closeTime = "23:00") }
        },
    )

    val chosen: Boolean get() = mode.isNotEmpty() && friday.isNotEmpty()

    /** **أضبط ساعاتِ الجمعة بيده؟** — وإلّا تبعت الوضعَ (٢٤ ساعة أو ساعات بقيّة الأسبوع). */
    private var fridayEdited = hasHours

    fun chooseMode(m: String) {
        if (m == "custom" && mode != "custom") {
            // **ومن ترك «٢٤ ساعة» إلى «ساعات محددة» يبدأ من ٩–١١** لا من ٢٤ ساعة لكلّ يوم.
            val others = draft.filter { it.dayOfWeek != FRIDAY && !it.closed }
            if (others.isNotEmpty() && others.all { it.isOpen24h }) {
                draft = draft.map {
                    if (it.isOpen24h && (it.dayOfWeek != FRIDAY || !fridayEdited)) it.with24h(false) else it
                }
            }
        }
        if (m == "24" && !fridayEdited) {
            draft = draft.map { if (it.dayOfWeek == FRIDAY) it.with24h(true) else it }
        }
        mode = m
    }

    fun chooseFriday(on: Boolean) {
        friday = if (on) "on" else "off"
        if (on) {
            val fri = draft.firstOrNull { it.dayOfWeek == FRIDAY }
            if (fri != null && (fri.closed || fri.openTime.isBlank() || (!fridayEdited && mode == "24" && !fri.isOpen24h))) {
                // **والجمعةُ تبدأ بما في بقيّة الأسبوع** — ٢٤ ساعة أو ساعاتُ أوّل يوم دوام.
                val like = draft.firstOrNull { it.dayOfWeek != FRIDAY && !it.closed }
                val auto = when {
                    mode == "24" || like == null -> fri.with24h(true)
                    else -> fri.copy(closed = false, open24h = like.open24h, openTime = like.openTime, closeTime = like.closeTime)
                }
                draft = draft.map { if (it.dayOfWeek == FRIDAY) auto else it }
            }
        }
    }

    fun update(d: DayHours) {
        if (d.dayOfWeek == FRIDAY) fridayEdited = true
        draft = draft.map { if (it.dayOfWeek == d.dayOfWeek) d else it }
    }

    /** **«طبّق نفس الساعات على كل الأيام»** — على أيّام الدوام وحدَها، والعطلةُ تبقى عطلة. */
    fun applyAll(sample: DayHours) {
        draft = draft.map {
            if (it.closed) it else it.copy(open24h = sample.open24h, openTime = sample.openTime, closeTime = sample.closeTime)
        }
    }

    /** **الأيّامُ السبعةُ كما تُحفظ.** */
    fun days(): List<DayHours> = draft.map { d ->
        when {
            d.dayOfWeek == FRIDAY -> if (friday == "off") d.copy(closed = true) else d.copy(closed = false)
            mode == "24" -> d.with24h(true)
            else -> d
        }
    }
}

/** **أيتساوى الدوامان معنًى؟** — فلا يظهر «حفظ» على ما لم يتغيّر. */
fun sameHours(a: List<DayHours>, b: List<DayHours>): Boolean {
    if (a.size != b.size) return false
    val bb = b.associateBy { it.dayOfWeek }
    return a.all { x ->
        val y = bb[x.dayOfWeek] ?: return@all false
        x.closed == y.closed && (
            x.closed || (x.isOpen24h && y.isOpen24h) ||
                (!x.isOpen24h && !y.isOpen24h && x.openTime.take(5) == y.openTime.take(5) && x.closeTime.take(5) == y.closeTime.take(5))
            )
    }
}

@Composable
fun dayName(day: Int): String = stringResource(
    when (day) {
        0 -> R.string.days_sun
        1 -> R.string.days_mon
        2 -> R.string.days_tue
        3 -> R.string.days_wed
        4 -> R.string.days_thu
        5 -> R.string.days_fri
        else -> R.string.days_sat
    },
)

/** **وقتُ اليوم في سطر** — «٢٤ ساعة» أو «9:00 ص — 11:00 م». */
@Composable
private fun rangeText(d: DayHours): String =
    if (d.isOpen24h) stringResource(R.string.hours_day_24)
    else stringResource(R.string.home_val_hours_range, clock12(d.openTime.take(5)), clock12(d.closeTime.take(5)))

/**
 * **قيمةُ الدوام في سطرٍ واحد** — «مفتوح 24 ساعة» أو «9:00 ص – 11:00 م» أو «ساعات مختلفة»؛
 * **أو فراغٌ إن لم يُضبط بعد.** (سطرُ «ساعات الدوام» في «متجري» وفي صفحة العميل عند المندوب.)
 */
@Composable
fun hoursSummary(hasHours: Boolean, hours: List<DayHours>): String? {
    if (!hasHours) return null
    val open = hours.filter { !it.closed }
    if (open.isEmpty()) return null
    if (open.all { it.isOpen24h }) return stringResource(R.string.hours_open_24h)
    val first = open.first()
    return if (open.all { !it.isOpen24h && it.openTime.take(5) == first.openTime.take(5) && it.closeTime.take(5) == first.closeTime.take(5) }) {
        stringResource(R.string.home_val_hours_range, clock12(first.openTime.take(5)), clock12(first.closeTime.take(5)))
    } else {
        stringResource(R.string.home_val_hours_mixed)
    }
}

/**
 * **«13:00» ⇒ «1:00 م»** — الساعةُ كما يقرؤها صاحبُ المتجر في منبّه هاتفه
 * (٢٠٢٦-١٠-١٣). والأرقامُ لاتينيّةٌ كبقيّة الشاشة.
 */
fun clock12(hhmm: String): String {
    val h = hhmm.take(2).toIntOrNull() ?: return hhmm
    val m = hhmm.drop(3).take(2).ifBlank { "00" }
    val h12 = when {
        h == 0 -> 12
        h > 12 -> h - 12
        else -> h
    }
    return "$h12:$m " + if (h < 12) "ص" else "م"
}

/**
 * **أيفتح بعد منتصف الليل؟** (٢٠٢٦-١٠-١٣) — يُسأل عنه قبل الحفظ (`NightHoursDialog`).
 * **و«مفتوح ٢٤ ساعة» ليس غلطاً** — ٠٠:٠٠ ← ٠٠:٠٠ مقصودة (٢٠٢٦-١٠-١٨).
 */
fun opensAtNight(d: DayHours): Boolean {
    val open = if (d.isOpen24h) -1 else d.openTime.take(2).toIntOrNull() ?: 9
    val close = d.closeTime.take(2).toIntOrNull() ?: 23
    return !d.closed && open in 0..5 && !(open == 0 && close >= 23)
}

/**
 * **«هل يفتح متجرك في الليل؟»** — قبل حفظ دوامٍ يفتح بعد منتصف الليل (`opensAtNight`).
 * «نعم، صحيح» يحفظ، و«لا، أريد التعديل» يعيده إلى المخطّط.
 */
@Composable
fun NightHoursDialog(days: List<DayHours>, onDismiss: () -> Unit, onConfirm: () -> Unit) {
    val first = days.first { opensAtNight(it) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.hours_night_title)) },
        text = { Text(stringResource(R.string.hours_night_text, clock12(first.openTime), clock12(first.closeTime))) },
        confirmButton = {
            RahalButton(onClick = onConfirm) { Text(stringResource(R.string.hours_night_yes)) }
        },
        dismissButton = {
            RahalOutlineButton(onClick = onDismiss) { Text(stringResource(R.string.hours_night_fix)) }
        },
    )
}

// ══════════════════════════════════════════════════════════════════════
// ٣ · المخطّط
// ══════════════════════════════════════════════════════════════════════

/**
 * **مخطّطُ الدوام** — خياران فوق: «مفتوح ٢٤ ساعة كل الأيام» · «ساعات محددة».
 * **وتحت المحدّدة سطرٌ واحدٌ لكلّ يوم**: اسمُه · مفتاحُ دوام/عطلة · وقتُه رقاقةً تُضغط
 * فتُفتح نافذةُ الوقت. **ثمّ سؤالُ الجمعة صراحةً** في الحالين، ودوامُها في سطرها.
 */
@Composable
fun HoursPlanner(plan: HoursPlan, enabled: Boolean = true) {
    var editing by remember { mutableStateOf<DayHours?>(null) }
    var applyingAll by remember { mutableStateOf(false) }

    ChoiceCard(
        title = stringResource(R.string.hours_mode_24),
        sub = "",
        selected = plan.mode == "24",
        enabled = enabled,
    ) { plan.chooseMode("24") }
    Spacer(Modifier.height(Rahal.space.sm))
    ChoiceCard(
        title = stringResource(R.string.wz_hours_custom),
        sub = "",
        selected = plan.mode == "custom",
        enabled = enabled,
    ) { plan.chooseMode("custom") }

    // **والأيّامُ تحت «ساعات محددة» مباشرةً وساعاتُها ظاهرة** (تجربةُ المالك ٢٠٢٦-١٠-٢٩) —
    // ولا تُخفى إلّا بـ«مفتوح 24 ساعة». **وضغطُ يومٍ يختار «ساعات محددة».**
    if (plan.mode != "24") {
        Spacer(Modifier.height(Rahal.space.xs))
        Column(
            Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.canvas)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
        ) {
            val week = plan.draft.filter { it.dayOfWeek != FRIDAY }.sortedBy { (it.dayOfWeek + 1) % 7 }
            week.forEachIndexed { i, d ->
                if (i > 0) HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
                DayRow(
                    d = d,
                    enabled = enabled,
                    onToggle = { open -> plan.chooseMode("custom"); plan.update(d.copy(closed = !open)) },
                    onTime = { plan.chooseMode("custom"); editing = d },
                )
            }
        }
        Spacer(Modifier.height(Rahal.space.sm))
        RahalOutlineButton(
            onClick = { plan.chooseMode("custom"); applyingAll = true },
            enabled = enabled && plan.draft.any { !it.closed },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.hours_apply_same), fontWeight = FontWeight.Bold) }
    }

    // ── سؤالُ الجمعة — صريحاً في الحالين ──
    if (plan.mode.isNotEmpty()) {
        Spacer(Modifier.height(Rahal.space.lg))
        Text(
            stringResource(R.string.hours_friday_q),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        IconChoicePair(
            first = IconChoice(
                icon = R.drawable.ic_time,
                title = stringResource(R.string.hours_on),
                sub = "",
                selected = plan.friday == "on",
            ) { plan.chooseFriday(true) },
            second = IconChoice(
                icon = R.drawable.ic_power,
                title = stringResource(R.string.hours_off),
                sub = "",
                selected = plan.friday == "off",
            ) { plan.chooseFriday(false) },
            enabled = enabled,
        )
        if (plan.friday == "on") {
            plan.draft.firstOrNull { it.dayOfWeek == FRIDAY }?.let { fri ->
                Spacer(Modifier.height(Rahal.space.sm))
                Box(
                    Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.md)
                        .background(Rahal.colors.canvas)
                        .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
                ) {
                    DayRow(
                        d = fri.copy(closed = false),
                        enabled = enabled,
                        onToggle = null,
                        onTime = { editing = fri.copy(closed = false) },
                    )
                }
            }
        }
    }

    editing?.let { d ->
        DayTimeDialog(
            title = dayName(d.dayOfWeek),
            start = d,
            onDismiss = { editing = null },
        ) { done -> plan.update(done); editing = null }
    }
    if (applyingAll) {
        val sample = plan.draft.firstOrNull { it.dayOfWeek != FRIDAY && !it.closed }
            ?: plan.draft.first()
        DayTimeDialog(
            title = stringResource(R.string.hours_apply_same_title),
            start = sample.copy(closed = false),
            onDismiss = { applyingAll = false },
        ) { done -> plan.applyAll(done); applyingAll = false }
    }
}

/**
 * **سطرُ يوم** — اسمُه · مفتاحُ دوام/عطلة · رقاقةُ وقته. **ويومُ العطلة رماديٌّ
 * يقول «عطلة».** وبلا مفتاحٍ (`onToggle` فارغ) لسطر الجمعة: دوامُها سُئل فوقه.
 */
@Composable
private fun DayRow(
    d: DayHours,
    enabled: Boolean,
    onToggle: ((Boolean) -> Unit)?,
    onTime: () -> Unit,
) {
    val open = !d.closed
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 56.dp)
            .background(if (open) Color.Transparent else Rahal.colors.field)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            dayName(d.dayOfWeek),
            color = if (open) Rahal.colors.ink else Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            modifier = Modifier.weight(1f),
        )
        if (onToggle != null) {
            Switch(
                checked = open,
                onCheckedChange = onToggle,
                enabled = enabled,
                colors = SwitchDefaults.colors(checkedTrackColor = Rahal.colors.brand),
            )
            Spacer(Modifier.width(Rahal.space.sm))
        }
        if (open) {
            Text(
                rangeText(d),
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                maxLines = 1,
                modifier = Modifier
                    .widthIn(min = 140.dp)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.brand.copy(alpha = 0.08f))
                    .border(Rahal.stroke.hair, Rahal.colors.brand.copy(alpha = 0.35f), Rahal.shape.pill)
                    .clickable(enabled = enabled, role = Role.Button, onClick = onTime)
                    .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
            )
        } else {
            Text(
                stringResource(R.string.hours_off),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .widthIn(min = 140.dp)
                    .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
            )
        }
    }
}

/**
 * **نافذةُ وقت اليوم** — «٢٤ ساعة» أو «ساعات محددة» من/إلى بمنتقي النظام (`TimeField`).
 * **ولا يُحفظ شيءٌ منها** — تعيد اليومَ إلى المخطّط، والحفظُ بزرّه.
 */
@Composable
private fun DayTimeDialog(
    title: String,
    start: DayHours,
    onDismiss: () -> Unit,
    onDone: (DayHours) -> Unit,
) {
    var all24 by remember { mutableStateOf(start.isOpen24h) }
    var from by remember { mutableStateOf(if (start.isOpen24h) "09:00" else start.openTime.take(5).ifBlank { "09:00" }) }
    var to by remember { mutableStateOf(if (start.isOpen24h) "23:00" else start.closeTime.take(5).ifBlank { "23:00" }) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title, fontWeight = FontWeight.Bold) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                ChoiceCard(
                    title = stringResource(R.string.hours_day_24),
                    sub = "",
                    selected = all24,
                ) { all24 = true }
                ChoiceCard(
                    title = stringResource(R.string.hours_day_specific),
                    sub = "",
                    selected = !all24,
                ) { all24 = false }
                if (!all24) {
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .padding(top = Rahal.space.sm),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                    ) {
                        Text(stringResource(R.string.hours_from), color = Rahal.colors.inkMuted)
                        TimeField(from) { from = it }
                        Text(stringResource(R.string.hours_to), color = Rahal.colors.inkMuted)
                        TimeField(to) { to = it }
                    }
                }
            }
        },
        confirmButton = {
            RahalButton(
                onClick = {
                    onDone(
                        if (all24) start.with24h(true)
                        else start.copy(closed = false, open24h = false, openTime = from, closeTime = to),
                    )
                },
            ) { Text(stringResource(R.string.hours_done)) }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_cancel)) }
        },
    )
}

// ══════════════════════════════════════════════════════════════════════
// ٤ · حقلُ الوقت
// ══════════════════════════════════════════════════════════════════════

/**
 * **حقلُ وقتٍ `HH:MM`** — صندوقٌ صغيرٌ يفتح منتقي الوقت من النظام.
 *
 * - **والساعةُ تُختار لا تُكتب** (بلاغُ المالك ٢٠٢٦-٠٨-٢٦) — حقلٌ نصّيٌّ حرٌّ كان
 *   يقبل `25:70` فيُحفظ متجرٌ لا يُفتح أبداً.
 * - **والمحفوظُ بأرقامٍ لاتينيّةٍ مهما كانت لغةُ الجهاز** (`Locale.ROOT`، ٢٠٢٦-٠٨-٢٩) —
 *   «٠٩:٣٠» يردّها المحرّك `ErrBadHours` على الأيّام السبعة معاً.
 * - **وصباحاً ومساءً لا ٢٤ ساعة** (٢٠٢٦-١٠-١٣) — والمحفوظُ يبقى `HH:mm`.
 * - **وزخرفةُ الأرقام لا تُسقط تطبيقاً**: إن تعذّر لفُّ السياق يُفتح الحوارُ بسياقه الأصليّ.
 */
@Composable
fun TimeField(value: String, onChange: (String) -> Unit) {
    val context = LocalContext.current
    // **والثواني تُقطع** — المحرّكُ يردّ `08:00:00`.
    val shown = value.take(5).ifBlank { "00:00" }
    val picker = {
        val h = shown.substringBefore(':').toIntOrNull()?.coerceIn(0, 23) ?: 0
        val m = shown.substringAfter(':').toIntOrNull()?.coerceIn(0, 59) ?: 0
        val themed = runCatching { latinDigits(context) }.getOrDefault(context)
        android.app.TimePickerDialog(
            themed,
            { _, hh, mm ->
                onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
            },
            h, m, false,
        ).let { dialog ->
            runCatching { dialog.show() }.onFailure {
                // **والسقوطُ الأخير**: حوارٌ بالسياق الأصليّ بلا لفّ.
                android.app.TimePickerDialog(
                    context,
                    { _, hh, mm ->
                        onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
                    },
                    h, m, false,
                ).show()
            }
        }
    }

    Box(
        Modifier
            .width(96.dp)
            .border(1.dp, Rahal.colors.line, Rahal.shape.sm)
            .clickable { picker() }
            .padding(vertical = 8.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = clock12(shown),
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
        )
    }
}

/**
 * **سياقٌ بأرقامٍ لاتينيّة — لنافذة الوقت نفسِها.** و`ar-u-nu-latn` عربيّةٌ بأرقامٍ
 * لاتينيّة: تبقى الكلماتُ عربيّةً ويتبدّل نظامُ الأرقام وحدَه.
 *
 * **ويُلَفُّ السياقُ لفّاً ولا يُبنى من جديد** — `createConfigurationContext` صنع سياقاً
 * بلا نافذةِ نشاطٍ فانهار التطبيقُ بـ`BadTokenException` (٢٠٢٦-٠٨-٢٩).
 */
private fun latinDigits(context: android.content.Context): android.content.Context {
    val config = android.content.res.Configuration(context.resources.configuration)
    config.setLocale(java.util.Locale.forLanguageTag("ar-u-nu-latn"))
    return android.view.ContextThemeWrapper(context, 0).apply {
        applyOverrideConfiguration(config)
    }
}
