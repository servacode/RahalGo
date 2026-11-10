package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
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
 * # وغير المقروء عريضٌ بأرضٍ خفيفة، والمقروءُ عاديّ
 *
 * (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩: كانا بشكلٍ واحدٍ إلّا نقطةً صغيرة.) **والأرضُ
 * خفيفةٌ جدّاً** — عشرةُ أخبارٍ جديدةٍ لا تصبغ الشاشةَ لوناً واحداً.
 *
 * # ولا مسافةَ شريط النظام مرّتين
 *
 * الصندوقُ يُعرض داخل `Scaffold` أعطاه مسافةَ شريطه العلويّ، **فـ`statusBarsPadding`
 * فوقها فراغٌ مضاعف.** و`statusBarInset = true` لمن يعرضه بلا شريط (السائقُ على الخريطة).
 */
@Composable
fun InboxSheet(
    items: List<Notice>,
    onMarkAll: () -> Unit,
    onOpen: (Notice) -> Unit = {},
    statusBarInset: Boolean = false,
) {
    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.surface)
            .then(if (statusBarInset) Modifier.statusBarsPadding() else Modifier),
    ) {
        Row(
            Modifier
                .fillMaxWidth()
                .heightIn(min = HeadHeight)
                .padding(horizontal = Rahal.space.lg),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.inbox_title),
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
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
                RahalTextButton(onClick = onMarkAll) {
                    Text(stringResource(R.string.inbox_mark_all))
                }
            }
        }

        if (items.isEmpty()) {
            Empty(stringResource(R.string.inbox_empty), icon = R.drawable.ic_bell)
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
        LazyColumn(Modifier.fillMaxSize()) {
            // **رأسُ اليوم من الفهرس لا من متغيّرٍ يُعدَّل أثناء الرسم** — فلا يضيع عند التمرير.
            itemsIndexed(items) { index, notice ->
                val d = dayText(notice.createdAt)
                if (index == 0 || d != dayText(items[index - 1].createdAt)) {
                    Column(Modifier.padding(horizontal = Rahal.space.lg)) { DayHead(d) }
                }
                // **والضغطُ على الخبر يعلّمه مقروءاً** (تجربةُ المالك ٢٠٢٦-١٠-٢٩: ضغط فلم يتغيّر شيء).
                // **كلُّ إشعارٍ بطاقةٌ** بأيقونةٍ في مربّعٍ ملوّن — غيرُ المقروء بلون العلامة وحدٍّ منها.
                Row(
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.xs)
                        .cardSurface(
                            Rahal.shape.md,
                            tone = if (notice.read) null else Rahal.colors.brand,
                            elevation = if (notice.read) 1.dp else 3.dp,
                        )
                        .clickable { onOpen(notice) }
                        .padding(Rahal.space.md),
                    verticalAlignment = Alignment.Top,
                ) {
                    IconTile(
                        icon = noticeIcon(notice.kind),
                        tint = if (notice.read) Rahal.colors.inkMuted else noticeTint(notice.kind),
                        size = 40.dp,
                    )
                    Spacer(Modifier.size(Rahal.space.md))
                    Column(Modifier.weight(1f)) {
                        Text(
                            BidiText.isolateNumbers(notice.title),
                            fontWeight = if (notice.read) FontWeight.Normal else FontWeight.Bold,
                            color = Rahal.colors.ink,
                        )
                        if (notice.body.isNotBlank()) {
                            Spacer(Modifier.height(Rahal.space.xs))
                            Text(
                                BidiText.isolateNumbers(notice.body),
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                        }
                    }
                    if (!notice.read) {
                        Box(
                            Modifier
                                .padding(top = Rahal.space.xs)
                                .size(Dot)
                                .clip(CircleShape)
                                .background(Rahal.colors.brand),
                        )
                    }
                }
            }
        }
    }
}

private val HeadHeight = 56.dp

/** **نقطةُ غير المقروء.** */
private val Dot = 8.dp

@Composable
private fun Box(modifier: Modifier, content: @Composable () -> Unit = {}) {
    androidx.compose.foundation.layout.Box(modifier) { content() }
}

/** **أيقونةٌ لكلّ نوعِ إشعار** (المالك ٢٠٢٦-١١-١٠) — مالٌ ومحفظة، طلب، دردشة، تقييم، تنبيهُ حساب. */
internal fun noticeIcon(kind: String): Int = when (kind) {
    "wallet" -> R.drawable.ic_wallet
    "order", "order_offer" -> R.drawable.ic_orders
    "chat" -> R.drawable.ic_chat
    "rating" -> R.drawable.ic_star
    "account" -> R.drawable.ic_warning
    "ticket" -> R.drawable.ic_info
    else -> R.drawable.ic_bell
}

@Composable
private fun noticeTint(kind: String): Color = when (kind) {
    "account" -> Rahal.colors.accent
    "wallet" -> Rahal.colors.success
    else -> Rahal.colors.brand
}
