package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأسطرُ والرؤوسُ والفواصل** (نظامُ التصميم الموحَّد ٢٠٢٦-١١-١١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * - [SectionHeader] عنوانُ مجموعةٍ داخل الشاشة، وفعلٌ صغيرٌ في طرفه.
 * - [ListRow] سطرُ قائمة: أيقونة · عنوان · سطرٌ تحته · قيمةٌ أو سهمٌ في الطرف · ضغط.
 * - [RahalDivider] الفاصلُ الواحد.
 * - [SheetHeader] رأسُ اللوح الصاعد: مقبضٌ وعنوانٌ وزرُّ إغلاق.
 * - [BottomActionBar] شريطُ الفعل الأساسيّ الثابت أسفلَ الشاشة.
 */

/**
 * **عنوانُ مجموعةٍ داخل الشاشة** — «الأصناف»، «طرق الدفع»، «آخر الطلبات».
 *
 * ```
 * SectionHeader("آخر الطلبات")
 * SectionHeader("الأقسام", action = "عرض الكل") { openAll() }
 * ```
 *
 * **ومسافتُه فوقه أكبرُ من تحته** — فيُقرأ رأسَ ما بعده لا ذيلَ ما قبله.
 */
@Composable
fun SectionHeader(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String = "",
    /** **نصُّ فعلٍ صغيرٍ في الطرف** — «عرض الكل»؛ وفارغٌ لا فعل. */
    action: String = "",
    onAction: (() -> Unit)? = null,
) {
    Column(modifier.fillMaxWidth().padding(top = Rahal.space.lg, bottom = Rahal.space.sm)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                title,
                style = Rahal.type.sectionTitle,
                color = Rahal.colors.ink,
                modifier = Modifier.weight(1f),
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            if (action.isNotEmpty() && onAction != null) {
                RahalSmallButton(onClick = onAction) { Text(action) }
            }
        }
        if (subtitle.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.xxs))
            Text(subtitle, style = Rahal.type.caption, color = Rahal.colors.inkMuted)
        }
    }
}

/**
 * **سطرُ القائمة الموحَّد** — كلُّ قائمةٍ في التطبيقات الأربعة: الإعدادات، العناوين، السجلّ، الأعضاء.
 *
 * ```
 * ListRow("عناويني", icon = R.drawable.ic_pin, chevron = true) { openAddresses() }
 * ListRow("الرصيد", subtitle = "آخر حركة اليوم", trailingText = money(balance))
 * ListRow("أحمد", leading = { Avatar(url, "أحمد", 40) }, trailing = { StatusChip("يعمل", StatusTone.Success) })
 * ```
 *
 * @param icon أيقونةٌ في مربّعٍ ملوّن (`IconBox`) — أو مرّر `leading` لصورةٍ أو غيرها.
 * @param trailingText قيمةٌ نصّيّةٌ في الطرف (مبلغ، عدد).
 * @param trailing قطعةٌ في الطرف (رقاقة، مفتاح) — تُرسم بعد `trailingText`.
 * @param chevron سهمُ «افتح» في الطرف — لسطرٍ يفتح صفحة.
 */
@Composable
fun ListRow(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String = "",
    @androidx.annotation.DrawableRes icon: Int? = null,
    iconTint: Color = Rahal.colors.brand,
    leading: (@Composable () -> Unit)? = null,
    trailingText: String = "",
    trailingColor: Color = Rahal.colors.ink,
    trailing: (@Composable RowScope.() -> Unit)? = null,
    chevron: Boolean = false,
    titleColor: Color = Rahal.colors.ink,
    enabled: Boolean = true,
    onClick: (() -> Unit)? = null,
) {
    Row(
        modifier
            .fillMaxWidth()
            .heightIn(min = Rahal.size.touch)
            .clip(Rahal.shape.md)
            .then(if (onClick != null) Modifier.clickable(enabled = enabled, role = Role.Button, onClick = onClick) else Modifier)
            .padding(horizontal = Rahal.space.xs, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        when {
            leading != null -> leading()
            icon != null -> IconBox(icon = icon, tint = iconTint)
        }
        Column(Modifier.weight(1f)) {
            Text(title, style = Rahal.type.cardTitle, color = titleColor, maxLines = 2, overflow = TextOverflow.Ellipsis)
            if (subtitle.isNotEmpty()) {
                Spacer(Modifier.height(Rahal.space.xxs))
                Text(subtitle, style = Rahal.type.caption, color = Rahal.colors.inkMuted, maxLines = 2, overflow = TextOverflow.Ellipsis)
            }
        }
        if (trailingText.isNotEmpty()) {
            Text(trailingText, style = Rahal.type.bodyStrong, color = trailingColor, maxLines = 1)
        }
        trailing?.invoke(this)
        if (chevron) {
            Icon(
                painterResource(R.drawable.ic_chevron_end),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(Rahal.icon.sm),
            )
        }
    }
}

/**
 * **الفاصلُ الواحد** — خطٌّ شعريٌّ بلون الحدّ. **وبديلُ `HorizontalDivider`** في الشاشات.
 *
 * @param inset إزاحةٌ من البداية — لفاصلٍ يبدأ بعد أيقونة السطر.
 */
@Composable
fun RahalDivider(modifier: Modifier = Modifier, inset: Dp = Rahal.space.none) {
    Box(
        modifier
            .fillMaxWidth()
            .padding(start = inset)
            .height(Rahal.stroke.hair)
            .background(Rahal.colors.line),
    )
}

/**
 * **رأسُ اللوح الصاعد** — مقبضٌ صغيرٌ وعنوانٌ وزرُّ إغلاق. **كلُّ لوحٍ سفليٍّ يبدأ به.**
 *
 * ```
 * ModalBottomSheet(…, dragHandle = null) {
 *     SheetHeader("اختر العنوان", onClose = { open = false })
 *     …
 * }
 * ```
 */
@Composable
fun SheetHeader(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String = "",
    onClose: (() -> Unit)? = null,
    /** **مقبضُ السحب** فوق العنوان — يُطفأ إن رسمه اللوحُ بنفسه. */
    handle: Boolean = true,
) {
    Column(modifier.fillMaxWidth().padding(horizontal = Rahal.space.lg)) {
        if (handle) {
            Box(
                Modifier
                    .padding(top = Rahal.space.sm)
                    .align(Alignment.CenterHorizontally)
                    .width(Rahal.space.xxl)
                    .height(Rahal.space.xs)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.line),
            )
        }
        Row(
            Modifier.fillMaxWidth().padding(top = Rahal.space.sm, bottom = Rahal.space.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text(title, style = Rahal.type.sectionTitle, color = Rahal.colors.ink, maxLines = 2, overflow = TextOverflow.Ellipsis)
                if (subtitle.isNotEmpty()) {
                    Text(subtitle, style = Rahal.type.caption, color = Rahal.colors.inkMuted)
                }
            }
            if (onClose != null) {
                IconButton(onClick = onClose, modifier = Modifier.size(Rahal.size.touch)) {
                    Icon(
                        painterResource(R.drawable.ic_close),
                        contentDescription = stringResource(R.string.close),
                        tint = Rahal.colors.inkMuted,
                    )
                }
            }
        }
    }
}

/**
 * **شريطُ الفعل الأساسيّ الثابت** — «أكمل الطلب»، «احفظ»، «أرسل» — ملتصقٌ أسفلَ الشاشة فوق شريط النظام.
 *
 * ```
 * Column(Modifier.fillMaxSize()) {
 *     Screen(Modifier.weight(1f)) { … }
 *     BottomActionBar { RahalButton(onClick = save, modifier = Modifier.fillMaxWidth()) { Text("احفظ") } }
 * }
 * ```
 *
 * **وفوق الزرّ سطرٌ اختياريّ** (`summary`): الإجماليّ أو ما يلزم قبل الضغط.
 */
@Composable
fun BottomActionBar(
    modifier: Modifier = Modifier,
    summary: (@Composable RowScope.() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(
        modifier
            .fillMaxWidth()
            .shadow(Rahal.elevation.raised, clip = false)
            .background(Rahal.colors.canvas)
            .navigationBarsPadding()
            .padding(horizontal = Rahal.space.gutter, vertical = Rahal.space.md),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        if (summary != null) {
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                content = summary,
            )
        }
        content()
    }
}
