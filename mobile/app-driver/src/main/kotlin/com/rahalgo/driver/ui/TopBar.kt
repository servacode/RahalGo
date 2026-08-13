package com.rahalgo.driver.ui

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
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
import androidx.compose.foundation.layout.widthIn
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
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
    onRating: () -> Unit,
    onMenu: () -> Unit,
    /** **يقلب السمة** — والأيقونةُ تُظهر الوجهةَ لا الحال. */
    onTheme: () -> Unit,
    /** **أنحن في الغامقة الآن** — فيُرسم الهلالُ أو الشمس. */
    dark: Boolean,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // ══════════════════════════════════════════════════════════════
        // **وثلاثةُ خطوطٍ تفتح القائمة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «٣ خطوطٍ عائمة تفتح القائمة… أحسن
        //  من «المزيد» ومن اسمٍ معيّن، فالكلّ يعرف أنّ لها معنىً
        //  واضحا».)
        //
        // **ولا تحتاج اسماً** — ومن رآها في أيّ تطبيقٍ عرف ما تفتح.
        // **وهي أصوبُ من تبويبٍ خامس**: الشريطُ السفليُّ أربعةٌ الآن،
        // وخامسٌ بأسماءٍ عربيّةٍ يضغطه حتّى تُقصّ الكلمات.
        Box(
            Modifier
                .clip(CircleShape)
                .clickable(onClick = onMenu)
                .padding(6.dp),
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_menu),
                contentDescription = stringResource(R.string.menu_open),
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(22.dp),
            )
        }
        Spacer(Modifier.size(4.dp))

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
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(24.dp),
            )
            if (unread > 0) {
                Box(
                    Modifier
                        .align(Alignment.TopEnd)
                        .offset(x = 4.dp, y = (-4).dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.danger)
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

        // ══════════════════════════════════════════════════════════════
        // **وأيقونةُ السمة — رسمٌ اليومَ وفعلٌ غدا**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «الآن ضع أيقونة الثيم الغامق
        //  والفاتح بالأعلى، بدون أيّ عملٍ حاليّاً فقط أيقونة».)
        //
        // **ولا تفعل شيئاً بعد** — والسمةُ الغامقة تحتاج طقمَ ألوانٍ
        // ثانياً لكلّ شاشة، **وهو عملٌ يُقاس ويُقرَّر** لا يُلحق بأيقونة.
        Box(
            Modifier
                .clip(CircleShape)
                .clickable(onClick = onTheme)
                .padding(8.dp),
        ) {
            Icon(
                // **والأيقونةُ تُظهر الوجهةَ لا الحال** — هلالٌ يقول
                // «انتقل إلى الغامقة»، وشمسٌ تقول «عُد إلى الفاتحة».
                // **ومن رسم حالَه الحاليَّ جعل اللمسةَ تفاجئ.**
                painter = painterResource(
                    if (dark) R.drawable.ic_theme_light else R.drawable.ic_theme,
                ),
                contentDescription = stringResource(
                    if (dark) R.string.top_theme_light else R.string.top_theme,
                ),
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(22.dp),
            )
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
                        .background(Rahal.colors.accent.copy(alpha = 0.10f))
                        // **وضغطُه يفتح تقييماته** — (قرارُ المالك
                        // ٢٠٢٦-٠٨-١٣: «عند النقر عليه يفتح صفحة التقييم
                        // كما هي بالويب، فيعرف ما هي التقييمات التي
                        // حصل عليها ومن أين»).
                        //
                        // **ورقمٌ بلا تفصيلٍ يُقلق ولا يُعلّم**: من نزل
                        // تقييمُه من ٥ إلى ٤٫٢ **لا يعرف أيَّ بابٍ كان
                        // ولا ماذا قيل**، فيسأل عمّا فعل ولا جواب.
                        .clickable(onClick = onRating)
                        .padding(horizontal = 10.dp, vertical = 7.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_star),
                        contentDescription = stringResource(R.string.top_rating),
                        tint = Rahal.colors.accent,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(Modifier.size(5.dp))
                    Text(
                        // **ومنزلة واحدة تكفي** — «٤٫٧» يقرؤها بنظرة،
                        // **و«4.6666» رقم حاسبة لا تقييم.**
                        // **وبأرقام غربيّة كالرصيد بجانبه** — `format`
                        // بلا لغة يكتب «٥٫٠» في جهاز عربيّ، **فيقع
                        // رقمان بخطّين في شريط واحد.**
                        text = "%.1f".format(java.util.Locale.US, rating),
                        color = Rahal.colors.accent,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    // **ولا عددَ بجانبه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣:
                    // «احذف عدد التقييمات من الأعلى، اترك التقييم
                    // فقط»).
                    //
                    // **وموضعُه الصفحةُ التي تُفتح بضغطه** — هناك
                    // يُقرأ مع ما يشرحه: من قيّم وكم ومتى.
                }
            } else {
                Text(
                    text = stringResource(R.string.top_rating_none),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }


            Spacer(Modifier.size(10.dp))

            // **والرصيد رقم لا أيقونة وحدها** — «محفظتك» بلا رقم لا تقول
            // شيئا.
            Row(
                Modifier
                    .clip(RoundedCornerShape(20.dp))
                    .background(Rahal.colors.brand.copy(alpha = 0.08f))
                    .clickable(onClick = onWallet)
                    .padding(horizontal = 12.dp, vertical = 7.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_wallet),
                    contentDescription = stringResource(R.string.top_wallet),
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = money(balance),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            Spacer(Modifier.size(10.dp))

            // ══════════════════════════════════════════════════════════
            // **وصورتُه واسمُه نزلا إلى الشريط السفليّ**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «شو رأيك نزّل ملفي لتحت، مو
            //  أفضل ونخفّف الازدحام بالأعلى؟».)
            //
            // **وكان الشريطُ خمسةَ عناصر** — والاسمُ أعرضُها (حتّى مئةٍ
            // وثلاثين)، **فبنزوله يخفّ الازدحامُ أكثرَ ممّا يوحي
            // عددُه.**
            //
            // **وقاعدةٌ تفصلهما**: الأعلى أخبارٌ تُقرأ بنظرةٍ — رصيدٌ
            // وتقييمٌ وجرس. **والأسفلُ أماكنُ يُنتقل إليها.**
            // **والصورةُ مكانٌ لا خبر.**
        }
    }
}
