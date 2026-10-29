package com.rahalgo.ui.offers

import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.OfferCard
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.OfferStatus
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.grouped
import com.rahalgo.ui.money
import com.rahalgo.ui.remainingLabel

/** **مربّعُ أيقونة الفراغ** — دائرةٌ كبيرةٌ تُقرأ رسماً لا سطراً. */
private val EmptyCircle = 88.dp
private val EmptyIcon = 40.dp

/** **أيقونةٌ داخل زرّ** — كما في محرّر الصنف. */
private val ButtonIcon = 18.dp

/** **مربّعُ نوع الخصم** — كبيرٌ يُضغط بالإبهام. */
private val TypeTileHeight = 96.dp
private val TypeGlyph = 48.dp

/** **مربّعُ صنفٍ أو مدّة** — أدنى ارتفاعٍ يُضغط بثقة. */
private val TileMin = 56.dp

/** **زرُّ الرجوع** — مساحةُ لمسٍ كاملة. */
private val BackTouch = 48.dp

/** **كم صنفاً تُعرض قبل البحث** — والباقي بالبحث أو بالقسم. */
private const val FirstItems = 8

/** **نسبٌ تُضغط ضغطةً** — والحقلُ باقٍ لما سواها. */
private val QuickPercents = listOf(10, 15, 20, 25, 30, 50)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **العروض — قائمةٌ ببطاقاتٍ احترافيّة، ونموذجُ إنشاءٍ في بطاقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-١٠-٢٩: «صفحة العروض بدها إعادة تصميم كامل احترافي»
 *  و«ما في داعي للشرح».) **عناوينُ وحقولٌ وأزرارٌ ورسائلُ خطأ — لا أسطرَ شرح.**
 *
 * - **رأسٌ بسهم رجوعٍ وعنوان** — في القائمة يُغلق الشاشة (`onBack`)،
 *   **وفي النموذج يعود إلى القائمة.**
 * - **القائمة**: زرُّ «أنشئ عرضاً» بارزٌ ثمّ البطاقات (`OfferCard`)،
 *   **والإيقافُ يُسأل عنه قبل أن يقع.** والفراغُ رسمٌ وزرٌّ كبير.
 * - **النموذج**: نوعُ الخصم مربّعان متساويان، والصنفُ مربّعاتٌ بعلامة ✓،
 *   والقيمةُ حقلٌ بوحدته، والمدّةُ مربّعات، ثمّ ملخّصٌ و«حفظ» و«إلغاء».
 *
 * # وشاشةٌ واحدةٌ للمندوب وللمتجر
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٣٠.) **المندوبُ يختار متجراً من عملائه** — **وحارسُ
 * `RO-01` باقٍ**: اسمُ المتجر فوق القائمة والنموذج. **والمتجرُ فرعُه مختار.**
 * **والعرضُ على صنفٍ واحدٍ بالليرة** — نسبةً أو مبلغاً، **كما يقبله المحرّك.**
 */
@Composable
fun OffersWizard(
    vm: OffersWizardViewModel,
    /** **سهمُ الرجوع في القائمة** — و`null` لا سهمَ فيها (المندوبُ يرجع بزرّ الجهاز). */
    onBack: (() -> Unit)? = null,
    /** **يفتح على «عرض جديد» مباشرةً** — من زرّ «أنشئ عرضاً» في الأصناف (٢٠٢٦-١٠-٢٩). */
    startNew: Boolean = false,
) {
    val ctx = LocalContext.current
    val focus = LocalFocusManager.current
    // **صفحتان لا صفحةٌ معجوقة** (طلبُ المالك ٢٠٢٦-١٠-٠٩) — القائمةُ ثمّ النموذج،
    // **ومن أنشأ عاد إلى القائمة.** ويبقيان مع تدوير الشاشة.
    var composing by rememberSaveable { mutableStateOf(startNew) }
    var percent by rememberSaveable { mutableStateOf("") }
    var hours by rememberSaveable { mutableStateOf(24) }
    var query by rememberSaveable { mutableStateOf("") }
    var askStop by rememberSaveable { mutableStateOf<String?>(null) }
    // **بعد اختيار الصنف تنطوي الشبكةُ سطراً** — «تغيير» يفتحها ثانية.
    var changeItem by rememberSaveable { mutableStateOf(false) }

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

    // **والإيقافُ يُسأل عنه** — عرضٌ نزل بالخطأ يُعاد إنشاؤه من أوّله.
    askStop?.let { id ->
        ConfirmDialog(
            title = stringResource(R.string.ow_stop_q),
            body = stringResource(R.string.ow_stop_body),
            confirm = stringResource(R.string.ow_stop_do),
            onConfirm = { vm.stop(id) },
            onDismiss = { askStop = null },
        )
    }

    val startCompose = {
        vm.clearCreated()
        vm.pickItem("")
        percent = ""
        query = ""
        changeItem = false
        composing = true
    }

    val value = percent.toLongOrNull() ?: 0L
    val picked = vm.items.firstOrNull { it.id == vm.pickedItem }
    val itemPrice = picked?.price ?: 0L
    val valueOK = if (vm.byPercent) value in 1..90
    else value > 0 && (itemPrice <= 0 || value < itemPrice)

    // **والحفظُ ثابتٌ تحت النموذج** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — لا آخرَ صفحةٍ طويلة.
    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.surface)
            .imePadding(),
    ) {
        LazyColumn(
            Modifier
                .fillMaxWidth()
                .weight(1f),
            contentPadding = PaddingValues(Rahal.space.lg),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            // ── الرأس ─────────────────────────────────────────────────────
            item {
                com.rahalgo.ui.PageHeader(
                    title = stringResource(if (composing) R.string.ow_new_title else R.string.ow_title),
                    onBack = when {
                        composing -> ({ composing = false })
                        else -> onBack
                    },
                )
            }

            if (vm.error.isNotEmpty()) {
                item {
                    Column(
                        Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.md)
                            // **لونٌ واحد** — أرضٌ حمراءُ خفيفةٌ وكتابةٌ حمراء، لا برتقاليٌّ تحت أحمر.
                            .background(Rahal.colors.danger.copy(alpha = 0.10f))
                            .padding(Rahal.space.md),
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
                    item { if (vm.busy) Muted(stringResource(R.string.ow_loading)) }
                } else {
                    item {
                        FormCard(stringResource(R.string.ow_s1)) {
                            if (vm.clients.isEmpty() && !vm.busy) Muted(stringResource(R.string.ow_no_clients))
                            if (vm.clients.isEmpty() && vm.busy) Muted(stringResource(R.string.ow_loading))
                            vm.clients.forEach { c ->
                                ChoiceRow(text = c.name, chosen = false, onClick = { vm.open(c.id, c.name) })
                            }
                        }
                    }
                }
                return@LazyColumn
            }

            // ── اسمُ المتجر (للمندوب) — `RO-01` ────────────────────────────
            if (vm.picksStore) {
                item {
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.md)
                            .background(Rahal.colors.canvas)
                            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                            .padding(start = Rahal.space.md),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            painterResource(R.drawable.ic_store),
                            contentDescription = null,
                            tint = Rahal.colors.brand,
                            modifier = Modifier.size(ButtonIcon),
                        )
                        Spacer(Modifier.width(Rahal.space.sm))
                        Text(
                            vm.merchantName,
                            fontWeight = FontWeight.Bold,
                            color = Rahal.colors.ink,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f),
                        )
                        if (!composing) {
                            RahalTextButton(onClick = { vm.clearMerchant() }) {
                                Text(stringResource(R.string.ow_change_store))
                            }
                        } else {
                            Spacer(Modifier.height(BackTouch))
                        }
                    }
                }
            }

            if (!composing) {
                // ══════════════════════════════════════════════════════════
                // **١ · القائمة**
                // ══════════════════════════════════════════════════════════
                if (vm.createdShown && vm.error.isEmpty()) {
                    item {
                        Row(
                            Modifier
                                .fillMaxWidth()
                                .clip(Rahal.shape.md)
                                .background(Rahal.colors.success.copy(alpha = 0.10f))
                                .padding(Rahal.space.md),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                stringResource(R.string.ow_created_top),
                                color = Rahal.colors.success,
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    }
                }
                val rows = vm.rows
                if (rows == null && vm.busy) item { Muted(stringResource(R.string.ow_loading)) }
                if (rows != null && rows.isEmpty()) {
                    item {
                        EmptyOffers(enabled = !vm.busy || vm.items.isNotEmpty(), onCreate = startCompose)
                    }
                }
                if (!rows.isNullOrEmpty()) {
                    item {
                        RahalButton(
                            onClick = startCompose,
                            enabled = !vm.busy || vm.items.isNotEmpty(),
                            modifier = Modifier.fillMaxWidth(),
                        ) { IconText(R.drawable.ic_plus, stringResource(R.string.ow_add)) }
                    }
                }
                items(rows.orEmpty(), key = { it.id }) { o ->
                    OfferCard(
                        itemName = o.itemName,
                        status = o.status,
                        priceBefore = o.priceBefore,
                        priceAfter = o.priceAfter,
                        percent = o.discountPercent,
                        amount = o.discountAmount,
                        stopping = vm.stopping == o.id,
                        onStop = { askStop = o.id },
                        endsAt = o.endsAt,
                        startsAt = o.startsAt,
                    )
                }
                return@LazyColumn
            }

            // ══════════════════════════════════════════════════════════════
            // **٢ · عرضٌ جديد — بطاقاتٌ بالترتيب**
            // ══════════════════════════════════════════════════════════════

            // ── الصنف: بحثٌ وأقسامٌ ومربّعاتٌ بعلامة ✓ ────────────────────
            item {
                val pool = if (vm.picksSection && vm.pickedSection.isNotEmpty()) {
                    vm.sections.firstOrNull { it.id == vm.pickedSection }?.items.orEmpty()
                } else {
                    vm.items
                }
                // **والشبكةُ نتائجُ البحث أو أوّلُ ثمانية** — متجرٌ بمئتي صنفٍ لا يُعرض كلُّه.
                val matched = if (query.isBlank()) pool
                else pool.filter { it.name.contains(query.trim(), ignoreCase = true) }
                val shown = if (query.isBlank() && vm.pickedSection.isEmpty()) matched.take(FirstItems) else matched
                // **وصنفٌ عليه عرضٌ يُقال بجانبه** — الجديدُ يحلّ محلّه (`OFFER-EXP`).
                val taken = vm.rows.orEmpty()
                    .filter { it.status == OfferStatus.ACTIVE || it.status == OfferStatus.SCHEDULED }
                    .mapNotNull { o -> o.menuItemId?.let { it to o.status } }
                    .toMap()
                FormCard(stringResource(R.string.ow_f_item)) {
                    if (picked != null && !changeItem) {
                        // ── «الصنف: شاورما — تغيير» ──
                        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(
                                    picked.name,
                                    fontWeight = FontWeight.Bold,
                                    color = Rahal.colors.ink,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                Text(money(picked.price), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
                            }
                            RahalTextButton(onClick = { changeItem = true }) {
                                Text(stringResource(R.string.ow_item_change), fontWeight = FontWeight.Bold)
                            }
                        }
                    } else {
                        OutlinedTextField(
                            value = query,
                            onValueChange = { query = it.take(40) },
                            placeholder = { Text(stringResource(R.string.ow_search)) },
                            leadingIcon = {
                                Icon(painterResource(R.drawable.ic_search), contentDescription = null)
                            },
                            singleLine = true,
                            shape = Rahal.shape.md,
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                            keyboardActions = KeyboardActions(onSearch = { focus.clearFocus() }),
                            modifier = Modifier.fillMaxWidth(),
                        )
                        if (vm.picksSection && vm.sections.size > 1) {
                            Spacer(Modifier.height(Rahal.space.sm))
                            Row(
                                Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                            ) {
                                Pill(
                                    text = stringResource(R.string.ow_all_sections),
                                    on = vm.pickedSection.isEmpty(),
                                    onClick = { vm.pickSection("") },
                                )
                                vm.sections.forEach { s ->
                                    Pill(text = s.name, on = vm.pickedSection == s.id, onClick = { vm.pickSection(s.id) })
                                }
                            }
                        }
                        Spacer(Modifier.height(Rahal.space.sm))
                        if (vm.busy && pool.isEmpty()) Muted(stringResource(R.string.ow_loading_items))
                        if (shown.isEmpty() && !vm.busy) {
                            Muted(stringResource(if (query.isNotBlank()) R.string.ow_no_match else R.string.ow_no_items_store))
                        }
                        TileGrid(shown, columns = 2) { i, mod ->
                            ItemTile(
                                name = i.name,
                                price = money(i.price),
                                chosen = vm.pickedItem == i.id,
                                note = when (taken[i.id]) {
                                    null -> null
                                    OfferStatus.SCHEDULED -> stringResource(R.string.ow_item_scheduled)
                                    else -> stringResource(R.string.ow_item_taken)
                                },
                                onClick = {
                                    vm.pickItem(i.id)
                                    changeItem = false
                                    focus.clearFocus()
                                },
                                modifier = mod,
                            )
                        }
                    }
                }
            }

            // ── نوعُ الخصم: مربّعان متساويان ──────────────────────────────
            item {
                FormCard(stringResource(R.string.ow_f_type)) {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                        TypeTile(
                            glyph = stringResource(R.string.ow_unit_percent),
                            label = stringResource(R.string.ow_mode_percent),
                            chosen = vm.byPercent,
                            onClick = { if (!vm.byPercent) { vm.pickMode(true); percent = "" } },
                            modifier = Modifier.weight(1f),
                        )
                        TypeTile(
                            glyph = stringResource(R.string.ow_unit_syp),
                            label = stringResource(R.string.ow_mode_fixed),
                            chosen = !vm.byPercent,
                            onClick = { if (vm.byPercent) { vm.pickMode(false); percent = "" } },
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }

            // ── قيمةُ الخصم ───────────────────────────────────────────────
            item {
                // **ورسالةُ الخطأ وحدَها تحت الحقل** — ولا سطرَ شرح (طلبُ المالك ٢٠٢٦-١٠-٢٩).
                val badValue: (@Composable () -> Unit)? = if (percent.isNotEmpty() && !valueOK) {
                    {
                        Text(
                            if (vm.byPercent) stringResource(R.string.ow_bad_percent)
                            else stringResource(R.string.ow_bad_amount, grouped(itemPrice)),
                            color = Rahal.colors.danger,
                        )
                    }
                } else {
                    null
                }
                FormCard(stringResource(R.string.ow_f_value)) {
                    OutlinedTextField(
                        value = percent,
                        onValueChange = { v ->
                            percent = v.filter { it.isDigit() }.take(if (vm.byPercent) 2 else 9)
                        },
                        label = { Text(stringResource(if (vm.byPercent) R.string.ow_percent else R.string.ow_amount)) },
                        suffix = {
                            Text(
                                stringResource(if (vm.byPercent) R.string.ow_unit_percent else R.string.ow_unit_syp),
                                fontWeight = FontWeight.Bold,
                                color = Rahal.colors.inkMuted,
                            )
                        },
                        singleLine = true,
                        shape = Rahal.shape.md,
                        textStyle = MaterialTheme.typography.titleMedium,
                        isError = percent.isNotEmpty() && !valueOK,
                        supportingText = badValue,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = { focus.clearFocus() }),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    if (vm.byPercent) {
                        Spacer(Modifier.height(Rahal.space.sm))
                        Row(
                            Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                        ) {
                            QuickPercents.forEach { p ->
                                Pill(
                                    text = p.toString() + stringResource(R.string.ow_unit_percent),
                                    on = value == p.toLong(),
                                    onClick = { percent = p.toString(); focus.clearFocus() },
                                )
                            }
                        }
                    }
                }
            }

            // ── المدّة: مربّعاتٌ أربعةٌ في الصفّ ثمّ موعدُ الانتهاء ─────────
            item {
                FormCard(stringResource(R.string.ow_f_period)) {
                    TileGrid(OfferDuration.PRESET_HOURS, columns = 4) { h, mod ->
                        DurationTile(
                            text = OfferDuration.label(ctx, h),
                            chosen = h == hours,
                            onClick = { hours = h },
                            modifier = mod,
                        )
                    }
                    // **وموعدُ الانتهاء يُقرأ قبل الحفظ** — والمحرّكُ يحكم عند الإنشاء.
                    val ends = java.time.Instant.ofEpochMilli(
                        OfferDuration.endsAtMillis(System.currentTimeMillis(), hours),
                    ).toString()
                    remainingLabel(ctx, ends)?.let {
                        Spacer(Modifier.height(Rahal.space.sm))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                painterResource(R.drawable.ic_time),
                                contentDescription = null,
                                tint = Rahal.colors.inkMuted,
                                modifier = Modifier.size(ButtonIcon),
                            )
                            Spacer(Modifier.width(Rahal.space.xs))
                            Text(it, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
            }

            // ── الملخّص ثمّ «حفظ» و«إلغاء» ───────────────────────────────
            if (picked != null) {
                item {
                    FormCard(stringResource(R.string.ow_preview_title)) {
                        Text(picked.name, fontWeight = FontWeight.Bold, color = Rahal.colors.ink)
                        // **ولا يُحسب السعرُ في الجهاز** (`OffersPolicyTest`): المحرّكُ يحسبه بهامش
                        // الصنف والقسم — **فيُقال الخصمُ وسعرُ الصنف، والنهائيُّ في القائمة.**
                        Text(stringResource(R.string.ow_preview_price, money(itemPrice)), color = Rahal.colors.ink)
                        if (valueOK) {
                            Text(
                                stringResource(
                                    R.string.ow_preview_discount,
                                    if (vm.byPercent) value.toString() + stringResource(R.string.ow_unit_percent)
                                    else money(value),
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
                }
            }
        }
        // ── شريطُ «حفظ» و«إلغاء» الثابت — في النموذج وحدَه ──
        if (composing && vm.merchantID.isNotEmpty()) {
            HorizontalDivider(thickness = Rahal.stroke.hair, color = Rahal.colors.line)
            Row(
                Modifier
                    .fillMaxWidth()
                    .background(Rahal.colors.canvas)
                    .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.sm),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                RahalButton(
                    onClick = {
                        focus.clearFocus()
                        vm.create(vm.pickedItem, value, hours)
                    },
                    enabled = !vm.busy && picked != null && valueOK,
                    modifier = Modifier.weight(1f),
                ) { IconText(R.drawable.ic_check_mark, stringResource(R.string.act_save)) }
                RahalOutlineButton(
                    onClick = { composing = false },
                    enabled = !vm.busy,
                    modifier = Modifier.weight(1f),
                ) { IconText(R.drawable.ic_close, stringResource(R.string.act_cancel)) }
            }
        }
    }
}

/** **بطاقةُ قسمٍ من النموذج** — بيضاءُ بحدٍّ رفيعٍ وعنوانٍ عريض. */
@Composable
private fun FormCard(title: String, body: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(Rahal.space.md),
    ) {
        Text(
            title,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
            color = Rahal.colors.ink,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        body()
    }
}

/** **فراغُ القائمة** — رسمٌ في دائرة، وعنوان، وزرٌّ كبير. */
@Composable
private fun EmptyOffers(enabled: Boolean, onCreate: () -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(EmptyCircle)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.brand.copy(alpha = 0.10f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(R.drawable.ic_offer),
                contentDescription = null,
                tint = Rahal.colors.brand,
                modifier = Modifier.size(EmptyIcon),
            )
        }
        Spacer(Modifier.height(Rahal.space.md))
        Text(
            stringResource(R.string.ow_empty_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
            color = Rahal.colors.ink,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.lg))
        RahalButton(onClick = onCreate, enabled = enabled, modifier = Modifier.fillMaxWidth()) {
            IconText(R.drawable.ic_plus, stringResource(R.string.ow_add))
        }
    }
}

/** **مربّعُ نوع الخصم** — رمزٌ كبيرٌ واسمٌ تحته، و✓ على المختار. */
@Composable
private fun TypeTile(
    glyph: String,
    label: String,
    chosen: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val tint = if (chosen) Rahal.colors.brand else Rahal.colors.inkMuted
    Box(
        modifier
            .heightIn(min = TypeTileHeight)
            .clip(Rahal.shape.md)
            .background(if (chosen) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (chosen) Rahal.stroke.focus else Rahal.stroke.hair,
                if (chosen) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .selectable(selected = chosen, role = Role.RadioButton, onClick = onClick)
            .padding(Rahal.space.sm),
    ) {
        Column(Modifier.align(Alignment.Center), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier
                    .size(TypeGlyph)
                    .clip(Rahal.shape.pill)
                    .background(tint.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(glyph, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium, color = tint)
            }
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                label,
                fontWeight = FontWeight.Bold,
                color = if (chosen) Rahal.colors.brand else Rahal.colors.ink,
                textAlign = TextAlign.Center,
            )
        }
        if (chosen) CheckBadge(Modifier.align(Alignment.TopEnd))
    }
}

/** **مربّعُ صنف** — اسمُه وسعرُه، و✓ على المختار، وتنبيهُ «عليه عرض» إن وُجد. */
@Composable
private fun ItemTile(
    name: String,
    price: String,
    chosen: Boolean,
    note: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier
            .heightIn(min = TileMin)
            .clip(Rahal.shape.sm)
            .background(if (chosen) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (chosen) Rahal.stroke.focus else Rahal.stroke.hair,
                if (chosen) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.sm,
            )
            .selectable(selected = chosen, role = Role.RadioButton, onClick = onClick)
            .padding(Rahal.space.sm),
    ) {
        Column(Modifier.fillMaxWidth().padding(end = if (chosen) Rahal.space.lg else 0.dp)) {
            Text(
                name,
                fontWeight = FontWeight.Bold,
                color = if (chosen) Rahal.colors.brand else Rahal.colors.ink,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(price, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
            if (note != null) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    note,
                    style = MaterialTheme.typography.labelSmall,
                    color = Rahal.colors.accentInk,
                    modifier = Modifier
                        .clip(Rahal.shape.sm)
                        .background(Rahal.colors.accent.copy(alpha = 0.12f))
                        .padding(horizontal = Rahal.space.xs, vertical = 2.dp),
                )
            }
        }
        if (chosen) CheckBadge(Modifier.align(Alignment.TopEnd))
    }
}

/** **مربّعُ مدّة** — نصٌّ في الوسط، والمختارُ بلون العلامة. */
@Composable
private fun DurationTile(text: String, chosen: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier) {
    Box(
        modifier
            .heightIn(min = TileMin)
            .clip(Rahal.shape.sm)
            .background(if (chosen) Rahal.colors.brand else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (chosen) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.sm)
            .selectable(selected = chosen, role = Role.RadioButton, onClick = onClick)
            .padding(Rahal.space.xs),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            color = if (chosen) Rahal.colors.onBrand else Rahal.colors.ink,
            textAlign = TextAlign.Center,
        )
    }
}

/** **علامةُ ✓ في زاوية المختار.** */
@Composable
private fun CheckBadge(modifier: Modifier = Modifier) {
    Box(
        modifier
            .size(ButtonIcon + Rahal.space.xs)
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.brand),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painterResource(R.drawable.ic_check_mark),
            contentDescription = null,
            tint = Rahal.colors.onBrand,
            modifier = Modifier.size(ButtonIcon - Rahal.space.xs),
        )
    }
}

/** **شبكةُ مربّعاتٍ متساوية** — صفوفٌ بعدد الأعمدة، والناقصُ يُملأ فراغاً. */
@Composable
private fun <T> TileGrid(list: List<T>, columns: Int, tile: @Composable (T, Modifier) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        list.chunked(columns).forEach { row ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                row.forEach { tile(it, Modifier.weight(1f)) }
                repeat(columns - row.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
}

/** **رقاقةٌ تُضغط** — قسمٌ أو نسبةٌ سريعة. */
@Composable
private fun Pill(text: String, on: Boolean, onClick: () -> Unit) {
    Text(
        text,
        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
        fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
        style = MaterialTheme.typography.bodyMedium,
        maxLines = 1,
        modifier = Modifier
            .clip(Rahal.shape.pill)
            .background(if (on) Rahal.colors.brand else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (on) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.pill)
            .selectable(selected = on, role = Role.RadioButton, onClick = onClick)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
    )
}

/** **سطرٌ يُضغط** — متجرٌ من عملاء المندوب. */
@Composable
private fun ChoiceRow(text: String, chosen: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = TileMin)
            .clip(Rahal.shape.sm)
            .background(if (chosen) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (chosen) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.sm)
            .clickable(onClick = onClick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painterResource(R.drawable.ic_store),
            contentDescription = null,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(ButtonIcon),
        )
        Spacer(Modifier.width(Rahal.space.sm))
        Text(
            text,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            fontWeight = FontWeight.Bold,
            color = Rahal.colors.ink,
            modifier = Modifier.weight(1f),
        )
    }
    Spacer(Modifier.height(Rahal.space.sm))
}

@Composable
private fun Muted(text: String) {
    Text(
        text,
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodyMedium,
        modifier = Modifier.padding(vertical = Rahal.space.xs),
    )
}

/** **أيقونةٌ ونصٌّ داخل زرّ** — لونُهما من الزرّ نفسِه. */
@Composable
private fun RowScope.IconText(@DrawableRes icon: Int, text: String) {
    Icon(painterResource(icon), contentDescription = null, modifier = Modifier.size(ButtonIcon))
    Spacer(Modifier.width(Rahal.space.xs))
    Text(text, fontWeight = FontWeight.Bold)
}
