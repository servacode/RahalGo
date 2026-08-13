package com.rahalgo.shared.driver

import com.rahalgo.shared.model.Inbox
import com.rahalgo.shared.model.Payout
import com.rahalgo.shared.model.PayoutInput
import com.rahalgo.shared.model.Reputation
import com.rahalgo.shared.model.WalletStatement
import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod

/** **ما يخصّ صاحب الحساب** — إشعاراته ومحفظته. */
class MeApi(private val api: ApiClient) {

    suspend fun inbox(limit: Int = 30): Inbox =
        api.call("/api/v1/me/notifications?limit=" + limit)

    /** **يُعلّم المقروء** — وبلاه تبقى الشارة حمراء وقد قرأها. */
    suspend fun markRead() {
        api.call<Ack>("/api/v1/me/notifications/read", HttpMethod.Post, mapOf<String, String>())
    }

    suspend fun wallet(): WalletStatement = api.call("/api/v1/my/wallet")

    /**
     * **سُمعتُه** — نجومُه ومن أعطاها وشكاواه.
     *
     * (`GET /api/v1/me/reputation` — والنداءُ نفسُه الذي تقرؤه شاشةُ
     * الويب، **فلا يفترق ما يراه في الاثنتين.**)
     */
    suspend fun reputation(): Reputation = api.call("/api/v1/me/reputation")

    /** **طلباتُ سحبه** — ما طلبه وما قرّرته المالية. */
    suspend fun payouts(): List<Payout> = api.call("/api/v1/me/payouts")

    /**
     * **يطلب سحباً** — والمبلغُ يُحجَز ولا يُصرَف حتّى يُقرَّر.
     *
     * **ومفتاحُ منع التكرار إلزاميّ**: شبكةٌ تنقطع بعد الإرسال وقبل
     * الردّ **تجعل الإصبعَ يعيد الضغط** — فيُحجَز المبلغُ مرّتين.
     */
    suspend fun requestPayout(amount: Long, note: String, key: String) {
        api.call<Ack>(
            "/api/v1/me/payouts",
            HttpMethod.Post,
            PayoutInput(amount, note),
            idempotencyKey = key,
        )
    }
}
