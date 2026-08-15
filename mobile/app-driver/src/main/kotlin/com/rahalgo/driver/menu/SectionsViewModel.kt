package com.rahalgo.driver.menu

import android.app.Application
import com.rahalgo.ui.apiError
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.ui.Refresh
import com.rahalgo.shared.model.CashPage
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.Reputation
import com.rahalgo.shared.net.ApiClient
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ القائمة — عقلٌ واحدٌ لها كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ بملء الأقسام من الويب» · «ركّز جيّداً
 *  على المركزيّة بكلّ شيء».)
 *
 * # لماذا واحدٌ لا سبعة
 *
 * **سبعةُ نماذجَ لسبعة أقسامٍ تعني سبعَ نسخٍ من الشيء نفسه**: حالُ
 * تحميلٍ، وحالُ فشلٍ، وترجمةُ رمز الخطأ إلى عربيّة. **ومن أصلح واحدةً
 * ترك ستّا.**
 *
 * **ولا يُجلب إلّا ما فُتح**: من فتح «صندوقي» لا تُنادى دردشاتُه ولا
 * مكافآتُه — **وسبعةُ نداءاتٍ عند كلّ فتحةٍ تستنزف حزمةَ سائقٍ في
 * الشارع.**
 *
 * # وما لا يُعاد جلبُه
 *
 * **نصوصُ الصفحات تُجلب مرّةً وتبقى**: شروطُ الاستخدام لا تتبدّل بين
 * فتحتين، **ونداءٌ عند كلّ فتحةٍ لوثيقةٍ ثابتةٍ عملٌ بلا فائدة.**
 */
class SectionsViewModel(app: Application) : AndroidViewModel(app) {

    private val backend = Backend.of(getApplication())

    init {
        // ══════════════════════════════════════════════════════════════
        // **وما يتبدّل يُبطَل — لا يبقى محفوظاً إلى الأبد**
        // ══════════════════════════════════════════════════════════════
        //
        // **من سلّم طلباً ثمّ فتح صندوقَه وجده كما كان** — والحركةُ وقعت.
        // **ورقمٌ في المال يُعرض قديماً أسوأُ من رقمٍ لا يُعرض.**
        //
        // **ولا يُعاد الجلبُ هنا** — يُمحى المحفوظُ فحسب، **فيُجلب حين
        // يُفتح القسم** لا وهو مطويّ: سائقٌ في رحلته لا يُنادى له سبعةُ
        // نداءات.
        //
        // **ونصوصُ الصفحات تبقى** — في عقلها هي، ولا تتبدّل بتسليم طلب.
        viewModelScope.launch {
            Refresh.tick.drop(1).collect {
                cash = null
                me = null
                reputation = null
            }
        }
    }

    /** **البندُ المعروضُ الآن** — ومنه يُعرف ما يُجلب. */
    var busy by mutableStateOf(false)
        private set

    /** **خطأُ آخر نداء** — بعربيّةٍ تُقرأ لا برمزٍ إنكليزيّ. */
    var error by mutableStateOf("")
        private set

    var cash by mutableStateOf<CashPage?>(null)
        private set

    /** **سقفُ صندوقه وما فيه** — من `driver/me`، وهو ما يقرؤه الويب. */
    var me by mutableStateOf<DriverMe?>(null)
        private set

    var reputation by mutableStateOf<Reputation?>(null)
        private set

    /**
     * **يجلب ما يلزم هذا البندَ وحدَه.**
     *
     * **ولا يُعاد ما وصل** — إلّا أن يُطلب صراحةً (`force`) من زرّ
     * «أعد المحاولة».
     */
    fun open(item: MenuItem, force: Boolean = false) {
        error = ""
        when (item) {
            MenuItem.Cash -> if (force || cash == null) load { cash = backend.driver.cash(); me = backend.driver.me() }
            MenuItem.Tickets -> if (force || reputation == null) load { reputation = backend.me.reputation() }
            // **والدردشاتُ والسجلُّ لهما عقلُهما** — رُفعت الدردشاتُ
            // إلى `:ui` (٢٠٢٦-٠٨-١٤) **لأنّ كلَّ دورٍ يدردش.**
            // **والأهدافُ والدردشاتُ والسجلُّ لها عقولُها** — رُفعت
            // الأهدافُ إلى `:ui` (٢٠٢٦-٠٨-١٤): **السائقُ له هدفٌ
            // والمندوبُ له هدف**، والمقياسُ يختلف والمعنى واحد.
            MenuItem.Rewards, MenuItem.Chats, MenuItem.History -> Unit
        }
    }

    private fun load(block: suspend () -> Unit) {
        busy = true
        viewModelScope.launch {
            try {
                block()
                error = ""
            } catch (e: Exception) {
                error = describe(e)
            }
            busy = false
        }
    }

    /**
     * **رمزُ الخطأ بعربيّة** — ومن رأى `unauthorized` ظنّ التطبيق معطوبا.
     *
     * **ومجهولُه يُعرض برمزه** لا يُبتلع: **من رآه أبلغ عنه**، ومن ابتلعه
     * ترك شاشةً صامتةً لا يُعرف سببُها.
     */
    /** **الرمزُ بعربيّة** — من الخريطة المركزيّة (`data/ApiErrors.kt`). */
    private fun describe(e: Exception): String = apiError(getApplication(), e)
}
