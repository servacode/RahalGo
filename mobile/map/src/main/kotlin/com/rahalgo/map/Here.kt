package com.rahalgo.map

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.util.Log
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import android.os.Handler
import android.os.Looper
import com.rahalgo.ui.Accuracy
import com.rahalgo.ui.LastPoint
import com.rahalgo.ui.Locating
import com.rahalgo.ui.locationServiceEnabled

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

    /** **كم يُنتظَر الطازجُ قبل أن يُقال «طال الانتظار»** (`MLW-07`). */
    const val TIMEOUT_MS: Long = 15_000L

    fun granted(context: Context): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) ==
            PackageManager.PERMISSION_GRANTED ||
            ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) ==
            PackageManager.PERMISSION_GRANTED

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يقرأ الموضعَ — على مرحلتين، ويقول ما وقع** (`MLW`، ٢٠٢٦-٠٩-١٥)
     * ══════════════════════════════════════════════════════════════════
     *
     * **وكانت تنادي وتمضي**: **لا حالَ ولا جواب** — **فمن رُفض إذنُه
     * ضغط «موقعي» فلم يقع شيءٌ ولا رسالة**، **ثمّ ضغط ثانيةً وثالثة.**
     *
     * # ومرحلتان لا واحدة
     *
     * **وآخرُ موضعٍ معروفٍ يصل في اللحظة** — **فيُوسَّط به فوراً**،
     * **ثمّ يُصحَّح بالطازج.** **وشاشةٌ تتحرّك في اللحظة تقول إنّ
     * الزرَّ عمل.**
     *
     * # ونداءٌ واحدٌ في الجوّ
     *
     * **و`start` تردّ `false` إن كان قائماً** — **وضغطتان سريعتان
     * نداءان يتسابقان فيصل أقدمُهما آخراً فيُوسَّط به.**
     *
     * # وما لا يُقاس لا يُحكَم به
     *
     * **والقياسُ الذي لا تكفي دقّتُه لا يُكتب في `LastPoint`** —
     * **وتوسيطُ خريطةٍ يقبل ما لا يقبله تأكيدُ نقطة، وتتبّعُ السائق
     * أشدُّها** (`Accuracy`).
     *
     * @param state **حالُ الطلب** — **تقرؤها الشاشةُ لترسم الزرّ
     *   والسبب.** **وفارغةٌ تعني منادياً لا يرسم شيئاً.**
     * @param limit **حدُّ الدقّة المقبولةُ لهذا الاستعمال.**
     * @param onPoint **يُنادى بالموضع وبدقّته** — **لمن يحتاج الدقّةَ
     *   نفسَها** (نقطةُ الاستكشاف تُحفَظ بدقّتها).
     */
    @SuppressLint("MissingPermission")
    fun refresh(
        context: Context,
        state: Locating? = null,
        limit: Float = Accuracy.CENTER_M,
        onPoint: ((Double, Double, Float) -> Unit)? = null,
    ) {
        // **ونداءٌ قائمٌ لا يُتبَع بثانٍ.**
        if (state != null && !state.start()) return

        // ══════════════════════════════════════════════════════════════
        // **والسببُ يُقال باسمه** (`MLW-06`)
        // ══════════════════════════════════════════════════════════════
        //
        // **وعلاجُ الإذن غيرُ علاج الخدمة المطفأة** — **ومن قيل له
        // «حاول ثانية» وخدمتُه مطفأةٌ حاول عشراً.**
        if (!granted(context)) {
            state?.failed(Locating.Problem.PERMISSION_DENIED)
            return
        }
        if (!locationServiceEnabled(context)) {
            state?.failed(Locating.Problem.SERVICE_OFF)
            return
        }

        // **ولا تُنهي المهلةُ نداءً انتهى** — **والحارسُ يقرأ `busy`.**
        if (state != null) {
            Handler(Looper.getMainLooper()).postDelayed({
                if (state.busy) state.failed(Locating.Problem.TIMEOUT)
            }, TIMEOUT_MS)
        }

        val client = LocationServices.getFusedLocationProviderClient(context)

        // ══════════════════════════════════════════════════════════════
        // **والمرحلةُ الأولى: آخرُ موضعٍ معروف**
        // ══════════════════════════════════════════════════════════════
        //
        // **ويصل في اللحظة لأنّه محفوظٌ في النظام** — **ولا يُتَّخذ
        // حقيقةَ توصيل**: **عقدُ الدفعة الخامسة قائمٌ، والتأكيدُ فعلٌ
        // صريحٌ من صاحبه.**
        //
        // **ولا يُقبَل قديمٌ لا تكفي دقّتُه** — **ولا يُقفز بالخريطة
        // إلى حيٍّ آخر.**
        runCatching {
            client.lastLocation.addOnSuccessListener { loc ->
                if (loc == null || !state.stillWanting()) return@addOnSuccessListener
                if (!Accuracy.enough(if (loc.hasAccuracy()) loc.accuracy else -1f, limit)) {
                    return@addOnSuccessListener
                }
                LastPoint.set(loc.latitude, loc.longitude)
                state?.provisional()
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والثانية: قياسٌ طازجٌ يُصحّح**
        // ══════════════════════════════════════════════════════════════
        runCatching {
            client.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null)
                .addOnSuccessListener { loc ->
                    if (loc == null) {
                        state?.failed(Locating.Problem.UNAVAILABLE)
                        return@addOnSuccessListener
                    }
                    val acc = if (loc.hasAccuracy()) loc.accuracy else -1f
                    // **ودقّةٌ لا تكفي لا تُكتب** — **ولا يُقال إنّها
                    // موضعُه**: **يُقال إنّها ضعيفة، ويحرّك الخريطةَ
                    // بيده.**
                    if (!Accuracy.enough(acc, limit)) {
                        state?.failed(Locating.Problem.WEAK_ACCURACY)
                        return@addOnSuccessListener
                    }
                    LastPoint.set(loc.latitude, loc.longitude)
                    onPoint?.invoke(loc.latitude, loc.longitude, acc)
                    state?.fixed()
                }
                .addOnFailureListener { e ->
                    Log.w("RahalGo/here", "تعذّرت قراءةُ الموضع", e)
                    state?.failed(Locating.Problem.UNAVAILABLE)
                }
        }.onFailure {
            Log.w("RahalGo/here", "تعذّر نداءُ الموضع", it)
            state?.failed(Locating.Problem.UNAVAILABLE)
        }
    }

    /**
     * **أما زال الطلبُ قائماً؟**
     *
     * **والقديمُ قد يصل بعد أن انتهى الطلبُ بمهلةٍ أو بفشل** — **فلا
     * يُقفز بالخريطة بعد أن قرأ صاحبُها السبب.**
     *
     * **وبلا حالٍ يُقبَل كلُّ شيء** — **فمنادٍ لا يرسم لا ينتظر.**
     */
    private fun Locating?.stillWanting(): Boolean = this == null || busy
}
