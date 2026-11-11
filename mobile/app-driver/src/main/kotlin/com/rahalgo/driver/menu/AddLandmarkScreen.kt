package com.rahalgo.driver.menu

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.ui.Alignment
import com.rahalgo.ui.ScreenTitle
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.ui.RahalButton

/** **أنواعُ المعلم ونصُّها** — كما يقرؤها الخادم (`landmarkKinds`). */
internal val LandmarkKinds = listOf(
    "mosque" to R.string.lm_mosque,
    "pharmacy" to R.string.lm_pharmacy,
    "school" to R.string.lm_school,
    "hospital" to R.string.lm_hospital,
    "shop" to R.string.lm_shop,
    "roundabout" to R.string.lm_roundabout,
    "other" to R.string.lm_other,
)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أضف معلم»** (قرارُ المالك ٢٠٢٦-١٠-٢١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **من مكانه الآن** — يقف عند المعلم ويكتب اسمَه ونوعَه. **ولا يظهر لأحدٍ قبل موافقة
 * الإدارة**، ثمّ يدخل «قرب المعلم» في طلبات السائقين.
 *
 * **وسطرُ «معالمك اليوم: ٢ من ٥»** (قرارُ المالك ٢٠٢٦-١١-٠٥) — والمكرَّرُ لا يُعدّ، يقوله الخادم.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun AddLandmarkScreen(
    onSend: (name: String, kind: String) -> Unit,
    sending: Boolean = false,
    today: com.rahalgo.shared.driver.LandmarkToday? = null,
) {
    var name by rememberSaveable { mutableStateOf("") }
    var kind by rememberSaveable { mutableStateOf("") }
    Column(
        Modifier
            .fillMaxWidth()
            .padding(Rahal.space.lg),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.md),
    ) {
        // **والعنوانُ رأسٌ بسهمٍ فوق الصفحة** — ويبقى سطرُ التعليمة لأنّه طريقةُ الاستعمال.
        ScreenTitle("", stringResource(R.string.landmark_hint))
        // **والعدُّ المطفأُ لا سطرَ له.**
        if (today != null && today.quota > 0) {
            Text(
                text = stringResource(R.string.landmark_today, today.today, today.quota),
                color = if (today.today >= today.quota) Rahal.colors.brand else Rahal.colors.ink,
                style = Rahal.type.bodyStrong,
            )
        }
        // **الحقلُ الموحّد** — اسمُه فوقه دائماً (إعادةُ التصميم ٢٠٢٦-١١-١١).
        com.rahalgo.ui.LabeledField(
            label = stringResource(R.string.landmark_name),
            value = name,
            onValueChange = { if (it.length <= 80) name = it },
        )
        FlowRow(horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm), verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            LandmarkKinds.forEach { (key, label) ->
                val on = kind == key
                Box(
                    modifier = Modifier
                        .clip(Rahal.shape.pill)
                        .background(if (on) Rahal.colors.brand else Rahal.colors.field)
                        .clickable(role = androidx.compose.ui.semantics.Role.RadioButton) { kind = key }
                        .heightIn(min = Rahal.size.touch)
                        .padding(horizontal = Rahal.space.md),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = stringResource(label),
                        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                        style = if (on) Rahal.type.bodyStrong else Rahal.type.body,
                    )
                }
            }
        }
        RahalButton(
            onClick = { onSend(name, kind) },
            // **ولا يُرسَل مرّتين** — يتعطّل حتّى يجيء الردّ، ودائرةٌ تقول إنّه يرسل.
            enabled = !sending && name.trim().length >= 2 && kind.isNotEmpty(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (sending) {
                CircularProgressIndicator(Modifier.size(Rahal.icon.sm), strokeWidth = Rahal.stroke.focus, color = Rahal.colors.onBrand)
            } else {
                Text(stringResource(R.string.landmark_send), style = Rahal.type.button)
            }
        }
    }
}
