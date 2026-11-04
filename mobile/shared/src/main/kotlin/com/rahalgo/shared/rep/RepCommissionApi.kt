package com.rahalgo.shared.rep

import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * **نسبةُ عمولة متجر العميل — يضبطها المندوب** (قرارُ المالك ٢٠٢٦-١١-٠٤: «بين ال0 وال10 بالمية»).
 *
 * فوراً بلا موافقة المكتب، **والحدّان من إعدادات المحرّك** فلا يُكتبان في التطبيق.
 * والطلبُ المسلَّمُ لا تتغيّر عمولتُه.
 */
class RepCommissionApi(private val api: ApiClient) {
    /** **يحفظ النسبة** — نصّاً كما كُتبت («7.5») لئلّا يُفسدها عائم. */
    suspend fun setStoreCommission(merchantID: String, percent: String): RepStoreCommission =
        api.call(
            "/api/v1/rep/stores/$merchantID/commission",
            HttpMethod.Patch,
            mapOf("percent" to percent),
        )
}

/** **نسبةُ المتجر الآن وحدّا ما يضبطه المندوب.** */
@Serializable
data class RepStoreCommission(
    @SerialName("store_commission_percent") val percent: Double = 0.0,
    @SerialName("store_commission_min") val min: Double = 0.0,
    @SerialName("store_commission_max") val max: Double = 10.0,
)
