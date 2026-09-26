package com.rahalgo.merchant

import androidx.compose.runtime.LaunchedEffect
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import com.rahalgo.ui.RahalButton
import com.rahalgo.design.Rahal
import com.rahalgo.ui.ShellViewModel
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.core.view.WindowCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.remember
import com.rahalgo.map.Here
import com.rahalgo.ui.Accuracy
import com.rahalgo.ui.Locating
import com.rahalgo.map.PickPoint
import com.rahalgo.map.PickPointViewModel
import com.rahalgo.merchant.drawer.MyReportsScreen
import com.rahalgo.merchant.drawer.MyReportsViewModel
import com.rahalgo.merchant.drawer.SalesScreen
import com.rahalgo.merchant.drawer.SalesViewModel
import com.rahalgo.merchant.drawer.WarningsScreen
import com.rahalgo.merchant.drawer.WarningsViewModel
import com.rahalgo.merchant.menu.MenuScreen
import com.rahalgo.merchant.menu.MenuViewModel
import com.rahalgo.merchant.orders.HistoryScreen
import com.rahalgo.merchant.orders.HistoryViewModel
import com.rahalgo.merchant.orders.OrdersScreen
import com.rahalgo.merchant.orders.OrdersViewModel
import com.rahalgo.merchant.store.StoreScreen
import com.rahalgo.merchant.store.StoreViewModel
import com.rahalgo.ui.AccountScreen
import com.rahalgo.ui.AccountViewModel
import com.rahalgo.ui.AppFrame
import com.rahalgo.ui.AuthGate
import com.rahalgo.ui.AuthViewModel
import com.rahalgo.ui.Avatar
import com.rahalgo.ui.Crash
import com.rahalgo.ui.Drawer
import com.rahalgo.ui.HelpRole
import com.rahalgo.ui.InboxSheet
import com.rahalgo.ui.Overlay
import com.rahalgo.ui.PagesViewModel
import com.rahalgo.ui.PlatformPages
import com.rahalgo.ui.PlatformScreen
import com.rahalgo.ui.ThemeState
import com.rahalgo.ui.TopBar
import com.rahalgo.ui.WalletScreen
import com.rahalgo.ui.WalletViewModel
import com.rahalgo.ui.rememberOverlay
import kotlinx.coroutines.launch
import com.rahalgo.ui.DrawerGestures

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تطبيقُ المتجر — بيتُ صاحبِ المطعم في جيبه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣: «بناء تطبيق المتجر بشكل كامل ومتكامل بناءً
 *  على المعطيات الموجودة لديك والمركزية الموجودة لديك».)
 *
 * # ولماذا بُني
 *
 * **كانت بوّابةُ الويب بيتَه الوحيد** — يفتح متصفّحاً ليقبل طلباً.
 * **وصاحبُ مطعمٍ يداه في العجين لا يفتح متصفّحاً**: يحتاج جرساً يرنّ
 * وزرّاً واحداً.
 *
 * **والمهلةُ تمضي وهو لا يعلم** — فيُحسب رفضاً ومخالفةً عليه.
 *
 * # وما كُتب هنا وما جاء من الوحدات
 *
 * **الدخولُ والحسابُ والمحفظةُ والإشعاراتُ والدرجُ وصفحاتُ المنصّة** —
 * **كلُّها من `:ui` بلا سطرٍ واحد** (القاعدة §7.1).
 *
 * **وما كُتب هنا**: أربعةُ تبويباتٍ وما تفتحه.
 *
 * # ولا خريطةَ فيه
 *
 * **المتجرُ لا يوصّل ولا يسجّل نفسَه** — نقطتُه تُوضع مرّةً عند
 * التسجيل من المندوب أو الإدارة. **ووحدةُ الخرائط ثقيلةٌ فلا تُفرض على
 * من لا يرسم.**
 *
 * # ولا تسجيلَ حسابٍ فيه
 *
 * **حسابُ المتجر من المنصّة** — كالسائق والمندوب. **وبابُ تسجيلٍ لمن
 * لا يستطيع أن يدخل منه يُملأ ثمّ يُردّ.**
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        com.rahalgo.ui.Net.install(this)
        com.rahalgo.ui.Images.install(this)
        Crash.start(debug = BuildConfig.DEBUG)
        Backend.of(this)
        // ══════════════════════════════════════════════════════════════
        // **ونقرةُ إشعارِ الطلب تفتح الطلبَ بعينه** (B3، ٢٠٢٦-٠٩-٢٦)
        // ══════════════════════════════════════════════════════════════
        //
        // **وكان المقصدُ يحمل وجهةَ الطلب ولا أحدَ يقرؤها** — فينقر صاحبُ
        // المتجر «طلبٌ جديد» فيُفتَح البيتُ لا طلبُه. **فتُقرأ هنا مرّةً**
        // (`Opened`)، **ويستهلكها الغلافُ** فيفتح تبويبَ الطلبات ويُبرز ذاك
        // الطلب. **وتُستهلَك مرّةً**: من أدار جهازَه لا يُساق إليه ثانية.
        com.rahalgo.ui.Opened.from(intent)
        WindowCompat.getInsetsController(window, window.decorView)
            .isAppearanceLightStatusBars = true
        setContent { MerchantApp() }
    }

    /**
     * **وإشعارٌ يُنقر والتطبيقُ مفتوح** — `onNewIntent` لا `onCreate`:
     * **بلا هذا يبقى مقصدُ الأمس فيُقرأ طلبُ أمس** (مرآةُ الزبون).
     */
    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        com.rahalgo.ui.Opened.from(intent)
    }
}

@Composable
private fun MerchantApp() {
    AppFrame { theme, dark ->
        val vm: AuthViewModel = viewModel()
        AuthGate(
            vm = vm,
            onSignedIn = { SignedIn(theme, dark, onLogout = vm::logout) },
            // ══════════════════════════════════════════════════════════
            // **وتحديثُ المتجر توزيعٌ مباشرٌ من الموقع لا Google Play** (B1)
            // ══════════════════════════════════════════════════════════
            //
            // **لا زرَّ متجرٍ يشير إلى قائمةٍ لا وجودَ للتطبيق فيها** — بل
            // تنزيلٌ مباشرٌ من صفحة تطبيق المتجر، ونصٌّ لدوره (يستقبل طلباتٍ
            // لا «يطلب»). **والرجوعُ لا يتخطّاه** (`UpdateGate` يُنهي النشاط)،
            // **ولا زرَّ تخطٍّ أو تأجيل** — الإصدارُ الأدنى قرارُ المالك.
            updateShowPlay = false,
            updateFallbackUrl = "https://rahalgo.com/download/merchant",
            updateBody = stringResource(R.string.update_body_merchant),
        )
    }
}

/**
 * **غلافُ المتجر** — شريطٌ علويٌّ ودرجٌ وأربعةُ تبويبات.
 *
 * **وترتيبُها بحسب ما يُفتح**: الطلباتُ عشرين مرّةً في اليوم، والقائمةُ
 * حين ينفد صنف، ومتجري صباحاً، **وحسابي نادراً.**
 */
@Composable
private fun SignedIn(theme: ThemeState, dark: Boolean, onLogout: () -> Unit) {
    val context = LocalContext.current

    // ══════════════════════════════════════════════════════════════════
    // **وحالُ «موقعي» تعيش مع الشاشة** (`MLW-02`، ٢٠٢٦-٠٩-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **وكان الزرُّ ينادي ويمضي** — **فمن رُفض إذنُه ضغطه فلم يقع شيءٌ
    // ولا رسالة.**
    val locating = remember { Locating() }

    // **وإذنُ الموقع يُطلب عند الزرّ لا عند الإقلاع** — **وسببُه أمام
    // عينه**: يصحّح دبّوسَ متجره.
    val askHere = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { ok ->
        if (ok) Here.refresh(context, locating, Accuracy.CONFIRM_M)
        else locating.failed(Locating.Problem.PERMISSION_DENIED)
    }

    // ══════════════════════════════════════════════════════════════════
    // **وإذنُ الإشعارات عصبُ هذا التطبيق**
    // ══════════════════════════════════════════════════════════════════
    //
    // **صاحبُ المتجر لا يجلس أمام الشاشة** — **وطلبٌ لا يرنّ طلبٌ
    // تمضي مهلتُه فيُحسب رفضاً عليه.**
    com.rahalgo.ui.AskNotifyPermission(enabled = true)

    val pagesVm: PagesViewModel = viewModel()
    val walletVm: WalletViewModel = viewModel()
    val accountVm: AccountViewModel = viewModel()
    val shell: ShellViewModel = viewModel()
    val ordersVm: OrdersViewModel = viewModel()
    val menuVm: MenuViewModel = viewModel()
    val storeVm: StoreViewModel = viewModel()
    val pickVm: PickPointViewModel = viewModel()
    // **وخريطةُ الدبّوس تغطّي الشاشةَ ولا تفتح صفحةً ثانية** — **ومن
    // خرج إلى صفحةٍ ليختار نقطةً عاد فلم يجد ما كتب.**
    var picking by rememberSaveable { mutableStateOf(false) }
    val historyVm: HistoryViewModel = viewModel()
    // ══════════════════════════════════════════════════════════════════
    // **وثلاثةُ أقسامٍ في الدرج تُبنى عند فتحها لا عند الإقلاع**
    // ══════════════════════════════════════════════════════════════════
    //
    // **و`viewModel()` هنا يبنيها كلَّها مع الشاشة** — وكلُّ واحدةٍ
    // تنادي المحرّكَ في `init`. **فأربعةُ نداءاتٍ تنطلق قبل أن يرى
    // صاحبُ المتجر طلباً واحداً**، على شبكةٍ ضعيفة.
    //
    // **فتُبنى داخل فرعها** — لا تُنادى حتّى يُفتح القسم.

    // ══════════════════════════════════════════════════════════════════
    // **وتبويبُ الطلبات لوضع المتاجر وحدَه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «قسمُ الطلبات هو فقط لاستقبال الطلبات
    //  الجديدة حين يكون وضعُ المتاجر يعمل».)
    //
    // **وفي وضع المنصّة المكتبُ يقبل ويحوّل** (`platform.orders_mode`)،
    // **والمحرّكُ يفرض `closed_only` من نفسه** — فالتبويبُ يعرض سجلّاً
    // في موضع عمل. **وتبويبٌ لا يملك فيه زرّاً يُقرأ عطباً لا سياسة.**
    //
    // **والسجلُّ في الدرج على كلّ حال** — فلا يفقد شيئاً.
    val selfManage = ordersVm.selfManage

    val drawer = rememberDrawerState(DrawerValue.Closed)
    val scope = rememberCoroutineScope()
    val overlay = rememberOverlay { key -> MERCHANT_ITEMS.any { it.key == key } }
    var tab by rememberSaveable { mutableStateOf(Tab.Orders) }

    // ══════════════════════════════════════════════════════════════════
    // **ونقرةُ إشعارِ الطلب تفتح ذاك الطلب بعينه** (B3، ٢٠٢٦-٠٩-٢٦)
    // ══════════════════════════════════════════════════════════════════
    //
    // **يقرأ الوجهةَ التي وضعها المقصدُ** (`Opened`)، **ويستهلكها مرّةً**
    // (`take`) فلا يُعاد فتحُها عند دوران الجهاز. **ووجهةُ الطلب تفتح تبويبَ
    // الطلبات وتُبرز ذاك الطلب** (`ordersVm.focus`) — والإبرازُ يبحث عنه في
    // الفروع إن لزم. **وفي وضع المنصّة لا تبويبَ للطلبات** فتُستهلَك الوجهةُ
    // بلا أثرٍ ضارّ (البابُ آمنٌ للطلب الشائخ).
    val waiting = com.rahalgo.ui.Opened.pending
    LaunchedEffect(waiting) {
        if (waiting.type == com.rahalgo.ui.Engagement.DEST_ORDER) {
            val dest = com.rahalgo.ui.Opened.take()
            overlay.clear()
            if (selfManage) {
                tab = Tab.Orders
                ordersVm.focus(dest.id)
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **جرسُ الطلب الجديد — يتكرّر ما دام طلبٌ ينتظر قراراً** (B2، ٢٠٢٦-٠٩-٢٦)
    // ══════════════════════════════════════════════════════════════════
    //
    // **مصدرُه قائمةُ الطلبات الحيّة** (تُنعشها النبضةُ والدورة، B4) — طلبٌ
    // في «pending» يعني قراراً معلّقاً. **فإن قُبل أو رُفض أو انقضت مهلتُه أو
    // عُولج من جهازٍ آخر، خرج من pending في أوّل إنعاش فيتوقّف الجرس** — لا
    // رنينَ يتيم. **ويُصمَت في الخلفيّة** (`ON_STOP`) ويعود عند الظهور
    // (`ON_RESUME`)، **فالإشعارُ العالي يتكفّل بالجيب المغلق.**
    val alerting = selfManage && ordersVm.orders.any { it.status == "pending" }
    LaunchedEffect(alerting) {
        com.rahalgo.merchant.push.MerchantOrderAlert.sync(context, alerting)
    }
    val alertOwner = androidx.compose.ui.platform.LocalLifecycleOwner.current
    androidx.compose.runtime.DisposableEffect(alertOwner, alerting) {
        val obs = androidx.lifecycle.LifecycleEventObserver { _, event ->
            when (event) {
                androidx.lifecycle.Lifecycle.Event.ON_STOP ->
                    com.rahalgo.merchant.push.MerchantOrderAlert.stop()
                androidx.lifecycle.Lifecycle.Event.ON_RESUME ->
                    com.rahalgo.merchant.push.MerchantOrderAlert.sync(context, alerting)
                else -> {}
            }
        }
        alertOwner.lifecycle.addObserver(obs)
        onDispose {
            alertOwner.lifecycle.removeObserver(obs)
            com.rahalgo.merchant.push.MerchantOrderAlert.stop()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **ومحرّرُ الصنف يُغلق بمغادرة بابه**
    // ══════════════════════════════════════════════════════════════════
    //
    // **والمحرّرُ يُرسم متى كان `editing` غيرَ فارغ** — لا متى كان
    // التبويبُ «إضافة صنف». **فمن فتحه ثمّ ذهب إلى «متجري» ورجع إلى
    // «الأصناف» وجده مفتوحاً**، ويظنّ التبويبَ عاطلاً.
    //
    // **و«الأصناف» تُغلقه بيدها** (انظر نداءَها) — **وهذه لِما عداها.**
    LaunchedEffect(tab) {
        if (tab != Tab.AddItem && tab != Tab.Menu) menuVm.cancelEdit()
    }

    BackHandler(enabled = drawer.isOpen) { scope.launch { drawer.close() } }

    ModalNavigationDrawer(
        drawerState = drawer,
        // ══════════════════════════════════════════════════════════════
        // **ولا تُفتح بالسحب — بالزرّ وحدَه**
        // ══════════════════════════════════════════════════════════════
        //
        // **(بلاغُ المالك ٢٠٢٦-٠٩-٠١:** «عند سحب الخريطة تُفتح القائمةُ
        // الجانبيّة — بكلّ التطبيقات».)
        //
        // **وحافّةُ الشاشة تلتقط السحبَ قبل الخريطة** — فمن حرّك
        // الخريطةَ يميناً فتح الدرجَ على نفسه، **ومن أراد أن يزيح
        // الدبّوسَ وجد قائمةً.**
        //
        // **والخريطةُ في الأربعة**: ملتقطُ النقطة في كلٍّ منها، والرحلةُ
        // في السائق. **فالتعطيلُ عامٌّ لا مشروط** — وشرطٌ يُكتب لكلّ
        // شاشةٍ يُنسى في الشاشة الخامسة.
        //
        // **ولا يُفقَد باب**: زرُّ «القائمة» في الشريط العلويّ في
        // التطبيقات كلِّها — **وهو أوضحُ من إيماءةٍ لا يعرفها إلّا من
        // جرّبها.**
        // ══════════════════════════════════════════════════════════════
        // **وتُفتَح بالسحب — إلّا وخريطةٌ تفاعليّةٌ حاضرة** (`DWR`، ٢٠٢٦-٠٩-١٤)
        // ══════════════════════════════════════════════════════════════
        //
        // **وكانت مُعطَّلةً عامّاً** بعد بلاغ المالك ٢٠٢٦-٠٩-٠١ («عند سحب
        // الخريطة تُفتح القائمةُ الجانبيّة») — **وسببُ التعميم مكتوبٌ
        // ومحقّ**: «شرطٌ يُكتب لكلّ شاشةٍ يُنسى في الشاشة الخامسة».
        //
        // **فصار الشرطُ واحداً في الوحدة المشتركة** (`DrawerGesturePolicy`)
        // **وتُعلنه الخريطةُ نفسُها** (`MapGestureLock` في `MapCanvas`
        // و`PickPoint`) — **فمن أضاف شاشةً سادسةً بخريطةٍ نال القفلَ بلا
        // أن يكتب سطراً.**
        //
        // **والدرجُ المفتوحُ يبقى قابلاً للإغلاق بالسحب** — **فلا يُحبَس
        // أحدٌ بدرجٍ لا يُغلَق**، **والخريطةُ خلفَ حاجبٍ حينها فلا تنازع.**
        gesturesEnabled = DrawerGestures.enabledFor(drawer),
        drawerContent = {
            ModalDrawerSheet(Modifier.width(200.dp)) {
                Drawer(
                    items = MERCHANT_ITEMS + PlatformPages.items,
                    onPick = { item ->
                        overlay.show(Overlay.Menu(item.key))
                        scope.launch { drawer.close() }
                    },
                    onLogout = onLogout,
                    dark = dark,
                    onTheme = { theme.toggle(dark) },
                )
            }
        },
    ) {
        val over = overlay.current

        Scaffold(
            topBar = {
                TopBar(
                    balance = shell.balance,
                    unread = shell.unread,
                    onMenu = { scope.launch { drawer.open() } },
                    onNotifications = {
                        overlay.toggle(Overlay.Inbox)
                        if (overlay.current == Overlay.Inbox) shell.openInbox()
                    },
                    onWallet = { overlay.show(Overlay.Wallet) },
                )
            },
            bottomBar = {
                NavigationBar {
                    // ══════════════════════════════════════════════════
                    // **الترتيبُ بحسب ما يُفتح — يميناً إلى يسار**
                    // ══════════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «أوّل قسمٍ يجب أن يكون
                    //  طلباتي في حال كان موجوداً، ثاني قسمٍ متجري، بعدها
                    //  إضافةُ صنف، بعدها الأصناف».)
                    //
                    // **والأوّلُ يقع تحت الإبهام** — وهو ما يُفتح أكثر.
                    if (selfManage) {
                        Tab(
                            selected = tab == Tab.Orders && over == Overlay.None,
                            onClick = { tab = Tab.Orders; overlay.clear() },
                            icon = com.rahalgo.ui.R.drawable.ic_orders,
                            label = R.string.nav_orders_all,
                        )
                    }
                    // **ومتجري ثانياً** — حالُه وأرقامُه يُفتحان صباحاً.
                    Tab(
                        selected = tab == Tab.Store && over == Overlay.None,
                        onClick = { tab = Tab.Store; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_store,
                        label = R.string.tab_store,
                    )
                    // ══════════════════════════════════════════════════
                    // **وإضافةُ صنفٍ تبويبٌ لا زرٌّ مخبوء**
                    // ══════════════════════════════════════════════════
                    //
                    // **وهو أكثرُ ما يفعله متجرٌ جديد** — يملأ بضاعتَه في
                    // أوّل أسبوع. **وفعلٌ يُفعل كلَّ يومٍ لا يُخبَّأ خلف
                    // شاشةٍ ثمّ قسمٍ ثمّ زرّ.**
                    Tab(
                        selected = tab == Tab.AddItem && over == Overlay.None,
                        onClick = {
                            tab = Tab.AddItem
                            overlay.clear()
                            menuVm.newItem()
                        },
                        icon = com.rahalgo.ui.R.drawable.ic_plus,
                        label = R.string.tab_add,
                    )
                    // ══════════════════════════════════════════════════
                    // **وأيقونةُ الأصناف بطاقةُ سعرٍ لا خطوطٌ ولا متجر**
                    // ══════════════════════════════════════════════════
                    //
                    // **الخطوطُ الثلاثةُ تعني «القائمة الجانبيّة»** في كلّ
                    // تطبيق — **ومعنيان لشكلٍ واحدٍ في شاشةٍ واحدةٍ يربكان.**
                    //
                    // **والمتجرُ محجوزٌ لتبويب «متجري»** — **وأيقونتان
                    // متطابقتان في شريطٍ واحدٍ تجعلان الضغطَ تخميناً.**
                    Tab(
                        // **ومضاءٌ في وضع المنصّة وإن كان الحالُ `Orders`**
                        // — **فتبويبٌ يُعرض محتواه ولا يُضاء يُقرأ عطباً**،
                        // ولا يعرف صاحبُ المتجر أين هو.
                        selected = (tab == Tab.Menu || (tab == Tab.Orders && !selfManage)) &&
                            over == Overlay.None,
                        // ══════════════════════════════════════════
                        // **و«الأصناف» تُغلق المحرّرَ لا تتركه مفتوحاً**
                        // ══════════════════════════════════════════
                        //
                        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٩: «ضغطتُ إضافة صنف
                        //  وفتح، فإذا رجعتُ وضغطتُ قسم الأصناف يبقى
                        //  بإضافة صنف ولا ينتقل».)
                        //
                        // **والمحرّرُ يُرسم متى كان `editing` غيرَ فارغ**
                        // — في التبويبين معاً. **فالتبويبُ يتبدّل ولا
                        // تتبدّل الشاشة**، ويُقرأ ذلك تعطّلاً في الزرّ.
                        //
                        // **والرجوعُ يُغلقه أصلاً** (`BackHandler` في
                        // `Tab.AddItem`) — **فالتبويبُ يفعل ما يفعله
                        // الرجوع**، ولا يفترق بابان عن بعضهما.
                        onClick = {
                            menuVm.cancelEdit()
                            tab = Tab.Menu
                            overlay.clear()
                        },
                        icon = com.rahalgo.ui.R.drawable.ic_offer,
                        label = R.string.mn_items,
                    )
                    // ══════════════════════════════════════════════════
                    // **وحسابي أيقونةُ شخصٍ لا صورةَ بروفايل**
                    // ══════════════════════════════════════════════════
                    //
                    // **وصورةُ الحساب فارغةٌ عند أكثر المتاجر** — فتُرسم
                    // حرفاً في دائرة، **وحرفٌ بين أيقوناتٍ يُقرأ شيئاً
                    // ناقصاً لا تبويباً.**
                    Tab(
                        selected = tab == Tab.Account && over == Overlay.None,
                        onClick = { tab = Tab.Account; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_user,
                        label = R.string.tab_account,
                    )
                }
            },
        ) { padding ->
            Column(
                Modifier
                    .fillMaxSize()
                    .padding(
                        top = padding.calculateTopPadding(),
                        bottom = padding.calculateBottomPadding(),
                    ),
            ) {
                // ══════════════════════════════════════════════════════
                // **ومبدّلُ الفرع فوق المحتوى** (B8، ٢٠٢٦-٠٩-٢٦)
                // ══════════════════════════════════════════════════════
                //
                // **لا يظهر لمن يملك متجراً واحداً** (`StoreSwitcher` يختفي
                // عند فرعٍ واحد)، **ولا فوق الخريطة ولا الصفحات المنبثقة ولا
                // حسابي** — تلك ليست مشهدَ فرع. **وتبديلُه يُعيد بناءَ
                // الشاشات كلِّها** (`SelectedStore.set` + `Refresh.bump`).
                if (!picking && over == Overlay.None && tab != Tab.Account) {
                    com.rahalgo.merchant.store.StoreSwitcher(
                        stores = storeVm.stores,
                        selectedId = com.rahalgo.merchant.SelectedStore.id,
                        onSelect = { id ->
                            com.rahalgo.merchant.SelectedStore.set(id)
                            com.rahalgo.ui.Refresh.bump()
                        },
                    )
                }
                Box(
                    Modifier
                        .fillMaxSize()
                        .weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    if (over != Overlay.None) BackHandler { overlay.clear() }
                    when {
                    // ══════════════════════════════════════════════
                    // **والخريطةُ فوقَ كلّ شيء**
                    // ══════════════════════════════════════════════
                    //
                    // **وأوّلُ الفروع لا آخرُها** — **ولو جاءت بعد
                    // تبويب «متجري» لَغطّاها هو فلا تُرى أبداً.**
                    picking -> PickPoint(
                        start = (storeVm.store?.lat to storeVm.store?.lng)
                            .let { (la, ln) ->
                                if (la != null && ln != null) {
                                    org.maplibre.android.geometry.LatLng(la, ln)
                                } else {
                                    com.rahalgo.ui.LastPoint.value?.let { p ->
                                        org.maplibre.android.geometry.LatLng(p.lat, p.lng)
                                    }
                                }
                            },
                        vm = pickVm,
                        onPick = { at, _ ->
                            storeVm.setPoint(at.latitude, at.longitude)
                            picking = false
                        },
                        onCancel = { picking = false },
                        // **ودبّوسُ المتجر يكفيه حدُّ التأكيد** —
                        // **ومئةُ مترٍ خطأً بابُ جارِه.**
                        onLocate = {
                            if (Here.granted(context)) {
                                Here.refresh(context, locating, Accuracy.CONFIRM_M)
                            } else {
                                askHere.launch(android.Manifest.permission.ACCESS_FINE_LOCATION)
                            }
                        },
                        // ══════════════════════════════════════════
                        // **وحالُ التحديد تُرسَم** (`MLW-02`)
                        // ══════════════════════════════════════════
                        //
                        // **فالزرُّ يدور ما دام النداءُ قائماً**،
                        // **والتعذّرُ يُقال بسببه لا «حدث خطأ».**
                        locating = locating,
                    )

                    over is Overlay.Menu && PlatformPages.has(over.key) ->
                        PlatformScreen(vm = pagesVm, key = over.key, role = HelpRole.Merchant)

                    // **وعروضُه** — **على المحرّك المركزيّ نفسِه**
                    // (`offers`)، **وبحارس قائمته** (`ownsMerchant`).
                    over is Overlay.Menu && over.key == MerchantItems.OFFERS -> {
                        val offersVm: com.rahalgo.merchant.offers.OffersViewModel = viewModel()
                        com.rahalgo.merchant.offers.OffersScreen(offersVm)
                    }

                    // **وسجلُّ الطلبات من الدرج** — قسمٌ منفصلٌ كما في الويب.
                    over is Overlay.Menu && over.key == MerchantItems.HISTORY ->
                        HistoryScreen(historyVm)

                    // **وما عليه** — يُقرأ قبل أن يُحظر.
                    over is Overlay.Menu && over.key == MerchantItems.WARNINGS -> {
                        val warnVm: WarningsViewModel = viewModel()
                        WarningsScreen(warnVm)
                    }

                    // **وما أرسله هو** — لا شكاوى الزبائن عليه.
                    over is Overlay.Menu && over.key == MerchantItems.REPORTS -> {
                        val repVm: MyReportsViewModel = viewModel()
                        MyReportsScreen(repVm)
                    }

                    // **ومدًى يختاره** — و«متجري» يقول يومَه وحدَه.
                    over is Overlay.Menu && over.key == MerchantItems.SALES -> {
                        val salesVm: SalesViewModel = viewModel()
                        SalesScreen(salesVm)
                    }

                    over == Overlay.Wallet -> WalletScreen(vm = walletVm, payouts = true)

                    over == Overlay.Inbox -> InboxSheet(
                        items = shell.inbox.orEmpty(),
                        onMarkAll = shell::markAllRead,
                    )

                    // **وفي وضع المنصّة لا تبويبَ للطلبات** — فمن كان
                    // عليه يُنقل إلى الأصناف، **ولا يبقى على شاشةٍ لا
                    // زرَّ له فيها.**
                    tab == Tab.Orders && selfManage -> OrdersScreen(ordersVm)

                    tab == Tab.Orders -> MenuScreen(menuVm)

                    tab == Tab.Menu -> MenuScreen(menuVm)

                    // **وصفحةُ الإضافة وجهةٌ بذاتها** — والرجوعُ منها
                    // يعيده إلى الأصناف لا إلى شاشةٍ لا زرَّ له فيها.
                    tab == Tab.AddItem -> {
                        BackHandler {
                            menuVm.cancelEdit()
                            tab = Tab.Menu
                        }
                        // ══════════════════════════════════════════════
                        // **ولا يُضاف صنفٌ قبل أن تُختار أقسامُ المتجر**
                        // ══════════════════════════════════════════════
                        //
                        // (قرارُ المالك ٢٠٢٦-٠٨-٢٩.)
                        //
                        // **والصنفُ ينزل في قسمٍ من أقسام السوق** — فإن
                        // لم يكن للمتجر أقسامٌ فلا قسمَ يُختار.
                        //
                        // **وبابٌ مغلقٌ بلا سببٍ يُقرأ عطباً** — فيُقال
                        // له لماذا، **ويُعطى الطريقَ لا مجرّدَ المنع.**
                        if (menuVm.suspended) {
                            // **والموقوفُ لا يضيف صنفاً** (B6) — يُقال له لماذا،
                            // ويُوجَّه إلى «متجري» حيث لافتةُ الإيقاف وتفصيلُها.
                            NeedSections(
                                title = stringResource(R.string.store_state_suspended),
                                body = stringResource(R.string.store_suspended_banner),
                            ) { tab = Tab.Store }
                        } else if (menuVm.sections.isEmpty() && !menuVm.loading) {
                            NeedSections { tab = Tab.Store }
                        } else {
                            MenuScreen(menuVm)
                        }
                    }

                    tab == Tab.Store -> StoreScreen(storeVm) { picking = true }

                    else -> AccountScreen(
                        vm = accountVm,
                        onLoggedOut = onLogout,
                        // ══════════════════════════════════════════
                        // **ولا عناوينَ في حساب المتجر**
                        // ══════════════════════════════════════════
                        //
                        // (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «العنوانُ يبقى فقط
                        //  للزبون لأنّه يلزم إضافةُ حسابٍ له — أمّا
                        //  الباقي فلا يلزمه».)
                        //
                        // **والعنوانُ حاجةُ من يُوصَّل إليه** — والمتجرُ عنوانُه في «متجري» بدبّوسه على الخريطة.
                        // **وقسمٌ يُفتح فلا يُملأ أبداً يُقرأ نقصاً في
                        // الحساب** لا اختياراً.
                        //
                        // **وافتراضُ الوسيط الإظهار** — فمن أضاف
                        // تطبيقاً ونسيه ورث عناوينَ لا تلزمه، **وهو ما
                        // وقع هنا حتّى قِيس.**
                        showAddresses = false,
                        // **وصاحبُ المتجر عاملٌ** — توثيقُ واتساب شرطُ عمله.
                        worker = true,
                        push = false,
                    )
                }
                }
            }
        }
    }
}

/** **تبويباتُ المتجر** — بترتيبها: يمينٌ إلى يسار. */
/**
 * **وجهاتُ الشريط السفليّ.**
 *
 * **و`AddItem` وجهةٌ لا فعل** (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «إضافة صنف
 * صفحة منفصلة عن الأصناف»).
 *
 * **وكان زرّاً يقفز إلى `Menu` ويفتح النموذج** — فيُضيء «الأصناف»
 * ويبقى مضيئاً، **فيظنّ صاحبُ المتجر أنّه في القائمة لا في صفحةِ
 * إضافة.** ورجوعُه كان يُلقيه في القائمة لا حيث كان.
 */
private enum class Tab { Orders, Menu, AddItem, Store, Account }

@Composable
private fun RowScope.Tab(
    selected: Boolean,
    onClick: () -> Unit,
    icon: Int,
    label: Int,
) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = { Icon(painterResource(icon), contentDescription = null) },
        label = { Text(stringResource(label)) },
    )
}


/**
 * **ما يُعرض لمن لم يختر أقسامَ متجره** — انظر `Tab.AddItem`.
 */
@Composable
private fun NeedSections(
    title: String = stringResource(R.string.need_sections_title),
    body: String = stringResource(R.string.need_sections_body),
    onGo: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            title,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            body,
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(20.dp))
        RahalButton(onClick = onGo) {
            Text(stringResource(R.string.need_sections_go))
        }
    }
}
