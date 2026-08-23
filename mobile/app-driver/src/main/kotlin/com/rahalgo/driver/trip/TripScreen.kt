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
    var follow by rememberSaveable { mutableStateOf(false) }
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
    val navSession = remember(routeSource) {
        com.rahalgo.navigation.NavigationSession(
            navContext,
            source = routeSource,
            // **والمخطِّطُ يُبنى مع الجلسة** — حالتُه عبورُ عتباتٍ
            // لا سجِلُّ ما قيل، **وذاك في المنسّق.**
            voice = com.rahalgo.navigation.VoicePlanner(),
            recorder = if (BuildConfig.DEBUG) {
                com.rahalgo.navigation.TraceRecorder(
                    dir = java.io.File(navContext.filesDir, "nav-traces"),
                    sessionId = java.util.UUID.randomUUID().toString().take(8),
                )
            } else {
                null
            },
        )
    }
    LaunchedEffect(follow) {
        if (follow) navSession.start() else navSession.stop()
    }
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
        voice?.offer(nav.cues, nav.progress?.progressM ?: 0.0, navigating = navSession.running)
    }
    // **ونهايةُ الملاحة تُنهي الصوت** — لا نهايةُ ظهورِ الشاشة.
    //
    // (أمرُ المالك، البند ١٩: «الصوتُ مرتبطٌ بـNavigation lifecycle
    //  وليس Activity visibility».)
    DisposableEffect(Unit) { onDispose { voice?.stop() } }
    // **ومغادرةُ الشاشة تُغلقها** — ومن خرج ونسي زرَّه ترك محرّكَ
    // موقعٍ يعمل بلا شاشةٍ تقرؤه.
    DisposableEffect(Unit) { onDispose { navSession.stop() } }

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
    val autoStatus = when {
        state.order?.kind == "custom" -> null
        state.order?.status == "assigned" -> "at_pickup"
        state.order?.status == "on_the_way" -> "at_dropoff"
        else -> null
    }
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
        Column(
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
        Column(Modifier.align(Alignment.BottomCenter)) {
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
                onFollow = { follow = !follow },
                onChat = actions.chat,
                chatting = chat != null,
                chatUnread = chatUnread,
                onNavigate = actions.navigate,
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
                val replayScope = rememberCoroutineScope()
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
                            navSession.startReplay(
                                com.rahalgo.navigation.ReplayDrive.fixes(g),
                                replayScope,
                            )
                        },
                        onStop = { navSession.stopReplay() },
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
 * ══════════════════════════════════════════════════════════════════════
 * **الاتّفاق على الطلب الخاصّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وثمن البضاعة اختياريّ** — قد تكون أمانةً لا ثمن لها، **فيُترك فارغا
 * ويُقرأ صفرا.** **ومن ألزم برقم في كلّ طلب** جعل السائق يكتب ما ليس
 * صحيحا ليمضي.
 *
 * **وأجرة التوصيل لا تُترك**: هي حقّه، **وطلبٌ بلا أجرة اتّفاقٌ ناقص.**
 */
@Composable
private fun AgreeDialog(onConfirm: (Long, Long) -> Unit, onDismiss: () -> Unit) {
    var goods by remember { mutableStateOf("") }
    var fee by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.agree_title)) },
        text = {
            Column {
                Text(stringResource(R.string.agree_hint), color = Rahal.colors.inkMuted)
                Spacer(Modifier.height(10.dp))
                OutlinedTextField(
                    value = goods,
                    onValueChange = { goods = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_goods)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = fee,
                    onValueChange = { fee = it.filter { c -> c.isDigit() } },
                    label = { Text(stringResource(R.string.agree_fee)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
            }
        },
        confirmButton = {
            RahalTextButton(
                onClick = { onConfirm(goods.toLongOrNull() ?: 0L, fee.toLongOrNull() ?: 0L) },
                enabled = fee.isNotBlank(),
            ) {
                Text(stringResource(R.string.agree_confirm))
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_cancel)) }
        },
    )
}

/**
 * **فعلٌ ثانويٌّ ملوّن** — أيقونةٌ وكلمةٌ على أرضٍ صلبة.
 *
 * **ولا حشوةَ عريضة**: ثلاثةُ أزرارٍ في صفٍّ واحدٍ على شاشةِ هاتف،
 * **وكلُّ نقطةٍ زائدةٍ تدفع الأوّلَ إلى سطرين.**
 */
@Composable
private fun SmallAction(
    icon: Int,
    label: Int,
    // **ونبرةٌ لا لون** — انظر `Tone`: اللونُ يقول الشكلَ والنبرةُ
    // تقول المعنى، **وثلاثةُ أزرارٍ في صفٍّ بألوانٍ مكتوبةٍ بيدٍ تنجو
    // من كلّ توحيدٍ لاحق.**
    tone: Tone,
    onClick: () -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    busy: Boolean = false,
) {
    RahalButton(
        onClick = onClick,
        enabled = enabled,
        tone = tone,
        // **وضيّقٌ لأنّ ثلاثةً في صفٍّ على شاشةِ هاتف** — والحشوةُ
        // العريضةُ تدفع الأوّلَ إلى سطرين.
        compact = true,
        modifier = modifier,
    ) {
        if (busy) {
            CircularProgressIndicator(
                Modifier.size(16.dp),
                strokeWidth = 2.dp,
                color = Color.White,
            )
            return@RahalButton
        }
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.size(4.dp))
        Text(
            text = stringResource(label),
            // **وحجمٌ واحدٌ للثلاثة** — (طلب المالك ٢٠٢٦-٠٨-١٢: «لازم
            // تكون بنفس الشكل»). **وزرٌّ أكبرُ من جاره** يُقرأ أهمَّ
            // منه، وهي ثلاثةُ أفعالٍ لا فعلٌ وحاشيتان.
            style = MaterialTheme.typography.labelMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بلاغُ الطارئ — ضغطةٌ واحدةٌ وتأكيد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا حقلَ يملؤه**: من عطلت درّاجتُه أو أُوقف في الطريق لا يكتب شرحا،
 * **وحقلٌ إلزاميٌّ في لحظةٍ كهذه** يجعله يترك الزرَّ ويتّصل بالمكتب.
 *
 * **وتأكيدٌ واحدٌ يسبقه**: بلاغٌ يُوقظ المكتبَ ويحرّر الطلب، **وضغطةٌ
 * بالخطأ في جيبٍ** تفعل ذلك كلَّه.
 */
@Composable
private fun EmergencyDialog(onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.emg_title)) },
        text = { Text(stringResource(R.string.emg_body), color = Rahal.colors.inkMuted) },
        confirmButton = {
            RahalTextButton(onClick = onConfirm) {
                Text(stringResource(R.string.emg_send), color = Rahal.colors.danger)
            }
        },
        dismissButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_back)) }
        },
    )
}

/**
 * **ما قد يقع للسائق نفسِه** — ثلاثةٌ تغطّي ما يقع فعلا.
 *
 * **ولا حقلَ حرّ**: من كتب «مشكلة» بيده لم يقل شيئا يُقاس، **ولا يُعدّ
 * ولا يُقارن شهرا بشهر.** وثلاثةُ ألفاظٍ تُختار في ثانيةٍ وهو واقف.
 */
private val MINE = listOf(
    R.string.problem_bike,
    R.string.problem_crash,
    R.string.problem_force,
)

/**
 * **أسباب التعذّر — تُختار ولا تُكتب.**
 *
 * **والسبب يقرّر من يتحمّل**: «المتجر مغلق» ذنب متجر يستوجب تعويض
 * السائق، **و«تأخّرت» ذنبه هو.** ونصّ حرّ لا يُعدّ ولا يُقاس.
 */
@Composable
private fun FailDialog(
    reasons: List<FailReasonItem>,
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
    onMine: (String) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        // **والعنوان سؤالٌ لا حكم** — (تصحيح المالك ٢٠٢٦-٠٨-١٢):
        // **«لماذا تعذّر» تفترض أنّ الطلب سقط**، والزرُّ يقول «لدي
        // مشكلة» — وأكثرُ المشاكل تُحلّ بسائقٍ ثانٍ لا بإلغاء.
        title = { Text(stringResource(R.string.problem_title)) },
        text = {
            Column {
                for (r in reasons) {
                    RahalTextButton(onClick = { onPick(r.code) }, modifier = Modifier.fillMaxWidth()) {
                        Text(reasonLabel(r.code), modifier = Modifier.fillMaxWidth())
                    }
                }
                // ══════════════════════════════════════════════════════
                // **والمشكلةُ قد تكون عنده هو — فيمضي ويأتي غيرُه**
                // ══════════════════════════════════════════════════════
                //
                // (قرار المالك ٢٠٢٦-٠٨-١٢: «المنصّة هي ترسل سائقاً
                //  ثانياً في حال حصلت مشكلة للسائق عند المتجر».)
                //
                // **وأسبابُ المتجر كلُّها ذنبُ متجر** — ومن عطلت
                // درّاجتُه فاختار «المتجر مغلق» ليمضي **حمّل متجراً
                // بريئاً ذنباً وتعويضا.**
                //
                // **والطلبُ لا يُلغى بل يعود للطابور**: الزبونُ ينتظر
                // طعامه، **وسائقٌ ثانٍ يأخذه في دقيقة** — وإلغاؤه
                // لعطلٍ في درّاجةٍ عقوبةٌ على من لا ذنب له.
                if (reasons.isNotEmpty()) {
                    HorizontalDivider(Modifier.padding(vertical = 6.dp))
                }
                Text(
                    text = stringResource(R.string.problem_mine),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelMedium,
                    modifier = Modifier.padding(bottom = 2.dp),
                )
                for (id in MINE) {
                    val label = stringResource(id)
                    RahalTextButton(
                        onClick = { onMine(label) },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(label, color = Rahal.colors.accent, modifier = Modifier.fillMaxWidth())
                    }
                }
            }
        },
        confirmButton = {
            RahalTextButton(onClick = onDismiss) { Text(stringResource(R.string.detail_cancel)) }
        },
    )
}

/** **ورمز بلا ترجمة يُعرض كما هو** — ليُعرف ويُضاف، لا ليُبتلع. */
@Composable
private fun reasonLabel(code: String): String = when (code) {
    "customer_absent" -> stringResource(R.string.reason_customer_absent)
    "customer_refused" -> stringResource(R.string.reason_customer_refused)
    "customer_unreachable" -> stringResource(R.string.reason_customer_unreachable)
    "address_wrong" -> stringResource(R.string.reason_address_wrong)
    "driver_late" -> stringResource(R.string.reason_driver_late)
    "merchant_closed" -> stringResource(R.string.reason_merchant_closed)
    "merchant_refused" -> stringResource(R.string.reason_merchant_refused)
    "merchant_not_ready" -> stringResource(R.string.reason_merchant_not_ready)
    "order_unknown" -> stringResource(R.string.reason_order_unknown)
    else -> code
}

/**
 * **أزرارُ الخريطة** — فوق البطاقة وفي جهة اليمين.
 *
 * (تصحيح المالك ٢٠٢٦-٠٨-١٢: «الأزرار بالعربيّ يجب أن تكون على اليمين».)
 *
 * **واليمينُ جهةُ الإبهام في شاشةٍ عربيّة** — كما تقع كلُّ أزرار
 * التطبيق: **ومن وضعها يسارا** جعل صاحبَها يعبر الشاشةَ بيده وهو يقود.
 *
 * **والملاحةُ مكتوبةٌ لا أيقونةً وحدَها**: هي الوحيدةُ التي تُخرجه من
 * التطبيق، **وبابٌ يَخرج منه بلا اسم** يُضغط بالخطأ فيجد نفسَه في
 * تطبيقٍ آخر ولا يعرف لماذا.
 */
@Composable
private fun MapButtons(
    follow: Boolean,
    onRecenter: () -> Unit,
    onFollow: () -> Unit,
    onChat: () -> Unit,
    chatting: Boolean,
    chatUnread: Int,
    onNavigate: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.Start,
        verticalAlignment = Alignment.Bottom,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            MapButton(R.drawable.ic_my_location, R.string.map_recenter, onRecenter)
            Spacer(Modifier.height(8.dp))
            // **والملاحقةُ تُضاء حين تعمل** — زرٌّ يفعل شيئا مستمرّا
            // **ولا يقول إنّه يعمل** يُضغط مرّتين فيُطفأ وهو يُظنّ مشتعلا.
            MapButton(
                icon = R.drawable.ic_navigation,
                label = R.string.map_follow,
                onClick = onFollow,
                on = follow,
            )
            Spacer(Modifier.height(8.dp))
            // ══════════════════════════════════════════════════════════
            // **وحديثُ الزبون قرصٌ عائمٌ لا سطرٌ في البطاقة**
            // ══════════════════════════════════════════════════════════
            //
            // (قرار المالك ٢٠٢٦-٠٨-١٢.)
            //
            // **وهو ما يُفتح فجأةً**: يتّصل الزبونُ ليقول «الباب الثاني»
            // — **فيكون في مرمى الإبهام دائما** لا يُبحث عنه في بطاقةٍ
            // قد تكون مطويّةً تحت.
            MapButton(
                icon = R.drawable.ic_chat,
                label = R.string.trip_chat,
                onClick = onChat,
                on = chatting,
                badge = chatUnread,
            )
            Spacer(Modifier.height(8.dp))
            NavigateButton(onNavigate)
        }
    }
}

/** **قرصٌ واحد** — أيقونةٌ في دائرةٍ ترتفع عن الخريطة بظلّها. */
@Composable
private fun MapButton(
    icon: Int,
    label: Int,
    onClick: () -> Unit,
    on: Boolean = false,
    /** **كم ينتظره خلف هذا الزرّ** — وصفرٌ يعني لا شارة. */
    badge: Int = 0,
) {
    Box(contentAlignment = Alignment.TopEnd) {
        // ══════════════════════════════════════════════════════════════
        // **وأرضُ القرص سطحٌ لا أرضُ الصفحة**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٣ بلقطةٍ من جهازه: «وزنُ الأيقونات
        //  مزعج».)
        //
        // **وقيس التباينُ في الغامقة: ١٫١١ إلى ١** — أرضُ القرص
        // (`canvas`) والأيقونةُ (`panel`) كحليّان متجاوران، **فتبدو
        // أقراصاً داكنةً صمّاءَ لا أزراراً.**
        //
        // **وسببُه أنّ اللونين كانا صحيحين في الفاتحة**: أبيضُ وكحليّ
        // — **١٥٫٢٩ إلى ١.** ثمّ جاءت الغامقةُ فصارا واحدا.
        //
        // **والسطحُ يرتفع عن الأرض درجةً** فيُرى القرص، **والحبرُ يقع
        // عليه**: ١١٫٧١ في الغامقة و٥٫٨٥ في الفاتحة.
        //
        // # وأخفُّ وزناً
        //
        // **أربعةُ أقراصٍ فوق بعضها على خريطةٍ ثقيلة** — فقُصّ قطرُها
        // إلى ثمانيةٍ وأربعين (وهو أدنى ما يُلمس بإبهام)، **وخفّ ظلُّها
        // من ستٍّ إلى ثلاث**، وضاق ما بينها.
        Box(
            Modifier
                .size(48.dp)
                .shadow(3.dp, CircleShape)
                .clip(CircleShape)
                .background(if (on) Rahal.colors.brand else Rahal.colors.surface)
                .clickable(onClick = onClick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(icon),
                contentDescription = stringResource(label),
                tint = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                modifier = Modifier.size(22.dp),
            )
        }
        // **والشارةُ تطفو على حافّته** — كما في كلّ تطبيق: **ومن وضعها
        // بجانبه** جعلها تُقرأ رقما آخر لا عدّ رسائل.
        if (badge > 0) {
            Box(
                Modifier
                    .clip(CircleShape)
                    .background(Rahal.colors.danger)
                    .padding(horizontal = 6.dp, vertical = 1.dp),
            ) {
                Text(
                    text = if (badge > 9) "+9" else badge.toString(),
                    color = Color.White,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelSmall,
                )
            }
        }
    }
}

/**
 * **بابُ الخروج — مكتوبٌ باسمه.**
 *
 * **ولونُه غيرُ لون أختيه**: هاتان تحرّكان كاميرا وتبقيان في المكان،
 * **وهذا يترك التطبيق** — واختلافُ الفعل يُقال باللون قبل أن يُقرأ.
 */
@Composable
private fun NavigateButton(onClick: () -> Unit) {
    Column(
        Modifier
            .shadow(6.dp, Rahal.shape.md)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.accent)
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        // **وسهمٌ إلى أعلى لا سهمُ إرسال** — (تصحيح المالك ٢٠٢٦-٠٨-١٢:
        // «والسهم للأعلى مو ع طرف»). **وهو ما تعرفه العينُ ملاحةً**
        // في كلّ تطبيقٍ يقودها.
        Icon(
            painter = painterResource(R.drawable.ic_arrow_up),
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(24.dp),
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = stringResource(R.string.trip_navigate),
            color = Color.White,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelSmall,
        )
    }
}

/**
 * **لوحُ الرحلة** — إلى أين، وكم بقي، وأين صار من مراحلها.
 *
 * # ولماذا لوحٌ واحد
 *
 * **لوحان فوق خريطةٍ يقضمان ثلثَها** — وهي ما يقود عليه. **وأحدُهما
 * يختفي عند الوقوف فيترك فراغاً معلّقا** لا يُقرأ شيئا.
 *
 * # والطورُ يغيب والمراحلُ تبقى
 *
 * **من وقف عند الباب لا يقرأ «الطريق إلى فلان» ولا مسافةً ولا زمنا** —
 * وهو واقفٌ فيه. **لكنّه يبقى يريد أن يعرف أين صار من الرحلة.**
 */
@Composable
private fun TripPanel(state: TripState) {
    val order = state.order ?: return
    val moving = state.step == TripStep.TO_PICKUP || state.step == TripStep.TO_CUSTOMER ||
        state.step == TripStep.PICKED_UP

    // ══════════════════════════════════════════════════════════════════
    // **وشريطُ المراحل يُسحب إلى الأعلى فيختفي**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «شريطُ الحالات لازم يختفي من أمام
    //  السائق — يعني يُسحب للأعلى أيضاً مثل الكرت السفليّ».)
    //
    // **والخريطةُ هي الشاشة**: لوحٌ فوقها وبطاقةٌ تحتها يقضمان ثلثيها،
    // **ومن يقود يريد أن يرى الشارعَ الذي أمامه.**
    //
    // **والمراحلُ يُنظر إليها مرّةً في الطور** — لا كلَّ ثانية.
    // **والوجهةُ والمسافةُ تبقيان**: هما ما يُقرأ في نظرةٍ خاطفة.
    //
    // **والحركةُ حركةُ البطاقة السفليّة نفسُها** — سحبٌ على اللوح
    // كلِّه ولمسةٌ على المقبض: **إيماءتان مختلفتان في شاشةٍ واحدةٍ
    // تُنسيان إحداهما.**
    //
    // **وتعود مع كلّ طلبٍ جديد** (`rememberSaveable(order.id)`) — من
    // بدأ طوراً جديداً يريد أن يرى أين صار.
    var stripOpen by rememberSaveable(order.id) { mutableStateOf(true) }

    Column(
        Modifier
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .clip(Rahal.shape.lg)
            .background(Rahal.colors.panel)
            .pointerInput(Unit) {
                detectVerticalDragGestures { _, dy ->
                    if (dy < -6f) stripOpen = false
                    if (dy > 6f) stripOpen = true
                }
            }
            .padding(horizontal = 14.dp, vertical = 10.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        if (moving) {
            // ══════════════════════════════════════════════════════════
            // **والوجهةُ باسمها لا بصفتها**
            // ══════════════════════════════════════════════════════════
            //
            // (وهو ما قرّره المالكُ في بطاقة الطلب ٢٠٢٦-٠٨-١٢: «ما في
            //  داعي لكلمة المتجر» — واللوحُ أولى به.)
            //
            // **و«الطريق إلى الزبون» لا تقول لمن يحمل ثلاثة طلبات
            // أيَّها هذا** — و«ابوطيف» تقول.
            val name = if (state.step >= TripStep.PICKED_UP) {
                order.customerName.ifBlank { stringResource(R.string.detail_customer) }
            } else {
                order.merchantName.ifBlank { stringResource(R.string.card_custom) }
            }
            // ══════════════════════════════════════════════════════════
            // **والخاصُّ قبل الشراء لا وجهةَ له**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نغلق الطلبَ الخاصَّ أوّلاً».)
            //
            // **كان يُكتب «الطريق إلى طلب خاصّ»** — واسمُ المتجر فيه
            // هو الكلمةُ «طلب خاصّ» نفسُها. **فتُقرأ الجملةُ طريقاً إلى
            // مكانٍ اسمُه «طلب خاصّ».**
            //
            // **وما يفعله في هذا الطور محادثةٌ واتّفاقٌ ثمّ شراء** —
            // لا سيرٌ إلى موضع. **فيُقال له ذلك.**
            val custom = order.kind == "custom"
            Text(
                text = if (custom && state.step < TripStep.PICKED_UP) {
                    stringResource(R.string.trip_p_custom)
                } else {
                    stringResource(phaseLabel(state.step), name)
                },
                color = Color.White,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleMedium,
            )

            // **ورقمُ المحرّك يسبق الهوائيّ** — «٩٨٢ م ودقيقتان» كانت
            // تعني في الواقع «١٫١ كم وسبعَ دقائق». (قيس ٢٠٢٦-٠٨-١٢.)
            val meters = if (state.routeM >= 0) state.routeM else state.remainingM
            if (meters >= 0) {
                Spacer(Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // **والمدّة من المحرّك إن وُجدت** — محسوبةً بسرعات
                    // الشوارع نفسِها لا بسرعةٍ واحدةٍ في الإعدادات.
                    val minutes = if (state.routeSec >= 0) {
                        minutesShort((state.routeSec / 60).toLong().coerceAtLeast(1))
                    } else {
                        eta(meters, state.avgSpeedKmh).substringAfter("~")
                    }
                    if (minutes.isNotEmpty()) {
                        PanelChip(R.drawable.ic_time, minutes)
                        Spacer(Modifier.size(14.dp))
                    }
                    PanelChip(R.drawable.ic_pin, distanceText(meters))
                }
            }
            if (stripOpen) {
                Spacer(Modifier.height(10.dp))
                HorizontalDivider(color = Color.White.copy(alpha = 0.12f))
            }
        }

        androidx.compose.animation.AnimatedVisibility(visible = stripOpen) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Spacer(Modifier.height(10.dp))
                LegStrip(
                    status = order.status,
                    custom = order.kind == "custom",
                    // **وعلامةُ الطور أنّ الثمنَ وُثّق** — لا حالٌ ثانيةٌ
                    // في المحرّك: يبقى `assigned` قبل التوثيق وبعده.
                    agreed = order.customFee != null,
                )
            }
        }

        // **ومقبضٌ يُرى** — شريطٌ فاتحٌ يقول «هذا يُسحب»، **ولوحٌ يُسحب
        // ولا يقول** لا يعرف أحدٌ أنّه يُسحب. **ومن أخفاه لا يجد ما
        // يعيده به.**
        Spacer(Modifier.height(8.dp))
        Box(
            Modifier
                .clip(Rahal.shape.pill)
                .background(Color.White.copy(alpha = 0.30f))
                .size(width = 44.dp, height = 5.dp)
                .clickable { stripOpen = !stripOpen },
        )
    }
}

/** **أيقونةٌ ورقم** — على الأرض الداكنة. */
@Composable
private fun PanelChip(icon: Int, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier.size(15.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = text,
            color = Color.White,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/**
 * **شريطُ المراحل الأربع** — دوائرُ موصولةٌ بخطّ.
 *
 * **والخطُّ بينها ليس زينة**: هو ما يقول إنّها **رحلةٌ واحدةٌ تمشي**، لا
 * أربعُ حالاتٍ متجاورة. **وما مضى منه ملوّنٌ وما بقي باهت** — فيُقرأ
 * التقدّمُ بالعين قبل أن تُقرأ الكلمات.
 *
 * **والحاليّةُ وحدَها كبيرةٌ برتقاليّة**: من لوّن الكلَّ جعل صاحبَه
 * يبحث عن موضعه بين أربعةٍ متشابهة.
 */
@Composable
private fun LegStrip(status: String, custom: Boolean = false, agreed: Boolean = false) {
    // ══════════════════════════════════════════════════════════════════
    // **ومراحلُ الخاصّ غيرُ مراحل العاديّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٣ بلقطةٍ من جهازه: «برأيي الخطواتُ هنا مو
    //  مزبوطة — بالطلب الخاصّ».)
    //
    // **كان يُعرض شريطُ العاديّ**: أوّلُ مرحلةٍ فيه «استلمت الطلب» —
    // **فيقرأ صاحبُ طلبٍ خاصٍّ أنّه في طور الاستلام** وهو لم يتّفق
    // على شيءٍ بعد. **والزرُّ تحته يقول «اشتريتُ الطلب».**
    //
    // **والمحرّكُ يعرف مراحلَ الخاصّ وحدَها** (`OpsCustomStages`):
    // توثيقٌ ثمّ شراءٌ ثمّ طريقٌ ثمّ وصولٌ ثمّ تسليم. **فتُقرأ منه لا
    // تُخترع هنا.**
    val legs = if (custom) CUSTOM_LEGS else LEGS
    val icons = if (custom) CUSTOM_LEG_ICONS else LEG_ICONS
    val at = if (custom) customLegOf(status, agreed) else legOf(status)
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Top) {
        for (i in legs.indices) {
            if (i > 0) {
                // **والخطُّ في مستوى الدوائر** — لا تحتها ولا فوقها.
                Box(
                    Modifier
                        .weight(1f)
                        .padding(top = 15.dp)
                        .height(2.dp)
                        .background(
                            if (i <= at) Rahal.colors.brand else Color.White.copy(alpha = 0.18f),
                        ),
                )
            }
            LegDot(legs[i], icons[i], i, at)
        }
    }
}

@Composable
private fun LegDot(label: Int, icon: Int, index: Int, at: Int) {
    val done = index < at
    val here = index == at
    val ground = when {
        here -> Rahal.colors.accent
        done -> Rahal.colors.brand
        else -> Color.White.copy(alpha = 0.12f)
    }
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.width(76.dp),
    ) {
        Box(
            Modifier.size(32.dp).clip(CircleShape).background(ground),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(if (done) R.drawable.ic_check_circle else icon),
                contentDescription = null,
                tint = if (here || done) Color.White else Color.White.copy(alpha = 0.45f),
                modifier = Modifier.size(17.dp),
            )
        }
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(label),
            color = when {
                here -> Color.White
                done -> Color.White.copy(alpha = 0.7f)
                else -> Color.White.copy(alpha = 0.4f)
            },
            fontWeight = if (here) FontWeight.Bold else FontWeight.Normal,
            style = MaterialTheme.typography.labelSmall,
            textAlign = TextAlign.Center,
            maxLines = 2,
        )
    }
}

private val LEGS = listOf(
    R.string.leg_picked,
    R.string.leg_way,
    R.string.leg_arrived,
    R.string.leg_done,
)

private val LEG_ICONS = listOf(
    R.drawable.ic_store,
    R.drawable.ic_moto,
    R.drawable.ic_pin,
    R.drawable.ic_check_circle,
)

/**
 * **مراحلُ الطلب الخاصّ** — كما يعرفها المحرّك (`OpsCustomStages`).
 *
 * **وخمسٌ لا أربع**: بين الإسناد والطريق طوران لا طور — **يتّفق ثمّ
 * يشتري**، وكلاهما فعلٌ يقع في وقتٍ ويُسأل عنه في الشكوى.
 */
private val CUSTOM_LEGS = listOf(
    R.string.leg_agree,
    R.string.leg_buy,
    R.string.leg_way,
    R.string.leg_arrived,
    R.string.leg_done,
)

private val CUSTOM_LEG_ICONS = listOf(
    R.drawable.ic_chat,
    R.drawable.ic_cash,
    R.drawable.ic_moto,
    R.drawable.ic_pin,
    R.drawable.ic_check_circle,
)

/**
 * **أيُّ مرحلةٍ من مراحل الخاصّ هو فيها.**
 *
 * **والحالُ لا يفرّق بين التوثيق والشراء** — يبقى `assigned` فيهما،
 * **والفارقُ أنّ الثمنَ وُثّق.** (`custom_fee` غيرُ فارغ.)
 */
private fun customLegOf(status: String, agreed: Boolean): Int = when (status) {
    "picked_up", "on_the_way" -> 2
    "at_dropoff" -> 3
    "delivered" -> 4
    else -> if (agreed) 1 else 0
}

/**
 * **أيُّ مرحلةٍ هو فيها الآن.**
 *
 * **وما قبل الاستلام كلُّه المرحلةُ الأولى**: `assigned` و`at_pickup`
 * طريقُه إلى المتجر — **وهي عندنا مرحلةٌ واحدة** لأنّ الزبون لا يفرّق
 * بينهما، **والسائقُ يقرأ ما عليه في الزرّ لا في الشريط.**
 */
private fun legOf(status: String): Int = when (status) {
    "picked_up", "on_the_way" -> 1
    "at_dropoff" -> 2
    "delivered" -> 3
    else -> 0
}

/**
 * **اسمُ الطور** — والرحلةُ طوران لا سبعة.
 *
 * **وما قبل الاستلام كلُّه «إلى المتجر»**، وما بعده كلُّه «إلى الزبون»
 * — **والوقوفُ عند أحدهما طورٌ ثالثٌ قصير** يُقال لأنّ الفعل التالي
 * يختلف.
 */
private fun phaseLabel(step: TripStep): Int = when (step) {
    TripStep.AT_PICKUP -> R.string.trip_p_at_store
    TripStep.PICKED_UP, TripStep.TO_CUSTOMER -> R.string.trip_p_to_customer
    TripStep.AT_CUSTOMER -> R.string.trip_p_at_customer
    TripStep.DELIVERED -> R.string.trip_p_delivered
    else -> R.string.trip_p_to_store
}

/** المسافة بالمتر أو بالكيلومتر — **لا «1400 م».** */
private fun distanceText(meters: Double): String = dist(meters)

/**
 * **الوقت المتوقّع** — من المسافة وسرعة السائق.
 *
 * **وفارغ إن لم تُضبط السرعة**: **رقمٌ مبنيٌّ على صفر يقول «الآن»**،
 * ووعدٌ كاذبٌ للزبون أسوأ من لا وعد.
 *
 * **ودقيقة على الأقلّ** — «٠ دقيقة» لا تُقال لمن لم يصل بعد.
 */
private fun eta(meters: Double, avgSpeedKmh: Long): String =
    etaText(meters, avgSpeedKmh)

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
            text = offer.merchantName.ifBlank { stringResource(R.string.card_custom) } +
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
                        order.merchantName.ifBlank { stringResource(R.string.card_custom) }
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
        RahalButton(onClick = onOrders) { Text(stringResource(R.string.nav_orders)) }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خطوات الرحلة السبع — وستّة أحوال في المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والفرق مقصود**: «قبلت الطلب» و«في الطريق إلى المتجر» **حالٌ واحد في
 * المحرّك** (`assigned`) — لكنّهما لحظتان مختلفتان عند السائق: **قَبِل،
 * ثمّ مشى.**
 *
 * # وخطأٌ وقع هنا
 *
 * **كُتب الشريطُ يقرأ خطوتَه من ترتيب القائمة والزرُّ يقرأ التالية منها**
 * — فظهر الشريطُ يقول «قبلت الطلب» والزرُّ يقول «استلمت البضاعة»:
 * **قفزةُ خطوةٍ كاملة.** (قيس على الجهاز ٢٠٢٦-٠٨-١٢.)
 *
 * **فصار كلٌّ منهما يُشتقّ من حال الطلب في المحرّك وحدَه** — لا من
 * الآخر.
 */
enum class TripStep(val label: Int) {
    ACCEPTED(R.string.trip_s_accepted),
    TO_PICKUP(R.string.trip_s_to_pickup),
    AT_PICKUP(R.string.trip_s_at_pickup),
    PICKED_UP(R.string.trip_s_picked_up),
    TO_CUSTOMER(R.string.trip_s_to_customer),
    AT_CUSTOMER(R.string.trip_s_at_customer),
    DELIVERED(R.string.trip_s_delivered),
    ;

    companion object {
        /**
         * **يقرأ الخطوة من حال الطلب.**
         *
         * **و`assigned` تُقرأ «في الطريق إلى المتجر»** لا «قبلت»: **القبول
         * لحظةٌ مضت**، وما يفعله الآن هو المشي.
         */
        fun of(status: String): TripStep = when (status) {
            "assigned" -> TO_PICKUP
            "at_pickup" -> AT_PICKUP
            "picked_up" -> PICKED_UP
            "on_the_way" -> TO_CUSTOMER
            "at_dropoff" -> AT_CUSTOMER
            "delivered" -> DELIVERED
            else -> ACCEPTED
        }
    }
}

/** الفعل التالي — **باسم الحال في المحرّك ونصِّ الزرّ.** */
data class NextAction(val status: String, val label: Int)

/**
 * **ما يفعله الآن** — يُشتقّ من حال الطلب لا من موضعه في الشريط.
 *
 * **والمحرّك يرفض أيّ قفزة** (`orders/statuses.go`) — فلو تأخّرت الشاشة
 * عن حاله الحقيقيّ ردّ خطأً ولم يقع شيء.
 */
fun nextAction(status: String, custom: Boolean = false): NextAction? = when (status) {
    // ══════════════════════════════════════════════════════════════════
    // **ولا «وصلتُ إلى المتجر» في الطلب الخاصّ**
    // ══════════════════════════════════════════════════════════════════
    //
    // (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «ما في شيء اسمه وصلتُ للمتجر» —
    //  وقرارُه ٢٠٢٦-٠٨-١٣: «نغلق الطلبَ الخاصَّ أوّلاً».)
    //
    // **لا متجرَ يقف عنده** — والمرحلةُ بعد الإسناد محادثةٌ واتّفاقٌ ثمّ
    // شراء. **وخارطةُ المحرّك تقولها صراحة**: الخاصُّ يمضي من «أُسند»
    // إلى «اشتريتُ» رأساً (`orders/statuses.go: customTransitions`).
    //
    // **وكان الزرُّ يرسل `at_pickup`** — وهو انتقالٌ لا تعرفه خارطةُ
    // الخاصّ. **فيضغطه السائقُ فيُردّ ولا يفهم لماذا**: الشاشةُ تعرض
    // خطوةً لا وجودَ لها في المحرّك.
    "assigned" ->
        if (custom) {
            NextAction("picked_up", R.string.step_bought)
        } else {
            NextAction("at_pickup", R.string.step_at_pickup)
        }
    "at_pickup" -> NextAction("picked_up", R.string.step_picked_up)
    "picked_up" -> NextAction("on_the_way", R.string.step_on_the_way)
    "on_the_way" -> NextAction("at_dropoff", R.string.step_at_dropoff)
    "at_dropoff" -> NextAction("delivered", R.string.step_delivered)
    else -> null
}

/** ما تعرضه الرحلة — **ولا تملكه هي.** */
data class TripState(
    val order: DriverOrder? = null,
    /** ما بقي من الطريق بالمتر — **وسالبٌ يعني لا يُعرف.** */
    val remainingM: Double = -1.0,
    // ══════════════════════════════════════════════════════════════════
    // **والمسارُ الحقيقيُّ يسبق المستقيم**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢.)
    //
    // **وفارغٌ يعني أنّ محرّك المسارات لم يردّ** — فيُرسم الخطُّ المستقيمُ
    // وتُقرأ المسافةُ الهوائيّة، **ولا تبقى الخريطةُ بلا خطّ.**
    /** نقاطُ خطّ الشوارع — **وفارغةٌ تعني المستقيم.** */
    val routeLine: List<LatLng> = emptyList(),
    /** طولُ الطريق بالشوارع — **وسالبٌ يعني لا يُعرف.** */
    val routeM: Double = -1.0,
    /** مدّتُه بالثواني كما يحسبها المحرّك بسرعات الشوارع. */
    val routeSec: Double = -1.0,
    // ══════════════════════════════════════════════════════════════════
    // **ومسارُ الملاحة — المرحلة ٣أ**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وحدةٌ واحدةٌ لا حقولٌ متفرّقة**: هندسةٌ وتراكميّةٌ ومناوراتٌ
    // معاً، **فلا لحظةَ تكون فيها جديدةٌ مع قديمة.**
    //
    // **وفارغٌ يعني «لا إرشاد»** — والخريطةُ ترسم `routeLine` كما
    // كانت. (انظر `NavRouteMapper`.)
    val navRoute: com.rahalgo.navigation.NavRoute? = null,
    /** سرعة السائق الوسطى من المحرّك — **وصفر يعني لا تُحسب مدّة.** */
    val avgSpeedKmh: Long = 0,
    /** **هل هو على بُعد خطوات من وجهته؟** — يُقترح ولا يُنفَّذ. */
    val nearDestination: Boolean = false,
    /** أتلزم صورة تسليم؟ — **يقرّره المحرّك** (`drivers.require_delivery_photo`). */
    val requirePhoto: Boolean = false,
    /** أنافذة الاتّفاق مفتوحة؟ — **للطلب الخاصّ وحدَه.** */
    val agreeOpen: Boolean = false,
    /** محطّاته كلّها — **وواحدةٌ منها هي المعروضة.** */
    val stops: List<Stop> = emptyList(),
    /** **أنافذة الطارئ مفتوحة؟** — تُفتح حين لا سببَ يُختار. */
    val emergencyOpen: Boolean = false,
    /** عرضٌ نزل وهو في رحلة — **وفارغ يعني لا عرض.** */
    val onRouteOffer: DriverOrder? = null,
    val failReasons: List<FailReasonItem>? = null,
    val step: TripStep = TripStep.ACCEPTED,
    val driver: LatLng? = null,
    val pickup: LatLng? = null,
    val dropoff: LatLng? = null,
    val busy: Boolean = false,
    val error: String = "",

    // ══════════════════════════════════════════════════════════════════
    // **خياراتُ المسار — إغلاقُ واجهة ٧، ٢٠٢٦-٠٨-٢١**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفارغةٌ تعني: لا لوحةَ تُعرض** (البند ٥) — ولا مساحةَ تُؤخذ من
    // شاشةِ سائقٍ يقود.
    val routeChoices: com.rahalgo.navigation.RouteChoices? = null,
    /** **المعايَنُ** — بصريٌّ محضٌ لا ملاحة (البند ١٣). */
    val choicePreview: String? = null,
    /** **وقد بطل** — إشعارٌ قصيرٌ ثمّ يزول (البند ١٩). */
    val choiceStale: Boolean = false,
    /** **مسارٌ اعتُمد وينتظر التسليم** — البند ١٧. */
    val committedRoute: com.rahalgo.navigation.NavRoute? = null,
    /**
     * **هويّةُ المسار النافذ** — المرحلة ٨ب.
     *
     * **معرّفٌ محايدٌ من الخادم** — لا يُخترع في الجوّال.
     */
    val navRouteId: String? = null,

    /** **سببُ آخرِ تركيب** — واختيارُ السائق لا يُطلق جلباً (البند ٦). */
    val installReason: com.rahalgo.navigation.RouteInstallReason =
        com.rahalgo.navigation.RouteInstallReason.INITIAL,
)

data class TripActions(
    val step: (String) -> Unit,
    /** يفتح الكاميرا لصورة التسليم. */
    val capture: () -> Unit,
    val release: () -> Unit,
    val chat: () -> Unit,
    val askAgree: () -> Unit,
    val agree: (Long, Long) -> Unit,
    val dismissAgree: () -> Unit,
    val pickStop: (String) -> Unit,
    val takeOffer: (String) -> Unit,
    val dismissOffer: () -> Unit,
    val askFail: () -> Unit,
    val fail: (String) -> Unit,
    val dismissFail: () -> Unit,
    /** **مشكلةٌ عند السائق نفسِه** — يُكتب سببُها ويُعاد الطلبُ أو
     *  تُنبَّه العمليات، بحسب موضعه من الرحلة. */
    val problem: (String) -> Unit,
    /** **بلاغُ الطارئ** — العملياتُ تُنبَّه وموضعُه يُقرأ. */
    val emergency: () -> Unit,
    val dismissEmergency: () -> Unit,
    val navigate: () -> Unit,
    val toOrders: () -> Unit,

    // ══════════════════════════════════════════════════════════════════
    // **اختيارُ المسار — إغلاقُ واجهة ٧**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا اختيارَ بلمسةٍ واحدة** (البند ١١): `previewRoute` تُعاين،
    // **و`confirmRoute` وحدَها تعتمد.**

    /** **يطلب البدائل** — عند حدثٍ لا في حلقة (البند ٤). */
    val loadAlternatives: (
        Long,
        com.rahalgo.navigation.RouteTarget,
        Double,
        Double,
        com.rahalgo.navigation.RouteInstallReason,
        Long,
        List<com.rahalgo.navigation.GeoPoint>,
    ) -> Unit = { _, _, _, _, _, _, _ -> },

    /** **يُعاين** — بصريٌّ محضٌ (البند ١٣)، و`null` يُلغي. */
    val previewRoute: (String?) -> Unit = {},

    /** **يعتمد** — بعد فحصٍ، ويردّ هل قُبل (البندان ١٧ و١٨). */
    val confirmRoute: (Long, com.rahalgo.navigation.RouteTarget, Double, Double, Double, Boolean) -> Boolean =
        { _, _, _, _, _, _ -> false },

    /** **بعد أن تسلّم الشاشةُ المسارَ المعتمد.** */
    val onRouteCommitted: () -> Unit = {},

    /** **مسحٌ صريح** — تبدّلُ وجهةٍ أو نجاحُ إعادةِ حساب (٢٢ و٢٣). */
    val clearChoices: () -> Unit = {},

    /** **يسأل الخادمَ عن ارتباط الأثر بالمسار** — المرحلة ٨ب. */
    val askCorrelation: (
        String,
        List<com.rahalgo.shared.model.CorrelationFix>,
        (com.rahalgo.navigation.RoadCorrelation) -> Unit,
    ) -> Unit = { _, _, _ -> },

    /** **تُخبر نموذجَ العرض بما رُكّب ولماذا** — البند ٧. */
    val onRouteInstalled: (Long, com.rahalgo.navigation.RouteTarget, com.rahalgo.navigation.RouteInstallReason) -> Unit =
        { _, _, _ -> },
)


/**
 * **زرُّ الرحلة المصنوعة** — أعلى يسارَ الخريطة، بعيداً عن الثلاثة.
 *
 * **ولونُه ليس لونَ العلامة** — **وزرُّ فحصٍ يشبه أزرارَ العمل يُضغط
 * سهواً**، ولو في بناء تطوير.
 *
 * **ويقول «تجريبيّة» بلفظه** — **ومن رأى سهمَه يمشي وهو جالسٌ ولم
 * يعرف أنّها محاكاةٌ ظنّ الموقعَ عطبان.**
 */
@Composable
private fun ReplayButton(
    running: Boolean,
    onStart: () -> Unit,
    onStop: () -> Unit,
) {
    Surface(
        color = if (running) Rahal.colors.danger else Color(0xFF444C56),
        contentColor = Color.White,
        shape = Rahal.shape.sm,
        modifier = Modifier
            .padding(12.dp)
            .clickable { if (running) onStop() else onStart() },
    ) {
        Text(
            text = if (running) "أوقف الرحلة التجريبيّة" else "رحلة تجريبيّة",
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
        )
    }
}
