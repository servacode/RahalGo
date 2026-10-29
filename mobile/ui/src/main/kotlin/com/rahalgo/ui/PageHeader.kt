package com.rahalgo.ui

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/** **لمسةُ سهم الرجوع** — ٤٨ كاملة. */
private val BackTouch = 48.dp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رأسُ الصفحة — سهمُ رجوعٍ وعنوان، واحدٌ لكلّ الصفحات المنبثقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩: شيءٌ من الصفحات له سهمٌ وشيءٌ لا — الإنذاراتُ
 *  والشكاوى والتقاريرُ والسجلّاتُ والمحفظةُ والإشعارات.) **فرأسٌ واحدٌ هنا**
 * بدل نسخةٍ في المتجر ونسخةٍ في العروض.
 *
 * **والسهمُ ينقلب وحدَه بالعربي** (`ic_arrow_back` بـ`autoMirrored`) فيشير يميناً.
 * و`onBack = null` عنوانٌ بلا سهم. **ولا مسافةَ تحته** — يضعها النداء.
 */
@Composable
fun PageHeader(
    title: String,
    onBack: (() -> Unit)?,
    modifier: Modifier = Modifier,
    /** **ما يقف في آخر الرأس** — زرٌّ صغيرٌ كـ«تعليم الكل كمقروء». */
    trailing: (@Composable RowScope.() -> Unit)? = null,
) {
    Row(modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        if (onBack != null) {
            IconButton(onClick = onBack, modifier = Modifier.size(BackTouch)) {
                Icon(
                    painterResource(R.drawable.ic_arrow_back),
                    contentDescription = stringResource(R.string.act_back),
                    tint = Rahal.colors.ink,
                )
            }
            Spacer(Modifier.width(Rahal.space.xs))
        }
        Text(
            title,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleLarge,
            color = Rahal.colors.ink,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        trailing?.invoke(this)
    }
}
