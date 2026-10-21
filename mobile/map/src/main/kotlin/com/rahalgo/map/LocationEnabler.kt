package com.rahalgo.map

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.location.LocationManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.IntentSenderRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import com.google.android.gms.common.api.ResolvableApiException
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.LocationSettingsRequest
import com.google.android.gms.location.Priority
import com.rahalgo.ui.locationServiceEnabled
import com.rahalgo.ui.openLocationSettings

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«تشغيل الموقع؟ موافق» — من داخل التطبيق** (قرارُ المالك ٢٠٢٦-١٠-٢١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان زرُّ «شغّل خدمة الموقع» يفتح صفحةَ الإعدادات** — فيخرج الزبونُ من التطبيق
 * ويضيع. **وهذه نافذةُ غوغل نفسُها** (التي في خرائطها): «موافق» فيشتغل الموقعُ
 * وهو في مكانه. **وإن لم تتوفّر النافذة** (جهازٌ بلا خدمات غوغل) فُتحت الإعداداتُ
 * كما كانت.
 *
 * **و`onResult(true)`** حين صار الموقعُ مشغَّلاً — فيُعاد التحديدُ أو تبدأ المشاركة.
 */
@Composable
fun rememberLocationEnabler(onResult: (enabled: Boolean) -> Unit): () -> Unit {
    val context = LocalContext.current
    val latest by rememberUpdatedState(onResult)
    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.StartIntentSenderForResult(),
    ) { r ->
        latest(r.resultCode == Activity.RESULT_OK || locationServiceEnabled(context))
    }
    return remember(context, launcher) {
        {
            if (locationServiceEnabled(context)) {
                latest(true)
            } else {
                val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, 5_000L).build()
                LocationServices.getSettingsClient(context)
                    .checkLocationSettings(
                        LocationSettingsRequest.Builder()
                            .addLocationRequest(req)
                            .setAlwaysShow(true)
                            .build(),
                    )
                    .addOnSuccessListener { latest(true) }
                    .addOnFailureListener { e ->
                        val shown = e is ResolvableApiException && runCatching {
                            launcher.launch(IntentSenderRequest.Builder(e.resolution).build())
                        }.isSuccess
                        if (!shown) openLocationSettings(context)
                    }
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **والتطبيقُ يحسّ بمفتاح الموقع** (قرارُ المالك ٢٠٢٦-١٠-٢١: «يشغل الموقع وهو
 * بالتطبيق مالازم يسكر التطبيق ويفتحه»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومن شغّله من الشريط العلويّ لا يمرّ بـ`ON_RESUME`** — النشاطُ لا يتوقّف، فلم
 * يكن أحدٌ يعلم. **فيُسمَع بثُّ النظام** ما دامت الشاشةُ حيّة، **ويُسأل المفتاحُ
 * نفسُه** عند كلّ بثّ — فالبثُّ يتكرّر لكلّ مزوّد، والمنادي يعيد التحديدَ مرّةً.
 */
@Composable
fun LocationServiceEffect(onEnabled: () -> Unit) {
    val context = LocalContext.current
    val latest by rememberUpdatedState(onEnabled)
    DisposableEffect(context) {
        val receiver = object : BroadcastReceiver() {
            override fun onReceive(c: Context, i: Intent) {
                if (locationServiceEnabled(c)) latest()
            }
        }
        // **ومُصدَّرٌ** لأنّ المرسلَ النظام؛ **ومن أرسله زوراً لا يكسب شيئاً**:
        // المفتاحُ يُسأل ثانيةً قبل أيّ فعل.
        ContextCompat.registerReceiver(
            context, receiver, IntentFilter(LocationManager.PROVIDERS_CHANGED_ACTION),
            ContextCompat.RECEIVER_EXPORTED,
        )
        onDispose { runCatching { context.unregisterReceiver(receiver) } }
    }
}
