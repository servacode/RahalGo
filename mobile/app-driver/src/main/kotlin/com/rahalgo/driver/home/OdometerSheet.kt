package com.rahalgo.driver.home

import com.rahalgo.ui.DialogDismissButton
import com.rahalgo.ui.DialogConfirmButton
import android.graphics.BitmapFactory
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.DialogProperties
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.shared.driver.OdometerInput
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.rememberCameraShot

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ورقةُ العدّاد — صورةٌ ورقم** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «اول مايفتح الدوام يطلب منه تصوير العداد الصبح، وعند اغلاق الدوام كمان.» **الكاميرا
 * وحدَها** (`rememberCameraShot` — لا معرض)، **والرقمُ يُكتب كما هو** (`OdometerInput`).
 *
 * **ولا تخطّي إلّا إن قال المحرّك** (`skippable`، وضعُ «يطلب») — وفي «إلزامي» لا يُفتح
 * الدوامُ ولا يُسكَّر بلاها. **والحكمُ كلُّه في المحرّك**: الرقمُ الأقلُّ مرفوض، والقفزةُ
 * الكبيرةُ تُؤكَّد. الشاشةُ تعرض وترسل.
 */
data class OdometerAsk(
    /** `start` عند الفتح · `end` عند السكّر. */
    val phase: String,
    val skippable: Boolean,
    /** **السكّرُ بالخروج من التطبيق** — يُكمَل الخروجُ بعده. */
    val forLogout: Boolean = false,
    val lastKm: Double? = null,
    val busy: Boolean = false,
    val error: String = "",
    /** **قفزةٌ كبيرةٌ تنتظر تأكيده** — نصُّ السؤال، وفارغٌ لا سؤال. */
    val confirm: String? = null,
    /** **زرُّ «بدّلت الدرّاجة» معروض** (`OdometerFlow.bikeChangeOffered`) — قرارُ المالك ٢٠٢٦-١١-٠٦. */
    val bikeOffer: Boolean = false,
    /** **سؤالُ «بدّلت الدرّاجة؟» مفتوح.** */
    val bikeAsk: Boolean = false,
    /** **أكّد التبديل** — الرقمُ يُرسل بعلامة `bike_change`. */
    val bikeChange: Boolean = false,
)

data class OdometerActions(
    val submit: (jpeg: ByteArray?, km: String, confirm: Boolean) -> Unit,
    val skip: () -> Unit,
    val dismiss: () -> Unit,
    /** **أُغلق سؤالُ التأكيد ليعدّل الرقم.** */
    val fix: () -> Unit,
    val cameraDenied: () -> Unit,
    /** **«بدّلت الدرّاجة / العدّاد»**: يفتح السؤال · يؤكّده · يغلقه. */
    val bikeAsk: () -> Unit = {},
    val bikeConfirm: () -> Unit = {},
    val bikeCancel: () -> Unit = {},
)

@Composable
fun OdometerSheet(ask: OdometerAsk, actions: OdometerActions) {
    var photo by remember { mutableStateOf<ByteArray?>(null) }
    var km by remember { mutableStateOf("") }
    val shoot = rememberCameraShot(onDenied = actions.cameraDenied) { photo = it }
    val preview = remember(photo) {
        photo?.let { b -> runCatching { BitmapFactory.decodeByteArray(b, 0, b.size)?.asImageBitmap() }.getOrNull() }
    }

    ask.confirm?.let { question ->
        com.rahalgo.ui.RahalDialog(
            onDismissRequest = actions.fix,
            title = { Text(stringResource(R.string.odo_check_title)) },
            text = { Text(question) },
            confirmButton = {
                DialogConfirmButton(onClick = { actions.submit(photo, km, true) }) {
                    Text(stringResource(R.string.odo_check_yes))
                }
            },
            dismissButton = {
                DialogDismissButton(onClick = actions.fix) { Text(stringResource(R.string.odo_check_fix)) }
            },
        )
    }

    if (ask.bikeAsk) {
        com.rahalgo.ui.RahalDialog(
            onDismissRequest = actions.bikeCancel,
            title = { Text(stringResource(R.string.odo_bike_title)) },
            text = { Text(stringResource(R.string.odo_bike_question)) },
            confirmButton = {
                DialogConfirmButton(onClick = actions.bikeConfirm) { Text(stringResource(R.string.odo_bike_yes)) }
            },
            dismissButton = {
                DialogDismissButton(onClick = actions.bikeCancel) { Text(stringResource(R.string.odo_cancel)) }
            },
        )
    }

    com.rahalgo.ui.RahalDialog(
        onDismissRequest = { if (!ask.busy) actions.dismiss() },
        properties = DialogProperties(dismissOnClickOutside = false),
        title = {
            Text(stringResource(if (ask.phase == "start") R.string.odo_title_start else R.string.odo_title_end))
        },
        text = {
            Column(
                Modifier.fillMaxWidth().verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                Text(stringResource(R.string.odo_hint), color = Rahal.colors.inkMuted)
                if (ask.bikeChange) {
                    Text(stringResource(R.string.odo_bike_on), color = Rahal.colors.danger)
                }
                preview?.let {
                    Image(
                        bitmap = it,
                        contentDescription = stringResource(R.string.odo_photo_desc),
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxWidth().heightIn(max = 220.dp).clip(Rahal.shape.md),
                    )
                }
                RahalOutlineButton(onClick = shoot, enabled = !ask.busy, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(if (photo == null) R.string.odo_shoot else R.string.odo_reshoot))
                }
                OutlinedTextField(
                    value = km,
                    onValueChange = { v -> OdometerInput.accept(v)?.let { km = it } },
                    label = { Text(stringResource(R.string.odo_km_label)) },
                    singleLine = true,
                    enabled = !ask.busy,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    modifier = Modifier.fillMaxWidth(),
                )
                ask.lastKm?.takeIf { !ask.bikeChange }?.let {
                    Text(
                        stringResource(R.string.odo_last, OdometerInput.show(it)),
                        color = Rahal.colors.inkMuted,
                    )
                }
                if (ask.error.isNotEmpty()) Text(ask.error, color = Rahal.colors.danger)
                if (ask.bikeOffer && !ask.bikeChange) {
                    RahalTextButton(onClick = actions.bikeAsk, enabled = !ask.busy) {
                        Text(stringResource(R.string.odo_bike_button))
                    }
                }
                if (ask.skippable) {
                    RahalTextButton(onClick = actions.skip, enabled = !ask.busy) {
                        Text(stringResource(R.string.odo_skip))
                    }
                }
            }
        },
        confirmButton = {
            RahalButton(onClick = { actions.submit(photo, km, false) }, enabled = !ask.busy) {
                Text(stringResource(if (ask.phase == "start") R.string.odo_start else R.string.odo_end))
            }
        },
        dismissButton = {
            DialogDismissButton(onClick = actions.dismiss, enabled = !ask.busy) {
                Text(stringResource(R.string.odo_cancel))
            }
        },
    )
}
