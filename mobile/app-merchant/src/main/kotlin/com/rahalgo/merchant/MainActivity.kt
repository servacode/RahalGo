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
import androidx.compose.foundation.layout.offset
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
import androidx.compose.runtime.remember
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
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind

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
            updateSelfKey = "merchant",
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
    // **والوصلةُ تُفتح لكلّ دخول** (٢٠٢٦-١٠-٠٦) — كانت تُفتح عند إنشاء النموذج وحدَه، **فمن خرج ودخل
    // في الجلسة نفسِها بقي بلا وصلةٍ ولا سؤالٍ احتياطيّ.** و`wake` لا يفتح مقبسين.
    LaunchedEffect(Unit) { shell.wake() }
    val ordersVm: OrdersViewModel = viewModel()
    val menuVm: MenuViewModel = viewModel()
    val storeVm: StoreViewModel = viewModel()

    // ══════════════════════════════════════════════════════════════════
    // **معالجُ الإعداد الأوّل يسبق التبويبات** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
    // ══════════════════════════════════════════════════════════════════
    //
    // **متجرٌ ينقصه إعدادٌ إلزاميّ لا يرى التبويبات حتّى يُكمله** — والناقصُ من
    // فحص «جهّز متجرك» نفسِه (`missingRequired`). **ومتجرٌ كاملٌ لا يراه أبداً.**
    LaunchedEffect(storeVm.store, storeVm.sectionsKnown, storeVm.mySections, storeVm.allSections) {
        storeVm.maybeStartWizard()
    }
    val wizardOn = storeVm.wizardSteps != null
    // **«لدي توصيلة»** (الخطوة ١٨) — ونقطةُ المستلِم على الخريطة نفسِها.
    // **ولا تحديدَ يدويّاً لنقطة المستلِم** (طلبُ المالك ٢٠٢٦-١٠-٠٩) — فرعُ خريطتها نُظّف
    // (تدقيقُ ٢٠٢٦-١٠-٢٧): الموقعُ من رابطٍ يشاركه الزبون.
    val deliveryVm: com.rahalgo.merchant.delivery.DeliveryViewModel = viewModel()
    // **«إحضار»: مبلغُ السائق ينتظر موافقتك — نافذةٌ فوق أيّ شاشة** (قرارُ المالك ٢٠٢٦-١٠-١٦).
    com.rahalgo.merchant.delivery.DeliveryTexts.pendingAmount(deliveryVm.list)?.let { d ->
        com.rahalgo.merchant.delivery.FetchAmountApproveDialog(
            d = d,
            busy = deliveryVm.amountBusy,
            error = deliveryVm.amountError,
            onApprove = { deliveryVm.decideAmount(d, true) },
            onReject = { deliveryVm.decideAmount(d, false) },
        )
    }
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
    // **ويفتح على «متجري»** (طلبُ المالك ٢٠٢٦-١٠-٠٨) — حالُه وما ينقصه أوّلُ ما يراه.
    var tab by rememberSaveable { mutableStateOf(Tab.Store) }

    // **و«رجوع» من تبويبٍ يعود إلى الأوّل لا يُخرج** (تقريرُ فحص المتجر) — كما
    // في تطبيق المندوب. **والخريطةُ والدرجُ والشاشاتُ المفتوحةُ تغلبه** (تُسجَّل
    // بعده فتُقدَّم)، **ولا يُفعَّل وهما مفتوحتان** — فيُسبَق بشرطهما هنا.
    androidx.activity.compose.BackHandler(
        enabled = tab != Tab.Store && !drawer.isOpen && !wizardOn,
    ) { tab = Tab.Store; menuVm.cancelEdit() }

    // ══════════════════════════════════════════════════════════════════
    // **ونقرةُ إشعارِ الطلب تفتح ذاك الطلب بعينه** (B3، ٢٠٢٦-٠٩-٢٦)
    // ══════════════════════════════════════════════════════════════════
    //
    // **يقرأ الوجهةَ التي وضعها المقصدُ** (`Opened`)، **ويستهلكها مرّةً**
    // (`take`) فلا يُعاد فتحُها عند دوران الجهاز. **ووجهةُ الطلب تفتح تبويبَ
    // الطلبات وتُبرز ذاك الطلب** (`ordersVm.focus`) — والإبرازُ يبحث عنه في
    // الفروع إن لزم. **وفي وضع المنصّة لا تبويبَ للطلبات** فتُستهلَك الوجهةُ
    // بلا أثرٍ ضارّ (البابُ آمنٌ للطلب الشائخ).
    // **أهذا الحسابُ موظّف؟** — يُسأل المحرّكُ مرّةً فيُخفى بابُ الموظّفين عنه.
    LaunchedEffect(Unit) { com.rahalgo.merchant.store.TeamAccess.probe() }

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
                androidx.lifecycle.Lifecycle.Event.ON_RESUME -> {
                    com.rahalgo.merchant.push.MerchantOrderAlert.sync(context, alerting)
                    // **والعودةُ تُجدّد كلَّ شاشة** (٢٠٢٦-١٠-٠٦) — ما وقع في الخلفيّة لم تحمله الوصلة.
                    shell.refresh()
                    com.rahalgo.ui.Refresh.bump()
                }
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

    // ══════════════════════════════════════════════════════════════════
    // **والنبضةُ تقرأ الظاهرَ وحدَه** (تدقيقُ ٢٠٢٦-١٠-٢٧) — انظر `StoreViewModel.active`.
    // ══════════════════════════════════════════════════════════════════
    val shown = overlay.current
    LaunchedEffect(tab, shown, selfManage, wizardOn) {
        val bare = shown == Overlay.None
        // **والمعالجُ يقرأ المتجرَ حيّاً** — هو الظاهرُ ما دام مفتوحاً.
        storeVm.active = (bare && tab == Tab.Store) || wizardOn
        menuVm.active = bare &&
            (tab == Tab.Menu || tab == Tab.AddItem || (tab == Tab.Orders && !selfManage))
        historyVm.active = shown is Overlay.Menu && shown.key == MerchantItems.HISTORY
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
        // **ولا درجَ والمعالجُ مفتوح** — لا تبويبَ ولا بابَ قبل أن يكتمل الإعداد.
        gesturesEnabled = !wizardOn && DrawerGestures.enabledFor(drawer),
        drawerContent = {
            ModalDrawerSheet(Modifier.width(200.dp)) {
                Drawer(
                    // **والموظّفُ لا يرى بابَ الموظّفين** — المحرّكُ يردّه (`notStaff`).
                    items = MERCHANT_ITEMS.filter {
                        !(it.key == MerchantItems.STAFF && com.rahalgo.merchant.store.TeamAccess.isStaff)
                    } + PlatformPages.items,
                    onPick = { item ->
                        if (item.key == MerchantItems.TUTORIAL) {
                            scope.launch { openTutorial(context) }
                        } else {
                            overlay.show(Overlay.Menu(item.key))
                        }
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

        if (wizardOn) {
            // **والدبّوسُ يُختار على خريطة الخطوة نفسِها** (٢٠٢٦-١٠-٢٨) — لا شاشةَ كاملةً فوقها.
            Box(Modifier.fillMaxSize()) {
                com.rahalgo.merchant.store.SetupWizard(
                    vm = storeVm,
                    onFinish = {
                        storeVm.finishWizard()
                        overlay.clear()
                        tab = Tab.Store
                    },
                )
            }
            return@ModalNavigationDrawer
        }

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
                // **شريطٌ أبيضُ بخطٍّ رفيعٍ فوقه، والمختارُ أيقونتُه في كبسولة** (التصميمُ المعتمد ٢٠٢٦-١٠-٢٨).
                val barLine = Rahal.colors.line
                NavigationBar(
                    containerColor = Rahal.colors.canvas,
                    tonalElevation = 0.dp,
                    modifier = Modifier.drawBehind {
                        drawLine(
                            color = barLine,
                            start = androidx.compose.ui.geometry.Offset(0f, 0f),
                            end = androidx.compose.ui.geometry.Offset(size.width, 0f),
                            strokeWidth = Rahal.stroke.hair.toPx(),
                        )
                    },
                ) {
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
                        // **ورقمٌ أحمرُ بعدد ما ينتظر قراره** (تدقيقُ ٢٠٢٦-١٠-٢٧) — الجرسُ يرنّ
                        // وهو في «متجري»، **والرقمُ يقول له أين يذهب.**
                        Tab(
                            selected = tab == Tab.Orders && over == Overlay.None,
                            onClick = { tab = Tab.Orders; overlay.clear() },
                            icon = com.rahalgo.ui.R.drawable.ic_orders,
                            label = R.string.nav_orders_all,
                            badge = ordersVm.pendingCount,
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
                    // **و«لدي توصيلة» في الشريط السفليّ** (الخطوة ١٨)
                    // ══════════════════════════════════════════════════
                    //
                    // (نصُّ المالك: «الزرّ يكون بالشريط السفلي».) **وقبل
                    // «حسابي»** — فعلُ عملٍ بين أفعال العمل، والحسابُ آخرُها.
                    Tab(
                        selected = tab == Tab.Delivery && over == Overlay.None,
                        onClick = { tab = Tab.Delivery; overlay.clear() },
                        icon = com.rahalgo.ui.R.drawable.ic_moto,
                        label = R.string.tab_delivery,
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
                if (over == Overlay.None && tab != Tab.Account && tab != Tab.Store) {
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
                    over is Overlay.Menu && PlatformPages.has(over.key) ->
                        PlatformScreen(vm = pagesVm, key = over.key, role = HelpRole.Merchant)

                    // **وعروضُه** — **على المحرّك المركزيّ نفسِه**
                    // (`offers`)، **وبحارس قائمته** (`ownsMerchant`).
                    over is Overlay.Menu && over.key == MerchantItems.OFFERS -> {
                        val offersVm: com.rahalgo.merchant.offers.OffersViewModel = viewModel()
                        // **وكلُّ فتحٍ يقرأ الفرعَ المختار** — النموذجُ يعيش مع النشاط.
                        LaunchedEffect(Unit) { offersVm.clearMerchant() }
                        com.rahalgo.ui.offers.OffersWizard(offersVm, onBack = { overlay.clear() })
                    }

                    // **وسجلُّ الطلبات من الدرج** — قسمٌ منفصلٌ كما في الويب.
                    over is Overlay.Menu && over.key == MerchantItems.HISTORY -> {
                        LaunchedEffect(Unit) { historyVm.load() }
                        HistoryScreen(historyVm)
                    }

                    // **وسجلُّ التوصيلات** — كلُّ «لدي توصيلة» أرسلها (الخطوة ١٨).
                    over is Overlay.Menu && over.key == MerchantItems.DELIVERIES -> {
                        val dhVm: com.rahalgo.merchant.delivery.DeliveryHistoryViewModel = viewModel()
                        com.rahalgo.merchant.delivery.DeliveryHistoryScreen(dhVm)
                    }

                    // **وما عليه** — يُقرأ قبل أن يُحظر.
                    over is Overlay.Menu && over.key == MerchantItems.WARNINGS -> {
                        val warnVm: WarningsViewModel = viewModel()
                        WarningsScreen(warnVm)
                    }

                    // **وما أرسله هو** — لا شكاوى الزبائن عليه.
                    over is Overlay.Menu && over.key == MerchantItems.REPORTS -> {
                        val repVm: MyReportsViewModel = viewModel()
                        LaunchedEffect(Unit) { repVm.load() }
                        MyReportsScreen(repVm)
                    }

                    // **ومدًى يختاره** — و«متجري» يقول يومَه وحدَه.
                    over is Overlay.Menu && over.key == MerchantItems.SALES -> {
                        val salesVm: SalesViewModel = viewModel()
                        LaunchedEffect(Unit) { salesVm.load() }
                        SalesScreen(salesVm)
                    }

                    // **ورقمُ الطلبات حسب الدوام وموظّفوه** (٢٠٢٦-١٠-٢٠) — للمتجر المختار،
                    // **والموقوفُ يقرأ ولا يعدّل** كما في الويب.
                    over is Overlay.Menu && over.key == MerchantItems.ORDER_PHONES -> {
                        val phonesVm: com.rahalgo.merchant.store.OrderPhonesViewModel = viewModel()
                        val st = storeVm.store
                        if (st == null) {
                            com.rahalgo.ui.Screen { com.rahalgo.ui.LoadState(storeVm.store == null, "") }
                        } else {
                            com.rahalgo.merchant.store.OrderPhonesScreen(
                                phonesVm,
                                storeId = st.id,
                                canManage = st.status != "suspended",
                                onBack = { overlay.clear() },
                            )
                        }
                    }

                    over is Overlay.Menu && over.key == MerchantItems.STAFF -> {
                        val staffVm: com.rahalgo.merchant.store.StaffViewModel = viewModel()
                        com.rahalgo.merchant.store.StaffScreen(
                            staffVm,
                            canManage = storeVm.store?.status != "suspended",
                            onBack = { overlay.clear() },
                        )
                    }

                    over == Overlay.Wallet -> WalletScreen(vm = walletVm, payouts = true)

                    over == Overlay.Inbox -> InboxSheet(
                        items = shell.inbox.orEmpty(),
                        onMarkAll = shell::markAllRead,
                        // **وخبرُ الطلب يفتح سجلَّ الطلبات** — ليرى ما يجهّز (٢٠٢٦-١٠-٢٩).
                        onOpen = { n ->
                            shell.markOneRead(n)
                            if (n.kind == "order") overlay.show(Overlay.Menu(MerchantItems.HISTORY))
                        },
                    )

                    // **وفي وضع المنصّة لا تبويبَ للطلبات** — فمن كان
                    // عليه يُنقل إلى الأصناف، **ولا يبقى على شاشةٍ لا
                    // زرَّ له فيها.**
                    tab == Tab.Orders && selfManage -> OrdersScreen(
                        ordersVm,
                        onHistory = { overlay.show(Overlay.Menu(MerchantItems.HISTORY)) },
                        onStore = { tab = Tab.Store },
                    )

                    tab == Tab.Orders -> MenuScreen(
                        menuVm,
                        onAddItem = { overlay.clear(); tab = Tab.AddItem; menuVm.newItem() },
                        onOffer = { overlay.show(Overlay.Menu(MerchantItems.OFFERS)) },
                    )

                    tab == Tab.Menu -> MenuScreen(
                        menuVm,
                        onAddItem = { overlay.clear(); tab = Tab.AddItem; menuVm.newItem() },
                        onOffer = { overlay.show(Overlay.Menu(MerchantItems.OFFERS)) },
                    )

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
                            NeedSections { storeVm.page = com.rahalgo.merchant.store.StorePage.Sections; tab = Tab.Store }
                        } else {
                            // **وبعد الحفظ أو الإلغاء يعود إلى «الأصناف»** (فحصُ ٢٠٢٦-١٠-٢٩) — كان
                            // يبقى على تبويب الإضافة فيرى الشبكةَ بلا «أنشئ عرضاً».
                            var opened by remember { mutableStateOf(false) }
                            LaunchedEffect(menuVm.editing != null) {
                                if (menuVm.editing != null) opened = true
                                else if (opened) tab = Tab.Menu
                            }
                            MenuScreen(
                                menuVm,
                                onOffer = { overlay.show(Overlay.Menu(MerchantItems.OFFERS)) },
                            )
                        }
                    }

                    // **وتنبيهُ «غير متوفر» يُحسب من الأصناف المحمّلة** — بلا نداءٍ جديد.
                    tab == Tab.Store -> StoreScreen(
                        storeVm,
                        unavailable = menuVm.groups.sumOf { g -> g.items.count { !it.available } },
                        onUnavailable = { menuVm.cancelEdit(); menuVm.showUnavailable(); tab = Tab.Menu },
                        // **الأزرارُ السريعة** (ملاحظةُ المالك ٦ب).
                        onOffers = { overlay.show(Overlay.Menu(MerchantItems.OFFERS)) },
                        onPrices = { menuVm.cancelEdit(); menuVm.bulk.show(); tab = Tab.Menu },
                        onDelivery = { overlay.clear(); tab = Tab.Delivery },
                        onTodayOrders = { overlay.show(Overlay.Menu(MerchantItems.HISTORY)) },
                        stores = storeVm.stores,
                        onSwitch = { id ->
                            com.rahalgo.merchant.SelectedStore.set(id)
                            com.rahalgo.ui.Refresh.bump()
                        },
                    )

                    tab == Tab.Delivery -> com.rahalgo.merchant.delivery.DeliveryScreen(deliveryVm)

                    // **وحسابي: شارةُ الدور أعلاه** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — «صاحب المتجر»
                    // أو «موظف». **ورُفع رأسُ الاسم والرقم والمحفظة** — مكرَّرٌ في الشاشة نفسِها.
                    else -> Column(Modifier.fillMaxSize()) {
                        RoleBadge()
                        Box(Modifier.weight(1f)) { AccountScreen(
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
                        worker = true,
                        // **ولا «وثّق حسابك»** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — حسابُ المتجر تصله
                        // رسائلُ الواتساب أصلاً.
                        whatsapp = false,
                        push = false,
                    ) }
                    }
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
private enum class Tab { Orders, Menu, AddItem, Store, Delivery, Account }

@Composable
private fun RowScope.Tab(
    selected: Boolean,
    onClick: () -> Unit,
    icon: Int,
    label: Int,
    badge: Int = 0,
) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = {
            Box {
                Icon(painterResource(icon), contentDescription = null)
                if (badge > 0) {
                    com.rahalgo.ui.CountBadge(
                        count = badge,
                        color = Rahal.colors.danger,
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .offset(x = 8.dp, y = (-6).dp),
                    )
                }
            }
        },
        // **ولا يلتفّ الاسمُ على شاشةٍ صغيرة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — ستّةُ تبويبات.
        label = {
            Text(
                stringResource(label),
                maxLines = 1,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
            )
        },
        // **والمختارُ بلون العلامة في كبسولةٍ فاتحة، والباقي باهت.**
        colors = androidx.compose.material3.NavigationBarItemDefaults.colors(
            selectedIconColor = Rahal.colors.brand,
            selectedTextColor = Rahal.colors.brand,
            indicatorColor = Rahal.colors.brand.copy(alpha = 0.12f),
            unselectedIconColor = Rahal.colors.inkMuted,
            unselectedTextColor = Rahal.colors.inkMuted,
        ),
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

/**
 * **شارةُ الدور أعلى «حسابي»** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — «صاحب المتجر» أو «موظف».
 *
 * **والدورُ من ردّ المحرّك** (`TeamAccess`: بابُ الموظّفين يردّ الموظّفَ `staff_not_allowed`)
 * — **ولا تُرسم قبل أن يُعرف**، فلا يقرأ موظّفٌ «صاحب المتجر» لحظةً ثمّ تتبدّل.
 * **ورُفع رأسُ الاسم والرقم والمحفظة** — كان يكرّر ما في الشاشة تحته.
 */
@Composable
private fun RoleBadge() {
    val access = com.rahalgo.merchant.store.TeamAccess
    if (!access.known) return
    val staff = access.isStaff
    androidx.compose.foundation.layout.Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            stringResource(if (staff) R.string.acc_role_staff else R.string.acc_role_owner),
            color = if (staff) Rahal.colors.ink else Rahal.colors.brand,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.labelLarge,
            modifier = Modifier
                .clip(Rahal.shape.pill)
                .background(if (staff) Rahal.colors.line else Rahal.colors.brand.copy(alpha = 0.12f))
                .padding(horizontal = Rahal.space.md, vertical = Rahal.space.xs),
        )
    }
}
