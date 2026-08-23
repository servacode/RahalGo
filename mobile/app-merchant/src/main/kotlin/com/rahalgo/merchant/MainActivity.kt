package com.rahalgo.merchant

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
import com.rahalgo.map.Here
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
        WindowCompat.getInsetsController(window, window.decorView)
            .isAppearanceLightStatusBars = true
        setContent { MerchantApp() }
    }
}

@Composable
private fun MerchantApp() {
    AppFrame { theme, dark ->
        val vm: AuthViewModel = viewModel()
        AuthGate(
            vm = vm,
            title = stringResource(R.string.login_title),
            onSignedIn = { SignedIn(theme, dark, onLogout = vm::logout) },
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

    BackHandler(enabled = drawer.isOpen) { scope.launch { drawer.close() } }

    ModalNavigationDrawer(
        drawerState = drawer,
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
                            label = R.string.tab_orders,
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
                        selected = false,
                        onClick = {
                            tab = Tab.Menu
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
                        onClick = { tab = Tab.Menu; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_offer,
                        label = R.string.tab_menu,
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
                        onLocate = { Here.refresh(context) },
                    )

                    over is Overlay.Menu && PlatformPages.has(over.key) ->
                        PlatformScreen(vm = pagesVm, key = over.key, role = HelpRole.Merchant)

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

/** **تبويباتُ المتجر** — بترتيبها: يمينٌ إلى يسار. */
private enum class Tab { Orders, Menu, Store, Account }

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
