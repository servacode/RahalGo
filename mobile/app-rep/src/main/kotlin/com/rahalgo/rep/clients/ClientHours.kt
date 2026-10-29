package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.DayHours
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Screen
import com.rahalgo.ui.hours.HoursPlan
import com.rahalgo.ui.hours.HoursPlanner
import com.rahalgo.ui.hours.NightHoursDialog
import com.rahalgo.ui.hours.opensAtNight
import com.rahalgo.ui.hours.sameHours

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ساعاتُ دوام متجر العميل — يضبطها المندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٣٠: «أكيد المندوب يقدر يضبط ساعات دوام عميله».)
 *
 * **بالمخطّط نفسِه الذي في تطبيق المتجر** (`com.rahalgo.ui.hours.HoursPlanner`) —
 * لا نسخةٌ ثانية. **والمسوّدةُ محلّيّةٌ حتّى يُضغط «حفظ»**، **وفتحٌ بعد منتصف الليل
 * يُسأل عنه قبل الحفظ** كما عند صاحب المتجر (`NightHoursDialog`).
 */
@Composable
fun ClientHours(vm: ClientsViewModel) {
    val suspended = vm.detail?.merchant?.status == "suspended"
    Screen {
        com.rahalgo.ui.PageHeader(
            title = stringResource(com.rahalgo.ui.R.string.home_row_hours),
            onBack = { vm.hoursOpen = false },
        )
        Spacer(Modifier.height(Rahal.space.sm))
        if (vm.hours.isEmpty()) {
            LoadState(vm.hoursLoading, vm.hoursError) { vm.openHours() }
            return@Screen
        }
        val sorted = vm.hours.sortedBy { it.dayOfWeek }
        val hasHours = vm.hoursSet
        val plan = remember(sorted, hasHours) { HoursPlan(sorted, hasHours) }
        var nightCheck by remember { mutableStateOf<List<DayHours>?>(null) }
        nightCheck?.let { days ->
            NightHoursDialog(days, onDismiss = { nightCheck = null }) { nightCheck = null; vm.saveHours(days) }
        }
        HoursPlanner(plan, enabled = !suspended)
        val days = plan.days()
        val dirty = plan.chosen && (!hasHours || !sameHours(days, sorted))
        if (dirty && !suspended) {
            Spacer(Modifier.height(Rahal.space.lg))
            RahalButton(
                onClick = { if (days.any { opensAtNight(it) }) nightCheck = days else vm.saveHours(days) },
                enabled = !vm.hoursSaving,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
            ) { Text(stringResource(com.rahalgo.ui.R.string.act_save), fontWeight = FontWeight.Bold) }
        }
        Spacer(Modifier.height(Rahal.space.xl))
    }
}
