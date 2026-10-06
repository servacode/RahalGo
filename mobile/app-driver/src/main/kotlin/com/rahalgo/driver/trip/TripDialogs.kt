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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
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
internal fun AgreeDialog(
    /** **من يحدّد الأجرة** (Batch 2c) — `admin_defined` أو `driver_defined`. */
    feeSource: String,
    /** **لقطةُ أجرة المنصة** — تُملأ سلفاً حين تحدّدها المنصة. */
    feeSnapshot: Long?,
    /** **أيجوز للسائق تعديلُها** — حين تحدّدها المنصة. */
    driverMayChange: Boolean,
    /**
     * **ما وُثّق قبلُ يُملأ سلفاً** (بلاغُ المالك ٢٠٢٦-١٠-٠٣: «أضغط زرَّ التوثيق يرجعلي الخانات
     * فاضية، كان بدّي أرجع أكتب السعرَ من جديد») — فالتعديلُ يبدأ ممّا اتُّفق عليه.
     */
    currentGoods: Long? = null,
    currentFee: Long? = null,
    /**
     * **أيُّ خطوة** (قرارُ المالك ٢٠٢٦-١٠-٠٣): `fee` الأجرةُ وحدَها، `goods` ثمنُ البضاعة وحدَه.
     */
    step: String = "fee",
    /** **ردُّ الخادم بعربيّة** — يبقى في النافذة ليصحّح الرقمَ ولا يُعاد كتابتُه. */
    error: String = "",
    /** **يُرسل** — فلا يُضغط التأكيدُ مرّتين. */
    busy: Boolean = false,
    onConfirm: (Long, Long) -> Unit,
    onDismiss: () -> Unit,
) {
    val adminDefined = feeSource == "admin_defined"
    // **مفروضةٌ لا تُعدَّل** حين تحدّدها المنصةُ ولا تأذن للسائق (2c) —
    // **والمحرّكُ يفرضها على كلّ حال**، والحقلُ المقفلُ يقول ذلك للسائق.
    val feeLocked = adminDefined && !driverMayChange
    var goods by remember { mutableStateOf(currentGoods?.takeIf { it > 0 }?.toString() ?: "") }
    // **تُملأ سلفاً من لقطة المنصة إن حدّدتها** — مقفلةً أو قابلةً للتعديل.
    var fee by remember {
        mutableStateOf(
            currentFee?.toString()
                ?: if (adminDefined) (feeSnapshot?.toString() ?: "0") else "",
        )
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                stringResource(if (step == "goods") R.string.agree_goods_title else R.string.agree_fee_title),
                fontWeight = FontWeight.Bold,
            )
        },
        text = {
            // **نافذةٌ بقالب التأكيد** (بلاغُ المالك ٢٠٢٦-١٠-٠٦: «نفس المشكلة البصرية») — المبلغُ
            // كبيرٌ بوحدته، **والزرُّ الأساسيُّ بعرض النافذة** لا نصٌّ صغيرٌ في زاويتها.
            Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    stringResource(if (step == "goods") R.string.agree_goods_hint else R.string.agree_fee_hint),
                    color = Rahal.colors.inkMuted,
                )
                Spacer(Modifier.height(10.dp))
                if (step == "goods") OutlinedTextField(
                    value = goods,
                    onValueChange = { goods = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_goods)) },
                    singleLine = true,
                    textStyle = MaterialTheme.typography.headlineSmall.copy(
                        textAlign = TextAlign.Center,
                        fontWeight = FontWeight.Bold,
                    ),
                    suffix = { Text(stringResource(R.string.currency_short), fontWeight = FontWeight.Bold) },
                    modifier = Modifier.fillMaxWidth(),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                if (step != "goods") OutlinedTextField(
                    value = fee,
                    onValueChange = { if (!feeLocked) fee = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_fee)) },
                    singleLine = true,
                    textStyle = MaterialTheme.typography.headlineSmall.copy(
                        textAlign = TextAlign.Center,
                        fontWeight = FontWeight.Bold,
                    ),
                    suffix = { Text(stringResource(R.string.currency_short), fontWeight = FontWeight.Bold) },
                    modifier = Modifier.fillMaxWidth(),
                    readOnly = feeLocked,
                    enabled = !feeLocked,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                if (feeLocked) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        stringResource(R.string.agree_fee_fixed),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (error.isNotEmpty()) {
                    Spacer(Modifier.height(8.dp))
                    Text(error, color = Rahal.colors.accent, style = MaterialTheme.typography.bodySmall)
                }
            }
        },
        confirmButton = {
            RahalButton(
                onClick = { onConfirm(goods.toLongOrNull() ?: 0L, fee.toLongOrNull() ?: 0L) },
                modifier = Modifier.fillMaxWidth(),
                // **والأجرةُ حقُّه فلا يمضي بلا رقم** — إلّا حين تُفرَض فتكون معلومة. **والثمنُ كذلك.**
                enabled = !busy && if (step == "goods") goods.isNotBlank() else feeLocked || fee.isNotBlank(),
            ) {
                Text(stringResource(R.string.agree_confirm), fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.act_back))
            }
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
// ══════════════════════════════════════════════════════════════════════
// **بلاغُ الطوارئ لا يُظهر نجاحاً غامضاً** (`DRV-DEF-001`)
// ══════════════════════════════════════════════════════════════════════
//
// **ثلاثُ حالاتٍ صريحة**: يُرسِل (لا يُغلَق ولا يُضغط ثانيةً) · سقط (يُقال «لم
// يصل» صراحةً مع إعادةٍ آمنةٍ يمنع المحرّكُ تكرارَها) · ونجاحٌ يُغلق البابَ
// من الخارجِ بعد إقرارِ الخادمِ لا قبله. **فلا يظنّ السائقُ العملياتِ أُبلغت
// وهي لم تُبلَّغ.**
@Composable
internal fun EmergencyDialog(
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    busy: Boolean = false,
    error: String = "",
    onRetry: () -> Unit = onConfirm,
) {
    val failed = error.isNotEmpty()
    AlertDialog(
        // **ولا يُغلَق بلمسةٍ خارجه وهو يُرسِل** — إغلاقٌ يُقرأ إلغاءً كاذبا.
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(stringResource(R.string.emg_title)) },
        text = {
            Column {
                Text(
                    if (failed) stringResource(R.string.emg_failed) else stringResource(R.string.emg_body),
                    color = if (failed) Rahal.colors.danger else Rahal.colors.inkMuted,
                )
                if (failed) {
                    Spacer(Modifier.height(6.dp))
                    Text(
                        error,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        },
        confirmButton = {
            when {
                busy -> RahalTextButton(onClick = {}, enabled = false) {
                    Text(stringResource(R.string.cta_sending), color = Rahal.colors.inkMuted)
                }
                failed -> RahalTextButton(onClick = onRetry) {
                    Text(stringResource(R.string.act_retry), color = Rahal.colors.danger)
                }
                else -> RahalTextButton(onClick = onConfirm) {
                    Text(stringResource(R.string.emg_send), color = Rahal.colors.danger)
                }
            }
        },
        dismissButton = {
            // **ولا رجوعَ وهو يُرسِل** — لا يُترك البلاغُ معلَّقاً بلا خبر.
            if (!busy) {
                RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_back)) }
            }
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
    status: String,
    loadedAtMs: Long,
    error: String,
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
    onMine: (String) -> Unit,
    onRetry: () -> Unit,
) {
    // ══════════════════════════════════════════════════════════════════
    // **لكلّ مرحلةٍ عملُها — والقرارُ الذي لا يُردّ يُؤكَّد** (٢٠٢٦-١٠-٠٢)
    // ══════════════════════════════════════════════════════════════════
    //
    // **قرارُ المالك**: «زرّ لدي مشكلة له عملٌ معيّنٌ بكلّ مرحلة… لا يُغلق الطلبُ قبل
    // الزبون». **فالخياراتُ ثلاثة أصناف**: بلاغٌ لا يمسّ الطلب، وسببٌ يُنهيه أو يسلّمه
    // للعمليات (بتأكيدٍ يقول أثرَه)، و«مشكلتي» — إعادةٌ قبل الاستلام وطارئٌ بعده.
    // **وانتظارُ الباب يُعدّ تنازليّاً** — «الزبونُ غير موجود» بعد خمس دقائق.
    var pending by remember { mutableStateOf<Pending?>(null) }
    var nowMs by remember { mutableStateOf(android.os.SystemClock.elapsedRealtime()) }
    androidx.compose.runtime.LaunchedEffect(loadedAtMs) {
        while (true) {
            nowMs = android.os.SystemClock.elapsedRealtime()
            kotlinx.coroutines.delay(1_000)
        }
    }
    val confirm = pending
    if (confirm != null) {
        val beforePickup = status == "assigned" || status == "at_pickup"
        AlertDialog(
            onDismissRequest = { pending = null },
            title = { Text(stringResource(R.string.confirm_title)) },
            text = {
                Text(
                    stringResource(
                        when (confirm) {
                            is Pending.Mine ->
                                if (beforePickup) R.string.confirm_release_body else R.string.confirm_emergency_body
                            is Pending.Reason -> when {
                                confirm.item.kind == "release" -> R.string.confirm_release_body
                                // **والبلاغُ لا يُعفيه** (قِيس في دورة المحاكي ٢٠٢٦-١٠-٠٣): مشكلةُ المتجر صارت
                                // بلاغاً والطلبُ معه — وكان النصُّ «وتُعفى أنت منه».
                                confirm.item.kind == "report" -> R.string.confirm_report_body
                                !confirm.item.closes -> R.string.confirm_merchant_body
                                confirm.item.fault == "customer" -> R.string.confirm_customer_body
                                else -> R.string.confirm_close_body
                            }
                        },
                    ),
                )
            },
            confirmButton = {
                RahalTextButton(
                    onClick = {
                        pending = null
                        when (confirm) {
                            is Pending.Mine -> onMine(confirm.label)
                            is Pending.Reason -> onPick(confirm.item.code)
                        }
                    },
                    tone = Tone.Danger,
                ) { Text(stringResource(R.string.act_confirm)) }
            },
            dismissButton = {
                RahalTextButton(onClick = { pending = null }) { Text(stringResource(R.string.act_cancel)) }
            },
        )
        return
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.problem_title)) },
        text = {
            Column(Modifier.verticalScroll(rememberScrollState())) {
                if (error.isNotEmpty()) {
                    Text(error, color = Rahal.colors.accent)
                    RahalTextButton(onClick = onRetry, modifier = Modifier.fillMaxWidth()) {
                        Text(stringResource(R.string.problem_retry), modifier = Modifier.fillMaxWidth())
                    }
                    HorizontalDivider(Modifier.padding(vertical = 6.dp))
                }
                val reports = reasons.filter { it.kind == "report" }
                val releases = reasons.filter { it.kind == "release" }
                val ends = reasons.filter { it.kind == "fail" }
                if (reports.isNotEmpty()) {
                    Section(R.string.problem_reports_title)
                    for (r in reports) {
                        RahalTextButton(onClick = { onPick(r.code) }, modifier = Modifier.fillMaxWidth()) {
                            Text(reasonLabel(r.code), modifier = Modifier.fillMaxWidth())
                        }
                    }
                    HorizontalDivider(Modifier.padding(vertical = 6.dp))
                }
                if (ends.isNotEmpty()) {
                    Section(if (ends.all { it.closes }) R.string.problem_ends_title else R.string.problem_ops_title)
                    for (r in ends) {
                        val left = (r.availableInSec - (nowMs - loadedAtMs) / 1000).coerceAtLeast(0)
                        RahalTextButton(
                            onClick = { pending = Pending.Reason(r) },
                            enabled = left == 0L,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(
                                if (left > 0) {
                                    reasonLabel(r.code) + " · " + stringResource(
                                        R.string.problem_wait, "%d:%02d".format(left / 60, left % 60),
                                    )
                                } else {
                                    reasonLabel(r.code)
                                },
                                modifier = Modifier.fillMaxWidth(),
                            )
                        }
                    }
                    HorizontalDivider(Modifier.padding(vertical = 6.dp))
                }
                Section(R.string.problem_mine)
                // **قبل الاستلام: أسبابُ التركِ من الخادم** (يذهب الطلبُ لغيره ويُغلَق دوامُه)؛
                // **وبعده: «مشكلتي» طارئٌ** يبقى معه الطلبُ وتُنبَّه العمليات.
                if (releases.isNotEmpty()) {
                    for (r in releases) {
                        RahalTextButton(
                            onClick = { pending = Pending.Reason(r) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(reasonLabel(r.code), color = Rahal.colors.accent, modifier = Modifier.fillMaxWidth())
                        }
                    }
                } else {
                    for (id in MINE) {
                        val label = stringResource(id)
                        RahalTextButton(
                            onClick = { pending = Pending.Mine(label) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(label, color = Rahal.colors.accent, modifier = Modifier.fillMaxWidth())
                        }
                    }
                }
            }
        },
        confirmButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.act_cancel)) }
        },
    )
}

/** **ما ينتظر تأكيداً** — سببٌ من الخادم أو «مشكلتي». */
private sealed interface Pending {
    data class Reason(val item: FailReasonItem) : Pending
    data class Mine(val label: String) : Pending
}

@Composable
private fun Section(title: Int) {
    Text(
        text = stringResource(title),
        color = Rahal.colors.inkMuted,
        style = MaterialTheme.typography.labelMedium,
        modifier = Modifier.padding(bottom = 2.dp),
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
    "bike_broken" -> stringResource(R.string.problem_bike)
    "accident" -> stringResource(R.string.problem_crash)
    "force_majeure" -> stringResource(R.string.problem_force)
    "customer_cancelled_by_phone" -> stringResource(R.string.reason_customer_cancelled_by_phone)
    "customer_no_answer" -> stringResource(R.string.reason_customer_no_answer)
    else -> code
}
