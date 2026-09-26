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
import com.rahalgo.shared.rep.Division
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
            value = vm.store,
            onValueChange = { vm.store = it },
            label = { Text(stringResource(R.string.mn_store_name)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.owner,
            onValueChange = { vm.owner = it },
            label = { Text(stringResource(R.string.ac_owner)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        // **ورقمُ الهاتف بالحقل المركزيّ** — **وتحقّقُ الشكل في موضعٍ
        // واحدٍ لا في كلّ نموذج.**
        PhoneField(value = vm.phone, onChange = { vm.phone = it }, enabled = !vm.busy)

        // ══════════════════════════════════════════════════════════════
        // **والمحافظةُ ثمّ المنطقةُ ثمّ العنوانُ التفصيليّ**
        // ══════════════════════════════════════════════════════════════
        //
        // **من العامّ إلى الخاصّ** — **وعكسُه يجعل أوّلَ ما يُملأ أغمضَ
        // ما فيه.**
        //
        // **ورقائقُ لا قائمةٌ منسدلة** — كالتصنيف حرفاً: **المندوبُ واقفٌ
        // في السوق بيدٍ واحدة**، والرقاقةُ تُضغط بالإبهام والمنسدلةُ
        // تحتاج ضغطتين ونافذةً تُغطّي النموذج.
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.ac_governorate),
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
            vm.governorates.forEach { g ->
                FilterChip(
                    selected = vm.pickedGovernorate == g.id,
                    onClick = { vm.pickGovernorate(g.id) },
                    label = { Text(g.name) },
                )
            }
        }

        // **ولا يُرسم لوحُ المناطق قبل أن تُختار محافظة** — **وصفٌّ فارغٌ
        // يُعلّم صاحبَه ألّا ينظر إليه**، ثمّ لا ينظر حين يمتلئ.
        if (vm.pickedGovernorate.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Text(
                stringResource(R.string.ac_district),
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
                vm.districts.forEach { d ->
                    FilterChip(
                        selected = vm.pickedDistrict == d.id,
                        onClick = { vm.pickDistrict(d.id) },
                        label = { Text(d.name) },
                    )
                }
            }
        }

        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = vm.area,
            onValueChange = { vm.area = it },
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
            value = vm.password,
            onChange = { vm.password = it },
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
            onClick = { vm.send() },
            // ══════════════════════════════════════════════════════════
            // **وكلُّ حقلٍ إلزاميّ — ولا يُرسَل بنقصِ واحد**
            // ══════════════════════════════════════════════════════════
            //
            // **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «حقول إضافة عميل كلُّها
            // إلزاميّة، ولا يمكن إرسالُ طلبٍ بدون أيّ عنصرٍ منها»).
            //
            // **وكان «العنوان الكامل» خارجَ الشرط** — فيصل الطلبُ
            // الإدارةَ بمنطقةٍ بلا تفصيل، **ومن يوافق عليه لا يعرف أين
            // المتجرُ في منطقته**، والسائقُ يبحث عنه بالنقطة وحدَها.
            //
            // **والمحافظةُ لا تُشترط منفصلةً** — لا تُختار منطقةٌ بلا
            // محافظة، **وشرطٌ لا يمكن كسرُه شرطٌ يزحم ولا يحرس.**
            enabled = !vm.busy && vm.store.isNotBlank() && vm.owner.isNotBlank() &&
                vm.phone.isNotBlank() && vm.pickedCategory.isNotEmpty() &&
                vm.pickedDistrict.isNotEmpty() && vm.area.isNotBlank() &&
                vm.password.isNotBlank() && vm.point != null,
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

    // ══════════════════════════════════════════════════════════════════
    // **وحقولُ النصّ هنا لا في الشاشة**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كانت `rememberSaveable` في `AddClientScreen`** — **فتُمحى كلَّما
    // فُتحت الخريطة.**
    //
    // # والسببُ أنّ الخريطةَ لا تغطّي النموذج
    //
    // التعليقُ في `MainActivity` يقول النيّةَ صحيحةً: «**الخريطةُ تغطّي
    // النموذجَ لا تفتح صفحةً ثانية — ومن خرج إلى صفحةٍ ليختار نقطةً عاد
    // فلم يجد ما كتب**». **والتنفيذُ خالفه**: `when` يُخرج
    // `AddClientScreen` من التركيب كلَّه، **و`rememberSaveable` لا ينجو
    // من خروجٍ ليس فيه `SaveableStateHolder`.**
    //
    // # وأثرُه في الميدان
    //
    // **المندوبُ واقفٌ في المتجر**: يكتب اسمَه واسمَ صاحبه ورقمَه، ثمّ
    // يفتح الخريطةَ ليلتقط النقطة — **وهي خطوةٌ يفرضها النموذجُ نفسُه**
    // — فيعود فيجد الثلاثةَ فارغة. **ويكتبها ثانيةً، أو يترك.**
    //
    // **ولا رسالةَ ولا انهيار** — فلا يُبلَّغ عنه، **ويُقرأ على أنّ
    // التطبيق «بطيءٌ في الإدخال».**
    //
    // # ولماذا `ViewModel` لا حاملُ حالة
    //
    // **الأربعةُ الأخرى هنا أصلاً** — النقطةُ والتصنيفُ والمحافظةُ
    // والمنطقة، **ولذلك نجت وحدَها في التجربة.** **وحقلان من نموذجٍ
    // واحدٍ في موضعين يفترقان في السلوك** — وهو ما وقع حرفاً.
    //
    // (كشفه اختبارُ الميدان على الجهاز ٢٠٢٦-٠٨-٣٠.)

    // **وتُكتب من الشاشة مباشرةً** — **و`private set` يولّد `setX`
    // فيصطدم بدالّةٍ بالاسم نفسِه** في الـJVM.
    var store by mutableStateOf("")
    var owner by mutableStateOf("")
    var phone by mutableStateOf("")
    var area by mutableStateOf("")
    var password by mutableStateOf("")

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

    // ══════════════════════════════════════════════════════════════════
    // **والمحافظةُ تُصفّي المنطقة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «اختيارُ المحافظة والمنطقة بفورم تسجيل
    //  متجرٍ جديد… وبهذا سينعكس أيضاً على المندوب».)
    //
    // **وكان نصّاً حرّاً** — «وسط المدينة» و«وسط البلد» و«المركز» ثلاثةُ
    // نصوصٍ لموضعٍ واحد، **لا تُصنَّف ولا تُصفّى.**

    var governorates by mutableStateOf<List<Division>>(emptyList())
        private set

    var districts by mutableStateOf<List<Division>>(emptyList())
        private set

    var pickedGovernorate by mutableStateOf("")
        private set

    var pickedDistrict by mutableStateOf("")
        private set

    fun loadCategories() {
        if (categories.isNotEmpty()) return
        viewModelScope.launch {
            runCatching { categories = api.categories() }
                .onFailure { error = apiError(getApplication(), it as Exception) }
            // **والمحافظاتُ في النداء نفسِه** — **وسقوطُها لا يُخفي
            // النموذج**: يبقى يملأ ما عداها ثمّ يُعيد.
            runCatching { governorates = api.governorates() }
        }
    }

    fun pickCategory(id: String) {
        pickedCategory = id
    }

    /**
     * **يختار محافظةً ويجلب مناطقَها.**
     *
     * **وتبديلُ المحافظة يمسح المنطقة** — وإلّا بقيت منطقةُ حلبَ مختارةً
     * تحت الرقّة، **فيُرسَل معرّفٌ لا ينتمي إلى ما يراه صاحبُه.**
     */
    fun pickGovernorate(id: String) {
        if (pickedGovernorate == id) return
        pickedGovernorate = id
        pickedDistrict = ""
        districts = emptyList()
        if (id.isEmpty()) return
        viewModelScope.launch {
            runCatching { districts = api.districts(id) }
                .onFailure { error = apiError(getApplication(), it as Exception) }
        }
    }

    fun pickDistrict(id: String) {
        pickedDistrict = id
    }

    fun setPoint(lat: Double, lng: Double, label: String) {
        point = lat to lng
        pointLabel = label.ifEmpty { "%.5f، %.5f".format(lat, lng) }
    }

    /**
     * **يرسل الطلبَ ويُفرّغ النموذجَ بعد أن يُقيَّد.**
     *
     * **ولا يأخذ رسماً ولا ردّاً** — الحقولُ كلُّها هنا، **وتفريغُها في
     * موضعٍ واحدٍ لا في ردٍّ تكتبه الشاشة**: من نسي حقلاً في الردّ ترك
     * ما كُتب معلّقاً في نموذجٍ يبدو فارغاً.
     */
    fun send() {
        if (busy) return
        busy = true
        error = ""
        done = false
        // **ومفتاحٌ ثابتٌ للمحاولة** (`Attempt.LEAD`) — يُولَّد مرّةً ويبقى على
        // القرص حتّى تنجح، **فإعادةٌ بعد جوابٍ غامضٍ (انقطاعُ شبكة) تحمله نفسَه**
        // والخادمُ يلفّ المسارَ (`s.idempotent`) فيردّ الطلبَ الأوّلَ ولا يُنشئ
        // ثانياً. **وردُّ الخادم حسمٌ فيُمحى** (خطأُ إدخال)، وانقطاعُ الشبكة ليس
        // حسماً فيبقى ليُعيد المحاولةَ بمفتاحه. والضغطُ المزدوجُ يمنعه `busy` أعلاه.
        val key = com.rahalgo.ui.Attempt.key(com.rahalgo.ui.Attempt.LEAD)
        viewModelScope.launch {
            try {
                api.createLead(
                    NewLead(
                        storeName = store.trim(),
                        ownerName = owner.trim(),
                        phone = phone.trim(),
                        area = area.trim(),
                        districtId = pickedDistrict,
                        categoryId = pickedCategory,
                        password = password,
                        lat = point?.first,
                        lng = point?.second,
                    ),
                    idempotencyKey = key,
                )
                com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.LEAD)
                done = true
                // **والنموذجُ يُفرَّغ بعد أن يُقيَّد لا قبله** — **ومن
                // فرّغه قبل الجواب خسر ما كتبه إن سقط النداء.**
                store = ""
                owner = ""
                phone = ""
                area = ""
                password = ""
                point = null
                pointLabel = ""
                pickedCategory = ""
                pickedGovernorate = ""
                pickedDistrict = ""
                districts = emptyList()
            } catch (e: Exception) {
                // **وردُّ الخادم حسمٌ فيُمحى المفتاح** — محاولةٌ جديدةٌ عن قصد.
                // **وانقطاعُ الشبكة ليس حسماً فيبقى** — إعادةٌ آمنةٌ بالمفتاح نفسِه.
                if (com.rahalgo.ui.isDecided(e)) {
                    com.rahalgo.ui.Attempt.clear(com.rahalgo.ui.Attempt.LEAD)
                }
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}
