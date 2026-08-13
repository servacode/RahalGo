package com.rahalgo.driver.menu

import android.app.Application
import com.rahalgo.driver.data.apiError
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.driver.R
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.data.Refresh
import com.rahalgo.shared.model.CashPage
import com.rahalgo.shared.model.ChatThread
import com.rahalgo.shared.model.ChatThreadRow
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.IncentivesPayload
import com.rahalgo.shared.model.Platform
import com.rahalgo.shared.model.Reputation
import com.rahalgo.shared.model.SiteContact
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
        // **ونصوصُ الصفحات تبقى** — شروطُ الاستخدام لا تتبدّل بتسليم طلب.
        viewModelScope.launch {
            Refresh.tick.drop(1).collect {
                cash = null
                me = null
                incentives = null
                reputation = null
                chats = null
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

    var chats by mutableStateOf<List<ChatThreadRow>?>(null)
        private set

    /** **حديثٌ مفتوحٌ للقراءة** — ومفتاحُه معرّفُ الطلب. */
    var thread by mutableStateOf<ChatThread?>(null)
        private set

    var openThread by mutableStateOf("")
        private set

    var incentives by mutableStateOf<IncentivesPayload?>(null)
        private set

    var reputation by mutableStateOf<Reputation?>(null)
        private set

    var contact by mutableStateOf<SiteContact?>(null)
        private set

    var platform by mutableStateOf<Platform?>(null)
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
            MenuItem.Chats -> if (force || chats == null) load {
                // **والمنتهيةُ وحدَها** — (قرارُ المالك ٢٠٢٦-٠٨-١٠:
                // «الدردشةُ عندما تُغلق فقط تظهر بالدردشات السابقة»).
                //
                // **وحديثٌ يجري في «السابقة» تناقضٌ في الاسم**: يُفتح من
                // موضعين فيُقرأ مرّتين، **وشارةُ ما لم يُقرأ تنطفئ في
                // أحدهما** فيظنّ صاحبُها أنّه ردّ ولم يفعل.
                chats = backend.chat.threads().threads.filter { !it.open }
            }
            MenuItem.Rewards -> if (force || incentives == null) load { incentives = backend.me.incentives() }
            MenuItem.Tickets -> if (force || reputation == null) load { reputation = backend.me.reputation() }
            MenuItem.Help, MenuItem.About, MenuItem.Terms, MenuItem.Privacy, MenuItem.Contact ->
                if (force || contact == null || platform == null) load {
                    contact = backend.auth.contact()
                    platform = backend.auth.platform()
                }
            MenuItem.History -> Unit
        }
    }

    /** **يفتح حديثاً مطويّاً** — والضغطةُ الثانية تطويه. */
    fun pickThread(orderId: String) {
        if (openThread == orderId) {
            openThread = ""
            thread = null
            return
        }
        openThread = orderId
        thread = null
        load { thread = backend.chat.thread(orderId) }
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
