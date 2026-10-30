package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.OrderLine

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الفاتورةُ سطرٌ لكلّ عنصر — بشكلٍ واحدٍ في كلّ التطبيقات** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «شاورما دبل × 3 ........ 90 ل.س» · «بندورة × 1 كيلو ........ 600 ل.س» ·
 * «بوط أسود × 1 نمرة 40 ........ 9,100 ل.س»: **النصُّ في البداية ومجموعُ السطر
 * في النهاية** — لا سطرَ خياراتٍ ولا «للواحد» ولا عمودَ عدد. **ولا يُكتب اسمُ
 * مجموعة الخيار** («وزن ثقيل»، «الحجم»). **والمشروباتُ (`drink_for`) كلُّها تحت الأصناف،
 * والمشروبُ نفسُه مع أكثرَ من صنفٍ سطرٌ واحدٌ بعدده ومجموعه** (`invoiceOrder`).
 */
data class InvoiceLine(
    val id: String = "",
    val drinkFor: String? = null,
    val text: String,
    /** **مجموعُ السطر** = سعرُ الوحدة × العدد — و`null`: لا يرى صاحبُ الشاشة السعر. */
    val amount: Long? = null,
    /** **ملاحظةُ الزبون على الصنف** («بلا بصل») — تُقرأ قبل الصنع. */
    val note: String = "",
    /**
     * **اسمُ البند وعددُه وخياراتُه** — ليُجمع المشروبُ نفسُه من أكثرَ من صنفٍ ويُعاد
     * نصُّه بالعدد المجموع. **وفراغُ الاسم**: يُجمع بنصّه المطابق وحدَه.
     */
    val name: String = "",
    val qty: Int = 0,
    val options: List<Pair<String, String>> = emptyList(),
)

private val weightWord = Regex("كيلو|كغ|غرام|(^|\\s|[0-9٠-٩])غ$")
private val numberOnly = Regex("^[0-9٠-٩]+([.,٫][0-9٠-٩]+)?$")

/**
 * **نصُّ سطر الفاتورة — القاعدةُ في مكانٍ واحد.**
 *
 * - **الوزنُ** (مجموعةٌ فيها «وزن» أو قيمةٌ بالكيلو/الغرام) بعد العدد كما هو: «× 1 كيلو».
 * - **الرقمُ المجرّد** (نمرةُ حذاء) بعد العدد: «× 1 نمرة 40» — ولمجموعةٍ رقميّةٍ غيرِها
 *   اسمُها مختصراً بلا «ال».
 * - **وكلُّ قيمةٍ غيرِها** (دبل، أسود، L) تُلحق بالاسم.
 *
 * `options` أزواجُ (المجموعة، القيمة)، والمجموعةُ قد تكون فارغةً لمن لا يعرفها.
 */
fun invoiceLineText(name: String, qty: Int, options: List<Pair<String, String>>): String {
    val plain = ArrayList<String>()
    val after = ArrayList<String>()
    options.forEach { (rawGroup, rawValue) ->
        val group = rawGroup.trim()
        val value = rawValue.trim()
        if (value.isEmpty()) return@forEach
        when {
            group.contains("وزن") || weightWord.containsMatchIn(value) -> after += value
            numberOnly.matches(value) -> {
                val short = group.removePrefix("ال").trim()
                after += when {
                    short.startsWith("نمرة") -> "نمرة $value"
                    short.isNotEmpty() -> "$short $value"
                    else -> value
                }
            }
            else -> plain += value
        }
    }
    val head = if (plain.isEmpty()) name.trim() else name.trim() + " " + plain.joinToString("، ")
    val tail = if (after.isEmpty()) "" else " " + after.joinToString(" ")
    return "$head × $qty$tail"
}

/**
 * **الأصنافُ كلُّها أوّلاً ثمّ المشروباتُ كلُّها تحتها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) —
 * «شاورما عادي» ثمّ «شاورما دبل» ثمّ «عيران × 2». **والمشروبُ نفسُه مع أكثرَ من صنفٍ
 * سطرٌ واحد** بعدده ومجموعه. وترتيبُ المشروبات بترتيب أصنافها، ومشروبٌ لا صنفَ له
 * في القائمة يُلحق آخراً ولا يضيع. **وواحدةٌ للزبون والمتجر والسائق.**
 */
fun invoiceOrder(lines: List<InvoiceLine>): List<InvoiceLine> {
    val items = lines.filter { it.drinkFor == null }
    val byItem = lines.filter { it.drinkFor != null }.groupBy { it.drinkFor }
    val drinks = ArrayList<InvoiceLine>()
    items.forEach { item -> if (item.id.isNotEmpty()) byItem[item.id]?.let { drinks += it } }
    lines.filter { it.drinkFor != null && it !in drinks }.forEach { drinks += it }

    val merged = LinkedHashMap<String, InvoiceLine>()
    drinks.forEach { d ->
        val key = if (d.name.isNotBlank()) {
            "n:" + d.name.trim() + "|" + d.options.joinToString(",") { it.first.trim() + "=" + it.second.trim() }
        } else {
            "t:" + d.text
        }
        val was = merged[key]
        merged[key] = if (was == null) {
            d
        } else {
            val qty = was.qty + d.qty
            was.copy(
                qty = qty,
                text = if (was.name.isNotBlank()) invoiceLineText(was.name, qty, was.options) else was.text,
                amount = if (was.amount != null && d.amount != null) was.amount + d.amount else null,
                note = listOf(was.note, d.note).filter { it.isNotBlank() }.distinct().joinToString(" · "),
            )
        }
    }
    return items + merged.values
}

/** **بندُ طلبٍ من المحرّك سطراً في الفاتورة** — المتجرُ يرى سعرَ البيع كما اليوم. */
fun OrderLine.toInvoiceLine(): InvoiceLine = InvoiceLine(
    id = id,
    drinkFor = drinkFor,
    text = invoiceLineText(name, qty, options.map { it.group to it.name }),
    amount = unitPrice * qty,
    note = note,
    name = name,
    qty = qty,
    options = options.map { it.group to it.name },
)

/** **أسطرُ الأصناف** — كلٌّ في سطرٍ: النصُّ في البداية والمبلغُ في النهاية. */
@Composable
fun InvoiceLines(lines: List<InvoiceLine>, modifier: Modifier = Modifier) {
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        invoiceOrder(lines).forEach { l ->
            Column(Modifier.fillMaxWidth()) {
                InvoiceRow(l.text, l.amount?.let { money(it) }, bold = true)
                if (l.note.isNotBlank()) {
                    Text(
                        l.note,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
    }
}

/**
 * **سطرٌ واحد** — للأصناف وللمجموع والتوصيل والخصم والإجماليّ تحت الخطّ.
 * `strong` للإجماليّ، و`amount` فارغٌ: سطرٌ بلا مبلغ.
 */
@Composable
fun InvoiceRow(
    label: String,
    amount: String?,
    strong: Boolean = false,
    bold: Boolean = false,
    amountColor: Color = Color.Unspecified,
) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            label,
            color = if (strong || bold) Rahal.colors.ink else Rahal.colors.inkMuted,
            fontWeight = if (strong || bold) FontWeight.Bold else FontWeight.Normal,
            style = if (strong) MaterialTheme.typography.bodyLarge else MaterialTheme.typography.bodySmall,
            modifier = Modifier.weight(1f),
        )
        if (amount != null) {
            Spacer(Modifier.width(12.dp))
            Text(
                amount,
                color = if (amountColor != Color.Unspecified) amountColor else if (strong) Rahal.colors.brand else Rahal.colors.ink,
                fontWeight = if (strong) FontWeight.Bold else FontWeight.Normal,
                style = (if (strong) MaterialTheme.typography.bodyLarge else MaterialTheme.typography.bodySmall)
                    .copy(fontFeatureSettings = "tnum"),
            )
        }
    }
}
