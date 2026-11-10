package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Screen

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ السوق لمتجر العميل — يضبطها المندوب** (قرارُ المالك ٢٠٢٦-١١-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أقسامُ المنصّة** (شاورما · برغر · مشاوي…) التي يظهر المتجرُ تحتها في السوق —
 * **لا أقسامُ قائمته الداخليّة.** والقواعدُ كما عند صاحب المتجر: القائمةُ تُستبدل كلُّها،
 * **وفارغُها يعني الكلّ**، والموقوفُ لا يُبدَّل. **والمسوّدةُ محلّيّةٌ حتّى «حفظ».**
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ClientSections(vm: ClientsViewModel) {
    val suspended = vm.detail?.merchant?.status == "suspended"
    Screen {
        com.rahalgo.ui.PageHeader(
            title = stringResource(R.string.cd_sections),
            onBack = { vm.sectionsOpen = false },
        )
        Spacer(Modifier.height(Rahal.space.sm))
        if (vm.sectionsLoading || vm.sectionsError.isNotEmpty()) {
            LoadState(vm.sectionsLoading, vm.sectionsError) { vm.openSections() }
            return@Screen
        }
        if (vm.allSections.isEmpty()) {
            com.rahalgo.ui.EmptyState(title = stringResource(R.string.cd_sections_none), icon = R.drawable.ic_grid)
            return@Screen
        }
        val saved = vm.mySections.map { it.id }.toSet()
        var picked by remember(vm.mySections) { mutableStateOf(saved) }
        Text(
            stringResource(R.string.cd_sections_hint),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.body,
        )
        Spacer(Modifier.height(Rahal.space.md))
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        ) {
            vm.allSections.forEach { sec ->
                val on = sec.id in picked
                FilterChip(
                    selected = on,
                    enabled = !suspended,
                    onClick = { picked = if (on) picked - sec.id else picked + sec.id },
                    label = { Text(sec.name, style = Rahal.type.label) },
                    shape = Rahal.shape.pill,
                )
            }
        }
        if (picked.isEmpty()) {
            Spacer(Modifier.height(Rahal.space.sm))
            Text(
                stringResource(R.string.cd_sections_empty),
                color = Rahal.colors.danger,
                style = Rahal.type.caption,
            )
        }
        if (picked != saved && !suspended) {
            Spacer(Modifier.height(Rahal.space.lg))
            RahalButton(
                onClick = { vm.saveSections(vm.allSections.map { it.id }.filter { it in picked }) },
                enabled = !vm.sectionsSaving,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(Rahal.size.action),
            ) { Text(stringResource(com.rahalgo.ui.R.string.act_save), fontWeight = FontWeight.Bold) }
        }
        Spacer(Modifier.height(Rahal.space.xl))
    }
}
