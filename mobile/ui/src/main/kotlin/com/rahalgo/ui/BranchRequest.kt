package com.rahalgo.ui

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.BranchApi
import com.rahalgo.shared.merchant.BranchRequest
import com.rahalgo.shared.merchant.BranchRequestInput
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أضف فرع» — نموذجٌ واحدٌ لتطبيقَي المتجر والمندوب** (قرارُ المالك ٢٠٢٦-١١-١٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * اسمُ الفرع وموقعُه على الخريطة (إلزاميّان) ورقمٌ اختياريّ، **ثمّ طلباتُه بحالتها**:
 * بانتظار الموافقة · تمت الموافقة · مرفوض (وسببُه). **والخريطةُ تُعطى من التطبيق**
 * (`pointCard`) — `:ui` لا تعرفها، كما في [PointField].
 *
 * **والمحرّكُ يحكم الصلاحيةَ**: صاحبُ المتجر لمتجره وحدَه، والمندوبُ لعميله وحدَه.
 */
open class BranchViewModel(app: Application, rep: Boolean) : AndroidViewModel(app) {

    private val api = BranchApi(AppCore.get().api, rep)

    var items by mutableStateOf<List<BranchRequest>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set

    /** **أُرسل الطلبُ للتوّ** — تُعرض رسالةُ التأكيد. */
    var sent by mutableStateOf(false)
        private set

    var name by mutableStateOf("")
    var phone by mutableStateOf("")
    var lat by mutableStateOf<Double?>(null)
        private set
    var lng by mutableStateOf<Double?>(null)
        private set

    fun setPoint(la: Double, ln: Double) {
        lat = la
        lng = ln
    }

    fun load(parentId: String) {
        viewModelScope.launch {
            loading = true
            runCatching { items = api.list(parentId) }
                .onSuccess { error = "" }
                .onFailure { error = err(it) }
            loading = false
        }
    }

    fun submit(parentId: String, needName: String, needPoint: String) {
        if (busy) return
        val la = lat
        val ln = lng
        error = when {
            name.isBlank() -> needName
            la == null || ln == null -> needPoint
            else -> ""
        }
        if (error.isNotEmpty() || la == null || ln == null) return
        busy = true
        sent = false
        viewModelScope.launch {
            runCatching {
                api.create(
                    parentId,
                    BranchRequestInput(
                        name = name.trim(),
                        lat = la,
                        lng = ln,
                        phone = phone.trim().ifEmpty { null },
                    ),
                )
            }.onSuccess {
                name = ""
                phone = ""
                lat = null
                lng = null
                sent = true
                runCatching { items = api.list(parentId) }
            }.onFailure { error = err(it) }
            busy = false
        }
    }
}

class MerchantBranchViewModel(app: Application) : BranchViewModel(app, rep = false)

class RepBranchViewModel(app: Application) : BranchViewModel(app, rep = true)

@Composable
fun BranchRequestScreen(
    vm: BranchViewModel,
    parentId: String,
    onBack: () -> Unit,
    /** **بطاقةُ الخريطة** من التطبيق: (خط العرض، خط الطول، مفعَّلة، عند الاختيار). */
    pointCard: @Composable (Double?, Double?, Boolean, (Double, Double) -> Unit) -> Unit,
) {
    LaunchedEffect(parentId) { vm.load(parentId) }
    val needName = stringResource(R.string.branch_need_name)
    val needPoint = stringResource(R.string.branch_need_point)

    Column(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Rahal.space.gutter),
        ) {
            PageHeader(title = stringResource(R.string.branch_title), onBack = onBack)
            Spacer(Modifier.height(Rahal.space.md))

            RahalCard {
                LabeledField(
                    label = stringResource(R.string.branch_name),
                    value = vm.name,
                    onValueChange = { vm.name = it },
                    enabled = !vm.busy,
                    leadingIcon = R.drawable.ic_store,
                )
                Spacer(Modifier.height(Rahal.space.sm))
                PhoneField(
                    value = vm.phone,
                    onChange = { vm.phone = it },
                    enabled = !vm.busy,
                    label = R.string.branch_phone,
                )
                SectionHeader(stringResource(R.string.branch_point))
                pointCard(vm.lat, vm.lng, !vm.busy) { la, ln -> vm.setPoint(la, ln) }
            }

            if (vm.error.isNotEmpty()) {
                Spacer(Modifier.height(Rahal.space.md))
                Note(vm.error, Rahal.colors.danger)
            }
            if (vm.sent) {
                Spacer(Modifier.height(Rahal.space.md))
                Note(stringResource(R.string.branch_sent), Rahal.colors.success)
            }

            SectionHeader(stringResource(R.string.branch_mine))
            if (vm.items.isEmpty() && !vm.loading) {
                EmptyState(title = stringResource(R.string.branch_none))
            }
            Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                vm.items.forEach { BranchRow(it) }
            }
            Spacer(Modifier.height(Rahal.space.tail))
        }
        BottomActionBar {
            RahalButton(
                onClick = { vm.submit(parentId, needName, needPoint) },
                enabled = !vm.busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.branch_send)) }
        }
    }
}

@Composable
private fun BranchRow(r: BranchRequest) {
    val (label, tone) = when (r.status) {
        "approved" -> stringResource(R.string.branch_st_approved) to StatusTone.Success
        "rejected" -> stringResource(R.string.branch_st_rejected) to StatusTone.Danger
        else -> stringResource(R.string.branch_st_pending) to StatusTone.Warning
    }
    RahalCard(variant = CardVariant.Outlined, modifier = Modifier.fillMaxWidth()) {
        Text(r.name, style = Rahal.type.body, color = Rahal.colors.ink)
        Text(
            stringResource(R.string.branch_of, r.parentName),
            style = Rahal.type.label,
            color = Rahal.colors.inkMuted,
        )
        Spacer(Modifier.height(Rahal.space.xs))
        StatusChip(label, tone)
        if (r.status == "rejected" && !r.rejectReason.isNullOrBlank()) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(R.string.branch_reason, r.rejectReason!!),
                style = Rahal.type.label,
                color = Rahal.colors.inkMuted,
            )
        }
    }
}
