package com.rahalgo.customer.address

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.TextLimits
import com.rahalgo.ui.AccountViewModel
import com.rahalgo.ui.PageHeader
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.addressKindLabel

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«عنوان التوصيل» — صفحةٌ واحدة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كصفحة موقع المتجر**، من الأعلى للأسفل: العنوانُ التفصيليُّ (الحقلُ الإلزاميُّ الوحيد) ·
 * الخريطةُ بدبّوسٍ ثابتٍ في الوسط (`PointMapCard`) تحدّد موقعَه وحدَها عند الفتح ثمّ
 * «حدّد موقعي بدقة» · النوع · «شارك موقعي المباشر» · «حفظ العنوان».
 *
 * **والمحرّكُ يريد `area_building` ونقطةً** (`address_handlers.go`) — فالنصُّ يُرسَل
 * منطقةً، والشارعُ والطابقُ فارغان، **والنقطةُ تُرسَل دائماً.**
 */
@Composable
fun AddressPage(
    vm: AccountViewModel,
    /** **عنوانٌ يُعدَّل** — أو فارغٌ لجديد. */
    existing: Address?,
    /** **والجديدُ يصير مختاراً للطلب** — من لوحة الاختيار. */
    selectNew: Boolean,
    onDone: () -> Unit,
) {
    val context = LocalContext.current
    val s = vm.state
    var text by rememberSaveable(existing?.id) { mutableStateOf(initialText(existing)) }
    var kind by rememberSaveable(existing?.id) { mutableStateOf(existing?.kind ?: "home") }
    var lat by rememberSaveable(existing?.id) { mutableStateOf(existing?.lat) }
    var lng by rememberSaveable(existing?.id) { mutableStateOf(existing?.lng) }

    BackHandler { onDone() }

    val canSave = !s.busy && text.isNotBlank() && lat != null && lng != null
    val save: () -> Unit = {
        if (existing == null) {
            vm.addAddress(text, "", "", kind, lat, lng, select = selectNew)
        } else {
            vm.updateAddress(existing.id, text, "", "", kind, lat, lng)
        }
        onDone()
    }

    // ══════════════════════════════════════════════════════════════════
    // **«حفظ العنوان» ثابتٌ تحت الصفحة** (إعادةُ التصميم ٢٠٢٦-١١-١١)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان آخرَ الصفحة بعد الخريطة والنوع والمشاركة** فيُمرَّر إليه. والآن في الشريط
    // الموحَّد (`BottomActionBar`) — يُرى من أوّل الصفحة، والمحتوى يمرّ فوقه.
    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.canvas)
            .statusBarsPadding()
            .imePadding(),
    ) {
    Column(
        Modifier
            .weight(1f)
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Rahal.space.gutter, vertical = Rahal.space.md),
    ) {
        PageHeader(stringResource(R.string.addrp_title), onBack = onDone)
        Spacer(Modifier.height(Rahal.space.md))

        // ── ١ · العنوانُ التفصيليّ ──
        // **حقلٌ موسومٌ بالقطعة الموحّدة** (`LabeledField`) — الوسمُ فوقه يبقى، والمثالُ داخلَه.
        com.rahalgo.ui.LabeledField(
            label = stringResource(R.string.addrp_detail),
            value = text,
            onValueChange = { text = TextLimits.fit(it, TextLimits.ADDRESS_PART) },
            placeholder = stringResource(R.string.addrp_detail_hint),
            singleLine = false,
            minLines = 2,
        )

        // ── ٢ · النوع ── قبل الخريطة (قرارُ المالك ٢٠٢٦-١١-٢٤: «اول شي العنوان التفصيلي
        // بعدها يختار منزل او العمل وبعدها الخريطة وبعدا حفظ»).
        com.rahalgo.ui.SectionHeader(stringResource(com.rahalgo.ui.R.string.addr_kind))
        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            KindChip("home", com.rahalgo.ui.R.drawable.ic_home, kind == "home", Modifier.weight(1f)) { kind = "home" }
            KindChip("work", com.rahalgo.ui.R.drawable.ic_work, kind == "work", Modifier.weight(1f)) { kind = "work" }
            KindChip("other", com.rahalgo.ui.R.drawable.ic_star, kind == "other", Modifier.weight(1f)) { kind = "other" }
        }

        // ── ٣ · الخريطةُ و«حدّد موقعي بدقة» ──
        Spacer(Modifier.height(Rahal.space.lg))
        com.rahalgo.map.PointMapCard(
            lat = lat,
            lng = lng,
            pending = false,
            enabled = true,
            onPoint = { la, ln ->
                lat = la
                lng = ln
            },
            // **والجديدُ وحدَه يحدّد موقعَه عند الفتح** — عنوانٌ محفوظٌ يُعدَّل لا تُنقل
            // نقطتُه إلى حيث يقف صاحبُه الآن.
            autoLocate = existing == null,
            dragHint = com.rahalgo.map.R.string.map_drag_to_set_address,
        )

        // **ومفتاحُ «شارك موقعي» انتقل إلى «حسابي ← الإعدادات»** (قرارُ المالك ٢٠٢٦-١١-٢٤) —
        // كان هنا فيطفئه من يعدّل عنوانَه بالغلط (`live/LiveAutoSetting.kt`).
        Spacer(Modifier.height(Rahal.space.lg))
    }
    // ── ٦ · الحفظ — في الشريط الثابت ──
    com.rahalgo.ui.BottomActionBar {
        com.rahalgo.ui.ClearStateButton(
            onClick = save,
            enabled = canSave,
            tone = Tone.Accent,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(com.rahalgo.ui.R.string.addr_save)) }
    }
    }
}

/**
 * **نصُّ العنوان المحفوظ في الحقل الواحد** — أجزاؤه القديمة (شارعٌ وطابق) يجمعها
 * سطرُ المحرّك، **فلا يضيع منها شيءٌ حين يُعدَّل.**
 */
internal fun initialText(a: Address?): String {
    if (a == null) return ""
    val joined = if (a.street.isBlank() && a.floor.isBlank()) a.areaBuilding else a.text.ifBlank { a.areaBuilding }
    return TextLimits.fit(joined, TextLimits.ADDRESS_PART)
}

/** **رقاقةُ نوع** — أيقونةٌ واسم، **والمختارُ بحدٍّ وأرضٍ ولون.** */
@Composable
private fun KindChip(
    kind: String,
    icon: Int,
    on: Boolean,
    modifier: Modifier = Modifier,
    onPick: () -> Unit,
) {
    Row(
        modifier
            .clip(Rahal.shape.pill)
            .background(if (on) Rahal.colors.warnTint else Rahal.colors.canvas)
            .border(
                width = if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                color = if (on) Rahal.colors.accent else Rahal.colors.line,
                shape = Rahal.shape.pill,
            )
            .selectable(selected = on, role = androidx.compose.ui.semantics.Role.RadioButton, onClick = onPick)
            .heightIn(min = Rahal.size.touch)
            .padding(horizontal = Rahal.space.xs, vertical = Rahal.space.sm),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = if (on) Rahal.colors.accentInk else Rahal.colors.inkMuted,
            modifier = Modifier.size(Rahal.icon.sm),
        )
        Spacer(Modifier.size(Rahal.space.xs))
        // **حبرُ البرتقاليّ الغامق للنصّ** — البرتقاليُّ نفسُه على الأبيض لا يُقرأ.
        Text(
            stringResource(addressKindLabel(kind)),
            color = if (on) Rahal.colors.accentInk else Rahal.colors.inkMuted,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Medium,
            style = Rahal.type.label,
            maxLines = 1,
        )
    }
}
