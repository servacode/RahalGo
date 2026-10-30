package com.rahalgo.customer.update

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.BuildConfig
import com.rahalgo.customer.R
import com.rahalgo.ui.ApkDownload
import com.rahalgo.ui.Flash
import com.rahalgo.ui.PlayUpdate
import com.rahalgo.ui.installedFromPlay
import com.rahalgo.design.Rahal
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.net.HttpURLConnection
import java.net.URL

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«في نسخة جديدة — حدّث»** (قرارُ المالك ٢٠٢٦-١٠-٢١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت شاشةُ التحديث تظهر حين تُمنع النسخةُ وحدَها** (`UpdateGate` عند ٤٢٦) —
 * ومن نسختُه مقبولةٌ وقديمةٌ لا يعلم بالجديدة. **فشريطٌ رقيقٌ تحت الأعلى** يُغلَق.
 *
 * **والتحديثُ من حيث ثُبِّت** (`installedFromPlay`): من غوغل بلاي ⇒ التحديثُ داخل
 * التطبيق، **فإن لم يُوصَل لغوغل** (محظورٌ بلا VPN) قيل له ذلك وفُتح المتجر؛ ومن
 * رابطنا ⇒ صفحةُ التنزيل عندنا، بلا VPN.
 */
object UpdateOffer {
    /** **النسخةُ الأحدث إن وُجدت** — وفارغٌ: لا جديد. */
    var newer by mutableStateOf<String?>(null)
        private set
    var dismissed by mutableStateOf(false)

    const val DOWNLOAD_PAGE = "https://rahalgo.com/download/customer"

    @Serializable
    private data class Releases(val data: Data = Data()) {
        @Serializable
        data class Data(val apps: List<App> = emptyList())

        @Serializable
        data class App(val key: String = "", val version: String = "")
    }

    private val json = Json { ignoreUnknownKeys = true }

    /** **يسأل الخادمَ مرّةً في الفتحة** — وتعثّرُه صمتٌ لا خطأ. */
    suspend fun check() {
        val latest = withContext(Dispatchers.IO) {
            runCatching {
                val c = URL(BuildConfig.API_BASE_URL + "/api/v1/public/releases").openConnection() as HttpURLConnection
                c.connectTimeout = 8_000
                c.readTimeout = 8_000
                c.inputStream.bufferedReader().use { it.readText() }
            }.mapCatching { body ->
                json.decodeFromString(Releases.serializer(), body).data.apps.firstOrNull { it.key == "customer" }?.version
            }.getOrNull()
        }
        newer = latest?.takeIf { isNewer(it, BuildConfig.VERSION_NAME) }
    }

    /** **أأحدثُ «أ» من «ب»؟** — بالأرقام لا بالحروف (١٫١٠ بعد ١٫٩). */
    fun isNewer(a: String, b: String): Boolean {
        fun parts(v: String) = v.trim().split('.', '-', ' ').mapNotNull { it.toIntOrNull() }
        val x = parts(a)
        val y = parts(b)
        if (x.isEmpty() || y.isEmpty()) return false
        for (i in 0 until maxOf(x.size, y.size)) {
            val p = x.getOrElse(i) { 0 }
            val q = y.getOrElse(i) { 0 }
            if (p != q) return p > q
        }
        return false
    }

    /** **يحدّث من حيث ثُبِّت.** */
    fun update(context: Context) {
        val activity = context as? Activity
        if (installedFromPlay(context) && activity != null) {
            PlayUpdate.start(activity) {
                Flash.fail(context.getString(R.string.update_play_unreachable))
                openUrl(context, "market://details?id=" + context.packageName)
            }
        } else if (!downloadApk(context)) {
            openUrl(context, DOWNLOAD_PAGE)
        }
    }

    /**
     * **من نزّل من موقعنا: الملفُّ ينزل على جهازه مباشرة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) —
     * لا صفحةٌ في المتصفّح. منزّلُ أندرويد يُظهر إشعاراً، **ويكبس عليه فيثبّت بيده.**
     * ولا إذنَ تثبيتٍ في التطبيق — غوغل بلاي يمنعه، والتثبيتُ يقوم به أندرويد نفسُه.
     */
    private fun downloadApk(context: Context): Boolean = ApkDownload.start(
        context,
        BuildConfig.API_BASE_URL + "/api/v1/public/app/customer",
        "rahalgo-${newer ?: "update"}.apk",
    )

    private fun openUrl(context: Context, url: String) {
        runCatching {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
        }
    }
}

/** **الشريطُ نفسُه** — «في نسخة جديدة من رحّال غو — حدّث» و«×». */
@Composable
fun UpdateStrip() {
    val context = LocalContext.current
    LaunchedEffect(Unit) { UpdateOffer.check() }
    val v = UpdateOffer.newer ?: return
    if (UpdateOffer.dismissed) return
    Row(
        Modifier
            .fillMaxWidth()
            .background(Rahal.colors.brand)
            .clickable { UpdateOffer.update(context) }
            .padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.update_offer, v),
            color = Rahal.colors.onBrand,
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.weight(1f),
        )
        Spacer(Modifier.width(8.dp))
        Text(
            text = "✕",
            color = Rahal.colors.onBrand,
            modifier = Modifier
                .clickable { UpdateOffer.dismissed = true }
                .padding(horizontal = 6.dp),
        )
    }
}
