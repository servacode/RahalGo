package com.rahalgo.ui.menu

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نموذجُ تحرير الصنف — واحدٌ للمتجر وللمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «الأصناف بالمندوب والمتجر لازم تكون
 *  مركزيّة… وهي لنفس الغرض».)
 *
 * **يأخذ رسماً وأفعالاً ويعيد رسماً** — ولا ينادي شبكةً ولا يعرف
 * `MerchantApi` من `RepApi`. **فالتطبيقُ يصل ما يشاء.**
 *
 * # الشكلُ (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩)
 *
 * - صورةٌ مربّعةٌ كبيرةٌ يميناً، وعن يسارها الاسمُ والوصفُ فوق بعض.
 * - السعرُ وجنبه دائرتا العملة «ل.س» و«$».
 * - قسمُ السوق حقلٌ يفتح نافذةَ مربّعاتٍ بصور الأقسام وعلامةِ ✓ على المختار.
 * - الخياراتُ سطرٌ واحدٌ لكلّ خيار: اسمٌ وسعرٌ بارتفاعٍ واحد، ومفتاحُ
 *   «متوفر / غير متوفر»، وزرُّ حذفٍ واضح.
 * - «أضف مشروباتي» مفتاحٌ لكلّ صنف (هجرة 0483).
 * - كلُّ فعلٍ زرٌّ بأيقونة — والحذفُ بلون الخطر.
 */

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.DEFAULT_BASE_GRAMS
import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.ModifierOption
import com.rahalgo.shared.merchant.OptionPreset
import com.rahalgo.shared.merchant.PRICE_MODE_NONE
import com.rahalgo.shared.merchant.PRICE_MODE_PRICED
import com.rahalgo.shared.merchant.PRICE_MODE_WEIGHT
import com.rahalgo.shared.merchant.PresetOption
import com.rahalgo.shared.merchant.QUICK_GRAMS
import com.rahalgo.shared.merchant.weightName
import com.rahalgo.shared.merchant.weightPrice
import com.rahalgo.shared.merchant.PlatformSectionRef as SectionRef
import com.rahalgo.ui.Card
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.Note
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.Tone
import com.rahalgo.ui.money
import com.rahalgo.ui.rememberImagePicker

// ══════════════════════════════════════════════════════════════════════
// **مقاساتُ هذا النموذج — في موضعٍ واحد**
// ══════════════════════════════════════════════════════════════════════
/** **صورةُ الصنف** — مربّعٌ يملأ ارتفاعَ الاسم والوصف معاً. */
private val PhotoSize = 136.dp

/** **ارتفاعُ حقل السطر** — الاسمُ والسعرُ والغرامُ بارتفاعٍ واحد. */
internal val RowFieldHeight = 44.dp

/** **لمسةُ زرّ الحذف** — ٤٤ لا أقلّ، فلا يُخطئه الإصبع. */
internal val TouchTarget = 44.dp

/** **عرضُ حقل السعر وحقل الغرام** في سطر الخيار. */
private val NumberFieldWidth = 84.dp

/** **حقلُ سعر الخيار في سطره الثاني** — أعرضُ من حقل الغرام ليتّسع لتسميته. */
private val PriceFieldWidth = 128.dp

/** **العدّادُ يظهر حين يبقى من الحدّ هذا العددُ أو أقلّ.** */
private const val NameWarnAt = 5

/** **دائرةُ العملة.** */
private val CurrencyCircle = 48.dp

/** **أيقونةٌ داخل زرّ.** */
internal val ButtonIcon = 18.dp

/** **مربّعُ المجموعة الجاهزة** — كلُّها بارتفاعٍ واحد. */
private val PresetBoxHeight = 60.dp

/** **صورةُ القسم المختار في الحقل.** */
private val SectionThumb = 40.dp

/** **أقصى ارتفاعٍ لشبكة الأقسام** — وما زاد يُمرَّر داخلها. */
private val SectionGridMax = 460.dp

/** **حشوةُ الزرّ الصغير** — «حذف المجموعة» في رأسها. */
private val SmallButtonPad = PaddingValues(horizontal = 10.dp, vertical = 6.dp)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الصنف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # والسعرُ سعرُ الشراء لا البيع
 *
 * **ولو حُرِّر سعرُ البيع لَضاع الهامشُ في أوّل تعديل** — رقمٌ يرتفع بكلّ
 * فتحةٍ للنافذة.
 *
 * # ولا مفتاحَ توفّرٍ للصنف هنا
 *
 * **يُقلب من نقطة السطر وحدَها** — وحقلٌ يُكتب من موضعين يمحو أحدُهما
 * ما فعله الآخر.
 */
@Composable
fun ItemEditor(
    d: ItemDraft,
    sections: List<SectionRef>,
    busy: Boolean,
    media: (String?) -> String?,
    onEdit: ((ItemDraft) -> ItemDraft) -> Unit,
    onPickImage: (ByteArray) -> Unit,
    onClearImage: () -> Unit,
    onSave: () -> Unit,
    onCancel: () -> Unit,
    onDelete: () -> Unit,
    // **وما اختلف بين المتجر والمندوب يُمرَّر، وما اتّفق يبقى هنا** —
    // المتجرُ يقرأ «السعر» والمندوبُ «سعر المتجر» (`pnpm check:menu-parity`).
    @StringRes priceLabel: Int = R.string.item_price,
    @StringRes descLabel: Int = R.string.item_desc,
    @StringRes sectionLabel: Int = R.string.item_section,
    /** **تنبيهُ «بلا قسم»** — وصفرٌ يعني لا تنبيه. */
    @StringRes sectionHint: Int = 0,
    /** **سببُ رفض الحفظ — فوق الزرّ لا منبثقةٌ عابرة.** وفارغٌ يعني لا خطأ. */
    error: String = "",
    /** **سعرُ صرف المتجر** — لعرض «≈ ١٧٥ ل.س» تحت السعر الدولاريّ. */
    exchangeRate: Double? = null,
    /** **ما يُقال إن غاب سعرُ الصرف** — المتجرُ يكتبه بنفسه، والمندوبُ ينتظر صاحبَه. */
    @StringRes rateMissing: Int = R.string.item_usd_need_rate,
    /** **خياراتُ الصنف الجاهزة لهذا المتجر** — و`null` لم تُقرأ فتبقى القوالبُ القديمة. */
    presets: List<OptionPreset>? = null,
    /** **«نسخ الصنف»** — يظهر في التعديل وحدَه؛ و`null` لا زرّ. */
    onDuplicate: (() -> Unit)? = null,
    /** **في المسوّدة ما لم يُحفظ؟** — فيُطفأ النسخُ ويُقال لماذا. */
    dirty: Boolean = false,
    /** **متجرٌ موقوف** — يرى صنفَه ولا يحفظ ولا يحذف ولا ينسخ. */
    readOnly: Boolean = false,
) {
    val pick = rememberImagePicker(crop = true, cropFrames = com.rahalgo.ui.ItemCropFrames) { bytes -> onPickImage(bytes) }
    var confirmDelete by remember { mutableStateOf(false) }
    // **فرقُ خيارٍ لم يُحوَّل لغياب سعر الصرف** — يُقال ولا يُصفَّر بصمت.
    var rateLost by remember(d.itemId) { mutableStateOf(false) }

    // ══════════════════════════════════════════════════════════════
    // **الحفظُ ثابتٌ تحت الشاشة** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩)
    // ══════════════════════════════════════════════════════════════
    // أكثرُ تعديلٍ هو السعرُ فوق، **وكان الحفظُ آخرَ الصفحة** بعد الأقسام والخيارات.
    // فالصفحةُ تُمرَّر وشريطُ «حفظ / إلغاء» تحتها ثابت — كمعالج الإعداد.
    Column(Modifier.fillMaxSize()) {
        Screen(Modifier.weight(1f)) {
            ScreenTitle(
                stringResource(if (d.itemId.isEmpty()) R.string.mn_item_new else R.string.mn_item_edit),
                "",
            )

            // ══════════════════════════════════════════════════════════════
            // **١ · الصورةُ يميناً، والاسمُ والوصفُ عن يسارها**
            // ══════════════════════════════════════════════════════════════
            // **واسمُ الصنف بحدٍّ يُعدّ وهو يُكتب** (قرارُ المالك ٢٠٢٦-١٠-١٤: ٣٠ حرفاً).
            val nameMax = ItemLimits.nameMax
            val nameLen = d.name.trim().length
            val nameOver = nameLen > nameMax
            // **والعدّادُ يظهر قربَ الحدّ أو بعده لا دائماً** — شرحٌ وقتَ الحاجة.
            val nameCounter: (@Composable () -> Unit)? = if (nameLen >= nameMax - NameWarnAt) {
                {
                    Text(
                        if (nameOver) stringResource(R.string.mn_name_over, nameMax.toString(), nameLen.toString())
                        else "$nameLen / $nameMax",
                        color = if (nameOver) Rahal.colors.danger else Rahal.colors.inkMuted,
                    )
                }
            } else {
                null
            }
            Row(verticalAlignment = Alignment.Top) {
                ItemPhoto(d, busy, media, onPick = { pick() }, onClear = onClearImage)
                Spacer(Modifier.width(Rahal.space.md))
                Column(Modifier.weight(1f)) {
                    OutlinedTextField(
                        value = d.name,
                        onValueChange = { v -> onEdit { it.copy(name = v.take(120)) } },
                        label = { Text(stringResource(R.string.mn_item_name)) },
                        singleLine = true,
                        isError = nameOver,
                        supportingText = nameCounter,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = d.description,
                        onValueChange = { v -> onEdit { it.copy(description = v.take(500)) } },
                        label = { Text(stringResource(descLabel)) },
                        maxLines = 3,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }

            // ══════════════════════════════════════════════════════════════
            // **٢ · السعرُ وجنبه دائرتا العملة**
            // ══════════════════════════════════════════════════════════════
            // **وفروقُ الخيارات تُحوَّل مع العملة** (تدقيقُ ٢٠٢٦-١٠-٢٨) — لا تُصفَّر بصمت.
            Spacer(Modifier.height(Rahal.space.md))
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (!d.isUsd) {
                    OutlinedTextField(
                        value = d.price,
                        onValueChange = { v -> onEdit { it.copy(price = v.filter(Char::isDigit)) } },
                        label = { Text(stringResource(priceLabel)) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.weight(1f),
                    )
                } else {
                    OutlinedTextField(
                        value = d.priceUsd,
                        onValueChange = { v ->
                            onEdit { it.copy(priceUsd = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(10)) }
                        },
                        label = { Text(stringResource(R.string.item_price_usd)) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        modifier = Modifier.weight(1f),
                    )
                }
                Spacer(Modifier.width(Rahal.space.md))
                CurrencyCircles(selected = d.currency, enabled = !busy) { c ->
                    if (c != d.currency) {
                        if (d.withCurrency(c, exchangeRate).second) rateLost = true
                        onEdit { it.withCurrency(c, exchangeRate).first }
                    }
                }
            }
            if (d.isUsd) {
                val usd = parseUsd(d.priceUsd)
                val noRate = exchangeRate == null || exchangeRate <= 0
                Text(
                    when {
                        noRate -> stringResource(rateMissing)
                        usd == null -> stringResource(R.string.item_usd_rate_is, usdText(exchangeRate))
                        else -> stringResource(R.string.item_usd_equals, usdToSyp(usd, exchangeRate!!), usdText(exchangeRate))
                    },
                    color = if (noRate) Rahal.colors.danger else Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(top = Rahal.space.xs),
                )
            }

            // ══════════════════════════════════════════════════════════════
            // **٣ · قسمُ السوق — مربّعاتٌ بصورها في الصفحة نفسِها، ثلاثةٌ بكلّ سطر**
            // ══════════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-١٠-٢٩: «بلا القائمة المنسدلة… بالصفحة نفسها صور، كل سطر
            //  ياخذ ٣ أقسام».) **فلا نافذةَ تُفتح** — يرى الأقسامَ ويضغط.
            Spacer(Modifier.height(Rahal.space.md))
            val missing = sectionHint != 0 && d.platformSectionId.isEmpty()
            Text(
                stringResource(sectionLabel),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.labelLarge,
                color = if (missing) Rahal.colors.accentInk else Rahal.colors.ink,
            )
            Spacer(Modifier.height(Rahal.space.sm))
            if (sections.isEmpty()) {
                Text(stringResource(R.string.item_section_empty), color = Rahal.colors.inkMuted)
            }
            sections.chunked(3).forEach { row ->
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                ) {
                    row.forEach { sec ->
                        SectionTile(
                            sec = sec,
                            on = sec.id == d.platformSectionId,
                            media = media,
                            enabled = !busy,
                            modifier = Modifier.weight(1f),
                        ) { onEdit { it.copy(platformSectionId = sec.id) } }
                    }
                    repeat(3 - row.size) { Spacer(Modifier.weight(1f)) }
                }
                Spacer(Modifier.height(Rahal.space.sm))
            }

            // ══════════════════════════════════════════════════════════════
            // **٤ · الإضافاتُ والخيارات**
            // ══════════════════════════════════════════════════════════════
            ModifiersEditor(
                d = d,
                busy = busy,
                presets = presets,
                onEdit = onEdit,
                itemPrice = d.itemPriceSyp(exchangeRate),
                onAddPreset = { p ->
                    val (g, lost) = p.toGroup().forItemCurrency(d.isUsd, exchangeRate)
                    if (lost) rateLost = true
                    onEdit { dd -> dd.copy(modifiers = dd.modifiers + g) }
                },
            )
            if (rateLost) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    stringResource(R.string.mod_usd_no_rate),
                    color = Rahal.colors.danger,
                    style = MaterialTheme.typography.bodySmall,
                )
            }

            // ══════════════════════════════════════════════════════════════
            // **٥ · النسخُ والحذف آخرَ الصفحة** — والحفظُ في الشريط الثابت تحتها.
            // ══════════════════════════════════════════════════════════════
            if (d.itemId.isNotEmpty() && !readOnly) {
                Spacer(Modifier.height(Rahal.space.xl))
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                    // **قميصٌ بخمسة ألوان يُعمل مرّةً ويُنسخ** (٢٠٢٦-١٠-٢٧).
                    if (onDuplicate != null) {
                        RahalOutlineButton(
                            onClick = { onDuplicate() },
                            enabled = !busy && !dirty,
                            modifier = Modifier.weight(1f),
                        ) { IconText(R.drawable.ic_copy, stringResource(R.string.item_duplicate)) }
                    }
                    RahalOutlineButton(
                        onClick = { confirmDelete = true },
                        enabled = !busy,
                        tone = Tone.Danger,
                        modifier = Modifier.weight(1f),
                    ) { IconText(R.drawable.ic_trash, stringResource(R.string.item_delete)) }
                }
                if (dirty && onDuplicate != null) {
                    Spacer(Modifier.height(Rahal.space.xs))
                    Text(
                        stringResource(R.string.item_duplicate_save_first),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
            if (confirmDelete) {
                ConfirmDialog(
                    title = stringResource(R.string.item_delete),
                    body = stringResource(R.string.item_delete_ask),
                    confirm = stringResource(R.string.item_delete),
                    onConfirm = onDelete,
                    onDismiss = { confirmDelete = false },
                )
            }
            Spacer(Modifier.height(Rahal.space.xl))
        }
        // ── الشريطُ الثابت: سببُ رفض الحفظ فوقه، ثمّ «حفظ» و«إلغاء» ──
        HorizontalDivider(thickness = Rahal.stroke.hair, color = Rahal.colors.line)
        Column(
            Modifier
                .fillMaxWidth()
                .background(Rahal.colors.canvas)
                .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.sm),
        ) {
            if (error.isNotEmpty()) {
                Note(error, Rahal.colors.danger)
                Spacer(Modifier.height(Rahal.space.sm))
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                if (!readOnly) {
                    RahalButton(
                        onClick = { onSave() },
                        enabled = !busy && d.name.trim().length <= ItemLimits.nameMax,
                        modifier = Modifier.weight(1f),
                    ) { IconText(R.drawable.ic_check_mark, stringResource(R.string.act_save)) }
                }
                RahalOutlineButton(
                    onClick = { onCancel() },
                    enabled = !busy,
                    modifier = Modifier.weight(1f),
                ) { IconText(R.drawable.ic_close, stringResource(R.string.act_cancel)) }
            }
        }
    }
}

/**
 * **صورةُ الصنف** — مربّعٌ يُضغط فيفتح الاختيارَ والقصّ، وشريطٌ أسفلَه يقول
 * ما يفعله الضغط، **وزرُّ ✕ صغيرٌ في زاويته يُزيلها** إن وُجدت.
 */
@Composable
private fun ItemPhoto(
    d: ItemDraft,
    busy: Boolean,
    media: (String?) -> String?,
    onPick: () -> Unit,
    onClear: () -> Unit,
) {
    val has = d.imageThumb != null || !d.imageMediaId.isNullOrEmpty()
    Box(
        Modifier
            .size(PhotoSize)
            .clip(Rahal.shape.md)
            .clickable(enabled = !busy) { onPick() },
    ) {
        RemoteImage(
            url = media(d.imageThumb),
            name = d.name.ifBlank { "?" },
            modifier = Modifier.fillMaxSize(),
            // **الصورة كاملة** — طوليّةٌ أو عريضةٌ لا تُقصّ في المعاينة.
            contentScale = androidx.compose.ui.layout.ContentScale.Fit,
        )
        Row(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(Rahal.colors.panel.copy(alpha = 0.72f))
                .padding(vertical = Rahal.space.xs),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_photo),
                contentDescription = null,
                tint = Rahal.colors.onPanel,
                modifier = Modifier.size(ButtonIcon),
            )
            Spacer(Modifier.width(Rahal.space.xs))
            Text(
                stringResource(if (has) R.string.item_photo_change else R.string.item_pick_image),
                color = Rahal.colors.onPanel,
                style = MaterialTheme.typography.labelMedium,
            )
        }
        if (has) {
            IconButton(
                onClick = onClear,
                enabled = !busy,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(Rahal.space.xs)
                    .size(TouchTarget),
            ) {
                Box(
                    Modifier
                        .size(28.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.panel.copy(alpha = 0.72f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_close),
                        contentDescription = stringResource(R.string.item_clear_image),
                        tint = Rahal.colors.onPanel,
                        modifier = Modifier.size(ButtonIcon),
                    )
                }
            }
        }
    }
}

/**
 * **دائرتا العملة** — «ل.س» و«$»، والمختارةُ ممتلئةٌ بلون المنصّة.
 * (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩.) **و`CurrencySwitch` أسفلَه باقٍ** لمبلغ
 * «لدي توصيلة» — لم يُطلب تغييرُه هناك.
 */
@Composable
private fun CurrencyCircles(selected: String, enabled: Boolean, onSelect: (String) -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        listOf(CURRENCY_SYP to R.string.item_currency_syp, CURRENCY_USD to R.string.item_currency_usd_short).forEach { (code, label) ->
            val on = selected == code
            Box(
                Modifier
                    .size(CurrencyCircle)
                    .clip(CircleShape)
                    .background(if (on) Rahal.colors.brand else Rahal.colors.field)
                    .border(Rahal.stroke.hair, if (on) Rahal.colors.brand else Rahal.colors.line, CircleShape)
                    .selectable(selected = on, enabled = enabled, role = Role.RadioButton) { onSelect(code) },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    stringResource(label),
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall,
                )
            }
        }
    }
}

/**
 * **مربّعُ قسم السوق** — صورتُه واسمُه، **و✓ وإطارٌ أزرقُ على المختار.**
 * (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩.) **وهو هنا لا في تطبيق المتجر** — فالمندوبُ يراه كما يراه صاحبُ المتجر.
 */
@Composable
private fun SectionTile(
    sec: SectionRef,
    on: Boolean,
    media: (String?) -> String?,
    enabled: Boolean,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    Column(
        modifier
            .clip(Rahal.shape.md)
            .border(
                if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                if (on) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.06f) else Rahal.colors.canvas)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(Rahal.space.xs),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box {
            RemoteImage(
                url = media(sec.imageThumbUrl ?: sec.imageUrl),
                name = sec.name,
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .clip(Rahal.shape.sm),
            )
            if (on) {
                Box(
                    Modifier
                        .align(Alignment.TopEnd)
                        .padding(Rahal.space.xs)
                        .size(24.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.brand),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_check_mark),
                        contentDescription = null,
                        tint = Rahal.colors.onBrand,
                        modifier = Modifier.size(16.dp),
                    )
                }
            }
        }
        Spacer(Modifier.height(Rahal.space.xs))
        Text(
            sec.name,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
            color = if (on) Rahal.colors.brand else Rahal.colors.ink,
            textAlign = TextAlign.Center,
            minLines = 2,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * ════════════════════════════════════════
 * **الإضافاتُ والمجموعات — «مشروبٌ غازيّ» و«جبنةٌ إضافيّة»**
 * ════════════════════════════════════════
 *
 * **و«اختر واحداً من ثلاثة» غيرُ «أضف ما شئت من خمسة»** — والفرقُ
 * `minSelect` و`maxSelect`، **ويُعرضان رقاقةً تُقرأ لا رقمين.**
 *
 * **وكلُّ المجموعات بشكلٍ واحد**: الحجمُ والوزنُ والنمرةُ والمقاسُ واللونُ
 * والإضافاتُ و«بدون» والخاصّة — رأسٌ فيه اسمُها ورقاقةُ طريقة سعرها وزرُّ
 * حذفها، ثمّ سطرٌ لكلّ خيار.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ModifiersEditor(
    d: ItemDraft,
    busy: Boolean,
    presets: List<OptionPreset>?,
    onEdit: ((ItemDraft) -> ItemDraft) -> Unit,
    /** **سعرُ الصنف بالليرة** — لمعاينة أسعار الأوزان؛ و`null` لم يُكتب. */
    itemPrice: Long?,
    /** **مجموعةٌ جاهزةٌ تُضاف** — بعملة الصنف (انظر `forItemCurrency`). */
    onAddPreset: (OptionPreset) -> Unit,
) {
    Spacer(Modifier.height(Rahal.space.xl))
    Text(
        stringResource(R.string.mod_title),
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.titleMedium,
    )
    // **ولا سطرَ شرح** (ملاحظةُ المالك ٢٠٢٦-١٠-٢٩: «ما في داعي للشرح»).
    Spacer(Modifier.height(Rahal.space.sm))

    // ══════════════════════════════════════════════════════════════
    // **«أضف مشروباتي»** (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩، هجرة 0483)
    // ══════════════════════════════════════════════════════════════
    // قائمةُ المشروبات يكتبها المتجرُ مرّةً («قائمة المشروبات»)، **وهذا
    // المفتاحُ يقول: أتظهر مع هذا الصنف أم لا.** والافتراضُ نعم كما كان.
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.brand.copy(alpha = 0.06f))
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.mod_drinks_switch),
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyLarge,
            )
            Text(
                stringResource(if (d.storeDrinks) R.string.mod_drinks_on else R.string.mod_drinks_off),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Switch(
            checked = d.storeDrinks,
            enabled = !busy,
            onCheckedChange = { on -> onEdit { it.copy(storeDrinks = on) } },
        )
    }
    Spacer(Modifier.height(Rahal.space.md))

    // ══════════════════════════════════════════════════════════════
    // **مجموعاتٌ جاهزةٌ من تصنيف المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — **مربّعاتٌ
    // متساويةٌ فوق بعض** (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩: لا رقاقاتٌ بعروضٍ مختلفة).
    // **و`null` لم تُقرأ** فتبقى القديمة. **والمجموعةُ تُعرف باسمها.**
    // **ومرآتُها `ItemEditor.tsx` في الويب.**
    // ══════════════════════════════════════════════════════════════
    val list = presets ?: legacyPresets()
    val names = list.map { it.name.trim() }.toSet()
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        list.forEach { p ->
            val on = d.modifiers.any { it.name.trim() == p.name.trim() }
            PresetBox(
                name = p.name,
                values = p.options.joinToString(" · ") { it.name },
                on = on,
                enabled = !busy,
            ) {
                if (!on) onAddPreset(p)
                else onEdit { dd -> dd.copy(modifiers = dd.modifiers.filterNot { it.name.trim() == p.name.trim() }) }
            }
        }
    }
    Spacer(Modifier.height(Rahal.space.sm))
    // **ومجموعةٌ خاصّةٌ لما لم يخطر لنا** — اسمُها وخياراتُها بيده.
    val customName = stringResource(R.string.mod_extras_name)
    RahalOutlineButton(onClick = {
        onEdit { dd -> dd.copy(modifiers = dd.modifiers + ModifierGroup(name = customName, minSelect = 0, maxSelect = 99, priceMode = PRICE_MODE_PRICED)) }
    }, enabled = !busy) { IconText(R.drawable.ic_plus, stringResource(R.string.mod_custom_btn)) }

    d.modifiers.forEachIndexed { gi, g ->
        val updateGroup: ((ModifierGroup) -> ModifierGroup) -> Unit = { f ->
            onEdit { dd -> dd.copy(modifiers = dd.modifiers.replaceAt(gi, f)) }
        }
        Spacer(Modifier.height(Rahal.space.md))
        GroupCard(
            g = g,
            isPreset = g.name.trim() in names,
            usd = d.isUsd,
            busy = busy,
            itemPrice = itemPrice,
            update = updateGroup,
            onRemove = { onEdit { dd -> dd.copy(modifiers = dd.modifiers.filterIndexed { i, _ -> i != gi }) } },
        )
    }
}

/**
 * **بطاقةُ مجموعة** — رأسٌ (اسمٌ ورقاقتان وزرُّ حذف) ثمّ أسطرُ الخيارات
 * ثمّ «أضف خياراً».
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun GroupCard(
    g: ModifierGroup,
    isPreset: Boolean,
    usd: Boolean,
    busy: Boolean,
    itemPrice: Long?,
    update: ((ModifierGroup) -> ModifierGroup) -> Unit,
    onRemove: () -> Unit,
) {
    // **طريقةُ السعر** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — «بلا سعر» لا حقلَ سعرٍ فيها،
    // و«حسب الوزن» أوزانٌ بالغرام وسعرُها محسوب.
    val mode = g.priceMode ?: PRICE_MODE_PRICED
    Card {
        // ── الرأس ──
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    g.name.ifBlank { stringResource(R.string.mod_custom_untitled) },
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(Rahal.space.xs))
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
                    verticalArrangement = Arrangement.spacedBy(Rahal.space.xs),
                ) {
                    when (mode) {
                        PRICE_MODE_NONE -> TagChip(stringResource(R.string.mod_chip_none), Rahal.colors.inkMuted)
                        PRICE_MODE_WEIGHT -> TagChip(stringResource(R.string.mod_chip_weight), Rahal.colors.accentInk)
                        else -> TagChip(stringResource(R.string.mod_chip_priced), Rahal.colors.brand)
                    }
                    TagChip(
                        stringResource(if (g.minSelect > 0) R.string.mod_chip_one else R.string.mod_chip_any),
                        Rahal.colors.inkMuted,
                    )
                }
            }
            Spacer(Modifier.width(Rahal.space.sm))
            // **وحذفُ المجموعة رماديّ** — الأحمرُ لحذف الصنف وحدَه (مراجعةُ ٢٠٢٦-١٠-٢٩).
            IconButton(onClick = onRemove, enabled = !busy, modifier = Modifier.size(TouchTarget)) {
                Icon(
                    painter = painterResource(R.drawable.ic_trash),
                    contentDescription = stringResource(R.string.mod_remove_group),
                    tint = Rahal.colors.inkMuted,
                )
            }
        }

        // ── المجموعةُ الخاصّة: اسمُها وقاعدتُها بيده ──
        if (!isPreset) {
            Spacer(Modifier.height(Rahal.space.sm))
            OutlinedTextField(
                value = g.name,
                onValueChange = { v -> update { it.copy(name = v) } },
                label = { Text(stringResource(R.string.mn_group_name)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.mod_rule_switch),
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.weight(1f),
                )
                Switch(
                    checked = g.minSelect > 0,
                    enabled = !busy,
                    onCheckedChange = { one ->
                        update { if (one) it.copy(minSelect = 1, maxSelect = 1) else it.copy(minSelect = 0, maxSelect = 99) }
                    },
                )
            }
        }

        Spacer(Modifier.height(Rahal.space.sm))
        if (mode == PRICE_MODE_WEIGHT) {
            WeightOptions(g, itemPrice, busy, update)
        } else {
            // **ولا عناوينَ أعمدة** — الاسمُ بسطرٍ كامل، والسعرُ تحته وتسميتُه داخلَ حقله.
            val priceHint = stringResource(if (usd) R.string.mod_opt_price_usd else R.string.mod_opt_price)
            g.options.forEachIndexed { oi, o ->
                val setOption: ((ModifierOption) -> ModifierOption) -> Unit = { f ->
                    update { gg -> gg.copy(options = gg.options.replaceAt(oi, f)) }
                }
                val priceField: @Composable RowScope.() -> Unit = {
                    RowField(
                        // **وبعملة صنفها** (٢٠٢٦-١٠-١٢): دولاريٌّ يُكتب بالدولار، **وبكسوره.**
                        value = if (usd) o.usdText else if (o.priceDelta == 0L) "" else o.priceDelta.toString(),
                        onChange = { v ->
                            if (usd) {
                                val t = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(8)
                                setOption { it.copy(usdText = t) }
                            } else {
                                val n = v.filter { it.isDigit() }.take(12).toLongOrNull() ?: 0L
                                setOption { it.copy(priceDelta = n) }
                            }
                        },
                        placeholder = priceHint,
                        keyboard = if (usd) KeyboardType.Decimal else KeyboardType.Number,
                        enabled = !busy,
                        modifier = Modifier.width(PriceFieldWidth),
                    )
                }
                OptionRow(
                    available = o.available,
                    busy = busy,
                    onAvailable = { av -> setOption { it.copy(available = av) } },
                    onDelete = { update { gg -> gg.copy(options = gg.options.filterIndexed { i, _ -> i != oi }) } },
                    second = if (mode == PRICE_MODE_PRICED) priceField else null,
                ) {
                    RowField(
                        value = o.name,
                        onChange = { v -> setOption { it.copy(name = v) } },
                        placeholder = stringResource(R.string.mn_option_name),
                        enabled = !busy,
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        }

        Spacer(Modifier.height(Rahal.space.sm))
        RahalOutlineButton(
            onClick = { update { gg -> gg.copy(options = gg.options + ModifierOption(name = "")) } },
            enabled = !busy,
        ) {
            IconText(
                R.drawable.ic_plus,
                stringResource(if (mode == PRICE_MODE_WEIGHT) R.string.mod_weight_add else R.string.mod_add_option),
            )
        }
    }
}

/**
 * **سطرُ خيارٍ واحد** — حقولُه (يمرّرها النداء) ثمّ مفتاحُ التوفّر ثمّ زرُّ الحذف.
 *
 * **«نمرة ٤٢ خلصت» — يطفئها وحدَها** (طلبُ المالك ٢٠٢٦-١٠-٢٧): والصنفُ يبقى
 * يُباع، **وعند الزبون تطلع رماديّةً لا تُختار.**
 */
@Composable
private fun OptionRow(
    available: Boolean,
    busy: Boolean,
    onAvailable: (Boolean) -> Unit,
    onDelete: () -> Unit,
    /**
     * **السطرُ الثاني** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩): السعرُ والمفتاحُ تحت الاسم
     * فيأخذ الاسمُ عرضَ السطر كلَّه. و`null` = سطرٌ واحدٌ والمفتاحُ بجانب الحقول.
     */
    second: (@Composable RowScope.() -> Unit)? = null,
    fields: @Composable RowScope.() -> Unit,
) {
    Column(Modifier.fillMaxWidth().padding(vertical = Rahal.space.xs)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            fields()
            if (second == null) {
                Spacer(Modifier.width(Rahal.space.xs))
                AvailSwitch(available, busy, onAvailable)
            }
            // **والحذفُ رماديّ** — الأحمرُ لحذف الصنف وحدَه.
            IconButton(onClick = onDelete, enabled = !busy, modifier = Modifier.size(TouchTarget)) {
                Icon(
                    painter = painterResource(R.drawable.ic_trash),
                    contentDescription = stringResource(R.string.mod_opt_delete),
                    tint = Rahal.colors.inkMuted,
                )
            }
        }
        if (second != null) {
            Spacer(Modifier.height(Rahal.space.xs))
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                second()
                Spacer(Modifier.weight(1f))
                AvailSwitch(available, busy, onAvailable)
                // **محاذاةٌ مع السطر الأوّل** — المفتاحُ يقف حيث يقف تحت السلّة.
                Spacer(Modifier.width(TouchTarget))
            }
        }
    }
}

/**
 * **مفتاحُ توفّر الخيار وكلمتُه بجانبه** — و«غير متوفر» رماديّة لا حمراء:
 * غيرُ المتوفّر حالٌ لا خطأ (كما في شبكة الأصناف).
 */
@Composable
private fun AvailSwitch(available: Boolean, busy: Boolean, onAvailable: (Boolean) -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(if (available) R.string.mn_available else R.string.mn_unavailable),
            color = if (available) Rahal.colors.success else Rahal.colors.inkMuted,
            style = MaterialTheme.typography.labelSmall,
            maxLines = 1,
        )
        Spacer(Modifier.width(Rahal.space.xs))
        Switch(checked = available, enabled = !busy, onCheckedChange = onAvailable)
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«حسب الوزن»** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * سعرُ الصنف لوزنٍ أساسيّ (١٠٠ غ افتراضاً)، والأوزانُ أرقامٌ بالغرام أو
 * أزرارٌ سريعة، **وسعرُ كلّ وزنٍ معاينةٌ هنا والخادمُ يحسبه ويحفظه.**
 * **ومرآتُها `ItemEditor.tsx` في الويب.**
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun WeightOptions(
    g: ModifierGroup,
    itemPrice: Long?,
    busy: Boolean,
    update: ((ModifierGroup) -> ModifierGroup) -> Unit,
) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.mod_weight_base_label),
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        RowField(
            value = g.baseGrams?.toString() ?: "",
            onChange = { v ->
                val n = v.filter(Char::isDigit).take(6).toIntOrNull()?.takeIf { it > 0 }
                update { it.copy(baseGrams = n) }
            },
            placeholder = DEFAULT_BASE_GRAMS.toString(),
            keyboard = KeyboardType.Number,
            suffix = stringResource(R.string.mod_weight_unit),
            enabled = !busy,
            modifier = Modifier.width(NumberFieldWidth),
        )
    }
    Text(
        stringResource(R.string.mod_weight_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
    )
    Spacer(Modifier.height(Rahal.space.xs))
    g.options.forEachIndexed { oi, o ->
        OptionRow(
            available = o.available,
            busy = busy,
            onAvailable = { av -> update { gg -> gg.copy(options = gg.options.replaceAt(oi) { it.copy(available = av) }) } },
            onDelete = { update { gg -> gg.copy(options = gg.options.filterIndexed { i, _ -> i != oi }) } },
        ) {
            RowField(
                value = o.grams?.toString() ?: "",
                onChange = { v ->
                    val n = v.filter(Char::isDigit).take(7).toIntOrNull()?.takeIf { it > 0 }
                    update { gg ->
                        gg.copy(options = gg.options.replaceAt(oi) { it.copy(grams = n, name = n?.let(::weightName) ?: "") })
                    }
                },
                placeholder = "250",
                keyboard = KeyboardType.Number,
                suffix = stringResource(R.string.mod_weight_unit),
                enabled = !busy,
                modifier = Modifier.width(NumberFieldWidth),
            )
            Spacer(Modifier.width(Rahal.space.sm))
            Column(Modifier.weight(1f)) {
                o.grams?.let { gr ->
                    Text(
                        weightName(gr),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    if (itemPrice != null) {
                        TagChip(
                            stringResource(R.string.mod_weight_price, money(weightPrice(itemPrice, gr, g.baseGrams))),
                            Rahal.colors.brand,
                        )
                    }
                }
            }
        }
    }
    // **أوزانٌ سريعة** — ٢٥٠ غ ونصف كيلو وكيلو؛ والموجودُ لا يُكرَّر.
    Spacer(Modifier.height(Rahal.space.xs))
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        Text(
            stringResource(R.string.mod_weight_quick),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.align(Alignment.CenterVertically),
        )
        QUICK_GRAMS.forEach { q ->
            val has = g.options.any { it.grams == q }
            PresetChip(weightName(q), on = has, enabled = !busy) {
                if (!has) update { gg -> gg.copy(options = gg.options + ModifierOption(name = weightName(q), grams = q)) }
            }
        }
    }
}

/**
 * **حقلُ السطر** — ارتفاعٌ واحدٌ لكلّ حقول الخيارات (الاسم والسعر والغرام)،
 * **بلا تسميةٍ عائمةٍ تكبّر الحقل**: العنوانُ فوق العمود، والتلميحُ داخل الحقل.
 */
@Composable
internal fun RowField(
    value: String,
    onChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    keyboard: KeyboardType = KeyboardType.Text,
    suffix: String = "",
    enabled: Boolean = true,
) {
    BasicTextField(
        value = value,
        onValueChange = onChange,
        singleLine = true,
        enabled = enabled,
        keyboardOptions = KeyboardOptions(keyboardType = keyboard),
        textStyle = MaterialTheme.typography.bodyMedium.copy(color = Rahal.colors.ink),
        cursorBrush = SolidColor(Rahal.colors.brand),
        modifier = modifier,
        decorationBox = { inner ->
            Row(
                Modifier
                    .height(RowFieldHeight)
                    .clip(Rahal.shape.sm)
                    .background(Rahal.colors.field)
                    .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.sm)
                    .padding(horizontal = Rahal.space.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(Modifier.weight(1f), contentAlignment = Alignment.CenterStart) {
                    if (value.isEmpty()) {
                        Text(
                            placeholder,
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodyMedium,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    inner()
                }
                if (suffix.isNotEmpty()) {
                    Spacer(Modifier.width(Rahal.space.xs))
                    Text(suffix, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
                }
            }
        },
    )
}

/** **عنوانُ عمودٍ فوق أسطر الخيارات.** */
@Composable
internal fun ColumnLabel(text: String, modifier: Modifier) {
    Text(
        text,
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.labelSmall,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        modifier = modifier,
    )
}

/** **رقاقةٌ تُقرأ** — طريقةُ السعر أو القاعدة أو سعرُ الوزن. */
@Composable
internal fun TagChip(text: String, color: Color) {
    Text(
        text,
        color = color,
        style = MaterialTheme.typography.labelSmall,
        fontWeight = FontWeight.Bold,
        maxLines = 1,
        modifier = Modifier
            .clip(Rahal.shape.pill)
            .background(color.copy(alpha = 0.10f))
            .padding(horizontal = Rahal.space.sm, vertical = 2.dp),
    )
}

/**
 * **مربّعُ مجموعةٍ جاهزة** — سطرٌ كاملُ العرض بارتفاعٍ واحد: علامةُ ✓ أو ＋،
 * ثمّ الاسمُ وتحته معاينةُ قيمها في سطرٍ واحد. **والمختارُ بحدٍّ وأرضٍ بلون المنصّة.**
 */
@Composable
private fun PresetBox(name: String, values: String, on: Boolean, enabled: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .height(PresetBoxHeight)
            .clip(Rahal.shape.md)
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.08f) else Rahal.colors.canvas)
            .border(
                if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                if (on) Rahal.colors.brand else Rahal.colors.line,
                Rahal.shape.md,
            )
            .selectable(selected = on, enabled = enabled, role = Role.Checkbox, onClick = onClick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(28.dp)
                .clip(CircleShape)
                .background(if (on) Rahal.colors.brand else Rahal.colors.brand.copy(alpha = 0.08f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(if (on) R.drawable.ic_check_mark else R.drawable.ic_plus),
                contentDescription = null,
                tint = if (on) Rahal.colors.onBrand else Rahal.colors.brand,
                modifier = Modifier.size(16.dp),
            )
        }
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                name,
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (values.isNotBlank()) {
                Text(
                    values,
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

/** **رقاقةٌ تُضغط** — وزنٌ سريع، و✓ على المختار. */
@Composable
private fun PresetChip(text: String, on: Boolean, enabled: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .heightIn(min = 40.dp)
            .clip(Rahal.shape.pill)
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.10f) else Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (on) Rahal.colors.brand else Rahal.colors.line, Rahal.shape.pill)
            .selectable(selected = on, enabled = enabled, role = Role.Checkbox, onClick = onClick)
            .padding(horizontal = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(if (on) R.drawable.ic_check_mark else R.drawable.ic_plus),
            contentDescription = null,
            tint = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
            modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.width(Rahal.space.xs))
        Text(
            text,
            color = if (on) Rahal.colors.brand else Rahal.colors.ink,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/** **أيقونةٌ ونصٌّ داخل زرّ** — لونُهما من الزرّ نفسِه. */
@Composable
internal fun RowScope.IconText(@DrawableRes icon: Int, text: String) {
    Icon(
        painter = painterResource(icon),
        contentDescription = null,
        modifier = Modifier.size(ButtonIcon),
    )
    Spacer(Modifier.width(Rahal.space.xs))
    Text(text, fontWeight = FontWeight.Bold, maxLines = 1, overflow = TextOverflow.Ellipsis)
}

/**
 * **القوالبُ القديمة** — لمحرّكٍ لم يُحدَّث أو قراءةٍ فشلت، كما كانت قبل
 * ٢٠٢٦-١٠-٢٧: الحجمُ والوزن (والإضافاتُ صارت «مجموعة خاصة»).
 */
@Composable
private fun legacyPresets(): List<OptionPreset> = listOf(
    OptionPreset(
        name = stringResource(R.string.mod_preset_size), minSelect = 1, maxSelect = 1,
        options = listOf(
            PresetOption(stringResource(R.string.mod_opt_normal)),
            PresetOption(stringResource(R.string.mod_opt_double)),
        ),
    ),
    OptionPreset(
        name = stringResource(R.string.mod_preset_weight), minSelect = 1, maxSelect = 1,
        options = listOf(
            PresetOption(stringResource(R.string.mod_opt_half_kg)),
            PresetOption(stringResource(R.string.mod_opt_kg)),
        ),
    ),
)

/** **يستبدل عنصراً بموضعه** — ولا يمسّ ما سواه. */
private inline fun <T> List<T>.replaceAt(i: Int, block: (T) -> T): List<T> =
    mapIndexed { idx, v -> if (idx == i) block(v) else v }

/**
 * **زرُّ العملة** — «ل.س» و«$» جنباً إلى جنب، والمختارُ بلون المنصّة.
 * (طلبُ المالك ٢٠٢٦-١٠-١٢: «زر ذكي سعر بالسوري او دولار».)
 *
 * **وهو زرُّ مبلغ «لدي توصيلة»** (قرارُ المالك ٢٠٢٦-١٠-٢٠). ومحرّرُ الصنف صار
 * بدائرتين (`CurrencyCircles`) بملاحظات المالك ٢٠٢٦-١٠-٢٩.
 */
@Composable
fun CurrencySwitch(selected: String, enabled: Boolean, onSelect: (String) -> Unit) {
    Row(
        Modifier
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf(CURRENCY_SYP to R.string.item_currency_syp, CURRENCY_USD to R.string.item_currency_usd).forEach { (code, label) ->
            val on = selected == code
            Box(
                Modifier
                    .clip(Rahal.shape.md)
                    .background(if (on) Rahal.colors.brand else Rahal.colors.canvas)
                    .clickable(enabled = enabled) { onSelect(code) }
                    .padding(horizontal = 18.dp, vertical = 8.dp),
            ) {
                Text(
                    stringResource(label),
                    color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
    }
}
