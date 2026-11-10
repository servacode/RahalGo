package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.ui.unit.sp
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **القائمة — ثلاثةُ خطوطٍ يعرفها الناسُ بلا اسم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «شو رأيك ٣ خطوطٍ عائمة تفتح القائمة ويكون
 *  مكتوبٌ فيها هذه الأقسام؟ أحسن من «المزيد» ومن اسمٍ معيّن — فالكلّ
 *  يعرف أنّ لها معنىً واضحا».)
 *
 * # وهو أصوبُ من تبويبٍ خامس
 *
 * **الشريطُ السفليُّ أربعةٌ الآن**، وخامسٌ بأسماءٍ عربيّةٍ يضغطها حتّى
 * تُقصّ. **والخطوطُ الثلاثةُ لا تحتاج اسماً أصلاً** — ومن رآها في أيّ
 * تطبيقٍ عرف ما تفتح.
 *
 * # والمجموعاتُ حدودٌ لا عناوين
 *
 * **كانت لكلّ مجموعةٍ اسمٌ فوقها** — «ما يخصّني · المنصّة ·
 * القانونيّة» — **فأخذت ثلاثةُ أسطرٍ مكانَ ثلاثةِ بنود.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «ما تلزم، ماخذة مساحة بدون فائدة».)
 *
 * **والحدُّ يفصل ما يفصله الاسم**: من رأى «شروط الاستخدام» تحت حدٍّ
 * عرف أنّها بابٌ آخرُ بلا أن يُقال له. **والاسمُ يشرح ما لا يحتاج
 * شرحا.**
 *
 * # ولماذا هنا لا في تطبيق
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «المنصّةُ مركزيٌّ أساسا · والقانونيّةُ
 *  مركزيٌّ أيضا · وزرُّ تسجيل خروجٍ أيضا».)
 *
 * **وأربعةُ تطبيقاتٍ لها القائمةُ نفسُها** — يختلف ما في «ما يخصّني»
 * وحدَه. **والمنصّةُ والقانونيّةُ والخروجُ واحدةٌ عند الجميع.**
 */
@Composable
fun Drawer(
    items: List<DrawerItem>,
    onPick: (DrawerItem) -> Unit,
    onLogout: () -> Unit,
    /**
     * **واسمُ زرّ القاع يتبع من يقرؤه.**
     *
     * **ومن لم يدخل بعدُ لا «يخرج»** — فيقول له الزرُّ «دخول»:
     * **زرُّ خروجٍ لضيفٍ يسأل «أخرج من ماذا؟».**
     */
    logoutLabel: Int? = null,
    /**
     * **مبدّلُ السمة** — أو فارغ.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٥: نُقل من الشريط العلويّ إلى هنا.)
     *
     * **والقائمةُ موضعُ الإعدادات** — حيث يبحث عنها من يريدها.
     */
    dark: Boolean? = null,
    onTheme: () -> Unit = {},
    /**
     * **صفٌّ فوق القائمة يضعه من ناداها** — أو فارغ.
     *
     * **و`:ui` لا تعرف مدناً ولا عناوين**: يضع تطبيقُ الزبون صفَّ
     * «تتسوّق في…» وحدَه، **وتبقى القائمةُ صالحةً للسائق والمندوبِ
     * اللذَين لا مدينةَ لهما.**
     *
     * **وفوقها لا تحتها**: من فتح القائمةَ رأى أوّلاً من أين يتسوّق،
     * **وسطرٌ في القاع يُقرأ بعد أن يكون قد اختار وجهتَه.**
     */
    header: (@Composable () -> Unit)? = null,
    /**
     * **ترويسةُ الحساب — واحدةٌ في التطبيقات الأربعة** (قرارُ المالك ٢٠٢٦-١١-١١: «الدرجُ يُشبه نفسَه في
     * كلّ تطبيق»): حرفُ الاسم، الاسم، الرقم معزولاً، ودورُ التطبيق.
     */
    profile: DrawerProfile? = null,
) {
    Column(
        Modifier
            .fillMaxSize()
            // **وصمّامُ أمانٍ لا أكثر**: القائمةُ مصمَّمةٌ لتسع بلا تمرير (انظر الأسفل)، وهذا لشاشةٍ قصيرةٍ جدّاً.
            .verticalScroll(rememberScrollState())
            .padding(vertical = 4.dp),
    ) {
        if (profile != null) {
            ProfileHeader(profile)
            HorizontalDivider()
            Spacer(Modifier.height(2.dp))
        }
        if (header != null) {
            header()
            Spacer(Modifier.height(2.dp))
            HorizontalDivider()
            Spacer(Modifier.height(2.dp))
        }

        // **بنودُ التطبيق أوّلاً، ثمّ صفحاتُ المنصّة** — بالترتيب نفسِه في كلّ تطبيق.
        val (platform, own) = items.partition { PlatformPages.has(it.key) }

        var lastGroup: Int? = null
        for (item in own) {
            // **والمجموعةُ تُعرف بحدّها** — **والاسمُ فوقها كان يشرح ما لا يحتاج شرحا.**
            if (item.group != lastGroup) {
                if (lastGroup != null) {
                    Spacer(Modifier.height(2.dp))
                    HorizontalDivider()
                    Spacer(Modifier.height(2.dp))
                }
                lastGroup = item.group
            }
            Line(item, onPick)
        }

        // **وصفحاتُ المنصّة والقانونيّةُ في عمودين** — خمسةُ بنودٍ في ثلاثة أسطر، فيبقى الدرجُ بلا تمرير
        // مع الترويسة. (والتمريرُ ممنوعٌ بقرار المالك.)
        if (platform.isNotEmpty()) {
            Spacer(Modifier.height(2.dp))
            HorizontalDivider()
            Spacer(Modifier.height(2.dp))
            platform.chunked(2).forEach { pair ->
                Row(Modifier.fillMaxWidth()) {
                    pair.forEach { item ->
                        DrawerRow(
                            icon = item.icon,
                            label = stringResource(item.label),
                            tint = Rahal.colors.brand,
                            onClick = { onPick(item) },
                            modifier = Modifier.weight(1f),
                            compact = true,
                        )
                    }
                    if (pair.size == 1) Spacer(Modifier.weight(1f))
                }
            }
        }

        // **والسمةُ ثمّ الخروج — في سطرٍ واحدٍ بعمودين في القاع** (بعيداً عن طريق الإبهام).
        // **والأيقونةُ تُظهر الوجهةَ لا الحال** — هلالٌ يقول «انتقل إلى الغامقة»، وشمسٌ «عُد إلى الفاتحة».
        Spacer(Modifier.height(2.dp))
        HorizontalDivider()
        Spacer(Modifier.height(2.dp))
        Row(Modifier.fillMaxWidth()) {
            if (dark != null) {
                DrawerRow(
                    icon = if (dark) R.drawable.ic_theme_light else R.drawable.ic_theme,
                    label = stringResource(if (dark) R.string.top_theme_light else R.string.top_theme),
                    tint = Rahal.colors.inkMuted,
                    onClick = onTheme,
                    modifier = Modifier.weight(1f),
                    compact = true,
                )
            }
            DrawerRow(
                icon = if (logoutLabel == null) R.drawable.ic_logout else R.drawable.ic_login,
                label = stringResource(logoutLabel ?: R.string.login_logout),
                tint = if (logoutLabel == null) Rahal.colors.danger else Rahal.colors.brand,
                labelColor = if (logoutLabel == null) Rahal.colors.danger else Rahal.colors.brand,
                onClick = onLogout,
                modifier = Modifier.weight(1f),
                compact = true,
            )
        }
        Spacer(Modifier.height(4.dp))
    }
}

/** **ما تحتاجه الترويسةُ** — يملؤه كلُّ تطبيقٍ من حسابه. */
data class DrawerProfile(
    val name: String,
    val phone: String,
    val avatarUrl: String? = null,
    /** **دورُ التطبيق** — «زبون»، «سائق»، «متجر»، «مندوب». */
    val role: Int,
)

@Composable
private fun ProfileHeader(p: DrawerProfile) {
    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(url = p.avatarUrl, name = p.name.ifBlank { "؟" }, size = 44)
        Spacer(Modifier.size(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            if (p.name.isNotBlank()) {
                Text(
                    p.name,
                    fontWeight = androidx.compose.ui.text.font.FontWeight.Bold,
                    color = Rahal.colors.ink,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
            }
            if (p.phone.isNotBlank()) {
                Text(
                    phoneText(p.phone),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                )
            }
            Text(
                stringResource(p.role),
                color = Rahal.colors.brand,
                style = MaterialTheme.typography.labelSmall,
                maxLines = 1,
            )
        }
    }
}

@Composable
private fun Line(item: DrawerItem, onPick: (DrawerItem) -> Unit) {
    DrawerRow(
        icon = item.icon,
        label = stringResource(item.label),
        tint = Rahal.colors.brand,
        onClick = { onPick(item) },
    )
}

/**
 * **سطرُ القائمة الواحد** — أيقونةٌ في مربّعٍ ملوّنٍ واسمٌ، بزاويةٍ مدوّرةٍ عند اللمس.
 * **والسطرُ نفسُه لكلّ البنود** فلا تختلف الأحجام؛ و`compact` لنصف العرض.
 */
@Composable
private fun DrawerRow(
    icon: Int,
    label: String,
    tint: androidx.compose.ui.graphics.Color,
    onClick: () -> Unit,
    labelColor: androidx.compose.ui.graphics.Color = Rahal.colors.ink,
    modifier: Modifier = Modifier,
    /** **نصفُ العرض** — أيقونةٌ أصغرُ ونصٌّ يصغر حتّى يسع ولا يُقصّ. */
    compact: Boolean = false,
) {
    Row(
        modifier
            .fillMaxWidth()
            .padding(horizontal = Rahal.space.sm)
            .clip(Rahal.shape.md)
            .clickable(onClick = onClick)
            .heightIn(min = if (compact) 40.dp else 44.dp)
            .padding(horizontal = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconTile(icon = icon, tint = tint, size = if (compact) 24.dp else 30.dp)
        Spacer(Modifier.size(if (compact) Rahal.space.sm else Rahal.space.md))
        if (compact) {
            val style = MaterialTheme.typography.bodyMedium.copy(color = labelColor)
            androidx.compose.foundation.text.BasicText(
                text = label,
                style = style,
                maxLines = 1,
                autoSize = androidx.compose.foundation.text.TextAutoSize.StepBased(
                    minFontSize = 10.sp,
                    maxFontSize = style.fontSize,
                ),
            )
        } else {
            Text(
                text = label,
                color = labelColor,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
            )
        }
    }
}

/**
 * **عرضُ الدرج — واحدٌ في التطبيقات الأربعة**: ٢٨٠dp أو ٨٠٪ من الشاشة، الأصغرُ. (كان ٢٠٠ في السائق
 * والمندوب و٢٨٠ في المتجر.) **ويُغلَّف الدرجُ هنا فلا يختار تطبيقٌ عرضَه.**
 */
@Composable
fun AppDrawerSheet(content: @Composable androidx.compose.foundation.layout.ColumnScope.() -> Unit) {
    val w = minOf(280, (androidx.compose.ui.platform.LocalConfiguration.current.screenWidthDp * 0.8f).toInt())
    androidx.compose.material3.ModalDrawerSheet(Modifier.width(w.dp), content = content)
}

data class DrawerItem(
    val key: String,
    /** **اسمُ المجموعة** — والبنودُ تُرتَّب بها لا بترتيب كتابتها. */
    val group: Int,
    val label: Int,
    val icon: Int,
)

/** **ترويسةُ الدرج من حساب الجلسة** — واحدةٌ للتطبيقات الأربعة. */
@Composable
fun drawerProfile(vm: AccountViewModel, role: Int): DrawerProfile {
    val me = vm.state.me
    return DrawerProfile(
        name = me?.fullName.orEmpty(),
        phone = me?.phone.orEmpty(),
        avatarUrl = me?.avatarThumbUrl,
        role = role,
    )
}
