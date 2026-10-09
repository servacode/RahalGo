package com.rahalgo.ui.offers

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.R
import com.rahalgo.ui.OfferCard
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.money
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إنشاءُ عرضٍ — خمسُ خطواتٍ تُمشى، لا نموذجٌ يُملأ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «إعادة تصميم صفحة العروض… يختار المتجر ثمّ
 *  القسم ثمّ الصنف ثمّ نسبة ثمّ المدّة».)
 *
 * # ما كان
 *
 * **كلُّ أصناف المتجر في عمودٍ واحدٍ بلا قسم** (`flatMap`)، **وسطورٌ
 * نصّيّةٌ تُضغط بلا ما يقول إنّها تُضغط** — لا إطارَ ولا أرضيّة.
 * **والمدّةُ خمسُ كلماتٍ رماديّةٍ في صفّ**، والنسبةُ حقلٌ حرّ. **فالشاشةُ
 * تُقرأ لوحةَ إعداداتٍ لا فعلاً يُمشى.**
 *
 * **ومتجرٌ فيه ثمانون صنفاً يصير بحثاً بالعين.**
 *
 * # وما صار
 *
 * **خطوةٌ واحدةٌ ظاهرةٌ في كلّ مرّة**، ورأسٌ يقول «الخطوة ٣ من ٥»،
 * **وما اختير يبقى مكتوباً فوقها** — فيُراجَع بلا رجوع.
 *
 * **والمتجرُ يُختار من قائمة عملائه بالاسم** لا بمعرّفٍ يُكتب — **وحارسُ
 * `RO-01` باقٍ**: اسمُ المتجر فوق كلّ خطوةٍ بعده، **ومن أنزل خصماً على
 * متجرٍ ظنّه غيرَه أضرّ برزق رجل.**
 *
 * # وشاشةٌ واحدةٌ للمندوب وللمتجر
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٣٠.) **المتجرُ لا يختار متجراً** — فرعُه المختارُ
 * يُفتح وحدَه وخطواتُه أربع. **والخصمُ نسبةٌ أو مبلغٌ ثابت** للاثنين.
 */
@Composable
fun OffersWizard(vm: OffersWizardViewModel) {
    val ctx = LocalContext.current
    val focus = LocalFocusManager.current
    // ══════════════════════════════════════════════════════════════════
    // **صفحتان لا صفحةٌ معجوقة** (طلبُ المالك ٢٠٢٦-١٠-٠٩)
    // ══════════════════════════════════════════════════════════════════
    //
    // «صفحة إنشاء العرض معجوقة مو مفهومة.» — **كانت الخطواتُ والاختياراتُ
    // ورسالةُ «تمّ» والعروضُ القديمةُ كلُّها فوق بعضها**، و«الخطوة ٣ من ٥»
    // تتبدّل وحدَها. **فصارت**: «العروض» (القائمة وزرُّ «عرض جديد») ثمّ
    // «عرض جديد» (نموذجٌ واحدٌ ظاهرٌ كلُّه ومعاينةٌ قبل الإنشاء)، **ومن
    // أنشأ عاد إلى القائمة.** ويبقيان مع تدوير الشاشة.
    var composing by rememberSaveable { mutableStateOf(false) }
    var percent by rememberSaveable { mutableStateOf("") }
    var hours by rememberSaveable { mutableStateOf(24) }
    var query by rememberSaveable { mutableStateOf("") }

    LaunchedEffect(vm.merchantID) {
        if (vm.merchantID.isEmpty()) vm.loadClients()
    }
    // **ونجاحُ الإنشاء يعيد إلى القائمة** — والجديدُ أوّلُها، ورسالةُ «تمّ» فوقها.
    LaunchedEffect(vm.created) {
        if (vm.created > 0) {
            percent = ""
            query = ""
            composing = false
        }
    }
    androidx.activity.compose.BackHandler(enabled = composing) { composing = false }

    val value = percent.toLongOrNull() ?: 0L
    val picked = vm.items.firstOrNull { it.id == vm.pickedItem }
    val itemPrice = picked?.price ?: 0L
    val valueOK = if (vm.byPercent) value in 1..90
    else value > 0 && (itemPrice <= 0 || value < itemPrice)

    LazyColumn(
        Modifier.fillMaxWidth().imePadding().padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (vm.error.isNotEmpty()) {
            item {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.md)
                        .background(Rahal.colors.warnTint)
                        .padding(12.dp),
                ) {
                    Text(vm.error, color = Rahal.colors.danger)
                    if (vm.merchantID.isNotEmpty() && vm.sections.isEmpty()) {
                        RahalTextButton(onClick = { vm.load() }, enabled = !vm.busy) {
                            Text(stringResource(R.string.act_retry))
                        }
                    }
                }
            }
        }

        // ── اختيارُ المتجر (للمندوب) ─────────────────────────────────
        if (vm.merchantID.isEmpty()) {
            if (!vm.picksStore) {
                item { if (vm.busy) Hint(stringResource(R.string.ow_loading)) }
            } else {
                item {
                    StepCard(stringResource(R.string.ow_s1)) {
                        if (vm.clients.isEmpty() && !vm.busy) Hint(stringResource(R.string.ow_no_clients))
                        if (vm.clients.isEmpty() && vm.busy) Hint(stringResource(R.string.ow_loading))
                        vm.clients.forEach { c ->
                            PickRow(text = c.name, chosen = false, onClick = { vm.open(c.id, c.name) })
                        }
                    }
                }
            }
            return@LazyColumn
        }

        // ── رأسُ المتجر ───────────────────────────────────────────────
        item {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    if (composing) stringResource(R.string.ow_new_title)
                    else stringResource(if (vm.picksStore) R.string.ow_current else R.string.ow_current_mine),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.ink,
                    modifier = Modifier.weight(1f),
                )
                when {
                    composing -> RahalTextButton(onClick = { composing = false }) {
                        Text(stringResource(R.string.ow_back_to_list))
                    }
                    vm.picksStore -> RahalTextButton(onClick = { vm.clearMerchant() }) {
                        Text(stringResource(R.string.ow_change_store))
                    }
                }
            }
            if (vm.picksStore) {
                Text(vm.merchantName, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
            }
        }

        if (!composing) {
            // ══════════════════════════════════════════════════════════
            // **١ · العروض — القائمةُ وزرُّ «عرض جديد»**
            // ══════════════════════════════════════════════════════════
            if (vm.createdShown && vm.error.isEmpty()) {
                item {
                    Text(
                        stringResource(R.string.ow_created_top),
                        color = Rahal.colors.success,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.md)
                            .background(Rahal.colors.success.copy(alpha = 0.10f))
                            .padding(12.dp),
                    )
                }
            }
            item {
                RahalButton(
                    onClick = {
                        vm.clearCreated()
                        vm.pickItem("")
                        percent = ""
                        query = ""
                        composing = true
                    },
                    enabled = !vm.busy || vm.items.isNotEmpty(),
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(R.string.ow_new_button)) }
            }
            val rows = vm.rows
            if (rows == null && vm.busy) item { Hint(stringResource(R.string.ow_loading)) }
            if (rows != null && rows.isEmpty()) item { Hint(stringResource(R.string.ow_offers_empty)) }
            items(rows.orEmpty(), key = { it.id }) { o ->
                OfferCard(
                    itemName = o.itemName,
                    status = o.status,
                    priceBefore = o.priceBefore,
                    priceAfter = o.priceAfter,
                    percent = o.discountPercent,
                    amount = o.discountAmount,
                    stopping = vm.stopping == o.id,
                    onStop = { vm.stop(o.id) },
                    endsAt = o.endsAt,
                )
            }
            return@LazyColumn
        }

        // ══════════════════════════════════════════════════════════════
        // **٢ · عرضٌ جديد — نموذجٌ واحدٌ ظاهرٌ كلُّه**
        // ══════════════════════════════════════════════════════════════

        // ── الصنف: بحثٌ وأقسامٌ للفلترة ─────────────────────────────
        item {
            val pool = if (vm.picksSection && vm.pickedSection.isNotEmpty()) {
                vm.sections.firstOrNull { it.id == vm.pickedSection }?.items.orEmpty()
            } else {
                vm.items
            }
            val shown = if (query.isBlank()) pool
            else pool.filter { it.name.contains(query.trim(), ignoreCase = true) }
            // **وصنفٌ عليه عرضٌ يُقال بجانبه** — الجديدُ يحلّ محلّه (`OFFER-EXP`).
            val taken = vm.rows.orEmpty()
                .filter {
                    it.status == com.rahalgo.ui.OfferStatus.ACTIVE ||
                        it.status == com.rahalgo.ui.OfferStatus.SCHEDULED
                }
                .mapNotNull { o -> o.menuItemId?.let { it to o.status } }
                .toMap()
            StepCard(stringResource(R.string.ow_f_item)) {
                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it.take(40) },
                    placeholder = { Text(stringResource(R.string.ow_search)) },
                    leadingIcon = {
                        androidx.compose.material3.Icon(
                            androidx.compose.ui.res.painterResource(R.drawable.ic_search),
                            contentDescription = null,
                        )
                    },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                    keyboardActions = KeyboardActions(onSearch = { focus.clearFocus() }),
                    modifier = Modifier.fillMaxWidth(),
                )
                if (vm.picksSection && vm.sections.size > 1) {
                    Spacer(Modifier.height(8.dp))
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        FilterChip(
                            selected = vm.pickedSection.isEmpty(),
                            onClick = { vm.pickSection("") },
                            label = { Text(stringResource(R.string.ow_all_sections)) },
                        )
                        vm.sections.forEach { s ->
                            FilterChip(
                                selected = vm.pickedSection == s.id,
                                onClick = { vm.pickSection(s.id) },
                                label = { Text(s.name) },
                            )
                        }
                    }
                }
                Spacer(Modifier.height(8.dp))
                if (vm.busy && pool.isEmpty()) Hint(stringResource(R.string.ow_loading_items))
                if (shown.isEmpty() && !vm.busy) {
                    Hint(
                        stringResource(
                            if (query.isNotBlank()) R.string.ow_no_match else R.string.ow_no_items_store,
                        ),
                    )
                }
                shown.forEach { i ->
                    PickRow(
                        text = i.name + "  ·  " + money(i.price),
                        chosen = vm.pickedItem == i.id,
                        note = when (taken[i.id]) {
                            null -> null
                            com.rahalgo.ui.OfferStatus.SCHEDULED -> stringResource(R.string.ow_item_scheduled)
                            else -> stringResource(R.string.ow_item_taken)
                        },
                        onClick = { vm.pickItem(i.id) },
                    )
                }
            }
        }

        // ── الخصم: نسبةٌ أو مبلغ ─────────────────────────────────────
        item {
            StepCard(stringResource(R.string.ow_f_discount)) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FilterChip(
                        selected = vm.byPercent,
                        onClick = { if (!vm.byPercent) { vm.pickMode(true); percent = "" } },
                        label = { Text(stringResource(R.string.ow_mode_percent)) },
                    )
                    FilterChip(
                        selected = !vm.byPercent,
                        onClick = { if (vm.byPercent) { vm.pickMode(false); percent = "" } },
                        label = { Text(stringResource(R.string.ow_mode_fixed)) },
                    )
                }
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = percent,
                    onValueChange = { v ->
                        percent = v.filter { it.isDigit() }.take(if (vm.byPercent) 2 else 9)
                    },
                    label = { Text(stringResource(if (vm.byPercent) R.string.ow_percent else R.string.ow_amount)) },
                    singleLine = true,
                    isError = percent.isNotEmpty() && !valueOK,
                    supportingText = {
                        Text(
                            when {
                                percent.isNotEmpty() && !valueOK && vm.byPercent ->
                                    stringResource(R.string.ow_bad_percent)
                                percent.isNotEmpty() && !valueOK ->
                                    stringResource(R.string.ow_bad_amount, itemPrice.toString())
                                vm.byPercent -> stringResource(R.string.ow_percent_hint)
                                else -> stringResource(R.string.ow_amount_hint)
                            },
                            color = if (percent.isNotEmpty() && !valueOK) Rahal.colors.danger else Rahal.colors.inkMuted,
                        )
                    },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { focus.clearFocus() }),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        // ── المدّة ───────────────────────────────────────────────────
        item {
            StepCard(stringResource(R.string.ow_s5)) {
                Row(
                    Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    OfferDuration.PRESET_HOURS.forEach { h ->
                        FilterChip(
                            selected = h == hours,
                            onClick = { hours = h },
                            label = { Text(OfferDuration.label(ctx, h)) },
                        )
                    }
                }
            }
        }

        // ── المعاينة والإنشاء ────────────────────────────────────────
        item {
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.success.copy(alpha = 0.08f))
                    .padding(12.dp),
            ) {
                Text(stringResource(R.string.ow_preview_title), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.labelMedium)
                Spacer(Modifier.height(6.dp))
                if (picked == null) {
                    Text(stringResource(R.string.ow_preview_pick), color = Rahal.colors.inkMuted)
                } else {
                    Text(picked.name, fontWeight = FontWeight.Bold, color = Rahal.colors.ink)
                    // **ولا يُحسب السعرُ في الجهاز** (`OffersPolicyTest`): المحرّكُ يحسبه بهامش
                    // الصنف والقسم — **فيُقال الخصمُ وسعرُ الصنف، والنهائيُّ في القائمة.**
                    Text(stringResource(R.string.ow_preview_price, money(itemPrice)), color = Rahal.colors.ink)
                    if (valueOK) {
                        Text(
                            stringResource(
                                R.string.ow_preview_discount,
                                if (vm.byPercent) value.toString() + "٪" else money(value),
                            ),
                            fontWeight = FontWeight.Bold,
                            color = Rahal.colors.success,
                        )
                    }
                    Text(
                        stringResource(R.string.ow_preview_for, OfferDuration.label(ctx, hours)),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                Spacer(Modifier.height(4.dp))
                if (picked != null) Hint(stringResource(R.string.ow_preview_final))
                Hint(stringResource(if (vm.picksStore) R.string.ow_borne_store else R.string.ow_borne_me))
                Spacer(Modifier.height(10.dp))
                RahalButton(
                    onClick = {
                        focus.clearFocus()
                        vm.create(vm.pickedItem, value, hours)
                    },
                    enabled = !vm.busy && picked != null && valueOK,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(R.string.ow_create)) }
            }
        }
    }
}

/** **لوحُ خطوةٍ** — عنوانٌ ورجوعٌ وما فيها. */
@Composable
private fun StepCard(
    title: String,
    onBack: (() -> Unit)? = null,
    body: @Composable () -> Unit,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                title,
                fontWeight = FontWeight.Bold,
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
            // **ورجوعٌ خطوةً واحدة** — **ومن أخطأ صنفاً لا يبدأ من أوّله.**
            if (onBack != null) {
                RahalTextButton(onClick = onBack) {
                    Text(stringResource(R.string.ow_back))
                }
            }
        }
        Spacer(Modifier.height(8.dp))
        body()
    }
}

/**
 * **سطرٌ يُضغط ويبدو أنّه يُضغط** — **وكان نصّاً عارياً**: يتبدّل لونُه
 * حين يُختار وحدَه، **فلا شيء يقول للمندوب أنّ هذا السطرَ بابٌ.**
 */
@Composable
private fun PickRow(
    text: String,
    chosen: Boolean,
    onClick: () -> Unit,
    enabled: Boolean = true,
    /** **وسببُ المنع بجانبه** — **وصفٌّ معطّلٌ بلا سببٍ يُقرأ عطبا.** */
    note: String? = null,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sm)
            .background(
                when {
                    !enabled -> Rahal.colors.surface
                    chosen -> Rahal.colors.brand.copy(alpha = 0.12f)
                    else -> Rahal.colors.surface
                },
            )
            .border(
                Rahal.stroke.hair,
                if (chosen) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.sm,
            )
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text,
            maxLines = 1,
            color = when {
                !enabled -> Rahal.colors.inkMuted
                chosen -> Rahal.colors.brand
                else -> Rahal.colors.ink
            },
            fontWeight = if (chosen) FontWeight.Bold else FontWeight.Normal,
            modifier = Modifier.weight(1f),
        )
        if (note != null) {
            Text(
                note,
                style = MaterialTheme.typography.labelMedium,
                color = Rahal.colors.accent,
                modifier = Modifier
                    .clip(Rahal.shape.sm)
                    .background(Rahal.colors.accent.copy(alpha = 0.12f))
                    .padding(horizontal = 6.dp, vertical = 2.dp),
            )
        } else if (chosen) {
            Text("✓", color = Rahal.colors.brand, fontWeight = FontWeight.Bold)
        }
    }
    Spacer(Modifier.height(6.dp))
}

@Composable
private fun Hint(text: String) {
    Text(
        text,
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
        modifier = Modifier.padding(vertical = 2.dp),
    )
}
