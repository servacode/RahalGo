package com.rahalgo.customer.shop

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.absoluteOffset
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.runtime.remember
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.rahalgo.customer.AddAction
import com.rahalgo.customer.Backend
import com.rahalgo.customer.PreCartViewModel
import com.rahalgo.customer.cart.Cart
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Section
import com.rahalgo.ui.Chip
import com.rahalgo.ui.CountBadge
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Refreshable
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.money
import com.rahalgo.ui.BannerSlide
import com.rahalgo.ui.BannerSlider
import com.rahalgo.ui.RahalLoader

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تسوّق — بحثٌ ثمّ أقسامٌ ثمّ أصناف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والترتيبُ ترتيبُ الويب حرفا** (`(site)/shop/page.tsx`) — **وشاشتان
 * لسوقٍ واحدٍ ترتيبُهما مختلفٌ تُربكان من يستعمل الاثنين.**
 *
 * # واسمُ المتجر سطرٌ في البطاقة — **قرارُ المالك ٢٠٢٦-١٠-١٥**
 *
 * **كانت البطاقةُ بلا أسماء متاجر** (٢٠٢٦-٠٨-٠٥: «يشتري من رحّال لا من مطعم
 * فلان»). **وعكسه المالكُ**: اسمُ المتجر تحت الصورة — انظر `ItemCard`.
 * **والتصفّحُ يبقى بالأقسام والأصناف لا بالمتاجر.**
 *
 * # وبطاقتان في الصفّ لا واحدة
 *
 * **الصورةُ هي ما يُشترى به** — وبطاقةٌ بعرض الشاشة تُظهر ثلاثةَ أصنافٍ
 * في شاشةٍ كاملة، **فيمرّ الإصبعُ طويلاً قبل أن يرى ما عنده.**
 */
@Composable
fun ShopScreen(
    vm: ShopViewModel,
    /** **الإعجابُ يحتاج حساباً** — والضيفُ يُساق إلى الدخول لا يُردّ. */
    onLike: (Item) -> Unit,
    liked: Set<String> = emptySet(),
    /**
     * **عنوانُ التوصيل المختار** — **وعليه تُقرأ الإتاحة** (`PC`).
     *
     * **ولا يُؤخذ موقعُ الجهاز عنواناً** — **من تسوّق من عمله يُوصَّل
     * إلى بيته.**
     */
    address: com.rahalgo.shared.model.Address? = null,
    /** **يُساق إلى اختيار عنوانٍ حين لا عنوانَ له** (`PC-02`). */
    onNeedAddress: () -> Unit = {},
    /** **المدنُ الفعّالة** (Batch 3c) — منها تُعرَف المدينةُ الرائدةُ لاستعراض سوقها. */
    cities: List<com.rahalgo.shared.model.City> = emptyList(),
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val media = { path: String? -> Backend.of(context).media(path) }

    // ══════════════════════════════════════════════════════════════════
    // **وحالُ العنوان تُقرأ قبل أن يُملأ شيء** (`PC`، ٢٠٢٦-٠٩-١٤)
    // ══════════════════════════════════════════════════════════════════
    //
    // **ومفتاحُها النقطةُ نفسُها** — **فتبديلُ العنوان يُبطل ما قبله**
    // **ولا تُقرأ حالُ عنوانٍ على عنوان.**
// **والنقطةُ السياقيّةُ**: **المؤكَّدةُ إن وُجدت، وإلّا فنقطةُ
    // الاستكشاف** — **وهي تُخبِر ولا تحكم** (`DL`).
    val ctx0 = com.rahalgo.customer.contextPoint(address, com.rahalgo.customer.Here.discovery)
    val point = com.rahalgo.customer.Orderable.pointKey(
        if (ctx0.source == com.rahalgo.customer.PointSource.NONE) null else ctx0.lat,
        if (ctx0.source == com.rahalgo.customer.PointSource.NONE) null else ctx0.lng,
    )
    val serviceVm: PreCartViewModel = androidx.lifecycle.viewmodel.compose.viewModel()
    androidx.compose.runtime.LaunchedEffect(point) { serviceVm.refreshAt(point, ctx0) }
    // **إشارةُ اللوحة/عودةُ الوصلة تُصحّحان إتاحةَ النقطة** (Batch 3b) — تجديدٌ
    // إجباريٌّ يتجاوز العمرَ، فيتبدّل السببُ/الحظرُ تلقائيّاً بلا إعادة فتحٍ يدويّة.
    androidx.compose.runtime.LaunchedEffect(point) {
        com.rahalgo.ui.Refresh.tick.collect { serviceVm.refreshAt(point, ctx0, force = true) }
    }
    // **العودةُ إلى الواجهة تُصحّح إتاحةَ العنوان دائماً** (Batch 3b، تصحيحُ المالك ٢):
    // تجديدٌ إجباريٌّ عند ON_RESUME يتجاوز عمرَ الدقيقتين — فالعائدُ لا يرى حظراً/سماحاً
    // شائخاً. (والعمرُ يبقى للاستطلاع الروتينيّ في تبدّل النقطة.)
    run {
        val shopOwner = androidx.compose.ui.platform.LocalLifecycleOwner.current
        androidx.compose.runtime.DisposableEffect(shopOwner, point) {
            val o = androidx.lifecycle.LifecycleEventObserver { _, e ->
                if (e == androidx.lifecycle.Lifecycle.Event.ON_RESUME) {
                    serviceVm.refreshAt(point, ctx0, force = true)
                }
            }
            shopOwner.lifecycle.addObserver(o)
            onDispose { shopOwner.lifecycle.removeObserver(o) }
        }
    }
    val availability = com.rahalgo.customer.Orderable.of(point)
    // **ولا يفتح الاستكشافُ الزرَّ** (`DL-03`، `DL-04`).
    val action = com.rahalgo.customer.addActionFor(ctx0, availability)

    // ══════════════════════════════════════════════════════════════════
    // **استعراضُ سوقِ المدينةِ المُطلَقة** (Batch 3c)
    // ══════════════════════════════════════════════════════════════════
    //
    // **المدينةُ الرائدة** (أوّلُ فعّالة = الرقّة) تُعرَض للقراءة فقط لمن لم تُطلَق
    // مدينتُه. **والدخولُ يُثبّت مدينةَ التصفّح، والخروجُ يُعيدها لعنوان الزبون** —
    // **ونقطةُ التوصيل لا تتبدّل، فتبقى الإضافةُ والطلبُ مغلقَين خادميّاً.**
    val flagship = com.rahalgo.customer.CityScope.flagship(cities)
    val previewing = com.rahalgo.customer.CityScope.preview
    val enterPreview: (() -> Unit)? = flagship?.let { fc ->
        { com.rahalgo.customer.CityScope.enterPreview(context, fc); vm.load() }
    }

    // **وما يقع عند الضغط** — **موضعٌ واحدٌ للقرار**: **وبطاقةٌ تقرّر
    // ونافذةُ خياراتٍ تقرّر تفترقان يوماً.**
    // **وفي وضع الاستعراض لا إضافة البتّة** (Batch 3c) — سوقُ القراءةِ فقط،
    // ونقطةُ التوصيلِ (مدينةُ الزبونِ التي لم تُطلَق) تحجب الإضافةَ خادميّاً كذلك.
    val blocked = action == AddAction.BLOCKED || previewing

    // **وتنبيهُ المتجر الثاني** (٢٠٢٦-١٠-١٤) — انظر `SourceNotice`.
    val sourceScope = androidx.compose.runtime.rememberCoroutineScope()
    com.rahalgo.customer.cart.SourceNoticeDialog()

    // **الصنفُ الذي تُختار خياراتُه الآن** — وفارغٌ حين لا نافذة.
    var picking by androidx.compose.runtime.remember {
        androidx.compose.runtime.mutableStateOf<Item?>(null)
    }
    picking?.let { target ->
        ItemOptionsSheet(
            item = target,
            api = vm.customerApi,
            onAdd = { chosen ->
                // **ولا تُضاف من النافذة ما مُنع من البطاقة** —
                // **وبابان لفعلٍ واحدٍ أحدُهما بلا حارسٍ بابٌ مفتوح.**
                if (!blocked) {
                    val inc = Cart.add(target, options = chosen)
                    com.rahalgo.customer.cart.SourceNotice.afterAdd(sourceScope, vm.customerApi, address, inc)
                    com.rahalgo.ui.Flash.ok(
                        context.getString(
                            if (inc) com.rahalgo.ui.R.string.cart_qty_updated
                            else com.rahalgo.ui.R.string.cart_item_added,
                        ),
                    )
                }
                picking = null
            },
            onClose = { picking = null },
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **والسحبُ إلى الأسفل يُنعش** — انظر `Refreshable`.
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «تحسّباً لأمرِ تحديثٍ لحظيٍّ لم يصل أو
    //  أيّ خللٍ آخر».)
    //
    // **والضيفُ لا وصلةَ حيّةَ له** (المقبسُ يحتاج توكناً) — **فهذه
    // طريقتُه الوحيدةُ ليرى الجديد.**
    Refreshable(refreshing = vm.refreshing, onRefresh = vm::refresh) {
    Box(Modifier.fillMaxSize()) {
    Column(Modifier.fillMaxSize()) {
        // **والبحثُ فوق الأقسام** — لمن يعرف ما يريد، **ولا ينزل تحتها
        // فيُبحث عنه.**
        // ══════════════════════════════════════════════════════════════
        // **وكلمةٌ تتبدّل في مربّع البحث**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نخلّيه ابحث عن، وكلماتٌ تتغيّر —
        //  مثال شاورما برغر والأصناف الموجودة… تظلّ تتبدّل الكلمةُ تلفت
        //  الانتباه».)
        //
        // **ومن أقسام السوق لا من قائمةٍ تُكتب بيد** — **وقائمةٌ ثابتةٌ
        // تعد بما ليس في السوق**: يقرأ «شاورما» فيبحث فلا يجد.
        //
        // **ولا تتبدّل وهو يكتب** — **ونصٌّ يتحرّك تحت إصبعه يُربكه**،
        // ولا معنى للإغراء بعد أن بدأ.
        // ══════════════════════════════════════════════════════════════
        // **وسلايدرُ اللافتات فوق البحث — كما في الويب حرفا**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «سلايدر صفحة التسوّق مثل سلايدر
        //  الصفحة الرئيسيّة، ولكن يجب أن تُطبَّق على الجوّال».)
        //
        // **والمحرّكُ يرسلها منذ زمنٍ ولا أحدَ يرسمها** — تُنشَر من
        // اللوحة ولا تُرى في الجوّال.
        //
        // **وفوق البحث لا تحته**: ترتيبُ الويب نفسُه (`HomeClient`)،
        // **وشاشتان بترتيبين تُحسّان تطبيقين.**
        //
        // **ولا تُرسم وهو يبحث** — نتائجُ بحثٍ تحتها لافتةُ عرضٍ
        // **تدفعها إلى نصف الشاشة**، ومن بحث يريد ما بحث عنه.
        if (!vm.searching && vm.banners.isNotEmpty()) {
            BannerSlider(
                items = vm.banners.map {
                    BannerSlide(
                        id = it.id,
                        title = it.title,
                        // **ونسخةُ ٩٦٠ لا الأصل** — انظر `mediaSized`:
                        // **اللوحُ عرضُه نحو ثلاثِ مئةٍ وأربعين نقطةً**،
                        // وألفٌ وستُّ مئةٍ فيه أربعةُ أضعافِ ما يُرى.
                        imageUrl = Backend.of(context)
                            .mediaSized(it.imageUrl, 960, it.sizes),
                        target = it.target,
                    )
                },
                auto = vm.bannerAuto,
                everyMs = vm.bannerEveryMs,
                // **ولا حشوةَ جانبيّةٌ هنا** — السلايدرُ يفسحها بنفسه
                // ليُظهر حافّةَ التالية.
                modifier = Modifier.padding(vertical = 10.dp),
            )
        }

        val hints = vm.visibleSections.map { it.name }
        var hint by remember { mutableStateOf(0) }
        LaunchedEffect(hints.size, vm.query.isEmpty()) {
            if (hints.isEmpty() || vm.query.isNotEmpty()) return@LaunchedEffect
            while (true) {
                kotlinx.coroutines.delay(2_000)
                hint = (hint + 1) % hints.size
            }
        }

        OutlinedTextField(
            value = vm.query,
            onValueChange = vm::type,
            // **ولا عنوانَ فوق الحقل** — **وعنوانٌ ثابتٌ يزاحم الكلمةَ
            // المتبدّلةَ فيُقرآن سطرين لا سطرا.**
            placeholder = {
                Text(
                    stringResource(
                        R.string.shop_search,
                        hints.getOrNull(hint).orEmpty(),
                    ),
                )
            },
            // **والعدسةُ في أوّل الحقل** — (طلبُ المالك)، **ومربّعٌ بلا
            // عدسةٍ يُقرأ حقلَ كتابةٍ لا بحثا.**
            leadingIcon = {
                Icon(
                    painter = painterResource(com.rahalgo.ui.R.drawable.ic_search),
                    contentDescription = null,
                    tint = Rahal.colors.inkMuted,
                    modifier = Modifier.size(20.dp),
                )
            },
            singleLine = true,
            // **ودائريٌّ كاملاً** — (طلبُ المالك ٢٠٢٦-٠٨-١٨): **وحقلُ
            // بحثٍ بزوايا حقلِ إدخالٍ يُقرأ حقلَ إدخال.**
            shape = Rahal.shape.pill,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 8.dp),
        )

        if (!vm.searching && vm.visibleSections.isNotEmpty()) {
            // ══════════════════════════════════════════════════════
            // **وحالُ العنوان تُقال فوق البضاعة** (`PC`، ٢٠٢٦-٠٩-١٤)
            // ══════════════════════════════════════════════════════
            //
            // **ومن قرأ «عنوانُك خارجَ النطاق» قبل أن يختار وفّر
            // رحلةً كاملة** — **ومن قرأه في السلّة قرأه عقوبة.**
            //
            // **وفوق الأقسام لا تحتها** — **ولافتةٌ تحت البضاعة
            // لا تُرى إلّا بعد أن يُملأ ما فوقها.**
            if (previewing) {
                // **لافتةُ الاستعراض الدائمة** (Batch 3c) — **أنت تتصفح سوق الرقة**،
                // والتوصيلُ إلى مدينتك غير متاح؛ وخروجٌ صريحٌ يعيدك إلى مدينتك.
                Column(Modifier.padding(horizontal = 12.dp)) {
                    com.rahalgo.ui.Note(
                        stringResource(
                            com.rahalgo.ui.R.string.preview_banner_named,
                            com.rahalgo.customer.CityScope.chosen?.name ?: flagship?.name ?: "",
                        ),
                        Rahal.colors.brand,
                    )
                    Spacer(Modifier.height(8.dp))
                    com.rahalgo.ui.RahalOutlineButton(
                        onClick = { com.rahalgo.customer.CityScope.exitPreview(context); vm.load() },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(com.rahalgo.ui.R.string.preview_exit)) }
                    Spacer(Modifier.height(10.dp))
                }
            } else {
                availability?.takeIf { !it.available }?.let { av ->
                    Column(Modifier.padding(horizontal = 12.dp)) {
                        com.rahalgo.customer.ServiceBlockNotice(
                            av,
                            address,
                            serviceVm,
                            // **ونقطةُ الاستكشاف تُصاغ «موقعك الحالي»** —
                            // **ولا تُقرأ عنوانَ توصيلٍ مؤكَّدا** (`A2`).
                            discovery = ctx0.source == com.rahalgo.customer.PointSource.DISCOVERY,
                            flagshipName = flagship?.name ?: "",
                            onBrowseFlagship = enterPreview,
                        )
                        Spacer(Modifier.height(10.dp))
                    }
                }
            }

            SectionRail(vm.visibleSections, vm.pick, media, vm::openSection)
        }

        when {
            vm.busy && vm.items.isEmpty() ->
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    RahalLoader()
                }

            vm.error.isNotEmpty() && vm.items.isEmpty() ->
                LoadState(loading = false, error = vm.error, onRetry = vm::load)

            // ══════════════════════════════════════════════════════
            // **وسوقٌ فارغةٌ غيرُ قسمٍ فارغ** (٢٠٢٦-٠٩-١٣)
            // ══════════════════════════════════════════════════════
            //
            // **والمنصّةُ تُنزَّل قبل أن تمتلئ**: **يُثبِّت الزبونُ
            // التطبيقَ ولا متجرَ بعد.** **فقال له «لا أصناف في هذا
            // القسم» وهو لا يرى قسماً أصلاً** — **فيظنّ أنّه أخطأ
            // الطريقَ أو أنّ التطبيقَ معطوب.**
            //
            // **ولا يُختلَق صنفٌ ليبدو عامراً** — **الصدقُ سطرٌ وزرٌّ
            // يعيد المحاولة.**
            // ══════════════════════════════════════════════════════
            // **وسوقٌ لم تمتلئ بعد — حالٌ مقصودةٌ بلا زرِّ إعادة**
            // ══════════════════════════════════════════════════════
            //
            // **وكان الشرطُ `sections.isEmpty()`** — **والأقسامُ لا
            // تفرغ أبداً**: تسعةٌ في القاعدة وإن لم يكن فيها صنف.
            // **فلم تُعرَض هذه الحالُ قطّ**، **وعُرض بدلَها «لا أصنافَ
            // في هذا القسم» في كلّ قسمٍ من التسعة.**
            //
            // **والآن يُسأل عن المحتوى لا عن عدد الأقسام** — `marketEmpty`.
            //
            // **ولا زرَّ إعادةٍ هنا**: **الردُّ ناجحٌ بصفر متاجر**،
            // **وإعادةُ نداءٍ ناجحٍ تردّ جوابَه عينَه.** **وزرٌّ لا
            // يغيّر شيئاً يُضغط ثمّ يُفقَد الرجاءُ بالتطبيق.**
            // **والسوقُ الفارغةُ تُعرَض أقساماً** (٢٠٢٦-١٠-٠٦) — ولا تُطوى إلى رسالةٍ إلّا خارجَ التغطية.
            (vm.sections.isEmpty() || (vm.marketEmpty && availability?.available == false)) && !vm.searching -> {
                // ══════════════════════════════════════════════════════
                // **وسوقٌ فارغةٌ خارجَ التغطيةِ ليست «قريباً»** (`CUST-07-034`)
                // ══════════════════════════════════════════════════════
                //
                // **خارجَ التغطيةِ لا متجرَ مخزَّناً**، فتفرغ `visibleSections`،
                // **فيسقط تنبيهُ التغطيةِ فوقُ** (شرطُه `visibleSections
                // .isNotEmpty()`) **ويُعرَض بدلَه «نعمل على إضافة المتاجر»**
                // — **فيقرأ المقصيُّ «قريباً» لا «لم نصل إليك».**
                //
                // **والفرقُ حكمٌ لا زخرفة**: **«قريباً» تقول انتظر، و«لم
                // نصل» تقول أخبرني** — **ودفترُ الطلب يُبنى على الثانية.**
                // (والسلّةُ تقولها صحيحةً منذ زمن.) **فحين تُقصى النقطةُ**
                // يُعرَض تنبيهُ الخدمةِ نفسُه — سببٌ صريحٌ وزرُّ «أخبرني»
                // (والزرُّ لا يظهر لنقطةِ استكشافٍ ولا بلا عنوانٍ مؤكَّد،
                // وهو عقدُ `CUST-07-030` القائم).
                val blocked0 = availability?.takeIf { !it.available }
                if (blocked0 != null) {
                    Column(Modifier.fillMaxSize().padding(horizontal = 12.dp, vertical = 24.dp)) {
                        com.rahalgo.customer.ServiceBlockNotice(
                            blocked0,
                            address,
                            serviceVm,
                            discovery = ctx0.source == com.rahalgo.customer.PointSource.DISCOVERY,
                            // **ولمدينةٍ لم تُطلَق: زرُّ «استعرض سوق الرقة»** (Batch 3c).
                            flagshipName = flagship?.name ?: "",
                            onBrowseFlagship = enterPreview,
                        )
                    }
                } else {
                    // **وداخلَ التغطيةِ سوقٌ لم تمتلئ بعدُ** — «قريباً» صادقة.
                    Empty(
                        text = stringResource(R.string.shop_market_empty),
                        hint = stringResource(R.string.shop_market_empty_hint),
                        actionLabel = "",
                        onAction = null,
                    )
                }
            }

            vm.items.isEmpty() -> Empty(
                text = stringResource(
                    if (vm.searching) R.string.shop_no_results else R.string.shop_section_empty,
                ),
                hint = stringResource(
                    if (vm.searching) R.string.shop_no_results_hint else R.string.shop_section_empty_hint,
                ),
                // **ولا زرَّ إعادةٍ على نجاحٍ بصفر** — **والخطأُ له
                // حالُه فوق، وفيها الإعادةُ تنفع.**
                actionLabel = "",
                onAction = null,
            )

            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(ShopCols),
                // ══════════════════════════════════════════════════════
                // **وذيلٌ يفسح لقرص الحديث**
                // ══════════════════════════════════════════════════════
                //
                // (رُئي في لقطة المتجر ٢٠٢٦-٠٨-٢٠: القرصُ الطافي يغطّي
                //  آخرَ بطاقةٍ في الشبكة.)
                //
                // **والقرصُ يطفو فوق المحتوى فلا يزيحه** — **وبطاقةٌ
                // نصفُها تحت زرٍّ تُقرأ عطباً**، ومن أرادها لم يبلغ
                // زرَّ إضافتها.
                //
                // **والذيلُ في الحشوة لا في عنصرٍ فارغٍ آخرَ القائمة** —
                // فيبقى التمريرُ ينتهي حيث ينتهي المحتوى.
                contentPadding = PaddingValues(
                    start = 12.dp, end = 12.dp, top = 12.dp, bottom = 96.dp,
                ),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                items(vm.items, key = { it.id }) { item ->
                    ItemCard(
                        item = item,
                        media = media,
                        liked = item.id in liked,
                        onAdd = {
                            // **وضعُ الاستعراض للقراءة فقط** (Batch 3c) — لا إضافةَ
                            // ولا سؤالَ عنوان؛ اللافتةُ فوقُ تشرح، والخروجُ صريح.
                            if (previewing) return@ItemCard
                            // **ولا عنوانَ لا إضافة** — **يُسأل أوّلاً**
                            // (`PC-02`)، **ولا يُخترَع له عنوان.**
                            if (action == AddAction.NEED_ADDRESS) {
                                onNeedAddress()
                                return@ItemCard
                            }
                            // **وعنوانٌ لا يُخدَم لا يُملأ له سلّة** —
                            // **والسببُ مكتوبٌ فوق القائمة** (`PC-03`…).
                            if (blocked) return@ItemCard
                            // **ومغلقٌ لا يُضاف** — ومن أضافه ثمّ رُدّ
                            // عند الدفع أضاع وقتَه.
                            if (item.available && !item.sourceClosed) {
                                // **وما له خياراتٌ يُسأل قبل أن يدخل** —
                                // **والمجموعةُ الإلزاميّةُ تُسقط الطلبَ
                                // كلَّه** إن دخل بلا اختيار.
                                if (item.hasOptions) {
                                    picking = item
                                } else {
                                    val inc = Cart.add(item)
                                    com.rahalgo.customer.cart.SourceNotice.afterAdd(sourceScope, vm.customerApi, address, inc)
                                    com.rahalgo.ui.Flash.ok(
                                        context.getString(
                                            if (inc) com.rahalgo.ui.R.string.cart_qty_updated
                                            else com.rahalgo.ui.R.string.cart_item_added,
                                        ),
                                    )
                                }
                            }
                        },
                        onLike = { onLike(item) },
                    )
                }
            }
        }

    }


    }
    }
}

/**
 * **كم صنفاً في السطر.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «سطرُ الهاتف لازم ياخذ ٣ منتجات وليس
 *  منتجين».)
 *
 * **واثنتان تعنيان ستّةً في شاشةٍ كاملة** — ومن عنده أربعون صنفاً
 * يمرّ إصبعُه سبعَ مرّاتٍ ليراها. **وثلاثةٌ تُري تسعة.**
 *
 * **وتقرؤه المفضّلةُ معها** — **شبكتان بعددين تجعلان الصنفَ سلعةً في
 * شاشةٍ وسطرَ جدولٍ في أخرى.**
 */
const val ShopCols = 3

/**
 * **مقاسُ السلّة العائمة.**
 *
 * **ورقمٌ واحدٌ يحكمها** — الأيقونةُ وموضعُ العدد في جوفها يُشتقّان
 * منه: **ورقمان مكتوبان في موضعين يفترقان يومَ يتبدّل أحدُهما.**
 */

/**
 * **شريطُ الأقسام — صورٌ دائريّةٌ تمشي.**
 *
 * **والسوقُ يُتصفَّح بالصور لا بالرموز**: يعرف الشاورما من صورتها قبل أن
 * يقرأ اسمَها، **ورمزٌ رماديٌّ لعشرة أقسامٍ يجعلها كلَّها شيئاً واحدا.**
 */
@Composable
private fun SectionRail(
    sections: List<Section>,
    pick: String?,
    media: (String?) -> String?,
    onPick: (String) -> Unit,
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = 14.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(sections, key = { it.id }) { s ->
            val on = s.id == pick
            // ══════════════════════════════════════════════════════════
            // **ولا أثرَ للضغطة هنا**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «يظهر مربّعٌ حول صورة القسم
            //  وهذا خطأٌ قاتلٌ غيرُ مقبولٍ تصميما — ألغِه».)
            //
            // **والصورةُ دائرةٌ والأثرُ مربّع**: الضغطةُ على العمود
            // كلِّه — اثنتان وسبعون نقطةً عرضاً — **فيُرسم مستطيلٌ
            // حول دائرة.**
            //
            // **والجوابُ يصل قبل أن يتلاشى الأثر**: الأصنافُ تتبدّل
            // واسمُ القسم يُلوَّن ويثقُل. **وأثرٌ يقول «سمعتُك» حيث
            // يقوله ما وقع فعلاً زائد.**
            // ══════════════════════════════════════════════════════════
            // **والمختارُ يكبر وتُحيطه حلقة**
            // ══════════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «يجب أن نضيف سكيل للقسم المختار
            //  ليكون واضحاً للمستخدم في أيّ قسمٍ هو، مع إضافة أوت لاين
            //  للقسم المختار».)
            //
            // **ولا يناقض قرارَه ٢٠٢٦-٠٨-١٥** — رفض يومَها **مربّعاً
            // حول دائرة**، وهو محقّ: الصورةُ دائرةٌ والأثرُ كان مستطيلاً
            // بعرض العمود كلِّه. **وهذه حلقةٌ دائريّةٌ تعانق الصورة.**
            //
            // **والحجمُ الخارجيُّ ثابت** — الكبرُ في الصورة وحدَها داخل
            // صندوقٍ لا يتبدّل. **وإلّا قفز الشريطُ كلُّه عند كلّ
            // ضغطة**، وذاك أسوأُ ممّا يُصلح.
            //
            // **والانتقالُ متحرّكٌ لا قفزة** — **وتبدّلٌ فوريٌّ يُقرأ
            // ارتجافاً لا استجابة.**
            val ring by animateDpAsState(
                targetValue = if (on) 2.5.dp else 0.dp,
                label = "sectionRing",
            )
            val face by animateDpAsState(
                targetValue = if (on) 64.dp else 58.dp,
                label = "sectionFace",
            )
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier
                    .width(72.dp)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onPick(s.id) },
            ) {
                Box(
                    modifier = Modifier.size(66.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    RemoteImage(
                        url = media(s.imageThumbUrl ?: s.imageUrl),
                        name = s.name,
                        modifier = Modifier
                            .size(face)
                            .border(ring, Rahal.colors.brand, CircleShape)
                            // **وفراغٌ بين الحلقة والصورة** — **وحلقةٌ
                            // ملاصقةٌ تُقرأ حافّةً للصورة لا اختيارا.**
                            .padding(ring + if (on) 1.5.dp else 0.dp)
                            .clip(CircleShape),
                    )
                }
                Spacer(Modifier.height(4.dp))
                // ══════════════════════════════════════════════════════
                // **واسمُ القسم يُكتب كاملاً — سطرين إن لزم**
                // ══════════════════════════════════════════════════════
                //
                // (شكوى المالك ٢٠٢٦-٠٨-١٨: «أسماءُ الأقسام بعضُها ناقصٌ
                //  بنقط، وهذا غلطٌ ما يصير».)
                //
                // **وسطرٌ واحدٌ في اثنتين وسبعين نقطةً لا يسع «بيتزا
                // وفطائر»** — فيصير «بيتزا وف…». **واسمٌ مقصوصٌ يُقرأ
                // قسماً آخرَ** أو يُقرأ عطباً في الرسم.
                //
                // **وسطران يكفيان أطولَ أقسامه** — والثالثُ يجعل
                // الشريطَ يعلو على حساب البضاعة تحته.
                //
                // **والارتفاعُ يُحجز للسطرين** (`minLines`) — **وإلّا
                // قفزت الدوائرُ صفّاً حين يطول اسمٌ ويقصر آخر**، فيبدو
                // الشريطُ مضطربا.
                Text(
                    text = s.name,
                    // **والمختارُ يُعرف بلونه وثقله** — لا بإطارٍ يزاحم
                    // الصورة.
                    color = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
                    fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
                    style = MaterialTheme.typography.labelMedium,
                    minLines = 2,
                    maxLines = 2,
                    overflow = TextOverflow.Visible, // **ولا قصّ** — نقاطُ ما تحت السطر (٢٠٢٦-١٠-١٤)
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ صنفٍ مؤطَّرة — شريطٌ فوق الصورة وشريطٌ تحتها** (قرارُ المالك ٢٠٢٦-١٠-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **إطارٌ رفيعٌ يضمّ ثلاثةَ طوابق**: فوقُ القلبُ والسعر، **ثمّ الصورة**، ثمّ
 * تحتُ **اسمُ المتجر** و«+». **واسمُ الصنف تحت الإطار** كما كان.
 *
 * # ولماذا اسمُ المتجر الآن
 *
 * **عكس المالكُ قرارَ حجب المصدر** (كان ٢٠٢٦-٠٨-٠٥): كلُّ صنفٍ يحمل اسمَ
 * متجره من المحرّك (`merchant_name`). **وفارغُه من خادمٍ أقدم يترك السطرَ
 * فارغاً ولا يكسر البطاقة.**
 *
 * # ولماذا السعرُ والقلبُ خرجا من الصورة
 *
 * **كانا قرصين على الصورة** — والسعرُ أبيضُ على قرصٍ داكن **لأنّ صورَ الطعام
 * زاهيةٌ تبتلع النصّ.** **وفي شريطٍ على سطح البطاقة يُقرآن بلونهما بلا قرص**،
 * والصورةُ تبقى صورةً كاملة.
 *
 * # وبطاقةٌ واحدةٌ لشاشتين
 *
 * **السوقُ والمفضّلةُ تعرضان الصنفَ نفسَه** — ونسختان من البطاقة تفترقان يوما
 * (كما افترقت المفضّلةُ فعرضت السعرَ الخامّ). **و`onAdd` فارغٌ لا «+» فيه.**
 *
 * **والمشطوبُ هو ما يجعل الخصمَ خصما**: «٧٧٬٢٥٠» وحدَه رقم، **و«١٠٣٬٠٠٠»
 * مشطوبةً تحته توفيرٌ يُرى.**
 */
@Composable
internal fun ItemCard(
    item: Item,
    media: (String?) -> String?,
    liked: Boolean,
    onAdd: (() -> Unit)?,
    onLike: () -> Unit,
) {
    val closed = item.sourceClosed || !item.available
    Column {
        Column(
            Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md)
                .background(Rahal.colors.surface),
        ) {
            // ══════════════════════════════════════════════════════════
            // **الشريطُ الأعلى — القلبُ في البداية والسعرُ في النهاية**
            // ══════════════════════════════════════════════════════════
            //
            // **وزرّان في طرفين لا في زاوية** — زرّان متلاصقان يُضغط أحدُهما
            // مكان الآخر.
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = Rahal.space.xs, vertical = Rahal.space.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // **وأحمرُ حين يُضغط ورماديٌّ حين لا** (ملاحظةُ المالك ٢٠٢٦-٠٨-١٨) —
                // **والأحمرُ لغةُ المفضّلة في كلّ تطبيق.**
                Box(
                    Modifier
                        .clip(CircleShape)
                        .clickable(onClick = onLike)
                        .padding(Rahal.space.xs),
                ) {
                    Icon(
                        painter = painterResource(
                            if (liked) com.rahalgo.ui.R.drawable.ic_heart
                            else com.rahalgo.ui.R.drawable.ic_heart_off,
                        ),
                        // **والوصفُ يتبع الحال** — **ومن يسمع ولا يرى
                        // لا يعرف أنّ ضغطتَه وقعت.**
                        contentDescription = stringResource(
                            if (liked) R.string.shop_unlike else R.string.shop_like,
                        ),
                        tint = if (liked) Rahal.colors.danger else Rahal.colors.inkMuted,
                        modifier = Modifier.size(18.dp),
                    )
                }
                Spacer(Modifier.weight(1f))
                // **والليرةُ فوق مع الدولار لا تحت الاسم** (ملاحظةُ المالك ٢٠٢٦-١٠-١٥:
                // «تحت اسم الصنف ما رح يكون واضح») — **يقرأ الزبونُ ما سيدفعه بلمحةٍ
                // واحدة** — **وفي سطرٍ واحدٍ مع الدولار** (ملاحظتُه الثانية في اليوم نفسِه):
                // `$1.8 ≈ 235 ل.س`. والليرةُ أصغرُ خطّاً، وإن ضاق العرضُ تُختصر هي لا الدولار.
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        // **وبالدولار لمتجرٍ اختاره** (الخيار ١، ٢٠٢٦-١٠-١٢).
                        text = item.priceUsd?.let { com.rahalgo.ui.dollars(it) } ?: money(item.price),
                        color = Rahal.colors.brand,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        style = MaterialTheme.typography.labelMedium,
                    )
                    if (item.priceUsd != null) {
                        Spacer(Modifier.width(Rahal.space.xs))
                        Text(
                            text = "≈ " + money(item.price),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.labelSmall,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }

            Box {
                RemoteImage(
                    // **والمصغّرةُ أوّلاً — لا الكاملة** (شكوى المالك ٢٠٢٦-٠٨-١٨):
                    // **قِيس على الخادم**: الكاملةُ ٢٦٣ ك.ب والمصغّرةُ ٤٠ — ستّةُ أضعافٍ
                    // ونصفٌ في بطاقةٍ عرضُها مئةٌ وبضعُ نقاط.
                    url = media(item.imageThumbUrl ?: item.imageUrl),
                    name = item.name,
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f),
                )

                // ══════════════════════════════════════════════════════
                // **والمغلقُ يُرى مغلقاً — لا شارةٌ تذوب في الصورة**
                // ══════════════════════════════════════════════════════
                //
                // (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «تصير الصورةُ خافتة ومكتوب بوضوح
                //  بالمنتصف».) **والتعتيمُ وحدَه يُقرأ ظلَّ تصميم، والنصُّ وحدَه
                // يضيع في الألوان — فاجتماعُهما هو ما يُرى من مترين.** **والخلصانُ
                // كالمغلق** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
                if (closed) {
                    Box(
                        Modifier
                            .matchParentSize()
                            // **وستّون بالمئة تُبقي الصورةَ مفهومةً وتُخمدها.**
                            .background(Color.Black.copy(alpha = 0.6f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = stringResource(
                                if (item.sourceClosed) R.string.shop_closed_now else R.string.mn_unavailable,
                            ),
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            textAlign = TextAlign.Center,
                            style = MaterialTheme.typography.labelLarge,
                            modifier = Modifier.padding(horizontal = Rahal.space.sm),
                        )
                    }
                }

                // **ونسبةُ الحسم فوق الصورة** — تُقرأ بلمحةٍ قبل أن يُقارَن الرقمان.
                item.discountPercent?.let {
                    Box(Modifier.padding(Rahal.space.xs)) {
                        Chip("-$it٪", Rahal.colors.accent)
                    }
                }
            }

            // ══════════════════════════════════════════════════════════
            // **الشريطُ الأسفل — اسمُ المتجر و«+» في نهايته**
            // ══════════════════════════════════════════════════════════
            //
            // **وارتفاعُه ثابتٌ ولو غاب الاسمُ أو الزرّ** — وإلّا تفاوتت بطاقاتُ
            // الصفّ الواحد. **و«+» لا يُعرض لمغلق** — من ضغطه فلم يقع شيءٌ ظنّ
            // التطبيقَ معطوبا.
            Row(
                Modifier
                    .fillMaxWidth()
                    .heightIn(min = Rahal.space.xl + Rahal.space.sm)
                    .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = item.merchantName.orEmpty(),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (!closed && onAdd != null) {
                    Spacer(Modifier.width(Rahal.space.xs))
                    Box(
                        Modifier
                            .clip(CircleShape)
                            .background(Rahal.colors.brand)
                            .clickable(onClick = onAdd)
                            .padding(5.dp),
                    ) {
                        Icon(
                            painter = painterResource(com.rahalgo.ui.R.drawable.ic_plus),
                            contentDescription = stringResource(R.string.shop_add),
                            tint = Rahal.colors.onBrand,
                            modifier = Modifier.size(15.dp),
                        )
                    }
                }
            }
        }
        Spacer(Modifier.height(5.dp))
        // ══════════════════════════════════════════════════════════════
        // **والاسمُ بالوسط تحت الإطار — ولا يُقصّ منه حرف** (ملاحظةُ المالك ٢٠٢٦-١٠-١٤)
        // ══════════════════════════════════════════════════════════════
        //
        // **الاختصارُ بثلاث نقاط يقصّ النصَّ على حدوده**، وتجوال ينزل تحت السطر
        // ويميل عن حافّته — فتختفي نقاطُ ما تحت السطر. **فلا قصّ**: سطران بالوسط.
        Text(
            text = item.name,
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Medium,
            maxLines = 2,
            overflow = TextOverflow.Visible,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )

        // **والمشطوبُ وحدَه ينزل سطرا** — ولا يقع إلّا في المحسوم.
        item.priceBefore?.let {
            Text(
                text = item.priceBeforeUsd?.let { u -> com.rahalgo.ui.dollars(u) } ?: money(it),
                color = Rahal.colors.inkMuted,
                textDecoration = TextDecoration.LineThrough,
                style = MaterialTheme.typography.labelSmall,
                maxLines = 1,
            )
        }
    }
}
