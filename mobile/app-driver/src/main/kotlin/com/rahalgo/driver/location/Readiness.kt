package com.rahalgo.driver.location

import android.content.Context
import android.content.Intent
import android.location.LocationManager
import android.os.Build
import android.provider.Settings
import androidx.core.content.ContextCompat

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أجاهزٌ للعمل؟ — حالٌ واحدةٌ لا عشرُ رايات** (`DR`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطبُ الذي كان
 *
 * **و`hasLocation()` تسأل عن الإذن وحدَه** — **والإذنُ ممنوحٌ وخدمةُ
 * الموقع مطفأةٌ حالٌ واقعةٌ كلَّ يوم**: **يُطفئها صاحبُها ليوفّر
 * بطّاريّةً ثمّ ينسى.**
 *
 * **فيبدو السائقُ جاهزاً ويفتح ورديّتَه** — **ولا موقعَ يُرسَل**،
 * **فيسأل المكتبُ «لماذا لا تصلك طلبات؟»** ولا أحدَ يعرف. **وهو
 * العطبُ الذي يحذّر منه نصُّ `LocationPermission` نفسُه**: **«فيبدو
 * للمكتب أنّ السائق واقفٌ وهو يسير».**
 *
 * # ولمَ حالٌ واحدةٌ لا فحوصٌ متفرّقة
 *
 * **وكلُّ شاشةٍ تفحص ما يخصُّها** — **فتقول إحداها «جاهز» وتقول
 * الأخرى «ينقصك شيء».** **ومن أضاف شرطاً رابعاً أضافه في موضعٍ
 * ونسيه في ثلاثة.**
 *
 * # ولا يُمنَع ما لا يحتاج جاهزيّة
 *
 * **والسجلُّ والإعدادات والمحادثة تُفتح على كلّ حال** — **ومن مُنع من
 * قراءة سجلّه لأنّ موقعَه مطفأٌ عُوقب بلا سبب.** **والمنعُ لحال العمل
 * وحدَها.**
 */
object Readiness {

    /** **ما ينقص ليعمل** — **وواحدٌ يكفي ليُمنَع العمل.** */
    enum class Blocker {
        /** **إذنُ الإشعارات** — **ودونه لا يُرى إشعارُ الخدمة** (١٣+). */
        NOTIFICATION_PERMISSION_REQUIRED,

        /** **إذنُ الموقع أصلاً.** */
        LOCATION_PERMISSION_REQUIRED,

        /**
         * **خدمةُ الموقع في النظام مطفأة.**
         *
         * **والإذنُ ممنوحٌ ولا موقعَ يُنتَج** — **وهذه أخطرُها لأنّها
         * لا تُرى**: **لا نافذةَ تُرفَض ولا رسالةَ تظهر.**
         */
        LOCATION_SERVICE_OFF,

        /**
         * **الموقعُ في الخلفيّة.**
         *
         * **ودونه يسكت الإرسالُ حين تُطفأ الشاشة** — **والجوّالُ في
         * الجيب هو الحالُ الطبيعيّةُ للسائق.**
         */
        BACKGROUND_LOCATION_REQUIRED,
    }

    /**
     * **حالُ الجاهزيّة.**
     *
     * **وفارغةٌ تعني جاهز** — **ولا رايةَ `ready` منفصلةٌ قد تفترق عن
     * قائمتها.**
     */
    data class State(val blockers: List<Blocker> = emptyList()) {
        val ready: Boolean get() = blockers.isEmpty()

        /** **أوّلُ ما يُقال** — **وقائمةٌ من أربعةٍ لا تُقرأ.** */
        val first: Blocker? get() = blockers.firstOrNull()
    }

    /**
     * of **يقرأ الحالَ من النظام.**
     *
     * **والترتيبُ مقصود**: **الإذنُ قبل الخدمة قبل الخلفيّة** —
     * **ومن قيل له «شغّل خدمةَ الموقع» وهو لم يمنح الإذنَ بعدُ ذهب
     * إلى إعداداتٍ لا تُصلح شيئاً.**
     */
    fun of(context: Context): State {
        val out = mutableListOf<Blocker>()
        if (!LocationPermission.granted(context)) {
            out += Blocker.LOCATION_PERMISSION_REQUIRED
        } else if (!serviceEnabled(context)) {
            // **ولا تُسأل الخدمةُ قبل الإذن** — **والترتيبُ أعلاه.**
            out += Blocker.LOCATION_SERVICE_OFF
        }
        if (!LocationPermission.backgroundGranted(context)) {
            out += Blocker.BACKGROUND_LOCATION_REQUIRED
        }
        if (LocationPermission.notificationsNeeded(context)) {
            out += Blocker.NOTIFICATION_PERMISSION_REQUIRED
        }
        return State(out)
    }

    /**
     * serviceEnabled **أخدمةُ الموقع مُشغَّلةٌ في النظام؟**
     *
     * **ومن `P` فصاعداً تُسأل الدالّةُ مباشرةً** — **وقبلها يُسأل
     * المزوّدان**: **الشبكةُ وحدَها تكفي للتقريب، والدقيقُ يحتاج
     * `GPS`**، **وأحدُهما مُشغَّلٌ يعني أنّ الخدمةَ ليست مطفأةً كلَّها.**
     *
     * **وعطبُ القراءة لا يُقرأ منعاً** — **ولا يُوقَف سائقٌ لأنّ
     * سؤالاً عن النظام سقط**: **المنعُ بعلمٍ لا بجهل** (كحال المنصّة).
     */
    fun serviceEnabled(context: Context): Boolean {
        val lm = ContextCompat.getSystemService(context, LocationManager::class.java)
            ?: return true
        return runCatching {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                lm.isLocationEnabled
            } else {
                @Suppress("DEPRECATION")
                lm.isProviderEnabled(LocationManager.GPS_PROVIDER) ||
                    lm.isProviderEnabled(LocationManager.NETWORK_PROVIDER)
            }
        }.getOrDefault(true)
    }

    /**
     * **يفتح صفحةَ إعدادات الموقع في النظام.**
     *
     * **ولا يُفتح «معلوماتُ التطبيق» لهذا** — **فالخطأُ ليس في
     * أذوننا بل في مفتاح النظام**، **ومن وقع على صفحة التطبيق بحث
     * فيها عمّا ليس فيها.** (وهو الدرسُ المكتوبُ في `LocationPermission`.)
     */
    fun openLocationSettings(context: Context) {
        context.startActivity(
            Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **وما يُمنَع وما لا يُمنَع** (`DR-11`، `DR-12`)
     * ══════════════════════════════════════════════════════════════════
     *
     * **والعملُ يحتاج موقعاً يُنتَج فعلاً** — **إذناً وخدمةً معاً.**
     *
     * **والخلفيّةُ والإشعارُ ينقصان الجودةَ ولا يمنعان البدء** —
     * **ومن مُنع من فتح ورديّته لأنّ إشعاراً غيرُ مسموحٍ حُرم عملَه
     * لأجل زينة.** **ويُقال له، ولا يُوقَف.**
     */
    fun canWork(context: Context): Boolean {
        val s = of(context)
        return !s.blockers.contains(Blocker.LOCATION_PERMISSION_REQUIRED) &&
            !s.blockers.contains(Blocker.LOCATION_SERVICE_OFF)
    }
}
