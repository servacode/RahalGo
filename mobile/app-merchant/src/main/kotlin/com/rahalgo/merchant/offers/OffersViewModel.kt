package com.rahalgo.merchant.offers

import android.app.Application
import com.rahalgo.merchant.SelectedStore
import com.rahalgo.merchant.noStoreMsg
import com.rahalgo.shared.merchant.MerchantApi
import com.rahalgo.shared.offers.StoreOffersApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.offers.OfferItem
import com.rahalgo.ui.offers.OfferSection
import com.rahalgo.ui.offers.OfferStore
import com.rahalgo.ui.offers.OffersWizardViewModel

/**
 * **عروضُ متجره — على الشاشة المركزيّة نفسِها التي يستعملها المندوب**
 * (`OffersWizard` في وحدة `ui`).
 *
 * **والصنفُ يُختار مباشرةً من كلّ أصنافه** بحقل بحث — بلا قسمٍ قبله.
 *
 * **والفرعُ المختارُ في كلّ فتح** (تقريرُ فحص المتجر) — **ولا يختار متجراً**،
 * **والأصنافُ المعتمَدةُ وحدَها**: صنفٌ ينتظر المراجعةَ لا يراه الزبونُ فلا خصمَ عليه.
 */
class OffersViewModel(app: Application) : OffersWizardViewModel(app) {

    private val merchant = MerchantApi(AppCore.get().api)
    override val offers = StoreOffersApi.merchant(AppCore.get().api)
    override val picksStore = false

    /** **ويختار الصنفَ مباشرةً** — بلا قسمٍ قبله (طلبُ المالك ٢٠٢٦-١٠-٠١). */
    override val picksSection = false

    override suspend fun stores(): List<OfferStore> {
        val s = SelectedStore.resolve(merchant.stores().stores)
            ?: throw IllegalStateException(noStoreMsg())
        return listOf(OfferStore(s.id, s.name))
    }

    override suspend fun menu(storeId: String): List<OfferSection> =
        merchant.menu(storeId).map { s ->
            OfferSection(
                s.id,
                s.name,
                // **و«غير المتوفر» لا يُعرض** — «مو معقول ينزل عرض لصنف مو موجود عنده».
                s.items.filter { it.approved && it.available }.map { OfferItem(it.id, it.name, it.price) },
            )
        }.filter { it.items.isNotEmpty() }
}
