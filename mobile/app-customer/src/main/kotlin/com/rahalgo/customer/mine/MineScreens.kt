package com.rahalgo.customer.mine

import android.content.Intent
import androidx.compose.foundation.clickable
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.Backend
import com.rahalgo.customer.CustomerItems
import com.rahalgo.customer.shop.ShopCols
import com.rahalgo.customer.R
import com.rahalgo.customer.cart.Cart
import com.rahalgo.shared.model.Item
import com.rahalgo.ui.Flash
import com.rahalgo.design.Rahal
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RemoteImage
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenPad
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.money
import com.rahalgo.ui.TicketMessage
import com.rahalgo.ui.TicketRow
import com.rahalgo.ui.TicketThreadScreen
import com.rahalgo.ui.TicketsScreen
import androidx.activity.compose.BackHandler
import com.rahalgo.ui.RahalButton

/**
 * **بابُ أقسام «ما يخصّني»** — يجلب ثمّ يوجّه.
 *
 * **والجلبُ عند الفتح لا عند الإقلاع** — من فتح مفضّلتَه لا تُنادى
 * شكاواه.
 */
@Composable
fun MineScreen(
    vm: MineViewModel,
    key: String,
    address: com.rahalgo.shared.model.Address? = null,
    /** **يفتح صفحةَ متجرٍ في السوق** — من اسمه على بطاقة المفضّلة أو في نافذة الخيارات. */
    onOpenStore: (String) -> Unit = {},
    /** **«تصفّح السوق» من الفراغ** — ينقل لتبويب التسوّق. */
    onBrowse: () -> Unit = {},
) {
    LaunchedEffect(key) { vm.open(key) }
    when (key) {
        CustomerItems.FAVORITES -> Favorites(vm, address, onOpenStore, onBrowse)
        CustomerItems.OFFERS -> Offers(vm, address, onOpenStore)
        CustomerItems.INVITE -> Invite(vm)
        CustomerItems.TICKETS -> Tickets(vm)
    }
}

/**
 * **المفضّلة — وشبكتُها شبكةُ السوق نفسُها.**
 *
 * **وقائمةٌ بمصغَّرةٍ صغيرةٍ تجعل الصنفَ سطرَ جدولٍ في شاشةٍ وسلعةً في
 * أخرى** — **فما يُتعلَّم في شاشةٍ لا يُعرف في الباقي.**
 */
@Composable
private fun Favorites(
    vm: MineViewModel,
    address: com.rahalgo.shared.model.Address?,
    onOpenStore: (String) -> Unit,
    onBrowse: () -> Unit,
) {
    val list = vm.favorites
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.FAVORITES, force = true) }
        return
    }
    val context = LocalContext.current
    val backend = Backend.of(context)

    // ══════════════════════════════════════════════════════════════════
    // **و«+» في المفضّلة بشروط السوق نفسِها** (تدقيقُ ٢٠٢٦-١٠-٢٧)
    // ══════════════════════════════════════════════════════════════════
    //
    // **من حفظ صنفاً ليعود إليه بسرعة كان لا يضيفه منها** — يبحث عنه في السوق.
    // **والحارسُ واحد**: بوّابةُ الخدمة (`rememberAddBlocked`)، والخياراتُ قبل الدخول،
    // **وتنبيهُ المتجر الثاني** (`SourceNotice`) — كما في السوق والعروض.
    val blocked = com.rahalgo.customer.rememberAddBlocked(address)
    val blockedText = stringResource(R.string.offer_add_blocked)
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    com.rahalgo.customer.cart.SourceNoticeDialog()
    var picking by remember { mutableStateOf<Item?>(null) }
    val addOne: (Item, Int, List<com.rahalgo.shared.model.ModifierOption>) -> Unit = { it, n, chosen ->
        if (blocked) {
            Flash.fail(blockedText)
        } else {
            val inc = Cart.add(it, qty = n, options = chosen)
            com.rahalgo.customer.cart.SourceNotice.afterAdd(scope, vm.customerApi, address, inc)
            Flash.ok(
                context.getString(
                    if (inc) com.rahalgo.ui.R.string.cart_qty_updated
                    else com.rahalgo.ui.R.string.cart_item_added,
                ),
            )
        }
    }
    picking?.let { target ->
        com.rahalgo.customer.shop.ItemOptionsSheet(
            item = target,
            api = vm.customerApi,
            onAdd = { chosen, n -> addOne(target, n, chosen); picking = null },
            onClose = { picking = null },
            onStore = com.rahalgo.customer.shop.StoreLink.target(target, null)
                ?.let { id -> { picking = null; onOpenStore(id) } },
            media = backend::media,
        )
    }
    // ══════════════════════════════════════════════════════════════════
    // **وتملأ الشاشةَ لتبدأ من أعلاها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٥: «العنوان يجب أن يكون أعلى الصفحة وليس
    //  بمنتصفها، والأعلى فارغٌ — هناك بادينغ كبير».)
    //
    // **ولا حشوةَ هناك أصلا**: الحاويةُ في `MainActivity` تُوسّط ما لا
    // يملأ طولَها (`Alignment.Center`)، **وهذه كانت تملأ العرضَ
    // وحدَه** — فهبطت إلى الوسط بقدر ما نقص من طولها.
    //
    // **وباقي الشاشات تملأ الطول** (`Screen`) فلم يظهر فيها — **وهو
    // عطبٌ يظهر في شاشةٍ واحدةٍ وسببُه في شاشةٍ أخرى.**
    //
    // **وحشوتُها حشوةُ الشاشات نفسُها** (`ScreenPad`) — لا رقمٌ
    // مرتجَل: **وكان عنوانُها يلتصق بحافّة الشاشة.**
    Column(
        Modifier
            .fillMaxSize()
            .padding(ScreenPad),
    ) {
        // **ولا سطرَ شرحٍ تحته** — **واسمُه يقول ما فيه.**
        ScreenTitle(stringResource(R.string.menu_favorites_title))
        if (list.isEmpty()) {
            // **والفراغُ يقول الخطوةَ التالية** (شرطُ المالك ٢٠٢٦-٠٩-١٣).
            Empty(
                text = stringResource(R.string.fav_none),
                hint = stringResource(R.string.fav_none_hint),
                actionLabel = stringResource(R.string.act_browse_market),
                onAction = onBrowse,
            )
            return@Column
        }
        // **وعددُ الأعمدة عددُ السوق** — لا رقمٌ ثانٍ يفترق عنه.
        LazyVerticalGrid(
            columns = GridCells.Fixed(ShopCols),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items(list, key = { it.id }) { item ->
                // ══════════════════════════════════════════════════════
                // **وبطاقتُها بطاقةُ السوق نفسُها** (قرارُ المالك ٢٠٢٦-١٠-١٥)
                // ══════════════════════════════════════════════════════
                //
                // **إطارٌ بشريطين واسمُ المتجر** — ونسختان من البطاقة تفترقان
                // يوماً كما افترقتا في السعر. **والقلبُ يُرفع من شاشته هو**
                // — ومن أراد أن يحذف لا يبحث عن الصنف في السوق.
                // **و«+» هنا الآن** (تدقيقُ ٢٠٢٦-١٠-٢٧) — بحارس السوق نفسِه أعلاه.
                // **والمصغَّرُ أوّلاً** — تطلبه البطاقةُ نفسُها (قِيس ٢٠٢٦-٠٨-٢٤).
                com.rahalgo.customer.shop.ItemCard(
                    item = item,
                    media = backend::media,
                    liked = true,
                    onAdd = {
                        if (item.hasOptions && !blocked) picking = item
                        else addOne(item, 1, emptyList())
                    },
                    onLike = { vm.toggleFavorite(item.id) },
                    // **واسمُ المتجر يفتح صفحتَه** كما في السوق (٢٠٢٦-١٠-١٨).
                    onStore = com.rahalgo.customer.shop.StoreLink.target(item, null)
                        ?.let { id -> { onOpenStore(id) } },
                )
            }
        }
    }
}

/**
 * **العروض — والفرقُ عن كود الخصم.**
 *
 * **الكودُ يُكتب والعرضُ يُرى** — ومن لم يسمع بالكود لا يستفيد منه
 * **ولا يعلم أنّه فاته.**
 */
@Composable
private fun Offers(
    vm: MineViewModel,
    address: com.rahalgo.shared.model.Address? = null,
    onOpenStore: (String) -> Unit = {},
) {
    val list = vm.offers
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.OFFERS, force = true) }
        return
    }
    val context = LocalContext.current
    // **والإشارةُ تُمحى بمغادرة الشاشة** — **ومن عاد إلى العروض بعد
    // يومٍ لا يُشار له إلى عرضِ خبرٍ قديم** (`DLINK-09`).
    androidx.compose.runtime.DisposableEffect(Unit) {
        onDispose { vm.clearFocus() }
    }
    // **ويُقرأ النصُّ في التركيب لا في المستمع** — `stringResource`
    // دالّةُ تركيبٍ ولا تُنادى داخل `onClick`.
    val addedText = stringResource(R.string.shop_added)
    // **وبابُ العروض يمرّ ببوّابةِ الخدمة نفسِها** (`CAF-12`/`CUST-ENG-005`) —
    // **ولا يُضاف عرضٌ إلى نقطةٍ لا نصلها** ثمّ يُردّ عند الإرسال.
    val blocked = com.rahalgo.customer.rememberAddBlocked(address)
    val blockedText = stringResource(R.string.offer_add_blocked)

    // **وتنبيهُ المتجر الثاني بعد كلّ إضافةٍ من هنا أيضاً** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت
    // العروضُ تضيف صامتةً فتُملأ السلّةُ من متاجرَ فوق السقف **ويُرَدّ الطلبُ برسالةٍ عامّة.**
    val sourceScope = androidx.compose.runtime.rememberCoroutineScope()
    com.rahalgo.customer.cart.SourceNoticeDialog()

    // **والصنفُ ذو الخياراتِ يُسأل قبل أن يدخل** — انظر `ItemOptionsSheet`.
    var picking by remember { mutableStateOf<Item?>(null) }
    picking?.let { target ->
        com.rahalgo.customer.shop.ItemOptionsSheet(
            item = target,
            api = vm.customerApi,
            onAdd = { chosen, n ->
                if (blocked) {
                    Flash.fail(blockedText)
                } else {
                    val inc = Cart.add(target, qty = n, options = chosen)
                    com.rahalgo.customer.cart.SourceNotice.afterAdd(sourceScope, vm.customerApi, address, inc)
                    Flash.ok(addedText)
                }
                picking = null
            },
            onClose = { picking = null },
            onStore = com.rahalgo.customer.shop.StoreLink.target(target, null)
                ?.let { id -> { picking = null; onOpenStore(id) } },
            media = Backend.of(context)::media,
        )
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_offers_title),
            stringResource(R.string.soon_offers),
        )
        // ══════════════════════════════════════════════════════════════
        // **والعرضُ الذي جاء به الخبرُ يُرفَع إلى أوّلها** (`DLINK-03`)
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا شاشةَ عرضٍ مفردة** — **فمن ساقه الخبرُ إلى قائمةٍ فيها
        // عشرون عرضاً لا يعرف أيَّها قُصد.**
        val focus = vm.focusOffer
        if (vm.focusGone) {
            // **وذهب قبل أن يُفتَح** — **فيُقال، ولا يُعرَض سعرٌ مضى
            // على أنّه سارٍ** (`DLINK-04`).
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(R.string.off_gone),
                color = Rahal.colors.inkMuted,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        // ══════════════════════════════════════════════════════════════
        // **وعرضُ التوصيل المجاني أوّلَ العروض** (تدقيقُ ٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **كانت الشاشةُ تقول «لا عروض سارية» ولزبونها توصيلةٌ مجانيّة** — فيقرأ أنّ
        // لا شيء له. **والأهليّةُ من الخادم** (`FreeOffer`) — لا تُخمَّن.
        val scopeFree = androidx.compose.runtime.rememberCoroutineScope()
        LaunchedEffect(Unit) { com.rahalgo.customer.FreeOffer.refresh(scopeFree) }
        val free = com.rahalgo.customer.FreeOffer.state
        if (free.any) {
            Spacer(Modifier.height(10.dp))
            Card(tone = Rahal.colors.success) {
                Text(
                    text = stringResource(R.string.free_offer_title),
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                )
                val kinds = listOfNotNull(
                    stringResource(R.string.free_kind_standard).takeIf { free.standard },
                    stringResource(R.string.free_kind_purchase).takeIf { free.purchase },
                    stringResource(R.string.free_kind_amanah).takeIf { free.amanah },
                ).joinToString("، ")
                Spacer(Modifier.height(4.dp))
                Text(
                    text = com.rahalgo.customer.freeOfferText(
                        free,
                        stringResource(R.string.free_offer_kinds, kinds),
                    ),
                    color = Rahal.colors.ink,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
        if (list.isEmpty()) {
            if (!free.any) {
                Empty(
                    text = stringResource(R.string.off_none),
                    hint = stringResource(R.string.off_none_hint),
                )
            }
            return@Screen
        }
        val ordered =
            if (focus == null) list else list.sortedByDescending { it.id == focus }
        ordered.forEach { o ->
            if (focus != null && o.id == focus) {
                Spacer(Modifier.height(10.dp))
                Text(
                    text = stringResource(R.string.off_from_notice),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelSmall,
                )
            }
            Spacer(Modifier.height(10.dp))
            Card {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    RemoteImage(
                        url = Backend.of(context).media(o.itemImageUrl ?: o.imageUrl),
                        name = o.title.ifEmpty { o.itemName },
                        modifier = Modifier
                            .size(64.dp)
                            .clip(Rahal.shape.sm),
                    )
                    Spacer(Modifier.size(10.dp))
                    Column(Modifier.fillMaxWidth()) {
                        Text(
                            text = o.title.ifEmpty { o.itemName },
                            color = Rahal.colors.ink,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        // **واسمُ المتجر تحت اسم الصنف** (قرارُ المالك ٢٠٢٦-١٠-١٥) —
                        // كما في بطاقة السوق. **وفارغُه من خادمٍ أقدم لا سطرَ له.**
                        o.merchantName?.takeIf { it.isNotEmpty() }?.let {
                            Text(
                                text = it,
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.labelSmall,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        if (o.body.isNotEmpty()) {
                            Text(
                                text = o.body,
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        Spacer(Modifier.height(4.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            if (o.priceAfter > 0) {
                                Text(
                                    text = money(o.priceAfter),
                                    color = Rahal.colors.brand,
                                    fontWeight = FontWeight.Bold,
                                )
                            }
                            if (o.priceBefore > o.priceAfter) {
                                Spacer(Modifier.size(6.dp))
                                Text(
                                    text = money(o.priceBefore),
                                    color = Rahal.colors.inkMuted,
                                    textDecoration = TextDecoration.LineThrough,
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                            o.discountPercent?.let {
                                Spacer(Modifier.size(6.dp))
                                Chip("-$it٪", Rahal.colors.accent)
                            }
                        }

                        // ══════════════════════════════════════════════
                        // **والعرضُ يُضاف من مكانه**
                        // ══════════════════════════════════════════════
                        //
                        // (قرارُ المالك ٢٠٢٦-٠٨-١٩: «بصفحة العروض ما في
                        //  زرّ إضافة إلى السلّة… مو معقول يطلع من صفحة
                        //  العروض يروح يدوّر على العرض بالقوائم».)
                        //
                        // **وصفحةُ عرضٍ لا يُشترى منها إعلانٌ لا سوق**:
                        // من رأى الحسمَ ثمّ طُلب منه أن يبحث عن الصنف
                        // في القوائم **يفقد الحسمَ في الطريق.**
                        //
                        // # ولا زرَّ لعرضٍ بلا صنف
                        //
                        // **بعضُ العروض إعلانٌ عامّ** (`menuItemId`
                        // فارغ) — **وزرٌّ يُضيف لا شيءَ يُقرأ عطبا.**
                        o.menuItemId?.takeIf { it.isNotEmpty() }?.let { itemId ->
                            Spacer(Modifier.height(8.dp))
                            RahalButton(
                                onClick = {
                                    // **والسعرُ سعرُ العرض** — وهو ما
                                    // رآه. **والمحرّكُ يُعيد الحسابَ
                                    // عند الإرسال** فلا يُدسّ رقم.
                                    // **وما له خياراتٌ يُسأل قبل أن
                                    // يدخل** — **والمجموعةُ الإلزاميّةُ
                                    // تُسقط الطلبَ كلَّه** إن دخل بلا
                                    // اختيار، ولا يُقال أيُّ صنفٍ سبّبه.
                                    val picked = Item(
                                        id = itemId,
                                        name = o.itemName.ifEmpty { o.title },
                                        price = o.priceAfter,
                                        imageUrl = o.itemImageUrl ?: o.imageUrl,
                                        priceBefore = o.priceBefore
                                            .takeIf { it > o.priceAfter },
                                        discountPercent = o.discountPercent,
                                        hasOptions = o.hasOptions,
                                    )
                                    if (picked.hasOptions) {
                                        picking = picked
                                    } else if (blocked) {
                                        // **بوّابةُ الخدمة نفسُها** (CAF-12) —
                                        // **لا يُضاف عرضٌ إلى نقطةٍ لا نصلها.**
                                        Flash.fail(blockedText)
                                    } else {
                                        val inc = Cart.add(picked)
                                        com.rahalgo.customer.cart.SourceNotice.afterAdd(
                                            sourceScope, vm.customerApi, address, inc,
                                        )
                                        // **ورسالةٌ تقول إنّه وقع** —
                                        // **وضغطةٌ بلا أثرٍ تُقرأ عطبا**،
                                        // والبطاقةُ لا تتغيّر بعدها.
                                        Flash.ok(addedText)
                                    }
                                },
                                compact = true,
                            ) { Text(stringResource(R.string.shop_add)) }
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **ادعُ صديقاً — والرقمُ قبل الفعل.**
 *
 * **ووعدٌ مبهمٌ لا يُحرّك أحدا**: «ادعُ أصدقاءك» لا تعني شيئاً، **و«ادعُ
 * صديقاً واربح ٥٬٠٠٠» تعني.**
 *
 * **والشرطُ يُقال من الإعداد لا من جملةٍ ثابتة** — كُتب في الويب «تُصرف
 * عند أوّل طلب» **والإعدادُ يقول عند التسجيل**، فقرأ الزبونُ شرطاً ووقع
 * غيرُه.
 */
@Composable
private fun Invite(vm: MineViewModel) {
    val r = vm.referral
    if (r == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.INVITE, force = true) }
        return
    }
    val context = LocalContext.current
    Screen {
        ScreenTitle(
            stringResource(R.string.menu_invite_title),
            stringResource(R.string.soon_invite),
        )
        Spacer(Modifier.height(12.dp))
        Card {
            Text(
                text = stringResource(R.string.inv_earn, money(r.nextReward)),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(6.dp))
            Text(
                text = stringResource(
                    if (r.rewardOn == "signup") R.string.inv_on_signup else R.string.inv_on_order,
                ),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **الرمزُ بخطٍّ كبيرٍ في مربّعٍ وجنبه «انسخ»** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **كان سطراً صغيراً بلا نسخ** — ومن أراد أن يقوله لصديقٍ على الهاتف يقرؤه
        // حرفاً حرفاً. **والأرقامُ الثلاثةُ مربّعاتٌ كاللوحة** (`StatRow`).
        Spacer(Modifier.height(10.dp))
        val copiedText = stringResource(R.string.inv_copied)
        val clipboard = androidx.compose.ui.platform.LocalClipboardManager.current
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .border(
                    androidx.compose.foundation.BorderStroke(1.5.dp, Rahal.colors.brand.copy(alpha = 0.5f)),
                    Rahal.shape.md,
                )
                .padding(horizontal = 14.dp, vertical = 10.dp),
        ) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = stringResource(R.string.inv_code),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.labelMedium,
                )
                Text(
                    text = r.code,
                    color = Rahal.colors.ink,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineSmall,
                    maxLines = 1,
                )
            }
            com.rahalgo.ui.RahalOutlineButton(
                onClick = {
                    clipboard.setText(androidx.compose.ui.text.AnnotatedString(r.code))
                    Flash.ok(copiedText)
                },
            ) { Text(stringResource(R.string.inv_copy)) }
        }
        Spacer(Modifier.height(10.dp))
        com.rahalgo.ui.StatRow {
            com.rahalgo.ui.StatBox(stringResource(R.string.inv_invited), r.invited.toString(), Modifier.weight(1f))
            com.rahalgo.ui.StatBox(stringResource(R.string.inv_rewarded), r.rewarded.toString(), Modifier.weight(1f))
            com.rahalgo.ui.StatBox(
                stringResource(R.string.inv_earned),
                money(r.earned),
                Modifier.weight(1f),
                color = Rahal.colors.success,
            )
        }

        Spacer(Modifier.height(12.dp))
        // **والمشاركةُ جملةٌ فيها الرمزُ والرابط** — رابطٌ وحدَه على واتساب لا يُعرف ما هو.
        val shareText = stringResource(R.string.inv_share_text, r.code, r.link)
        RahalButton(
            onClick = {
                // **ويُشارَك بما يعرفه هاتفُه** — واتساب أو رسالة:
                // **ونسخُ رابطٍ إلى الحافظة يُوجب عليه أن يفتح تطبيقاً
                // ويلصق**، وأكثرُهم لا يفعل.
                runCatching {
                    context.startActivity(
                        Intent.createChooser(
                            Intent(Intent.ACTION_SEND).apply {
                                type = "text/plain"
                                putExtra(Intent.EXTRA_TEXT, shareText)
                            },
                            null,
                        ),
                    )
                }
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.act_share_link)) }
        Spacer(Modifier.height(24.dp))
    }
}

/**
 * **شكاواي وأين وصلت.**
 *
 * **ومن اشتكى ولم يرَ جواباً ظنّ أنّ شكواه ضاعت** — فيشتكي ثانيةً، أو
 * يتّصل، **أو يسكت ويذهب.** والسكوتُ أسوأ: **نخسر الزبونَ ولا نعرف
 * لماذا.**
 */
@Composable
private fun Tickets(vm: MineViewModel) {
    // ══════════════════════════════════════════════════════════════════
    // **شكوًى مفتوحةٌ ⇒ خيطُها** (`SUP-013`/`014`) — والرجوعُ يعود للقائمة.
    // ══════════════════════════════════════════════════════════════════
    //
    // **وزرُّ النظام يُغلق الخيطَ لا القسمَ كلَّه** — يسبق حارسَ الطبقة الأعلى.
    if (vm.openTicketId != null) {
        BackHandler { vm.closeTicket() }
        TicketDetail(vm)
        return
    }

    val list = vm.tickets
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.TICKETS, force = true) }
        return
    }
    // ══════════════════════════════════════════════════════════════════
    // **والشاشةُ من `:ui`** — (قرارُ المالك ٢٠٢٦-٠٨-٣١: مركزيّةٌ للثلاثة).
    //
    // **وتبويبان الآن** (Batch 4، C2): «شكاوى قدّمتها» و«بلاغات ضدّي». كان
    // يُفترَض أنّ الزبونَ لا يُشتكى عليه — **والسائقُ يُبلّغ عنه**، فبلاغُه كان
    // يتسلّل إلى قائمة شكاواه (تسريب). فصار له تبويبٌ مُقنَّعٌ (بلا اسمِ مُبلِّغ).
    TicketsScreen(
        title = stringResource(R.string.menu_tickets_title),
        hint = stringResource(R.string.soon_tickets),
        mine = list.map { t ->
            TicketRow(
                id = t.id,
                key = ticketKey(t),
                title = t.subject.ifEmpty { com.rahalgo.ui.TicketReasons.text(t.reason) },
                status = t.status,
                orderNumber = t.orderNumber?.toString().orEmpty(),
                resolution = t.resolution,
            )
        },
        mineEmpty = stringResource(R.string.tkt_none),
        // **بلاغاتٌ ضدّي — مُقنَّعة**: لا معرّفَ يُفتَح (خيطٌ مخفيّ) ولا اسمَ
        // مُبلِّغ؛ موضوعٌ وحالٌ ورقمُ طلبٍ فقط.
        againstMe = vm.againstMe.orEmpty().map { c ->
            TicketRow(
                key = "#" + c.number,
                title = c.subject,
                status = c.status,
                orderNumber = c.orderNumber?.toString().orEmpty(),
            )
        },
        againstMeEmpty = stringResource(R.string.tkt_against_none),
        onOpen = { row -> vm.openTicket(row.id) },
    )
}

/**
 * **خيطُ شكوًى بعينها** — تحميلُه وفشلُه كسائر الأقسام، ثمّ عرضُه.
 */
@Composable
private fun TicketDetail(vm: MineViewModel) {
    val t = vm.ticketDetail
    val id = vm.openTicketId
    if (t == null) {
        // **تحميلٌ أو فشلٌ بإعادة** — والإعادةُ تُعيد جلبَ الخيط نفسِه.
        LoadState(vm.detailBusy, vm.detailError) { id?.let { vm.openTicket(it, force = true) } }
        return
    }
    TicketThreadScreen(
        subject = t.subject,
        status = t.status,
        createdAt = t.createdAt,
        messages = t.replies.map { r ->
            TicketMessage(body = r.body, mine = r.mine, at = r.createdAt)
        },
        emptyReplies = stringResource(R.string.tik_no_replies),
        canReply = com.rahalgo.ui.ticketCanReply(t.status),
        draft = vm.replyDraft,
        onDraftChange = vm::editReplyDraft,
        onSend = vm::sendReply,
        sending = vm.replying,
        sendError = vm.detailError,
        onBack = vm::closeTicket,
    )
}


/**
 * **رقمُ الشكوى في القائمة** — «#511» كما يعرفه المكتبُ ويسأل به.
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: عُرضت «#08d6f3» — ستّةُ أحرفٍ من معرّفها، **لأنّ
 *  النموذجَ كان يقرأ `order_code` الذي لا يرسله المحرّك.**) **ورقمٌ لا يعرفه
 * المكتبُ لا يُسأل به عن شكوى.**
 *
 * **والمعرّفُ احتياطٌ لردٍّ قديمٍ بلا رقم** — لا يُترك السطرُ بلا مفتاح.
 */
internal fun ticketKey(t: com.rahalgo.shared.customer.Ticket): String =
    "#" + if (t.number > 0) t.number.toString() else t.id.take(6)
