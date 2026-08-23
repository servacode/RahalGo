package com.rahalgo.map

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
 * # ولماذا في وحدة الخرائط
 *
 * **كان هذا الملفُّ منسوخاً في تطبيق الزبون وتطبيق المندوب** — نسختان
 * متطابقتان حرفاً بحرف. **وثالثةٌ في تطبيق المتجر (٢٠٢٦-٠٨-٢٣) تجعلها
 * ثلاثاً تفترق يومَ يُصلَح أحدُها**، ثمّ لا يعلم أحدٌ أيُّها الحقّ.
 *
 * **وموضعُه الطبيعيُّ هنا**: **من احتاج قراءةَ موضعٍ احتاج خريطةً
 * يضعه عليها** — ولا عكس.
 *
 * # ولماذا لا خدمةَ كخدمة السائق
 *
 * **السائقُ يُتتبَّع ما دامت ورديّتُه مفتوحة** — الزبونُ يراه يتحرّك.
 * **وصاحبُ المتجر لا يُتتبَّع أبداً**: موقعُه يُقرأ مرّةً لتبدأ منه
 * الخريطةُ حين يصحّح دبّوسَ متجره.
 *
 * **وتطبيقٌ يتتبّع من لا حاجةَ لتتبّعه يستنزف بطّاريّتَه** — ويُسأل عنه
 * في متجر غوغل، **ويُرفض إن لم يكن له سببٌ ظاهر.**
 *
 * # ولماذا `Priority.HIGH_ACCURACY`
 *
 * **الدبّوسُ يقف عليه سائق** — ومئةُ مترٍ خطأً بابُ جارِه. **وقراءةٌ
 * واحدةٌ دقيقةٌ أرخصُ من عشرٍ تقريبيّة.**
 *
 * # وفشلُها لا يُسقط شيئا
 *
 * **من رفض الإذنَ أو أطفأ موقعَه يحرّك الخريطةَ بيده** — والدبّوسُ
 * يبقى حيث كان. **وشاشةٌ تنتظر نقطةً لا تجيء أسوأُ من خريطةٍ تبدأ من
 * مركز المدينة.**
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
