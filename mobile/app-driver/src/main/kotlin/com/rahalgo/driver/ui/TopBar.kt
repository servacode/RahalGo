package com.rahalgo.driver.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشريط العلويّ — محفظتك وإشعاراتك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرار المالك ٢٠٢٦-٠٨-١٢: «بالأعلى يجب أن يكون هناك أيقونة إشعارات على
 *  اليمين ومحفظتك على اليسار، تكون ظاهرة».)
 *
 * # ولماذا ظاهران دائما
 *
 * **رصيدك سؤال يتكرّر في اليوم عشرات المرّات** — ومن خبّأه في شاشة
 * ثالثة جعل السائق يبحث عنه، **أو يتّصل بالمكتب ليسأل.**
 *
 * **والإشعار الذي لا تُرى شارته لم يصل**: من أغلق التطبيق وفتحه لا يعرف
 * أنّ خبرا ينتظره.
 *
 * # والشارة على الجرس لا بجانبه
 *
 * **تطفو فوقه في زاويته** — وهو ما تفعله كلّ التطبيقات، **ومن وضعها
 * بجانبه** جعلها تُقرأ رقما آخر لا عدّ إشعارات.
 */
@Composable
fun TopBar(
    balance: Long,
    rating: Double,
    ratingCount: Int,
    unread: Int,
    onWallet: () -> Unit,
    onNotifications: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // **والجرس أوّل ما يقع عليه الإبهام** — في جهة القراءة.
        Box(
            Modifier
                .clip(CircleShape)
                .clickable(onClick = onNotifications)
                .padding(8.dp),
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_bell),
                contentDescription = stringResource(R.string.top_notifications),
                tint = InkMuted,
                modifier = Modifier.size(24.dp),
            )
            if (unread > 0) {
                Box(
                    Modifier
                        .align(Alignment.TopEnd)
                        .offset(x = 4.dp, y = (-4).dp)
                        .clip(CircleShape)
                        .background(StateRed)
                        .padding(horizontal = 5.dp, vertical = 1.dp),
                ) {
                    Text(
                        // **وتسعة فما فوق تُكتب «+٩»** — رقم من ثلاث
                        // خانات يخرج من دائرته.
                        text = if (unread > 9) "+9" else unread.toString(),
                        color = Color.White,
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }

        Row(verticalAlignment = Alignment.CenterVertically) {
            // ══════════════════════════════════════════════════════════
            // **وتقييمُه بجانب محفظته**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «بالأعلى جانب المحفظة لازم يكون
            //  موجود تقييم السائق».)
            //
            // **والتقييم يعرفه المكتب ولا يعرفه صاحبه**: يُحاسَب على رقم
            // لم يره قطّ. **ومن رآه ينزل** عرف قبل أن يُستدعى.
            //
            // **ومن لم يُقيَّم بعد يُكتب له «جديد»** — لا «٠٫٠»: صفرٌ في
            // مقياس من واحدٍ إلى خمسة لا يقع، **ومن رآه قرأه حكما عليه.**
            if (ratingCount > 0) {
                Row(
                    Modifier
                        .clip(RoundedCornerShape(20.dp))
                        .background(BrandOrange.copy(alpha = 0.10f))
                        .padding(horizontal = 10.dp, vertical = 7.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_star),
                        contentDescription = stringResource(R.string.top_rating),
                        tint = BrandOrange,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(Modifier.size(5.dp))
                    Text(
                        // **ومنزلة واحدة تكفي** — «٤٫٧» يقرؤها بنظرة،
                        // **و«4.6666» رقم حاسبة لا تقييم.**
                        text = "%.1f".format(rating),
                        color = BrandOrange,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Spacer(Modifier.size(4.dp))
                    // **والعدد بجانبه** — «٤٫٨ من تقييمين» غير «٤٫٨ من
                    // مئتين»، **والأوّل يتبدّل بتقييم واحد.**
                    Text(
                        text = "(" + ratingCount + ")",
                        color = InkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            } else {
                Text(
                    text = stringResource(R.string.top_rating_none),
                    color = InkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }

            Spacer(Modifier.size(8.dp))

            // **والرصيد رقم لا أيقونة وحدها** — «محفظتك» بلا رقم لا تقول
            // شيئا.
            Row(
                Modifier
                    .clip(RoundedCornerShape(20.dp))
                    .background(BrandTeal.copy(alpha = 0.08f))
                    .clickable(onClick = onWallet)
                    .padding(horizontal = 12.dp, vertical = 7.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_wallet),
                    contentDescription = stringResource(R.string.top_wallet),
                    tint = BrandTeal,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = money(balance),
                    color = BrandTeal,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
    }
}
