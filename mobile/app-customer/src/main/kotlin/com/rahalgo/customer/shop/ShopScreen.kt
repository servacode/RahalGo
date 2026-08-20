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
import androidx.compose.ui.draw.shadow
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
    /** **الإعجابُ يحتاج حساباً** — والضيفُ يُساق إلى الدخول لا يُردّ. */
    onLike: (Item) -> Unit,
    liked: Set<String> = emptySet(),
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val media = { path: String? -> Backend.of(context).media(path) }

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
                // ══════════════════════════════════════════════════════
                // **والمصغّرةُ أوّلاً — لا الكاملة**
                // ══════════════════════════════════════════════════════
                //
                // (شكوى المالك ٢٠٢٦-٠٨-١٨: «السلايدر وصور المنتجات
                //  تتأخّر بالظهور… لا يجب أن يلاحظ المستخدمُ هذه
                //  المشاكل».)
                //
                // **وكان الترتيبُ مقلوباً** — تُجلب الكاملةُ والمصغّرةُ
                // احتياطٌ لها. **وقِيس على الخادم**: الكاملةُ ٢٦٣ ك.ب
                // والمصغّرةُ ٤٠ — **ستّةُ أضعافٍ ونصفٌ في بطاقةٍ عرضُها
                // مئةٌ وثلاثٌ وعشرون نقطة.**
                //
                // **وستُّ بطاقاتٍ في الشاشة**: ميغا ونصفٌ بدل مئتين
                // وأربعين ك.ب — **على حزمةِ من يتصفّح في الرقّة.**
                url = media(item.imageThumbUrl ?: item.imageUrl),
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
            // ══════════════════════════════════════════════════════════
            // **والسعرُ على الصورة — لا يقاسم الاسمَ سطرَه**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٨: «السعر نحطّه فوق ع زاوية اليسار
            //  بشكلٍ أنيق».)
            //
            // **وقِيس سببُ القصّ**: ثلاثةُ أعمدةٍ تعطي البطاقةَ نحو ١٢٣
            // نقطة، **ويأخذ السعرُ منها ٥٣** — فيبقى للاسم سبعون، **أي
            // سبعةُ أحرفٍ عربيّة.** و«شاورما دجاج» أحدَ عشر.
            //
            // **فرفعُه يعيد العرضَ كلَّه للاسم** — من سبعة أحرفٍ إلى
            // ثلاثةَ عشر.
            //
            // # وقرصٌ داكنٌ لا نصٌّ عارٍ
            //
            // **صورُ الطعام زاهيةٌ ومزدحمة** — بطاطا صفراءُ وخبزٌ فاتح.
            // **ونصٌّ أبيضُ عليها يختفي في نصف الصور**، ولا يُعرف أنّه
            // كان هناك.
            //
            // # وفي الزاوية المقابلة للقلب
            //
            // **`TopEnd` هي اليسرى في العربيّة** — والقلبُ في `TopStart`
            // أي اليمنى. **وزرّان في زاويةٍ واحدةٍ يُضغط أحدُهما مكان
            // الآخر**، وقد كُتب هذا في «+» من قبل.
            Box(
                Modifier
                    .align(Alignment.TopEnd)
                    .padding(4.dp)
                    .clip(Rahal.shape.pill)
                    .background(Color.Black.copy(alpha = 0.55f))
                    .padding(horizontal = 7.dp, vertical = 3.dp),
            ) {
                Text(
                    text = money(item.price),
                    color = Color.White,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    style = MaterialTheme.typography.labelSmall,
                )
            }

            // ══════════════════════════════════════════════════════════
            // **والقلبُ يُرى — قرصٌ صلبٌ وحمرةٌ حين يُضغط**
            // ══════════════════════════════════════════════════════════
            //
            // (ملاحظةُ المالك ٢٠٢٦-٠٨-١٨: «أيقونةُ القلب ليست واضحةً
            //  بالشكل المطلوب، لو نقدر نخلّيها أوضحَ واحترافيّةً أكثر».)
            //
            // # ثلاثةُ أسبابٍ كانت تُخفيه
            //
            // **١ · قرصٌ شفّافٌ بخمسةٍ وثمانين بالمئة** — تظهر الصورةُ
            // خلفه فيذوب في زحمة الطعام.
            //
            // **٢ · وبرتقاليٌّ في الحالين** — لونُ العلامة نفسُه،
            // **فلا يُعرف المفضَّلُ من غيره إلّا بتدقيقٍ في امتلاء
            // الشكل.**
            //
            // **٣ · وخمسَ عشرةَ نقطةً في قرصٍ من ثلاثٍ وعشرين** — أصغرُ
            // من أن يُقصد بالإبهام.
            //
            // # وما صار
            //
            // **قرصٌ أبيضُ صلبٌ بظلٍّ خفيف** — يرتفع عن الصورة فيُقرأ
            // زرّاً لا رسماً عليها.
            //
            // **وأحمرُ حين يُضغط ورماديٌّ حين لا** — **والأحمرُ لغةُ
            // المفضّلة في كلّ تطبيق**، تُعرف بلا تعلّم. والبرتقاليُّ
            // لونُ علامتنا يُقرأ زينةً هنا لا حالا.
            //
            // **وثمانيَ عشرةَ نقطةً في قرصٍ من ثلاثين** — يُقصد بالإبهام.
            Box(
                Modifier
                    .align(Alignment.TopStart)
                    .padding(5.dp)
                    .shadow(3.dp, CircleShape)
                    .clip(CircleShape)
                    .background(Color.White)
                    .clickable(onClick = onLike)
                    .padding(6.dp),
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
        // **والاسمُ وحدَه في سطره — والسعرُ صعد إلى الصورة**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٨.)
        //
        // **وكانا يقتسمان السطرَ منذ ٢٠٢٦-٠٨-١٥** — «ما هذا؟» و«بكم؟»
        // سؤالٌ واحد. **وبقيا يُقرآن معاً**: السعرُ فوق الصورة والاسمُ
        // تحتها، **والعينُ تمسحهما في نظرةٍ واحدة.**
        //
        // **وسطرٌ واحدٌ بعد** — يُجرَّب أوّلاً، فإن بقي القصُّ صار
        // سطرين. (وهو ما اتُّفق عليه: نُجرّب ثمّ نقرّر.)
        Text(
            text = item.name,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Medium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth(),
        )

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