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
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.unit.dp
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
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun AddLandmarkScreen(onSend: (name: String, kind: String) -> Unit, sending: Boolean = false) {
    var name by rememberSaveable { mutableStateOf("") }
    var kind by rememberSaveable { mutableStateOf("") }
    Column(
        Modifier
            .fillMaxWidth()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        // **وبعنوان شاشات القائمة نفسِه** (`ScreenTitle`) — لا شكلٌ ثانٍ.
        ScreenTitle(stringResource(R.string.landmark_title), stringResource(R.string.landmark_hint))
        OutlinedTextField(
            value = name,
            onValueChange = { if (it.length <= 80) name = it },
            label = { Text(stringResource(R.string.landmark_name)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            LandmarkKinds.forEach { (key, label) ->
                val on = kind == key
                Box(
                    modifier = Modifier
                        .clip(Rahal.shape.sm)
                        .background(if (on) Rahal.colors.brand else Rahal.colors.field)
                        .clickable { kind = key }
                        .heightIn(min = 44.dp)
                        .padding(horizontal = 14.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = stringResource(label),
                        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                        fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
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
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = androidx.compose.ui.graphics.Color.White)
            } else {
                Text(stringResource(R.string.landmark_send))
            }
        }
    }
}
