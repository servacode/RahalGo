package com.rahalgo.shared.driver

import com.rahalgo.shared.model.Inbox
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
}
