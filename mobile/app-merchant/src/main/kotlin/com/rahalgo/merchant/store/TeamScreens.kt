package com.rahalgo.merchant.store

import android.app.Application
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
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
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.PhoneField
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Screen
import com.rahalgo.ui.err
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موظّفو المتجر ورقمُ الطلبات حسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **نسخةُ لوحة المتجر بالويب** (`StoreStaffAdmin.tsx`) — نفسُ السلوك:
 * رقمٌ واحدٌ أو صباحيٌّ ومسائيٌّ بساعةِ تبديل، وموظّفٌ يُضاف برقمه ويُحذف بتأكيد،
 * وكلمةُ سرّه تُعرض مرّةً لمن أضافه.
 *
 * **وبالشكل الجديد لتطبيق المتجر** (طلبُ المالك ٢٠٢٦-١٠-٢٩): رأسٌ بسهم رجوع،
 * بطاقاتٌ بيضاءُ مدوّرة، بطاقتا اختيارٍ متساويتان بعلامة صحّ، **وبلا أسطرِ شرح**
 * («وما في داعي للشرح») — عناوينُ وحقولٌ وأزرارٌ وأخطاءُ الحقول وحدَها.
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

    /** **أعُرف الجوابُ بعد؟** — شارةُ الدور في «حسابي» لا تُرسم قبله (٢٠٢٦-١٠-٢٩). */
    var known by mutableStateOf(false)
        private set

    suspend fun probe() {
        runCatching { MerchantApi(AppCore.get().api).staff() }
            .onSuccess { isStaff = false; known = true }
            .onFailure { e ->
                if (e is ApiClient.ApiException && e.body.code == "staff_not_allowed") {
                    isStaff = true
                    known = true
                }
            }
    }
}

// ══════════════════════════════════════════════════════════════════════
// **قِطَعُ الشاشتين**
// ══════════════════════════════════════════════════════════════════════

/** **رأسُ الشاشة** — سهمُ رجوعٍ وعنوان، كرأس صفحات «متجري» (`PageHeader`). */
@Composable
private fun TeamHeader(title: String, onBack: (() -> Unit)?) {
    if (onBack != null) {
        PageHeader(title = title, onBack = onBack)
    } else {
        Text(
            title,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleLarge,
            modifier = Modifier.padding(vertical = Rahal.space.sm),
        )
        Spacer(Modifier.height(Rahal.space.sm))
    }
}

/** **بطاقةٌ بيضاءُ مدوّرةٌ بحدٍّ رفيع** — وعنوانٌ باهتٌ فوقها إن وُجد. */
@Composable
private fun WhiteCard(
    title: String = "",
    content: @Composable ColumnScope.() -> Unit,
) {
    if (title.isNotEmpty()) {
        Text(
            title,
            color = Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall,
            modifier = Modifier.padding(horizontal = Rahal.space.xs),
        )
        Spacer(Modifier.height(Rahal.space.sm))
    }
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg)
            .padding(Rahal.space.lg),
        content = content,
    )
}

/** **خطأُ الحقل تحته** — سطرٌ أحمرُ صغير. */
@Composable
private fun FieldError(text: String?) {
    if (text.isNullOrEmpty()) return
    Text(
        text,
        color = Rahal.colors.danger,
        style = MaterialTheme.typography.bodySmall,
        modifier = Modifier.padding(start = Rahal.space.md, top = Rahal.space.xs),
    )
}

/** **أيقونةٌ قبل نصّ الزرّ.** */
@Composable
private fun ButtonIcon(icon: Int) {
    Icon(painterResource(icon), contentDescription = null, modifier = Modifier.size(20.dp))
    Spacer(Modifier.width(Rahal.space.sm))
}

/**
 * **أيصلح الرقم؟** — فحصٌ أوّليٌّ قبل الإرسال (من ٩ إلى ١٥ رقماً)،
 * **والفصلُ للمحرّك** (`NormalizePhone`).
 */
private fun phoneLooksValid(s: String): Boolean = s.count { it.isDigit() } in 9..15

@Composable
private fun phoneError(value: String, tried: Boolean): String? = when {
    !tried -> null
    value.isBlank() -> stringResource(R.string.team_phone_needed)
    !phoneLooksValid(value) -> stringResource(R.string.team_phone_bad)
    else -> null
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
fun StaffScreen(vm: StaffViewModel, canManage: Boolean, onBack: (() -> Unit)? = null) {
    val title = stringResource(R.string.menu_staff)
    if (vm.loading || !vm.loaded) {
        Screen {
            TeamHeader(title, onBack)
            LoadState(vm.loading, vm.error) { vm.load() }
        }
        return
    }
    var removing by remember { mutableStateOf<StaffMember?>(null) }
    var tried by remember { mutableStateOf(false) }

    Screen {
        TeamHeader(title, onBack)

        // ── القائمة ──
        if (vm.items.isEmpty()) {
            StaffEmpty()
        } else {
            Text(
                stringResource(R.string.staff_list_title),
                color = Rahal.colors.inkMuted,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall,
                modifier = Modifier.padding(horizontal = Rahal.space.xs),
            )
            Spacer(Modifier.height(Rahal.space.sm))
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.lg)
                    .background(Rahal.colors.canvas)
                    .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg),
            ) {
                vm.items.forEachIndexed { i, row ->
                    if (i > 0) HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
                    StaffRow(
                        row = row,
                        canRemove = canManage,
                        enabled = !vm.busy,
                        onRemove = { removing = row },
                    )
                }
            }
        }

        // ── الإضافة ──
        if (canManage) {
            Spacer(Modifier.height(Rahal.space.xl))
            val phoneErr = phoneError(vm.phone, tried)
            WhiteCard(title = stringResource(R.string.staff_new_title)) {
                PhoneField(
                    value = vm.phone,
                    onChange = { vm.phone = it },
                    enabled = !vm.busy,
                    label = R.string.staff_phone,
                )
                FieldError(phoneErr)
                Spacer(Modifier.height(Rahal.space.md))
                OutlinedTextField(
                    value = vm.name,
                    onValueChange = { vm.name = it.take(60) },
                    label = { Text(stringResource(R.string.staff_name)) },
                    leadingIcon = {
                        Icon(painterResource(com.rahalgo.ui.R.drawable.ic_user), contentDescription = null)
                    },
                    singleLine = true,
                    enabled = !vm.busy,
                    shape = Rahal.shape.md,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Rahal.space.lg))
                RahalButton(
                    onClick = {
                        tried = true
                        if (phoneLooksValid(vm.phone)) {
                            vm.add()
                            tried = false
                        }
                    },
                    enabled = !vm.busy,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(52.dp),
                ) {
                    ButtonIcon(com.rahalgo.ui.R.drawable.ic_plus)
                    Text(stringResource(R.string.staff_add), fontWeight = FontWeight.Bold)
                }
            }
        }

        vm.added?.let { (password, sent) ->
            Spacer(Modifier.height(Rahal.space.md))
            Note(
                stringResource(R.string.staff_added, ltr(password)) + " — " +
                    stringResource(if (sent) R.string.staff_sent else R.string.staff_not_sent),
                Rahal.colors.success,
            )
        }
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.md))
            Note(vm.error, Rahal.colors.danger)
        }
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

/** **لا موظّفين بعد** — أيقونةٌ في دائرةٍ وسطرٌ واحد. */
@Composable
private fun StaffEmpty() {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.lg)
            .padding(vertical = Rahal.space.xl, horizontal = Rahal.space.lg),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(56.dp)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.brand.copy(alpha = 0.10f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(com.rahalgo.ui.R.drawable.ic_user),
                contentDescription = null,
                tint = Rahal.colors.brand,
                modifier = Modifier.size(28.dp),
            )
        }
        Spacer(Modifier.height(Rahal.space.md))
        Text(
            stringResource(R.string.staff_none),
            color = Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyLarge,
        )
    }
}

/** **سطرُ موظّف** — دائرةٌ بحرفه الأوّل، اسمُه ورقمُه وآخرُ دخوله، وزرُّ حذفٍ أحمر. */
@Composable
private fun StaffRow(
    row: StaffMember,
    canRemove: Boolean,
    enabled: Boolean,
    onRemove: () -> Unit,
) {
    val context = LocalContext.current
    val initial = row.name.trim().firstOrNull()?.toString()
        ?: row.phone.lastOrNull { it.isDigit() }?.toString()
        ?: "?"
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 72.dp)
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(44.dp)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.brand.copy(alpha = 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                initial,
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )
        }
        Spacer(Modifier.width(Rahal.space.md))
        Column(Modifier.weight(1f)) {
            Text(
                row.name.ifBlank { ltr(row.phone) },
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (row.name.isNotBlank()) {
                Text(
                    ltr(row.phone),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                )
            }
            Text(
                row.lastLoginAt?.let {
                    stringResource(R.string.staff_last_login, Dates.stamp(context, it))
                } ?: stringResource(R.string.staff_never_login),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.labelSmall,
                maxLines = 1,
            )
        }
        if (canRemove) {
            Spacer(Modifier.width(Rahal.space.sm))
            IconButton(
                onClick = onRemove,
                enabled = enabled,
                modifier = Modifier
                    .size(44.dp)
                    .clip(Rahal.shape.pill)
                    .background(Rahal.colors.danger.copy(alpha = 0.10f)),
            ) {
                Icon(
                    painterResource(com.rahalgo.ui.R.drawable.ic_trash),
                    contentDescription = stringResource(R.string.staff_remove),
                    tint = Rahal.colors.danger,
                    modifier = Modifier.size(22.dp),
                )
            }
        }
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
fun OrderPhonesScreen(
    vm: OrderPhonesViewModel,
    storeId: String,
    canManage: Boolean,
    onBack: (() -> Unit)? = null,
) {
    LaunchedEffect(storeId) { vm.load(storeId) }
    val title = stringResource(R.string.menu_order_phones)
    val p = vm.phones
    if (vm.loading || p == null) {
        Screen {
            TeamHeader(title, onBack)
            LoadState(vm.loading, vm.error) { vm.load(storeId) }
        }
        return
    }
    var tried by remember { mutableStateOf(false) }
    val editable = canManage && !vm.busy

    Screen {
        TeamHeader(title, onBack)

        // ── بطاقتا الاختيار ──
        IconChoicePair(
            first = IconChoice(
                icon = com.rahalgo.ui.R.drawable.ic_phone,
                title = stringResource(R.string.order_phones_full),
                sub = "",
                selected = p.mode == "full",
            ) { vm.edit(p.copy(mode = "full")); tried = false },
            second = IconChoice(
                icon = com.rahalgo.ui.R.drawable.ic_time,
                title = stringResource(R.string.order_phones_split),
                sub = "",
                selected = p.mode == "split",
            ) { vm.edit(p.copy(mode = "split")) },
            enabled = editable,
        )

        // ── الرقمان وساعةُ التبديل ──
        if (p.mode == "split") {
            Spacer(Modifier.height(Rahal.space.xl))
            WhiteCard(title = stringResource(R.string.order_phones_numbers_title)) {
                PhoneField(
                    value = p.morning,
                    onChange = { vm.edit(p.copy(morning = it)) },
                    enabled = editable,
                    label = R.string.order_phones_morning,
                )
                FieldError(phoneError(p.morning, tried))
                Spacer(Modifier.height(Rahal.space.md))
                PhoneField(
                    value = p.evening,
                    onChange = { vm.edit(p.copy(evening = it)) },
                    enabled = editable,
                    label = R.string.order_phones_evening,
                )
                FieldError(phoneError(p.evening, tried))
                Spacer(Modifier.height(Rahal.space.lg))
                HorizontalDivider(color = Rahal.colors.line, thickness = Rahal.stroke.hair)
                Spacer(Modifier.height(Rahal.space.md))
                Row(
                    Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painterResource(com.rahalgo.ui.R.drawable.ic_time),
                        contentDescription = null,
                        tint = Rahal.colors.brand,
                        modifier = Modifier.size(20.dp),
                    )
                    Spacer(Modifier.width(Rahal.space.sm))
                    Text(
                        stringResource(R.string.order_phones_evening_from),
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.weight(1f),
                    )
                    Spacer(Modifier.width(Rahal.space.sm))
                    TimeChip(p.eveningFrom, enabled = editable) { vm.edit(p.copy(eveningFrom = it)) }
                }
            }
        }

        if (canManage) {
            Spacer(Modifier.height(Rahal.space.xl))
            RahalButton(
                onClick = {
                    tried = true
                    val ok = p.mode != "split" || (phoneLooksValid(p.morning) && phoneLooksValid(p.evening))
                    if (ok) vm.save()
                },
                enabled = !vm.busy,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
            ) {
                ButtonIcon(R.drawable.ic_check)
                Text(stringResource(R.string.order_phones_save), fontWeight = FontWeight.Bold)
            }
        }
        if (vm.saved) {
            Spacer(Modifier.height(Rahal.space.md))
            Note(stringResource(R.string.order_phones_saved), Rahal.colors.success)
        }
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.md))
            Note(vm.error, Rahal.colors.danger)
        }
    }
}

/**
 * **رقاقةُ الساعة** — «4:00 م» بلون العلامة كرقاقات مخطّط الدوام، تُضغط فتُفتح
 * نافذةُ الوقت من النظام بنظام ١٢ ساعة (ص/م) كما في `TimeField`. **والقيمةُ تُحفظ
 * `HH:MM`** كما يريدها المحرّك.
 */
@Composable
private fun TimeChip(value: String, enabled: Boolean, onChange: (String) -> Unit) {
    val context = LocalContext.current
    val shown = value.take(5).ifBlank { "16:00" }
    val open = {
        val h = shown.substringBefore(':').toIntOrNull()?.coerceIn(0, 23) ?: 16
        val m = shown.substringAfter(':').toIntOrNull()?.coerceIn(0, 59) ?: 0
        val pick = android.app.TimePickerDialog.OnTimeSetListener { _, hh, mm ->
            onChange(String.format(java.util.Locale.ROOT, "%02d:%02d", hh, mm))
        }
        // **وأرقامٌ لاتينيّةٌ في النافذة** كبقيّة الشاشة — وإن تعذّر فبسياقها الأصليّ.
        val themed = runCatching {
            val config = android.content.res.Configuration(context.resources.configuration)
            config.setLocale(java.util.Locale.forLanguageTag("ar-u-nu-latn"))
            android.view.ContextThemeWrapper(context, 0).apply { applyOverrideConfiguration(config) }
        }.getOrDefault(context)
        runCatching { android.app.TimePickerDialog(themed, pick, h, m, false).show() }
            .onFailure { android.app.TimePickerDialog(context, pick, h, m, false).show() }
        Unit
    }
    Row(
        Modifier
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.brand.copy(alpha = 0.08f))
            .border(Rahal.stroke.hair, Rahal.colors.brand.copy(alpha = 0.35f), Rahal.shape.pill)
            .clickable(enabled = enabled, role = Role.Button, onClick = open)
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            clock12(shown),
            color = if (enabled) Rahal.colors.brand else Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyLarge,
        )
    }
}

/** **رقمٌ داخل نصٍّ عربيّ** — معزولٌ يساراً فلا تقفز «+» إلى الطرف الآخر. */
private fun ltr(s: String): String = "⁦$s⁩"
