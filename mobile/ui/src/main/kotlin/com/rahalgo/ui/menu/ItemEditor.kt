package com.rahalgo.ui.menu

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نموذجُ تحرير الصنف — واحدٌ للمتجر وللمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «الأصناف بالمندوب والمتجر لازم تكون
 *  مركزيّة… وهي لنفس الغرض».)
 *
 * **كان نموذجان بمئتين وستّين سطراً بينهما** — يفعلان الشيءَ نفسَه:
 * الاسمُ والوصفُ والسعرُ والصورةُ وقسمُ السوق والإضافات.
 *
 * **وثمنُهما دُفع مرّتين في يومٍ واحد** (٢٠٢٦-٠٨-٢٩): صورةٌ لا تظهر
 * أُصلحت في الاثنين، **وزرٌّ بلا إطارٍ أُصلح في واحدٍ ونُسي في الآخر**
 * حتّى رآه المالك.
 *
 * # ولا يعرف هذا المكوّنُ محرّكاً
 *
 * **يأخذ رسماً وأفعالاً ويعيد رسماً** — ولا ينادي شبكةً ولا يعرف
 * `MerchantApi` من `RepApi`. **فالتطبيقُ يصل ما يشاء.**
 */

import com.rahalgo.ui.R
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Card
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.rememberImagePicker
import androidx.annotation.StringRes
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Checkbox
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.MenuItem
import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.ModifierOption
import com.rahalgo.shared.merchant.PlatformSectionRef as SectionRef

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الصنف — كنموذج الويب حرفاً بحرف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «بنفس طريقة الويب: إضافةُ صورةٍ وتعديلُ
 *  معلوماتٍ ومتوفّر/غير متوفّر وحذف».)
 *
 * # والسعرُ سعرُ الشراء لا البيع
 *
 * **ولو حُرِّر سعرُ البيع لَضاع الهامشُ في أوّل تعديل**: يفتح الصنفَ فيرى
 * رقماً أكبرَ من سعره، **فيحفظ فيصير سعرُ شرائه ذاك**، ويُضاف عليه
 * الهامشُ من جديد — **ورقمٌ يرتفع بكلّ فتحةٍ للنافذة.**
 *
 * # ولا مفتاحَ توفّرٍ هنا
 *
 * **يُقلب من نقطة السطر وحدَها** — **وحقلٌ يُكتب من موضعين يمحو أحدُهما
 * ما فعله الآخر**: من أوقف صنفاً ثمّ عدّل اسمَه أعاده متوفّراً.
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
    // ══════════════════════════════════════════════════════════════════
    // **وثلاثةُ ألفاظٍ تُمرَّر ولا تُثبَّت**
    // ══════════════════════════════════════════════════════════════════
    //
    // **لأنّ الجمهورين مختلفان**: المتجرُ يكتب سعرَ نفسِه فيقرأ «السعر»
    // عامّاً (قرارُ المالك: «لا شراءَ ولا مبيع، أفضلُ وأعمّ»)،
    // **والمندوبُ يكتب سعرَ متجرِ غيرِه** فيقرأ «سعر المتجر».
    //
    // **ونصوصُ المندوب مطابقةٌ للويب حرفاً** (قرارُ المالك ٢٠٢٦-٠٨-١٨:
    // «لا يجوز أن يشعر الشخصُ بالفرق بين الويب والتطبيق أصلاً»)
    // **ويحرسها `pnpm check:menu-parity`.**
    //
    // **وتوحيدُ المحرّر ٢٠٢٦-٠٨-٣٠ ثبّت ألفاظَ المتجر على الاثنين**
    // فانكسرت المطابقة: صار المندوبُ يقرأ «السعر» و«وصف مختصر» و«قسم
    // السوق» بدل «سعر المتجر» و«الوصف» و«قسم المنصة». **والحارسُ لم
    // يرَها** — يقرأ الملفَّ الذي لم تعد الشاشةُ تقرؤه.
    //
    // **فما اختلف يُمرَّر، وما اتّفق يبقى في المحرّر.**
    @StringRes priceLabel: Int = R.string.item_price,
    @StringRes descLabel: Int = R.string.item_desc,
    @StringRes sectionLabel: Int = R.string.item_section,
    /** **تنبيهُ «بلا قسم»** — وصفرٌ يعني لا تنبيه. */
    @StringRes sectionHint: Int = 0,
) {
    val pick = rememberImagePicker { bytes -> onPickImage(bytes) }
    var confirmDelete by remember { mutableStateOf(false) }

    Screen {
        ScreenTitle(
            stringResource(if (d.itemId.isEmpty()) R.string.mn_item_new else R.string.mn_item_edit),
            "",
        )

        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            RemoteImage(
                url = media(d.imageThumb),
                name = d.name.ifBlank { "?" },
                modifier = Modifier
                    .size(84.dp)
                    .clip(Rahal.shape.sm)
                    .clickable(enabled = !busy) { pick() },
            )
            Spacer(Modifier.size(12.dp))
            Column {
                // **واختيارُ الصورة فعلٌ لا عنوان** — انظر زرَّ المجموعة.
                RahalOutlineButton(onClick = { pick() }, enabled = !busy) {
                    Text(stringResource(R.string.item_pick_image))
                }
                if (d.imageThumb != null || !d.imageMediaId.isNullOrEmpty()) {
                    RahalTextButton(onClick = { onClearImage() }, enabled = !busy) {
                        Text(
                            stringResource(R.string.item_clear_image),
                            color = Rahal.colors.inkMuted,
                        )
                    }
                }
            }
        }

        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = d.name,
            onValueChange = { v -> onEdit { it.copy(name = v) } },
            label = { Text(stringResource(R.string.mn_item_name)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = d.description,
            onValueChange = { v -> onEdit { it.copy(description = v) } },
            label = { Text(stringResource(descLabel)) },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = d.price,
            onValueChange = { v -> onEdit { it.copy(price = v.filter(Char::isDigit)) } },
            label = { Text(stringResource(priceLabel)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(sectionLabel),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelLarge,
        )
        Spacer(Modifier.height(4.dp))
        Card {
            sections.forEach { sec ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clickable { onEdit { it.copy(platformSectionId = sec.id) } }
                        .padding(vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        Modifier
                            .size(16.dp)
                            .clip(CircleShape)
                            .background(
                                if (sec.id == d.platformSectionId) Rahal.colors.brand
                                else Rahal.colors.line,
                            ),
                    )
                    Spacer(Modifier.size(10.dp))
                    Text(sec.name, style = MaterialTheme.typography.bodyMedium)
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وبلا قسمٍ يُحفظ الصنفُ ولا يراه زبون**
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يُمنع الحفظ**: لوحةُ الويب تسمح به، **ومنعُ التطبيقِ ما
        // يسمح به الويبُ فرقٌ بين شاشتين لعملٍ واحد.**
        //
        // **لكنّه يُقال قبل أن يُحفظ** — ومن بنى قائمةَ عميلِه كلَّها
        // بلا أقسامٍ لم يعرف لماذا لا تظهر في السوق، **ولا شيءَ في
        // الشاشة يدلّه.**
        //
        // (كشفه جردُ الحالات ٢٠٢٦-٠٨-٣٠.)
        if (sectionHint != 0 && d.platformSectionId.isEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(
                stringResource(sectionHint),
                color = Rahal.colors.accent,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        ModifiersEditor(d, onEdit)

        Spacer(Modifier.height(16.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(onClick = { onSave() }, enabled = !busy) {
                Text(stringResource(R.string.act_save))
            }
            RahalTextButton(onClick = { onCancel() }, enabled = !busy) {
                Text(stringResource(R.string.act_cancel))
            }
        }

        if (d.itemId.isNotEmpty()) {
            Spacer(Modifier.height(20.dp))
            if (!confirmDelete) {
                RahalTextButton(onClick = { confirmDelete = true }, enabled = !busy) {
                    Text(stringResource(R.string.item_delete), color = Rahal.colors.danger)
                }
            } else {
                Card {
                    Text(
                        stringResource(R.string.item_delete_ask),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Spacer(Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        RahalButton(
                            onClick = onDelete,
                            enabled = !busy,
                            tone = com.rahalgo.ui.Tone.Danger,
                        ) { Text(stringResource(R.string.item_delete)) }
                        RahalTextButton(onClick = { confirmDelete = false }) {
                            Text(stringResource(R.string.act_cancel))
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * ════════════════════════════════════════
 * **الإضافاتُ والمجموعات — «مشروبٌ غازيّ» و«جبنةٌ إضافيّة»**
 * ════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «شغلُ الإضافات والمجموعات الخاصّة بكلّ صنفٍ
 *  غيرُ موجود… الأكلُ دائماً الشخصُ يطلب شيئاً إضافيّاً أو عصير كولا
 *  عيران وهكذا. لازم تكون من نفس المتجر».)
 *
 * # ولا حارسَ يمنع الخلط
 *
 * **والإضافةُ مربوطةٌ بالصنف، والصنفُ لمتجرٍ واحد** — فلا يمكن بنيةً أن
 * تأتي من متجرٍ آخر. **والسائقُ يشتري من بابٍ واحد.**
 *
 * # ومجموعةٌ لا قائمةٌ مسطّحة
 *
 * **و«اختر واحداً من ثلاثة مشروبات» غيرُ «أضف ما شئت من خمسة»** —
 * والفرقُ `minSelect` و`maxSelect`. **ومن سطّحها جعل الزبونَ يختار
 * ثلاثةَ مشروباتٍ في طلبٍ واحد.**
 *
 * # ورقمان عاريان لا يفهمهما صاحبُ مطعم
 *
 * **فيُعرضان سؤالين**: «إلزاميّ؟» و«يختار أكثر من واحد؟».
 */
@Composable
private fun ModifiersEditor(d: ItemDraft, onEdit: ((ItemDraft) -> ItemDraft) -> Unit) {
    Spacer(Modifier.height(16.dp))
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            stringResource(R.string.mod_title),
            modifier = Modifier.weight(1f),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
        )
        // ══════════════════════════════════════════════════════════
        // **وزرُّ الإضافة يُرى زرّاً**
        // ══════════════════════════════════════════════════════════
        //
        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٩: «أضف مجموعة لازم يكون زرّاً مبيّناً
        //  ليعرف المستخدم أنّ هذا زرّ».)
        //
        // **و`RahalTextButton` نصٌّ ملوَّنٌ بلا إطارٍ ولا تعبئة** —
        // **يُقرأ عنواناً لا فعلاً**، فيمرّ عليه صاحبُ المتجر ولا يعرف
        // أنّ تحته باباً.
        //
        // **والإطارُ يفصل ما يُضغط عمّا يُقرأ** — وهو الفرقُ الوحيدُ
        // الذي تراه العينُ قبل أن تجرّب.
        RahalOutlineButton(onClick = {
            onEdit { it.copy(modifiers = it.modifiers + ModifierGroup(name = "")) }
        }) { Text(stringResource(R.string.mod_add_group)) }
    }
    Text(
        stringResource(R.string.mod_hint),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.bodySmall,
    )

    d.modifiers.forEachIndexed { gi, g ->
        Spacer(Modifier.height(10.dp))
        Card {
            Column(Modifier.padding(4.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        value = g.name,
                        onValueChange = { v ->
                            onEdit { dd ->
                                dd.copy(modifiers = dd.modifiers.replaceAt(gi) { it.copy(name = v) })
                            }
                        },
                        label = { Text(stringResource(R.string.mn_group_name)) },
                        singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    RahalTextButton(onClick = {
                        onEdit { dd ->
                            dd.copy(modifiers = dd.modifiers.filterIndexed { i, _ -> i != gi })
                        }
                    }) { Text(stringResource(R.string.act_delete), color = Rahal.colors.danger) }
                }

                Row(verticalAlignment = Alignment.CenterVertically) {
                    Checkbox(
                        checked = g.minSelect > 0,
                        onCheckedChange = { on ->
                            onEdit { dd ->
                                dd.copy(
                                    modifiers = dd.modifiers.replaceAt(gi) {
                                        it.copy(minSelect = if (on) 1 else 0)
                                    },
                                )
                            }
                        },
                    )
                    Text(
                        stringResource(R.string.mn_required),
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Spacer(Modifier.weight(1f))
                    Checkbox(
                        checked = g.maxSelect > 1,
                        onCheckedChange = { on ->
                            onEdit { dd ->
                                dd.copy(
                                    modifiers = dd.modifiers.replaceAt(gi) {
                                        it.copy(maxSelect = if (on) 99 else 1)
                                    },
                                )
                            }
                        },
                    )
                    Text(
                        stringResource(R.string.mod_multi),
                        style = MaterialTheme.typography.bodySmall,
                    )
                }

                g.options.forEachIndexed { oi, o ->
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        OutlinedTextField(
                            value = o.name,
                            onValueChange = { v ->
                                onEdit { dd ->
                                    dd.copy(
                                        modifiers = dd.modifiers.replaceAt(gi) { gg ->
                                            gg.copy(options = gg.options.replaceAt(oi) { it.copy(name = v) })
                                        },
                                    )
                                }
                            },
                            label = { Text(stringResource(R.string.mn_option_name)) },
                            singleLine = true,
                            modifier = Modifier.weight(1f),
                        )
                        Spacer(Modifier.width(6.dp))
                        OutlinedTextField(
                            value = if (o.priceDelta == 0L) "" else o.priceDelta.toString(),
                            onValueChange = { v ->
                                val n = v.filter { it.isDigit() }.toLongOrNull() ?: 0L
                                onEdit { dd ->
                                    dd.copy(
                                        modifiers = dd.modifiers.replaceAt(gi) { gg ->
                                            gg.copy(options = gg.options.replaceAt(oi) { it.copy(priceDelta = n) })
                                        },
                                    )
                                }
                            },
                            label = { Text(stringResource(R.string.mod_opt_price)) },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                            modifier = Modifier.width(110.dp),
                        )
                        RahalTextButton(onClick = {
                            onEdit { dd ->
                                dd.copy(
                                    modifiers = dd.modifiers.replaceAt(gi) { gg ->
                                        gg.copy(options = gg.options.filterIndexed { i, _ -> i != oi })
                                    },
                                )
                            }
                        }) { Text("×", color = Rahal.colors.danger) }
                    }
                }

                // **وخيارٌ يُضاف كمجموعةٍ تُضاف** — انظر أعلاه.
                RahalOutlineButton(onClick = {
                    onEdit { dd ->
                        dd.copy(
                            modifiers = dd.modifiers.replaceAt(gi) { gg ->
                                gg.copy(options = gg.options + ModifierOption(name = ""))
                            },
                        )
                    }
                }) { Text(stringResource(R.string.mod_add_option)) }
            }
        }
    }
}

/** **يستبدل عنصراً بموضعه** — ولا يمسّ ما سواه. */
private inline fun <T> List<T>.replaceAt(i: Int, block: (T) -> T): List<T> =
    mapIndexed { idx, v -> if (idx == i) block(v) else v }
