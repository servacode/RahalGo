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
import com.rahalgo.ui.hours.CheckBadge
import com.rahalgo.ui.hours.HoursPlan
import com.rahalgo.ui.hours.HoursPlanner
import com.rahalgo.ui.hours.NightHoursDialog
import com.rahalgo.ui.hours.opensAtNight
import com.rahalgo.ui.hours.sameHours
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
 * - **مخطّطُ الدوام وبطاقتا الاختيار** انتقلا إلى `com.rahalgo.ui.hours` (٢٠٢٦-١٠-٣٠) —
 *   يستعملهما المندوبُ أيضاً لضبط دوام متجر عميله.
 */

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
// ٣ · صفحاتُ «متجري» بالقِطَع نفسِها
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
        NightHoursDialog(days, onDismiss = { nightCheck = null }) { nightCheck = null; vm.saveHours(days) }
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
