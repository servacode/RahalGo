package com.rahalgo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
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
) {
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(vertical = 8.dp),
    ) {
        var lastGroup: Int? = null
        for (item in items) {
            // **والمجموعةُ تُعرف بحدّها** — **والاسمُ فوقها كان يشرح ما
            // لا يحتاج شرحا.**
            if (item.group != lastGroup) {
                if (lastGroup != null) {
                    Spacer(Modifier.height(6.dp))
                    HorizontalDivider()
                    Spacer(Modifier.height(6.dp))
                }
                lastGroup = item.group
            }
            Line(item, onPick)
        }

        // ══════════════════════════════════════════════════════════════
        // **ومبدّلُ السمة فوق الخروج**
        // ══════════════════════════════════════════════════════════════
        //
        // **والأيقونةُ تُظهر الوجهةَ لا الحال** — هلالٌ يقول «انتقل إلى
        // الغامقة»، وشمسٌ تقول «عُد إلى الفاتحة». **ومن رسم حالَه
        // الحاليَّ جعل اللمسةَ تفاجئ.**
        //
        // **ولا يُخلط بالأقسام** — بحدٍّ فوقه: **هو إعدادٌ لا موضعٌ
        // يُنتقل إليه.**
        if (dark != null) {
            Spacer(Modifier.height(10.dp))
            HorizontalDivider()
            Row(
                Modifier
                    .fillMaxWidth()
                    .clickable(onClick = onTheme)
                    .padding(horizontal = 14.dp, vertical = 14.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(
                        if (dark) R.drawable.ic_theme_light else R.drawable.ic_theme,
                    ),
                    contentDescription = null,
                    tint = Rahal.colors.inkMuted,
                    modifier = Modifier.size(20.dp),
                )
                Spacer(Modifier.size(12.dp))
                Text(
                    stringResource(
                        if (dark) R.string.top_theme_light else R.string.top_theme,
                    ),
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والخروجُ في القاع — بعيداً عن طريق الإبهام**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «برأيك زرُّ تسجيل الخروج وين مكانه
        //  الصحيح؟ أيضاً بالقائمة الجانبيّة بالأسفل صحيح — هذا أفضل
        //  مكانٍ له».)
        //
        // **وثلاثةُ أسبابٍ تجعله صحيحا:**
        //
        // **١ · القاعُ آخرُ ما يبلغه الإبهام** — وفعلٌ يُخرجه من حسابه
        // لا يُوضع في طريق مرور. **ومن خرج سهواً يعود بكلمة مرورٍ قد
        // لا يحفظها.**
        //
        // **٢ · وهو حيث يتوقّعه** — كلُّ تطبيقٍ يضعه هناك، **فيُوجَد
        // بلا بحث.**
        //
        // **٣ · وموضعٌ واحدٌ لا موضعان** — كان في لوحة العمل، **ورُفع
        // منها**: فعلٌ في مكانين يُنسى أحدُهما فيبقى قديماً حين يتبدّل.
        //
        // **وأحمرُ بحدٍّ فوقه** — لا يُخلط بما قبله من أسماء أقسام.
        Spacer(Modifier.height(10.dp))
        HorizontalDivider()
        Row(
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onLogout)
                .padding(horizontal = 14.dp, vertical = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(
                    if (logoutLabel == null) R.drawable.ic_logout else R.drawable.ic_login,
                ),
                contentDescription = null,
                tint = if (logoutLabel == null) Rahal.colors.danger else Rahal.colors.brand,
                modifier = Modifier.size(20.dp),
            )
            Spacer(Modifier.size(12.dp))
            Text(
                stringResource(logoutLabel ?: R.string.login_logout),
                color = if (logoutLabel == null) Rahal.colors.danger else Rahal.colors.brand,
                style = MaterialTheme.typography.bodyLarge,
            )
        }
        Spacer(Modifier.height(16.dp))
    }
}

@Composable
private fun Line(item: DrawerItem, onPick: (DrawerItem) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable { onPick(item) }
            .padding(horizontal = 14.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(item.icon),
            contentDescription = null,
            tint = Rahal.colors.inkMuted,
            modifier = Modifier.size(20.dp),
        )
        Spacer(Modifier.size(10.dp))
        Text(
            text = stringResource(item.label),
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
        )
    }
}

/**
 * **بندٌ في القائمة** — التطبيقُ يصفه والقائمةُ ترسمه.
 *
 * **و`key` نصٌّ لا تعداد**: أقسامُ كلّ تطبيقٍ غيرُ أقسام الآخر، **ولو
 * حملت الوحدةُ تعدادَ السائق لَحملها الزبونُ معه** — فيرى «صندوقي» في
 * شيفرته وهو لا يملك صندوقاً. **وهو نفسُه مفتاحُ `Overlay.Menu`.**
 */
data class DrawerItem(
    val key: String,
    /** **اسمُ المجموعة** — والبنودُ تُرتَّب بها لا بترتيب كتابتها. */
    val group: Int,
    val label: Int,
    val icon: Int,
)
