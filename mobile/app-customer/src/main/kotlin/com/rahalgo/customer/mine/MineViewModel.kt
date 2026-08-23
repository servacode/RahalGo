package com.rahalgo.customer.mine

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.customer.CustomerItems
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.Ticket
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Offer
import com.rahalgo.shared.model.Referral
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Refresh
import com.rahalgo.ui.apiError
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ «ما يخصّني» — عقلٌ واحدٌ لها كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (وهو عينُ ما فُعل في تطبيق السائق `SectionsViewModel`.)
 *
 * # لماذا واحدٌ لا خمسة
 *
 * **خمسةُ نماذجَ لخمسة أقسامٍ تعني خمسَ نسخٍ من الشيء نفسه**: حالُ
 * تحميلٍ، وحالُ فشلٍ، وترجمةُ رمز الخطأ إلى عربيّة. **ومن أصلح واحدةً
 * ترك أربعا.**
 *
 * # ولا يُجلب إلّا ما فُتح
 *
 * **من فتح مفضّلتَه لا تُنادى شكاواه ولا دعوتُه** — **وخمسةُ نداءاتٍ
 * عند كلّ فتحةٍ تستنزف حزمةَ من يتصفّح.**
 */
class MineViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    /** **تقرؤه نافذةُ الخيارات** — انظر `ItemOptionsSheet`. */
    val customerApi: CustomerApi get() = api

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    var favorites by mutableStateOf<List<Item>?>(null)
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ما أعجبه — مجموعةُ معرّفاتٍ لا قائمةُ أصناف**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وشاشةُ التسوّق تسأل عن كلّ صنفٍ «أهذا في مفضّلته؟»** — ومن
     * قرأها من `favorites` قرأ `null` **حتّى يفتح شاشةَ المفضّلة.**
     *
     * **فكان القلبُ لا يمتلئ أبدا** (شكوى المالك ٢٠٢٦-٠٨-١٥: «زرّ
     * المفضّلة لا يعمل») — **والنداءُ كان يقع والخادمُ يُقيّده**،
     * لكنّ الشاشةَ لا ترى أثرَه: **فيُقرأ الزرُّ ميّتا.**
     *
     * **وتُجلب عند الإقلاع لا عند فتح شاشتها** — لأنّ من يسأل عنها
     * هو السوقُ، وهو أوّلُ ما يُفتح.
     */
    var liked by mutableStateOf<Set<String>>(emptySet())
        private set

    init {
        // **وسقوطُه لا يُسقط شيئا**: الضيفُ يُردّ ٤٠١ بلا نداءٍ أصلا
        // (`ApiClient` يحرسه)، **وقلوبٌ فارغةٌ حالٌ صحيحةٌ لمن لا
        // حسابَ له.**
        viewModelScope.launch {
            runCatching { api.favorites().items }.onSuccess {
                favorites = it
                liked = it.map { item -> item.id }.toSet()
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والعروضُ تتبع اللوحةَ لحظةً بلحظة**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٨: «أيُّ تعديلٍ من لوحة الأدمن فوراً
        //  يُطبَّق حتّى ولو الزبونُ فاتحٌ التطبيق».)
        //
        // **والعرضُ يُكتب في اللوحة وينتهي فيها** — **وعرضٌ منتهٍ يبقى
        // معروضاً وعدٌ لا يُوفى**: يفتحه الزبونُ فيُردّ كودُه.
        //
        // **ولا يُجلب ما لم يُفتح بعد**: `null` تعني «لم تُطلَب»،
        // **ونداءٌ هنا لكلّ نبضةٍ يجلب شاشةً لم يزرها صاحبُها.**
        viewModelScope.launch {
            Refresh.tick.drop(1).collect {
                if (favorites != null) open(CustomerItems.FAVORITES, force = true)
                if (offers != null) open(CustomerItems.OFFERS, force = true)
            }
        }
    }

    var offers by mutableStateOf<List<Offer>?>(null)
        private set

    var referral by mutableStateOf<Referral?>(null)
        private set

    var tickets by mutableStateOf<List<Ticket>?>(null)
        private set

    /** **يجلب ما يلزم هذا البندَ وحدَه** — ولا يُعاد ما وصل إلّا بطلب. */
    fun open(key: String, force: Boolean = false) {
        error = ""
        when (key) {
            CustomerItems.FAVORITES ->
                if (force || favorites == null) load { favorites = api.favorites().items }

            CustomerItems.OFFERS ->
                if (force || offers == null) load { offers = api.offers().offers }

            CustomerItems.INVITE ->
                if (force || referral == null) load { referral = api.referral() }

            CustomerItems.TICKETS ->
                if (force || tickets == null) load { tickets = api.tickets().tickets }
        }
    }

    /** **يقلب المفضّلة** — والشاشتان تريان أثرَه. */
    fun toggleFavorite(itemId: String) {
        viewModelScope.launch {
            runCatching { api.toggleFavorite(itemId) }
                .onSuccess { added ->
                    // **والقلبُ يمتلئ أو يفرغ فورا** — ومن ضغطه فلم
                    // يتغيّر شيءٌ ظنّ ضغطتَه لم تقع.
                    liked = if (added) liked + itemId else liked - itemId
                    if (!added) {
                        favorites = favorites?.filter { it.id != itemId }
                    } else {
                        // **والمضافُ يُقرأ من الخادم** — الصنفُ الذي
                        // ضُغط قلبُه في السوق **ليس بين يدي هذا
                        // النموذج**، فلا يُبنى سطرٌ بالظنّ.
                        runCatching { api.favorites().items }
                            .onSuccess { favorites = it }
                    }
                }
                .onFailure { error = apiError(getApplication(), it as Exception) }
        }
    }

    private fun load(block: suspend () -> Unit) {
        busy = true
        viewModelScope.launch {
            try {
                block()
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}
