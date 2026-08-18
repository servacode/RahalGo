package com.rahalgo.customer.shop

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import com.rahalgo.customer.Backend
import com.rahalgo.customer.cart.Cart
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Section
import com.rahalgo.ui.Chip
import com.rahalgo.ui.CountBadge
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
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
 * # ولا أسماءَ متاجرَ في البطاقة
 *
 * **المنصّةُ سوقٌ يجلب منها** — وهو يشتري «من رحّال» لا «من مطعم فلان».
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٥، **ويفرضه المحرّكُ لا الشاشة.**)
 *
 * # وبطاقتان في الصفّ لا واحدة
 *
 * **الصورةُ هي ما يُشترى به** — وبطاقةٌ بعرض الشاشة تُظهر ثلاثةَ أصنافٍ
 * في شاشةٍ كاملة، **فيمرّ الإصبعُ طويلاً قبل أن يرى ما عنده.**
 */
@Composable
fun ShopScreen(
    vm: ShopViewModel,
    onOpenCart: () -> Unit,
    /** **الإعجابُ يحتاج حساباً** — والضيفُ يُساق إلى الدخول لا يُردّ. */
    onLike: (Item) -> Unit,
    liked: Set<String> = emptySet(),
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val media = { path: String? -> Backend.of(context).media(path) }

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
                        imageUrl = media(it.imageUrl),
                        target = it.target,
                    )
                },
                auto = vm.bannerAuto,
                everyMs = vm.bannerEveryMs,
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
            )
        }

        val hints = vm.sections.map { it.name }
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

        if (!vm.searching && vm.sections.isNotEmpty()) {
            SectionRail(vm.sections, vm.pick, media, vm::openSection)
        }

        when {
            vm.busy && vm.items.isEmpty() ->
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    RahalLoader()
                }

            vm.error.isNotEmpty() && vm.items.isEmpty() ->
                LoadState(loading = false, error = vm.error, onRetry = vm::load)

            vm.items.isEmpty() -> Empty(
                stringResource(
                    if (vm.searching) R.string.shop_no_results else R.string.shop_empty_section,
                ),
            )

            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(ShopCols),
                contentPadding = PaddingValues(12.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                items(vm.items, key = { it.id }) { item ->
                    ItemCard(
                        item = item,
                        media = media,
                        liked = item.id in liked,
                        onAdd = {
                            // **ومغلقٌ لا يُضاف** — ومن أضافه ثمّ رُدّ
                            // عند الدفع أضاع وقتَه.
                            if (item.available && !item.sourceClosed) Cart.add(item)
                        },
                        onLike = { onLike(item) },
                    )
                }
            }
        }

    }

    // ══════════════════════════════════════════════════════════════════
    // **والسلّةُ تطفو فوق كلّ شيء**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «لازم السلّة تظهر فقط بصفحة التسوّق،
    //  أيقونة عائمة مشان تكون واضحة».)
    //
    // **والشريطُ العلويُّ أخبارٌ تُقرأ بنظرة** — والسلّةُ **فعلٌ
    // يُضغط.** **وأيقونةٌ بين الجرس والمحفظة تُقرأ خبراً ثالثاً**
    // فتُمسح العينُ عليها ولا تُضغط.
    //
    // **ولا تمشي مع القائمة** — فتبقى ظاهرةً وهو ينزل في الأصناف.
    //
    // **وتظهر وهي فارغة** — (قرارُ المالك ٢٠٢٦-٠٨-١٥: «حتّى ولو فارغة،
    //  تلفت انتباه المستخدم»).
    //
    // **وكنتُ أخفيها**: «أيقونةٌ تقول ٠ تأخذ مكاناً ولا تقول شيئا» —
    // **وذاك حسابُ مساحةٍ لا حسابُ سوق.** **وزرٌّ يظهر فجأةً لا
    // يُتعلَّم**: من لم يره قطُّ لا يعرف أنّ للسوق سلّة، **وظهورُه
    // فارغاً هو ما يُعلّمه.**
    //
    // **وأيقونةٌ عاريةٌ بلا حاوية** — (قرارُ المالك نفسِه: «بدون دائرة
    //  أو صندوق مربّع أو كرت، بس السلّة أفضل»).
    //
    // **والحاويةُ كانت تقول «زرُّ نظام»** — والسلّةُ تقول ما هي بشكلها.
    //
    // **وأكبرُ ممّا كانت لا ضخمة** — (قرارُ المالك ٢٠٢٦-٠٨-١٥:
    //  «صغّره قليلا»). **وستّةٌ وتسعون كانت ربعَ عرض الشاشة.**
    //
    // ══════════════════════════════════════════════════════════════════
    // **ودائرةٌ برتقاليّةٌ تحملها السلّة على حافّتها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «خلّي الرقم بدائرةٍ برتقاليّة ولكن فوق
    //  السلّة، وكأنّ السلّةَ تحمله».)
    //
    // **ولا في جوفها**: رقمٌ أبيضُ داخل الرسم يُقرأ جزءاً من الرسم —
    // **ودائرةٌ بلونٍ ثانٍ تُقرأ خبراً على رسم.**
    //
    // **ولا في الزاوية**: شارةُ الزاوية رقمٌ ملصوقٌ بأيقونة. **وهذه
    // تجلس على حافّة السلّة مركزَ حِملها** — فتُقرأ **ما تحمله**.
    //
    // **وموضعُها ليس أعلى الأيقونة**: أعلاها فراغٌ فوق حافّة السلّة —
    // **والحافّةُ عند خُمس ارتفاعها**، ومركزُ السلّة أزيحُ يميناً عن
    // مركز الرسم لأنّ يدَها إلى اليسار.
    //
    // **والإزاحةُ مطلقةٌ لا تتبع الاتّجاه** (`absoluteOffset`): الرسمُ
    // لا يُقلب في العربيّة، **ولو تبعت الاتّجاهَ لطارت الدائرةُ إلى
    // الجهة الأخرى.**
    //
    // **والقصُّ على الأيقونة وحدَها** — (شكوى المالك: «الرقم يظهر
    // مقصوصاً»): `clip` كان فوق الجميع ليكون أثرُ الضغطة دائريّاً،
    // **فقصّ كلَّ ما خرج عن الدائرة.** **فصارت طبقتين**: خارجيّةٌ بلا
    // قصٍّ تحمل الدائرة، وداخليّةٌ مقصوصةٌ تحمل الرسمَ وأثرَ الضغطة.
    Box(
        Modifier
            .align(Alignment.BottomEnd)
            .padding(16.dp),
    ) {
        Box(
            Modifier
                .clip(CircleShape)
                .clickable(onClick = onOpenCart)
                .padding(6.dp),
        ) {
            Icon(
                painter = painterResource(com.rahalgo.ui.R.drawable.ic_cart),
                contentDescription = stringResource(R.string.cart_title),
                tint = Rahal.colors.brand,
                modifier = Modifier.size(CartSize),
            )
        }
        // **وتختفي وهي فارغة** — الأيقونةُ تبقى والعددُ يذهب: **«٠»
        // رقمٌ يُقرأ ويُشغل، والفراغُ يُقرأ «لا شيءَ بعد».**
        //
        // **وهي شارةُ الجرس نفسُها** (`CountBadge`) — **ونسختان
        // تفترقان يومَ يتبدّل شكلُ إحداهما.**
        CountBadge(
            count = Cart.count,
            color = Rahal.colors.accent,
            size = 22.dp,
            modifier = Modifier
                .align(Alignment.Center)
                .absoluteOffset(
                    // **مركزُ السلّة لا مركزُ الرسم.**
                    x = CartSize * 1.5f / 24f,
                    // **حافّتُها العليا** — تجلس عليها لا فوقها.
                    y = -CartSize * 8f / 24f,
                ),
        )
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
private val CartSize = 64.dp

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
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier
                    .width(72.dp)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onPick(s.id) },
            ) {
                RemoteImage(
                    url = media(s.imageThumbUrl ?: s.imageUrl),
                    name = s.name,
                    modifier = Modifier
                        .size(58.dp)
                        .clip(CircleShape),
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    text = s.name,
                    // **والمختارُ يُعرف بلونه وثقله** — لا بإطارٍ يزاحم
                    // الصورة.
                    color = if (on) Rahal.colors.brand else Rahal.colors.inkMuted,
                    fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
                    style = MaterialTheme.typography.labelMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

/**
 * **بطاقةُ صنف** — صورةٌ واسمٌ وسعر.
 *
 * **والمشطوبُ هو ما يجعل الخصمَ خصما**: «٧٧٬٢٥٠» وحدَه رقم، **و«١٠٣٬٠٠٠»
 * مشطوبةً فوقه توفيرٌ يُرى.**
 */
@Composable
private fun ItemCard(
    item: Item,
    media: (String?) -> String?,
    liked: Boolean,
    onAdd: () -> Unit,
    onLike: () -> Unit,
) {
    val closed = item.sourceClosed || !item.available
    Column(Modifier.clip(Rahal.shape.md)) {
        Box {
            RemoteImage(
                url = media(item.imageUrl ?: item.imageThumbUrl),
                name = item.name,
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .clip(Rahal.shape.md),
            )
            // **ونسبةُ الحسم فوق الصورة** — تُقرأ بلمحةٍ قبل أن يُقارَن
            // الرقمان.
            item.discountPercent?.let {
                Box(Modifier.padding(6.dp)) {
                    Chip("-$it٪", Rahal.colors.accent)
                }
            }
            // **والقلبُ في الزاوية العليا** — حيث تعوّدته الأصابع.
            Box(
                Modifier
                    .align(Alignment.TopStart)
                    .padding(4.dp)
                    .clip(CircleShape)
                    .background(Color.White.copy(alpha = 0.85f))
                    .clickable(onClick = onLike)
                    .padding(4.dp),
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
                    tint = Rahal.colors.accent,
                    modifier = Modifier.size(15.dp),
                )
            }

            // **و«+» في الزاوية السفلى** — بعيداً عن القلب: **زرّان
            // متلاصقان في زاويةٍ واحدةٍ يُضغط أحدُهما مكان الآخر.**
            //
            // **ولا يُعرض لمغلق** — ومن ضغطه فلم يقع شيءٌ ظنّ التطبيقَ
            // معطوبا.
            if (!closed) {
                Box(
                    Modifier
                        .align(Alignment.BottomEnd)
                        .padding(4.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.brand)
                        .clickable(onClick = onAdd)
                        .padding(5.dp),
                ) {
                    Icon(
                        painter = painterResource(com.rahalgo.ui.R.drawable.ic_plus),
                        contentDescription = stringResource(R.string.shop_add),
                        tint = Color.White,
                        modifier = Modifier.size(15.dp),
                    )
                }
            } else {
                // **ومغلقٌ يُقال على الصورة** — لا يُكتشف عند الضغط.
                Box(Modifier.align(Alignment.BottomStart).padding(6.dp)) {
                    Chip(
                        stringResource(
                            if (item.sourceClosed) R.string.shop_closed
                            else R.string.shop_unavailable,
                        ),
                        Rahal.colors.inkMuted,
                    )
                }
            }
        }
        Spacer(Modifier.height(5.dp))
        // ══════════════════════════════════════════════════════════════
        // **والاسمُ وسعرُه في سطرٍ واحد**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
        //
        // **وهما يُقرآن معاً دائما**: «ما هذا؟» و«بكم؟» سؤالٌ واحد،
        // **وسطران يجعلان العينَ تنزل بين كلّ صنفين.**
        //
        // **والسعرُ لا يُقصّ والاسمُ يُقصّ** — لأنّ «٢٠٠ ل.س» مقصوصةً
        // رقمٌ كاذب، **واسمٌ مقصوصٌ ما زال يُعرَف من صورته فوقه.**
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = item.name,
                style = MaterialTheme.typography.bodySmall,
                fontWeight = FontWeight.Medium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
            Spacer(Modifier.size(5.dp))
            Text(
                text = money(item.price),
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
            )
        }
        // **والمشطوبُ وحدَه ينزل سطرا** — **وهو ما يجعل الخصمَ خصما**:
        // «٢٠٠» وحدَها رقم، **و«٢٥٠» مشطوبةً فوقها توفيرٌ يُرى.**
        // **ولا يقع إلّا في المحسوم** — فسطرٌ ثانٍ في بضعة أصنافٍ لا
        // في كلّها.
        item.priceBefore?.let {
            Text(
                text = money(it),
                color = Rahal.colors.inkMuted,
                textDecoration = TextDecoration.LineThrough,
                style = MaterialTheme.typography.labelSmall,
                maxLines = 1,
            )
        }
    }
}