package com.rahalgo.rep

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.core.view.WindowCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.Rahal
import com.rahalgo.map.PickPoint
import com.rahalgo.map.PickPointViewModel
import com.rahalgo.rep.link.LinkScreen
import com.rahalgo.rep.board.BoardScreen
import com.rahalgo.rep.board.BoardViewModel
import com.rahalgo.rep.add.AddClientScreen
import com.rahalgo.rep.add.AddClientViewModel
import com.rahalgo.rep.clients.ClientsScreen
import com.rahalgo.rep.menu.MenuScreen
import com.rahalgo.rep.menu.MenuViewModel
import com.rahalgo.rep.clients.ClientsViewModel
import com.rahalgo.ui.AccountScreen
import com.rahalgo.ui.AccountViewModel
import com.rahalgo.ui.AppFrame
import com.rahalgo.ui.AuthGate
import com.rahalgo.ui.AuthViewModel
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Avatar
import com.rahalgo.ui.Crash
import com.rahalgo.ui.Drawer
import com.rahalgo.ui.InboxSheet
import com.rahalgo.ui.IncentivesScreen
import com.rahalgo.ui.IncentivesViewModel
import androidx.lifecycle.viewmodel.compose.viewModel as vmOf
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.rahalgo.shared.rep.RepApi
import com.rahalgo.ui.LastPoint
import com.rahalgo.ui.Overlay
import com.rahalgo.ui.PagesViewModel
import com.rahalgo.ui.HelpRole
import com.rahalgo.ui.PlatformPages
import com.rahalgo.ui.PlatformScreen
import com.rahalgo.ui.ThemeState
import com.rahalgo.ui.TopBar
import com.rahalgo.ui.WalletScreen
import com.rahalgo.ui.WalletViewModel
import com.rahalgo.ui.knowsKey
import com.rahalgo.ui.rememberOverlay
import com.rahalgo.ui.rememberTheme
import kotlinx.coroutines.launch
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إقلاع تطبيق المندوب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «ننهي مبدئيّاً تطبيق المندوب — هو قسمٌ
 *  خفيفٌ ونظيفٌ جدّاً».)
 *
 * # وكم كُتب هنا
 *
 * **الدخولُ والاستعادةُ والحسابُ والمحفظةُ والإشعاراتُ والدردشاتُ
 * والقائمةُ وصفحاتُ المنصّة** — **كلُّها من `:ui` بلا سطرٍ واحد.**
 *
 * **وما كُتب هنا**: تبويباتُه الثلاثةُ وما تفتحه.
 *
 * # ولا تسجيلَ حسابٍ فيه
 *
 * **حسابُ المندوب من المنصّة** — كالسائق. **وبابُ تسجيلٍ لمن لا يستطيع
 * أن يدخل منه يُملأ ثمّ يُردّ.**
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
        Crash.start(debug = BuildConfig.DEBUG)
        // **والنواةُ تُركَّب قبل أوّل شاشة** — `AuthViewModel` يصنعه
        // أندرويدُ لا نحن، **فيقرؤها من المُسجَّل.**
        Backend.of(this)
        WindowCompat.getInsetsController(window, window.decorView)
            .isAppearanceLightStatusBars = true
        setContent { RepApp() }
    }
}

@Composable
private fun RepApp() {
    // **والإطارُ من الوحدة** — السمةُ وشريطا النظام: **قِيس أنّها
    // متطابقةٌ في الثلاثة** (٢٠٢٦-٠٨-١٤).
    AppFrame { theme, dark ->
        val vm: AuthViewModel = viewModel()
        AuthGate(
            vm = vm,
            title = stringResource(R.string.login_title),
            // **ولا تسجيلَ حسابٍ للمندوب** — حسابُه من المنصّة كالسائق.
            onSignedIn = { SignedIn(theme, dark, onLogout = vm::logout) },
        )
    }
}

/**
 * **غلافُ المندوب** — شريطٌ علويٌّ ودرجٌ وثلاثةُ تبويبات.
 *
 * **وترتيبُها بحسب ما يُفتح**: لوحتُه كلَّ صباحٍ (كم بقي على الهدف؟)،
 * ومتاجرُه حين يتابع، **وحسابُه نادرا.**
 */
@Composable
private fun SignedIn(theme: ThemeState, dark: Boolean, onLogout: () -> Unit) {
    val context = LocalContext.current

    // **وإذنُ الإشعارات هنا** — هذه الشاشةُ لا تُرسم إلّا لمن دخل.
    // **وأمسك الحارسُ غيابَه** (٢٠٢٦-٠٨-١٩): المندوبُ ينتظر إشعاراتِ
    // عملائه وعمولاته، **وتُبتلع كلُّها بصمتٍ من أندرويد ١٣.**
    com.rahalgo.ui.AskNotifyPermission(enabled = true)
    val pagesVm: PagesViewModel = viewModel()
    val walletVm: WalletViewModel = viewModel()
    val accountVm: AccountViewModel = viewModel()
    val pickVm: PickPointViewModel = viewModel()
    val shell: ShellViewModel = viewModel()
    val clientsVm: ClientsViewModel = viewModel()
    val addVm: AddClientViewModel = viewModel()
    val boardVm: BoardViewModel = viewModel()
    // **وأصنافُ العميل** — يبنيها المندوبُ نيابةً عنه.
    val menuVm: MenuViewModel = viewModel()
    // **وهدفُه من الوحدة** — والبابُ `rep/incentives`.
    val app = context.applicationContext as android.app.Application
    val goalsVm: IncentivesViewModel = vmOf(
        factory = viewModelFactory {
            initializer {
                IncentivesViewModel(app) { RepApi(AppCore.get().api).incentives() }
            }
        },
    )
    // **والخريطةُ تغطّي النموذجَ ثمّ تعود بنقطةٍ واسم.**
    var picking by rememberSaveable { mutableStateOf(false) }

    val drawer = rememberDrawerState(DrawerValue.Closed)
    val scope = rememberCoroutineScope()
    val overlay = rememberOverlay { key -> knowsKey(REP_ITEMS, key) }
    var tab by rememberSaveable { mutableStateOf(Tab.Board) }

    // ══════════════════════════════════════════════════════════════════
    // **وإذنُ الموقع يُطلب عند فتح «إضافة عميل» — لا عند الإقلاع**
    // ══════════════════════════════════════════════════════════════════
    //
    // **إذنٌ يُطلب في أوّل شاشةٍ بلا سببٍ ظاهرٍ يُرفض** — ومن رفضه
    // مرّتين أُغلق البابُ في أندرويد ولا يُفتح إلّا من الإعدادات.
    //
    // **وهنا سببُه أمام عينه**: يقف في متجرٍ ليُسجّله.
    val askHere = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { ok -> if (ok) Here.refresh(context) }

    LaunchedEffect(tab) {
        if (tab != Tab.AddClient) return@LaunchedEffect
        if (Here.granted(context)) Here.refresh(context)
        else askHere.launch(android.Manifest.permission.ACCESS_FINE_LOCATION)
    }

    // **والرجوعُ من القائمة يغلقها — لا يُخرج من التطبيق.**
    BackHandler(enabled = drawer.isOpen) { scope.launch { drawer.close() } }
    BackHandler(enabled = picking && !drawer.isOpen) { picking = false }

    ModalNavigationDrawer(
        drawerState = drawer,
        drawerContent = {
            // **والدرجُ رفيعٌ لا عريض** — عرضُه الافتراضيُّ نصفُ الشاشة.
            ModalDrawerSheet(Modifier.width(200.dp)) {
                Drawer(
                    items = REP_ITEMS + PlatformPages.items,
                    onPick = { item ->
                        overlay.show(Overlay.Menu(item.key))
                        scope.launch { drawer.close() }
                    },
                    onLogout = onLogout,
                    // **ومبدّلُ السمة هنا** — (قرارُ المالك
                    // ٢٠٢٦-٠٨-١٥: «نخلّيها بالقائمة الجانبيّة»).
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
                    // **وضغطةٌ ثانيةٌ تُغلقها** — (قرارُ المالك
                    // ٢٠٢٦-٠٨-١٣، وكان في السائق وحدَه).
                    onNotifications = {
                        overlay.toggle(Overlay.Inbox)
                        if (overlay.current == Overlay.Inbox) shell.openInbox()
                    },
                    onWallet = { overlay.show(Overlay.Wallet) },
                )
            },
            bottomBar = {
                NavigationBar {
                    // **ولوحتُه أوّلا فهي في اليمين** — يفتحها كلَّ
                    // صباحٍ ليرى كم بقي على هدفه.
                    Tab(
                        selected = tab == Tab.Board && over == Overlay.None,
                        onClick = { tab = Tab.Board; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_star,
                        label = R.string.nav_board,
                    )
                    // ══════════════════════════════════════════════
                    // **وإضافةُ عميلٍ تبويبٌ لا زرٌّ مخبوء**
                    // ══════════════════════════════════════════════
                    //
                    // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «أضِف عنصراً رابعاً…
                    //  بين لوحتي وعملائي» — ثمّ: «أو نخلّيه إضافة
                    //  عميل، أفضل من متجر».)
                    //
                    // **والاسمان يتّفقان**: القائمةُ «عملائي»، **ومن
                    // أضاف «متجراً» ثمّ بحث عنه في «عملائي» توقّف
                    // لحظة.**
                    //
                    // **وما يُسجَّل ليس متجراً بعد** — المحرّكُ يسمّيه
                    // `lead` **ويبقى معلّقاً حتّى موافقة الإدارة**:
                    // **عميلٌ يُصبح متجراً حين يُقبل.**
                    //
                    // **وهو عملُ المندوب كلُّه** — يخرج صباحاً ليُسجّل
                    // متاجر. **وفعلٌ يُفعل كلَّ يومٍ لا يُخبَّأ خلف
                    // شاشةٍ أخرى.**
                    Tab(
                        selected = tab == Tab.AddClient && over == Overlay.None,
                        onClick = { tab = Tab.AddClient; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_plus,
                        label = R.string.nav_add_client,
                    )
                    // **وعملائي لا «متاجري»** — (تصحيحُ المالك
                    // ٢٠٢٦-٠٨-١٤). **والاسمُ يقول ما هي العلاقة**:
                    // «متاجري» تقول إنّه يملكها، **و«عملائي» تقول إنّه
                    // جلبهم ويتابعهم** — وهي الحقيقة.
                    Tab(
                        selected = tab == Tab.Clients && over == Overlay.None,
                        onClick = { tab = Tab.Clients; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_store,
                        label = R.string.nav_clients,
                    )
                    // **وحسابي آخرا فهو في اليسار** — وصورتُه لا أيقونة.
                    NavigationBarItem(
                        selected = tab == Tab.Account && over == Overlay.None,
                        onClick = { tab = Tab.Account; overlay.clear() },
                        icon = {
                            Avatar(
                                url = Backend.of(context).media(shell.me?.avatarThumbUrl),
                                name = shell.me?.fullName.orEmpty(),
                                size = 30,
                            )
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
                when {
                    over is Overlay.Menu && PlatformPages.has(over.key) ->
                        PlatformScreen(vm = pagesVm, key = over.key, role = HelpRole.Rep)

                    over is Overlay.Menu && over.key == RepItems.LINK -> LinkScreen(boardVm)

                    over is Overlay.Menu && over.key == RepItems.REWARDS -> {
                        LaunchedEffect(Unit) { goalsVm.load() }
                        IncentivesScreen(goalsVm)
                    }

                    over == Overlay.Wallet -> WalletScreen(vm = walletVm, payouts = true)

                    over == Overlay.Inbox -> InboxSheet(
                        items = shell.inbox.orEmpty(),
                        onMarkAll = shell::markAllRead,
                    )

                    // ══════════════════════════════════════════════
                    // **والخريطةُ تغطّي النموذجَ لا تفتح صفحةً ثانية**
                    // ══════════════════════════════════════════════
                    //
                    // **ومن خرج إلى صفحةٍ ليختار نقطةً عاد فلم يجد ما
                    // كتب.**
                    picking -> PickPoint(
                        start = LastPoint.value?.let { LatLng(it.lat, it.lng) },
                        vm = pickVm,
                        onPick = { at, name ->
                            addVm.setPoint(at.latitude, at.longitude, name)
                            picking = false
                        },
                        onCancel = { picking = false },
                        // **وأيقونةُ «موقعي»** — انظر تطبيقَ الزبون.
                        onLocate = { Here.refresh(context) },
                    )

                    tab == Tab.Board -> BoardScreen(boardVm)

                    tab == Tab.AddClient -> AddClientScreen(addVm) { picking = true }

                    // ══════════════════════════════════════════════
                    // **وأصنافُ العميل تغطّي تفصيلَه**
                    // ══════════════════════════════════════════════
                    //
                    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «لا يوجد بالتطبيق زرُّ
                    //  إضافة منتجات العميل».)
                    //
                    // **وقبل تبويب العملاء في الترتيب** — **ولو جاءت
                    // بعده لَغطّاه تبويبُ العملاء فلا تُرى أبدا.**
                    menuVm.merchantID.isNotEmpty() -> MenuScreen(menuVm)

                    tab == Tab.Clients -> ClientsScreen(clientsVm) { id, name ->
                        menuVm.open(id, name)
                    }

                    tab == Tab.Account -> AccountScreen(
                        vm = accountVm,
                        onLoggedOut = onLogout,
                        // **والمندوبُ عاملٌ** — توثيقُ واتساب شرطُ عمله.
                        worker = true,
                        // **ولا إشعاراتِ بعد** — تُضاف في خطوتها.
                        push = false,
                        picker = { onPick, onCancel ->
                            PickPoint(
                                start = LastPoint.value?.let { LatLng(it.lat, it.lng) },
                                vm = pickVm,
                                onPick = { at, name -> onPick(at.latitude, at.longitude, name) },
                                onCancel = onCancel,
                            )
                        },
                    )

                    // **وشاشتاه تُبنيان في دفعتيهما** — والقالبُ أوّلا.
                    else -> Soon(
                        when (tab) {

                            else -> "profile"
                        },
                    )
                }
            }
        }
    }
}

/** **تبويباتُ المندوب** — بترتيبها: يمينٌ إلى يسار. */
private enum class Tab { Board, AddClient, Clients, Account }

/** **بندٌ برسمٍ واسم** — وثلاثُ نسخٍ منه في شريطٍ واحدٍ حشوٌ يُنسخ. */
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
 * **قسمٌ لم يُبنَ بعد — ويقول ما ينتظره.**
 *
 * **وشاشةٌ بيضاءُ تُقرأ عطبا**: من فتحها ظنّ التطبيقَ مكسوراً فأغلقه.
 */
@Composable
private fun Soon(key: String) {
    val title: Int
    val hint: Int
    when (key) {
        "clients" -> {
            title = R.string.nav_clients
            hint = R.string.soon_clients
        }
        RepItems.LINK -> {
            title = R.string.menu_link
            hint = R.string.soon_link
        }
        RepItems.REWARDS -> {
            title = com.rahalgo.ui.R.string.menu_rewards
            hint = R.string.soon_rewards
        }
        else -> {
            title = R.string.nav_profile
            hint = R.string.soon_profile
        }
    }
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
        modifier = Modifier.padding(28.dp),
    ) {
        Text(stringResource(title), style = MaterialTheme.typography.headlineSmall)
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(hint),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
        )
    }
}
