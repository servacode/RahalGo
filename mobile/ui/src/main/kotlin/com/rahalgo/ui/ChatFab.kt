package com.rahalgo.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp

import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قرصُ الحديث الطافي — لا زرٌّ داخلَ بطاقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٩: «الدردشةُ عند الزبون يجب أن تكون أيقونةً
 *
 *  عائمةً مثل السائق وليس داخلَ الكرت، فتكون واضحةً الدردشةُ والرسائلُ
 *  للزبون».)
 *
 * # ولماذا طافياً
 *
 * **الزرُّ في البطاقة يُقرأ تفصيلاً من تفاصيلها** — بين «ألغِ» و«اشتكِ»
 * و«قيّم»، **فيضيع بينها.** ومن انتظر ردَّ سائقه لا يريد أن يبحث عن
 * مكانِ حديثِه في قائمة.
 *
 * **والطافي يبقى في موضعٍ واحدٍ مهما تمرّرت القائمة** — **وموضعٌ ثابتٌ
 * يُتعلَّم مرّةً**، وهو ما يفعله تطبيقُ السائق.
 *
 * # ولا يظهر بلا مُحدَّث
 *
 * **قرصُ حديثٍ بلا سائقٍ يُضغط فيُفتح فراغ** — وهي القاعدةُ نفسُها التي
 * حكمت الزرَّ في البطاقة، **ولا تتبدّل بتبدّل موضعِه.**
 *
 * # وفوق شريط التنقّل لا تحته
 *
 * **حشوةٌ من الأسفل تُبعده عن التبويبات** — **وقرصٌ ملتصقٌ بشريط
 * التنقّل يُضغط أحدُهما مكانَ الآخر.**
 */
@Composable
fun BoxScope.ChatFab(
    unread: Int = 0,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    Box(
        modifier
            .align(Alignment.BottomStart)
            .padding(start = 18.dp, bottom = 22.dp),
    ) {
        BadgedBox(
            badge = {
                // **ولا شارةَ لصفر** — **شارةٌ فارغةٌ تُقرأ رسالةً لم
                // تُقرأ**، فيُفتح الحديثُ فلا جديدَ فيه.
                if (unread > 0) {
                    Badge { Text(if (unread > 99) "99+" else "$unread") }
                }
            },
        ) {
            FloatingActionButton(
                onClick = onClick,
                shape = CircleShape,
                containerColor = Rahal.colors.brand,
                contentColor = Rahal.colors.onBrand,
                modifier = Modifier
                    .size(58.dp)
                    // **وظلٌّ يرفعه عن القائمة** — **وقرصٌ بلا ظلٍّ
                    // يُقرأ جزءاً من البطاقة تحته.**
                    .shadow(10.dp, CircleShape),
            ) {
                Icon(
                    // **وأيقونةُ المستودع نفسُها** (`ic_chat`) — **ولا
                    // مكتبةَ أيقوناتٍ جديدةٌ لأجل قرصٍ واحد**، ورمزان
                    // للحديث في تطبيقٍ واحدٍ يُقرآن شيئين.
                    painter = painterResource(R.drawable.ic_chat),
                    contentDescription = stringResource(R.string.ord_chat),
                    modifier = Modifier.size(26.dp),
                )
            }
        }
    }
}
