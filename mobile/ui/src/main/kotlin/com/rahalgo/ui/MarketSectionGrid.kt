package com.rahalgo.ui

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal
import com.rahalgo.ui.hours.CheckBadge

/** **قسمٌ من أقسام السوق للاختيار** — اسمُه وصورتُه، أيّاً كان التطبيقُ الذي جاء منه. */
data class SectionChoice(val id: String, val name: String, val imageUrl: String?)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شبكةُ أقسام السوق — واحدةٌ للمتجر والمندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * ثلاثةُ مربّعاتٍ متساويةٍ في كلّ سطر، صورةُ القسم والاسمُ تحتها، **والمختارُ بحدّ العلامة
 * وعلامةِ صحّ**، وبحثٌ فوقها يصفّي بالاسم. (ملاحظةُ المالك ٢٠٢٦-١٠-٢٨ للمتجر.)
 *
 * **ونُقلت من تطبيق المتجر إلى هنا** (قرارُ المالك ٢٠٢٦-١١-١١: «اقسام السوق… لازم تكون صور
 * نفس الاقسام العادية») — كان المندوبُ يرى الأقسامَ رقاقاتٍ نصّيّة.
 */
@Composable
fun MarketSectionGrid(
    all: List<SectionChoice>,
    picked: Set<String>,
    enabled: Boolean = true,
    onToggle: (String) -> Unit,
) {
    var query by rememberSaveable { mutableStateOf("") }
    OutlinedTextField(
        value = query,
        onValueChange = { query = it.take(40) },
        placeholder = { Text(stringResource(R.string.ms_search)) },
        leadingIcon = {
            Icon(
                painterResource(R.drawable.ic_search),
                contentDescription = null,
                tint = Rahal.colors.inkMuted,
            )
        },
        trailingIcon = if (query.isNotEmpty()) {
            {
                IconButton(onClick = { query = "" }) {
                    Icon(
                        painterResource(R.drawable.ic_close),
                        contentDescription = stringResource(R.string.act_cancel),
                        tint = Rahal.colors.inkMuted,
                    )
                }
            }
        } else {
            null
        },
        singleLine = true,
        shape = Rahal.shape.md,
        colors = OutlinedTextFieldDefaults.colors(
            unfocusedContainerColor = Rahal.colors.field,
            focusedContainerColor = Rahal.colors.field,
            unfocusedBorderColor = Color.Transparent,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(Rahal.space.sm))
    // **والعددُ ما يُرى عليه ✓ لا طولُ القائمة** (رُئي على جوال المالك ٢٠٢٦-١١-١١: «31» وعليه ✓ واحدة) —
    // المتجرُ قد يحمل معرّفاتٍ لأقسامٍ ليست في السوق الآن.
    val pickedShown = all.count { it.id in picked }
    Text(
        stringResource(R.string.ms_picked, pickedShown),
        color = if (pickedShown == 0) Rahal.colors.inkMuted else Rahal.colors.brand,
        fontWeight = FontWeight.Bold,
        style = Rahal.type.body,
    )
    Spacer(Modifier.height(Rahal.space.sm))
    val q = query.trim()
    val shown = if (q.isEmpty()) all else all.filter { it.name.contains(q, ignoreCase = true) }
    if (shown.isEmpty()) {
        Text(
            stringResource(R.string.ms_no_match),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.body,
            modifier = Modifier.padding(vertical = Rahal.space.md),
        )
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
        shown.chunked(3).forEach { row ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                row.forEach { sec ->
                    MarketSectionTile(
                        sec = sec,
                        on = sec.id in picked,
                        enabled = enabled,
                        modifier = Modifier.weight(1f),
                    ) { onToggle(sec.id) }
                }
                // **والسطرُ الناقصُ لا يمدّ مربّعاته** — تبقى بعرض أخواتها.
                repeat(3 - row.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
}

@Composable
private fun MarketSectionTile(
    sec: SectionChoice,
    on: Boolean,
    enabled: Boolean,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    Column(
        modifier
            .clip(Rahal.shape.md)
            .clickable(enabled = enabled, role = Role.Checkbox, onClick = onClick)
            .padding(Rahal.space.xs),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(1f),
        ) {
            // **وقسمٌ بلا صورةٍ يُرسم بحرفه الأوّل على أرضٍ باهتة** (`RemoteImage`).
            RemoteImage(
                url = AppCore.get().media(sec.imageUrl),
                name = sec.name,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(Rahal.shape.md)
                    .border(
                        if (on) Rahal.stroke.focus else Rahal.stroke.hair,
                        if (on) Rahal.colors.brand else Rahal.colors.line,
                        Rahal.shape.md,
                    ),
            )
            if (on) {
                CheckBadge(
                    Modifier
                        .align(Alignment.TopEnd)
                        .padding(Rahal.space.xs),
                )
            }
        }
        Spacer(Modifier.height(Rahal.space.xs))
        Text(
            sec.name,
            color = if (on) Rahal.colors.brand else Rahal.colors.ink,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
            style = Rahal.type.body,
            textAlign = TextAlign.Center,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}
