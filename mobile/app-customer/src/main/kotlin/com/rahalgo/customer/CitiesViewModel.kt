package com.rahalgo.customer

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.City
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.launch

/**
 * **مدنُ المنصّة — تُجلب مرّةً في عمر التطبيق.**
 *
 * **ولا تتغيّر إلّا حين يفتح المالكُ مدينةً** — **ونداءٌ في كلّ شاشةٍ
 * لقائمةٍ فيها سطرٌ واحدٌ إسرافُ شبكةٍ على هاتفٍ في الرقّة.**
 *
 * **وفشلُها لا يُسقط شيئاً**: القائمةُ تبقى فارغةً، **والفارغةُ تعني
 * «لا تُرشِّح»** لا «لا سوقَ لك». (انظر `CityScope`.)
 */
class CitiesViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    var cities by mutableStateOf<List<City>>(emptyList())
        private set

    init {
        viewModelScope.launch {
            runCatching { api.cities().cities }.onSuccess { cities = it }
        }
    }
}
