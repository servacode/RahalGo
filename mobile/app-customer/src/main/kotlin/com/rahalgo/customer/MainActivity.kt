package com.rahalgo.customer

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.width
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.rememberDrawerState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.foundation.layout.RowScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.LaunchedEffect
import androidx.core.view.WindowCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.Rahal
import com.rahalgo.design.RahalGoTheme
import com.rahalgo.ui.AppFrame
import com.rahalgo.ui.AskNotifyPermission
import com.rahalgo.ui.AuthGate
import com.rahalgo.ui.AuthViewModel
import com.rahalgo.ui.Crash
import com.rahalgo.ui.LastPoint
import com.rahalgo.ui.Drawer
import com.rahalgo.ui.Overlay
import com.rahalgo.ui.PointPicker
import com.rahalgo.ui.PagesViewModel
import com.rahalgo.ui.HelpRole
import com.rahalgo.ui.PlatformPages
import com.rahalgo.ui.ChatFab
import com.rahalgo.ui.ChatsScreen
import com.rahalgo.ui.ChatsViewModel
import com.rahalgo.ui.PlatformScreen
import com.rahalgo.ui.knowsKey
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.ui.Avatar
import com.rahalgo.ui.InboxSheet
import com.rahalgo.ui.ThemeState
import com.rahalgo.ui.TopBar
import com.rahalgo.ui.WalletScreen
import com.rahalgo.ui.WalletViewModel
import com.rahalgo.ui.AccountScreen
import com.rahalgo.ui.AccountViewModel
import com.rahalgo.ui.selectedAddress
import com.rahalgo.ui.DeliveryAddress
import com.rahalgo.ui.AddressHost
import com.rahalgo.ui.addressKindLabel
import com.rahalgo.map.PickPoint
import com.rahalgo.map.PickPointViewModel
import org.maplibre.android.geometry.LatLng
import com.rahalgo.customer.shop.ShopScreen
import com.rahalgo.customer.shop.ShopViewModel
import com.rahalgo.customer.orders.OrdersScreen
import com.rahalgo.customer.orders.RateDialog
import com.rahalgo.customer.orders.OrdersViewModel
import com.rahalgo.customer.orders.HistoryScreen
import com.rahalgo.customer.custom.CustomScreen
import com.rahalgo.customer.custom.CustomViewModel
import com.rahalgo.customer.mine.MineScreen
import com.rahalgo.customer.mine.MineViewModel
import com.rahalgo.customer.cart.CartScreen
import com.rahalgo.customer.cart.CartViewModel
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import com.rahalgo.ui.rememberOverlay
import com.rahalgo.ui.rememberTheme
import com.rahalgo.ui.RahalButton
import kotlinx.coroutines.launch
import com.rahalgo.ui.CountBadge

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إقلاع تطبيق الزبون — الخطوة صفر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «البناءُ يجب أن يكون كخطوات، ما نبني كلَّ
 *  شيءٍ دفعةً واحدة».)
 *
 * # وما فيها اليوم
 *
 * **يُقلع · يُنشئ حساباً · يدخل · يُظهر اسمَه · يخرج.** ولا شيءَ غيرَ
 * ذلك — **والواجهةُ والمتاجرُ والسلّةُ خطواتٌ تليها.**
 *
 * # ولا شاشةَ دخولٍ كُتبت هنا
 *
 * **كلُّها من `:ui`** — الشاشةُ ومنطقُها والاستعادةُ والتسجيل. **وما
 * كُتب هنا ثلاثةُ أشياء**: عنوانُ الشاشة، وأنّ التسجيلَ معروض، وما
 * يُرى بعد الدخول.
 *
 * **وهذا هو الاختبار**: لو لزم أن يُكتب هنا منطقُ دخولٍ لَكان الرفعُ
 * ناقصا.
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // **ومراقبُ الشبكة يُسجَّل مرّةً** — انظر `Net`: **مراقبٌ لكلّ
        // شاشةٍ يعني عشرةً يوقظهم النظامُ معاً.**
        com.rahalgo.ui.Net.install(this)
        // **وجالبُ الصور يُسجَّل بيدنا** — انظر `Images`: **R8 يحذف ما
        // يُكتشَف بـ`ServiceLoader`**، فتفشل كلُّ صورةٍ بعيدةٍ في نسخة
        // الإصدار وحدَها.
        com.rahalgo.ui.Images.install(this)
        // **وتقاريرُ الانهيار تبدأ قبل أوّل شاشة** — والسقوطُ في الإقلاع
        // أكثرُ ما يقع، **ومن بدأ التقاريرَ بعده لا يراه.**
        Crash.start(debug = BuildConfig.DEBUG)
        // **والنواةُ تُركَّب قبل أوّل شاشة** — `AuthViewModel` يصنعه
        // أندرويدُ لا نحن، **فيقرؤها من المُسجَّل.**
        Backend.of(this)
        // **والسلّةُ تُستعاد قبل أوّل شاشة** — (تصحيحُ المالك
        // ٢٠٢٦-٠٨-١٨: «يجب أن تحتفظ به حتّى بعد الخروج أو إعادة تشغيل
        // التطبيق»). **ومن نسي هذا النداءَ لا تُحفظ ولا يظهر خطأ.**
        com.rahalgo.customer.cart.Cart.install(this)
        // ══════════════════════════════════════════════════════════════
        // **ورمزُ الدعوة يُقرأ قبل أوّل شاشة**
        // ══════════════════════════════════════════════════════════════
        //
        // **من الرابط إن فُتح به** — و**من المتجر إن نُزّل منه**:
        // **من نزّله ثمّ سجّل بلا رمزٍ لا يُنسب لمن دعاه**، ولا يأخذ
        // أحدُهما مكافأة.
        Invited.fromLink(this, intent)
        Invited.fromStore(this)
        WindowCompat.getInsetsController(window, window.decorView)
            .isAppearanceLightStatusBars = true
        setContent { CustomerApp() }
    }

    /**
     * **ورابطٌ يصل والتطبيقُ مفتوح** — `singleTask` لا تُعيد إنشاءه،
     * **فبلا هذه يبقى `intent` القديمُ ويُقرأ رابطُ الأمس.**
     */
    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        Invited.fromLink(this, intent)
    }
}

@Composable
private fun CustomerApp() {
    // **والإطارُ من الوحدة** — السمةُ وشريطا النظام.
    AppFrame { theme, dark -> Signed(theme, dark) }
}

@Composable
private fun Signed(theme: ThemeState, dark: Boolean) {
    val vm: AuthViewModel = viewModel()
    // **ويُطفأ متى دخل** — وإلّا بقيت شاشةُ الدخول فوق حسابه.
    var asking by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(vm.user) { if (vm.user != null) asking = false }

    // ══════════════════════════════════════════════════════════════════
    // **ومن جاء برمزِ دعوةٍ يُفتح له التسجيلُ لا التصفّح**
    // ══════════════════════════════════════════════════════════════════
    //
    // **من ضغط رابطَ صديقه جاء ليُنشئ حسابا** — **ومن وقع على السوق
    // بحث عن «إنشاء حساب» بنفسه**، وأكثرُهم لا يبحث.
    //
    // **ولا يُفتح لمن له حسابٌ أصلا** — ولا بعد أن سجّل: **الرمزُ
    // يُمحى حين يُستعمل.**
    val ctx = LocalContext.current
    LaunchedEffect(vm.user, vm.restoring) {
        if (vm.restoring || vm.user != null) return@LaunchedEffect
        val code = Invited.code(ctx)
        if (code.isNotEmpty() && vm.signup == null) vm.openSignup(code)
    }
    LaunchedEffect(vm.user) { if (vm.user != null) Invited.clear(ctx) }

    // ══════════════════════════════════════════════════════════════════
    // **وإذنُ الإشعارات يُطلب بعد الدخول** — انظر `AskNotifyPermission`.
    // ══════════════════════════════════════════════════════════════════
    //
    // (تدقيقُ الجاهزيّة ٢٠٢٦-٠٨-١٩.)
    //
    // **ولا يُطلب من ضيف**: لا طلباتِ له تُتابَع، **ونافذةٌ تظهر قبل
    // أن يفهم التطبيقَ تُرفض** — والرفضُ لا يُعاد سؤالُه.
    AskNotifyPermission(enabled = vm.user != null)

    // ══════════════════════════════════════════════════════════════════
    // **والرجوعُ يعود خطوةً — لا يُخرج من التطبيق**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٤: «زرُّ الرجوع يخرج من التطبيق… المفروض
    //  يعود إلى التسوّق».)
    //
    // **وشاشاتُ الدخول تُبدَّل بالغلاف كلِّه لا تُفتح فوقه** —
    // **وأندرويد لا يعرف أنّ بينها ترتيبا**: يرى شاشةً واحدةً فيُغلق
    // التطبيق.
    //
    // **والدرجات**: التسجيلُ ← الدخولُ ← التسوّق ← خارج.
    //
    // **ولا يُلتقط الرجوعُ في التسوّق** — من أراد أن يخرج يخرج:
    // **تطبيقٌ لا يُغلق بزرّ الرجوع يُقرأ معلَّقا.**
    BackHandler(enabled = vm.signup != null) { vm.closeSignup() }
    BackHandler(enabled = vm.reset != null) { vm.closeReset() }
    BackHandler(enabled = asking && vm.signup == null && vm.reset == null) {
        asking = false
    }

    // ══════════════════════════════════════════════════════════════════
    // **والبوّابةُ من الوحدة** — وترتيبُ حالاتها واحدٌ في الثلاثة
    // ══════════════════════════════════════════════════════════════════
    //
    // **والزبونُ وحدَه له «ضيف»**: يتصفّح السوقَ قبل أن يدخل، **والدخولُ
    // عند أوّل ما يخصّه.**
    AuthGate(
        vm = vm,
        title = stringResource(R.string.login_title),
        signup = true,
        asking = asking,
        onSignedIn = {
            SignedIn(
                theme = theme,
                dark = dark,
                guest = false,
                onLogout = vm::logout,
                onAskLogin = { asking = true },
            )
        },
        guest = {
            SignedIn(
                theme = theme,
                dark = dark,
                guest = true,
                onLogout = vm::logout,
                onAskLogin = { asking = true },
            )
        },
    )
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يُرى بعد الدخول — القائمةُ وما تفتحه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والقائمةُ من الوحدة** (`ui.Drawer`) وبنودُ المنصّة معها — **ولم
 * يُكتب هنا سطرٌ من رسمها.**
 *
 * **و«ما يخصّني» فارغةٌ الآن بأمر المالك** (٢٠٢٦-٠٨-١٤) — فلا تظهر
 * مجموعتُها أصلاً: **عنوانُ مجموعةٍ بلا بنودٍ تحته يُقرأ عطبا.**
 */
@Composable
private fun SignedIn(
    theme: ThemeState,
    dark: Boolean,
    /**
     * **يتصفّح ولم يدخل بعد.**
     *
     * **ويرى السوقَ وصفحاتِ المنصّة** — **وما يخصّه يطلب حسابا**: لا
     * طلباتٍ لمن لا حسابَ له، ولا مفضّلةَ تُحفظ.
     */
    guest: Boolean,
    onLogout: () -> Unit,
    onAskLogin: () -> Unit,
) {
    val pagesVm: PagesViewModel = viewModel()
    val shell: ShellViewModel = viewModel()
    // **ومن دخل يبدأ غلافُه عملَه** — الرصيدُ والشارةُ والوصلةُ الحيّة.
    LaunchedEffect(guest) { if (!guest) shell.wake() }
    val walletVm: WalletViewModel = viewModel()
    val accountVm: AccountViewModel = viewModel()
    val pickVm: PickPointViewModel = viewModel()
    val shopVm: ShopViewModel = viewModel()
    val ordersVm: OrdersViewModel = viewModel()
    val customVm: CustomViewModel = viewModel()
    val mineVm: MineViewModel = viewModel()
    val chatsVm: ChatsViewModel = viewModel()
    // **وحديثُ الطلب الجاري** — انظر `LiveChatViewModel`.
    val liveChat: com.rahalgo.customer.chat.LiveChatViewModel = viewModel()
    val cartVm: CartViewModel = viewModel()

    // ══════════════════════════════════════════════════════════════════
    // **والخريطةُ تُعطى من هنا — مرّةً لثلاث شاشات**
    // ══════════════════════════════════════════════════════════════════
    //
    // **`:ui` لا تعرف الخريطة** — **ولو عرفتها لَحملها كلُّ تطبيقٍ
    // معه**، وفيهم من لا خريطةَ له.
    //
    // **وثلاثُ شاشاتٍ تسأل عن موضع**: الحسابُ والسلّةُ والطلبُ الخاصّ.
    // **وثلاثُ نسخٍ من هذا اللمبدا تفترق يومَ تُزاد رايةٌ لإحداها.**
    // **والسياقُ يُقرأ هنا لا داخلَ اللمبدا** — `context` في هذه الدالّة
    // يُعرَّف بعدَه، **و`context(...)` في كوتلن كلمةٌ محجوزةٌ للمُعامِلات
    // السياقيّة** فيُقرأ نداءَ دالّةٍ لا متغيّرا.
    val ctx = LocalContext.current
    val mapPicker: PointPicker = { onPick, onCancel ->
        PickPoint(
            start = LastPoint.value?.let { LatLng(it.lat, it.lng) },
            vm = pickVm,
            onPick = { at, name -> onPick(at.latitude, at.longitude, name) },
            onCancel = onCancel,
            // **وأيقونةُ «موقعي» تنادي جهازَ التموضع من هنا** — وحدةُ
            // الخرائط لا تعرفه. (طلبُ المالك ٢٠٢٦-٠٨-١٨.)
            onLocate = { Here.refresh(ctx) },
        )
    }

    // **والسلّةُ تُفتح فوق التبويب** — ويُرجع منها إليه.
    val drawer = rememberDrawerState(DrawerValue.Closed)
    val scope = rememberCoroutineScope()
    val overlay = rememberOverlay { key -> knowsKey(CUSTOMER_ITEMS, key) }
    val context = LocalContext.current
    // **والتبويبُ يبقى بعد دوران الجهاز** — من كان في طلباته لا يُردّ
    // إلى التسوّق لأنّه أمال هاتفَه.
    var tab by rememberSaveable { mutableStateOf(Tab.Shop) }
    // **ومن خرج من حسابه وهو في «طلباتي» يُردّ إلى التسوّق** — **وإلّا
    // بقي في تبويبٍ لا زرَّ له في الشريط.**
    LaunchedEffect(guest) {
        if (guest && tab != Tab.Shop && tab != Tab.Custom) tab = Tab.Shop
    }

    // ══════════════════════════════════════════════════════════════════
    // **وإذنُ الموقع يُطلب عند فتح «حسابي» — لا عند الإقلاع**
    // ══════════════════════════════════════════════════════════════════
    //
    // **إذنٌ يُطلب في أوّل شاشةٍ بلا سببٍ ظاهرٍ يُرفض** — ومن رفضه
    // مرّتين أُغلق البابُ في أندرويد ولا يُفتح إلّا من الإعدادات.
    //
    // **وهنا سببُه أمام عينه**: يحفظ عنواناً فيريد نقطته.
    val askHere = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { ok -> if (ok) Here.refresh(context) }

    LaunchedEffect(tab, guest) {
        // **ولا يُسأل ضيفٌ عن موقعه** — قِيس على المحاكي (٢٠٢٦-٠٨-١٤):
        // **طُلب الإذنُ ممّن لا يرى الشاشةَ أصلا.**
        //
        // **وإذنٌ يُطلب بلا سببٍ يراه يُرفض** — ومن رفضه مرّتين أُغلق
        // البابُ في أندرويد ولا يُفتح إلّا من الإعدادات.
        if (guest || tab != Tab.Account) return@LaunchedEffect
        if (Here.granted(context)) Here.refresh(context)
        else askHere.launch(android.Manifest.permission.ACCESS_FINE_LOCATION)
    }

    // **والرجوعُ من القائمة يغلقها — لا يُخرج من التطبيق.**
    // (كُشف بالقياس في تطبيق السائق ٢٠٢٦-٠٨-١٤، **فلا يُعاد هنا.**)
    BackHandler(enabled = drawer.isOpen) { scope.launch { drawer.close() } }
    BackHandler(enabled = tab == Tab.Cart && !drawer.isOpen) { tab = Tab.Shop }


    // ══════════════════════════════════════════════════════════════════
    // **ومضيفُ العنوان يُركَّب مرّةً — والشاشاتُ تناديه ولا تبنيه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «المفروض تصير طريقةُ الخريطة مركزيّة مشان
    //  ما نظلّ نبنيها بكلّ هالصفحات… مركزيّ ونستدعيها».)
    //
    // **وأربعُ شاشاتٍ كانت تسأل السؤالَ نفسَه** — الشريطُ و«حسابي»
    // والطلبُ الخاصُّ والسلّة، **وكلٌّ تحمل رايتَها وتفتح لوحتَها.**
    //
    // **وقبل الدرج والهيكل** — صفحةُ الخريطة تملأ الشاشة، **وشريطٌ
    // علويٌّ فوقها يجعلها جزءاً من شاشةٍ أخرى.**
    if (AddressHost(accountVm, mapPicker)) return

    ModalNavigationDrawer(
        drawerState = drawer,
        drawerContent = {
            // **والدرجُ رفيعٌ لا عريض** — عرضُ `ModalDrawerSheet`
            // الافتراضيُّ ٣٦٠، وهو نصفُ الشاشة.
            ModalDrawerSheet(Modifier.width(200.dp)) {
                Drawer(
                    // **والضيفُ يرى المنصّةَ والقانونيّةَ وحدَهما** —
                    // **وبنودٌ تُعرض لتقول «تحتاج حساباً» تُطيل القائمةَ
                    // ولا تُفيد.**
                    items = (if (guest) emptyList() else CUSTOMER_ITEMS) +
                        PlatformPages.items,
                    onPick = { item ->
                        overlay.show(Overlay.Menu(item.key))
                        scope.launch { drawer.close() }
                    },
                    onLogout = if (guest) onAskLogin else onLogout,
                    dark = dark,
                    onTheme = { theme.toggle(dark) },
                    logoutLabel = if (guest) R.string.guest_enter else null,
                )
            }
        },
    ) {
        val over = overlay.current


        Scaffold(
            topBar = {
                // **والشريطُ العلويُّ من الوحدة** — الجرسُ والسمةُ
                // والمحفظة. **ولا رقاقةَ تقييمٍ للزبون**: هو لا يُحاسَب
                // على رقمٍ، **ورقاقةٌ لا معنى لها ازدحامٌ لا خبر.**
                TopBar(
                    balance = shell.balance,
                    unread = shell.unread,
                    onMenu = { scope.launch { drawer.open() } },
                    // **وضغطةٌ ثانيةٌ تُغلقها** — (قرارُ المالك
                    // ٢٠٢٦-٠٨-١٣، وكان في السائق وحدَه).
                    onNotifications = {
                        overlay.toggle(Overlay.Inbox)
                        if (overlay.current == Overlay.Inbox) shell.openInbox()
                    },
                    onWallet = { overlay.show(Overlay.Wallet) },
                    guest = guest,
                    // **ولا عنوانَ لضيف** — (قرارُ المالك ٢٠٢٦-٠٨-١٤:
                    // «لا يمكن أن يرى كلَّ المعلومات وهو لم يسجّل
                    // دخولاً بعد»). **ومن لا حسابَ له لا عناوينَ له**،
                    // وزرٌّ يفتح قائمةً فارغةً أبداً يُقرأ عطبا.
                    // ══════════════════════════════════════════════
                    // **ولا يُعرض إلّا حيث يُطلب**
                    // ══════════════════════════════════════════════
                    //
                    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «زرُّ التوصيل إلى يظهر
                    //  فقط في قسم تسوّق وطلب خاصّ، باقي الأقسام لا
                    //  ضرورةَ أن يكون ظاهراً أصلاً».)
                    //
                    // **والعنوانُ يخصّ ما سيُطلب** — **ومن يقرأ طلباتِه
                    // أو يعدّل حسابَه لا يسأل «إلى أين أُوصّل».**
                    //
                    // **وعنصرٌ يُعرض حيث لا يُستعمل يُضعف معناه حيث
                    // يُستعمل** — تعتاد العينُ تخطّيه.
                    onAddress = if (guest || (tab != Tab.Shop && tab != Tab.Custom)) {
                        null
                    } else {
                        { DeliveryAddress.open() }
                    },
                    // ══════════════════════════════════════════════
                    // **واسمُ العنوان لا سطرُه**
                    // ══════════════════════════════════════════════
                    //
                    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «بدل كتابة العنوان
                    //  المفروض نكتب البيت أو العمل حسب اسم العنوان،
                    //  هيك يصير احترافيّاً أكثر».)
                    //
                    // **وسطرٌ فيه منطقةٌ وشارعٌ وطابقٌ يُقصّ بنقاطٍ في
                    // شريطٍ ضيّق** — فيُقرأ نصفُه ولا يُعرف أيُّ عنوانٍ
                    // هو. **والاسمُ يُقرأ بنظرة.**
                    addressLabel = accountVm.state.addresses
                        .let { selectedAddress(it) }
                        ?.let { stringResource(addressKindLabel(it.kind)) }
                        .orEmpty(),
                )
            },
            bottomBar = {
                // ══════════════════════════════════════════════════════
                // **وحسابي في يسار الشريط السفليّ**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «ومن الأسفل حسابي على
                //  يسار».)
                //
                // **وآخرُ البنود هو الأيسر** — الشاشةُ من اليمين إلى
                // اليسار، **ومن رتّبها بعينه رتّبها معكوسةً في
                // العربيّة.**
                //
                // **وقاعدةٌ تفصل الشريطين**: الأعلى أخبارٌ تُقرأ
                // بنظرةٍ — رصيدٌ وجرس. **والأسفلُ أماكنُ يُنتقل إليها.**
                NavigationBar {
                    // **وتسوّق أوّلا فهي في اليمين** — الشاشةُ من اليمين
                    // إلى اليسار، **وأوّلُ البنود أيمنُها.** ومن رتّبها
                    // بعينه رتّبها معكوسةً في العربيّة.
                    //
                    // **وهي وجهةُ من فتح التطبيق** — لا سجلٌّ ولا حساب:
                    // **من فتحه جاء ليطلب.**
                    Tab(
                        selected = tab == Tab.Shop && over == Overlay.None,
                        // ══════════════════════════════════════════
                        // **والتبويبُ يُنزل رايةَ السلّة أيضا**
                        // ══════════════════════════════════════════
                        //
                        // (شكوى المالك ٢٠٢٦-٠٨-١٨: «إذا كنتُ في سلّتي
                        //  وضغطتُ على أيقونة قسمٍ آخر لا ينتقل إليها،
                        //  يبقى في قسم سلّتي».)
                        //
                        // **والسلّةُ طبقةٌ تغطّي التبويب** — فيُبدَّل
                        // التبويبُ تحتها ولا يُرى. **وضغطةٌ لا يقع لها
                        // أثرٌ ظاهرٌ تُعاد وتُعاد** حتّى يُظنّ أنّ
                        // الزرَّ معطوب.
                        onClick = { tab = Tab.Shop; overlay.clear() },
                        icon = R.drawable.ic_store,
                        label = R.string.nav_shop,
                    )
                    // ══════════════════════════════════════════════
                    // **والسلّةُ بعد التسوّق مباشرةً**
                    // ══════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٨: «السلّةُ خلّيها أيقونةً
                    //  بالصفّ التحت، ألغِها من مكانها، لأنّ بعض
                    //  المنتجات راح تبطّل تبيّن».)
                    //
                    // **وكانت طافيةً فوق الشبكة** — فتغطّي آخرَ صفٍّ من
                    // الأصناف، **ومن نزل إلى آخرها لم يرَ ما تحتها.**
                    //
                    // **وتجاور التسوّقَ لأنّهما فعلٌ واحد**: يختار ثمّ
                    // يراجع. **وبينهما «طلباتي» يفصل ما يُبنى عمّا
                    // أُرسل.**
                    //
                    // **وتُعرض للضيف** — انظر `needsAccount`: من أضاف
                    // ولم يجد أين يرى يقرأ التطبيقَ معطوبا.
                    NavigationBarItem(
                        selected = tab == Tab.Cart && over == Overlay.None,
                        onClick = { tab = Tab.Cart; overlay.clear() },
                        icon = {
                            // **والعددُ يطفو على الأيقونة** — `CountBadge`
                            // نفسُها التي في الجرس، **ونسختان تفترقان
                            // يومَ يتبدّل شكلُ إحداهما.**
                            Box {
                                Icon(
                                    painter = painterResource(
                                        com.rahalgo.ui.R.drawable.ic_cart,
                                    ),
                                    contentDescription = null,
                                )
                                CountBadge(
                                    count = com.rahalgo.customer.cart.Cart.count,
                                    color = Rahal.colors.accent,
                                    size = 16.dp,
                                    modifier = Modifier.align(Alignment.TopEnd),
                                )
                            }
                        },
                        label = { Text(stringResource(R.string.cart_title)) },
                    )

                    // **وطلباتي بينهما** — يُفتح كثيراً بعد الطلب
                    // (أين وصل؟)، **وأقلَّ من التسوّق وأكثرَ من الحساب.**
                    //
                    // **ولا يُعرض لضيف**: لا طلباتِ لمن لا حسابَ له،
                    // **وتبويبٌ يُضغط فيقول «تحتاج حساباً» بابٌ يُفتح
                    // ليُغلق.**
                    if (!guest) {
                        Tab(
                            selected = tab == Tab.Orders && over == Overlay.None,
                            onClick = { tab = Tab.Orders; overlay.clear() },
                            icon = R.drawable.ic_orders,
                            label = R.string.nav_orders,
                        )
                    }
                    // ══════════════════════════════════════════════
                    // **والطلبُ الخاصّ تبويبٌ لا بندٌ في قائمة**
                    // ══════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «نضيف طلب خاصّ أيضاً
                    //  لأنّه مميّزٌ بموقعنا، بعد طلباتي».)
                    //
                    // **وهو ما يفرّق المنصّةَ عن غيرها**: من أراد شيئاً
                    // لا متجرَ له في التطبيق **يكتبه بلفظه** فيشتريه
                    // السائق. **وبابٌ يميّزك مدفونٌ في قائمةٍ جانبيّةٍ
                    // لا يُفتح** — ومن لم يفتحه لم يعرف أنّه موجود.
                    Tab(
                        selected = tab == Tab.Custom && over == Overlay.None,
                        onClick = { tab = Tab.Custom; overlay.clear() },
                        icon = R.drawable.ic_custom,
                        label = R.string.nav_custom,
                    )
                    // **وحسابي آخرا فهو في اليسار** — بأمر المالك
                    // (٢٠٢٦-٠٨-١٤).
                    //
                    // ══════════════════════════════════════════════════
                    // **وصورتُه إن رفعها — وأيقونةٌ إن لم يرفع**
                    // ══════════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «حساب الزبون أضف
                    //  أيقونة».)
                    //
                    // **وأكثرُ الزبائن لا يرفع صورة** — **فكانت
                    // الحروفُ الأولى من اسمه في دائرةٍ ملوّنة**: شكلٌ
                    // رابعٌ في شريطٍ ثلاثةُ بنودِه رسومٌ خطّيّة،
                    // **يُقرأ زينةً لا وجهةً يُنتقل إليها.**
                    //
                    // **ومن رفع صورتَه يراها** — هي أدلُّ عليه من أيّ
                    // رسم. (وقاعدةٌ تفصل الشريطين: الأعلى أخبارٌ تُقرأ
                    // بنظرةٍ، **والأسفلُ أماكنُ يُنتقل إليها.**)
                    //
                    // **ولا حسابَ لضيف** — **وصورةُ حسابٍ فارغةٌ لمن لم
                    // يدخل تسأل «حسابُ من؟».**
                    if (!guest) NavigationBarItem(
                        selected = tab == Tab.Account && over == Overlay.None,
                        onClick = { tab = Tab.Account; overlay.clear() },
                        icon = {
                            val photo = Backend.of(context).media(shell.me?.avatarThumbUrl)
                            if (photo.isNullOrEmpty()) {
                                Icon(
                                    painter = painterResource(com.rahalgo.ui.R.drawable.ic_user),
                                    contentDescription = null,
                                )
                            } else {
                                Avatar(
                                    url = photo,
                                    name = shell.me?.fullName.orEmpty(),
                                    // **وأكبرُ من أيقونة** — أربعةٌ
                                    // وعشرون مقاسُ رسمٍ خطّيّ،
                                    // **والصورةُ دائرةٌ فيها وجه**
                                    // فتُقرأ نقطةً لا وجها.
                                    size = 30,
                                )
                            }
                        },
                        label = { Text(stringResource(R.string.nav_profile)) },
                    )
                }
            },
        ) { padding ->
            Box(
                Modifier
                    .fillMaxSize()
                    .padding(
                        top = padding.calculateTopPadding(),
                        bottom = padding.calculateBottomPadding(),
                    ),
                contentAlignment = Alignment.Center,
            ) {
                if (over != Overlay.None) BackHandler { overlay.clear() }

                // ══════════════════════════════════════════════════════
                // **قرصُ الحديث — فوق كلّ شاشةٍ لا في شاشةِ الطلبات**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٩: «زرُّ الدردشة يجب أن يكون
                //  عائماً فوق كلّ الصفحات… مو معقول إلّا يفوت على
                //  الطلبات مشان يشوف الدردشة».)
                //
                // **ورُكّب أوّلاً في شاشة الطلبات فكان عيبَه نفسَه**:
                // من ينتظر ردَّ سائقه يتصفّح السوقَ أو يقرأ حسابَه،
                // **والقرصُ الذي يظهر حيث لا تحتاجه لا يُغني.**
                //
                // **وهنا يُركَّب مرّةً** — كالرسالة الطافية وشريط
                // الشبكة والافتتاح: **وثلاثةُ تركيباتٍ تُنسى في واحد.**
                when {
                    // ══════════════════════════════════════════════════
                    // **وما يخصّه يطلب حساباً — قبل النداء لا بعده**
                    // ══════════════════════════════════════════════════
                    //
                    // **وبابٌ يُفتح لضيفٍ ثمّ يردّه المحرّكُ بـ٤٠١**
                    // يُقرأ عطباً في التطبيق: **الشاشةُ تدور ثمّ تقول
                    // «الجلسة منتهية» لمن لم يدخل قطّ.**
                    //
                    // **والسوقُ وصفحاتُ المنصّة ليست منه** — عامّةٌ في
                    // المحرّك، **فتُفتح كما تُفتح في الويب.**
                    guest && needsAccount(tab, over) ->
                        NeedAccount(onAskLogin)

                    over is Overlay.Menu && PlatformPages.has(over.key) ->
                        // **وتعليماتُ الزبون لا تعليماتُ السائق** —
                        // سلّةٌ وعنوانٌ مقابلَ ورديّةٍ وصندوق.
                        PlatformScreen(vm = pagesVm, key = over.key, role = HelpRole.Customer)

                    // **وسجلُّ الطلبات يقرأ نموذجَ الطلبات نفسَه** —
                    // بابٌ واحدٌ بشرطٍ مختلف، **ونموذجان يعنيان نسختين
                    // من ترجمة الحال ومن حسبة المهلة.**
                    over is Overlay.Menu && over.key == CustomerItems.HISTORY ->
                        HistoryScreen(ordersVm)

                    over is Overlay.Menu && over.key == CustomerItems.CHATS -> {
                        androidx.compose.runtime.LaunchedEffect(Unit) { chatsVm.load() }
                        ChatsScreen(chatsVm)
                    }

                    over is Overlay.Menu -> MineScreen(mineVm, over.key)

                    // ══════════════════════════════════════════════
                    // **والمحفظةُ شاشةُ السائق نفسُها — بلا سحب**
                    // ══════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «ابنِ المحفظة بشكلٍ
                    //  صحيحٍ وكامل».)
                    //
                    // **والزبونُ لا يسحب** — رصيدُه يُنفَق لا يُقبَض،
                    // **وزرُّ سحبٍ لمن لا يستطيع يُملأ نموذجُه ثمّ
                    // يُردّ.**
                    //
                    // **وما سواه واحد**: الرصيدُ والحركاتُ بأسمائها
                    // والكشفُ المطبوع.
                    over == Overlay.Wallet -> WalletScreen(
                        vm = walletVm,
                        payouts = false,
                    )

                    // **والصندوقُ يُقرأ ثمّ يُغلق** — وفارغٌ حتّى يصل،
                    // **فلا تُعرض قائمةٌ فارغةٌ على أنّها «لا
                    // إشعارات».**
                    over == Overlay.Inbox -> InboxSheet(
                        items = shell.inbox.orEmpty(),
                        onMarkAll = shell::markAllRead,
                    )

                    // ══════════════════════════════════════════════
                    // **وحسابي شاشةُ السائق نفسُها**
                    // ══════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «الآن صفحة حسابي».)
                    //
                    // **الاسمُ والصورةُ وواتساب والرقمُ وكلمةُ المرور
                    // والعناوينُ وحذفُ الحساب** — كلُّها واحدةٌ عند
                    // الدورين، **ولا بندَ فيها يخصّ سائقا.**
                    // **والسوقُ يُتصفَّح بلا حساب** — من فتح التطبيقَ
                    // جاء يرى بضاعة.
                    // **والسلّةُ تغطّي السوق** — ومن أرسل طلبَه
                    // انتقل إلى «طلباتي» ليتابعه: **شاشةُ نجاحٍ تُغلق
                    // ثمّ يُسأل «وأين طلبي؟».**
                    tab == Tab.Cart -> CartScreen(
                        cartVm,
                        address = selectedAddress(accountVm.state.addresses),
                    ) {
                        tab = Tab.Orders
                        ordersVm.load()
                    }

                    tab == Tab.Shop -> ShopScreen(
                        vm = shopVm,
                        // **والإعجابُ يحتاج حساباً** — والضيفُ يُساق
                        // إلى الدخول لا يُردّ بصمت: **قلبٌ يُضغط فلا
                        // يقع شيءٌ يُقرأ عطبا.**
                        onLike = { item ->
                            if (guest) onAskLogin() else mineVm.toggleFavorite(item.id)
                        },
                        liked = mineVm.liked,
                    )

                    // ══════════════════════════════════════════════
                    // **ولوحةُ العنوان تغطّي ما تحتها**
                    // ══════════════════════════════════════════════
                    //
                    // **وقبل التبويبات في الترتيب** — ولو جاءت بعدها
                    // لَغطّاها التبويبُ فلا تُرى أبدا.
                    tab == Tab.Orders -> OrdersScreen(ordersVm)

                    // **ونجاحُ الطلب الخاصّ ينقله إلى «طلباتي»** —
                    // (شكوى المالك ٢٠٢٦-٠٨-١٨): **نموذجٌ يبقى مملوءاً
                    // بعد الإرسال يُقرأ «لم يُرسَل»**، فيُضغط ثانيةً
                    // وثالثةً فيخرج ثلاثةُ سائقين إلى بابٍ واحد.
                    tab == Tab.Custom -> CustomScreen(
                        customVm,
                        // **والعنوانُ من حسابه** — (طلبُ المالك
                        // ٢٠٢٦-٠٨-١٨)، **وبابُ الاختيار هو بابُ الشريط
                        // نفسُه**: لوحةٌ واحدةٌ لا اثنتان تفترقان.
                        address = selectedAddress(accountVm.state.addresses),
                        onOpenAddresses = { DeliveryAddress.open() },
                        onSent = { tab = Tab.Orders },
                    )

                    tab == Tab.Account -> AccountScreen(
                        vm = accountVm,
                        onLoggedOut = onLogout,
                        // **والزبونُ ليس عاملا** — لا دوامَ له، **وسببُ
                        // توثيقِ واتساب عنده أن يُحفظ حسابُه وتصله
                        // أخبارُ طلبه.**
                        worker = false,
                        // **ولا إشعاراتِ بعد** — نقطةُ Firebase تُضاف
                        // في خطوتها، **وقسمُ فحصٍ لميزةٍ لم تُبنَ يُقرأ
                        // عطبا.**
                        push = false,
                        picker = mapPicker,
                        // **وزرُّ «حسابي» يفتح الصفحةَ نفسَها** — (طلبُ
                        // المالك ٢٠٢٦-٠٨-١٨: «يفتح نفس الموقع بنفس
                        // الطريقة، والتعديل أيضا»).
                        onEditAddress = { a -> DeliveryAddress.edit(a?.id) },
                    )

                    // **ولا حالَ رابعة** — التبويباتُ أربعةٌ كلُّها
                    // مبنيّة، **وفرعٌ يبقى «ينتظر بناءه» بعد أن بُني
                    // يُخفي عطباً في التوجيه.**
                    else -> Unit
                }

                // ══════════════════════════════════════════════════════
                // **ورأيُه يُطلب حين يصل طلبُه — لا حين يبحث عن الزرّ**
                // ══════════════════════════════════════════════════════
                //
                // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «لا يظهر التقييمُ بشكلٍ
                //  تلقائيّ بعد استلام الطلب، وهيك المفروض».)
                //
                // **وموضعُها الغلافُ لا شاشةُ الطلبات**: من كان في
                // السوق حين وصل طلبُه **يُسأل وهو فيه** — ومن لا يفتح
                // «طلباتي» لا يُسأل أبدا.
                //
                // **ولا تُعرض لضيف** — ولا وهو في السلّة يدفع.
                if (!guest && tab != Tab.Cart) {
                    ordersVm.askRate?.let { o ->
                        RateDialog(
                            hasDriver = !o.driverName.isNullOrEmpty(),
                            onConfirm = { stars, driverStars, note ->
                                ordersVm.rate(o.id, stars, driverStars, note)
                                ordersVm.skipRate()
                            },
                            onDismiss = ordersVm::skipRate,
                        )
                    }
                }
                // **ويُرسَم بعد المحتوى** — وفي الصندوق يعلو الأخير.
                if (!guest) {
                    // **ويُقرأ عند الدخول وعند كلّ إنعاش** — ولا ينتظر
                    // زيارةَ شاشةٍ ليعرف أنّ للزبون سائقاً.
                    androidx.compose.runtime.LaunchedEffect(Unit) {
                        com.rahalgo.ui.Refresh.tick.collect { liveChat.load() }
                    }
                    liveChat.orderId?.let { id ->
                        ChatFab { liveChat.open = true }
                        if (liveChat.open) {
                            val cvm: com.rahalgo.ui.OrderChatViewModel = viewModel()
                            com.rahalgo.ui.OrderChatSheet(vm = cvm, orderId = id) {
                                liveChat.open = false
                            }
                        }
                    }
                }
            }
        }
    }
}

/**
 * **تبويباتُ الزبون** — بترتيبها في الشريط: يمينٌ إلى يسار.
 *
 * **والسلّةُ منها منذ ٢٠٢٦-٠٨-١٨** — (قرارُ المالك: «السلّةُ خلّيها
 * أيقونةً بالصفّ التحت، ألغِها من مكانها، لأنّ بعض المنتجات راح تبطّل
 * تبيّن»).
 *
 * **وكانت أيقونةً عائمةً** (قرارُه ٢٠٢٦-٠٨-١٥) — **وطافيةٌ فوق شبكةٍ
 * تغطّي آخرَ صفٍّ منها**، ومن نزل إلى آخر الأصناف لم يرَ ما تحتها.
 */
private enum class Tab { Shop, Cart, Orders, Custom, Account }

/** **بندٌ برسمٍ واسم** — وثلاثةُ نسخٍ منه في شريطٍ واحدٍ حشوٌ يُنسخ. */
@Composable
private fun RowScope.Tab(selected: Boolean, onClick: () -> Unit, icon: Int, label: Int) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = { Icon(painterResource(icon), contentDescription = null) },
        label = { Text(stringResource(label)) },
    )
}



/**
 * **أيلزم هذا الموضعَ حساب؟**
 *
 * **والقاعدةُ من المحرّك لا من الذوق**: ما كان تحت `/my` أو `/me` يلزمه
 * توكن، **وما كان تحت `/public` لا.**
 */
private fun needsAccount(tab: Tab, over: Overlay): Boolean = when {
    // ══════════════════════════════════════════════════════════════════
    // **والسلّةُ تُفتح بلا حساب**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٨: تنزل إلى الشريط السفليّ.)
    //
    // **ومن أضاف أصنافاً ثمّ لم يجد أين يراها يقرأ التطبيقَ معطوبا** —
    // وقد زالت الأيقونةُ العائمةُ التي كانت بابَه إليها.
    //
    // **والتسعيرةُ عامّةٌ في المحرّك** (`/public/quote`) — فيرى مجموعَه
    // وأجرتَه. **والحسابُ يُطلب عند الإرسال وحدَه**، وهناك موضعُه:
    // **يبني سلّتَه ثمّ يسجّل ليرسلها**، لا يُردّ على الباب.
    tab == Tab.Cart -> false
    // **والعروضُ عامّةٌ كالسوق** — `public/offers` بلا توكن، **وحجبُها
    // عن ضيفٍ يحجب أقوى ما يجذبه**: (قِيس ٢٠٢٦-٠٨-١٤).
    over is Overlay.Menu ->
        !PlatformPages.has(over.key) && over.key != CustomerItems.OFFERS
    over == Overlay.Wallet || over == Overlay.Inbox || over == Overlay.Account -> true
    else -> tab != Tab.Shop
}

/**
 * **ما يُعرض للضيف حيث يلزم حساب.**
 *
 * **ويقول لماذا** — لا «سجّل الدخول» وحدَها: **من عرف السببَ سجّل، ومن
 * قُرع بابُه بلا سببٍ خرج.**
 */
@Composable
private fun NeedAccount(onAskLogin: () -> Unit) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
        modifier = Modifier.padding(28.dp),
    ) {
        Text(
            text = stringResource(R.string.guest_title),
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = stringResource(R.string.guest_hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        RahalButton(onClick = onAskLogin) {
            Text(stringResource(R.string.guest_enter))
        }
    }
}
