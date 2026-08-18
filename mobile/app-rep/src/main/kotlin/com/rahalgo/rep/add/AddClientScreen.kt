package com.rahalgo.rep.add

import android.app.Application
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.NewLead
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.shared.rep.RepCategory
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Note
import com.rahalgo.ui.PasswordField
import com.rahalgo.ui.PhoneField
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.apiError
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إضافةُ عميل — من الميدان**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: تبويبٌ رابعٌ بين لوحتي وعملائي.)
 *
 * # ولا يصير متجراً بضغطة
 *
 * **يبقى معلّقاً حتّى موافقة الإدارة** — **ومن فتح هذا البابَ بلا
 * مراجعةٍ فتح بابَ من يُسجّل متاجرَ وهميّةً ليأخذ عمولتَها.**
 *
 * # والنقطةُ تُلتقط وهو داخل المتجر
 *
 * **«التقط الموقعَ وأنت داخل المتجر ليصل السائقُ بدقّة»** (نصُّ الويب) —
 * **ومتجرٌ بنقطةٍ خاطئةٍ يُرسل كلَّ سائقٍ إلى الشارع الخطأ**، لا مرّةً
 * واحدة.
 *
 * # وكلمةُ المرور تُسلَّم لصاحبه
 *
 * **ولا تبقى عند المندوب** — **ومن احتفظ بها دخل حسابَ متجرٍ ليس له.**
 */
@Composable
fun AddClientScreen(vm: AddClientViewModel, pick: () -> Unit) {
    var store by rememberSaveable { mutableStateOf("") }
    var owner by rememberSaveable { mutableStateOf("") }
    var phone by rememberSaveable { mutableStateOf("") }
    var area by rememberSaveable { mutableStateOf("") }
    var password by rememberSaveable { mutableStateOf("") }

    LaunchedEffect(Unit) { vm.loadCategories() }

    Screen {
        ScreenTitle(
            stringResource(R.string.nav_add_client),
            stringResource(R.string.ac_title_hint),
        )

        if (vm.done) {
            Spacer(Modifier.height(10.dp))
            // **ويقول ماذا يقع بعده** — **و«تمّ» وحدَها تترك صاحبَها
            // ينتظر شيئاً لا يعرف متى يجيء.**
            Note(stringResource(R.string.ac_done), Rahal.colors.success)
        }
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Note(vm.error, Rahal.colors.danger)
        }

        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = store,
            onValueChange = { store = it },
            label = { Text(stringResource(R.string.ac_store)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = owner,
            onValueChange = { owner = it },
            label = { Text(stringResource(R.string.ac_owner)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        // **ورقمُ الهاتف بالحقل المركزيّ** — **وتحقّقُ الشكل في موضعٍ
        // واحدٍ لا في كلّ نموذج.**
        PhoneField(value = phone, onChange = { phone = it }, enabled = !vm.busy)

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = area,
            onValueChange = { area = it },
            label = { Text(stringResource(R.string.ac_area)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )

        // ══════════════════════════════════════════════════════════════
        // **والتصنيفُ يُختار من قائمة المحرّك لا يُكتب**
        // ══════════════════════════════════════════════════════════════
        //
        // **ونصٌّ حرٌّ يجعل «مطاعم» و«مطعم» و«مطاعم ووجبات» ثلاثةَ
        // تصنيفات** — فلا يُعثر على المتجر في بابه.
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.ac_category),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
        )
        Spacer(Modifier.height(6.dp))
        Row(
            Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            vm.categories.forEach { c ->
                FilterChip(
                    selected = vm.pickedCategory == c.id,
                    onClick = { vm.pickCategory(c.id) },
                    label = { Text(c.name) },
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والنقطةُ على الخريطة**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = vm.pointLabel,
            onValueChange = {},
            readOnly = true,
            label = { Text(stringResource(R.string.ac_point)) },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(6.dp))
        RahalOutlineButton(onClick = pick, modifier = Modifier.fillMaxWidth()) {
            Text(
                stringResource(
                    if (vm.point == null) R.string.ac_pick else R.string.ac_repick,
                ),
            )
        }
        Spacer(Modifier.height(4.dp))
        Text(
            stringResource(R.string.ac_point_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )

        // ══════════════════════════════════════════════════════════════
        // **وكلمةُ مرورِ صاحب المتجر**
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(12.dp))
        PasswordField(
            value = password,
            onChange = { password = it },
            enabled = !vm.busy,
            label = R.string.ac_password,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            stringResource(R.string.ac_password_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )

        Spacer(Modifier.height(16.dp))
        RahalButton(
            onClick = {
                vm.send(
                    NewLead(
                        storeName = store.trim(),
                        ownerName = owner.trim(),
                        phone = phone.trim(),
                        area = area.trim(),
                        categoryId = vm.pickedCategory,
                        password = password,
                        lat = vm.point?.first,
                        lng = vm.point?.second,
                    ),
                ) {
                    store = ""; owner = ""; phone = ""; area = ""; password = ""
                }
            },
            enabled = !vm.busy && store.isNotBlank() && owner.isNotBlank() &&
                phone.isNotBlank() && vm.pickedCategory.isNotEmpty() &&
                password.isNotBlank() && vm.point != null,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (vm.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(R.string.ac_send))
            }
        }

        // **ويُقال ما ينقص قبل أن يُضغط** — **ومن رأى زرّاً لا يعمل ولا
        // يعرف لماذا أغلق التطبيق.**
        if (vm.point == null) {
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.ac_need_point),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(28.dp))
    }
}

class AddClientViewModel(app: Application) : AndroidViewModel(app) {

    private val api = RepApi(AppCore.get().api)

    var categories by mutableStateOf<List<RepCategory>>(emptyList())
        private set

    var pickedCategory by mutableStateOf("")
        private set

    /** **نقطةُ المتجر واسمُها** — تُلتقط من الخريطة لا من موضعه هو. */
    var point by mutableStateOf<Pair<Double, Double>?>(null)
        private set

    var pointLabel by mutableStateOf("")
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    var done by mutableStateOf(false)
        private set

    fun loadCategories() {
        if (categories.isNotEmpty()) return
        viewModelScope.launch {
            runCatching { categories = api.categories() }
                .onFailure { error = apiError(getApplication(), it as Exception) }
        }
    }

    fun pickCategory(id: String) {
        pickedCategory = id
    }

    fun setPoint(lat: Double, lng: Double, label: String) {
        point = lat to lng
        pointLabel = label.ifEmpty { "%.5f، %.5f".format(lat, lng) }
    }

    fun send(lead: NewLead, onSent: () -> Unit) {
        if (busy) return
        busy = true
        error = ""
        done = false
        viewModelScope.launch {
            try {
                api.createLead(lead)
                done = true
                // **والنموذجُ يُفرَّغ بعد أن يُقيَّد لا قبله** — **ومن
                // فرّغه قبل الجواب خسر ما كتبه إن سقط النداء.**
                point = null
                pointLabel = ""
                pickedCategory = ""
                onSent()
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}
