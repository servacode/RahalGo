package com.rahalgo.rep.offers

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
import com.rahalgo.rep.R
import com.rahalgo.ui.OfferCard
import com.rahalgo.ui.OfferDuration
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
 * # والخصمُ بالمئة وحدَه — ويُقال
 *
 * **طلب المالكُ «نسبة ثابتة أو بالمئة»** — **والمحرّكُ لا يعرف إلّا
 * المئة**: `offers.go:90` حقلٌ واحدٌ `discount_percent`، وتحقّقُه
 * `1..90`، والعمودُ في القاعدة كذلك. **والمبلغُ الثابت يمسّ حسابَ السعر
 * بعد الخصم ولقطةَ اقتصاد الطلب** — **فهو دفعةُ مالٍ بقرارٍ صريح، لا
 * حقلٌ يُضاف في شاشة.**
 *
 * **فيُقال للمندوب صريحاً أنّه غيرُ متاح بعد** — **ولا يُترك يبحث عن
 * زرٍّ لا وجودَ له.**
 */
@Composable
fun RepOffersScreen(vm: RepOffersViewModel) {
    val ctx = LocalContext.current
    val focus = LocalFocusManager.current
    var percent by remember { mutableStateOf("") }
    var hours by remember { mutableStateOf(24) }

    // **ولوحُ العملاء يُحمَّل عند الفتح بلا متجر** — **وخطوةٌ أولى فارغةٌ
    // تُقرأ عطباً.**
    LaunchedEffect(vm.merchantID) {
        if (vm.merchantID.isEmpty()) vm.loadClients()
    }

    // **وقيمةٌ واحدةٌ تُقرأ بحسب الطريقة** — **ولا حقلان يملأ أحدَهما
    // وينسى الآخر** فيُرسَل عرضٌ بنسبةٍ ومبلغٍ معاً ويُردّ.
    val value = percent.toLongOrNull() ?: 0L
    // **والمبلغُ دون سعر الصنف** (`OFFER-EXP`): ٢٠٠ على ١٥٠ قُبل على الجهاز
    // فصار الصنفُ «٠ ل.س». **والخادمُ يرفضه أيضاً** — وهذا ليُقال قبل الإرسال.
    val itemPrice = vm.items.firstOrNull { it.id == vm.pickedItem }?.price ?: 0L
    val valueOK = if (vm.byPercent) value in 1..90
    else value > 0 && (itemPrice <= 0 || value < itemPrice)

    // **ونجاحُ الإنشاء يمسح القيمة** — **لا الضغطُ**: من رُفض عرضُه يجد
    // رقمَه كما كتبه فيصحّحه.
    LaunchedEffect(vm.created) { if (vm.created > 0) percent = "" }
    val step = when {
        vm.merchantID.isEmpty() -> 1
        vm.pickedSection.isEmpty() -> 2
        vm.pickedItem.isEmpty() -> 3
        !valueOK -> 4
        else -> 5
    }

    // **و`imePadding`**: لوحةُ المفاتيح كانت تغطّي «أنشئ العرض» فيقع الضغطُ
    // عليها (`OFFER-EXP`، رُئي على الجهاز) — **فالقائمةُ تقصر فوقها.**
    LazyColumn(
        Modifier.fillMaxWidth().imePadding().padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item { StepHeader(step) }

        // **وما اختير يبقى مكتوباً** — **فتُراجَع الخطواتُ بلا رجوع**،
        // **ولا يُنزَل خصمٌ على متجرٍ أو صنفٍ يظنّه غيرَه.**
        if (vm.merchantID.isNotEmpty()) {
            item {
                ChosenBar(
                    merchant = vm.merchantName,
                    section = vm.sections.firstOrNull { it.id == vm.pickedSection }?.name,
                    item = vm.items.firstOrNull { it.id == vm.pickedItem }?.name,
                    onChangeMerchant = { vm.clearMerchant() },
                )
            }
        }

        // **و«تمّ» يُقال** — كانت الشاشةُ تعود للخطوة ٤ صامتة.
        if (vm.createdShown && vm.error.isEmpty()) {
            item {
                Text(
                    stringResource(R.string.of_created),
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

        if (vm.error.isNotEmpty()) {
            item {
                // **وردُّ الخادم يُقال بلفظه** — **ومنعُ التخويل يُقرأ
                // «هذا المتجر ليس من عملائك» لا «حدث خطأ».**
                Text(
                    vm.error,
                    color = Rahal.colors.danger,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.md)
                        .background(Rahal.colors.warnTint)
                        .padding(12.dp),
                )
            }
        }

        // ── ١ · المتجر ───────────────────────────────────────────────
        if (vm.merchantID.isEmpty()) {
            item {
                StepCard(stringResource(R.string.of_s1)) {
                    if (vm.clients.isEmpty() && !vm.busy) {
                        Hint(stringResource(R.string.of_no_clients))
                    }
                    vm.clients.forEach { c ->
                        PickRow(
                            text = c.name,
                            chosen = false,
                            onClick = { vm.open(c.id, c.name) },
                        )
                    }
                }
            }
        }

        // ── ٢ · القسم ────────────────────────────────────────────────
        if (step == 2) {
            item {
                StepCard(stringResource(R.string.of_s2)) {
                    if (vm.sections.isEmpty() && !vm.busy) {
                        Hint(stringResource(R.string.of_no_sections))
                    }
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        vm.sections.forEach { s ->
                            FilterChip(
                                selected = vm.pickedSection == s.id,
                                onClick = { vm.pickSection(s.id) },
                                label = { Text(s.name) },
                            )
                        }
                    }
                }
            }
        }

        // ── ٣ · الصنف ────────────────────────────────────────────────
        if (step == 3) {
            item {
                val inSection = vm.sections
                    .firstOrNull { it.id == vm.pickedSection }?.items.orEmpty()
                StepCard(
                    stringResource(R.string.of_s3),
                    onBack = { vm.pickSection("") },
                ) {
                    if (inSection.isEmpty()) Hint(stringResource(R.string.of_no_items))
                    // ══════════════════════════════════════════════════
                    // **وصنفٌ عليه عرضٌ جارٍ يُقال قبل أن يُختار**
                    // ══════════════════════════════════════════════════
                    //
                    // **قِيس حيّاً ٢٠٢٦-٠٩-٣٠** بنداءٍ إلى التجهيز:
                    // `POST /rep/stores/{id}/offers ⇒ 409
                    //  item_already_discounted`.
                    //
                    // **والمنصّةُ محقّةٌ** — «خصمان على صنفٍ واحدٍ سؤالٌ بلا
                    // جواب» (`offers_one_live_per_item`). **والعطبُ في
                    // الشاشة**: تُمشّي المندوبَ الخطواتِ الخمسَ كلَّها ثمّ
                    // تردّه في آخرها، **ولا تقول لماذا.**
                    //
                    // **فيُوسَم المحجوزُ ويُمنَع ضغطُه** — والسببُ مكتوبٌ
                    // بجانبه. **ومن رأى الجوابَ في أوّل الطريق لم يمشِه.**
                    //
                    // **ولا حجزَ بعد اليوم** (`OFFER-EXP`، قرارُ المالك
                    // ٢٠٢٦-٠٩-٣٠: «لازم نقدر نعمل عرض إيمت ما بدنا»):
                    // **الخادمُ يُنزل القائمَ ويُدرج الجديد.** فالصنفُ يبقى
                    // قابلاً للاختيار، **ويُقال بجانبه إنّ الجديدَ يحلّ محلّ
                    // عرضه** — فلا يُستبدَل خصمٌ سارٍ بلا علم.
                    val taken = vm.rows.orEmpty()
                        .filter {
                            it.status == com.rahalgo.ui.OfferStatus.ACTIVE ||
                                it.status == com.rahalgo.ui.OfferStatus.SCHEDULED
                        }
                        .mapNotNull { o -> o.menuItemId?.let { it to o.status } }
                        .toMap()
                    inSection.forEach { i ->
                        val holder = taken[i.id]
                        PickRow(
                            text = i.name,
                            chosen = vm.pickedItem == i.id,
                            enabled = true,
                            note = when (holder) {
                                null -> null
                                com.rahalgo.ui.OfferStatus.SCHEDULED ->
                                    stringResource(R.string.of_item_scheduled)
                                else -> stringResource(R.string.of_item_taken)
                            },
                            onClick = { vm.pickItem(i.id) },
                        )
                    }
                }
            }
        }

        // ── ٤ و٥ · النسبة والمدّة ────────────────────────────────────
        if (step >= 4) {
            item {
                StepCard(
                    stringResource(R.string.of_s4b),
                    onBack = { vm.pickItem("") },
                ) {
                    // ══════════════════════════════════════════════════
                    // **وطريقتان — نسبةٌ أو مبلغٌ ثابت**
                    // ══════════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٩-٣٠.)
                    //
                    // **وتبديلُ الطريقة يمسح القيمة** — **و«٢٠» تعني
                    // عشرينَ بالمئة في واحدةٍ وعشرينَ ليرةً في الأخرى**،
                    // فرقمٌ باقٍ من طريقةٍ سابقةٍ خصمٌ لم يُقصد.
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        FilterChip(
                            selected = vm.byPercent,
                            onClick = { if (!vm.byPercent) { vm.pickMode(true); percent = "" } },
                            label = { Text(stringResource(R.string.of_mode_percent)) },
                        )
                        FilterChip(
                            selected = !vm.byPercent,
                            onClick = { if (vm.byPercent) { vm.pickMode(false); percent = "" } },
                            label = { Text(stringResource(R.string.of_mode_fixed)) },
                        )
                    }
                    Spacer(Modifier.height(10.dp))
                    OutlinedTextField(
                        value = percent,
                        onValueChange = { v ->
                            // **والنسبةُ رقمان والمبلغُ تسعة** — **وحدُّ
                            // الطولِ يمنع لصقةً بمليارٍ تصير خصماً كاملاً.**
                            percent = v.filter { it.isDigit() }
                                .take(if (vm.byPercent) 2 else 9)
                            vm.clearCreated()
                        },
                        label = {
                            Text(
                                stringResource(
                                    if (vm.byPercent) R.string.offer_percent else R.string.of_amount,
                                ),
                            )
                        },
                        singleLine = true,
                        isError = percent.isNotEmpty() && !valueOK,
                        // **والخطأُ يُقال بالأحمر تحت الحقل** — كانت ٩٥٪ تُعلق
                        // الشاشةَ بلا سبب، والسطرُ الرماديُّ وحدَه.
                        supportingText = if (percent.isNotEmpty() && !valueOK) {
                            {
                                Text(
                                    if (vm.byPercent) {
                                        stringResource(R.string.of_bad_percent)
                                    } else {
                                        stringResource(
                                            R.string.of_bad_amount,
                                            itemPrice.toString(),
                                        )
                                    },
                                    color = Rahal.colors.danger,
                                )
                            }
                        } else {
                            null
                        },
                        // **و«تمّ» في لوحة المفاتيح يطويها.**
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Number,
                            imeAction = ImeAction.Done,
                        ),
                        keyboardActions = KeyboardActions(onDone = { focus.clearFocus() }),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(4.dp))
                    Hint(
                        stringResource(
                            if (vm.byPercent) R.string.of_percent_hint else R.string.of_amount_hint,
                        ),
                    )
                }
            }
        }

        if (step == 5) {
            item {
                StepCard(stringResource(R.string.of_s5)) {
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        // **وكلُّ المدد لا خمسٌ منها** — **والشهرُ كان
                        // يُقصّ بـ`take(5)` فلا يبلغه المندوبُ أبداً**،
                        // وهو مدّةٌ يقبلها المحرّكُ (`MAX_HOURS = 24*30`).
                        OfferDuration.PRESET_HOURS.forEach { h ->
                            FilterChip(
                                selected = h == hours,
                                onClick = { hours = h },
                                label = { Text(OfferDuration.label(ctx, h)) },
                            )
                        }
                    }

                    Spacer(Modifier.height(10.dp))
                    // **ويُقرأ ما سيقع قبل أن يقع** — **بجملةٍ واحدةٍ
                    // فيها الصنفُ والنسبةُ والمدّة.**
                    Text(
                        if (vm.byPercent) {
                            stringResource(
                                R.string.of_preview,
                                vm.items.firstOrNull { it.id == vm.pickedItem }?.name.orEmpty(),
                                value.toInt(),
                                OfferDuration.label(ctx, hours),
                            )
                        } else {
                            stringResource(
                                R.string.of_preview_fixed,
                                vm.items.firstOrNull { it.id == vm.pickedItem }?.name.orEmpty(),
                                value.toString(),
                                OfferDuration.label(ctx, hours),
                            )
                        },
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(Rahal.shape.sm)
                            .background(Rahal.colors.success.copy(alpha = 0.10f))
                            .padding(10.dp),
                    )

                    Spacer(Modifier.height(6.dp))
                    // **ومن يتحمّله يُقال للمندوب أيضاً** — **فهو يشرحه
                    // لصاحب المتجر وهو واقفٌ عنده.**
                    Hint(stringResource(R.string.offer_borne_note))

                    Spacer(Modifier.height(8.dp))
                    RahalButton(
                        onClick = {
                            focus.clearFocus()
                            vm.create(vm.pickedItem, value, hours)
                        },
                        enabled = !vm.busy,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(R.string.offer_create)) }
                }
            }
        }

        // ── عروضُ المتجر القائمة ─────────────────────────────────────
        val rows = vm.rows
        if (vm.merchantID.isNotEmpty() && rows != null) {
            item {
                Spacer(Modifier.height(6.dp))
                Text(
                    stringResource(R.string.of_current),
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.ink,
                )
            }
            if (rows.isEmpty()) {
                item { Hint(stringResource(R.string.offers_empty)) }
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
                onStop = { vm.stop(o.id) },
            )
        }
    }
}

/** **رأسٌ يقول أين هو من الطريق** — **ومن لا يعرف كم بقي يظنّه لا ينتهي.** */
@Composable
private fun StepHeader(step: Int) {
    Column(Modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.of_step, step, 5),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.labelMedium,
        )
        Spacer(Modifier.height(6.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            repeat(5) { i ->
                Box(
                    Modifier
                        .weight(1f)
                        .height(4.dp)
                        .clip(Rahal.shape.sm)
                        .background(
                            if (i < step) Rahal.colors.brand else Rahal.colors.line,
                        ),
                )
            }
        }
    }
}

/** **وما اختير يبقى مكتوباً** — سطرٌ واحدٌ يحمل المتجرَ والقسمَ والصنف. */
@Composable
private fun ChosenBar(
    merchant: String,
    section: String?,
    item: String?,
    onChangeMerchant: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.surface)
            .padding(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                merchant,
                fontWeight = FontWeight.Bold,
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
            )
            RahalTextButton(onClick = onChangeMerchant) {
                Text(stringResource(R.string.of_change_store))
            }
        }
        val trail = listOfNotNull(section, item).joinToString(" ← ")
        if (trail.isNotEmpty()) {
            Text(
                "✓ $trail",
                color = Rahal.colors.success,
                style = MaterialTheme.typography.bodySmall,
            )
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
                    Text(stringResource(R.string.of_back))
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
