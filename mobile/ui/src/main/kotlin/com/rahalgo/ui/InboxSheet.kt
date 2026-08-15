package com.rahalgo.ui

import androidx.compose.foundation.background
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.shared.model.Notice

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صندوق الإشعارات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وشارة الجرس تُقرأ ولا تُفتح على فراغ** — زرّ يعرض عددا ثمّ لا يفتح
 * شيئا **يُقرأ عطبا.**
 *
 * # وغير المقروء يُعلَّم بنقطة
 *
 * **لا بخلفيّة ملوّنة**: عشرة إشعارات غير مقروءة تجعل الشاشة كلّها
 * لونا واحدا، **فلا يُقرأ منها شيء.**
 *
 * # ولا تُفتح شاشة بالضغط
 *
 * (قرار المالك ٢٠٢٦-٠٨-١١: «الإشعارات عند الضغط تفتح صفحات غير موجودة
 * — يكفي أن تكون مقروءة وغير مقروءة وتوصل لنا الخبر».)
 */
@Composable
fun InboxSheet(items: List<Notice>, onMarkAll: () -> Unit) {
    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.canvas)
            .statusBarsPadding(),
    ) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.inbox_title),
                style = MaterialTheme.typography.titleMedium,
            )
            // ══════════════════════════════════════════════════════════
            // **ولا زرَّ رجوعٍ هنا — الجرسُ يفتح ويُغلق**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «وألغِ زرَّ الرجوع، لازم في
            //  تعليم الكلّ كمقروء صحّ».)
            //
            // **وموضعُ الزرِّ صار لفعلٍ يخصُّ الصندوق** — والخروجُ من
            // حيث دخل: **الجرسُ نفسُه، ورجوعُ النظام.**
            //
            // **ولا يظهر إلّا إن كان فيه ما لم يُقرأ** — وزرٌّ يُضغط فلا
            // يتغيّر شيءٌ يُقرأ عطبا. (وهو عينُ ما تفعله صفحةُ الويب.)
            if (items.any { !it.read }) {
                TextButton(onClick = onMarkAll) {
                    Text(stringResource(R.string.inbox_mark_all))
                }
            }
        }

        if (items.isEmpty()) {
            Text(
                text = stringResource(R.string.inbox_empty),
                color = Rahal.colors.inkMuted,
                modifier = Modifier.fillMaxWidth().padding(24.dp),
            )
            return@Column
        }

        // ══════════════════════════════════════════════════════════════
        // **ومجموعةٌ بالأيّام — كما في صفحة الويب**
        // ══════════════════════════════════════════════════════════════
        //
        // (أمرُ المالك ٢٠٢٦-٠٨-١٣: «اعمل جرداً للإشعارات» · «التطبيقُ
        //  والويبُ نفسُ النموذج».)
        //
        // **وثلاثون خبراً في قائمةٍ متّصلةٍ تُقرأ كتلةً واحدة** — ومن فتح
        // ليعرف ما جدّ اليومَ لا يعرف أين ينتهي اليومُ ويبدأ الأمس.
        var day = ""
        LazyColumn(Modifier.fillMaxSize().padding(horizontal = 16.dp)) {
            items(items) { notice ->
                val d = dayText(notice.createdAt)
                if (d != day) {
                    day = d
                    DayHead(d)
                }
                Row(Modifier.fillMaxWidth().padding(vertical = 10.dp)) {
                    // **ونقطة لغير المقروء** — تُرى ولا تصبغ السطر.
                    Box(
                        Modifier
                            .padding(top = 6.dp)
                            .size(8.dp)
                            .clip(CircleShape)
                            .background(if (notice.read) Color.Transparent else Rahal.colors.brand),
                    )
                    Spacer(Modifier.size(10.dp))
                    Column(Modifier.weight(1f)) {
                        Text(notice.title, fontWeight = FontWeight.Bold)
                        if (notice.body.isNotBlank()) {
                            Spacer(Modifier.height(2.dp))
                            Text(notice.body, color = Rahal.colors.inkMuted)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun Box(modifier: Modifier, content: @Composable () -> Unit = {}) {
    androidx.compose.foundation.layout.Box(modifier) { content() }
}
