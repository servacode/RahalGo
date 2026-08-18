package com.rahalgo.ui

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
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.foundation.layout.Column
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشريط العلويّ — محفظتك وإشعاراتك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «نضيف الجرسَ والمحفظةَ بالأعلى وزرَّ الثيم…
 *  هذا أيضاً مركزيٌّ بكلّ التطبيقات».)
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
 * # ولا زرَّ سمةٍ هنا
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «نخلّيها بالقائمة الجانبيّة أفضل».)
 *
 * **والسمةُ إعدادٌ يُضبط مرّةً** — **وفعلٌ يُفعل مرّةً في العمر يأخذ
 * من موضعٍ يُنظر إليه مئةَ مرّةٍ في اليوم.**
 *
 * # والشارة على الجرس لا بجانبه
 *
 * **تطفو فوقه في زاويته** — وهو ما تفعله كلّ التطبيقات، **ومن وضعها
 * بجانبه** جعلها تُقرأ رقما آخر لا عدّ إشعارات.
 */
@Composable
fun TopBar(
    balance: Long,
    unread: Int,
    /**
     * **رقاقةُ التقييم — لمن يُقيَّم وحدَه.**
     *
     * **والسائقُ يُحاسَب على رقمٍ لم يره قطّ** إن غاب، **والزبونُ لا
     * يُحاسَب عليه** فلا تُعرض له: **رقاقةٌ لا معنى لها ازدحامٌ لا خبر.**
     *
     * **وفارغةٌ تعني لا تُرسم** — لا رايةٌ ثانيةٌ تُنسى فتتناقض معها.
     */
    onRating: (() -> Unit)? = null,
    rating: Double = 0.0,
    ratingCount: Int = 0,
    onWallet: () -> Unit,
    onNotifications: () -> Unit,
    /**
     * **ثلاثةُ خطوطٍ تفتح القائمة — أو فراغٌ فلا تُرسم.**
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «القائمةُ الجانبيّةُ يجب ألّا تُفتح في
     *  قسم الرحلة أبداً، كي لا تُفتح وتُغلق شاشةَ الرحلة».)
     *
     * **وزرٌّ يُعطَّل ولا يُخفى أسوأ**: يُضغط فلا يقع شيءٌ فيُعاد ضغطُه.
     * **وما لا يُفعل في هذه الشاشة لا يُرسم فيها.**
     */
    onMenu: (() -> Unit)? = null,
    /**
     * **يتصفّح ولم يدخل بعد.**
     *
     * **فلا جرسَ ولا رصيد**: **شارةُ إشعاراتٍ لمن لا حسابَ له تقول
     * «لديك خبر» وليس له خبر**، **ورصيدُ صفرٍ يُقرأ «محفظتي فارغة» لا
     * «لا محفظةَ لي».**
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا يمكن أن يرى كلَّ المعلومات وهو لم
     *  يسجّل دخولاً بعد».)
     */
    guest: Boolean = false,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **بابُ عنوان التوصيل — أوّلُ ما يُقرأ في الشريط**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨ بصورةِ مرجع: «ضيف لي الزرَّ بالأعلى،
     *  البناءُ سيعتمد كلُّه على هذا».)
     *
     * **والزبونُ وحدَه له عنوانُ توصيل** — السائقُ يتحرّك والمندوبُ
     * يزور، **فيبقى فارغاً عندهما فلا يُرسم.**
     *
     * # ولماذا في الشريط لا في السلّة
     *
     * **العنوانُ يقرّر ما يُعرض**: أيُّ متجرٍ يصل إليه وكم أجرتُه.
     * **ومن اختاره عند الدفع بنى سلّةً من متاجرَ لا تصله**، فيُلغيها
     * ويبدأ من جديد.
     *
     * **وهو أوّلُ ما يُقرأ**: «التوصيل إلى…» سطرٌ يجيب عن سؤالٍ يسأله
     * كلُّ من يفتح تطبيقَ توصيل.
     */
    onAddress: (() -> Unit)? = null,
    /** **العنوانُ المختار** — وفارغٌ يعني «اختر عنواناً». */
    addressLabel: String = "",
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
        if (onMenu != null) {
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
        } else {
            // **وفراغٌ بمقاسه** — وإلّا انزاح الرصيدُ والجرسُ إلى اليمين
            // كلّما دخل رحلةً وخرج منها: **شريطٌ يرقص لا يُقرأ ثابتا.**
            Spacer(Modifier.size(34.dp))
        }
        Spacer(Modifier.size(4.dp))

        // ══════════════════════════════════════════════════════════════
        // **وعنوانُ التوصيل — سطران يُضغطان**
        // ══════════════════════════════════════════════════════════════
        //
        // (صورةُ المالك المرجعيّة ٢٠٢٦-٠٨-١٨.)
        //
        // **وسطرٌ فوق سطر**: «التوصيل إلى» يقول ما هو، **والثاني يقول
        // ما اخترتَ** — **ورقاقةٌ بسطرٍ واحدٍ تُقرأ زرّاً لا جوابا.**
        //
        // **ويأخذ ما بقي من العرض** (`weight`) — **وعنوانٌ طويلٌ يُقصّ
        // بنقاطٍ خيرٌ من شريطٍ يدفع الجرسَ خارجَ الشاشة.**
        if (onAddress != null) {
            Row(
                Modifier
                    .weight(1f)
                    .clip(Rahal.shape.md)
                    .clickable(onClick = onAddress)
                    .padding(horizontal = 6.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_pin),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(20.dp),
                )
                Spacer(Modifier.size(8.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        text = stringResource(R.string.top_deliver_to),
                        color = Rahal.colors.brand,
                        style = MaterialTheme.typography.labelSmall,
                    )
                    Text(
                        text = addressLabel.ifEmpty {
                            stringResource(R.string.top_pick_address)
                        },
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
        }

        Row(verticalAlignment = Alignment.CenterVertically) {
            if (guest) return@Row
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
            if (onRating != null && ratingCount > 0) {
                Row(
                    Modifier
                        .clip(Rahal.shape.lg)
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
            } else if (onRating != null) {
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
                    .clip(Rahal.shape.lg)
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
            Spacer(Modifier.size(4.dp))

            // ══════════════════════════════════════════════════════════
            // **والجرسُ في الطرف — بعد المحفظة**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «حطّوه أقصى اليسار بعد
            //  المحفظة، هون مكانه الطبيعيّ — مو بنصّ الشاشة».)
            //
            // **وكان في وسط الشريط** بين القائمة والمحفظة — **فوقع
            // في فراغٍ لا يقصده أحد**: العينُ تمسح الأطرافَ أوّلاً،
            // **وما في الوسط بلا جارٍ يُقرأ زائدا.**
            //
            // **والأخبارُ تجتمع**: رصيدٌ وجرسٌ في ركنٍ واحد — نظرةٌ
            // واحدةٌ تقرؤهما.
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
                // **والشارةُ على الجرس في زاويته** — وهو ما تفعله
                // كلُّ التطبيقات، **ومن وضعها بجانبه** جعلها تُقرأ
                // رقماً آخرَ لا عدَّ إشعارات.
                CountBadge(
                    count = unread,
                    color = Rahal.colors.danger,
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .offset(x = 5.dp, y = (-5).dp),
                )
            }

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
