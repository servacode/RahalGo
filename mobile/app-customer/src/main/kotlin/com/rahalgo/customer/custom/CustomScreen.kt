package com.rahalgo.customer.custom

import android.app.Application
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.NewCustom
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.LastPoint
import com.rahalgo.ui.Note
import com.rahalgo.ui.PointField
import com.rahalgo.ui.PointPicker
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.apiError
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ خاصّ — ما ليس في المنصّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «الزبون يريد أكلاً من مطعمٍ محدَّد أو شيئاً
 *  من سوقٍ غير موجودٍ بالمتجر».)
 *
 * # ولا يُسأل عن سعرٍ ولا متجر
 *
 * **هو يطلب ما لا نعرف سعرَه** — والسائقُ يشتريه ويتّفق معه في المحادثة
 * بعد الإسناد. **وسؤالٌ لا جوابَ له يُوقف من يملأ نموذجا.**
 *
 * # والعنوانُ يُكتب ويُؤشَّر — كما في السلّة حرفاً بحرف
 *
 * **ونموذجان للعنوان يفترقان**: يُشدَّد في أحدهما ويبقى الآخرُ يقبل
 * عنواناً بلا نقطة، **فيُرسل السائقُ إلى لا مكان.**
 */
@Composable
fun CustomScreen(vm: CustomViewModel, picker: PointPicker? = null) {
    var request by rememberSaveable { mutableStateOf("") }
    var address by rememberSaveable { mutableStateOf("") }
    var notes by rememberSaveable { mutableStateOf("") }
    val here = LastPoint.value

    Screen {
        ScreenTitle(
            stringResource(R.string.nav_custom),
            stringResource(R.string.soon_custom),
        )

        if (vm.done.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Note(vm.done, Rahal.colors.success)
        }
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Note(vm.error, Rahal.colors.danger)
        }

        Spacer(Modifier.height(12.dp))
        // **وحقلُ الطلب واسعٌ** — من يكتب «كيلو لحمة من ملحمة أبو أحمد
        // ونصف كيلو خبز» في سطرٍ واحدٍ لا يرى ما كتب.
        OutlinedTextField(
            value = request,
            onValueChange = { request = it },
            label = { Text(stringResource(R.string.cst_what)) },
            // **والمثالُ يقول ما يُطلب** — **وسؤالٌ مفتوحٌ بلا مثالٍ
            // يُوقف من لا يعرف ما تقبله المنصّة.**
            //
            // **وتحت الحقل لا داخلَه**: `placeholder` في Material لا
            // يظهر إلّا بعد أن يُلمس الحقل — **ومن لم يلمسه لم يره
            // فلم يعرف ماذا يكتب**، وهو الموقفُ نفسُه الذي أُصلح.
            supportingText = { Text(stringResource(R.string.cst_what_hint)) },
            minLines = 3,
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(10.dp))
        OutlinedTextField(
            value = address,
            onValueChange = { address = it },
            label = { Text(stringResource(R.string.cst_address)) },
            modifier = Modifier.fillMaxWidth(),
        )

        // ══════════════════════════════════════════════════════════════
        // **والموضعُ يُختار على خريطةٍ لا يُقرأ رقمين**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٥ — ويَنسخ قراره في شاشة الحساب
        //  ٢٠٢٦-٠٨-١٣: «مو ضروريّ تفتح خريطة». **وذاك كان في حفظ
        //  عنوانٍ وهو في بيته، وهذا في طلبٍ إلى مكانٍ قد لا يكون
        //  فيه.**)
        //
        // **وبلا نقطةٍ لا يُرسَل الطلب**: عنوانٌ بإحداثيٍّ صفر نقطةٌ في
        // المحيط الأطلسيّ، **تُرسل سائقاً إلى لا مكان.**
        Spacer(Modifier.height(10.dp))
        PointField(picker)

        Spacer(Modifier.height(10.dp))
        OutlinedTextField(
            value = notes,
            onValueChange = { notes = it },
            label = { Text(stringResource(R.string.cst_notes)) },
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))
        Button(
            onClick = {
                here?.let { vm.send(request.trim(), address.trim(), it.lat, it.lng, notes.trim()) }
            },
            enabled = !vm.busy && request.isNotBlank() && address.isNotBlank() && here != null,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (vm.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(R.string.cst_send))
            }
        }

        // **ويُقال ما ينقص قبل أن يُضغط** — لا زرٌّ معطّلٌ بلا سبب:
        // **من رأى زرّاً لا يعمل ولا يعرف لماذا أغلق التطبيق.**
        if (here == null) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(R.string.cst_need_point),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(28.dp))
    }
}

class CustomViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    var done by mutableStateOf("")
        private set

    /** **يُطلب موضعُه الآن** — لا يُقرأ في الخفاء لحظةَ الإرسال. */
    fun locate() {
        com.rahalgo.customer.Here.refresh(getApplication())
    }

    fun send(request: String, address: String, lat: Double, lng: Double, notes: String) {
        if (busy) return
        busy = true
        error = ""
        done = ""
        viewModelScope.launch {
            try {
                val ref = api.createCustom(NewCustom(request, address, lat, lng, notes))
                done = getApplication<Application>()
                    .getString(R.string.cst_done, ref.code.ifEmpty { ref.id.take(6) })
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}
