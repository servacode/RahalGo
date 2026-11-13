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

import androidx.compose.foundation.layout.offset
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.absoluteOffset
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInWindow
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
/** **ارتفاعُ ما يحجبه القرصُ من أسفل الشاشة** (٥٨ + ٢٢ + هامش ١٦). */
val ChatFabInset = 96.dp

/**
 * **حشوةُ الذيل التي تتركها كلُّ شاشةٍ تمريرٍ للقرص** — صفرٌ ما دام القرصُ غائباً.
 * (رُئي ٢٠٢٦-١١-٠٤: القرصُ يغطّي آخرَ اسمٍ في السوق وثاني بطاقةٍ في «طلباتي» وحقلَ كلمة السرّ.)
 */
val LocalChatFabInset = androidx.compose.runtime.compositionLocalOf { 0.dp }

/**
 * **موضعُ القرص بعد سحبه** (قرارُ المالك ٢٠٢٦-١١-١٣: «زر الدردشة العائم يغطي على كثير اشياء لازم نقدر
 * نحركها») — يبقى حيث تُرك طولَ الجلسة وفي كلّ الشاشات، **ويعود إلى زاويته بإعادة فتح التطبيق.**
 */
object ChatFabPos {
    var x by androidx.compose.runtime.mutableFloatStateOf(0f)
    var y by androidx.compose.runtime.mutableFloatStateOf(0f)
    /** **متى انتهى آخرُ سحب** — كي لا تُقرأ رفعةُ الإصبع بعد السحب ضغطةً تفتح الحديث. */
    internal var draggedAt = 0L
    internal var winX = 0f
    internal var winY = 0f
    internal var placedX = 0f
    internal var placedY = 0f
}

@Composable
fun BoxScope.ChatFab(
    unread: Int = 0,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val density = androidx.compose.ui.platform.LocalDensity.current
    val cfg = androidx.compose.ui.platform.LocalConfiguration.current
    val rtl = androidx.compose.ui.platform.LocalLayoutDirection.current == androidx.compose.ui.unit.LayoutDirection.Rtl
    // **حدودُ السحب داخل الشاشة** — من زاويته إلى الطرف الآخر أفقيّاً، وإلى ما تحت الرأس عموديّاً.
    val spanX = with(density) { (cfg.screenWidthDp.dp - 58.dp - 36.dp).toPx() }
    val spanY = with(density) { (cfg.screenHeightDp.dp - 58.dp - 220.dp).toPx() }
    val screenW = with(density) { cfg.screenWidthDp.dp.toPx() }
    Box(
        modifier
            .align(Alignment.BottomStart)
            .padding(start = 18.dp, bottom = 22.dp)
            // **إزاحةٌ مطلقةٌ لا تتبع جهةَ القراءة** — الموجبُ يمينٌ دائماً، فلا قلبَ إشارةٍ يُخطئ.
            .absoluteOffset {
                // **ما رُسم عنده فعلاً** — إحداثيّاتُ الحدث محسوبةٌ على هذا الموضع.
                ChatFabPos.placedX = ChatFabPos.x
                ChatFabPos.placedY = ChatFabPos.y
                androidx.compose.ui.unit.IntOffset(ChatFabPos.x.toInt(), ChatFabPos.y.toInt())
            }
            // **وموضعُه على الشاشة بعد كلّ رسم** — يُضاف إلى موضع الإصبع داخلَه فيصير موضعاً على الشاشة.
            .onGloballyPositioned { c ->
                val p = c.positionInWindow()
                ChatFabPos.winX = p.x
                ChatFabPos.winY = p.y
            }
            .pointerInput(spanX, spanY, screenW) {
                // **السحبُ بموضع الإصبع على الشاشة** — إحداثيّاتُ القرص تتحرّك معه فيضيع الفرق
                // (رُئي على الجهاز ٢٠٢٦-١١-١٣: القرصُ يقطع خُمسَ المسافة).
                awaitEachGesture {
                    val down = awaitFirstDown(requireUnconsumed = false)
                    val startX = ChatFabPos.x
                    val startY = ChatFabPos.y
                    // **أيُّ زاويةٍ هو فيها** — يميناً (العربيّة) فيُسحب يساراً، أو يساراً فيُسحب يميناً.
                    val baseX = ChatFabPos.winX - startX
                    val rightSide = baseX + size.width / 2f > screenW / 2f
                    val fx = down.position.x + ChatFabPos.placedX
                    val fy = down.position.y + ChatFabPos.placedY
                    var dragging = false
                    while (true) {
                        val ev = awaitPointerEvent()
                        val ch = ev.changes.firstOrNull { it.id == down.id } ?: break
                        if (!ch.pressed) break
                        val dx = ch.position.x + ChatFabPos.placedX - fx
                        val dy = ch.position.y + ChatFabPos.placedY - fy
                        if (!dragging && kotlin.math.hypot(dx, dy) > viewConfiguration.touchSlop) dragging = true
                        if (dragging) {
                            ch.consume()
                            val nx = startX + dx
                            ChatFabPos.x = if (rightSide) nx.coerceIn(-spanX, 0f) else nx.coerceIn(0f, spanX)
                            ChatFabPos.y = (startY + dy).coerceIn(-spanY, 0f)
                        }
                    }
                    if (dragging) ChatFabPos.draggedAt = System.currentTimeMillis()
                }
            },
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
                onClick = { if (System.currentTimeMillis() - ChatFabPos.draggedAt > 350) onClick() },
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
