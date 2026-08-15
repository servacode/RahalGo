package com.rahalgo.map

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.geo.Place
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **منتقي نقطةٍ — يضعها حيث يريد لا حيث هو**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: نبدأ بالمندوب — **ويلزمه أن يضع نقطةَ متجرٍ
 *  ليست نقطتَه.**)
 *
 * # والفرقُ عن زرّ «حدّد موقعي»
 *
 * **الزبونُ يقف في بيته فيقرأ هاتفُه موضعَه** — وذلك يكفيه.
 *
 * **والمندوبُ يضيف متجراً وهو في مكتبه**، **والمتجرُ يصحّح نقطتَه بعد
 * أن أخطأ من أدخلها**، **والزبونُ يحفظ بيتَ أمّه.** **وثلاثتُهم يضعون
 * نقطةً ليست تحت أقدامهم.**
 *
 * # والدبّوسُ ثابتٌ والخريطةُ تتحرّك
 *
 * **لا دبّوسٌ يُسحب بالإصبع**: الإصبعُ يغطّيه فلا يُرى أين يقع.
 * **والخريطةُ تتحرّك تحت دبّوسٍ في وسطها** — فيرى ما تحته دائما، **وهو
 * ما تفعله كلُّ التطبيقات** فيُعرف بلا تعليم.
 *
 * # والعنوانُ يُقرأ بعد أن تستقرّ
 *
 * **ونداءٌ مع كلّ حركةٍ يُغرق المزوّدَ ويستنزف الحزمة** — فيُنادى حين
 * يرفع إصبعَه.
 *
 * # ولا تُغلق إلّا بنقطة
 *
 * **ومن خرج بلا اختيارٍ يعود بلا شيء** — **ونقطةٌ بإحداثيٍّ صفرٍ تُرسل
 * سائقاً إلى المحيط الأطلسيّ.**
 */
@Composable
fun PickPoint(
    /** **من أين تبدأ الخريطة** — موضعُه إن عُرف، وإلّا فمركزُ المدينة. */
    start: LatLng?,
    /** **اسمُ الموضع الحاليّ** — يُملأ من البحث أو من قراءة النقطة. */
    vm: PickPointViewModel,
    onPick: (LatLng, String) -> Unit,
    onCancel: () -> Unit,
) {
    var query by remember { mutableStateOf("") }

    Box(Modifier.fillMaxSize()) {
        MapCanvas(
            start = start ?: RAQQA,
            onSettle = vm::readAddress,
            jumpTo = vm.jumpTo,
            onJumped = vm::jumped,
            modifier = Modifier.fillMaxSize(),
        )

        // **والدبّوسُ في وسط الشاشة لا على الخريطة** — لا يتحرّك معها،
        // **فما تحته هو المختار.**
        Icon(
            painter = painterResource(R.drawable.ic_pin_center),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier
                .align(Alignment.Center)
                // **ورأسُ الدبّوس هو النقطة لا وسطُه** — فيُرفع بنصف
                // طوله: **من حاذى وسطَه وضع نقطةً تحت ما يشير إليه.**
                .offset(y = (-18).dp)
                .size(36.dp),
        )

        // ══════════════════════════════════════════════════════════════
        // **والبحثُ في الأعلى** — لمن يضيف متجراً وهو بعيدٌ عنه
        // ══════════════════════════════════════════════════════════════
        Column(
            Modifier
                .align(Alignment.TopCenter)
                .fillMaxWidth()
                .padding(12.dp),
        ) {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it; vm.search(it) },
                label = { Text(stringResource(R.string.pick_search)) },
                singleLine = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Rahal.colors.canvas, RoundedCornerShape(12.dp)),
            )
            // **ونتائجُ البحث تحته** — تُضغط فتقفز الخريطةُ إليها.
            vm.results.take(4).forEach { p ->
                Spacer(Modifier.height(4.dp))
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .background(Rahal.colors.canvas)
                        .padding(10.dp),
                ) {
                    TextButton(onClick = { query = p.label; vm.goTo(p) }) {
                        Text(
                            text = p.label,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وما تحت الدبّوس مكتوبٌ قبل أن يُثبَّت**
        // ══════════════════════════════════════════════════════════════
        //
        // **ومن ثبّت نقطةً بلا أن يقرأ اسمَها لا يعرف ماذا اختار** —
        // **والخريطةُ في حيٍّ لا لافتاتِ فيه تبدو متشابهة.**
        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(Rahal.colors.canvas)
                .padding(14.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (vm.reading) {
                    CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    Spacer(Modifier.size(8.dp))
                }
                Text(
                    text = vm.label.ifEmpty { stringResource(R.string.pick_move_map) },
                    color = if (vm.label.isEmpty()) Rahal.colors.inkMuted else Color.Unspecified,
                    fontWeight = FontWeight.Medium,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = { vm.point?.let { onPick(it, vm.label) } },
                    // **ولا يُثبَّت قبل أن تُقرأ نقطة** — نقطةٌ صفريّةٌ
                    // تُرسل سائقاً إلى لا مكان.
                    enabled = vm.point != null,
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.pick_confirm)) }
                OutlinedButton(onClick = onCancel, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.pick_cancel))
                }
            }
        }
    }
}

/** **مركزُ الرقّة** — حيث تبدأ الخريطةُ لمن لا موضعَ له. */
val RAQQA = LatLng(35.9528, 39.0079)
