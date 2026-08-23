package com.rahalgo.customer.mine

import android.content.Intent
import androidx.compose.foundation.clickable
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
import com.rahalgo.ui.ticketStatusColor
import com.rahalgo.ui.ticketStatusText
import com.rahalgo.ui.RahalButton

/**
 * **بابُ أقسام «ما يخصّني»** — يجلب ثمّ يوجّه.
 *
 * **والجلبُ عند الفتح لا عند الإقلاع** — من فتح مفضّلتَه لا تُنادى
 * شكاواه.
 */
@Composable
fun MineScreen(vm: MineViewModel, key: String) {
    LaunchedEffect(key) { vm.open(key) }
    when (key) {
        CustomerItems.FAVORITES -> Favorites(vm)
        CustomerItems.OFFERS -> Offers(vm)
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
private fun Favorites(vm: MineViewModel) {
    val list = vm.favorites
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.FAVORITES, force = true) }
        return
    }
    val context = LocalContext.current
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
            Empty(stringResource(R.string.fav_none))
            return@Column
        }
        // **وعددُ الأعمدة عددُ السوق** — لا رقمٌ ثانٍ يفترق عنه.
        LazyVerticalGrid(
            columns = GridCells.Fixed(ShopCols),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items(list, key = { it.id }) { item ->
                Column {
                    Box {
                        RemoteImage(
                            url = Backend.of(context).media(item.imageUrl ?: item.imageThumbUrl),
                            name = item.name,
                            modifier = Modifier
                                .fillMaxWidth()
                                .aspectRatio(1f)
                                .clip(Rahal.shape.md),
                        )
                        // **والقلبُ يُرفع من شاشته هو** — ومن أراد أن
                        // يحذف لا يبحث عن الصنف في السوق.
                        Box(
                            Modifier
                                .align(Alignment.TopEnd)
                                .padding(6.dp)
                                .clickable { vm.toggleFavorite(item.id) },
                        ) {
                            Text("♥", color = Rahal.colors.accent)
                        }
                    }
                    Spacer(Modifier.height(6.dp))
                    Text(
                        text = item.name,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = money(item.price),
                        color = Rahal.colors.brand,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
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
private fun Offers(vm: MineViewModel) {
    val list = vm.offers
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.OFFERS, force = true) }
        return
    }
    val context = LocalContext.current
    // **ويُقرأ النصُّ في التركيب لا في المستمع** — `stringResource`
    // دالّةُ تركيبٍ ولا تُنادى داخل `onClick`.
    val addedText = stringResource(R.string.shop_added)

    // **والصنفُ ذو الخياراتِ يُسأل قبل أن يدخل** — انظر `ItemOptionsSheet`.
    var picking by remember { mutableStateOf<Item?>(null) }
    picking?.let { target ->
        com.rahalgo.customer.shop.ItemOptionsSheet(
            item = target,
            api = vm.customerApi,
            onAdd = { chosen ->
                Cart.add(target, options = chosen)
                picking = null
                Flash.ok(addedText)
            },
            onClose = { picking = null },
        )
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_offers_title),
            stringResource(R.string.soon_offers),
        )
        if (list.isEmpty()) {
            Empty(stringResource(R.string.off_none))
            return@Screen
        }
        list.forEach { o ->
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
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.bodyMedium,
                        )
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
                                    } else {
                                        Cart.add(picked)
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

        Spacer(Modifier.height(10.dp))
        Card {
            KeyValue(stringResource(R.string.inv_code), r.code)
            Spacer(Modifier.height(8.dp))
            HorizontalDivider()
            Spacer(Modifier.height(8.dp))
            KeyValue(stringResource(R.string.inv_invited), r.invited.toString())
            KeyValue(stringResource(R.string.inv_rewarded), r.rewarded.toString())
            KeyValue(stringResource(R.string.inv_earned), money(r.earned))
        }

        Spacer(Modifier.height(12.dp))
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
                                putExtra(Intent.EXTRA_TEXT, r.link)
                            },
                            null,
                        ),
                    )
                }
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.inv_share)) }
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
    val list = vm.tickets
    if (list == null) {
        LoadState(vm.busy, vm.error) { vm.open(CustomerItems.TICKETS, force = true) }
        return
    }
    Screen {
        ScreenTitle(
            stringResource(R.string.menu_tickets_title),
            stringResource(R.string.soon_tickets),
        )
        if (list.isEmpty()) {
            Empty(stringResource(R.string.tkt_none))
            return@Screen
        }
        list.forEach { t ->
            Spacer(Modifier.height(8.dp))
            Card {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = t.subject.ifEmpty { t.reason },
                        fontWeight = FontWeight.Medium,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Chip(ticketStatusText(t.status), ticketStatusColor(t.status))
                }
                if (t.orderCode.isNotEmpty()) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = "#" + t.orderCode,
                        color = Rahal.colors.inkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                // **وجوابُ المكتب يُعرض** — وشكوى بلا جوابٍ ظاهرٍ تُقرأ
                // مهملة.
                if (t.resolution.isNotEmpty()) {
                    Spacer(Modifier.height(6.dp))
                    Text(t.resolution, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

