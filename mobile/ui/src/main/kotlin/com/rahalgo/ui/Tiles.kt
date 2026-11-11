package com.rahalgo.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المربّعاتُ والأزرارُ المدوَّرة** (نظامُ التصميم الموحَّد ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * - [IconBox] أيقونةٌ في مربّعٍ مدوَّرٍ بلونها — بدايةُ سطرٍ أو رأسُ بطاقة.
 * - [IconTile] **مربّعُ وظيفة** — أيقونةٌ واسمٌ وقيمةٌ ويُضغط: الدوام، العمولة، الأقسام.
 * - [StatTile] **مربّعُ رقم** — الاسمُ فوق والرقمُ تحت: طلباتُ اليوم، المبيعات.
 * - [IconAction] **زرُّ أيقونةٍ مدوَّر** — اتّصال، واتساب، كاميرا — باسمٍ تحته أو بلا اسم.
 * - [AvatarWithEdit] **شعارٌ وعليه كاميرا** — يُضغط فيُغيَّر.
 * - [ActionRow] **صفٌّ بخاناتٍ متساوية** — ٢ أو ٣ أو ٤ من القطع أعلاه، بعرضٍ واحدٍ وارتفاعٍ واحد.
 */

/** **أيقونةٌ في مربّعٍ مدوَّرٍ بلونها** — لغةُ الأيقونات في «متجري» والقائمة والإشعارات. */
@Composable
fun IconBox(
    icon: Int,
    tint: Color,
    modifier: Modifier = Modifier,
    size: Dp = Rahal.size.iconBox,
) {
    Box(
        modifier
            .size(size)
            .clip(Rahal.shape.md)
            .background(tint.copy(alpha = 0.12f)),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(size * 0.55f),
        )
    }
}

/**
 * **مربّعُ وظيفة** — أيقونةٌ فوق اسمِ الوظيفة وقيمتِها الحاليّة، ويُضغط كلُّه.
 *
 * ```
 * ActionRow {
 *     IconTile(R.drawable.ic_time, "الدوام", "٩ص – ١١م") { openHours() }
 *     IconTile(R.drawable.ic_percent, "العمولة", "١٠٪") { editCommission() }
 *     IconTile(R.drawable.ic_grid, "الأقسام") { openSections() }
 * }
 * ```
 *
 * **ويملأ خانتَه عرضاً وارتفاعاً** — فتتساوى المربّعاتُ في صفّها (`ActionRow`).
 *
 * @param value القيمةُ الحاليّة — وفارغةٌ لا سطرَ لها.
 * @param valueColor لونُ القيمة — الأحمرُ لِما لم يُضبط بعد.
 * @param onClick وبلا قيمةٍ لا يُضغط (مربّعُ عرضٍ فقط).
 */
@Composable
fun IconTile(
    icon: Int,
    label: String,
    value: String = "",
    modifier: Modifier = Modifier,
    valueColor: Color = Rahal.colors.inkMuted,
    tint: Color = Rahal.colors.brand,
    enabled: Boolean = true,
    onClick: (() -> Unit)? = null,
) {
    Column(
        modifier
            .fillMaxWidth()
            .fillMaxHeight()
            .heightIn(min = Rahal.size.tileMin)
            .clip(Rahal.shape.md)
            .background(tint.copy(alpha = 0.07f))
            .then(
                if (onClick != null) Modifier.clickable(enabled = enabled, role = Role.Button, onClick = onClick) else Modifier,
            )
            .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.md),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(Rahal.space.xs, Alignment.CenterVertically),
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(Rahal.icon.md))
        // **يصغر ولا تنكسر كلمتُه** (الفحصُ البصريّ ٢٠٢٦-١١-١١: «العرو/ض» على ٣٦٠ بخطٍّ ١٫٣).
        FitText(
            label,
            color = Rahal.colors.ink,
            style = Rahal.type.label.copy(fontWeight = Rahal.type.bodyStrong.fontWeight),
            textAlign = TextAlign.Center,
            maxLines = 2,
        )
        if (value.isNotEmpty()) {
            Text(
                value,
                color = valueColor,
                style = Rahal.type.caption,
                textAlign = TextAlign.Center,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/**
 * **مربّعُ رقم** — الاسمُ فوقه والرقمُ تحته (طلبُ المالك ٢٠٢٦-٠٨-٣١).
 *
 * **والاسمُ سطران دائماً** فتتساوى المربّعاتُ، **والرقمُ بحجمٍ واحدٍ مهما طال.**
 * يوضع في `StatRow` (بـ`Modifier.weight(1f)`) أو في `ActionRow`.
 *
 * ```
 * StatRow {
 *     StatTile("طلبات اليوم", "12", Modifier.weight(1f))
 *     StatTile("المبيعات", money(150000), Modifier.weight(1f), valueColor = Rahal.colors.success)
 * }
 * ```
 */
@Composable
fun StatTile(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    valueColor: Color = Rahal.colors.ink,
    @androidx.annotation.DrawableRes icon: Int? = null,
    onClick: (() -> Unit)? = null,
    /**
     * **مربّعٌ مرصوص** (قرارُ المالك ٢٠٢٦-١١-١١: «مربّعات اليوم طويلةٌ وفيها فراغ») — الأيقونةُ والاسمُ
     * في سطرٍ واحد، والرقمُ تحتهما، وحشواتٌ أصغر. `false` الشكلُ القديم لمن لا يزال يعتمد عليه.
     */
    compact: Boolean = false,
) {
    if (compact) {
        Column(
            modifier
                .fillMaxWidth()
                .fillMaxHeight()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.brand.copy(alpha = 0.07f))
                .then(if (onClick != null) Modifier.clickable(role = Role.Button, onClick = onClick) else Modifier)
                .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.sm),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (icon != null) {
                    Icon(painterResource(icon), contentDescription = null, tint = Rahal.colors.brand, modifier = Modifier.size(Rahal.icon.xs))
                    androidx.compose.foundation.layout.Spacer(Modifier.size(Rahal.space.xs))
                }
                // **يصغر ولا تنكسر كلمتُه** — سطرٌ واحد.
                FitText(
                    label,
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                    textAlign = TextAlign.Center,
                    maxLines = 1,
                )
            }
            Text(
                text = value,
                color = valueColor,
                style = Rahal.type.sectionTitle,
                textAlign = TextAlign.Center,
                maxLines = 1,
            )
        }
        return
    }
    Column(
        modifier
            // **ويملأ خانتَه عرضاً** كـ`IconTile` (الفحصُ البصريّ ٢٠٢٦-١١-١١): كان في `ActionRow`
            // بعرض نصّه فتطفو مربّعاتٌ ضيّقةٌ بينها فراغ.
            .fillMaxWidth()
            .fillMaxHeight()
            // **مربّعٌ بصبغة العلامة** لا بطاقةٌ بظلّ — فيصلح داخل بطاقةٍ وعلى الأرض الرماديّة معاً.
            .clip(Rahal.shape.md)
            .background(Rahal.colors.brand.copy(alpha = 0.07f))
            .then(if (onClick != null) Modifier.clickable(role = Role.Button, onClick = onClick) else Modifier)
            .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.md),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        if (icon != null) {
            Icon(painterResource(icon), contentDescription = null, tint = Rahal.colors.brand, modifier = Modifier.size(Rahal.icon.sm))
            androidx.compose.foundation.layout.Spacer(Modifier.size(Rahal.space.xs))
        }
        Text(
            text = label,
            color = Rahal.colors.inkMuted,
            style = Rahal.type.caption,
            textAlign = TextAlign.Center,
            minLines = 2,
            maxLines = 2,
        )
        androidx.compose.foundation.layout.Spacer(Modifier.size(Rahal.space.xs))
        Text(
            text = value,
            color = valueColor,
            style = Rahal.type.sectionTitle,
            textAlign = TextAlign.Center,
            maxLines = 1,
        )
    }
}

/**
 * **زرُّ أيقونةٍ مدوَّر** — اتّصال، واتساب، كاميرا، مشاركة. **٤٨ لمساً**، وأرضٌ خفيفةٌ وحدٌّ من لونه.
 *
 * ```
 * IconAction(R.drawable.ic_phone, "اتّصال") { dial(phone) }
 * IconAction(R.drawable.ic_whatsapp, "واتساب", tint = Rahal.colors.success, label = "واتساب") { … }
 * ```
 *
 * @param description ما يقوله قارئُ الشاشة — **إجباريّ**: زرٌّ بلا اسمٍ لا يُعرف.
 * @param label اسمٌ مرئيٌّ تحت الدائرة — وفارغٌ لا اسم.
 */
@Composable
fun IconAction(
    icon: Int,
    description: String,
    modifier: Modifier = Modifier,
    tint: Color = Rahal.colors.brand,
    label: String = "",
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    val c = if (enabled) tint else tint.copy(alpha = 0.4f)
    Column(
        modifier
            .clip(Rahal.shape.md)
            .clickable(enabled = enabled, role = Role.Button, onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(Rahal.size.action)
                .clip(Rahal.shape.pill)
                .background(c.copy(alpha = 0.12f))
                .border(BorderStroke(Rahal.stroke.hair, c.copy(alpha = 0.35f)), Rahal.shape.pill),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = description, tint = c, modifier = Modifier.size(Rahal.icon.md))
        }
        if (label.isNotEmpty()) {
            // **سطران يصغران ولا يُقصّان** (الفحصُ البصريّ: «أهدافي والمكا…» على ٣٦٠) — والصفُّ
            // يصفّ الدوائرَ من أعلاها (`ActionRow`) فلا يُزيحها اسمٌ من سطرين.
            FitText(
                label,
                color = c,
                style = Rahal.type.label,
                textAlign = TextAlign.Center,
                maxLines = 2,
                modifier = Modifier.padding(top = Rahal.space.xs),
            )
        }
    }
}

/**
 * **الشعارُ وعليه كاميرا صغيرة** — يُضغط فيفتح «كاميرا أم معرض» (يبنيه المنادي في `onPick`).
 *
 * ```
 * AvatarWithEdit(url = logo, name = store.name, description = "تغيير الشعار", onPick = { pickLogo() })
 * ```
 *
 * @param enabled وبلا تفعيلٍ تُعرض الصورةُ والكاميرا ولا تُضغطان (أثناء الرفع).
 * @param size قطرُ الصورة — `Rahal.size.avatarLg` افتراضاً.
 */
@Composable
fun AvatarWithEdit(
    url: String?,
    name: String,
    description: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    size: Dp = Rahal.size.avatarLg,
    onPick: () -> Unit,
) {
    val badge = size * 0.4f
    Box(modifier.size(size)) {
        RemoteImage(
            url = url,
            name = name,
            modifier = Modifier
                .size(size * 0.94f)
                .clip(Rahal.shape.md)
                .clickable(enabled = enabled, onClick = onPick),
        )
        Box(
            Modifier
                .align(Alignment.BottomEnd)
                .size(badge)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.brand)
                .border(BorderStroke(Rahal.stroke.focus, Rahal.colors.canvas), Rahal.shape.pill)
                .clickable(enabled = enabled, role = Role.Button, onClick = onPick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(R.drawable.ic_photo),
                contentDescription = description,
                tint = Rahal.colors.onBrand,
                modifier = Modifier.size(badge * 0.55f),
            )
        }
    }
}

/**
 * **صفٌّ بخاناتٍ متساوية** — كلُّ قطعةٍ تأخذ خانةً بعرضٍ واحد، والصفُّ بارتفاع أطولِها.
 *
 * **ولا `weight` يُكتب في النداء**: الصفُّ يقسم العرضَ بنفسه، فلا تنسى قطعةٌ وزنَها فتصغر.
 * المربّعاتُ (`IconTile`) تملأ خانتَها، والأزرارُ المدوَّرةُ (`IconAction`) تتوسّطها.
 *
 * ```
 * ActionRow { IconAction(…){}; IconAction(…){}; IconAction(…){} }
 * ```
 */
@Composable
fun ActionRow(
    modifier: Modifier = Modifier,
    gap: Dp = Rahal.space.sm,
    content: @Composable () -> Unit,
) {
    Layout(content = content, modifier = modifier.fillMaxWidth()) { measurables, constraints ->
        val n = measurables.size
        if (n == 0) return@Layout layout(constraints.minWidth, 0) {}
        val gapPx = gap.roundToPx()
        val width = if (constraints.hasBoundedWidth) constraints.maxWidth else constraints.minWidth
        val cell = ((width - gapPx * (n - 1)) / n).coerceAtLeast(0)
        // **ارتفاعُ الصفّ من أطول قطعة** — فالمربّعُ الذي يملأ ارتفاعَه يتساوى مع جيرانه.
        val rowH = measurables.maxOf { it.maxIntrinsicHeight(cell) }
        val placeables = measurables.map {
            it.measure(Constraints(minWidth = 0, maxWidth = cell, minHeight = 0, maxHeight = rowH))
        }
        val h = placeables.maxOf { it.height }
        layout(width, h) {
            placeables.forEachIndexed { i, p ->
                val x = i * (cell + gapPx) + (cell - p.width) / 2
                // **والاتّجاهُ من اليمين** — `placeRelative` يقلب الترتيبَ في العربيّة.
                // **ومن أعلى الخانة** — الدوائرُ تتحاذى ولو كان اسمُ إحداها سطرين.
                p.placeRelative(x, 0)
            }
        }
    }
}
