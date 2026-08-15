package com.rahalgo.customer

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.util.Log
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.rahalgo.ui.LastPoint

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أين هو الآن — قراءةٌ واحدةٌ عند الحاجة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا لا خدمةَ كخدمة السائق
 *
 * **السائقُ يُتتبَّع ما دامت ورديّتُه مفتوحة** — الزبونُ يراه يتحرّك.
 * **والزبونُ لا يُتتبَّع أبدا**: موقعُه لا يلزم إلّا لحظةَ يحفظ عنواناً،
 * **ومرّةً في العمر أو مرّتين.**
 *
 * **وتطبيقٌ يتتبّع من لا حاجةَ لتتبّعه يستنزف بطّاريّتَه** — ويُسأل عنه
 * في متجر غوغل، **ويُرفض إن لم يكن له سببٌ ظاهر.**
 *
 * # ولماذا `Priority.HIGH_ACCURACY`
 *
 * **العنوانُ يُسلَّم إليه طلبٌ** — ومئةُ مترٍ خطأً بابُ جارِه. **وقراءةٌ
 * واحدةٌ دقيقةٌ أرخصُ من عشرٍ تقريبيّة.**
 *
 * # وفشلُها لا يُسقط شيئا
 *
 * **من رفض الإذنَ أو أطفأ موقعَه يكتب عنوانَه نصّاً** — والحقلُ يبقى
 * فارغاً ويعرف أنّه فارغ. **وشاشةٌ تنتظر نقطةً لا تجيء أسوأُ من عنوانٍ
 * بلا نقطة.**
 */
object Here {

    fun granted(context: Context): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) ==
            PackageManager.PERMISSION_GRANTED ||
            ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) ==
            PackageManager.PERMISSION_GRANTED

    /**
     * **يقرأ الموضعَ مرّةً ويكتبه في الحامل المشترك.**
     *
     * **والإذنُ يُفحص هنا لا عند المنادي** — `@SuppressLint` بلا فحصٍ
     * هو الذي يُسقط التطبيقاتِ في يد من رفض.
     */
    @SuppressLint("MissingPermission")
    fun refresh(context: Context) {
        if (!granted(context)) return
        runCatching {
            LocationServices.getFusedLocationProviderClient(context)
                .getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null)
                .addOnSuccessListener { loc ->
                    if (loc != null) LastPoint.set(loc.latitude, loc.longitude)
                }
                .addOnFailureListener { e ->
                    Log.w("RahalGo/here", "تعذّرت قراءةُ الموضع", e)
                }
        }
    }
}
