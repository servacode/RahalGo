package com.rahalgo.driver.trip

import com.rahalgo.navigation.TripMap
import com.rahalgo.navigation.MarkerIcons
import androidx.compose.animation.AnimatedVisibility
import com.rahalgo.ui.Countdown
import androidx.compose.foundation.layout.IntrinsicSize
import com.rahalgo.design.Rahal
import com.rahalgo.ui.etaText
import com.rahalgo.ui.minutesShort
import com.rahalgo.ui.dist
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.material3.Surface
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.layout.width
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.driver.BuildConfig
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.RahalTextButton
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرحلة — خريطة وشريط وبطاقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفة المالك ٢٠٢٦-٠٨-١٢، `docs/DRIVER-APP-PLAN.md` §٣ب.)
 *
 * **ثلاث طبقات**: شريط خطّة السير فوق كلّ شيء · الخريطة · بطاقة سفليّة
 * فيها التفاصيل والأزرار.
 *
 * # ولماذا الشريط فوق
 *
 * **السائق ينظر إلى الشاشة ثانيةً واحدةً وهو واقف** — والشريط يقول أين
 * هو من الرحلة بلا قراءة: **قبلت ← إلى المتجر ← وصلت ← استلمت ← إلى
 * الزبون ← وصلت ← سلّمت.**
 *
 * # والوجهة تتبدّل بنفسها
 *
 * **عند «استلمت الطلب» تصير الوجهة الزبون** — لا يضغط شيئا. (مواصفة
 * المالك: «هون بشكل تلقائي الرحلة تتحوّل إلى الزبون».)
 */
@Composable
fun TripScreen(
    state: TripState,
    actions: TripActions,
    /** **الحديث المفتوح** — وفارغٌ يعني لوحا مطويّا. */
    chat: ChatState? = null,
    chatActions: ChatActions = ChatActions(send = {}, close = {}),
    /** **كم رسالةً تنتظره** — وصفرٌ يعني لا شارة. */
    chatUnread: Int = 0,
    /**
     * **بابُ إعادة الحساب** — وفارغٌ يعني «لا إعادة».
     *
     * **ويُبنى في الـViewModel** حيث يُعرف الطلبُ والخادم.
     */
    routeSource: com.rahalgo.navigation.RouteSource? = null,
    /**
     * **منسّقُ الكلام** — وفارغٌ يعني «لا إرشادَ صوتيّ».
     *
     * **ويُبنى في الـViewModel** فيعيش عبرَ إعادةِ إنشاء الشاشة.
     */
    voice: VoiceOrchestrator? = null,
    /**
     * **جلسةُ الملاحة** — **تُمرَّر ولا تُبنى هنا.**
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «المفروض الرحلة تبقى مستمرّة مهما
     *  حصل وأينما ذهب».)
     *
     * **وكانت تُبنى في التركيب فتموت بمغادرة الشاشة** — وقِيس أنّ
     * فتحَ تبويب «الطلبات» يُغلقها، **وأنّها لا تعود عند الرجوع.**
     * **وهي الآن في `OrdersViewModel`** فتنجو من التبويب ومن دورانِ
     * الجهاز ومن إعادةِ بناء الشاشة.
     */
    navSession: com.rahalgo.navigation.NavigationSession,
    /** **أالملاحقةُ تعمل؟** — من نموذج العرض، فتنجو من إعادة البناء. */
    following: Boolean = false,
    /** **ويُقلبها فعلُ إنسان** — لا أثرُ تركيب. */
    onFollow: (Boolean) -> Unit = {},
    /** **الرحلةُ التجريبيّة** — وقائمةٌ فارغةٌ تعني «أوقفها». */
    onReplay: (List<com.rahalgo.navigation.NavFix>) -> Unit = {},
    /**
     * **أالصوتُ مكتوم؟** — (طلبُ المالك ٢٠٢٦-٠٨-٢٤: «تأكّد من زرّ
     * الصوت بحيث يستجيب بشكلٍ فوريّ».)
     *
     * **ويُمرَّر ولا يُقرأ من `VoiceOrchestrator.muted`** — **ذاك حقلٌ
     * عاديٌّ لا تراقبه الواجهة**، فيُكتم الصوتُ فوراً **وتبقى الأيقونةُ
     * على حالها** حتّى يُعيد شيءٌ آخرُ رسمَ الشاشة. **فيُضغط الزرُّ
     * فيُظنّ أنّه لم يعمل.**
     *
     * **وهذا حالٌ في نموذج العرض** — تراقبه الواجهةُ فيتبدّل في الإطار
     * التالي.
     */
    voiceMuted: Boolean = false,
) {
    val order = state.order
    if (order == null) {
        NoTrip(onOrders = actions.toOrders)
        return
    }

    // ══════════════════════════════════════════════════════════════════
    // **ثلاثة أزرارٍ على الخريطة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة.)
    //
    //	ردَّني    ←  الكاميرا تعود إلى موضعي بعد أن قلّبتُ الخريطة بيدي
    //	اتبعني   ←  تلاحقني وأنا أسير، فلا أمسّها كلَّ دقيقة
    //	الملاحة  ←  أخرج إلى تطبيق الملاحة — الطريق والصوت ليسا عندنا
    //
    // **والملاحقة مُطفأةٌ حتّى تُطلب**: الفتحةُ الأولى تُظهر النقاط
    // كلَّها — **من رأى نفسَه ولم ير وجهتَه** لا يعرف أيّ جهةٍ يمضي.
    // **و«اتبعني» يُقرأ من نموذج العرض** — انظر `OrdersViewModel.following`:
    // **حالٌ تحكم شيئاً باقياً لا تعيش في شاشةٍ تُبنى وتُهدم.**
    val follow = following
    var recenter by rememberSaveable { mutableIntStateOf(0) }

    // ══════════════════════════════════════════════════════════════════
    // **و«اتبعني» هو بوّابةُ الملاحة — لا زرَّ ثانٍ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (المرحلة ١، بأمر المالك ٢٠٢٦-٠٨-٢٠: «عندما يكون السائق داخل
    //  جلسة رحلة نشطة».)
    //
    // **وزرّان لفعلٍ واحدٍ يسألان صاحبَهما أيَّهما يضغط** — والزرُّ
    // قائمٌ منذ ٢٠٢٦-٠٨-١٢ ومعناه «لاحقني وأنا أسير». **وهذا هو
    // معنى الملاحة بعينه.**
    //
    // **والجلسةُ تُفتح وتُغلق معه** — فمن أطفأه عاد الجهازُ إلى
    // ورديّته العاديّة في اللحظة: **بطّاريّةٌ لا تُستنزف في جيبٍ
    // واقف.**
    val navContext = LocalContext.current
    // ══════════════════════════════════════════════════════════════════
    // **ومسجّلُ الرحلة في بناء التطوير وحدَه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «اجعله خاصّاً ببناء Debug/QA قدر
    //  الإمكان، وليس تسجيلَ GPS دائماً في نسخة الإنتاج».)
    //
    // **وتسجيلُ مسارِ كلّ سائقٍ في كلّ رحلةٍ ليس فحصاً بل تتبّعا** —
    // **وملفٌّ يمتلئ في جيب من لا يعلم به لا يُبرَّر بأنّه محلّيّ.**
    //
    // **والقرارُ هنا لا في الوحدة**: الوحدةُ تقبل مسجّلاً أو لا تقبل،
    // **والتطبيقُ وحدَه يعرف أيَّ بناءٍ هو.**
    // **ولا `LaunchedEffect` تُشغّل وتُطفئ** — **إعادةُ بناء الشاشة
    // تُعيد تنفيذَها بقيمةٍ مُصفَّرةٍ فتُطفئ ملاحةً تعمل.** والقرارُ
    // في `onFollow` وحدَه: **فعلُ إنسانٍ لا أثرُ تركيب.**
    // ══════════════════════════════════════════════════════════════════
    // **والمسارُ يُسلَّم للجلسة — لا تحسبه الشاشة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (المرحلة ٣أ، أمرُ المالك: «ولا تجعل UI نفسَها تحسب
    //  `RouteProgress` أو `OffRoute`».)
    //
    // **ويُنادى حين يتبدّل المسارُ وحدَه** — لا في كلّ رسم: **مسارٌ
    // يُسلَّم مرّتين يمحو تقدّمَ السائق مرّتين.**
    // **وعند تبدّل الطور أيضاً** (البند ١٩): من استلم البضاعةَ
    // **تتبدّل وجهتُه، فيزيد الجيلُ ويُطرح جوابٌ قديمٌ في الطريق.**
    /**
     * **سياقُ اللحظة** — يُحسب من الجلسة الحيّة لا يُخزَّن.
     *
     * **والحالُ الباقي في نموذج العرض** (البند ١٢)، **وهذا مشتقٌّ منه
     * ومن الملاحة** — فلا يضيع بالتدوير ولا يُكرَّر تخزينُه.
     */
    val routeTarget = if (state.step >= TripStep.PICKED_UP) {
        com.rahalgo.navigation.RouteTarget.DROPOFF
    } else {
        com.rahalgo.navigation.RouteTarget.PICKUP
    }

    /**
     * **تركيبُ المسار الموصى به** — كما كان.
     *
     * **وسببُه يُعلَن** (البند ٧): **تبدّلُ الطور وجهةٌ جديدة، وما
     * عداه أوّلُ مسارٍ للساق.** **واختيارُ السائق لا يمرّ من هنا** —
     * يمرّ من `state.committedRoute` ويُعلن سببَه بنفسه.
     */
    /**
     * ══════════════════════════════════════════════════════════════
     * **والهدفُ التجاريّ يُسلّم مع المسار**
     * ══════════════════════════════════════════════════════════════
     *
     * (إغلاقُ نقطة الالتقاط، ٢٠٢٦-٠٨-٢١.)
     *
     * **وهو من القاعدة لا من المحرّك**: `navLat/navLng` للمتجر،
     * و`lat/lng` للزبون — **وهي نفسُ ما تقيس عليه `near()`**،
     * فلا رقمان لحقيقةٍ واحدة.
     */
    LaunchedEffect(state.order?.id, state.step, state.order?.lat, state.order?.lng) {
        val o = state.order
        navSession.setArrivalTarget(
            when {
                o == null -> null
                state.step >= TripStep.PICKED_UP ->
                    com.rahalgo.navigation.GeoPoint(o.lat, o.lng)
                o.navLat != null && o.navLng != null ->
                    com.rahalgo.navigation.GeoPoint(o.navLat!!, o.navLng!!)
                else -> null
            },
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════
     * **وطلبُ الارتباط يُرسَل من هنا — المرحلة ٨ب**
     * ══════════════════════════════════════════════════════════════
     *
     * **و`driver-navigation` لا تعرف شبكة**: المُحلِّلُ يُخرج طلباً،
     * **والشاشةُ ترسله وتُعيد الحكم.**
     *
     * **ولا إعادةَ حسابٍ من هذا الردّ** (البند ٢١): الحكمُ يدخل
     * المُحلِّلَ، **وحالُه وحدَها تُنتج سبباً.**
     */
    val correlationAsk = navSession.correlationRequest
    LaunchedEffect(correlationAsk) {
        val ask = correlationAsk ?: return@LaunchedEffect
        actions.askCorrelation(
            ask.routeId,
            ask.fixes.map {
                com.rahalgo.shared.model.CorrelationFix(
                    lat = it.lat, lng = it.lng,
                    accuracyM = it.accuracyM.toDouble(), atMs = it.atMs,
                )
            },
        ) { navSession.onCorrelation(it) }
    }

    LaunchedEffect(state.navRoute, state.step) {
        navSession.setRoute(state.navRoute)
        // **والهويّةُ تُسلَّم مع المسار** — المرحلة ٨ب.
        navSession.routeId = state.navRouteId
        if (state.navRoute != null) {
            actions.onRouteInstalled(
                navSession.generation,
                routeTarget,
                com.rahalgo.navigation.RouteInstallReason.INITIAL,
            )
        }
    }
    // ══════════════════════════════════════════════════════════════════
    // **وطرفُ الرحلة يُحقَن — ولا تعرف الملاحةُ طلبا**
    // ══════════════════════════════════════════════════════════════════
    //
    // (المرحلة ٤، البند ٢٥.)
    LaunchedEffect(state.step) {
        navSession.voicePlanner?.target = if (state.step >= TripStep.PICKED_UP) {
            com.rahalgo.navigation.TripTarget.DROPOFF
        } else {
            com.rahalgo.navigation.TripTarget.PICKUP
        }
    }
    // **وما قرّره المخطِّطُ يُسلَّم للمنسّق** — والشاشةُ ناقلٌ لا حاكم.
    LaunchedEffect(navSession.nav?.let { it to navSession.route }) {
        val nav = navSession.nav ?: return@LaunchedEffect
        // **ولا تنقل الشاشةُ التعليمات** — تنتقل من الجلسة إلى
        // المنسّق مباشرةً (`OrdersViewModel`): **وناقلٌ في شاشةٍ
        // ينقطع بفتح تبويب.**
    }
    // **ونهايةُ الملاحة تُنهي الصوت** — لا نهايةُ ظهورِ الشاشة.
    //
    // (أمرُ المالك، البند ١٩: «الصوتُ مرتبطٌ بـNavigation lifecycle
    //  وليس Activity visibility».)
    DisposableEffect(Unit) { onDispose { voice?.stop() } }
    // **ومغادرةُ الشاشة تُغلقها** — ومن خرج ونسي زرَّه ترك محرّكَ
    // موقعٍ يعمل بلا شاشةٍ تقرؤه.
    // **ولا تُغلق الملاحةُ بمغادرة الشاشة** — بلاغُ المالك
    // ٢٠٢٦-٠٨-٢٤. **وتُغلق بانتهاء الطلب** (`OrdersViewModel.closeNav`)
    // **أو بموت نموذج العرض** — وذاك مغادرةٌ حقّاً.

    // ══════════════════════════════════════════════════════════════════
    // **والنافذةُ الطافيةُ تُفتح للملاحة وحدَها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٤: «ما لازم تتوقّف الخريطة».)
    //
    // **ومغادرةُ الشاشة تُطفئ العلم** — **ومن تركه مرفوعاً فتح نافذةً
    // طافيةً بعد أن أغلق السائقُ رحلتَه**، وهي تبقى على شاشته حتّى
    // يطردها بيده.
    // **والنشاطُ يُطلب من السياق ولو لُفّ** — `ContextWrapper` طبقاتٌ
    // في بعض السمات، **ومن قارن مرّةً واحدةً ردّ فارغاً بلا سبب.**
    val host = remember(navContext) {
        var c: android.content.Context? = navContext
        while (c is android.content.ContextWrapper && c !is com.rahalgo.driver.MainActivity) {
            c = c.baseContext
        }
        c as? com.rahalgo.driver.MainActivity
    }
    val floating = host?.floating
    // **وحضورُ شاشة الرحلة يُعلَن ويُسحب** — فاعتراضُ الرجوع لها
    // وحدَها، **ومن اعترضه في كلّ شاشةٍ حبس صاحبَه في «حسابي».**
    DisposableEffect(Unit) {
        host?.onTripScreen = true
        onDispose { host?.onTripScreen = false }
    }
    DisposableEffect(navSession.running) {
        host?.onNavigatingChanged(navSession.running)
        // **ولا يُطفأ عند مغادرة الشاشة** — الملاحةُ تعمل والتبويبُ
        // نظرة. **ويُطفأ بانتهاء الطلب** (`OrdersViewModel.closeNav`).
        onDispose { }
    }
    val inPip = floating?.inPip == true


    // ══════════════════════════════════════════════════════════════════
    // **اختيارُ المسار — إغلاقُ واجهة ٧، ٢٠٢٦-٠٨-٢١**
    // ══════════════════════════════════════════════════════════════════

    /**
     * ══════════════════════════════════════════════════════════════
     * **وحالُ الملاحة صحيح؟** — إغلاقُ صحّة الواجهة، البند ١٠
     * ══════════════════════════════════════════════════════════════
     *
     * **كان يُبنى على `offRoute` وحدَها وهو ناقص**: كاشفُ الاتّجاه
     * المعاكس **يقيس نسبةً إلى المسار**، **فقد يكون السائقُ داخلَ
     * الممرّ تماماً وهو يسير عكسَه** — `offRoute` تقول `ON_ROUTE`
     * والحالُ `WRONG_WAY`.
     *
     * **فصار من `NavSituation`** — الحالُ المحسومةُ من المرحلة ٥،
     * **وفيها الشكوكُ أيضاً تمنع.**
     */
    val navHealthy = com.rahalgo.navigation.RouteChoiceHealth.of(navSession.nav)

    val progressM = navSession.nav?.progress?.progressM ?: -1.0

    val choiceCtx = com.rahalgo.navigation.RouteChoiceMachine.Context(
        generation = navSession.generation,
        target = routeTarget,
        originLat = state.driver?.latitude ?: 0.0,
        originLng = state.driver?.longitude ?: 0.0,
        progressM = progressM,
        ageMs = 0L,
        healthy = navHealthy,
    )

    /**
     * **وما يُعرض** — يُحسب من الخيارات والسياق.
     *
     * **والبدائلُ الميّتةُ تختفي فرديّاً** (البند ٢٠).
     */
    val choiceUi = com.rahalgo.navigation.RouteChoiceMachine.present(
        state.routeChoices, choiceCtx,
    ).let { base ->
        when {
            base !is com.rahalgo.navigation.RouteChoiceUi.Available -> base
            state.choiceStale -> com.rahalgo.navigation.RouteChoiceUi.Stale(base.choices)
            state.choicePreview != null &&
                base.choices.alternatives.any { it.routeId == state.choicePreview } ->
                com.rahalgo.navigation.RouteChoiceUi.Previewing(
                    base.choices, state.choicePreview,
                )
            else -> base
        }
    }

    /**
     * **طلبُ البدائل بأحداثٍ لا في حلقة** — البندان ٢ و٤.
     *
     * **والملاحةُ لا تنتظرها**: `state.navRoute` تُسلَّم أعلاه،
     * **وهذه بعدها.**
     *
     * **والمفتاحُ الطورُ والجيل** — فتُطلب عند بدء ساقٍ نحو المتجر
     * وبعد الاستلام، **ولا تُطلب مع كلّ نبضةِ موقع.**
     */
    LaunchedEffect(state.order?.id, state.step, navSession.generation, navHealthy) {
        val current = navSession.route
        val driver = state.driver
        val reason = state.installReason

        /**
         * **والقرارُ من موضعٍ واحد** — البندان ٦ و٨.
         *
         * **وفيه أربعةُ شروط**: مسارٌ مركّب، وحالٌ صحيح،
         * وموضعٌ معلوم، **وسببٌ ليس اختيارَ السائق.**
         *
         * **فجيلٌ سبّبه اختيارُه لا يُطلق جلباً** — ولا تعود
         * اللوحةُ فورَ ما اختار.
         */
        val fetch = com.rahalgo.navigation.RouteChoiceGate.shouldFetch(
            hasRoute = current != null,
            healthy = navHealthy,
            hasOrigin = driver != null,
            reason = reason,
        )
        if (!fetch || current == null || driver == null) return@LaunchedEffect

        actions.loadAlternatives(
            navSession.generation,
            routeTarget,
            driver.latitude,
            driver.longitude,
            reason,
            com.rahalgo.navigation.RouteFingerprint.of(current),
            current.geometry,
        )
    }

    /**
     * **وتبدّلُ الوجهة يمسح فوراً** — البند ٢٢.
     *
     * **بدائلُ إلى المتجر لا تصلح بعد الاستلام** — ولا يبقى خطُّها
     * على الخريطة.
     */
    LaunchedEffect(routeTarget) { actions.clearChoices() }

    // ══════════════════════════════════════════════════════════════════
    // **الوصولُ يُسجَّل وحدَه — ولا زرَّ يُضغط**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «المقصودُ تقليلُ الأزرار والضغطِ
    //  عليها» — عند المتجر وعند الزبون كليهما.)
    //
    // # وكان الحارسُ مبنيّاً ولا أحدَ يسمعه
    //
    // **`ArrivalGuard` كاملٌ ومختبَرٌ منذ ٢٠٢٦-٠٨-٢١** بثلاث حالاتٍ
    // وحدودٍ ضبطها المالك (١٥ متراً). **وصفرُ استعمالٍ في هذا
    // التطبيق**: يقول «وصل» ولا شيء يقرؤه إلّا الصوتُ وكاشفُ
    // الشوارع المتوازية.
    //
    // # ولا «تراجع» — لأنّ الطريقَ في اتّجاهٍ واحد
    //
    // **جدولُ المحرّك** (`orders/statuses.go`) لا يعرف
    // `at_pickup → assigned` ولا `at_dropoff → on_the_way`.
    // **فشريطُ تراجعٍ يَعِد بما لا يستطيع.**
    //
    // **فالحمايةُ في شرط الإطلاق لا في زرٍّ بعده** — ثلاثةٌ معاً:
    //
    //   ١ · دقّةٌ ≤ ٣٠ م ونصفُ قطرٍ ١٥ م   ← داخلَ `ArrivalGuard`
    //   ٢ · **مكثٌ لا لحظة**              ← هنا
    //   ٣ · حالٌ مطابقٌ تماماً             ← هنا
    //
    // **والمكثُ هو الحارسُ الذي لا يملكه `ArrivalGuard`**: من مرّ
    // بالمتجر عابراً في طريقه إلى غيره يلمس الخمسةَ عشرَ متراً
    // ثانيةً واحدة، **فيُسجَّل وصولٌ لم يقع ولا سبيلَ لردّه.**
    //
    // # ولا يُسجَّل وصولٌ للمتجر في الطلب الخاصّ
    //
    // (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «ما في شيء اسمه وصلتُ للمتجر».)
    // **والمحرّكُ يرفضه أصلاً** — فالحارسُ هنا يمنع نداءً يُردّ.
    val arrived = navSession.nav?.arrivedAtTarget == true
    val autoStatus = autoArrivalTarget(
        status = state.order?.status.orEmpty(),
        custom = state.order?.kind == "custom",
    )
    // **ومرّةً واحدةً لكلّ طور** — والمفتاحُ الحالُ نفسُه:
    // **فمن سُجّل وصولُه إلى المتجر لا يُعاد تسجيلُه**، ووصولُ
    // الزبون طورٌ آخرُ بمفتاحٍ آخر.
    val autoFired = rememberSaveable { mutableStateOf("") }
    LaunchedEffect(arrived, autoStatus, state.busy) {
        val to = autoStatus ?: return@LaunchedEffect
        if (!arrived || state.busy || autoFired.value == to) return@LaunchedEffect
        // **والمكثُ ستُّ ثوانٍ** — **ودونها يكفي أن يقف عند إشارةٍ
        // أمام المتجر ليُحسب واصلا.**
        //
        // **ويُعاد الفحصُ بعدها**: `arrived` يتبدّل مع كلّ قراءة،
        // **فمن ابتعد أثناء المكث أُلغيت الدورةُ** — لأنّ
        // `LaunchedEffect` يُقتل عند تبدّل مفتاحه.
        kotlinx.coroutines.delay(6_000)
        if (navSession.nav?.arrivedAtTarget != true) return@LaunchedEffect
        autoFired.value = to
        android.util.Log.i("RahalGo/nav", "وصولٌ تلقائيّ → $to")
        actions.step(to)
    }

    /**
     * **وتسليمُ المعتمد** — البند ١٧.
     *
     * **بعد الفحص وحدَه**: نموذجُ العرض يفحص ثمّ يضع، **والشاشةُ
     * تسلّم.** و`setRoute` تزيد الجيلَ **فتنتهي المجموعةُ بنيويّاً.**
     */
    LaunchedEffect(state.committedRoute) {
        val committed = state.committedRoute ?: return@LaunchedEffect
        // **ولا يُعاد تركيبُ مسارٍ مُركَّب** — **وإعادةُ بناء الشاشة
        // تُعيد تنفيذَ هذا الأثر**، فيقفز الجيلُ (قِيس ٢٠٢٦-٠٨-٢٤:
        // ١←٢ عند العودة من تبويب) **ويُنسى ما قيل فتُعاد التعليماتُ
        // من أوّلها.**
        // **والمقارنةُ بالهُويّة لا بالمرجع** — **والمسارُ يُبنى كائناً
        // جديداً مع كلّ قراءةِ حالة**، فالمرجعُ يختلف والمسارُ هو هو.
        if (navSession.route?.let { it.geometry == committed.geometry } == true) {
            return@LaunchedEffect
        }
        navSession.setRoute(committed)
        // **ويُعلَن أنّ الجيلَ من اختيار السائق** — فلا جلبَ بعده.
        actions.onRouteInstalled(
            navSession.generation,
            routeTarget,
            com.rahalgo.navigation.RouteInstallReason.USER_SELECTION,
        )
        actions.onRouteCommitted()
    }

    Box(Modifier.fillMaxSize()) {
        TripMap(
            // **وأيقوناتُ التطبيق تُمرَّر** — نسخُها تختلف عن نسخِ
            // `:ui`، **فالنقلُ لا يغيّر ما يُرى.** (المرحلة ٠.)
            icons = MarkerIcons(
                driver = R.drawable.ic_moto,
                pickup = R.drawable.ic_store,
                dropoff = R.drawable.ic_pin,
            ),
            driver = state.driver,
            // **وقبل الاستلام تُعرض النقطتان** — بعده تُطفأ نقطة المتجر:
            // **انتهى شأنه منها**، وخريطة فيها ما لم يعد يلزم تشوّش.
            pickup = if (state.step >= TripStep.PICKED_UP) null else state.pickup,
            // ══════════════════════════════════════════════════════════
            // **وموضعُ الزبون لا يُرسم قبل الاستلام**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «موقع الزبون ما يلزمنا بالرحلة
            //  القسم الأوّل».)
            //
            // **والرحلةُ طوران، ولكلّ طورٍ وجهةٌ واحدة**: نقطتان على
            // الشاشة تجعلان الكاميرا تتّسع لتضمّهما، **فيصغر الشارعُ
            // الذي يسير فيه الآن** ليُرى مكانٌ لا شأنَ له به بعد.
            dropoff = if (state.step >= TripStep.PICKED_UP) state.dropoff else null,
            route = state.routeLine,
            follow = follow,
            recenter = recenter,
            // **وما تُرسمه الملاحةُ حين تعمل** — وفارغٌ يعني «الخريطةُ
            // كما كانت حرفاً بحرف».
            nav = navSession.render,
            // **البدائلُ تُرسم ولا يُلاحَ عليها** — البند ٢٨ من ٧.
            alternatives = choiceUi.choicesOrNull?.alternatives.orEmpty().map {
                com.rahalgo.navigation.AltRouteLayer.Drawable(it.routeId, it.route)
            },
            previewRouteId = choiceUi.previewRouteId,
            // **ولمسُ الخطّ يُعاين ولا يعتمد** — البندان ٢٤ و٢٧.
            onRouteTapped = { routeId -> actions.previewRoute(routeId) },
            modifier = Modifier.fillMaxSize(),
        )

        // ══════════════════════════════════════════════════════════════
        // **ولا شريطَ خطواتٍ فوق الخريطة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «بالرحلة الشريط العلويّ تبع الرحلة
        //  ألغه».)
        //
        // **وسبعُ خطواتٍ تُقرأ في كلّ نظرة** وهو لا يحتاج منها إلّا
        // واحدة: **ما الذي أفعله الآن** — وهي مكتوبةٌ في الزرّ أسفل
        // الشاشة بلفظها. **والباقي تاريخٌ ومستقبلٌ يزاحمان الخريطة.**
        // ══════════════════════════════════════════════════════════════
        // **وفي النافذة الطافية لا يُعرض إلّا لوحُ الطور**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-٢٤.)
        //
        // **ومربّعٌ من بضعة سنتيمترات لا يتّسع لأزرارٍ ولا لحديث** —
        // **واللمسُ لا يصل نافذةً طافيةً أصلاً**، فزرٌّ فيها خدعةٌ
        // تُضغط ولا تعمل.
        //
        // **والخريطةُ والتعليمةُ التاليةُ هما ما يُقرأ في نظرة.**
        // **ولا لوحَ طورٍ في النافذة الطافية** — (بلاغُ المالك
        // ٢٠٢٦-٠٨-٢٤: «شريط تبع الرحلة لا يلزم أيضاً بالنافذة
        // المصغّرة»). **الخريطةُ وحدَها تقول له طريقَه.**
        if (!inPip) Column(
            Modifier.align(Alignment.TopCenter).statusBarsPadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            // ══════════════════════════════════════════════════════════
            // **لوحُ الطور — إلى أين وكم بقي**
            // ══════════════════════════════════════════════════════════
            //
            // (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة: لوحٌ داكنٌ فوق الخريطة
            //  يقول «في الطريق إلى المتجر» وتحته المسافة والدقائق.)
            //
            // **والرحلة طوران**: إلى المتجر ثمّ — بعد الاستلام — إلى
            // الزبون. **ومن لا يعرف في أيّهما هو** يقرأ المسافة ولا
            // يعرف إلى أين هي.
            //
            // **وهو فوق الخريطة لا تحتها**: عينُه على الطريق، **وما
            // يُقرأ في نظرةٍ خاطفةٍ يكون في أعلى الشاشة** حيث لا يحجبه
            // إبهامٌ ولا يُطلب منه أن ينزل بعينه إلى أسفلها.
            // ══════════════════════════════════════════════════════════
            // **لوحُ الرحلة — طورُها ومراحلُها في أرضٍ واحدة**
            // ══════════════════════════════════════════════════════════
            //
            // (مواصفة المالك ٢٠٢٦-٠٨-١٢ بصورة.)
            //
            // **والمراحلُ تبقى والطورُ يغيب**: من وقف عند الباب لا يقرأ
            // «الطريق إلى ابوطيف» ولا مسافةً ولا زمنا — **وهو واقفٌ
            // فيه** — **لكنّه يبقى يريد أن يعرف أين صار من الرحلة.**
            //
            // **وأرضٌ واحدةٌ لهما لا لوحان**: لوحان فوق خريطةٍ يقضمان
            // ثلثَها، **وأحدُهما يختفي فيترك فراغاً معلّقا.**
            TripPanel(state)

            // ══════════════════════════════════════════════════════════
            // **ومن يحمل أكثر من طلب يرى محطّاته**
            // ══════════════════════════════════════════════════════════
            //
            // (البند العاشر في قائمة المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **والمعروض هو «التالي»** — والباقي يُضغط فيصير هو التالي:
            // **من حمل ثلاثة ولا يعرف أيّها أوّلا** يقرّر بالحدس، ويقف
            // في الشارع يقلّب.
            if (state.stops.size > 1) {
                StopsRow(stops = state.stops, current = order.id, onPick = actions.pickStop)
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **طلبٌ على طريقك — وأنت ماشٍ**
        // ══════════════════════════════════════════════════════════════
        //
        // (البند الحادي عشر في قائمة المالك ٢٠٢٦-٠٨-١٢.)
        //
        // **والمحرّك يحسبه أصلا** (`orders/sameroute.go`): المتجران خلال
        // ٨٠٠ متر والزبونان خلال ٢٠٠٠ — **فما يصل الطابور وأنت في رحلة
        // هو على طريقك فعلا.**
        //
        // **ولافتةٌ لا شاشة**: يقرؤها بطرف عينه وهو يقود، **ويأخذها أو
        // يتركها — وهو حرّ.**
        // **وأزرارُ الرحلة تُطوى في النافذة الطافية** — انظر أعلاه.
        if (!inPip) Column(Modifier.align(Alignment.BottomCenter)) {
            // ══════════════════════════════════════════════════════════
            // **لوحةُ اختيار المسار** — فوق عناصر التحكّم (البند ٦)
            // ══════════════════════════════════════════════════════════
            //
            // **ولا تُعاد الشاشةُ تصميماً** (البند ٤٠): عنصرٌ مستقلٌّ
            // صغير، **وما تحته لم يُمَسّ.**
            RouteChoicePanel(
                ui = choiceUi,
                onSelect = { actions.previewRoute(it) },
                onCancel = { actions.previewRoute(null) },
                onConfirm = {
                    val driver = state.driver
                    if (driver != null) {
                        actions.confirmRoute(
                            navSession.generation,
                            routeTarget,
                            driver.latitude,
                            driver.longitude,
                            progressM,
                            navHealthy,
                        )
                    }
                },
            )

            // ══════════════════════════════════════════════════════════
            // **وموضعُها فوق اللوح لا فوق شريط الخطوات**
            // ══════════════════════════════════════════════════════════
            //
            // (شكوى المالك ٢٠٢٦-٠٨-١٣ بلقطةٍ من جهازه: «الرسالةُ بمكانٍ
            //  غير مناسب».)
            //
            // **كانت تحت الشريط العلويّ فتحجب خطواتِ رحلته** — الطورَ
            // الذي هو فيه وما بقي منه. **ومن يقود يقرأ حالَه من هناك.**
            //
            // **والإبهامُ في أسفل الشاشة أصلاً** — على «اشتريتُ الطلب»
            // و«لدي مشكلة». **وقرارٌ عاجلٌ يُطلب في أعلى الشاشة يحتاج
            // يداً تترك المقود.**
            if (state.onRouteOffer != null) {
                OnRouteBanner(
                    offer = state.onRouteOffer,
                    busy = state.busy,
                    // **وجوابُ الضغطة في اللافتة نفسِها** — لا في لوحٍ
                    // مطويٍّ تحتها: **من ضغط ولم يقع شيءٌ يعيد الضغط.**
                    error = state.error,
                    onTake = { actions.takeOffer(state.onRouteOffer.id) },
                    onDismiss = actions.dismissOffer,
                )
            }
            MapButtons(
                follow = follow,
                onRecenter = { recenter++ },
                onFollow = { onFollow(!follow) },
                onChat = actions.chat,
                chatting = chat != null,
                chatUnread = chatUnread,
                onNavigate = actions.navigate,
                voiceMuted = voiceMuted,
                onVoice = { actions.toggleVoice() },
            )

            // ══════════════════════════════════════════════════════════
            // **رحلةٌ تمشي والسائقُ جالس — في بناء التطوير وحدَه**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «تخلّي شاشةَ الرحلة قدّامي
            //  وتشوفني شلون الطلبُ يمشي عالخريطة على المتجر وعلى
            //  الزبون بدون أن أتحرّك أنا».)
            //
            // **والحارسُ `BuildConfig.DEBUG` لا إعدادٌ ولا خاصّيّة** —
            // **زرٌّ يحرّك سهمَ سائقٍ حقيقيٍّ بلا أن يتحرّك هو أخطرُ
            // ما يُترك في يد الإنتاج**: تُسجَّل رحلةٌ لم تقع، ويُسلَّم
            // طلبٌ من غرفة.
            //
            // **ويتبع المسارَ الحاضر لا سيناريو مكتوباً** — فإن كان
            // الطورُ «إلى المتجر» مشى إليه، **وإن ضُغط «استلمتُ»
            // انقلب المسارُ إلى الزبون فمشت الإعادةُ إليه وحدَها.**
            if (BuildConfig.DEBUG) {
                state.routeLine?.let { line ->
                    ReplayButton(
                        running = navSession.replaying,
                        onStart = {
                            // **والهندسةُ من الجلسة إن وُجدت** — وهي
                            // التي أسقط عليها المحرّكُ الخطوات؛
                            // **وخطُّ الشاشة يكفي قبل أن تبدأ.**
                            val g = navSession.route?.geometry
                                ?: line.map {
                                    com.rahalgo.navigation.GeoPoint(it.latitude, it.longitude)
                                }
                            // **ومجالُها في نموذج العرض** — انظر
                            // `OrdersViewModel.startReplay`.
                            onReplay(com.rahalgo.navigation.ReplayDrive.fixes(g))
                        },
                        onStop = { onReplay(emptyList()) },
                    )
                }
            }
            // ══════════════════════════════════════════════════════════
            // **والحديث يحلّ محلّ البطاقة ولا يغطّي الشاشة**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «الدردشة ما تفتح صفحة لحالها،
            //  تكون عائمة مشان ما تغطّي الخريطة — تضغط الأيقونة تفتح،
            //  تضغط ترجع تختفي».)
            //
            // **وشاشةٌ كاملةٌ تسرق الطريق**: يفتحها وهو على إشارةٍ فلا
            // يرى أين هو، **ويبحث عن باب الرجوع** بيدٍ واحدةٍ على مقود.
            //
            // **والأزرارُ فوقه تبقى** — أيقونةُ الحديث هي البابُ نفسُه:
            // **تُضغط فيُفتح وتُضغط فيُطوى.**
            if (chat != null) {
                ChatSheet(state = chat, actions = chatActions)
            } else {
                TripCard(order = order, state = state, actions = actions)
            }
        }
    }

    if (state.agreeOpen) {
        AgreeDialog(onConfirm = actions.agree, onDismiss = actions.dismissAgree)
    }

    if (state.emergencyOpen) {
        EmergencyDialog(onConfirm = actions.emergency, onDismiss = actions.dismissEmergency)
    }

    if (state.failReasons != null) {
        FailDialog(
            reasons = state.failReasons,
            onPick = actions.fail,
            onDismiss = actions.dismissFail,
            onMine = actions.problem,
        )
    }
}

/**
 * **لافتة «طلب على طريقك».**
 *
 * **ولا تحجب الخريطة** — سطران وزرّان، **ومن ملأ الشاشة بعرضٍ وسائقُه
 * يقود** أجبره على قرارٍ في غير وقته.
 */
@Composable
private fun OnRouteBanner(
    offer: DriverOrder,
    busy: Boolean,
    error: String,
    onTake: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.brand)
            .padding(14.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.trip_on_route),
                color = Color.White,
                fontWeight = FontWeight.Bold,
            )
            // ══════════════════════════════════════════════════════════
            // **وعدّادُ المهلة كما في بطاقة الطلب**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يقبل بشكلٍ سريع مع عدّاد وقت».)
            //
            // **والعرضُ ينقضي وإن لم تُعرض مهلتُه** — فمن قرأه ومضى ثمّ
            // عاد ليضغط **وجد الطلبَ ذهب ولم يعرف أنّه كان يسابق.**
            //
            // **وهو العدّادُ نفسُه** (`ui/Countdown.kt`) — لا نسخةٌ
            // ثانيةٌ تفترق في تنسيقها.
            offer.offerExpiresAt?.let {
                Countdown(it, onExpired = onDismiss, onColor = Color.White)
            }
        }
        Spacer(Modifier.height(2.dp))
        Text(
            // **واسمُ المصدر أو «طلب خاصّ»** — لا سطرٌ يبدأ بنقطة.
            text = offer.merchantName.ifBlank { stringResource(R.string.nav_custom_order) } +
                " · " + money(offer.cashDue),
            color = Color.White.copy(alpha = 0.9f),
        )
        Spacer(Modifier.height(10.dp))
        // ══════════════════════════════════════════════════════════════
        // **«موافق» و«لاحقا» — لا «رفض»**
        // ══════════════════════════════════════════════════════════════
        //
        // (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «زرُّ خذ واترك يجب أن يكون موافق
        //  ورفض» — **ثمّ ٢٠٢٦-٠٨-١٤: «السائقُ لا علاقة له بالرفض، هو
        //  إمّا يوافق أو يترك الطلبَ لغيره».)**
        //
        // # وهذا الزرُّ لم يكن رفضاً أصلا
        //
        // **`dismissOffer` إخفاءٌ محلّيٌّ لا نداءٌ للمحرّك** — الطلبُ
        // يبقى في الطابور **وعدّادُه يجري.** فاسمُ «رفض» كذبٌ على
        // صاحبه: **من ضغطه ظنّ أنّه ردّ الطلبَ وهو لم يفعل.**
        //
        // **و«لاحقاً» تقول ما يقع فعلا** — تُزيح الشريطَ عن الطريق وهو
        // يقود، **والطلبُ لمن يأخذه.**
        //
        // **وفعلٌ واحدٌ باسمين في شاشتين يُقرأ فعلين** — من تعلّم
        // «موافق» في الطلبات يتردّد أمام «خذ الطلب» في الرحلة.
        Row(
            Modifier.fillMaxWidth().height(IntrinsicSize.Min),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            RahalButton(
                onClick = onTake,
                enabled = !busy,
                tone = Tone.Success,
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.order_agree)) }
            RahalButton(
                onClick = onDismiss,
                enabled = !busy,
                tone = Tone.Danger,
                modifier = Modifier.weight(1f),
            ) { Text(stringResource(R.string.order_later)) }
        }
        // ══════════════════════════════════════════════════════════════
        // **وجوابُ الضغطة تحتها مباشرة**
        // ══════════════════════════════════════════════════════════════
        //
        // (كشفته دورةٌ حقيقيّةٌ على المحاكي ٢٠٢٦-٠٨-١٣: ضُغط «موافق»
        //  فرُدَّ بـ«معك طلبات بعدد حدّك» — **ولم يظهر شيء.**)
        //
        // **والخبرُ كان يُكتب في اللوح السفليّ وهو مطويّ** — فلا يُقرأ
        // إلّا بسحبه. **وجوابٌ يحتاج بحثاً عنه ليس جوابا.**
        if (error.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = error,
                color = Color.White,
                style = MaterialTheme.typography.bodySmall,
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

/**
 * **محطّاته حين يحمل أكثر من طلب.**
 *
 * **والحاليّة معلّمة** — وما عداها يُضغط فينتقل إليه.
 */
@Composable
private fun StopsRow(stops: List<Stop>, current: String, onPick: (String) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(Rahal.colors.canvas.copy(alpha = 0.94f))
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 12.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        for (stop in stops) {
            val now = stop.id == current
            Text(
                text = "#" + stop.number,
                color = if (now) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                fontWeight = if (now) FontWeight.Bold else FontWeight.Normal,
                modifier = Modifier
                    .clip(Rahal.shape.sm)
                    .background(if (now) Rahal.colors.brand else Rahal.colors.field)
                    .clickable { onPick(stop.id) }
                    .padding(horizontal = 12.dp, vertical = 6.dp),
            )
        }
    }
}

/** محطّة في قائمة من يحمل أكثر من طلب. */
data class Stop(val id: String, val number: Long)

/**
 * **البطاقة السفليّة.**
 *
 * **وما يقرّر به السائق أوّلا**: إلى أين يذهب الآن، وكم يقبض.
 */
@Composable
private fun TripCard(
    order: DriverOrder,
    state: TripState,
    actions: TripActions,
    modifier: Modifier = Modifier,
) {
    // ══════════════════════════════════════════════════════════════════
    // **والبطاقةُ تُطوى وتُسحب**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «الكرت السفليّ لازم يكون قابل للطيّ —
    //  السائق يسحبه وقت يلزمه، لأنّه ما في شي يلزم غير وقت بدّو يسلّم
    //  الزبون».)
    //
    // **وهو يقود**: العنوانُ والمبلغُ خبرٌ لا فعلَ عليه الآن، **والخريطةُ
    // هي ما يقرّر بها.** فتُطوى البطاقةُ إلى سطرٍ واحد.
    //
    // # وتنفتح بنفسها عند الوقوف
    //
    // **وحين يصل يحتاجها كلَّها في اللحظة نفسِها**: العنوانُ ليجد الباب،
    // والمبلغُ ليقبض، والأزرارُ ليُنهي. **ومن طُلب منه أن يسحبها وهو
    // واقفٌ أمام الزبون** يسحبها بيدٍ ويحمل الكيسَ بالأخرى.
    //
    // **ويبقى السحبُ بيده**: من أراد العنوانَ وهو في الطريق سحبها،
    // **وآليّةٌ لا يملك أحدٌ تجاوزَها** تُقرأ عنادا.
    var expanded by rememberSaveable(order.id) { mutableStateOf(false) }
    LaunchedEffect(order.status) {
        expanded = order.status == "at_pickup" || order.status == "at_dropoff"
    }

    Column(
        modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sheet)
            .background(Rahal.colors.canvas)
            // **والسحبُ على البطاقة كلِّها لا على المقبض وحدَه** —
            // **ومقبضٌ بعرض إصبعين** يُخطئه من يقود.
            .pointerInput(Unit) {
                detectVerticalDragGestures { _, dy ->
                    if (dy > 6f) expanded = false
                    if (dy < -6f) expanded = true
                }
            }
            .padding(horizontal = 20.dp, vertical = 12.dp),
    ) {
        // **ومقبضٌ يُرى** — شريطٌ رماديٌّ يقول «هذه تُسحب»، **وبطاقةٌ
        // تُسحب ولا تقول** لا يعرف أحدٌ أنّها تُسحب.
        Box(
            Modifier
                .align(Alignment.CenterHorizontally)
                .clip(Rahal.shape.pill)
                .background(Rahal.colors.inkMuted.copy(alpha = 0.35f))
                .size(width = 44.dp, height = 5.dp)
                .clickable { expanded = !expanded },
        )
        Spacer(Modifier.height(10.dp))

        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // **وأيقونةٌ بجانب الاسم** — (قرار المالك ٢٠٢٦-٠٨-١٢).
            //
            // **وهي تقول أيَّ طورٍ أنت فيه بلا قراءة**: متجرٌ فأنت
            // ذاهبٌ إليه، **ودبّوسٌ فالبضاعةُ معك وأنت إلى الزبون.**
            Row(verticalAlignment = Alignment.CenterVertically) {
                val toCustomer = state.step >= TripStep.PICKED_UP
                Icon(
                    painter = painterResource(
                        if (toCustomer) R.drawable.ic_pin else R.drawable.ic_store,
                    ),
                    contentDescription = null,
                    tint = if (toCustomer) Rahal.colors.brand else Rahal.colors.accent,
                    modifier = Modifier.size(22.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    // **والعنوان يقول الوجهة الحاليّة** — لا اسم الطلب:
                    // **من قرأ «طيف» وهو في طريقه للزبون** قرأ ما مضى.
                    text = if (toCustomer) {
                        order.customerName.ifBlank { stringResource(R.string.detail_customer) }
                    } else {
                        // **ولا اسمَ متجرٍ في الخاصّ** — فيُقال ما هو.
                        order.merchantName.ifBlank { stringResource(R.string.nav_custom_order) }
                    },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
            }
            Text("#${order.number}", color = Rahal.colors.inkMuted)
        }

        // **والعنوانُ والمبلغُ وما بعدهما يُطوى** — سطرُ الوجهة يبقى.
        AnimatedVisibility(visible = expanded) {
            Column {
                Spacer(Modifier.height(4.dp))
                Text(
                    text = if (state.step >= TripStep.PICKED_UP) order.addressText else "",
                    color = Rahal.colors.inkMuted,
                )

        // **وما بقي صعد إلى لوح الطور** — ولا يُكتب هنا ثانية:
        // **رقمان لشيءٍ واحدٍ في شاشةٍ واحدة** يُقرأ أحدهما شيئا آخر.

        // ══════════════════════════════════════════════════════════════
        // **ولا مالَ في الطريق إلى المتجر**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «بالنسبة إلى تقبض نقداً والسعر لا
        //  داعي له، لأنّ السائق لن يدفع ولن يقبض من المتجر أساسا».)
        //
        // **والمبلغُ يُقبض عند باب الزبون** — وذكرُه وهو في طريقه إلى
        // المتجر **سطرٌ لا فعلَ عليه الآن**، ويزاحم ما عليه أن يفعله.
        //
        // **ويظهر حين يصير له معنى**: بعد الاستلام، وهو ماضٍ إلى من
        // يقبض منه.
        if (state.step >= TripStep.PICKED_UP) {
        Spacer(Modifier.height(10.dp))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            // ══════════════════════════════════════════════════════════
            // **والمبلغُ باسم من يُقبض منه**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢: «بدل تقبض نقداً نخلّيها المبلغ
            //  الإجماليّ المطلوب من اسم الزبون».)
            //
            // **و«تقبض نقدا» تصف طريقةَ الدفع** — والسائقُ لا يسأل عن
            // الطريقة، **يسأل: كم آخذُ ومن مَن.** وباسمه يُقرأ الجواب
            // كاملا في سطر.
            //
            // **ومن حمل ثلاثة طلبات** يقرأ ثلاثةَ أسطرٍ متشابهةٍ تقول
            // كلُّها «تقبض نقدا» — **ولا يعرف أيُّها لهذا الباب.**
            val due = order.cashDue > 0
            Text(
                text = if (due) {
                    stringResource(
                        R.string.trip_due_from,
                        order.customerName.ifBlank { stringResource(R.string.detail_customer) },
                    )
                } else {
                    stringResource(R.string.trip_prepaid_note)
                },
                color = Rahal.colors.inkMuted,
            )
            if (due) {
                Text(
                    text = money(order.cashDue),
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.brand,
                )
            }
        }
        }

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Text(state.error, color = Rahal.colors.accent)
        }

        // **واقتراحٌ لا فعل** — الزرّ نفسه تحته، **وهو من يضغطه.**
        if (state.nearDestination) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(
                    if (order.status == "assigned") R.string.trip_near_pickup
                    else R.string.trip_near_dropoff,
                ),
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        }

            // ══════════════════════════════════════════════════════════════
            // **صفٌّ واحدٌ لا ثلاثة — والبطاقةُ تقصر**
            // ══════════════════════════════════════════════════════════════
            //
            // (تصحيح المالك ٢٠٢٦-٠٨-١٢: «أحسّ الكرت عريض أكثر من اللازم،
            //  ليش الأزرار ما حاطهن بشكل أفقي بجانب وصلت للمتجر؟».)
            //
            // **وكلُّ سطرٍ في البطاقة يقضم الخريطة**: هي ما يقود عليه،
            // **والبطاقةُ حاشيةٌ عليها لا العكس.**
            //
            // **والفعلُ الأوّل يأخذ العرض** — هو ما يُضغط في تسعٍ من عشر،
            // **و«لدي مشكلة» قرصٌ بجانبه**: يُعرف بشكله لا بعرضه.
            Spacer(Modifier.height(14.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                val next = nextAction(order.status, order.kind == "custom")
                if (next != null) {
                    SmallAction(
                        icon = R.drawable.ic_check_circle,
                        label = next.label,
                        tone = Tone.Brand,
                        // **والتسليم يمرّ بالصورة إن طلبها المحرّك** — وإلّا
                        // ردّ «يلزم إثبات» بعد أن ظنّ صاحبه أنّه أنهى.
                        onClick = {
                            if (next.status == "delivered" && state.requirePhoto) {
                                actions.capture()
                            } else {
                                actions.step(next.status)
                            }
                        },
                        enabled = !state.busy,
                        busy = state.busy,
                        modifier = Modifier.weight(1f),
                    )
                }

                // ══════════════════════════════════════════════════════════
                // **ولونُ الزرّ يقول ما يفعل قبل أن تُقرأ كلمتُه**
                // ══════════════════════════════════════════════════════════
                //
                // (قرار المالك ٢٠٢٦-٠٨-١٢: «خلّي خلفية الزرّ أحمر ولون الخطّ
                //  أبيض، وأعِد للطابور خلّيه برتقالي بنفس الصفّ مع أيقونة
                //  إعادة».)
                //
                // **أحمرُ صلبٌ يُلمح ولا يُقرأ** — والسائقُ ينظر لحظةً وهو
                // واقف. **وحرفٌ ملوّنٌ على أرضٍ بيضاءَ** يحتاج قراءةً.
                //
                // **والبرتقاليُّ ليس أحمر**: إعادةُ الطلب ليست عطبا — هي
                // خيارٌ مشروع، **ولونُ الخطر عليها** يجعل صاحبَها يتردّد
                // فيمسك طلباً لا يقدر عليه.
                SmallAction(
                    icon = R.drawable.ic_warning,
                    label = R.string.trip_problem,
                    tone = Tone.Danger,
                    onClick = actions.askFail,
                    enabled = !state.busy,
                    modifier = Modifier.weight(1f),
                )

                // ══════════════════════════════════════════════════════════
                // **والإعادةُ في الطريق وحدَه — لا عند باب المتجر**
                // ══════════════════════════════════════════════════════════
                //
                // (قرار المالك ٢٠٢٦-٠٨-١٢: «بما أنّ السائق وصل للمتجر لا
                //  يوجد داعٍ لزرّ أعِد للطابور».)
                //
                // **ومن وصل صار خبرُه خبرا**: المتجرُ مغلقٌ أو الطلبُ غيرُ
                // جاهزٍ أو مشكلةٌ عنده هو — **وكلُّها تُقال بسببها في «لدي
                // مشكلة»**، لا بزرٍّ صامتٍ يُعيد الطلبَ ولا يقول لماذا.
                //
                // **وسببٌ مكتوبٌ فرقُه في المال**: «المتجر مغلق» ذنبُ متجرٍ
                // يُعوَّض عليه السائق، **وإعادةٌ بلا سبب** تُقرأ تردّداً منه.
                if (order.status == "assigned") {
                    SmallAction(
                        icon = R.drawable.ic_undo,
                        // **واللفظُ قصيرٌ هنا** — ثلاثةُ أزرارٍ في صفٍّ
                        // على شاشةِ هاتف، **و«أعد الطلب للطابور» تدفع
                        // الأوّلَ إلى سطرين.** والأيقونةُ تقول «إعادة».
                        label = R.string.trip_release_short,
                        tone = Tone.Accent,
                        onClick = actions.release,
                        enabled = !state.busy,
                        modifier = Modifier.weight(1f),
                    )
            }
        }

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
            // **والملاحة صعدت إلى الخريطة** — قرصٌ برتقاليٌّ بجانب
            // «ردّني» و«تابعني». **وزرّان يفعلان الشيءَ نفسَه في شاشةٍ
            // واحدة** يجعلان صاحبَهما يسأل: أيّهما؟ وهو يقود.
            //
            // **وحديث الزبون صار قرصا عائما على الخريطة** — لا رقم
            // هاتف في الطرفين، **وما يُفتح فجأةً يكون في مرمى الإبهام.**
            // **والاتّفاق للطلب الخاصّ وحدَه** — العاديّ سعرُه معروف
            // سلفا، **وزرٌّ يظهر فيه يسأل عمّا لا يُسأل عنه.**
            if (order.kind == "custom") {
                RahalTextButton(onClick = actions.askAgree, enabled = !state.busy) {
                    Text(stringResource(R.string.agree_button), color = Rahal.colors.accent)
                }
            }
            // **وزرُّ «لدي مشكلة» صعد إلى صفّ الفعل** — بجانب ما يُضغط
            // كلَّ مرّة. **وأسبابُه عند الباب وبلاغُ طارئٍ في الطريق**،
            // ولا يُطلب من صاحبه أن يعرف الفرق.
        }

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Text(state.error, color = Rahal.colors.danger)
        }
            }
        }
    }
}

@Composable
private fun NoTrip(onOrders: () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.trip_none),
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = stringResource(R.string.trip_none_hint),
            color = Rahal.colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(18.dp))
        RahalButton(onClick = onOrders) { Text(stringResource(R.string.nav_orders_all)) }
    }
}
