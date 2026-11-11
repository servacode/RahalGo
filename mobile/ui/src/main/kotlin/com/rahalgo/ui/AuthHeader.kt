package com.rahalgo.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.ui.Alignment
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ترويسةُ الهويّة — شعارُ رحّال غو فوق شاشاتِ الدخول** (Batch 5، B)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **موضعٌ واحدٌ للشعار وحجمِه ومسافتِه** — فلا يتكرّر في كلّ شاشةٍ ولا
 * يفترقن. **يُصدَر داخلَ عمودٍ يُوسِّط أفقيّاً** (كما شاشاتُ الدخول)،
 * فالصورةُ في الوسط، وRTL مضمونٌ من السمة (`Theme`).
 *
 * **والشعارُ الرسميُّ `intro_logo`** — نفسُه على الدخول والافتتاح، **مولَّدٌ
 * ولا يُحرَّر بيد.**
 *
 * # ولماذا حجمٌ متغيّر
 *
 * **الدخولُ أقلُّ حقولاً** فيسعه ١٤٠dp — **وهو ما كان، فلا يتغيّر.**
 * **والإنشاءُ والاستعادةُ يحملان حقولاً كثيرةً** فشعارٌ كبيرٌ يزحمها —
 * **فيُصغَّر إلى نحو ١٠٠dp ومسافةٌ أقلُّ فوقَه**، فلا ازدحام.
 *
 * @param logoSize قطرُ الشعار — ١٤٠dp للدخول (كما كان)، وأصغرُ للمزدحم.
 * @param topSpace المسافةُ فوقَ الشعار.
 * @param bottomSpace المسافةُ تحتَه قبلَ العنوان.
 */
@Composable
fun AuthHeader(
    logoSize: Dp = 140.dp,
    topSpace: Dp = Rahal.space.xxl + Rahal.space.sm,
    bottomSpace: Dp = Rahal.space.xs + Rahal.space.xxs,
) {
    Spacer(Modifier.height(topSpace))
    Image(
        painter = painterResource(com.rahalgo.design.R.drawable.intro_logo),
        contentDescription = null,
        modifier = Modifier.size(logoSize),
    )
    Spacer(Modifier.height(bottomSpace))
}

/**
 * **أرضُ شاشات الدخول كلِّها** — عمودٌ يتوسّط ويمرَّر، على أرضٍ رماديّةٍ خفيفةٍ لتبرز عليها
 * بطاقةُ النموذج (`AuthCard`). (مراجعةُ التصميم الشاملة ٢٠٢٦-١١-٠٩.)
 *
 * **موضعٌ واحدٌ** للدخول والإنشاء والاستعادة وتبديل الكلمة: كانت كلُّ شاشةٍ تكتب عمودَها
 * بحشوةٍ ومحاذاةٍ تختلف قليلاً.
 */
@Composable
fun AuthPage(
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Rahal.colors.surface)
            // **والكيبوردُ لا يغطّي الزرّ** (رُئي على جوال المالك ٢٠٢٦-١٠-٠٩) — العمودُ يقصر فوقه.
            .imePadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Rahal.space.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
        content = content,
    )
}

/** **بطاقةُ النموذج** — الحقولُ والأزرارُ في بطاقةٍ بيضاءَ بظلّ، كبطاقات معالج الإعداد. */
@Composable
fun AuthCard(content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .cardSurface(Rahal.shape.lg, elevation = Rahal.elevation.raised)
            .padding(Rahal.space.lg),
        content = content,
    )
}
