package com.rahalgo.merchant.store

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import android.annotation.SuppressLint
import android.os.SystemClock
import com.google.android.gms.location.LocationServices
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import com.rahalgo.map.Here
import com.rahalgo.map.LiveFix
import com.rahalgo.map.LocationServiceEffect
import com.rahalgo.map.rememberLocationEnabler
import com.rahalgo.ui.Locating
import com.rahalgo.ui.LocatingText
import com.rahalgo.ui.locationServiceEnabled
import kotlinx.coroutines.delay
import org.maplibre.android.geometry.LatLng
import kotlin.math.abs
import kotlin.math.ceil

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قِطَعُ الإعداد المشتركة — للمعالج ولصفحات «متجري» معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظاتُ المالك السبع ٢٠٢٦-١٠-٢٨.) **والقطعةُ الواحدةُ في الموضعين** — فما
 * يراه في المعالج يراه حين يعدّل لاحقاً، **ولا يتعلّم شاشتين لشيءٍ واحد.**
 *
 * - **شبكةُ الأقسام**: ثلاثةُ مربّعاتٍ متساويةٍ بصورها وبحثٌ فوقها.
 * - **بطاقةُ الخريطة**: خريطةٌ صغيرةٌ بدبّوس المتجر تفتح الملتقطَ الواحد.
 * - **بطاقتا اختيارٍ بأيقونة**: الاستلامُ والترجيع.
 * - **مخطّطُ الدوام**: «٢٤ ساعة» أو «ساعات محددة»، سطرٌ لكلّ يوم، وسؤالُ الجمعة.
 */

/** **الجمعة** — كما يرقّمها المحرّك (٠ الأحد … ٦ السبت). */
internal const val FRIDAY = 5

// ══════════════════════════════════════════════════════════════════════
// ١ · شبكةُ أقسام السوق
// ══════════════════════════════════════════════════════════════════════

/**
 * **ثلاثةُ مربّعاتٍ متساويةٍ في كلّ سطر** (ملاحظةُ المالك ٢: «الأقسام بشعة جداً،
 * عشوائيّة») — صورةُ القسم والاسمُ تحتها، **والمختارُ بحدّ العلامة وعلامةِ صحّ.**
 * **وبحثٌ فوقها** يصفّي بالاسم.
 */
@Composable
internal fun SectionGrid(
    all: List<PlatformSectionRef>,
    picked: Set<String>,
    enabled: Boolean = true,
    onToggle: (String) -> Unit,
) {
    var query by rememberSaveable { mutableStateOf("") }
    OutlinedTextField(
        value = query,
        onValueChange = { query = it.take(40) },
        placeholder = { Text(stringResource(R.string.wz_sections_search)) },
        leadingIcon = {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_search),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
            )
        },
        trailingIcon = if (query.isNotEmpty()) {
            {
                IconButton(onClick = { query = "" }) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_close),
                        contentDescription = stringResource(com.rahalgo.ui.R.string.act_cancel),
                        tint = Rahal.colors.inkMuted,
                    )
                }
            }
        } else {
            null
        },
        singleLine = true,
        shape = Rahal.shape.md,
        colors = OutlinedTextFieldDefaults.colors(
            unfocusedContainerColor = Rahal.colors.field,
            focusedContainerColor = Rahal.colors.field,
            unfocusedBorderColor = Color.Transparent,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(Rahal.space.sm))
    Text(
        stringResource(R.string.wz_sections_picked, picked.size),
        color = if (picked.isEmpty()) Rahal.colors.inkMuted else Rahal.colors.brand,
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.bodyMedium,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    val q = query.trim()
    val shown = if (q.isEmpty()) all else all.filter { it.name.contains(q, ignoreCase = true) }
    if (shown.isEmpty()) {
        Text(
            stringResource(R.string.wz_sections_no_match),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.padding(vertical = Rahal.space.md),
        )
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
        shown.chunked(3).forEach { row ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                row.forEach { sec ->
                    SectionTile(
                        sec = sec,
                        on = sec.id in picked,
                        enabled = enabled,
                        modifier = Modifier.weight(1f),
                    ) { onToggle(sec.id) }
                }
                // **والسطرُ الناقصُ لا يمدّ مربّعاته** — تبقى بعرض أخواتها.
                repeat(3 - row.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
}

@Composable
private fun SectionTile(
    sec: PlatformSectionRef,
    on: Boolean,
    enabled: Boolean,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    Column(
        modifier
            .clip(Rahal.shape.md)
            .clickable(enabled = enabled, role = Role.Checkbox, onClick = onClick)
            .padding(Rahal.space.xs),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(1f),
        ) {
            // **وقسمٌ بلا صورةٍ يُرسم بحرفه الأوّل على أرضٍ باهتة** (`RemoteImage`).
            com.rahalgo.ui.RemoteImage(
                url = com.rahalgo.ui.AppCore.get().media(
                    sec.imageThumbUrl?.takeIf { it.isNotBlank() } ?: sec.imageUrl,
                ),
                name = sec.name,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(Rahal.shape.md)
                    .border(
                        if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                        if (on) Rahal.colors.brand else Rahal.colors.line,
                        Rahal.shape.md,
                    ),
            )
            if (on) {
                CheckBadge(
                    Modifier
                        .align(Alignment.TopEnd)
                        .padding(Rahal.space.xs),
                )
            }
        }
        Spacer(Modifier.height(Rahal.space.xs))
        Text(
            sec.name,
            color = if (on) Rahal.colors.brand else Rahal.colors.ink,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

/** **علامةُ «مختار»** — دائرةٌ بلون العلامة فيها صحّ، في زاوية المربّع أو البطاقة. */
@Composable
internal fun CheckBadge(modifier: Modifier = Modifier) {
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

// ══════════════════════════════════════════════════════════════════════
// ٢ · بطاقةُ الخريطة
// ══════════════════════════════════════════════════════════════════════
/**
 * **بطاقةُ موقع المتجر** — الخريطةُ الصغيرةُ المشتركة (`map.PointMapCard`)، نفسُها عند
 * المندوب في «أضف عميلاً» (طلبُ المالك ٢٠٢٦-١٠-٢٩).
 */
@Composable
internal fun StoreMapCard(vm: StoreViewModel, enabled: Boolean) {
    com.rahalgo.map.PointMapCard(
        lat = vm.pickedLat ?: vm.store?.lat,
        lng = vm.pickedLng ?: vm.store?.lng,
        pending = vm.pickedLat != null,
        enabled = enabled,
        onPoint = { la, ln -> vm.setPoint(la, ln) },
    )
}

// ══════════════════════════════════════════════════════════════════════
// ٣ · بطاقتا اختيارٍ بأيقونة
// ══════════════════════════════════════════════════════════════════════

/** **خيارٌ واحدٌ في البطاقتين** — أيقونتُه وعنوانُه وسطرُ شرحه. */
internal class IconChoice(
    val icon: Int,
    val title: String,
    val sub: String,
    val selected: Boolean,
    val onClick: () -> Unit,
)

/**
 * **بطاقتان كبيرتان متجاورتان** (ملاحظةُ المالك ٥) — أيقونةٌ في دائرة، وعنوان،
 * وسطرُ شرح؛ **والمختارةُ بحدّ العلامة وعلامةِ صحّ.** والبطاقتان بطولٍ واحد.
 */
@Composable
internal fun IconChoicePair(first: IconChoice, second: IconChoice, enabled: Boolean = true) {
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
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                c.sub,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
                textAlign = TextAlign.Center,
            )
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
// ٤ · مخطّطُ الدوام — للمعالج ولصفحة «ساعات الدوام»
// ══════════════════════════════════════════════════════════════════════

/**
 * **حالُ مخطّط الدوام** (ملاحظتا المالك ٤ و٧).
 *
 * - `mode`: «» لم يختر بعد · «24» مفتوحٌ ٢٤ ساعة · «custom» ساعاتٌ محدّدة.
 * - `friday`: «» لم يُجب · «on» دوام · «off» عطلة — **يُسأل صراحةً في الحالين.**
 * - `draft`: الأيّامُ السبعةُ كما يراها — **والجمعةُ منها تتبع `friday`.**
 *
 * **والحفظُ بنداء الساعات القائم** (`saveHours`) بالأيّام السبعة كما يبنيها `days()`.
 */
@Stable
internal class HoursPlan(saved: List<DayHours>, hasHours: Boolean) {
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
internal fun sameHours(a: List<DayHours>, b: List<DayHours>): Boolean {
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
internal fun dayName(day: Int): String = stringResource(
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
 * **مخطّطُ الدوام** (ملاحظةُ المالك ٧: «كل يوم سطران فيهما ٢٤ ساعة و9:00 ص — 11:00 م
 * معاً — مربك»).
 *
 * **خياران فوق**: «مفتوح ٢٤ ساعة كل الأيام» · «ساعات محددة». **وفي المحدّدة سطرٌ
 * واحدٌ لكلّ يوم**: اسمُه · مفتاحُ دوام/عطلة · وقتُه رقاقةً واحدةً تُضغط فتُفتح نافذةُ
 * الوقت (من/إلى، أو «٢٤ ساعة» لذاك اليوم). **ثمّ سؤالُ الجمعة صراحةً** (ملاحظة ٤) —
 * في الحالين، **ودوامُها يُضبط في سطرها.**
 */
@Composable
internal fun HoursPlanner(plan: HoursPlan, enabled: Boolean = true) {
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
                icon = com.rahalgo.ui.R.drawable.ic_time,
                title = stringResource(R.string.hours_on),
                sub = stringResource(R.string.hours_friday_on_sub),
                selected = plan.friday == "on",
            ) { plan.chooseFriday(true) },
            second = IconChoice(
                icon = com.rahalgo.ui.R.drawable.ic_power,
                title = stringResource(R.string.hours_off),
                sub = stringResource(R.string.hours_friday_off_sub),
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
            RahalTextButton(onClick = onDismiss) { Text(stringResource(com.rahalgo.ui.R.string.act_cancel)) }
        },
    )
}

// ══════════════════════════════════════════════════════════════════════
// ٥ · صفحاتُ «متجري» بالقِطَع نفسِها
// ══════════════════════════════════════════════════════════════════════

/**
 * **أقسامُ السوق — الشبكةُ نفسُها التي في المعالج.**
 *
 * **وتنحفظ لحالها بلا زرّ** (طلبُ المالك ٢٠٢٦-١٠-١٢) — بعد لحظةٍ من آخر ضغطة.
 * **ومتجرٌ أزال كلَّ أقسامه يرى السوقَ كلَّه** كما قبل الاختيار (`saveSections`).
 */
@Composable
internal fun SectionsEditor(vm: StoreViewModel) {
    if (vm.allSections.isEmpty()) return
    var picked by remember(vm.mySections) { mutableStateOf(vm.mySections.map { it.id }.toSet()) }
    val dirty = picked != vm.mySections.map { it.id }.toSet()
    val suspended = vm.store?.status == "suspended"
    androidx.compose.runtime.LaunchedEffect(picked) {
        if (dirty && !suspended) {
            kotlinx.coroutines.delay(600)
            vm.saveSections(picked.toList())
        }
    }
    Text(
        stringResource(R.string.wz_sections_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodyMedium,
    )
    Spacer(Modifier.height(Rahal.space.md))
    SectionGrid(all = vm.allSections, picked = picked, enabled = !suspended) { id ->
        picked = if (id in picked) picked - id else picked + id
    }
    if (picked.isEmpty()) {
        Spacer(Modifier.height(Rahal.space.sm))
        Text(
            stringResource(R.string.store_sections_empty),
            color = Rahal.colors.danger,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

/**
 * **ساعاتُ الدوام — المخطّطُ نفسُه الذي في المعالج** (ملاحظةُ المالك ٧).
 *
 * **والمسوّدةُ محلّيّةٌ حتّى يُضغط «حفظ»**، **والمحرّكُ يريد الأيّامَ السبعةَ معاً**
 * (`SetHours` يردّ ما دونها). **وفتحٌ بعد منتصف الليل يُسأل عنه قبل الحفظ**
 * (٢٠٢٦-١٠-١٣، `opensAtNight`).
 */
@Composable
internal fun HoursEditor(vm: StoreViewModel) {
    if (vm.hours.isEmpty()) return
    val store = vm.store ?: return
    val sorted = vm.hours.sortedBy { it.dayOfWeek }
    val plan = remember(sorted, store.hasHours) { HoursPlan(sorted, store.hasHours) }
    val suspended = store.status == "suspended"
    var nightCheck by remember { mutableStateOf<List<DayHours>?>(null) }
    nightCheck?.let { days ->
        val d = days.first { opensAtNight(it) }
        AlertDialog(
            onDismissRequest = { nightCheck = null },
            title = { Text(stringResource(R.string.hours_night_title)) },
            text = { Text(stringResource(R.string.hours_night_text, clock12(d.openTime), clock12(d.closeTime))) },
            confirmButton = {
                RahalButton(onClick = { nightCheck = null; vm.saveHours(days) }) { Text(stringResource(R.string.hours_night_yes)) }
            },
            dismissButton = {
                RahalOutlineButton(onClick = { nightCheck = null }) { Text(stringResource(R.string.hours_night_fix)) }
            },
        )
    }
    HoursPlanner(plan, enabled = !suspended)
    val days = plan.days()
    val dirty = plan.chosen && (!store.hasHours || !sameHours(days, sorted))
    if (dirty && !suspended) {
        Spacer(Modifier.height(Rahal.space.lg))
        RahalButton(
            onClick = { if (days.any { opensAtNight(it) }) nightCheck = days else vm.saveHours(days) },
            enabled = !vm.saving,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
        ) { Text(stringResource(com.rahalgo.ui.R.string.act_save), fontWeight = FontWeight.Bold) }
    }
    Spacer(Modifier.height(Rahal.space.xl))
}

/**
 * **العنوانُ والموقع** — حقلُ العنوان، **وتحته بطاقةُ الخريطة** (ملاحظةُ المالك ٣)
 * كما في المعالج، والحفظُ تحتهما معاً.
 *
 * **والنقطةُ المختارةُ لا تُحفظ حتّى يُضغط «حفظ»** (`pickedLat`) — فلا تبدو محفوظةً
 * وهي لم تصل المحرّك.
 */
@Composable
internal fun AddressEditor(vm: StoreViewModel) {
    val store = vm.store ?: return
    val suspended = store.status == "suspended"
    var text by remember(store.addressText) { mutableStateOf(store.addressText) }
    val dirty = text.trim() != store.addressText || vm.pickedLat != null
    OutlinedTextField(
        value = text,
        onValueChange = { text = it },
        label = { Text(stringResource(R.string.act_address)) },
        placeholder = { Text(stringResource(R.string.store_address_hint)) },
        shape = Rahal.shape.md,
        enabled = !suspended,
        modifier = Modifier.fillMaxWidth(),
        minLines = 2,
    )
    Spacer(Modifier.height(Rahal.space.md))
    StoreMapCard(vm, enabled = !suspended)
    if (dirty && !suspended) {
        Spacer(Modifier.height(Rahal.space.md))
        RahalButton(
            onClick = { vm.saveAddress(text, vm.pickedLat, vm.pickedLng) },
            enabled = !vm.saving,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
        ) { Text(stringResource(com.rahalgo.ui.R.string.act_save), fontWeight = FontWeight.Bold) }
    }
}
