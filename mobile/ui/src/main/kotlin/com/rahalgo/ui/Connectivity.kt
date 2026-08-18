package com.rahalgo.ui

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الاتّصال — يُعرف قبل أن يُسأل عنه الخادم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «شاشةُ عدم توفّر إنترنت، أو في حال انقطع
 *
 *	الإنترنت عن التطبيق — أيضاً يجب أن تكون مركزيّةً بصورةٍ احترافيّةٍ
 *	جميلةٍ مناسبة».)
 *
 * # ما كان يقع
 *
 * **لا شيءَ يعرف حالَ الشبكة.** فمن انقطع عنه الإنترنتُ ينتظر النداءَ
 * حتّى ينتهي وقتُه، **ثمّ يقرأ «تعذّر الاتّصال بالخادم»** — فيظنّ أنّ
 * المنصّةَ معطّلةٌ وهو من فقد الشبكة.
 *
 * # ورايةٌ واحدةٌ ساكنة
 *
 * **لا يُسجَّل مراقبٌ لكلّ شاشة**: عشرُ شاشاتٍ تعني عشرةَ مراقبين
 * يوقظهم النظامُ معاً. **ويُسجَّل مرّةً عند الإقلاع ويبقى.**
 */
object Net {

    /** **أمتّصلٌ الآن؟** — ويبدأ متفائلاً. */
    var online by mutableStateOf(true)
        private set

    @Volatile private var wired = false

    /**
     * **يُنادى مرّةً في `onCreate`.**
     *
     * **ويبدأ بقراءة الحال لا بانتظار حدث**: `NetworkCallback` لا
     * تُنادى إن لم يتغيّر شيء، **فمن فتح تطبيقَه وهو مقطوعٌ لا يُبلَّغ
     * أبدا.**
     */
    fun install(context: Context) {
        if (wired) return
        wired = true
        val cm = context.applicationContext
            .getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return

        online = has(cm)

        // **والقدرةُ هي المقياسُ لا وجودُ الشبكة**: واي-فايٌ بلا إنترنت
        // **شبكةٌ موصولةٌ لا تُوصِّل** — `VALIDATED` هي ما يفرّق.
        val req = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        runCatching {
            cm.registerNetworkCallback(
                req,
                object : ConnectivityManager.NetworkCallback() {
                    override fun onAvailable(network: Network) {
                        online = has(cm)
                    }

                    override fun onLost(network: Network) {
                        online = has(cm)
                    }

                    override fun onCapabilitiesChanged(n: Network, c: NetworkCapabilities) {
                        online = has(cm)
                    }
                },
            )
        }
    }

    private fun has(cm: ConnectivityManager): Boolean {
        val c = cm.getNetworkCapabilities(cm.activeNetwork) ?: return false
        return c.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
            c.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطٌ ينزل من الأعلى — لا شاشةٌ تحجب ما بين يديه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا شريطٌ لا شاشة
 *
 * **من انقطع عنه الإنترنتُ وهو يقرأ سلّتَه لا يريد أن تُمحى الشاشةُ من
 * أمامه** — ما جُلب لا يزال صالحاً للقراءة، **وحجبُه عقوبةٌ على عطبٍ
 * ليس منه.**
 *
 * **والشاشةُ الكاملةُ لحالٍ واحدة**: أن لا يكون هناك شيءٌ يُعرض أصلاً —
 * وتلك `OfflineScreen`.
 *
 * # ويعلو الرسالةَ الطافية
 *
 * **رسالةُ `Flash` تنصرف بعد ثوانٍ وهذا يبقى** — فلو تحته لَغطّته
 * لحظتَها، **ولو فوقه لَقفز الشريطُ حين تنصرف.**
 */
@Composable
fun NetBanner(modifier: Modifier = Modifier) {
    AnimatedVisibility(
        visible = !Net.online,
        enter = slideInVertically { -it },
        exit = slideOutVertically { -it },
        modifier = modifier,
    ) {
        Row(
            Modifier
                .fillMaxWidth()
                .background(Rahal.colors.danger)
                .padding(horizontal = 14.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_no_signal),
                contentDescription = null,
                tint = Color.White,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.size(8.dp))
            Text(
                text = stringResource(R.string.net_offline_banner),
                color = Color.White,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.labelLarge,
            )
        }
    }
}

/**
 * **شاشةُ الانقطاع الكاملة** — حين لا شيءَ يُعرض أصلا.
 *
 * **وزرُّ الإعادة يبقى ظاهراً وإن كان مقطوعاً**: من أعاد الاتّصالَ من
 * إعداداته **يريد أن يُجرّب بيده** ولا ينتظر أن يلتقط النظامُ التغيّر.
 */
@Composable
fun OfflineScreen(onRetry: (() -> Unit)? = null, modifier: Modifier = Modifier) {
    Column(
        modifier
            .fillMaxSize()
            .padding(ScreenPad),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            Modifier
                .size(96.dp)
                .background(Rahal.colors.danger.copy(alpha = 0.10f), Rahal.shape.pill),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_no_signal),
                contentDescription = null,
                tint = Rahal.colors.danger,
                modifier = Modifier.size(44.dp),
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            text = stringResource(R.string.net_offline_title),
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.net_offline_body),
            color = Rahal.colors.inkMuted,
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.fillMaxWidth(),
        )
        if (onRetry != null) {
            Spacer(Modifier.height(20.dp))
            RahalOutlineButton(onClick = onRetry) {
                Text(stringResource(R.string.act_retry))
            }
        }
    }
}
