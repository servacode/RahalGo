package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.shared.model.ChatMessage

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حديث الطلب — داخل الرحلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند السابع في قائمة المالك ٢٠٢٦-٠٨-١٢.)
 *
 * # ولماذا لا رقم هاتف
 *
 * (قرار المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن
 * طريق المنصّة».)
 *
 * **والمحرّك لا يرسل رقم الزبون أصلا** — لا يُخفى في الشاشة: **الإخفاء
 * في الشاشة ستارة لا قفل.**
 *
 * # وتُفتح بنفسها ولا تُطلب
 *
 * **المحرّك يفتح الحديث لحظة يقبل السائق** ويكتب أوّل سطر — **فلا يجد
 * صاحبه قناةً فارغةً** ينتظر كلٌّ فيها الآخر.
 *
 * # وتُغلق بانتهاء الطلب
 *
 * **والحقل يقوله المحرّك** (`open`) — فحين تُغلق يبقى الحديث يُقرأ
 * **ولا يُكتب فيه**: قناةٌ تعيش بعد طلبها بابٌ لمن أراد أن يصل بغير
 * المنصّة.
 */
@Composable
fun ChatSheet(state: ChatState, actions: ChatActions, modifier: Modifier = Modifier) {
    var draft by remember { mutableStateOf("") }
    // **ونحوُ ثلثَي الشاشة** (تجربةُ المالك ٢٠٢٦-١١-١٢: «صغيرة ما بتنشاف الصور ولا الرسائل»).
    val tall = (androidx.compose.ui.platform.LocalConfiguration.current.screenHeightDp * CHAT_SHEET_SCREEN_SHARE).dp

    Column(
        modifier
            .fillMaxWidth()
            .padding(horizontal = Rahal.space.md)
            // **وارتفاعٌ نحوُ ثلثَي الشاشة** (تجربةُ المالك ٢٠٢٦-١١-١٢) — كان ٣٤٠ فلا تُرى صورةٌ
            // ولا رسالتان. **ويملأ ما يُعطى حتّى حدّه** (`fillMaxHeight` بعد الحدّ)، والشريطُ
            // العلويّ وزرُّ الخطوة يبقيان: الشاشةُ تعطيه ما بقي تحتهما وحدَه (`weight`).
            .heightIn(max = tall)
            .fillMaxHeight()
            .shadow(Rahal.elevation.overlay, Rahal.shape.lg)
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.canvas)
            .imePadding(),
    ) {
        Row(
            Modifier.fillMaxWidth().padding(start = Rahal.space.lg, end = Rahal.space.xs, top = Rahal.space.xs),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = state.peerName.ifBlank { stringResource(R.string.detail_customer) },
                style = Rahal.type.sectionTitle,
                color = Rahal.colors.ink,
            )
            // **ويُغلق من زرّه ومن أيقونته** — من فتحه بضغطةٍ يغلقه
            // بمثلها، **ومن بحث عن باب الخروج وهو يقود** رفع عينه عن
            // الطريق.
            IconButton(onClick = actions.close) {
                Icon(
                    painter = painterResource(R.drawable.ic_close_circle),
                    contentDescription = stringResource(R.string.act_back),
                    tint = Rahal.colors.inkMuted,
                    modifier = Modifier.size(Rahal.icon.md),
                )
            }
        }

        // **والفارغُ يقول ماذا يفعل** — لا نافذةٌ بيضاء بلا كلمة.
        if (state.messages.isEmpty()) {
            Text(
                text = stringResource(R.string.chat_empty_driver),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.body,
                modifier = Modifier.fillMaxWidth().padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
            )
        }
        // **والقائمةُ نفسُها في كلّ التطبيقات** (`ui/ChatConversation.kt`، قرارُ المالك ٢٠٢٦-١١-١٢) —
        // تملأ ما بين العنوان والكتابة وتمرّر داخلها، والكتابةُ مثبّتةٌ تحت. ولكلّ رسالةٍ وقتُها
        // ولكلّ يومٍ تاريخُه (قرارُ المالك ٢٠٢٦-٠٨-١٢: إثباتٌ بالوقت للشكاوى).
        com.rahalgo.ui.ChatMessageList(
            state.messages,
            Modifier.weight(1f).fillMaxWidth().padding(horizontal = Rahal.space.md),
        )

        if (state.open) {
            // ══════════════════════════════════════════════════════════
            // **والإرسال أيقونةٌ في الحقل ومفتاحُ لوحةٍ معا**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «بدّل كلمة أرسل خلّيها أيقونة
            //  داخل الحقل، وزرّ الإنتر يعطي إرسال أيضا للسهولة — وهو
            //  فاتح الكيبورد يكتب ويرسل».)
            //
            // **ولوحةُ المفاتيح تغطّي نصف الشاشة**: زرٌّ خارج الحقل قد
            // يقع تحتها، **فيكتب ولا يجد أين يضغط** — ويده على مقود.
            //
            // **ومفتاحُ اللوحة يقول «إرسال» لا «تمّ»**: صورةُ المفتاح
            // نفسِها تقول ما يفعل قبل أن يُضغط.
            val send = {
                val body = draft.trim()
                if (body.isNotEmpty()) {
                    actions.send(body)
                    draft = ""
                }
            }
            // ══════════════════════════════════════════════════════════
            // **وسقوطُ الإرسال يُقال — والنصُّ يعود** (٢٠٢٦-١٠-٠٢)
            // ══════════════════════════════════════════════════════════
            //
            // **كان يُبتلع في السجلّ**: يُمحى الحقلُ ولا تظهر الرسالةُ ولا يُقال
            // لماذا، **فيظنّ الزبونَ قرأها ولم يردّ.**
            androidx.compose.runtime.LaunchedEffect(state.unsent) {
                if (state.unsent.isNotEmpty() && draft.isEmpty()) draft = state.unsent
            }
            if (state.error.isNotEmpty()) {
                Text(
                    text = state.error,
                    color = Rahal.colors.danger,
                    style = Rahal.type.caption,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = Rahal.space.md),
                )
            }
            // ══════════════════════════════════════════════════════════
            // **وردودٌ جاهزةٌ بضغطة** (تدقيق ٢٠٢٦-١٠-٢٧) — وهو يقود، والكتابةُ خطر.
            // ══════════════════════════════════════════════════════════
            val quick = listOf(
                stringResource(R.string.chat_quick_arrived),
                stringResource(R.string.chat_quick_downstairs),
                stringResource(R.string.chat_quick_searching),
                stringResource(R.string.chat_quick_five_min),
            )
            Row(
                Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState())
                    .padding(start = Rahal.space.md, end = Rahal.space.md, top = Rahal.space.sm),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                quick.forEach { q ->
                    Box(
                        Modifier
                            .clip(Rahal.shape.pill)
                            .background(Rahal.colors.field)
                            .clickable(enabled = !state.busy) { actions.send(q) }
                            .heightIn(min = Rahal.size.touch)
                            .padding(horizontal = Rahal.space.md),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(q, color = Rahal.colors.ink, style = Rahal.type.bodyStrong)
                    }
                }
            }
            // ══════════════════════════════════════════════════════════
            // **وصورةٌ ورسالةٌ صوتيّة** (قرارُ المالك ٢٠٢٦-١١-١٢) — وهو يقود، الصوتُ أسهلُ من الكتابة.
            // ══════════════════════════════════════════════════════════
            // **وشريطُ الكتابة نفسُه في كلّ التطبيقات** (`ui/ChatConversation.kt`) — صورةٌ وحقلٌ
            // ومايكٌ مكانَ «إرسال»، ومفتاحُ اللوحة يُرسل (قرارُ المالك ٢٠٢٦-٠٨-١٢).
            com.rahalgo.ui.ChatComposer(
                draft = draft,
                onDraft = { draft = it },
                busy = state.busy,
                onSend = send,
                onPhoto = actions.sendPhoto,
                onVoice = actions.sendVoice,
                modifier = Modifier.fillMaxWidth().padding(Rahal.space.md),
            )
        } else {
            // **وحديثٌ مغلقٌ يُقال إنّه مغلق** — لا حقلُ كتابةٍ لا يعمل.
            Text(
                text = stringResource(R.string.chat_closed),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.body,
                modifier = Modifier.fillMaxWidth().padding(Rahal.space.lg),
            )
        }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **والفقاعةُ وتاريخُها صارا مركزيّين** — `ui/ChatBubble.kt`
// ══════════════════════════════════════════════════════════════════════
//
// **ولزمت «دردشاتي السابقة» الشكلَ نفسَه** — ونسخُها يعني حديثاً يُقرأ
// بشكلين: **أزرقُ في الرحلة ورماديٌّ في السجلّ، والرسالةُ هي هي.**

/** ما يعرضه الحديث — **ولا يملكه هو.** */
data class ChatState(
    val messages: List<ChatMessage> = emptyList(),
    val peerName: String = "",
    val open: Boolean = true,
    val busy: Boolean = false,
    /** **سقط الإرسال** — نصُّ الخادم (٢٠٢٦-١٠-٠٢)؛ وفارغٌ لا خطأ. */
    val error: String = "",
    /** **ما لم يُرسَل** — يعود إلى الحقل فلا يُكتب ثانيةً. */
    val unsent: String = "",
)

data class ChatActions(
    val send: (String) -> Unit,
    val close: () -> Unit,
    /** **صورةٌ في الحديث** (قرارُ المالك ٢٠٢٦-١١-١٢). */
    val sendPhoto: (ByteArray) -> Unit = {},
    /** **رسالةٌ صوتيّة** — البايتاتُ والمدّةُ بالميلي ثانية. */
    val sendVoice: (ByteArray, Long) -> Unit = { _, _ -> },
)

/** **حصّةُ نافذة الدردشة من ارتفاع الشاشة** (تجربةُ المالك ٢٠٢٦-١١-١٢). */
internal const val CHAT_SHEET_SCREEN_SHARE = 0.62f
