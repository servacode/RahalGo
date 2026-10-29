package com.rahalgo.merchant.store

import com.rahalgo.ui.RahalOutlineButton
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.DropdownMenu
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.Card
import com.rahalgo.ui.StatRow
import com.rahalgo.ui.StatBox
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money

/** **أدنى سعرِ صرفٍ يُحفظ وحدَه** — دونه كتابةٌ لم تكتمل لا سعر (تدقيقُ ٢٠٢٦-١٠-٢٧). */
internal const val MIN_RATE = 10.0

/**
 * **سؤالٌ وزرّان كبيران** (طلبُ المالك ٢٠٢٦-١٠-١٢) — المختارُ ممتلئٌ والآخرُ مؤطَّر،
 * **وبلا اختيارٍ يُقال تحتهما «اختار وحدة»** فيُعرف أنّه إلزاميّ.
 */
@Composable
internal fun BigChoice(question: String, options: List<Pair<String, Boolean>>, enabled: Boolean, onPick: (Int) -> Unit) {
    Text(question, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
    Spacer(Modifier.height(8.dp))
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        options.forEachIndexed { i, (label, on) ->
            if (on) {
                RahalButton(onClick = { onPick(i) }, enabled = enabled, modifier = Modifier.weight(1f)) {
                    Text(label, fontWeight = FontWeight.Bold)
                }
            } else {
                RahalOutlineButton(onClick = { onPick(i) }, enabled = enabled, modifier = Modifier.weight(1f)) {
                    Text(label)
                }
            }
        }
    }
    if (options.none { it.second }) {
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.choose_one), color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
    }
}
