package com.rahalgo.merchant.store

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.Dates
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.merchant.OrderPhones
import com.rahalgo.shared.merchant.StaffMember
import com.rahalgo.shared.net.ApiClient
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.PhoneField
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.Tone
import com.rahalgo.ui.err
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موظّفو المتجر ورقمُ الطلبات حسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **نسخةُ لوحة المتجر بالويب** (`StoreStaffAdmin.tsx`) — نفسُ النصوص ونفسُ السلوك:
 * رقمٌ واحدٌ أو صباحيٌّ ومسائيٌّ بساعةِ تبديل، وموظّفٌ يُضاف برقمه ويُحذف بتأكيد،
 * وكلمةُ سرّه تُعرض مرّةً لمن أضافه.
 *
 * **والموظّفُ لا يرى بابَ الموظّفين** — المحرّكُ يردّه `staff_not_allowed`
 * (`notStaff`)، **والتطبيقُ يسأله مرّةً (`TeamAccess`) فيُخفي البندَ من الدرج.**
 * **والمتجرُ الموقوفُ يقرأ ولا يعدّل** — كما في الويب (`canManage = !suspended`).
 */

// ══════════════════════════════════════════════════════════════════════
// **أهذا الحسابُ موظّف؟** — يُعرف من ردّ المحرّك لا من الشاشة
// ══════════════════════════════════════════════════════════════════════

object TeamAccess {
    /** **`true` إن ردّ المحرّكُ `staff_not_allowed`** — والافتراضُ إظهارُ البند. */
    var isStaff by mutableStateOf(false)
        private set

    suspend fun probe() {
        runCatching { MerchantApi(AppCore.get().api).staff() }
            .onSuccess { isStaff = false }
            .onFailure { e ->
                if (e is ApiClient.ApiException && e.body.code == "staff_not_allowed") isStaff = true
            }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **١ · موظّفو المتجر**
// ══════════════════════════════════════════════════════════════════════

class StaffViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var items by mutableStateOf<List<StaffMember>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set

    /** **وصلت القائمةُ مرّةً** — وقبلها الخطأُ يُعرض مع «أعد المحاولة» لا مع النموذج. */
    var loaded by mutableStateOf(false)
        private set

    /** **كلمةُ سرّ الموظّف الجديد وأوصلت الواتساب أم لا** — تُعرض مرّةً. */
    var added by mutableStateOf<Pair<String, Boolean>?>(null)
        private set

    var phone by mutableStateOf("")
    var name by mutableStateOf("")

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            runCatching { items = api.staff() }
                .onSuccess { error = ""; loaded = true }
                .onFailure { error = err(it) }
            loading = false
        }
    }

    fun add() {
        if (busy || phone.isBlank()) return
        busy = true
        error = ""
        added = null
        viewModelScope.launch {
            runCatching { api.addStaff(phone.trim(), name.trim()) }
                .onSuccess {
                    added = it.password to it.sent
                    phone = ""
                    name = ""
                    runCatching { items = api.staff() }
                }
                .onFailure { error = err(it) }
            busy = false
        }
    }

    fun remove(row: StaffMember) {
        if (busy) return
        busy = true
        error = ""
        viewModelScope.launch {
            runCatching { api.removeStaff(row.id) }
                .onSuccess { runCatching { items = api.staff() } }
                .onFailure { error = err(it) }
            busy = false
        }
    }
}

@Composable
fun StaffScreen(vm: StaffViewModel, canManage: Boolean) {
    val title = stringResource(R.string.menu_staff)
    if (vm.loading || !vm.loaded) {
        Screen {
            ScreenTitle(title, "")
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }
    var removing by remember { mutableStateOf<StaffMember?>(null) }
    val context = LocalContext.current

    Screen {
        ScreenTitle(title, stringResource(R.string.staff_hint))

        Card {
            if (vm.items.isEmpty()) {
                Text(
                    stringResource(R.string.staff_none),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            vm.items.forEachIndexed { i, row ->
                if (i > 0) HorizontalDivider()
                Row(
                    Modifier.fillMaxWidth().padding(vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(row.name.ifBlank { ltr(row.phone) }, fontWeight = FontWeight.Bold)
                        Text(
                            ltr(row.phone),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodySmall,
                        )
                        Text(
                            row.lastLoginAt?.let {
                                stringResource(R.string.staff_last_login, Dates.stamp(context, it))
                            } ?: stringResource(R.string.staff_never_login),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.labelSmall,
                        )
                    }
                    if (canManage) {
                        RahalTextButton(
                            onClick = { removing = row },
                            enabled = !vm.busy,
                            tone = Tone.Danger,
                        ) { Text(stringResource(R.string.staff_remove)) }
                    }
                }
            }
        }

        if (canManage) {
            Spacer(Modifier.height(14.dp))
            PhoneField(
                value = vm.phone,
                onChange = { vm.phone = it },
                enabled = !vm.busy,
                label = R.string.staff_phone,
            )
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(
                value = vm.name,
                onValueChange = { vm.name = it },
                label = { Text(stringResource(R.string.staff_name)) },
                singleLine = true,
                enabled = !vm.busy,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(10.dp))
            RahalButton(
                onClick = { vm.add() },
                enabled = !vm.busy && vm.phone.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.staff_add)) }
            Spacer(Modifier.height(10.dp))
        }

        vm.added?.let { (password, sent) ->
            Note(
                stringResource(R.string.staff_added, ltr(password)) + " — " +
                    stringResource(if (sent) R.string.staff_sent else R.string.staff_not_sent),
                Rahal.colors.success,
            )
        }
        if (vm.error.isNotEmpty()) Note(vm.error, Rahal.colors.danger)
    }

    removing?.let { row ->
        ConfirmDialog(
            title = stringResource(R.string.staff_remove_confirm, row.name.ifBlank { ltr(row.phone) }),
            body = "",
            confirm = stringResource(R.string.staff_remove),
            onConfirm = { vm.remove(row) },
            onDismiss = { removing = null },
        )
    }
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · رقمُ الطلبات حسب الدوام**
// ══════════════════════════════════════════════════════════════════════

class OrderPhonesViewModel(app: Application) : AndroidViewModel(app) {

    private val api = MerchantApi(AppCore.get().api)

    var phones by mutableStateOf<OrderPhones?>(null)
    var loading by mutableStateOf(true)
        private set
    var busy by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var saved by mutableStateOf(false)
        private set

    private var storeId = ""

    fun load(id: String) {
        if (id.isBlank()) return
        storeId = id
        loading = true
        viewModelScope.launch {
            runCatching { phones = api.orderPhones(id) }
                .onSuccess { error = "" }
                .onFailure { error = err(it) }
            loading = false
        }
    }

    fun edit(p: OrderPhones) {
        phones = p
        saved = false
    }

    fun save() {
        val p = phones ?: return
        if (busy || storeId.isBlank()) return
        busy = true
        error = ""
        saved = false
        viewModelScope.launch {
            runCatching { api.setOrderPhones(storeId, p) }
                .onSuccess { phones = it; saved = true }
                .onFailure { error = err(it) }
            busy = false
        }
    }
}

@Composable
fun OrderPhonesScreen(vm: OrderPhonesViewModel, storeId: String, canManage: Boolean) {
    LaunchedEffect(storeId) { vm.load(storeId) }
    val title = stringResource(R.string.menu_order_phones)
    val p = vm.phones
    if (vm.loading || p == null) {
        Screen {
            ScreenTitle(title, "")
            LoadState(vm.loading, vm.error) { vm.load(storeId) }
        }
        return
    }

    Screen {
        ScreenTitle(title, stringResource(R.string.order_phones_hint))

        listOf(
            "full" to R.string.order_phones_full,
            "split" to R.string.order_phones_split,
        ).forEach { (mode, label) ->
            // **المختارُ ممتلئٌ والآخرُ مؤطَّر** — كأسئلة «متجري» (`BigChoice`).
            if (p.mode == mode) {
                RahalButton(
                    onClick = { vm.edit(p.copy(mode = mode)) },
                    enabled = canManage && !vm.busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(label), fontWeight = FontWeight.Bold) }
            } else {
                RahalOutlineButton(
                    onClick = { vm.edit(p.copy(mode = mode)) },
                    enabled = canManage && !vm.busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(label)) }
            }
            Spacer(Modifier.height(8.dp))
        }

        if (p.mode == "split") {
            Spacer(Modifier.height(6.dp))
            PhoneField(
                value = p.morning,
                onChange = { vm.edit(p.copy(morning = it)) },
                enabled = canManage && !vm.busy,
                label = R.string.order_phones_morning,
            )
            Spacer(Modifier.height(8.dp))
            PhoneField(
                value = p.evening,
                onChange = { vm.edit(p.copy(evening = it)) },
                enabled = canManage && !vm.busy,
                label = R.string.order_phones_evening,
            )
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(stringResource(R.string.order_phones_evening_from), style = MaterialTheme.typography.bodyMedium)
                if (canManage) {
                    TimeField(p.eveningFrom) { vm.edit(p.copy(eveningFrom = it)) }
                } else {
                    Text(clock12(p.eveningFrom), style = MaterialTheme.typography.bodyMedium)
                }
            }
        }

        if (canManage) {
            Spacer(Modifier.height(14.dp))
            RahalButton(
                onClick = { vm.save() },
                enabled = !vm.busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.order_phones_save)) }
            Spacer(Modifier.height(10.dp))
        }
        if (vm.saved) Note(stringResource(R.string.order_phones_saved), Rahal.colors.success)
        if (vm.error.isNotEmpty()) Note(vm.error, Rahal.colors.danger)
    }
}

/** **رقمٌ داخل نصٍّ عربيّ** — معزولٌ يساراً فلا تقفز «+» إلى الطرف الآخر. */
private fun ltr(s: String): String = "\u2066$s\u2069"
