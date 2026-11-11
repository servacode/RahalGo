package com.rahalgo.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.isSpecified
import androidx.compose.ui.unit.sp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السعرُ بالعملتين — قطعةٌ واحدةٌ في كلّ مكانٍ يظهر فيه الدولارُ والليرةُ معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظاتُ المالك ٢٠٢٦-١١-١١، ٣ و٤ و٥: «$0.13 · 15 ل.س» تُقرأ رقماً واحداً — «لازم الزبون يعرف مين
 * الدولار ومين الليرة»، وقاعدتُه: **كلُّ مكانٍ فيه العملتان يُعرضان بوضوحٍ وبطريقةٍ واحدة.**)
 *
 * **الدولارُ عريضٌ بلون العلامة — هو السعرُ المعروض. والليرةُ أصغرُ باهتةٌ بعد «≈» — هي التقريبُ المدفوع.**
 * **و«$» لا ينقلب** (كلُّ جزءٍ معزولٌ يساراً)، **ولا «·» يتيم**: لا فاصلَ أصلاً، والليرةُ وحدةٌ واحدةٌ
 * تنزل كلَّها تحت الدولار إن ضاق السطر (`FlowRow`)، أو دائماً بـ`stacked`.
 *
 * **وبلا دولارٍ** (متجرٌ بالليرة) تُعرض الليرةُ وحدَها بحجم الدولار.
 *
 * **والويبُ مرآتُه** (`Price` في `web/apps/rahalgo/src/lib/store/usd.ts`) — ومن غيّر شكلاً غيّره في الآخر.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun DualPrice(
    usd: Double?,
    syp: Long,
    modifier: Modifier = Modifier,
    /** **سطران دائماً** — لخانةٍ ضيّقةٍ كبطاقة المنتج. وإلّا سطرٌ واحدٌ ينزل وحدَه إن ضاق. */
    stacked: Boolean = false,
    /** **نمطُ الدولار** — والليرةُ تُشتقّ منه أصغرَ. */
    style: TextStyle = Rahal.type.price,
    usdColor: Color = Rahal.colors.brand,
    sypColor: Color = Rahal.colors.inkMuted,
    horizontalAlignment: Alignment.Horizontal = Alignment.Start,
) {
    val main = style.copy(fontWeight = FontWeight.Bold, color = usdColor)
    if (usd == null) {
        Text(
            money(syp),
            style = main,
            modifier = modifier,
            maxLines = 1,
        )
        return
    }
    val small = style.copy(
        fontWeight = FontWeight.Normal,
        color = sypColor,
        fontSize = (if (style.fontSize.isSpecified) style.fontSize.value - 2f else 13f)
            .coerceAtLeast(Rahal.type.meta.fontSize.value).sp,
    )
    val usdPart = dualUsdPart(usd)
    val sypPart = dualSypPart(syp)
    // ══════════════════════════════════════════════════════════════════
    // **العملتان في سطرٍ واحدٍ دائماً** (ملاحظةُ المالك ٢٠٢٦-١١-١١: «سعر السوري والدولار لازم
    // يكون بنفس السطر، هيك منظرهم بشع بسطرين»)
    // ══════════════════════════════════════════════════════════════════
    //
    // **والليرةُ تصغر حتّى تسع** في خانةٍ ضيّقةٍ كبطاقة المنتج — ولا تنزل سطراً.
    // و`stacked` باقٍ في التوقيع لمن يناديه، ولا يغيّر شيئاً.
    androidx.compose.foundation.layout.Row(
        modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.xs),
    ) {
        Text(usdPart, style = main, maxLines = 1, softWrap = false)
        FitText(
            text = sypPart,
            style = small,
            maxLines = 1,
            minSize = 8.sp,
            modifier = Modifier.weight(1f, fill = false),
        )
    }
}

private const val LRI = "⁦"
private const val PDI = "⁩"

/** **الدولار معزولٌ يساراً** — فلا ينقلب «$» إلى آخر الرقم في سطرٍ عربيّ. */
fun dualUsdPart(usd: Double): String = LRI + dollars(usd) + PDI

/** **الليرةُ بعد «≈»** — معزولةٌ كذلك. */
fun dualSypPart(syp: Long): String = "≈ " + LRI + grouped(syp) + PDI + " ل.س"

/**
 * **السعرُ نصّاً واحداً** — لما لا يرسم قطعةً (إشعارٌ، قائمةُ نصوص، زرٌّ يقبل نصّاً وحدَه): الدولارُ ثمّ
 * «≈» والليرة، كلٌّ معزول. **وما أمكن فالقطعةُ `DualPrice` أوضح.**
 */
fun dualPriceText(usd: Double?, syp: Long): String =
    if (usd == null) money(syp) else dualUsdPart(usd) + " " + dualSypPart(syp)

/**
 * **مبلغٌ بالليرة بشكلٍ واحد** — `15,000 ل.س` بدور السعر. **وما فيه دولارٌ يمرّ بـ[DualPrice]**،
 * وما عملتُه متغيّرةٌ (`لدي توصيلة`) بـ`currency`.
 *
 * ```
 * MoneyText(15000)                                    // سعر
 * MoneyText(total, large = true)                      // إجماليّ
 * MoneyText(units, currency = "USD")                  // بالسنتات
 * MoneyText(debt, color = Rahal.colors.danger)        // دَين
 * ```
 */
@Composable
fun MoneyText(
    amount: Long,
    modifier: Modifier = Modifier,
    /** **العملة** — `"USD"` بالسنتات، وغيرُها ليرة (`amountText`). */
    currency: String? = null,
    large: Boolean = false,
    color: Color = Rahal.colors.ink,
    style: TextStyle = if (large) Rahal.type.priceLarge else Rahal.type.price,
) {
    Text(
        // **والنصُّ من `amountText` نفسِه** — الموضعُ الواحدُ الذي يعرف الليرةَ والسنتات.
        amountText(amount, currency),
        style = style,
        color = color,
        modifier = modifier,
        maxLines = 1,
    )
}
