package com.rahalgo.customer.cart

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Item
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«يُطلب معه» — صفٌّ في السلّة لا شاشةٌ ثانية**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٢: «مع الأكل مثلاً كولا، عيران، والأحجام — هي
 *  حركةٌ ذكيّةٌ بالأكل تعرض عليه كولا أو عيران، هي أكثرُ شيءٍ تنطلب».)
 *
 * # ولماذا في السلّة لا في السوق
 *
 * **الاقتراحُ يقع حين يكون الطلبُ قد تقرّر** — من فتح سلّتَه انتهى من
 * الاختيار، **وهذه لحظةُ «نسيتَ المشروب» الوحيدة.** واقتراحٌ في وسط
 * التصفّح يزاحم ما يبحث عنه.
 *
 * # وصنفٌ مستقلٌّ لا خيارٌ ملصوق
 *
 * **«يُضاف» يلصق الكولا بالساندويش** — سطرٌ واحدٌ بسعرٍ واحد. **ومن طلب
 * ثلاثةَ ساندويشاتٍ وكولتين لا يستطيع قولَها** بالخيارات: إمّا كولا
 * لكلٍّ أو لا شيء.
 *
 * # ولا يُعرض فارغاً
 *
 * **عنوانٌ بلا محتوى يُقرأ عطباً** — فالصفُّ كلُّه يختفي حين لا اقتراح،
 * ولا يُترك عنوانٌ فوق فراغ.
 */
@Composable
fun SuggestRow(items: List<Item>, media: (String?) -> String?, onAdd: (Item) -> Unit) {
    if (items.isEmpty()) return

    Column(Modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.suggest_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(8.dp))
        LazyRow(
            contentPadding = PaddingValues(vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            items(items, key = { it.id }) { item ->
                Column(
                    Modifier
                        .width(104.dp)
                        .clip(Rahal.shape.md)
                        // **وضغطةٌ واحدةٌ تُدخله** — **ونافذةُ تأكيدٍ
                        // لأجل كولا تُلغي فائدةَ الاقتراح كلَّها.**
                        .clickable { onAdd(item) },
                ) {
                    RemoteImage(
                        // **والمصغّرةُ أوّلاً** — كبطاقة السوق.
                        url = media(item.imageThumbUrl ?: item.imageUrl),
                        name = item.name,
                        modifier = Modifier.fillMaxWidth().aspectRatio(1f).clip(Rahal.shape.sm),
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(
                        item.name,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Text(
                        money(item.price),
                        color = Rahal.colors.brand,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
    }
}
