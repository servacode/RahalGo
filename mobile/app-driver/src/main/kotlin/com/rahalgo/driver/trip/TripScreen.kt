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
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import com.rahalgo.ui.Locating
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
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
    onReplayRetarget: (List<com.rahalgo.navigation.NavFix>) -> Unit = {},
    /** **الرحلةُ التجريبيّةُ مشغولةٌ لهذا الطلب** — فتبدأ الساقَ التالية وحدَها. */
    demoTrip: Boolean = false,
    demoLegEnd: com.rahalgo.navigation.GeoPoint? = null,
    /** **تشغيلُ التجربة قبل أن يوجد طريق** — الخاصُّ قبل الشراء. */
    onArmDemo: () -> Unit = {},
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

    // **ورحلةٌ فاعلةٌ تقفل الدرجَ الجانبيّ من لحظتها** (قرارُ المالك ٢٠٢٦-٠٩-٢٨):
    // لا يُفتَح بالسحب من الحافّة ولا ينازع الخريطةَ، **ولو قبل أن تُركَّب
    // الخريطة.** يُفتَح القفلُ حتماً عند نهاية الرحلة أو الخروج منها.
    com.rahalgo.ui.TripGestureLock()

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
    // **و«موقعي» عند السائق توسيطٌ لا نداءُ موضع** (`MLW-14`، ٢٠٢٦-٠٩-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وموضعُه يصل بتيّارٍ متّصلٍ من خدمة الورديّة** — **فلا يُفتح له
    // نداءٌ ثانٍ**: **ونداءُ موضعٍ ثالثٌ فوق تيّارٍ يعمل استنزافُ
    // بطّاريّةٍ بلا فائدة.**
    //
    // **والعطبُ الذي كان**: **`recenter` تُوسّط إن عُرف موضعُه، وإلّا
    // اتّسعت الكاميرا لتضمّ النقاطَ صامتة** — **فيضغط «موقعي» فتقفز
    // الخريطةُ إلى مكانٍ ليس هو**، **ولا سطرَ يقول لماذا.**
    //
    // **وأكثرُ ما يقع هذا حين تُطفأ خدمةُ الموقع** — **وهي التي لا
    // تُرى.**
    val locating = remember { Locating() }
    val hereContext = LocalContext.current

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
    LaunchedEffect(state.order?.id, state.step, state.order?.lat, state.order?.lng, state.order?.dropoffKnown) {
        val o = state.order
        navSession.setArrivalTarget(
            when {
                o == null -> null
                // **ولا وصولَ إلى بابٍ لا يُعرف** — «لدي توصيلة» بلا نقطةٍ يُكتب
                // مكانَها موقعُ المتجر، **فتُعلن الملاحةُ وصولَه وهو عند المتجر.**
                state.step >= TripStep.PICKED_UP ->
                    dropoffPoint(o)?.let { (la, ln) -> com.rahalgo.navigation.GeoPoint(la, ln) }
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
            android.util.Log.i("RahalGo/perf", "route installed on TripScreen")
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
        showable = com.rahalgo.navigation.RouteChoiceHealth.showable(navSession.nav),
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
    // **وكلَّ دقيقتين وهو يسير: هل من طريقٍ أفضل؟** (طلبُ المالك ٢٠٢٦-١٠-٠٢)
    // ══════════════════════════════════════════════════════════════════
    //
    // **لا يُسأل** واقفاً، ولا قربَ الوجهة (٣٠٠م)، ولا والحالُ غيرُ سليمة أو يُعاد الحساب،
    // **ولا والبدائلُ معروضةٌ الآن لهذا الجيل** — فلا يُبدَّل ما ينظر إليه.
    LaunchedEffect(state.order?.id, routeTarget) {
        while (true) {
            kotlinx.coroutines.delay(BETTER_ROUTE_EVERY_MS)
            val nav = navSession.nav ?: continue
            val lat = nav.lat ?: continue
            val lng = nav.lng ?: continue
            if (nav.speedMps < 3.0 || nav.remainingM in 0.0..300.0) continue
            if (!com.rahalgo.navigation.RouteChoiceHealth.of(nav)) continue
            val shown = state.routeChoices
            if (shown != null && shown.generation == navSession.generation &&
                shown.alternatives.isNotEmpty() &&
                choiceUi !is com.rahalgo.navigation.RouteChoiceUi.Hidden
            ) {
                continue
            }
            actions.checkBetterRoute(navSession.generation, routeTarget, lat, lng)
        }
    }

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
        dropoffKnown = state.order?.dropoffKnown != false,
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
        // **وصارت ثلاثين** — طلبُ المالك ٢٠٢٦-١٠-٠٢: «بعد ٣٠ ثانيةً يتحوّل كأنّه
        // ضُغط»، **وهي مدّةُ `OrdersViewModel.ARRIVAL_HOLD_MS` نفسُها** فلا مدّتان.
        kotlinx.coroutines.delay(30_000L)
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
            // **والخطُّ المرسومُ هو الطريقُ الذي يُرشد عليه المحرّك** (فحصُ الملاحة
            // ١.٢): كان من نموذج الطلبات فبقي القديمُ بعد إعادة الحساب، **وقُصَّ
            // بتقدّمٍ يُقاس على طريقٍ آخر.**
            route = navSession.route?.geometry?.map { LatLng(it.lat, it.lng) } ?: state.routeLine,
            follow = follow,
            recenter = recenter,
            // **وما تُرسمه الملاحةُ حين تعمل** — وفارغٌ يعني «الخريطةُ
            // كما كانت حرفاً بحرف».
            nav = navSession.render,
            // **والبوصلةُ للواقف والبطيء** — ولا في الرحلة التجريبيّة: الجهازُ لا يتحرّك فيها.
            compass = com.rahalgo.navigation.rememberCompassHeading(
                enabled = navSession.render != null && !navSession.replaying,
            ),
            // **البدائلُ تُرسم ولا يُلاحَ عليها** — البند ٢٨ من ٧.
            alternatives = choiceUi.choicesOrNull?.alternatives.orEmpty().map {
                com.rahalgo.navigation.AltRouteLayer.Drawable(
                    it.routeId, it.route, it.engineDurationS, it.deltaDurationS,
                )
            },
            previewRouteId = choiceUi.previewRouteId,
            // **ولمسُ الخطّ يُعاين ولا يعتمد** — البندان ٢٤ و٢٧.
            // ══════════════════════════════════════════════════════
            // **واللمسُ على الخريطة يبدّل، لا يعاين**
            // ══════════════════════════════════════════════════════
            //
            // **(طلبُ المالك ٢٠٢٦-٠٨-٣١:** «بغوغل ماب يعطيك الطرقَ
            // المتاحة للوصول إلى نفس المكان، **وبمجرّد أن تختار
            // الطريقَ يتغيّر**».)
            //
            // **والبطاقةُ كانت تبدّل باللمسة والخريطةُ لا** — من
            // ضغط الخطَّ رأى لونَه يتغيّر ثمّ لا شيء، **فظنّ أنّ
            // اللمسَ لم يقع.**
            //
            // **والمعاينةُ تسبق الاعتماد لأنّها هي التي تكتبه** —
            // `confirmRoute` تقرأ ما عيّنته `previewRoute`، وكلتاهما
            // كتابةُ حقلٍ لا رحلةُ حال.
            //
            // **ولمسُ المسار الحاليّ لا يفعل شيئاً** — تُرجع
            // `previewRoute` فراغاً فتردّ `confirmRoute` كاذبة،
            // **وهو الصواب: لا تبديلَ إلى ما أنت فيه.**
            onRouteTapped = { routeId ->
                actions.previewRoute(routeId)
                state.driver?.let { driver ->
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
        // ══════════════════════════════════════════════════════════════
        // **«رحلة تجريبيّة» على شاشة الرحلة نفسِها — نسخُ التجربة وحدَها**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-١٠-٠٢: «الرحلة التجريبيّة تكون بنفس شاشة الخريطة،
        //  مو نخترع شاشة وأسلوب جديد» — كما في directory-platform.) **نقاطٌ
        // مصنوعةٌ على خطّ الطريق نفسِه بسرعة موتور (٨٫٣ م/ث) تدخل مجرى
        // الملاحة الحقيقيّ** — فيمشي السهمُ وتلحقه الكاميرا وينطق الصوتُ كما
        // لو كان يقود. **والإصدارُ لا يرى الزرّ** (`ReplayLabIsolationTest`).
        // **والرحلةُ التجريبيّةُ تتبع الطريقَ إذا تبدّل** — بديلٌ اختير أو إعادةُ حساب
        // (بلاغُ المالك ٢٠٢٦-١٠-٠٢). تكمل من أقرب نقطةٍ على الجديد إلى موضعها الآن.
        val activeGeometry = navSession.route?.geometry
        LaunchedEffect(navSession.generation) {
            val g = activeGeometry ?: return@LaunchedEffect
            val hereLat = navSession.nav?.lat
            val hereLng = navSession.nav?.lng
            // **وانتهت عند المتجر والتجربةُ مشغولة؟ تبدأ الساقَ الجديدةَ من أوّلها** (٢٠٢٦-١٠-٠٣) —
            // بعد الاستلام يُرسم الطريقُ إلى الزبون فتمشي عليه بلا ضغطة.
            if (!navSession.replaying) {
                // **ولا تُعاد الساقُ نفسُها** (بلاغُ المالك ٢٠٢٦-١٠-٠٣: «وصلت المتجر ولكن عاد الرحلة من
                // البداية») — عند «وصلت المتجر» يُعاد تركيبُ الطريق إلى المتجر نفسِه، **ونهايتُه تحت السهم.**
                // فلا تبدأ إلّا ساقٌ نهايتُها بعيدةٌ عن موضعه — الطريقُ إلى الزبون بعد الاستلام.
                // **والمقارنةُ بنهاية الساق التي مُشيت لا بموضع السهم** — السهمُ يضيع بعد انتهاء
                // الإعادة (قِيس ٢٠٢٦-١٠-٠٣: أُعيدت الساقُ رغم الشرط الأوّل).
                val done = demoLegEnd
                val endFar = done == null ||
                    com.rahalgo.navigation.GpsQuality.metersBetween(
                        done.lat, done.lng, g.last().lat, g.last().lng,
                    ) > 150.0
                if (demoTrip && g.size >= 2 && endFar) {
                    onReplay(com.rahalgo.navigation.ReplayDrive.fixes(g, startMs = System.currentTimeMillis()))
                }
                return@LaunchedEffect
            }
            if (g.size < 2 || hereLat == null || hereLng == null) {
                return@LaunchedEffect
            }
            var best = 0
            var bestD = Double.MAX_VALUE
            for (i in g.indices) {
                val d = com.rahalgo.navigation.GpsQuality.metersBetween(hereLat, hereLng, g[i].lat, g[i].lng)
                if (d < bestD) {
                    bestD = d
                    best = i
                }
            }
            val rest = g.subList(best, g.size)
            if (rest.size >= 2) {
                onReplayRetarget(
                    com.rahalgo.navigation.ReplayDrive.fixes(rest, startMs = System.currentTimeMillis()),
                )
            }
        }
        val replayRoute = navSession.route?.geometry
            ?: state.navRoute?.geometry
            ?: state.routeLine.map { com.rahalgo.navigation.GeoPoint(it.latitude, it.longitude) }
        if (!inPip && com.rahalgo.driver.BuildConfig.BUILD_TYPE != "release") {
            if (navSession.replaying) {
                Text(
                    text = stringResource(R.string.replay_banner),
                    color = Rahal.colors.canvas,
                    style = MaterialTheme.typography.labelMedium,
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(bottom = 8.dp)
                        .background(Rahal.colors.ink.copy(alpha = 0.85f), Rahal.shape.sm)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
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
            // ══════════════════════════════════════════════════════════
            // **ولا سرعةَ في الرأس** (قرارُ المالك ٢٠٢٦-٠٩-٢٩)
            // ══════════════════════════════════════════════════════════
            //
            // **حُذف عرضُ `كم/س` وحدَه** — انظر الشرحَ في `TripPanel`.
            // **ومعه حُذفت نبضةُ الثانية وحسابُ العرض** فلا تُعاد تركيبَ
            // لوحةٍ كلَّ ثانيةٍ لرقمٍ لا يُعرض.
            //
            // **ومنطقُ السرعة الداخليُّ باقٍ ولم يُمسّ**: `SpeedFilter`
            // و`NavigationSession.currentSpeedKmh` و`LastPoint.speedMps`
            // **تُقرأ في توقيت المناورة والصوت والتقدّم وإعادة الحساب.**
            // **والمسافةُ والوقتُ من التقدّم الحيّ** (فحصُ الملاحة ٤.٢) — كانا رقمَي
            // الخادم الثابتين: «١٫٣ كم · ١ د» من القبول إلى الوصول.
            val live = navSession.nav
            // **والوقتُ يتبع سرعتَه ولا يقفز** (`EtaSmoother`).
            val eta = androidx.compose.runtime.remember(order.id) { com.rahalgo.navigation.EtaSmoother() }
            // **ووصل إلى الهدف؟** — كان يُعرض رقمُ الخادم «١٫١ كم» بعد أن انتهى الخطّ (قِيس ٢٠٢٦-١٠-٠٣).
            val arrivedHere = live != null && (live.arrivedAtTarget || live.progress?.arrived == true)
            TripPanel(
                arrived = arrivedHere,
                state = if (live != null && live.remainingM >= 0) {
                    state.copy(
                        routeM = live.remainingM,
                        routeSec = eta.update(
                            live.remainingM, live.remainingSec, live.speedMps,
                            navSession.route?.totalM ?: -1.0,
                        ),
                    )
                } else {
                    state
                },
            )

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
            // **وحسابٌ موقوفٌ يُقال فوق الرحلة** (٢٠٢٦-١٠-٠٢) — يُكمل هذا الطلبَ
            // وحدَه، **ولا يُفاجأ بعده بأبوابٍ مغلقةٍ بلا سبب.**
            if (state.suspended) {
                TopNotice(stringResource(R.string.suspended_banner), Rahal.colors.danger)
            }
            // ══════════════════════════════════════════════════════════
            // **وخطأُ الخطوة يُرى ولو طُويت البطاقة** (٢٠٢٦-١٠-٠٢)
            // ══════════════════════════════════════════════════════════
            //
            // **كان يُكتب في البطاقة وحدَها** — ومن طواها ليرى الطريقَ ثمّ سقط
            // وصولٌ تلقائيٌّ أو خطوةٌ **لم يرَ شيئاً، فظنّ أنّها وقعت.** (ولافتةُ
            // العرض تقول خطأها بنفسها.)
            if (!TripCollapse.bottom && state.onRouteOffer == null) {
                if (state.error.isNotEmpty()) {
                    TopNotice(state.error, Rahal.colors.danger)
                } else if (state.notice.isNotEmpty()) {
                    TopNotice(state.notice, Rahal.colors.brand)
                }
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
            // ══════════════════════════════════════════════════════════
            // **وتعذّرُ التوسيط يُقال بسببه** (`MLW-14`)
            // ══════════════════════════════════════════════════════════
            //
            // **ومن ضغط «موقعي» فاتّسعت الخريطةُ إلى مكانٍ ليس هو
            // ظنّها معطوبة** — **والسببُ أنّ موضعَه لم يصل**، **وأكثرُ
            // ما يكون ذلك بخدمةٍ مطفأة.**
            locating.problem?.let { why ->
                androidx.compose.material3.Text(
                    text = com.rahalgo.ui.LocatingText.message(hereContext, why),
                    style = MaterialTheme.typography.bodyMedium,
                    color = Rahal.colors.ink,
                    modifier = Modifier
                        .padding(horizontal = 14.dp)
                        .clip(Rahal.shape.md)
                        .background(Rahal.colors.warnTint)
                        .clickable {
                            // **والعلاجُ فعلٌ** — **وخدمةٌ مطفأةٌ
                            // تُفتح صفحتُها، ولا تُفتح صفحةُ التطبيق.**
                            if (com.rahalgo.ui.fixProblem(hereContext, why)) locating.reset()
                        }
                        .padding(12.dp),
                )
            }
            // **«رحلة تجريبيّة» فوق زرّ تحديد الموقع** (طلبُ المالك ٢٠٢٦-١٠-٠٢: «مشان
            // يكون مبيّن») — في نسخ التجربة وحدَها.
            // **وفي الخاصّ قبل الشراء أيضاً** (٢٠٢٦-١٠-٠٣) — لا طريقَ بعد، **فيُشغَّل ويبدأ وحدَه**
            // حين يُرسم الطريقُ إلى الزبون بعد «تم شراء المطلوب».
            val customBeforeRoute = state.order?.kind == "custom" && state.order?.status == "assigned"
            if (com.rahalgo.driver.BuildConfig.BUILD_TYPE != "release" &&
                (replayRoute.size >= 2 || navSession.replaying || customBeforeRoute || demoTrip)
            ) {
                Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterEnd) {
                    ReplayButton(
                        running = navSession.replaying || demoTrip,
                        onStart = {
                            if (replayRoute.size >= 2 && !customBeforeRoute) {
                                onReplay(
                                    com.rahalgo.navigation.ReplayDrive.fixes(
                                        replayRoute, startMs = System.currentTimeMillis(),
                                    ),
                                )
                            } else {
                                onArmDemo()
                            }
                        },
                        onStop = { onReplay(emptyList()) },
                    )
                }
            }
            MapButtons(
                onRecenter = {
                    // **ولا يُوسَّط على مجهول** — **والسببُ يُقال
                    // باسمه**: **مطفأةٌ خدمتُه أو لم يصل قياسٌ بعد.**
                    if (state.driver != null) {
                        locating.reset()
                        recenter++
                    } else if (locating.start()) {
                        locating.failed(
                            if (!com.rahalgo.ui.locationServiceEnabled(hereContext)) {
                                Locating.Problem.SERVICE_OFF
                            } else {
                                Locating.Problem.UNAVAILABLE
                            },
                        )
                    }
                },
                onChat = actions.chat,
                showChat = state.order?.kind != "merchant_delivery",
                chatting = chat != null,
                chatUnread = chatUnread,
                voiceMuted = voiceMuted,
                onVoice = { actions.toggleVoice() },
            )

            // ══════════════════════════════════════════════════════════
            // **لا زرَّ «رحلة تجريبيّة» على شاشة الرحلة الحقيقيّة** (قرارُ
            // المالك ٢٠٢٦-٠٩-٢٨)
            // ══════════════════════════════════════════════════════════
            //
            // **حُذف من المسار الذي يبلغه السائق**: لا مدخلَ إعادةٍ على شاشة
            // رحلةٍ حقيقيّة، حتّى في بناء التطوير. **ومحرّكُ الإعادة يبقى في
            // `src/debug` (ReplayTripActivity، يُطلَق بـ`adb` للهندسة)** —
            // مركونٌ ولا يبلغه سائق، والإصدارُ لا يحمله أصلاً.
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
        AgreeDialog(
            // **سلطةُ الأجرة من لقطة الطلب** (Batch 2c) — لا من إعدادٍ عامّ.
            feeSource = order.customFeeSource,
            feeSnapshot = order.customFeeSnapshot,
            driverMayChange = order.customDriverMayChangeFee,
            currentGoods = order.customGoods,
            currentFee = order.customFee,
            step = if (order.customFee == null) "fee" else "goods",
            onConfirm = actions.agree,
            onDismiss = actions.dismissAgree,
        )
    }

    if (state.emergencyOpen) {
        EmergencyDialog(
            busy = state.emergencyBusy,
            error = state.emergencyError,
            onConfirm = actions.emergency,
            onRetry = actions.retryEmergency,
            onDismiss = actions.dismissEmergency,
        )
    }

    // **والقائمةُ لا تتبدّل تحت إصبعه** (فحصُ جهاز المالك ٢٠٢٦-١٠-٠٣) — فتحها في الطريق ثمّ
    // سُجّل وصولُه فصارت أسبابُ الباب مكانَ أسباب الطريق وهو يقرأ. **فتُغلق ويُقال له لماذا.**
    val stageNow = state.order?.status.orEmpty()
    val stageChanged = stringResource(R.string.problem_stage_changed)
    LaunchedEffect(stageNow) {
        if (state.failReasons != null && state.problemStatus.isNotEmpty() && stageNow != state.problemStatus) {
            actions.dismissFail()
            com.rahalgo.ui.Flash.ok(stageChanged)
        }
    }
    if (state.failReasons != null) {
        FailDialog(
            reasons = state.failReasons,
            status = state.problemStatus,
            loadedAtMs = state.reasonsAtMs,
            error = state.problemError,
            onPick = actions.fail,
            onDismiss = actions.dismissFail,
            onMine = actions.problem,
            onRetry = actions.retryProblem,
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
            // **وأجرتُه هو لا نقدُ الزبون** (٢٠٢٦-١٠-٠٢) — كان يُعرض `cash_due`
            // فيقرأ ما يقبضه للمتجر على أنّه ما يكسبه. **وكما في بطاقة الطلب**:
            // الخاصُّ أجرتُه تُتّفق لاحقاً.
            text = offer.merchantName.ifBlank { stringResource(R.string.nav_custom_order) } +
                " · " + stringResource(R.string.card_fee) + " " +
                if (offer.kind == "custom") stringResource(R.string.card_agreed_later) else money(offer.deliveryFee),
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
        // **و«لاحقاً» تقول ما يقع فعلا** — تُزيح الشريطَ عن الطريق وهو
        // يقود، **والطلبُ لمن يأخذه**: يُقال للخادم فينتقل الدورُ فوراً
        // (٢٠٢٦-١٠-٠٢) — **كان إخفاءً محلّيّاً يُبقي العرضَ عليه مهلتَه كلَّها.**
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

/** **سطرُ خبرٍ فوق الخريطة** — يُقرأ ولو طُويت البطاقة. */
@Composable
internal fun TopNotice(text: String, ground: Color) {
    Text(
        text = text,
        color = Color.White,
        fontWeight = FontWeight.Bold,
        style = MaterialTheme.typography.bodyMedium,
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 4.dp)
            .clip(Rahal.shape.md)
            .background(ground)
            .padding(horizontal = 14.dp, vertical = 10.dp),
    )
}

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
    // ══════════════════════════════════════════════════════════════════
    // **وحالٌ واحدةٌ للبطاقة — لا حالان**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وفصلتُ المقبضَ عن `expanded` بالأمس** — فصار المقبضُ يطوي
    // الأزرارَ والتبويبات، **و`expanded` لا يفتحها أحد**: تبقى مغلقةً
    // إلّا عند باب المتجر أو باب الزبون. **فاختفت أزرارُ المرحلة عن
    // طلبٍ مُسنَد.**
    //
    // **ومقبضٌ واحدٌ يقود شيئاً واحداً** — وحالان لبطاقةٍ واحدةٍ
    // تفترقان.
    val expanded = TripCollapse.bottom

    // **ومشوارُ الإرجاع بطاقتُه وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — وجهةٌ وزرٌّ واحد.
    if (isReturnTrip(order)) {
        ReturnCard(order = order, state = state, actions = actions, modifier = modifier)
        return
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
                    // **والسحبُ يطوي البطاقةَ كلَّها** — (طلبُ المالك
                    // ٢٠٢٦-٠٩-٠١): معها الأزرارُ وشريطُ التبويبات،
                    // **لتكون شاشةً كبيرةً للخرائط.**
                    if (dy > 6f) TripCollapse.bottom = false
                    if (dy < -6f) TripCollapse.bottom = true
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
                .clickable { TripCollapse.bottom = !TripCollapse.bottom },
        )
        Spacer(Modifier.height(10.dp))

        // ══════════════════════════════════════════════════════════════
        // **الطلبُ الخاصّ على خطوتين** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — الأجرةُ ثمّ (في المشتريات)
        // ثمنُ البضاعة، والزبونُ يوافق على كلٍّ. **والشراءُ بعدهما وحدَهما.**
        // ══════════════════════════════════════════════════════════════
        val customOpen = order.kind == "custom" && order.status == "assigned"
        val feeDone = order.customFee != null
        val quoteOk = feeDone && order.quoteConfirmedVersion == order.quoteVersion
        val needGoods = customOpen && order.customGoodsPending
        val buyBlocked = customOpen && (!feeDone || !quoteOk || needGoods)
        val next = nextAction(order.status, order.kind == "custom")
        // **واسمُ الزرّ بنوع الطلب** — «استلمت الأمانة» أو «تم شراء المطلوب».
        val nextLabel = when {
            next == null -> 0
            order.kind == "custom" && next.status == "picked_up" ->
                if (order.customMode == "amanah") R.string.step_bought_amanah else R.string.step_bought_purchase
            else -> next.label
        }
        // **وزرُّ «وصلت» لا يظهر قبل الوصول** — طلبُ المالك ٢٠٢٦-١٠-٠٢. ويُضغط
        // وحدَه بعد ٣٠ ثانيةً عند الوجهة (`OrdersViewModel.watchArrival`).
        // **ومن لا يُعرف موضعُه لا يُحبس عن الزرّ.**
        // **والخاصُّ عند الزبون كالعاديّ** (٢٠٢٦-١٠-٠٣).
        val arriveLater = next != null && (order.kind != "custom" || next.status == "at_dropoff") &&
            next.status in setOf("at_pickup", "at_dropoff") &&
            state.locationKnown && !state.nearDestination ||
            // **ولا «استلمت الطلب» وقرارُ المتجر بيد الإدارة** (٢٠٢٦-١٠-٠٣).
            state.awaitingOffice && next?.status == "picked_up"
        val onNext: () -> Unit = {
            if (next != null) {
                // **والتسليم يمرّ بالصورة إن طلبها المحرّك** — وإلّا
                // ردّ «يلزم إثبات» بعد أن ظنّ صاحبه أنّه أنهى.
                if (next.status == "delivered" && state.requirePhoto) actions.capture() else actions.step(next.status)
            }
        }

        // **وما تحت المقبض يُطوى معه** — والمقبضُ وحدَه يبقى، **فلا
        // يُحبَس صاحبُه عن إعادتها.** **إلّا الزرَّ الأساسيّ** (٢٠٢٦-١٠-٠٣): من طوى ليرى
        // الطريقَ لا يُحبَس عن «وصلت» أو «تم التسليم».
        if (!TripCollapse.bottom) {
            if (next != null && !buyBlocked && !arriveLater) {
                SmallAction(
                    icon = R.drawable.ic_check_circle,
                    label = nextLabel,
                    tone = Tone.Brand,
                    onClick = onNext,
                    enabled = !state.busy,
                    busy = state.busy,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            return@Column
        }

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
        // ══════════════════════════════════════════════════════════════
        // **و«لدي توصيلة»: ما يحمل، وممّن يقبض، وأين الباب** (٢٠٢٦-١٠-٠٢)
        // ══════════════════════════════════════════════════════════════
        //
        // **كانت الثلاثةُ في شاشةٍ لا تُفتح** (`OrderDetailScreen`، حُذفت) — والبطاقةُ
        // هنا هي ما يقرؤه. **ومن قبض «أنا نقداً» يقبضها من المتجر قبل أن يمضي.**
        val pickedUp = state.step >= TripStep.PICKED_UP
        val cashFrom = cashFrom(order, pickedUp)
        // ══════════════════════════════════════════════════════════════
        // **وفي الطريق إلى المتجر: متى يجهز، ورقمُه** (٢٠٢٦-١٠-٠٢)
        // ══════════════════════════════════════════════════════════════
        //
        // **كان `ready_at` و`prep_minutes` و`merchant_phone` تصل ولا تُعرض** —
        // فيقف عند المطبخ لا يعرف متى، **ولا يملك أن يسأل.** **ورقمُ المتجر
        // مكشوفٌ للسائق عمداً** (`authz.FieldPolicy`)؛ **ورقمُ الزبون لا يصله أبداً.**
        if (!pickedUp && order.kind != "custom") {
            val prep = when (val p = prepState(order)) {
                PrepState.Ready -> stringResource(R.string.trip_store_ready)
                is PrepState.Around -> stringResource(R.string.trip_store_ready_at, p.hhmm)
                PrepState.Unknown -> ""
            }
            val phone = order.merchantPhone?.takeIf { it.isNotBlank() }
            if (prep.isNotEmpty() || phone != null) {
                Spacer(Modifier.height(8.dp))
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(prep, color = Rahal.colors.inkMuted, modifier = Modifier.weight(1f))
                    if (phone != null) {
                        val ctx = LocalContext.current
                        // **زرُّ اتّصالٍ يُرى زرَّ اتّصال** (طلبُ المالك ٢٠٢٦-١٠-٠٣) — إطارٌ أخضرُ
                        // وسمّاعة، لا نصٌّ يُقرأ رابطاً.
                        com.rahalgo.ui.RahalOutlineButton(tone = Tone.Success, onClick = {
                            runCatching {
                                ctx.startActivity(
                                    android.content.Intent(
                                        android.content.Intent.ACTION_DIAL,
                                        android.net.Uri.parse("tel:$phone"),
                                    ),
                                )
                            }
                        }) {
                            Icon(
                                painter = painterResource(com.rahalgo.ui.R.drawable.ic_phone),
                                contentDescription = null,
                                modifier = Modifier.size(18.dp),
                            )
                            Spacer(Modifier.width(6.dp))
                            Text(stringResource(R.string.trip_call_store), fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }
        }
        if (order.parcelNote.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(stringResource(R.string.detail_parcel), color = Rahal.colors.inkMuted)
                Text(order.parcelNote, fontWeight = FontWeight.Bold)
            }
        }
        if (order.driverNote.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(stringResource(R.string.detail_driver_note), color = Rahal.colors.inkMuted)
                Spacer(Modifier.width(12.dp))
                Text(
                    order.driverNote,
                    fontWeight = FontWeight.Bold,
                    color = Rahal.colors.brand,
                    textAlign = TextAlign.End,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        if (cashFrom == CashFrom.STORE) {
            Spacer(Modifier.height(8.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(
                    stringResource(R.string.detail_fee_from_store),
                    color = Rahal.colors.inkMuted,
                    modifier = Modifier.weight(1f),
                )
                Text(money(order.cashDue), fontWeight = FontWeight.Bold, color = Rahal.colors.brand)
            }
        }
        if (!order.dropoffKnown) {
            Spacer(Modifier.height(8.dp))
            Text(stringResource(R.string.detail_no_point), color = Rahal.colors.danger, fontWeight = FontWeight.Bold)
        }
        if (pickedUp) {
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
            // **و«أنا نقداً» قُبضت عند المتجر** — فلا يُطلب من المستلِم شيء.
            val due = cashFrom == CashFrom.RECIPIENT
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

        // **والخطأُ يُقال في موضعٍ واحد** (كان يُعرض مرّتين — فحصُ دورة السائق)، **وهنا
        // الخبرُ**: «وصل بلاغُك للإدارة».
        if (state.notice.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Text(state.notice, color = Rahal.colors.brand, fontWeight = FontWeight.Bold)
        }

        // ══════════════════════════════════════════════════════════════
        // **«لدي توصيلة»: اتّصالٌ وواتساب بالمستلِم** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
        // ══════════════════════════════════════════════════════════════
        //
        // «زرُّ اتّصالٍ وزرُّ واتس اب أسرع» — **والرسالةُ جاهزةٌ** بنصّ المالك واسمِ المتجر:
        // يرسل له المستلمُ موقعَه هناك إن لم تكن نقطة، فيفتحه السائقُ كما يناسبه.
        if (order.kind == "merchant_delivery" && order.recipientPhone.isNotBlank()) {
            val ctx = LocalContext.current
            val phone = order.recipientPhone
            val hello = stringResource(R.string.recipient_wa_message, order.merchantName)
            Spacer(Modifier.height(10.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                com.rahalgo.ui.RahalOutlineButton(
                    tone = Tone.Success,
                    modifier = Modifier.weight(1f),
                    onClick = {
                        runCatching {
                            ctx.startActivity(
                                android.content.Intent(
                                    android.content.Intent.ACTION_DIAL,
                                    android.net.Uri.parse("tel:$phone"),
                                ),
                            )
                        }
                    },
                ) {
                    Icon(
                        painter = painterResource(com.rahalgo.ui.R.drawable.ic_phone),
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(stringResource(R.string.recipient_call), fontWeight = FontWeight.Bold)
                }
                com.rahalgo.ui.RahalOutlineButton(
                    tone = Tone.Success,
                    modifier = Modifier.weight(1f),
                    onClick = {
                        runCatching {
                            ctx.startActivity(
                                android.content.Intent(
                                    android.content.Intent.ACTION_VIEW,
                                    android.net.Uri.parse(
                                        "https://wa.me/" + waNumber(phone) + "?text=" +
                                            android.net.Uri.encode(hello),
                                    ),
                                ),
                            )
                        }
                    },
                ) {
                    Text(stringResource(R.string.recipient_whatsapp), fontWeight = FontWeight.Bold)
                }
            }
        }

        // **ولا عبارةَ «وصلت»** — طلبُ المالك ٢٠٢٦-١٠-٠٢: «ما لها داعٍ». الزرُّ يظهر عند
        // الوصول ويُسجَّل وحدَه بعد ٣٠ ثانية، وذاك يكفي.

        // **وأمرُ الإدارة عند الباب يُقرأ واضحاً** (٢٠٢٦-١٠-٠٢: «الطلبُ يبقى مع السائق حتّى
        // تحلّ الإدارةُ القصّة… يصل أمرٌ للسائق»).
        if (order.doorInstruction == "deliver_now") {
            Spacer(Modifier.height(10.dp))
            Text(
                // **وقبل الباب «أكمل التوصيل»** (٢٠٢٦-١٠-٠٣) — الأمرُ نفسُه والسائقُ في الطريق.
                text = stringResource(
                    when (order.status) {
                        "at_dropoff" -> R.string.door_deliver_now
                        "at_pickup" -> R.string.door_collect
                        // **وقبل المتجر «أكمل الطلب»** (٢٠٢٦-١٠-٠٣) — الزبونُ لم يُلغِ.
                        "assigned" -> R.string.door_keep
                        else -> R.string.door_continue
                    },
                ) +
                    if (order.doorNote.isNotBlank()) " — " + order.doorNote else "",
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
            // **والأزرارُ تنطوي مع اللوحة** — (بلاغُ المالك ٢٠٢٦-٠٨-٣١:
            // «لازم حتّى الأزرارُ تنزل تختفي مشان يصير مجالٌ أوسعُ
            // للخريطة»). **ومن سحب اللوحةَ ليرى الطريقَ كان يربح سطراً
            // واحداً** والأزرارُ تحتها كما هي.
            //
            // **وتعود بلمسةٍ على المقبض** — فلا يُحبَس عن مرحلته.
            androidx.compose.animation.AnimatedVisibility(visible = TripCollapse.bottom) {
            Spacer(Modifier.height(14.dp))
            }
            androidx.compose.animation.AnimatedVisibility(visible = TripCollapse.bottom) {
            // ══════════════════════════════════════════════════════════════
            // **عمودٌ لا تراكُب** (بلاغُ المالك ٢٠٢٦-١٠-٠٣: «الشاشة واقفة على وثّق الاتفاق
            // ولا يستطيع السائقُ فعلَ شيء»)
            // ══════════════════════════════════════════════════════════════
            //
            // **`AnimatedVisibility` يضع أبناءَه فوق بعضهم** — وكان فيه ابنان: صفُّ الأفعال
            // وزرُّ الاتّفاق. **فغطّى زرُّ الاتّفاق «اشتريتُ الطلب» و«لدي مشكلة» تماماً** في
            // الطلب الخاصّ (قِيس بتفريغ الشاشة: لا زرَّ تحته). **فيُجمعان في عمود.**
            Column(Modifier.fillMaxWidth()) {
            // **وما وُثّق يُرى موثَّقاً** — «✓ أجرة التوصيل» ثمّ «✓ ثمن البضاعة».
            if (customOpen && feeDone) {
                Text(
                    stringResource(R.string.agree_fee_done, money(order.customFee ?: 0)),
                    color = Rahal.colors.success,
                    modifier = Modifier.fillMaxWidth(),
                )
                if (order.customMode != "amanah" && !order.customGoodsPending) {
                    Text(
                        stringResource(R.string.agree_goods_done, money(order.customGoods ?: 0)),
                        color = Rahal.colors.success,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                Spacer(Modifier.height(6.dp))
            }
            // **«الزبونُ أكّد الطلب — قم بالشراء»** (طلبُ المالك ٢٠٢٦-١٠-٠٣) — يعرف أنّ دورَه جاء.
            if (customOpen && quoteOk && !needGoods) {
                Text(
                    stringResource(
                        if (order.customMode == "amanah") R.string.drv_customer_confirmed_amanah
                        else R.string.drv_customer_confirmed,
                    ),
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(bottom = 10.dp),
                )
            }
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // ══════════════════════════════════════════════════════════
                // **ولا يبدأ الشراءُ قبل أن يؤكّد الزبونُ العرضَ** (Batch 2c)
                // ══════════════════════════════════════════════════════════
                //
                // **المحرّكُ يمنع `picked_up` حتّى يؤكّد الزبونُ النسخةَ الحاليّة**
                // (`quote_not_confirmed`) — **والشاشةُ تقول ذلك بدل أن يُضغط
                // فيُردّ.** ويبقى زرُّ الاتّفاق ليعدّل إن لزم. **والحارسُ في
                // المحرّك لا في إخفاء الزرّ.**
                val awaitingConfirm = customOpen && feeDone && !quoteOk
                if (customOpen && !feeDone) {
                    // **زرُّ التوثيق تحتُ يكفي** — ولا زرَّ شراءٍ قبله.
                    Spacer(Modifier.weight(1f))
                } else if (needGoods && quoteOk) {
                    Text(
                        stringResource(R.string.drv_buy_then_document),
                        color = Rahal.colors.brand,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.weight(1f),
                    )
                } else if (awaitingConfirm) {
                    Text(
                        // **والأمانةُ بلا ثمن** (قِيس في دورة المحاكي ٢٠٢٦-١٠-٠٣): كانت تقول «ثمن البضاعة».
                        stringResource(
                            if (order.customGoodsPending || order.customMode == "amanah") R.string.drv_awaiting_fee
                            else R.string.drv_awaiting_goods,
                        ),
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.weight(1f),
                    )
                } else if (arriveLater) {
                    // **فراغٌ لا عبارة** — طلبُ المالك: «العبارةُ ما لها داعٍ أصلاً».
                    Spacer(Modifier.weight(1f))
                } else if (next != null) {
                    SmallAction(
                        icon = R.drawable.ic_check_circle,
                        label = nextLabel,
                        tone = Tone.Brand,
                        onClick = onNext,
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

                // **ولا «إعادة للطابور»** — قرارُ المالك ٢٠٢٦-١٠-٠٢: «مجرّد ما ينطلق السائق ما يصير
                // ينعاد للطابور». **والتركُ بسببٍ من «لدي مشكلة»** وحدَه (تعطّلت · حادث · ظرفٌ قاهر).
            }

        // ══════════════════════════════════════════════════════════════
        // **والاتّفاقُ فعلٌ أوّلٌ لا فعلٌ جانبيّ — فيُعطى شكلَه**
        // ══════════════════════════════════════════════════════════════
        //
        // (شهده المالك ٢٠٢٦-٠٩-٢٩ على جهازه: «زر وثق الاتفاق فوق زر لدي
        //  مشكلة غير واضح، صلحه وصلح مكانه».)
        //
        // **كان `RahalTextButton` في صفٍّ ثانٍ بلا فاصل** — فيهبط على صفّ
        // الأفعال فيبدو عائماً بينها (قُيس: الثلاثةُ عند `cy≈1970` نفسِها).
        // **وزرُّ النصّ أخفضُ توكيدٍ في النظام**، فيُقرأ رابطاً لا فعلاً.
        //
        // **والحقيقةُ أنّه الفعلُ الوحيدُ الذي يُقدّم الطلبَ في هذه المرحلة**:
        // ما لم يُوثَّق الاتّفاقُ لا شراءَ ولا تسليم، **والطلبُ واقفٌ على
        // الزبون بعده.** فجُعل **زرّاً أساسيّاً بعرض الشاشة فوق صفّ الأفعال،
        // بفاصلٍ يفصله عنه.**
        //
        // **والاتّفاق للطلب الخاصّ وحدَه** — العاديّ سعرُه معروف سلفا،
        // **وزرٌّ يظهر فيه يسأل عمّا لا يُسأل عنه.**
        // **وقبل الاستلام وحدَه** (Batch 2c) — بعده يُقفَل السعرُ ولا
        // يعدّله السائق (يفرضه المحرّك؛ والزرُّ يُخفى كذلك).
        // **ويختفي بعد التوثيق** (طلبُ المالك ٢٠٢٦-١٠-٠٣: «وثّق الاتفاق هنا لازم يختفي»).
        if (customOpen && (!feeDone || (needGoods && quoteOk))) {
            Spacer(Modifier.height(12.dp))
            RahalButton(
                onClick = actions.askAgree,
                enabled = !state.busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(if (!feeDone) R.string.agree_fee_button else R.string.agree_goods_button))
            }
        }
            }
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

/** **كم بين بحثين عن طريقٍ أفضل** — كغوغل تقريباً. */
private const val BETTER_ROUTE_EVERY_MS = 120_000L

/**
 * **رقمُ واتساب دوليٌّ بلا «+»** — `wa.me` لا يفهم غيرَه. **والمحلّيُّ السوريّ** (`09…`) يصير
 * `9639…`، **والدوليُّ** (`+963…` أو `00963…`) يُنزع رأسُه.
 */
internal fun waNumber(phone: String): String {
    val digits = phone.filter { it.isDigit() }
    return when {
        digits.startsWith("00") -> digits.drop(2)
        digits.startsWith("0") -> "963" + digits.drop(1)
        else -> digits
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ مشوار الإرجاع** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الوجهةُ والعنوانُ وزرٌّ واحد: «سلّمت البضاعة».** لا مالَ يُقبض ولا زبونَ يُتّصل به —
 * الطلبُ أُنهي، **والسائقُ يُرجع البضاعة إلى المكتب أو المتجر.** والزرُّ ظاهرٌ دائماً:
 * **لا وصولَ تلقائيّاً يُنتظر** (المكتبُ قد لا يكون مدبَّساً).
 */
@Composable
private fun ReturnCard(
    order: DriverOrder,
    state: TripState,
    actions: TripActions,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sheet)
            .background(Rahal.colors.canvas)
            .padding(horizontal = 20.dp, vertical = 14.dp),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    painter = painterResource(R.drawable.ic_store),
                    contentDescription = null,
                    tint = Rahal.colors.accent,
                    modifier = Modifier.size(22.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = returnTitle(order),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
            }
            Text("#${order.number}", color = Rahal.colors.inkMuted)
        }
        if (order.addressText.isNotBlank()) {
            Spacer(Modifier.height(4.dp))
            Text(text = order.addressText, color = Rahal.colors.inkMuted)
        }
        if (!order.dropoffKnown) {
            Spacer(Modifier.height(4.dp))
            Text(text = stringResource(R.string.trip_return_no_point), color = Rahal.colors.inkMuted)
        }
        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(text = state.error, color = Rahal.colors.danger)
        }
        Spacer(Modifier.height(10.dp))
        SmallAction(
            icon = R.drawable.ic_check_circle,
            label = R.string.step_goods_handed,
            tone = Tone.Brand,
            onClick = actions.handGoods,
            enabled = !state.busy,
            busy = state.busy,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

/** **عنوانُ مشوار الإرجاع** — «أرجِع البضاعة إلى المكتب» أو «… إلى {المتجر}». */
@Composable
internal fun returnTitle(order: DriverOrder): String =
    if (order.returnTo == "store") {
        stringResource(
            R.string.trip_return_store,
            order.returnLabel.ifBlank { stringResource(R.string.trip_return_store_fallback) },
        )
    } else {
        stringResource(R.string.trip_return_office)
    }
