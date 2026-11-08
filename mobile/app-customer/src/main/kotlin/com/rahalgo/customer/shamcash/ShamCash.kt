package com.rahalgo.customer.shamcash

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.customer.ShamCashApi
import com.rahalgo.shared.customer.ShamConfig
import com.rahalgo.ui.AppCore

/**
 * Sham Cash state shared by the wallet, the cart and the status screen.
 *
 * Switched off until the server says `enabled`: [enabled] is false when the call fails
 * (an old server answers 404) so every Sham Cash element stays hidden.
 */
object ShamNav {

    var config by mutableStateOf<ShamConfig?>(null)
        private set

    val enabled: Boolean get() = config?.enabled == true

    /** The bill whose status screen is open, or null. */
    var billId by mutableStateOf<String?>(null)
        private set

    /** The top-up amount screen is open. */
    var topup by mutableStateOf(false)
        private set

    /** A payment page to open in the real browser as soon as the status screen shows. */
    private var pendingUrl: String? = null

    suspend fun refresh() {
        config = runCatching { ShamCashApi(AppCore.get().api).config() }.getOrNull()
    }

    fun openTopup() {
        billId = null
        topup = true
    }

    fun show(id: String, urlToOpen: String? = null) {
        pendingUrl = urlToOpen?.takeIf { it.isNotBlank() }
        topup = false
        billId = id
    }

    fun takePendingUrl(): String? = pendingUrl.also { pendingUrl = null }

    fun close() {
        pendingUrl = null
        topup = false
        billId = null
    }

    /** The app was opened from a rahalgo.com link; open the status screen if it is a Sham Cash return. */
    fun fromIntent(intent: Intent?) {
        val id = ShamLink.billId(intent?.data?.host, intent?.data?.path) ?: return
        show(id)
    }
}

/** The return link Sham Cash redirects the browser to: `https://rahalgo.com/wallet/shamcash/<billId>`. */
object ShamLink {

    private const val PREFIX = "/wallet/shamcash"
    private val HOSTS = setOf("rahalgo.com", "www.rahalgo.com")
    private val ID = Regex("^[A-Za-z0-9-]{1,64}$")

    fun billId(host: String?, path: String?): String? {
        if (host == null || host.lowercase() !in HOSTS || path == null) return null
        if (!path.startsWith("$PREFIX/")) return null
        val id = path.removePrefix("$PREFIX/").trim('/').substringBefore('/')
        return id.takeIf { ID.matches(it) }
    }
}

/**
 * Opens the payment page in the phone's real browser (not a WebView): Sham Cash's page
 * deep-links into the Sham Cash app and an in-app WebView breaks that.
 * Returns false when no browser could be started.
 */
fun openPaymentPage(context: Context, url: String): Boolean {
    if (!url.startsWith("https://") && !url.startsWith("http://")) return false
    val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
    if (context !is Activity) intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    return try {
        context.startActivity(intent)
        true
    } catch (e: ActivityNotFoundException) {
        false
    } catch (e: SecurityException) {
        false
    }
}
