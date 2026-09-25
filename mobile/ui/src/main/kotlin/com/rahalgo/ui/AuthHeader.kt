package com.rahalgo.ui

import androidx.compose.foundation.Image
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
    topSpace: Dp = 40.dp,
    bottomSpace: Dp = 6.dp,
) {
    Spacer(Modifier.height(topSpace))
    Image(
        painter = painterResource(com.rahalgo.design.R.drawable.intro_logo),
        contentDescription = null,
        modifier = Modifier.size(logoSize),
    )
    Spacer(Modifier.height(bottomSpace))
}
