package com.rahalgo.driver.trip

import com.rahalgo.navigation.TripMap
import com.rahalgo.navigation.MarkerIcons
import androidx.compose.animation.AnimatedVisibility
import com.rahalgo.ui.Countdown
import androidx.compose.foundation.layout.IntrinsicSize
import com.rahalgo.design.Rahal
import com.rahalgo.ui.etaText
import com.rahalgo.ui.minutesShort
import com.rahalgo.ui.dist
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.material3.Surface
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.layout.width
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.driver.BuildConfig
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.RahalTextButton
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نوافذُ الرحلة** — ما يُسأل قبل فعلٍ لا يُردّ.
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أُخرج من `TripScreen.kt`** (٢٠٢٦-٠٨-٢٣، فحصُ التطبيق):
 * **ألفان ومئتا سطرٍ في ملفٍّ واحدٍ يصعب تعديلُه بلا كسر.**
 */

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الاتّفاق على الطلب الخاصّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثمن البضاعة اختياريّ** — قد تكون أمانةً لا ثمن لها، **فيُترك فارغا
 * ويُقرأ صفرا.** **ومن ألزم برقم في كلّ طلب** جعل السائق يكتب ما ليس
 * صحيحا ليمضي.
 *
 * **وأجرة التوصيل لا تُترك**: هي حقّه، **وطلبٌ بلا أجرة اتّفاقٌ ناقص.**
 */
@Composable
internal fun AgreeDialog(onConfirm: (Long, Long) -> Unit, onDismiss: () -> Unit) {
    var goods by remember { mutableStateOf("") }
    var fee by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.agree_title)) },
        text = {
            Column {
                Text(stringResource(R.string.agree_hint), color = Rahal.colors.inkMuted)
                Spacer(Modifier.height(10.dp))
                OutlinedTextField(
                    value = goods,
                    onValueChange = { goods = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_goods)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = fee,
                    onValueChange = { fee = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_fee)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = { onConfirm(goods.toLongOrNull() ?: 0L, fee.toLongOrNull() ?: 0L) },
                enabled = fee.isNotBlank(),
            ) {
                Text(stringResource(R.string.agree_confirm))
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_cancel)) }
        },
    )
}

/**
 * **فعلٌ ثانويٌّ ملوّن** — أيقونةٌ وكلمةٌ على أرضٍ صلبة.
 *
 * **ولا حشوةَ عريضة**: ثلاثةُ أزرارٍ في صفٍّ واحدٍ على شاشةِ هاتف،
 * **وكلُّ نقطةٍ زائدةٍ تدفع الأوّلَ إلى سطرين.**
 */
@Composable
internal fun SmallAction(
    icon: Int,
    label: Int,
    // **ونبرةٌ لا لون** — انظر `Tone`: اللونُ يقول الشكلَ والنبرةُ
    // تقول المعنى، **وثلاثةُ أزرارٍ في صفٍّ بألوانٍ مكتوبةٍ بيدٍ تنجو
    // من كلّ توحيدٍ لاحق.**
    tone: Tone,
    onClick: () -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    busy: Boolean = false,
) {
    RahalButton(
        onClick = onClick,
        enabled = enabled,
        tone = tone,
        // **وضيّقٌ لأنّ ثلاثةً في صفٍّ على شاشةِ هاتف** — والحشوةُ
        // العريضةُ تدفع الأوّلَ إلى سطرين.
        compact = true,
        modifier = modifier,
    ) {
        if (busy) {
            CircularProgressIndicator(
                Modifier.size(16.dp),
                strokeWidth = 2.dp,
                color = Color.White,
            )
            return@RahalButton
        }
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.size(4.dp))
        Text(
            text = stringResource(label),
            // **وحجمٌ واحدٌ للثلاثة** — (طلب المالك ٢٠٢٦-٠٨-١٢: «لازم
            // تكون بنفس الشكل»). **وزرٌّ أكبرُ من جاره** يُقرأ أهمَّ
            // منه، وهي ثلاثةُ أفعالٍ لا فعلٌ وحاشيتان.
            style = MaterialTheme.typography.labelMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بلاغُ الطارئ — ضغطةٌ واحدةٌ وتأكيد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا حقلَ يملؤه**: من عطلت درّاجتُه أو أُوقف في الطريق لا يكتب شرحا،
 * **وحقلٌ إلزاميٌّ في لحظةٍ كهذه** يجعله يترك الزرَّ ويتّصل بالمكتب.
 *
 * **وتأكيدٌ واحدٌ يسبقه**: بلاغٌ يُوقظ المكتبَ ويحرّر الطلب، **وضغطةٌ
 * بالخطأ في جيبٍ** تفعل ذلك كلَّه.
 */
@Composable
internal fun EmergencyDialog(onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.emg_title)) },
        text = { Text(stringResource(R.string.emg_body), color = Rahal.colors.inkMuted) },
        confirmButton = {
            RahalTextButton(onClick = onConfirm) {
                Text(stringResource(R.string.emg_send), color = Rahal.colors.danger)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
        },
    )
}

/**
 * **ما قد يقع للسائق نفسِه** — ثلاثةٌ تغطّي ما يقع فعلا.
 *
 * **ولا حقلَ حرّ**: من كتب «مشكلة» بيده لم يقل شيئا يُقاس، **ولا يُعدّ
 * ولا يُقارن شهرا بشهر.** وثلاثةُ ألفاظٍ تُختار في ثانيةٍ وهو واقف.
 */
private val MINE = listOf(
    R.string.problem_bike,
    R.string.problem_crash,
    R.string.problem_force,
)

/**
 * **أسباب التعذّر — تُختار ولا تُكتب.**
 *
 * **والسبب يقرّر من يتحمّل**: «المتجر مغلق» ذنب متجر يستوجب تعويض
 * السائق، **و«تأخّرت» ذنبه هو.** ونصّ حرّ لا يُعدّ ولا يُقاس.
 */
@Composable
internal fun FailDialog(
    reasons: List<FailReasonItem>,
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
    onMine: (String) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        // **والعنوان سؤالٌ لا حكم** — (تصحيح المالك ٢٠٢٦-٠٨-١٢):
        // **«لماذا تعذّر» تفترض أنّ الطلب سقط**، والزرُّ يقول «لدي
        // مشكلة» — وأكثرُ المشاكل تُحلّ بسائقٍ ثانٍ لا بإلغاء.
        title = { Text(stringResource(R.string.problem_title)) },
        text = {
            Column {
                for (r in reasons) {
                    RahalTextButton(onClick = { onPick(r.code) }, modifier = Modifier.fillMaxWidth()) {
                        Text(reasonLabel(r.code), modifier = Modifier.fillMaxWidth())
                    }
                }
                // ══════════════════════════════════════════════════════
                // **والمشكلةُ قد تكون عنده هو — فيمضي ويأتي غيرُه**
                // ══════════════════════════════════════════════════════
                //
                // (قرار المالك ٢٠٢٦-٠٨-١٢: «المنصّة هي ترسل سائقاً
                //  ثانياً في حال حصلت مشكلة للسائق عند المتجر».)
                //
                // **وأسبابُ المتجر كلُّها ذنبُ متجر** — ومن عطلت
                // درّاجتُه فاختار «المتجر مغلق» ليمضي **حمّل متجراً
                // بريئاً ذنباً وتعويضا.**
                //
                // **والطلبُ لا يُلغى بل يعود للطابور**: الزبونُ ينتظر
                // طعامه، **وسائقٌ ثانٍ يأخذه في دقيقة** — وإلغاؤه
                // لعطلٍ في درّاجةٍ عقوبةٌ على من لا ذنب له.
                if (reasons.isNotEmpty()) {
                    HorizontalDivider(Modifier.padding(vertical = 6.dp))
                }
                Text(
                    text = stringResource(R.string.problem_mine),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelMedium,
                    modifier = Modifier.padding(bottom = 2.dp),
                )
                for (id in MINE) {
                    val label = stringResource(id)
                    RahalTextButton(
                        onClick = { onMine(label) },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(label, color = Rahal.colors.accent, modifier = Modifier.fillMaxWidth())
                    }
                }
            }
        },
        confirmButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_cancel)) }
        },
    )
}

/** **ورمز بلا ترجمة يُعرض كما هو** — ليُعرف ويُضاف، لا ليُبتلع. */
@Composable
internal fun reasonLabel(code: String): String = when (code) {
    "customer_absent" -> stringResource(R.string.reason_customer_absent)
    "customer_refused" -> stringResource(R.string.reason_customer_refused)
    "customer_unreachable" -> stringResource(R.string.reason_customer_unreachable)
    "address_wrong" -> stringResource(R.string.reason_address_wrong)
    "driver_late" -> stringResource(R.string.reason_driver_late)
    "merchant_closed" -> stringResource(R.string.reason_merchant_closed)
    "merchant_refused" -> stringResource(R.string.reason_merchant_refused)
    "merchant_not_ready" -> stringResource(R.string.reason_merchant_not_ready)
    "order_unknown" -> stringResource(R.string.reason_order_unknown)
    else -> code
}
