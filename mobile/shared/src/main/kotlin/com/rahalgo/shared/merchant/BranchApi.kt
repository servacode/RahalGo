package com.rahalgo.shared.merchant

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أضف فرع» — طلباتُ الفروع** (قرارُ المالك ٢٠٢٦-١١-١٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كما يكتبها المحرّك: `server/branch_requests.go`. صاحبُ المتجر ومندوبُه يطلبان بالبابين،
 * والمكتبُ يوافق ويحدّد العمولة. **والحالةُ نصٌّ**: `pending` · `approved` · `rejected`.
 */

import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** **طلبُ فرعٍ كما يُعرض** — بحالته وسببِ الرفض إن رُفض. */
@Serializable
data class BranchRequest(
    val id: String = "",
    @SerialName("parent_merchant_id") val parentId: String = "",
    @SerialName("parent_name") val parentName: String = "",
    val name: String = "",
    val phone: String? = null,
    val status: String = "pending",
    @SerialName("reject_reason") val rejectReason: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class BranchRequestPage(val requests: List<BranchRequest> = emptyList())

@Serializable
data class BranchRequestInput(
    val name: String,
    val lat: Double,
    val lng: Double,
    val phone: String? = null,
    @SerialName("address_text") val addressText: String? = null,
)

/**
 * **بابا الطلب** — صاحبُ المتجر (`/merchant`) والمندوبُ (`/rep`). **المعرّفُ للمتجر الأمّ.**
 */
class BranchApi(private val api: ApiClient, private val rep: Boolean) {

    /** **طلباتُ هذا المتجر** — المندوبُ يقرأ بالمتجر، وصاحبُ المتجر بكلّ متاجره فيُرشَّح هنا. */
    suspend fun list(parentId: String): List<BranchRequest> =
        if (rep) {
            api.call<BranchRequestPage>("/api/v1/rep/merchants/$parentId/branch-requests").requests
        } else {
            api.call<BranchRequestPage>("/api/v1/merchant/branch-requests").requests
        }

    suspend fun create(parentId: String, input: BranchRequestInput): BranchRequest =
        api.call(
            if (rep) "/api/v1/rep/merchants/$parentId/branch-requests"
            else "/api/v1/merchant/stores/$parentId/branch-requests",
            HttpMethod.Post,
            input,
        )
}
