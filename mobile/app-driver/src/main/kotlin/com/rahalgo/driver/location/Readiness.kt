package com.rahalgo.driver.location

import android.content.Context

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
    fun of(context: Context): State = evaluate(
        permission = LocationPermission.granted(context),
        service = serviceEnabled(context),
        background = LocationPermission.backgroundGranted(context),
        notifications = !LocationPermission.notificationsNeeded(context),
    )

    /**
     * evaluate **القرارُ وحدَه — بلا نظامٍ ولا سياق.**
     *
     * **وقراءةُ النظام في `of` والحكمُ هنا** — **فيُقاس الحكمُ في آلةٍ
     * بلا أندرويد.**
     *
     * **وقِيس**: **شاهدٌ سلبيٌّ رفع فحصَ الخدمة ولم يسقط شيء** —
     * **لأنّ الفحصَ كان يبني صفَّ بياناتٍ بيده ويقرؤه، لا يقيس
     * القرار.** **وحارسٌ لا يحرس القرارَ زينة.**
     */
    fun evaluate(
        permission: Boolean,
        service: Boolean,
        background: Boolean,
        notifications: Boolean,
    ): State {
        val out = mutableListOf<Blocker>()
        if (!permission) {
            out += Blocker.LOCATION_PERMISSION_REQUIRED
        } else if (!service) {
            // **ولا تُسأل الخدمةُ قبل الإذن** — **والترتيبُ أعلاه.**
            out += Blocker.LOCATION_SERVICE_OFF
        }
        if (!background) {
            out += Blocker.BACKGROUND_LOCATION_REQUIRED
        }
        if (!notifications) {
            out += Blocker.NOTIFICATION_PERMISSION_REQUIRED
        }
        return State(out)
    }

    /** **أيُسمَح بالعمل بهذه الحال؟** — **والقرارُ يُقاس كغيره.** */
    fun canWork(s: State): Boolean =
        !s.blockers.contains(Blocker.LOCATION_PERMISSION_REQUIRED) &&
            !s.blockers.contains(Blocker.LOCATION_SERVICE_OFF)

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
    fun serviceEnabled(context: Context): Boolean =
        com.rahalgo.ui.locationServiceEnabled(context)

    /**
     * **يفتح صفحةَ إعدادات الموقع في النظام.**
     *
     * **ولا يُفتح «معلوماتُ التطبيق» لهذا** — **فالخطأُ ليس في
     * أذوننا بل في مفتاح النظام**، **ومن وقع على صفحة التطبيق بحث
     * فيها عمّا ليس فيها.** (وهو الدرسُ المكتوبُ في `LocationPermission`.)
     */
    fun openLocationSettings(context: Context) =
        com.rahalgo.ui.openLocationSettings(context)

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ودرجاتٌ لا رايةٌ واحدة** (`DRF-01`، ٢٠٢٦-٠٩-١٥)
     * ══════════════════════════════════════════════════════════════════
     *
     * **ولكلّ فعلٍ ما يحتاجه** — **ورايةٌ واحدةٌ تمنع الكلَّ لأجل
     * نقصٍ يخصّ فعلاً واحداً.**
     */
    enum class Level {
        /** **يُقرأ السجلُّ وتُفتَح الإعداداتُ والمحادثة** — على كلّ حال. */
        APP_USABLE,

        /**
         * **يُعلَن متاحاً لطلبٍ جديد.**
         *
         * **وهذه أشدُّها** — **لأنّ معناها «أرسلوا إليّ عملاً»**:
         * **ومن أُعلن متاحاً ولا يبلغه النداءُ يُحسَب رافضاً وهو لا
         * يعلم.**
         */
        CAN_GO_ONLINE,

        /**
         * **يُتابع رحلةً في يده.**
         *
         * **والطلبُ مقبولٌ وهو يعرفه** — **فلا يحتاج نداءً يُوقظه.**
         */
        CAN_RUN_ACTIVE_DELIVERY,
    }

    /**
     * allows **أتكفي هذه الحالُ لهذه الدرجة؟**
     *
     * # والقياسُ لا الحدس (٢٠٢٦-٠٩-١٥)
     *
     * **الموقعُ في الخلفيّة: لا يُمنَع به عمل.**
     * **وقِيس في المستودع**: **الخدمةُ أماميّةٌ من نوع `location`**
     * (`foregroundServiceType="location"` في البيان،
     * `FOREGROUND_SERVICE_TYPE_LOCATION` في `startForeground`)،
     * **وتُشغَّل من شاشة الورديّة والتطبيقُ مرئيّ.** **وأندرويد يُبقي
     * الموقعَ لخدمةٍ أماميّةٍ من هذا النوع أُطلقت والتطبيقُ ظاهرٌ بلا
     * إذن الخلفيّة.** **فالمسارُ الطبيعيُّ يعمل** — **ويبقى الإذنُ
     * مهمّاً للتعافي بعد موت العمليّة** (`START_STICKY` يُعيدها
     * والتطبيقُ غيرُ مرئيّ) — **فيُقال ولا يُمنَع.**
     *
     * **والإشعارُ: يُمنَع به الإعلانُ عن التوفّر.**
     * **وقِيس**: **كشفُ الطلبات عندنا دفعٌ من `FCM`**
     * (`RahalPushService.onMessageReceived`) — **وكلُّ ما تفعله
     * الدالّةُ أن تبني إشعاراً** (`manager.notify`): **لا تُنعش حالاً
     * في التطبيق ولا تُبلّغ بطريقٍ آخر.**
     *
     * **فالإذنُ مرفوضٌ ⇒ تصل الرسالةُ ويُسقطها النظامُ صامتاً** —
     * **والجوّالُ في الجيب فلا يرى شيئاً**: **يُكتشَف الطلبُ بفتح
     * التطبيق وسحبِه باليد.**
     *
     * **فليس زينةً كما قلتُ في تقريري السابق** — **بل هو طريقُ
     * العمل نفسُه.** **ويُمنَع به `CAN_GO_ONLINE` وحدَها** —
     * **ورحلةٌ في يده يعرفها فلا تُوقَف.**
     */
    fun allows(s: State, level: Level): Boolean {
        // **ولا موقعَ يُنتَج ⇒ لا عملَ من أيّ درجة.**
        val located = !s.blockers.contains(Blocker.LOCATION_PERMISSION_REQUIRED) &&
            !s.blockers.contains(Blocker.LOCATION_SERVICE_OFF)
        return when (level) {
            Level.APP_USABLE -> true
            Level.CAN_RUN_ACTIVE_DELIVERY -> located
            Level.CAN_GO_ONLINE ->
                located && !s.blockers.contains(Blocker.NOTIFICATION_PERMISSION_REQUIRED)
        }
    }

    /** **وبقراءة النظام.** */
    fun allows(context: Context, level: Level): Boolean = allows(of(context), level)

    /**
     * **وما يُنتج موقعاً** — **إذناً وخدمةً معاً.**
     *
     * **ويبقى معنىً قائماً**: **متابعةُ رحلةٍ في اليد.**
     */
    fun canWork(context: Context): Boolean = canWork(of(context))
}
