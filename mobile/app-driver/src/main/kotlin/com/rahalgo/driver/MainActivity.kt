package com.rahalgo.driver

import android.os.Bundle
import com.rahalgo.driver.wallet.WalletViewModel
import com.rahalgo.driver.wallet.WalletScreen
import com.rahalgo.driver.rating.RatingViewModel
import com.rahalgo.driver.rating.RatingScreen
import androidx.activity.compose.BackHandler
import com.rahalgo.driver.account.AccountViewModel
import com.rahalgo.driver.account.AccountScreen
import android.provider.Settings
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandCanvas
import com.rahalgo.design.RahalGoTheme
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.intro.BrandIntro
import com.rahalgo.driver.login.LoginActions
import com.rahalgo.driver.login.LoginScreen
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.compose.LifecycleResumeEffect
import com.rahalgo.design.InkMuted
import androidx.core.content.ContextCompat
import androidx.core.view.WindowCompat
import com.rahalgo.driver.home.HomeActions
import com.rahalgo.driver.home.HomeScreen
import com.rahalgo.driver.home.HomeViewModel
import com.rahalgo.driver.location.LastPoint
import com.rahalgo.driver.data.Backend
import com.rahalgo.driver.location.LocationPermission
import com.rahalgo.driver.ui.InboxSheet
import com.rahalgo.driver.ui.TopBar
import com.rahalgo.driver.login.LoginViewModel
import com.rahalgo.driver.orders.DetailActions
import com.rahalgo.driver.orders.OrderDetailScreen
import com.rahalgo.driver.orders.OrdersActions
import com.rahalgo.driver.orders.OrdersScreen
import com.rahalgo.driver.orders.OrdersViewModel
import com.rahalgo.driver.trip.Proof
import com.rahalgo.driver.trip.ChatActions
import com.rahalgo.driver.trip.TripActions
import com.rahalgo.driver.trip.TripScreen
import com.rahalgo.driver.trip.TripState
import com.rahalgo.driver.login.ResetActions
import com.rahalgo.driver.login.ResetScreen

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إقلاع تطبيق السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * ```
 * فتحٌ باردٌ  →  شاشةُ النظام  →  حركةُ العلامة  →  الوجهة
 * ```
 *
 * **وشاشةُ النظام قصيرةٌ كما يريدها أندرويد** — لا انتظارَ مصطنعٌ فيها.
 * تُسلَّم فورَ أن يبدأ Compose، ثمّ تعمل الحركةُ في شاشتنا نحن.
 *
 * # وحركةُ الافتتاح لا تتكرّر
 *
 * (طلب المالك: «لا أريد مشاهدة Intro كاملة كل مرة يرجع المستخدم من
 *  الخلفية».)
 *
 * **والحالةُ في الكائن المرافق لا في الشاشة** — فتبقى ما بقيت العمليةُ
 * حيّة. **فإعادةُ بناء الشاشة** (دوران الجهاز · تبدّل اللغة · عودةٌ من
 * الخلفية) **لا تُعيدها**، ولا يراها المستخدمُ إلّا حين يُقلع النظامُ
 * العمليةَ من جديد — وهو معنى «الفتح البارد» بالضبط.
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        // **قبل `super`** — وهي شرطُ المكتبة: تُركّب على النافذة قبل أن
        // يُنشئ النظامُ محتواها.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        // ══════════════════════════════════════════════════════════════
        // **وشريطُ النظام يبقى** — الساعةُ والشبكةُ والبطّاريّة
        // ══════════════════════════════════════════════════════════════
        //
        // **جُرّب مدُّ النافذة تحته فرُفض** (المالك ٢٠٢٦-٠٨-١٢: «أنت شلت
        // الشريط الخاصّ بالبطارية وهذا غلط، ما هيك طلبي أبدا»).
        //
        // **والذي كان يُطلب رفعُه شريطٌ آخرُ تحته** — فراغٌ أبيضُ من
        // صنعنا لا من صنع النظام.
        //
        // **وأيقوناتُ النظام داكنة** — أرضُ شريطه فاتحة، **وأيقونةٌ
        // بيضاءُ عليها تختفي.**
        WindowCompat.getInsetsController(window, window.decorView)
            .isAppearanceLightStatusBars = true
        setContent { DriverApp() }
    }

    companion object {
        /** **أعُرضت الحركةُ في عمر هذه العمليّة؟** */
        @Volatile
        var introShown: Boolean = false
    }
}

@Composable
private fun DriverApp() {
    val context = LocalContext.current
    // **وحركاتُ النظام تُقرأ من إعداداته** — من أطفأها أراد ذلك،
    // **وتطبيقٌ يتجاهله يُقرأ معطوباً لا أنيقاً.**
    val reduceMotion = remember {
        Settings.Global.getFloat(
            context.contentResolver,
            Settings.Global.ANIMATOR_DURATION_SCALE,
            1f,
        ) == 0f
    }
    var showIntro by remember { mutableStateOf(!MainActivity.introShown) }

    RahalGoTheme {
        Box(
            Modifier
                .fillMaxSize()
                // **وأرضٌ صريحةٌ تحت الاثنتين** — فلا ومضةٌ بيضاءُ ولا
                // سوداءُ في لحظة التبديل.
                .background(BrandCanvas)
        ) {
            Destination()

            AnimatedVisibility(
                visible = showIntro,
                enter = fadeIn(tween(0)),
                // **وتخرج صاعدةً لا مختفيةً فجأةً** — فيبدو أنّ الشاشةَ
                // التالية خرجت من الحركة نفسِها.
                exit = fadeOut(tween(200)) + slideOutVertically(tween(220)) { -it / 12 },
            ) {
                BrandIntro(
                    tagline = stringResource(R.string.intro_tagline),
                    brandWord = stringResource(R.string.intro_tagline_brand),
                    reduceMotion = reduceMotion,
                    onFinished = {
                        MainActivity.introShown = true
                        showIntro = false
                    },
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الوجهة — تُقرَّر هنا لا في شاشة الافتتاح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شرط المالك: «يجب ألّا تقرّر Splash بنفسها أين يذهب المستخدم».)
 *
 * **وثلاث حالات**: يُفحص التوكن المحفوظ · فإن صحّ فلوحته · وإلّا فالدخول.
 *
 * **وشاشة انتظار أثناء الفحص لا شاشة دخول**: من له جلسة حيّة **لا يُرى
 * شاشة دخول لحظة** ثمّ تُبدَّل — وهو وميض يُقرأ عطبا.
 */
@Composable
private fun Destination() {
    val vm: LoginViewModel = viewModel()

    when {
        // **وانتظارٌ يُرى لا بياضٌ صامت** — الخادم النائم يستيقظ في
        // نصف دقيقة (قيس ٢٠٢٦-٠٨-١٢: أربعون ثانية)، **وشاشة بيضاء هذه
        // المدّة تُقرأ عطبا** فيُعاد فتح التطبيق مرّة بعد مرّة.
        vm.restoring -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator()
        }

        // **ومن دخل يُسلَّم للوحته** — ونموذجها مستقلّ عن نموذج الدخول:
        // **حال الوردية والمال لا يخصّ بابا دخل منه.**
        // **ومن دخل يُسلَّم للوحته.**
        vm.user != null -> SignedIn(onLogout = vm::logout)

        // **وجلسة محفوظة لم تُتحقَّق: شاشة اتّصال لا شاشة دخول.**
        vm.offline -> Offline(onRetry = vm::retryRestore)

        // **والاستعادة تسبق الدخول في الترتيب** — من ضغط «نسيت» يرى
        // شاشتها، **ولو قُدّم الدخول عليها لبقيت الشاشة مكانها** والزرّ
        // لا يفعل شيئا.
        vm.reset != null -> ResetScreen(
            state = vm.reset!!,
            actions = ResetActions(
                setPhone = vm::setResetPhone,
                sendCode = vm::sendResetCode,
                verifyCode = vm::verifyResetCode,
                confirm = vm::confirmReset,
                cancel = vm::closeReset,
            ),
        )

        else -> LoginScreen(
            state = vm.state,
            actions = LoginActions(
                login = vm::login,
                setMode = vm::setMode,
                sendCode = vm::sendLoginCode,
                verifyCode = vm::verifyLoginCode,
                resetCode = vm::clearCode,
                forgot = vm::openReset,
            ),
        )
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما بعد الدخول — تبويبتان لا أكثر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا تبويب لشاشة لم تُبنَ**: زرّ يفتح فراغا يُقرأ عطبا، **ومن ملأ
 * الشريط بأسماء قادمة** جعل نصفه لا يعمل. **يُضاف التبويب مع شاشته.**
 *
 * # والحال يُعاد قراءته عند كلّ عودة
 *
 * **السائق يخرج من التطبيق ويعود بعد ربع ساعة** — وطلب عُرض عليه قد
 * أُخذ، وورديّته قد أُغلقت من المكتب. **وشاشة تعرض ما كان** أسوأ من
 * شاشة تُحمّل.
 */
@Composable
private fun SignedIn(onLogout: () -> Unit) {
    var tab by rememberSaveable { mutableStateOf(0) }
    // **وحسابُه يبقى مفتوحاً بعد دوران الشاشة** — من كتب اسمَه نصفاً ثمّ
    // أمال هاتفَه **لا يجد نفسَه في شاشةٍ أخرى.**
    var account by rememberSaveable { mutableStateOf(false) }
    /** **وشاشةُ تقييماته** — تُفتح من نجمة الشريط. */
    var rating by rememberSaveable { mutableStateOf(false) }
    /** **ومحفظتُه** — تُفتح من رقاقة الرصيد. */
    var wallet by rememberSaveable { mutableStateOf(false) }
    val home: HomeViewModel = viewModel()
    val orders: OrdersViewModel = viewModel()
    val accountVm: AccountViewModel = viewModel()
    val ratingVm: RatingViewModel = viewModel()
    val walletVm: WalletViewModel = viewModel()
    val context = LocalContext.current

    // ══════════════════════════════════════════════════════════════════
    // **طلب الإذن على مرحلتين — لأنّ النظام يرفض غير ذلك**
    // ══════════════════════════════════════════════════════════════════
    //
    // **أوّلا الموقع الدقيق** أثناء الاستعمال، **ثمّ يُطلب التوسيع إلى
    // «طوال الوقت»** من الإعدادات — وأندرويد ١١ فما فوق **لا يعرض
    // نافذة له أصلا.**
    //
    // **ومن طلبهما معا رُدّ طلبه كلّه** بلا أن يُعرض على صاحبه شيء.
    // ══════════════════════════════════════════════════════════════════
    // **الكاميرا — صورةٌ مصغّرة لا ملفّ كامل**
    // ══════════════════════════════════════════════════════════════════
    //
    // **و`TakePicturePreview` تعيد صورةً صغيرةً في الذاكرة** — لا تحتاج
    // مزوّد ملفّات ولا إذن تخزين، **وهي كلّ ما يلزم لإثبات باب.**
    val camera = rememberLauncherForActivityResult(
        ActivityResultContracts.TakePicturePreview(),
    ) { bitmap ->
        if (bitmap != null) {
            orders.sendProof(Proof.shrink(bitmap), LastPoint.value)
        }
    }

    val askCamera = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted -> if (granted) camera.launch(null) }

    val askBackground = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { home.recheckLocation() }

    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { granted ->
        home.recheckLocation()
        // **ثمّ يُطلب «طوال الوقت» كإذن** — فيعرض النظام صفحته وفيها
        // الخيار ظاهرا. **وبلاه يسكت الموقع بمجرّد أن تُطفأ الشاشة**،
        // فيبدو للمكتب واقفا وهو يسير.
        if (granted.values.any { it } && !LocationPermission.backgroundGranted(context)) {
            askBackground.launch(LocationPermission.BACKGROUND)
        }
    }

    // **ويُعاد الفحص عند كلّ عودة إلى الشاشة** — قد يكون غيّره من
    // الإعدادات، **فلا تبقى البطاقة تطلب ما أُعطي.**
    LifecycleResumeEffect(Unit) {
        home.recheckLocation()
        onPauseOrDispose { }
    }

    // ══════════════════════════════════════════════════════════════════
    // **وتبويب الرحلة يظهر حين تكون ثمّة رحلة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرار المالك ٢٠٢٦-٠٨-١٢: «قسم الرحلة يجب أن يكون مخفيّا أساسا،
    //  يظهر فقط عند بدء الرحلة».)
    //
    // **وتبويب فارغ ثلثَ اليوم يُقرأ عطبا** — يفتحه صاحبه فيجد «لا رحلة
    // الآن»، **ثمّ يكفّ عن فتحه** فلا يراه يوم تكون فيه رحلة.
    val hasTrip = orders.state.mine.isNotEmpty()
    /** **أهو في الدوام** — وعليه يظهر تبويبُ الطلبات. */
    val onShift = home.state.me?.onShift == true

    // **ومن انتهت رحلته يُعاد إلى الطلبات** — لا يبقى في تبويب اختفى.
    LaunchedEffect(hasTrip, onShift) {
        if (!hasTrip && tab == 0) tab = 1
        // **ومن انصرف وهو في الطلبات يُردّ إلى لوحته** — لا يبقى في
        // شاشةٍ لم يعد لها تبويب.
        if (!onShift && tab == 1) tab = 2
    }

    // **ومن قبِل طلبا فُتحت رحلته** — (قرار المالك ٢٠٢٦-٠٨-١٢).
    LaunchedEffect(orders.startTrip) {
        if (orders.startTrip) {
            tab = 0
            orders.tripOpened()
        }
    }

    Scaffold(
        // ══════════════════════════════════════════════════════════════
        // **ولا شريطَ علويًّا في الرحلة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «التوب بار كمان ألغه بالرحلة، ما
        //  يلزمه».)
        //
        // **ومن يقود لا يقرأ رصيده ولا تقييمه** — والخريطةُ تريد الشاشةَ
        // كلَّها. **وهي موجودةٌ في تبويبين آخرين** يفتحهما حين يقف.
        topBar = {
            if (tab != 0) TopBar(
                name = home.state.me?.fullName.orEmpty(),
                // **والمسار النسبيّ يصير عنوانا هنا** — المحرّك يرسل
                // `/media/...`، **وهو نفسه في المحلّيّ والإنتاج.**
                avatarUrl = Backend.media(home.state.me?.avatarUrl),
                balance = home.state.me?.balance ?: 0,
                rating = home.state.me?.rating ?: 0.0,
                ratingCount = home.state.me?.ratingCount ?: 0,
                unread = home.unread,
                // **ورقاقةُ المحفظة تفتح المحفظة** — (قرارُ المالك
                // ٢٠٢٦-٠٨-١٣: «نبني المحفظة بنفس الويب»). **وكانت
                // تنقله إلى اللوحة** حيث سطرُ رصيدٍ لا كشفُ حساب.
                onWallet = { wallet = true; account = false; rating = false },
                onNotifications = home::openInbox,
                onProfile = { account = true; rating = false; wallet = false },
                onRating = { rating = true; account = false; wallet = false },
            )
        },
        bottomBar = {
            // ══════════════════════════════════════════════════════════
            // **الرحلة أوّلا — وهي ما يفعله السائق**
            // ══════════════════════════════════════════════════════════
            //
            // (مواصفة المالك ٢٠٢٦-٠٨-١٢: «أوّل قسم يكون الخريطة نسمّيها
            //  الرحلة، والقسم الثاني الطلبات».)
            NavigationBar {
                if (hasTrip) {
                    NavigationBarItem(
                        // ══════════════════════════════════════════════
                        // **والتبويبُ يخرج من الحساب — كما في الويب**
                        // ══════════════════════════════════════════════
                        //
                        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «إذا كنتُ داخل حسابي
                        //  وضغطتُ الشاشة الرئيسيّة يجب أن يذهب إلى
                        //  الشاشة الرئيسيّة أو أيّ قسمٍ آخر… انظر كيف
                        //  هي على الويب، يجب أيضاً أن تكون كذلك على
                        //  التطبيق».)
                        //
                        // **وكان الحسابُ يغطّي فوق التبويبات**، فيضغط
                        // «الرئيسيّة» فيتبدّل التبويبُ تحته **ولا يتغيّر
                        // ما يراه** — فيظنّ الزرَّ معطّلا.
                        //
                        // **وشريطٌ يُضغط ولا يستجيب أسوأُ من شريطٍ
                        // مخفيّ**: المخفيُّ يقول «لا مخرجَ هنا»،
                        // **والصامتُ يقول «معطّل».**
                        selected = tab == 0 && !account && !rating && !wallet,
                        onClick = { account = false; rating = false; wallet = false; tab = 0; orders.refresh() },
                        icon = {
                            Icon(painterResource(R.drawable.ic_trip), contentDescription = null)
                        },
                        label = { Text(stringResource(R.string.nav_trip)) },
                    )
                }
                // ══════════════════════════════════════════════════════
                // **وتبويبُ الطلبات يختفي بانصرافه**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «إذا كان الدوام معطّلاً…
                //  الأفضل أن القسم يختفي ولا يظهر، مثل الرحلة».)
                //
                // **ومنصرفٌ لا يصله طلب** — فالشاشةُ فارغةٌ بيقين،
                // **وتبويبٌ يُفتح ليُرى فارغاً يُتعب.**
                //
                // **ولا يختفي لأنّ الطابور فارغٌ وهو في الدوام**:
                // **الطابورُ الفارغُ هو شاشةُ انتظاره** — ينظر إلى
                // المكان الذي سيجيء منه العمل. **ولو أُخفي لَسأل: هل
                // التطبيقُ يعمل أصلا؟**
                //
                // **والشريطُ لا يتبدّل تحت إبهامه** مع كلّ طلبٍ يجيء
                // ويذهب — **فيضغط ما لم يقصد.**
                if (onShift) NavigationBarItem(
                    selected = tab == 1 && !account && !rating && !wallet,
                    onClick = { account = false; rating = false; wallet = false; tab = 1; orders.refresh() },
                    icon = {
                        Icon(painterResource(R.drawable.ic_orders), contentDescription = null)
                    },
                    label = { Text(stringResource(R.string.nav_orders)) },
                )
                NavigationBarItem(
                    selected = tab == 2 && !account && !rating && !wallet,
                    onClick = { account = false; rating = false; wallet = false; tab = 2; home.refresh() },
                    icon = {
                        Icon(painterResource(R.drawable.ic_home), contentDescription = null)
                    },
                    label = { Text(stringResource(R.string.nav_home)) },
                )
            }
        },
    ) { padding ->
        // **والرحلةُ بلا حاشيةٍ عليا** — الخريطةُ تمتدّ تحت شريط النظام،
        // **وما فوقها يدفع نفسَه بنفسه.** وسائرُ الشاشات تُحاذي شريطَها.
        Box(
            Modifier.padding(
                top = if (tab == 0) 0.dp else padding.calculateTopPadding(),
                bottom = padding.calculateBottomPadding(),
            ),
        ) {
            // **وصندوق الإشعارات يغطّي** — يُقرأ ثمّ يُغلق.
            val notices = home.inbox
            if (notices != null) {
                InboxSheet(items = notices, onClose = home::closeInbox)
                return@Box
            }

            // ══════════════════════════════════════════════════════════
            // **وحسابُه يغطّي التبويبَ ولا يصير تبويبا**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بالطبع تفتح صفحةً كاملة».)
            //
            // **وتبويبٌ رابعٌ يزاحم ثلاثةً تُستعمل كلَّ دقيقة** — والحسابُ
            // يُفتح مرّةً في الشهر. **ويُدخَل من صورته حيث يتوقّعه**، لا
            // من شريطٍ سفليٍّ يتعلّمه.
            //
            // **والرجوعُ من زرّ النظام أيضاً** — إبهامٌ يعود بما تعوّد.
            if (account) {
                BackHandler { account = false }
                AccountScreen(
                    vm = accountVm,
                    onLoggedOut = onLogout,
                )
                return@Box
            }

            // **وتقييماتُه تغطّي كذلك** — تُفتح من نجمته وتُغلق برجوعه.
            if (rating) {
                BackHandler { rating = false }
                RatingScreen(vm = ratingVm)
                return@Box
            }

            // **ومحفظتُه من رقاقة رصيده.**
            if (wallet) {
                BackHandler { wallet = false }
                WalletScreen(vm = walletVm)
                return@Box
            }

            when {
                tab == 0 -> TripScreen(
                    state = orders.trip(LastPoint.value),
                    actions = TripActions(
                        step = orders::step,
                        capture = {
                            // **والإذن يُطلب عند الحاجة لا عند الدخول** —
                            // **كاميرا تُطلب في أوّل فتحة** تُرفض.
                            if (ContextCompat.checkSelfPermission(
                                    context,
                                    android.Manifest.permission.CAMERA,
                                ) == android.content.pm.PackageManager.PERMISSION_GRANTED
                            ) {
                                camera.launch(null)
                            } else {
                                askCamera.launch(android.Manifest.permission.CAMERA)
                            }
                        },
                        release = orders::releaseCurrent,
                        chat = {
                            if (orders.chat != null) orders.closeChat() else orders.openChat()
                        },
                        askAgree = orders::askAgree,
                        agree = orders::agree,
                        dismissAgree = orders::dismissAgree,
                        pickStop = orders::open,
                        takeOffer = orders::accept,
                        dismissOffer = orders::dismissOffer,
                        askFail = orders::askFail,
                        fail = orders::fail,
                        dismissFail = orders::dismissFail,
                        problem = { orders.reportProblem(it, LastPoint.value) },
                        emergency = { orders.emergency(LastPoint.value) },
                        dismissEmergency = orders::dismissEmergency,
                        navigate = { openMaps(context, orders.trip(LastPoint.value)) },
                        toOrders = { tab = 1 },
                    ),
                    chat = orders.chat,
                    chatActions = ChatActions(
                        send = orders::sendMessage,
                        close = orders::closeChat,
                    ),
                    chatUnread = orders.chatUnread,
                )

                tab == 2 -> HomeScreen(
                    state = home.state,
                    actions = HomeActions(
                        toggleShift = home::toggleShift,
                        refresh = home::refresh,
                        enableLocation = {
                            when {
                                !LocationPermission.granted(context) ->
                                    ask.launch(LocationPermission.FIRST_STEP)

                                !LocationPermission.backgroundGranted(context) ->
                                    askBackground.launch(LocationPermission.BACKGROUND)

                                // **وآخر ملجأ الإعدادات** — لمن رفض
                                // نهائيّا فلا يعرض النظام له نافذة بعدها.
                                else -> LocationPermission.openSettings(context)
                            }
                        },
                        logout = onLogout,
                    ),
                )

                else -> OrdersScreen(
                    state = orders.state,
                    actions = OrdersActions(
                        accept = orders::accept,
                        decline = orders::decline,
                        startTrip = { id ->
                            orders.open(id)
                            tab = 0
                        },
                        refresh = orders::refresh,
                    ),
                )
            }
        }
    }
}

/**
 * **تعذّر الاتّصال ومعه حساب محفوظ.**
 *
 * **ولا تُعرض شاشة الدخول هنا** — حسابه سليم، والشبكة هي الغائبة.
 * **ومن أراه شاشة دخول** جعله يظنّ أنّ حسابه ضاع.
 */
@Composable
private fun Offline(onRetry: () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.offline_title),
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.offline_text),
            color = InkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(20.dp))
        Button(onClick = onRetry) { Text(stringResource(R.string.home_retry)) }
    }
}

/**
 * **يسلّم الوجهة لتطبيق الخرائط في الجهاز.**
 *
 * **ولا تُبنى ملاحة داخل تطبيق توصيل** — السائق يعرف تطبيقه ويثق بصوته،
 * **وبناء ملاحة يعني خادم توجيه وصوتا وتحديث خرائط** لا طائل منه.
 *
 * **والوجهة هي وجهة اللحظة**: المتجر قبل الاستلام، والزبون بعده.
 */
private fun openMaps(context: android.content.Context, trip: TripState) {
    val target = if (trip.step >= com.rahalgo.driver.trip.TripStep.PICKED_UP) {
        trip.dropoff
    } else {
        trip.pickup ?: trip.dropoff
    } ?: return
    val uri = android.net.Uri.parse("geo:${target.latitude},${target.longitude}?q=${target.latitude},${target.longitude}")
    val intent = android.content.Intent(android.content.Intent.ACTION_VIEW, uri)
    // **ولو لم يكن في الجهاز تطبيق خرائط** — لا يسقط التطبيق.
    runCatching { context.startActivity(intent) }
}
