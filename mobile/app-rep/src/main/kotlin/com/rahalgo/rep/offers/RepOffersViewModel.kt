package com.rahalgo.rep.offers

import android.app.Application
import com.rahalgo.shared.offers.StoreOffersApi
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.offers.OfferItem
import com.rahalgo.ui.offers.OfferSection
import com.rahalgo.ui.offers.OfferStore
import com.rahalgo.ui.offers.OffersWizardViewModel

/**
 * **عروضُ عميله — على الشاشة المركزيّة** (`OffersWizard` في وحدة `ui`).
 *
 * **والمتجرُ يُختار من عملائه** (`/rep/merchants`) — **ولو كُتب معرّفٌ بيدٍ
 * لَرُدّ من الخادم** (`repClient`): **وشاشةٌ تحرس وحدَها ليست حارساً.**
 */
class RepOffersViewModel(app: Application) : OffersWizardViewModel(app) {

    private val rep = RepApi(AppCore.get().api)
    override val offers = StoreOffersApi.rep(AppCore.get().api)
    override val picksStore = true

    override suspend fun stores(): List<OfferStore> =
        rep.merchants().map { OfferStore(it.id, it.name) }

    override suspend fun menu(storeId: String): List<OfferSection> =
        rep.menu(storeId).map { s ->
            // **و«غير المتوفر» لا يُعرض** (نصُّ المالك ٢٠٢٦-١٠-٠١) — والخادمُ يرفضه أيضاً.
            OfferSection(
                s.id,
                s.name,
                s.items.filter { it.available }.map { OfferItem(it.id, it.name, it.price) },
            )
        }.filter { it.items.isNotEmpty() }
}
