package com.rahalgo.driver.orders

import androidx.compose.foundation.background
import com.rahalgo.ui.CountdownButton
import com.rahalgo.design.Rahal
import com.rahalgo.ui.Empty
import com.rahalgo.ui.minutes
import com.rahalgo.ui.dist
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.grouped
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.RahalLoader

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطلبات — ما عُرض عليه وما في يده**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الخامسة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # ولماذا في شاشة واحدة
 *
 * **السائق يوازن بينهما في اللحظة نفسها**: أيأخذ عرضا جديدا وفي يده
 * اثنان؟ **ومن فصلهما شاشتين** جعله يبدّل ليقرّر.
 *
 * # والعرض أوّلا
 *
 * **العرض يفوت** — يأخذه غيره أو تنقضي مهلته. **وما في يده لا يفوت.**
 * فالذي يزول يُعرض قبل الذي يبقى.
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** —
 * والنداءات في `OrdersViewModel`.
 */
@Composable
fun OrdersScreen(state: OrdersState, actions: OrdersActions) {
    if (state.loading) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            RahalLoader()
        }
        return
    }

    LazyColumn(
        Modifier
            .fillMaxSize()
            // **وتحت شريط الحالة لا خلفه** — الشاشة تُرسم من حافة إلى
            // حافة. **وشريط النظام السفليّ يحسبه `Scaffold` مرّة**،
            // فلو أُضيف هنا `safeDrawing` لحُسب مرّتين وارتفع المحتوى.
            .statusBarsPadding(),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = Rahal.space.gutter,
            end = Rahal.space.gutter,
            top = Rahal.space.sm,
            bottom = Rahal.space.tail,
        ),
    ) {
        // **وخطأُ الفعل يُقرأ كخطأ القراءة** — كلاهما يُقال في رأس
        // الشاشة، **ولا يُمحى أحدُهما بنجاح الآخر.**
        val shown = state.error.ifEmpty { state.actionError }
        if (shown.isNotEmpty()) {
            item {
                // **والخطأُ أحمرُ دائماً** — البرتقاليُّ على الأبيض لا يُقرأ تحت الشمس.
                com.rahalgo.ui.RahalCard(
                    modifier = Modifier.padding(top = Rahal.space.md),
                    variant = com.rahalgo.ui.CardVariant.Tinted,
                    tone = com.rahalgo.ui.StatusTone.Danger,
                ) {
                    Text(shown, color = Rahal.colors.danger, style = Rahal.type.bodyStrong)
                    RahalTextButton(onClick = actions.refresh, tone = com.rahalgo.ui.Tone.Danger) {
                        Text(stringResource(R.string.home_retry))
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **ورديّة مغلقة تُقال هنا لا تُترك فراغا**
        // ══════════════════════════════════════════════════════════════
        //
        // **من ورديّته مغلقة لا يصله عرض أبدا** — فيرى قائمة فارغة
        // **ويظنّ أنّ العمل راكد.** والسبب في يده وهو لا يعرفه.

        // ══════════════════════════════════════════════════════════════
        // **ولا عنوانَ فوق العروض ولا عدد**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «ما في داعي لكلمة معروض عليك أساسا
        //  وعدد الطلبات — نحن أساسا بصفحة الطلبات».)
        //
        // **والبطاقة تقول عن نفسها**: شارةُ «طلب جديد» تفصل المعروضَ عن
        // المقبول، **فسطرٌ يعيد ما تقوله البطاقة حشو.**
        if (state.offers.isEmpty()) {
            // **والفراغ لا يُترك فراغا** — يُقال سببه.
            item { WhyNoOrders(state, actions.openBoard) }
        } else {
            items(state.offers, key = { it.id }) { order ->
                OrderCard(
                    order = order,
                    offer = true,
                    avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
                    busy = state.acceptingId == order.id,
                    // **ولا يُقبل طلبان معا** — الضغطة الثانية أثناء الأولى
                    // تفتح نداءين، **وقد يعود الأوّل بالرفض والثاني بالقبول**
                    // فلا يعرف صاحبه ما الذي وقع.
                    enabled = state.acceptingId == null,
                    onAccept = { actions.accept(order.id) },
                    // **لا «رفض» في الطابور** — قرارُ المالك ٢٠٢٦-١٠-٠٦، أُكِّد ٢٠٢٦-١١-١٠.
                    onDecline = null,
                    onOpen = null,
                    actionLabel = R.string.order_agree,
                    // ══════════════════════════════════════════════════
                    // **والرفض في الحالين** — (قرار المالك ٢٠٢٦-٠٨-١٢)
                    // ══════════════════════════════════════════════════
                    //
                    // **ومعناه يختلف**: في «بالدور» ينقل الدور فورا إلى
                    // من بعده، **وفي «للجميع» يخفيه عن شاشته ويبقى
                    // لغيره** — فلا يقرأ في كلّ تحديث طلبا لا يريده.
                    // **وانقضاء المهلة يعيد القراءة** — البطاقة لم تعد
                    // له، **ومن أبقاها** جعله يضغطها فيُردّ «سبقك غيرك».
                    onExpired = actions.refresh,
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **ولا «طلباتي» هنا — الرحلةُ تقولها**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «قسم طلباتك ما في داعي لها، لأنّها
        //  تنزل بالرحلة — شو الفايدة تكون بالرحلة وهو ما رح يترك الجهاز
        //  والقيادة ليشوف طلباته؟» · «لو سمحنا لقسم طلباتي يمكن أن يأخذ
        //  السائقُ طلباً ويؤخّره إلى حين أن ينتهي، وهذا خطأٌ قاتلٌ
        //  للمنصّة — بالنهاية السائق يهتمّ لمصلحته فقط».)
        //
        // # وحجّةُ المالك الثانيةُ أقوى من الأولى
        //
        // **قائمةٌ يختار منها أيَّهما يتابع تجعل الترتيبَ بيده** — فيبدأ
        // بالأقرب أو بالأربح **ويؤخّر الآخر حتّى ينتهي.** والزبونُ
        // المؤخَّرُ لا يعرف أنّه أُخِّر، **والمنصّةُ تُحاسَب على تأخيرٍ
        // اختاره غيرُها.**
        //
        // **والرحلةُ تفرض الترتيب**: ما بيده يُفتح واحداً، **وما بعده
        // يُقال في شاشتها لا يُختار منها.**
        //
        // # ولم يبقَ ما يُخفى
        //
        // **السقفُ صار واحدا** (٢٠٢٦-٠٨-١٣)، **والثاني لا يجيء إلّا على
        // مساره** — فلا طلبَ «ضائعٌ» في قائمةٍ محذوفة.
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لماذا لا تصلني طلبات؟**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهذا أكثر ما يُسأل عنه المكتب** — والسائق يرى قائمة فارغة فيظنّ
 * العمل راكدا، **والسبب في يده وهو لا يعرفه.**
 *
 * **وكلّ الأسباب في `driver/me` اليوم** — لا نداء جديد ولا حقل جديد:
 * وردية · موقع · نقد بلغ سقفه · طلبات بعدد الحدّ.
 *
 * **وتُقرأ بالترتيب**: الأوّل المانع هو السبب، **ولا يُعرض خمسة أسباب
 * معا** فلا يعرف أيّها يعالج.
 */
@Composable
private fun WhyNoOrders(state: OrdersState, openBoard: () -> Unit = {}) {
    val me = state.me
    // **وما تعدّه لوحتُه ينقصه تعدّه القائمةُ** — مصدرٌ واحد (`Readiness.missingCount`). كانت تقول
    // «كلُّ شيءٍ جاهز» واللوحةُ «ينقصك (١)» (الموقعُ «أثناء الاستخدام» فقط) — ٢٠٢٦-١١-١١.
    val context = androidx.compose.ui.platform.LocalContext.current
    val missing = if (me != null) com.rahalgo.driver.location.Readiness.missingCount(context, me.onShift) else 0
    val reason: Pair<Int, Boolean>? = when {
        me == null -> null
        // **والحسابُ الموقوفُ أوّلُ الأسباب** — كان يُقرأ «كلّ شيء جاهز» ولا طلب.
        me.suspended -> R.string.why_suspended to true
        !me.onShift -> R.string.why_shift_closed to true
        !state.locationOn -> R.string.why_location_off to true
        me.cashLimit > 0 && me.cashHeld >= me.cashLimit -> R.string.why_cash_full to true
        me.maxActiveOrders > 0 && me.activeOrders >= me.maxActiveOrders ->
            R.string.why_too_many to true

        missing > 0 -> R.string.why_missing to true

        else -> R.string.why_all_good to false
    }

    if (reason == null) {
        // ══════════════════════════════════════════════════════════
        // **ولا عملَ الآن — وماذا يفعل** (٢٠٢٦-٠٩-١٣، شرطُ المالك)
        // ══════════════════════════════════════════════════════════
        //
        // **وسائقٌ في دوامه ينظر إلى سطرٍ يقول «لا طلبات»** ولا يعرف
        // أيَنتظر أم يعود. **فيُقال له إنّ الطابورَ يُحدَّث وحدَه.**
        //
        // **والمكوّنُ المشترك لا محلّيٌّ يحجبه** — **ومكوّنان لشيءٍ
        // واحدٍ يفترقان يومَ يُبدَّل أحدُهما.**
        Empty(
            text = stringResource(R.string.orders_no_offers),
            hint = stringResource(R.string.orders_no_offers_hint),
        )
        return
    }

    val (text, blocking) = reason
    // **«كل شيء جاهز» حالةٌ فارغةٌ لقائمة الطلبات لا بطاقةٌ ضيّقة** (المالك ٢٠٢٦-١١-١٠) —
    // مكوّنُ `Empty` المشترك: أيقونةٌ ونصٌّ وسطرٌ يقول ما يفعله، بعرضٍ كامل.
    if (!blocking) {
        Empty(
            text = stringResource(text),
            hint = stringResource(R.string.orders_no_offers_hint),
            icon = R.drawable.ic_orders,
        )
        return
    }
    // **والمانعُ حالةٌ فارغةٌ بأيقونة التنبيه** — بشكل «كل شيء جاهز» نفسه، لا شريطٌ ضيّق
    // (إعادةُ التصميم ٢٠٢٦-١١-١١): السببُ عنوانٌ كبيرٌ يُقرأ بنظرة.
    // **والورديةُ المغلقةُ بزرٍّ يفتح «لوحتي»** — الكلامُ يقول «افتحها من لوحتي» فالزرُّ يأخذه إليها.
    val shiftClosed = text == R.string.why_shift_closed
    // **والنقصُ يأخذه إلى «لوحتي»** حيث أزرارُ إصلاحه — الزرُّ نفسُه الذي هناك لا نسخةٌ ثانية.
    val missingFix = text == R.string.why_missing
    Empty(
        text = stringResource(text),
        icon = R.drawable.ic_warning,
        actionLabel = when {
            shiftClosed -> stringResource(R.string.act_my_board)
            missingFix -> stringResource(R.string.why_missing_button)
            else -> ""
        },
        onAction = if (shiftClosed || missingFix) openBoard else null,
    )
}

@Composable
private fun SectionTitle(text: String, modifier: Modifier = Modifier) {
    Text(
        text = text,
        modifier = modifier.fillMaxWidth(),
        style = Rahal.type.sectionTitle,
        fontWeight = FontWeight.Bold,
    )
}


/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة الطلب — كما رسمها المالك ٢٠٢٦-٠٨-١٢**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ستّة أسطر يقرؤها في ثانيتين وهو واقف**:
 *
 * ١ · **رأس**: «طلب جديد» · عدّاد المهلة · رقمه.
 * ٢ · **المتجر** باسمه.
 * ٣ · **من أين وإلى أين** بأسماء الأحياء — لا إحداثيات.
 * ٤ · **كم يبعد عنك** و**كم المشوار**.
 * ٥ · **أجرتك** و**كيف يُدفع** و**كم يستغرق**.
 * ٦ · **موافق** و**رفض**.
 *
 * # ولماذا اسم الحيّ لا العنوان الكامل
 *
 * **«حي الروضة» يعرفه السائق في لحظة** — والعنوان الكامل سطران يقرؤهما
 * بعد أن يقبل. **والقرار يحتاج الجهة لا الباب.**
 *
 * # وأجرتك غير إجماليّ الطلب
 *
 * **الأولى ما يكسبه، والثاني ما يقبضه للمتجر** — وكان يرى الثاني وحدَه
 * فيظنّه كسبه.
 */
@Composable
private fun OrderCard(
    order: DriverOrder,
    offer: Boolean,
    busy: Boolean,
    enabled: Boolean,
    avgSpeedKmh: Long,
    onAccept: () -> Unit,
    onOpen: (() -> Unit)?,
    actionLabel: Int,
    onExpired: () -> Unit = {},
    /** **«رفض»** — للعروض فقط (قرارُ المالك ٢٠٢٦-١١-١٠)؛ والخادمُ ينقل الدورَ فوراً. */
    onDecline: (() -> Unit)? = null,
) {
    // **بطاقةُ النظام الموحّد** (إعادةُ التصميم ٢٠٢٦-١١-١١) — مرفوعةٌ بظلٍّ خفيفٍ لأنّها تُقبل،
    // وأرضُها أرضُ السمة في الفاتحة والغامقة.
    com.rahalgo.ui.RahalCard(
        modifier = Modifier.padding(top = Rahal.space.md),
        variant = com.rahalgo.ui.CardVariant.Elevated,
        onClick = onOpen,
        padding = Rahal.space.lg,
    ) {
        // ══════════════════════════════════════════════════════════════
        // **والخاصُّ يُعرف مرّةً ويُقرأ في كلّ سطر**
        // ══════════════════════════════════════════════════════════════
        //
        // **طلبٌ بلا متجرٍ ولا سعرٍ متّفقٍ عليه بعد** — وبطاقتُه تقول
        // ذلك، لا تعرض أصفاراً.
        //
        // **و`kind` من المحرّك لا من الاسم**: متجرٌ يسمّي نفسَه «طلب
        // خاصّ» يُقرأ خاصّاً لو قيس بالاسم.
        val custom = order.kind == "custom"
        // **وموقعُ الأمانة على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — حدّده الزبون، فتُقال المسافةُ إليه.
        val amanahPoint = custom && order.customMode == "amanah" && order.navLat != null && order.navLng != null

        // ══════════════════════════════════════════════════════════════
        // **الرأس ثلاثة: المتجر · الشارة · الرقم**
        // ══════════════════════════════════════════════════════════════
        //
        // (تصميم المالك ٢٠٢٦-٠٨-١٢: «يمين وسط يسار».)
        //
        // **واسم المتجر أوّل ما تقع عليه العين** — وهو أوّل ما يقرّر به:
        // مطعم يعرفه أم لا. **وفي العربيّة أوّل الموضع اليمين.**
        //
        // **والعدّاد تحت الشارة**: «طلب جديد» تقول ماذا، **والعدّاد يقول
        // كم بقي** — والاثنان خبر واحد فيبقيان معا.
        //
        // **والإجماليّ تحت الرقم**: هو ما يقبضه من الزبون، **وشامل أجرة
        // التوصيل** (المحرّك يحسبه كذلك، فلا يُجمع هنا ثانية).
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.Top,
        ) {
            // ── يمين: الشارة والعدّاد ──
            //
            // **(تبديل المالك ٢٠٢٦-٠٨-١٢.)** والشارةُ تقول ما هذه
            // البطاقة قبل أن يُقرأ ما فيها.
            Column(horizontalAlignment = Alignment.Start) {
                if (offer) {
                    com.rahalgo.ui.StatusChip(
                        text = stringResource(R.string.card_new_order),
                        tone = com.rahalgo.ui.StatusTone.Info,
                        icon = R.drawable.ic_bell,
                    )
                    // **ونوعُه جنبها** (ملاحظةُ المالك ٢٠٢٦-١٠-١٢) — عاديّ · توصيلة · مشتريات · أمانة.
                    Spacer(Modifier.height(Rahal.space.xs))
                    com.rahalgo.ui.Badge(
                        text = stringResource(com.rahalgo.driver.trip.kindLabel(order)),
                        tone = com.rahalgo.ui.StatusTone.Warning,
                    )
                } else {
                    Text(
                        text = statusText(order.status),
                        color = Rahal.colors.brand,
                        fontWeight = FontWeight.Bold,
                        style = Rahal.type.caption,
                    )
                }
            }

            // ── وسط: المتجر ──
            //
            // **والأيقونةُ يمينَ الاسم** — (تصحيح المالك ٢٠٢٦-٠٨-١٢).
            // **وأوّلُ الصفّ في العربيّة يمينُه**، فتُكتب قبله.
            // ══════════════════════════════════════════════════════════
            // **والعرضُ يُقسَّم صراحةً — وإلّا انكسر الرقم حرفاً حرفاً**
            // ══════════════════════════════════════════════════════════
            //
            // (شهده المالك ٢٠٢٦-٠٩-٢٩ في بطاقةٍ على جهازه: «إجمالي
            //  الفاتورة الرقم كل رقم لحاله».)
            //
            // **كان هذا الصفُّ بلا `weight`** — فيأخذ اسمُ المتجر ما يشاء
            // («مطعم بيت الرقة» ستَّ عشرةَ حرفاً)، **ويبقى لعمود السعر
            // فُتاتٌ**، فيلتفّ الرقمُ على نفسه: ١٥، ثمّ ١٥ ثمّ ٠ ثمّ ل.
            // ثمّ س — **خمسةَ أسطرٍ لرقمٍ واحد.**
            //
            // **ورقمٌ مكسورٌ أسوأُ من رقمٍ غائب**: الغائبُ يُسأل عنه،
            // **والمكسورُ يُقرأ خطأً** — «١٥» تُقرأ خمسةَ عشرَ ألفاً أو
            // خمسةَ عشرَ، **والسائقُ يقبض بها.**
            //
            // **فالاسمُ يأخذ المتبقّي ويُقصّ، والسعرُ لا يُسحق**: أولويّةُ
            // العرض للرقم لأنّه المال.
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.weight(1f, fill = false),
            ) {
                Icon(
                    // **ولا أيقونةَ متجرٍ لطلبٍ بلا متجر** — والصورةُ
                    // تُقرأ قبل الحرف: **من رأى واجهةَ دكّانٍ انتظر
                    // فاتورةً جاهزةً وبضاعةً تُستلم.**
                    painter = painterResource(
                        if (custom) R.drawable.ic_orders else R.drawable.ic_store,
                    ),
                    // **ويُسمّى للقارئ الصوتيّ** — ومن يقود ويسمع لا يرى.
                    contentDescription = stringResource(com.rahalgo.driver.trip.kindLabel(order)),
                    tint = Rahal.colors.accent,
                    modifier = Modifier.size(Rahal.icon.md),
                )
                Spacer(Modifier.size(Rahal.space.sm))
                Text(
                    text = com.rahalgo.driver.Bidi.isolate(order.merchantName).ifBlank { order.holderName.ifBlank { stringResource(com.rahalgo.driver.trip.kindLabel(order)) } },
                    style = Rahal.type.sectionTitle,
                    fontWeight = FontWeight.Bold,
                    // ══════════════════════════════════════════════════
                    // **والاسمُ لا يُقصّ — نزل السعرُ ليتّسع له**
                    // ══════════════════════════════════════════════════
                    //
                    // (بلاغُ المالك ٢٠٢٦-٠٩-٢٩: «اسم المطعم أو المتجر
                    //  مقصوص انتبه».)
                    //
                    // **وأوّلُ إصلاحٍ قصّه**: مُنح الصفُّ `weight` فكفّ
                    // الرقمُ عن الانكسار، **لكنّ صفَّ «إجمالي الفاتورة»
                    // كان يلتهم ٤٣٠ بكسلاً من الرأس** (قِيس على الجهاز:
                    // الاسمُ رُسم بـ١٣٥ ومثلُه في السطر التالي ٤١٨)،
                    // **فبقي للاسم فُتاتٌ فقُصّ.**
                    //
                    // **فنزل السعرُ سطراً مستقلّاً** (أدناه): الرأسُ
                    // للشارة والاسم والرقم، **والسعرُ بعرض البطاقة** —
                    // **فلا اسمٌ يُقصّ ولا رقمٌ يُكسر، ولا مقايضةَ
                    // بينهما.**
                    //
                    // **والسطران أرخصُ من أحدهما ناقصاً.**
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    // **واسمُ الخاصّ بلون التنبيه** — ليس متجراً يُقرأ
                    // كسائر المتاجر، **وطريقتُه في المال مختلفة.**
                    color = if (custom) Rahal.colors.accent else Color.Unspecified,
                )
            }

            // ── يسار ──
            Column(horizontalAlignment = Alignment.End) {
                Text(
                    text = com.rahalgo.driver.Bidi.isolate("#" + order.number),
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.meta,
                )
            }
        }

        // ── سطرُ المال نزل مربّعاً تحت الأجرة والدفع (ملاحظةُ المالك ٢٠٢٦-١٠-١٢) ──
        // ══════════════════════════════════════════════════════════════
        // **وما طلبه الزبونُ بلفظه — قبل كلّ شيء**
        // ══════════════════════════════════════════════════════════════
        //
        // (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن يُذكر به ما نوعُ الطلب».)
        //
        // **وبطاقةٌ تقول «طلب خاصّ» وحدَها تطلب منه أن يقبل ما لا
        // يعرفه**: أدواءٌ من صيدليّة أم أسمنتٌ من مستودع؟ **والفرقُ
        // بينهما درّاجةٌ ووقت.**
        //
        // **وموضعُه فوق المسافة**: هو أوّلُ ما يقرّر به القبولَ من
        // الرفض، **والمسافةُ بعده.**
        // **و«لدي توصيلة»: ما يحمله قبل أن يقبل** (٢٠٢٦-١٠-١٢) — كان يظهر بعد القبول وحدَه.
        if (order.kind == "merchant_delivery" && order.parcelNote.isNotBlank()) {
            Spacer(Modifier.height(Rahal.space.md))
            WhatBox(stringResource(com.rahalgo.driver.trip.parcelLabel(order)), order.parcelNote)
        }
        // **ومالٌ يدفعه من جيبه قبل أن يقبل** (قرارُ المالك ٢٠٢٦-١٠-١٥) — «ادفع للمتجر X من
        // جيبك، واقبض من المستلم X»: يعرفه قبل أن يقبل لا عند المتجر.
        // **وفي «إحضار» بالعكس** (٢٠٢٦-١٠-١٦): يدفع بمكان الاستلام ويقبضه من المتجر.
        com.rahalgo.driver.trip.collectForStore(order)?.let { amt ->
            Spacer(Modifier.height(Rahal.space.md))
            // **وعنوانُه يقول ما يفعل** — لا «لدي توصيلة» (اسمُ الخدمة عند المتجر).
            WhatBox(
                stringResource(R.string.card_out_of_pocket),
                stringResource(com.rahalgo.driver.trip.collectTextRes(order), com.rahalgo.driver.trip.orderAmount(order, amt)),
            )
        }
        if (custom && order.customRequest.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.md))
            WhatBox(stringResource(R.string.card_custom_what), order.customRequest)
        }
        // **وصورُ قائمة الزبون أو الغرض** (قرارُ المالك ٢٠٢٦-١١-١١) — تحت وصف الطلب.
        if (custom && order.customPhotos.isNotEmpty()) {
            Spacer(Modifier.height(Rahal.space.sm))
            com.rahalgo.ui.CustomPhotoStrip(
                urls = order.customPhotos,
                media = { p -> com.rahalgo.ui.AppCore.get().media(p) },
            )
        }

        // **وفاصلٌ واحدٌ لا اثنان** — كان `Spacer(12)` مكرَّراً سطرين
        // متتاليين فصار الفراغُ فوق الطرفين ٢٤dp بينما سائرُ البطاقة ١٠–١٢
        // (فحصُ البطاقة بصريّاً ٢٠٢٦-٠٩-٢٩).
        Spacer(Modifier.height(Rahal.space.md))
        // ══════════════════════════════════════════════════════════════
        // **والمسافة في سطر طرفها لا في صندوق تحته**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «لا داعي لكتابة المسافة تحته، بل
        //  دعها بنفس السطر».)
        //
        // **والمسافتان من طرفين مختلفين**: الأولى من موضعك إلى المتجر،
        // **والثانية من المتجر إلى باب الزبون** — لا من موضعك إليه.
        //
        // **والاسم في العنوان لا «المتجر» و«الزبون»**: (تصحيح المالك
        // ٢٠٢٦-٠٨-١٢) — **«المسافة إلى الزبون» لا تقول لمن يحمل**،
        // و«ابوطيف» يقول. **وثلاثة طلبات في يده لا يفرّق بينها** إن كان
        // كلُّها «إلى الزبون».
        //
        // **والاسم في السطر لا كلمة «المتجر»** — (تصحيح المالك
        // ٢٠٢٦-٠٨-١٢): **«المسافة إلى (طيف)» تقول كلَّ شيء**، وكلمةُ
        // «المتجر» قبلها حشوٌ يعرفه من رأى الأيقونة.
        //
        // **ولا يُكرَّر الاسمُ تحته**: عنوانٌ فارغٌ لا يُملأ بالاسم —
        // **سطرٌ يعيد ما فوقه يُقرأ عطبا**، ولا يُضيف شيئا.
        // **ولا سطرَ إلى متجرٍ لا وجودَ له** — الخاصُّ لا مصدرَ له حتّى
        // يقوله الزبون. **وكان يُكتب «المسافة إلى (طلب خاصّ)» بلا مسافةٍ
        // ولا عنوان** — سطرٌ يشغل مكاناً ولا يقول شيئا.
        val outsideShop = custom && com.rahalgo.driver.trip.isOutsideShop(order)
        if (!custom || amanahPoint || outsideShop) Leg(
            label = if (outsideShop) {
                stringResource(R.string.card_buy_from, com.rahalgo.driver.Bidi.isolate(order.merchantName))
            } else stringResource(
                R.string.card_dist_to,
                if (amanahPoint) stringResource(R.string.amanah_place)
                else com.rahalgo.driver.Bidi.isolate(order.merchantName.ifBlank { stringResource(R.string.nav_custom_order) }),
            ),
            value = if (amanahPoint) "" else order.pickupAddress,
            far = legDistanceText(order.toPickupM),
            // **والزمن بجانب المسافة** — (قرار المالك ٢٠٢٦-٠٨-١٢).
            //
            // **والمسافة وحدها لا تقرّر**: «٣ كم» تعني ربع ساعة في زحمة
            // ودقيقتين على طريق فارغ، **ومن يوازن بين طلبين يوازن
            // بالوقت** لا بالمتر.
            mins = legEta(order.toPickupM, avgSpeedKmh),
            tint = Rahal.colors.brand,
        )
        Leg(
            // **واسمُ الزبون بين القوسين** — (تصحيح المالك ٢٠٢٦-٠٨-١٢).
            // **وطلبُ الزائر لا اسمَ له**، فيُكتب «الزبون» ولا يُترك
            // قوسان فارغان يظنّهما صاحبُهما عطبا.
            label = stringResource(
                R.string.card_dist_to,
                com.rahalgo.driver.trip.customerShownName(order.customerName) ?: stringResource(R.string.card_customer),
            ),
            value = com.rahalgo.driver.trip.cleanAddress(order.addressText),
            far = legDistanceText(order.legM),
            mins = legEta(order.legM, avgSpeedKmh),
            tint = Rahal.colors.accent,
        )

        Spacer(Modifier.height(Rahal.space.md))
        // **والصناديق متساوية الارتفاع** — `IntrinsicSize.Min` تقيسها
        // بأطولها، **ولولاها لطال صندوقٌ سطراه** وقصر جاراه.
        Row(
            Modifier.fillMaxWidth().height(IntrinsicSize.Min),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
        ) {
            Metric(
                // **وأيقونةُ نقدٍ لا علامةُ دولار** — (تصحيح المالك
                // ٢٠٢٦-٠٨-١٢): **رمز عملة أجنبيّة على مبلغ بالليرة**
                // يقرؤه صاحبه لحظةً بغير ما هو.
                icon = R.drawable.ic_cash,
                label = stringResource(R.string.card_fee),
                // **وأجرةُ الخاصّ تُتّفق ولا تُحسب** — وصفرٌ مكانَها
                // يُقرأ «بلا أجرة»، **فيُرفض طلبٌ أجرتُه لم تُقَل بعد.**
                // **وأجرةُ المنصّة المفروضةُ رقمٌ لا «تتفق»** (٢٠٢٦-١٠-١٢).
                value = when {
                    com.rahalgo.driver.trip.fixedCustomFee(order) != null ->
                        money(com.rahalgo.driver.trip.fixedCustomFee(order) ?: 0)
                    custom -> stringResource(R.string.card_agreed_later)
                    else -> money(order.deliveryFee)
                },
                // **مجانيّةٌ للزبون أم يدفعها** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — والأجرةُ أعلاه كاملةٌ
                // في الحالين: المنصّةُ تتحمّل المجانيّة. وتوصيلُ المتجر له «من يدفع» في الصندوق التالي.
                note = when {
                    order.freeWaived > 0 -> stringResource(R.string.card_fee_free)
                    order.kind == "merchant_delivery" -> null
                    custom && com.rahalgo.driver.trip.fixedCustomFee(order) == null -> null
                    else -> stringResource(R.string.card_fee_paid)
                },
                noteTone = if (order.freeWaived > 0) Rahal.colors.success else null,
                strong = true,
                modifier = Modifier.weight(1f),
            )
            // **ولون المربّع يقول حال المال بلا قراءة** — (قرار المالك
            // ٢٠٢٦-٠٨-١٢): **أحمر يعني اقبض من الزبون**، وأخضر يعني
            // مدفوع مسبقا فسلّم وامضِ.
            //
            // **ومن خلط بينهما** طالب زبونا دفع، **أو مشى بلا نقد فخسر
            // ثمن الطلب من جيبه.**
            val cash = order.cashDue > 0
            Metric(
                // **والأيقونة تقول ما يقوله اللون** — نقدٌ في اليد أو
                // محفظة، **فمن أخطأ اللون لم يخطئ الشكل.**
                icon = if (cash || custom) R.drawable.ic_cash else R.drawable.ic_wallet,
                label = stringResource(R.string.card_payment),
                // **والخاصُّ يشتريه من جيبه ويستردّ عند التسليم** —
                // وهو ما يقوله النموذجُ نفسُه. **و«مدفوع» عليه أخطرُ ما
                // في البطاقة**: يمضي بلا أن يقبض ثمنَ ما اشتراه.
                // **ومن مَن يقبض** (ملاحظةُ المالك ٢٠٢٦-١٠-١٢: «لازم يعرف من وين بدو يقبض من
                // المتجر ولا من المستلم») — **والأمانةُ بلا شراء.**
                value = when {
                    custom && order.customMode == "amanah" -> stringResource(R.string.card_amanah_pay)
                    custom -> stringResource(R.string.card_custom_pay)
                    order.kind == "merchant_delivery" -> when (order.feePayer) {
                        "recipient" -> stringResource(R.string.card_pay_recipient)
                        "merchant_cash" -> stringResource(R.string.card_pay_store)
                        else -> stringResource(R.string.card_pay_store_wallet)
                    }
                    cash -> stringResource(R.string.card_cash)
                    else -> stringResource(R.string.card_wallet)
                },
                tone = if (cash || custom) Rahal.colors.danger else Rahal.colors.success,
                modifier = Modifier.weight(1f),
            )
            // **والمدّة تُحسب ولا تُترك فارغة** — إلّا إن لم تُضبط السرعة:
            // **رقم مبنيّ على صفر يقول «الآن»** وهو وعد كاذب.
            val minutes = eta(order.toPickupM, order.legM, avgSpeedKmh)
            if (minutes > 0) {
                Metric(
                    icon = R.drawable.ic_time,
                    label = stringResource(R.string.card_eta),
                    value = minutesText(minutes),
                    modifier = Modifier.weight(1f),
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **زرٌّ واحدٌ — يوافق أو يترك**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٤: «زرُّ رفضٍ عند السائق — يجب أن يوافق
        //  على الطلب فقط. الطابورُ كلُّ السائقين يشاهدونه وليس سائقاً
        //  واحداً… السائقُ لا علاقة له بالرفض، هو إمّا يوافق أو يترك
        //  الطلبَ لغيره».)
        //
        // # ولماذا الرفضُ لا معنى له
        //
        // **الطابورُ يراه الجميع** — ومن لا يريده يتركه، **فيأخذه أوّلُ
        // من يريده.** والرفضُ لا يُحرّك شيئاً: **يُخفي البطاقةَ عن
        // صاحبها وحدَه.**
        //
        // **والمُسنَدُ له عدّاد** — فإن انقضى **عاد إلى الطابور من نفسه**
        // (`reclaimSilentAssignments`) موسوماً ألّا يعود إليه. **فالزمنُ
        // يفعل ما كان الزرُّ يفعله**، بلا أن يُطلب من السائق قرارٌ سلبيّ.
        //
        // # وضررُه الذي قِيس
        //
        // **زرٌّ أحمرُ بحجم الأخضر يقول إنّ الرفضَ قرارٌ يُتّخذ** — فمن
        // تردّد ضغطه ليُنهي البطاقة، **ثمّ ندم ولا يجدها.**
        //
        // **وبابُه في المحرّك يبقى** (`/decline`) — لا يُنادى من هنا،
        // **ولوحةُ الإدارة قد تحتاجه.**
        // ══════════════════════════════════════════════════════════════
        // **والإجماليُّ مربّعٌ تحت الأجرة والدفع** (ملاحظةُ المالك ٢٠٢٦-١٠-١٢: «اجمالي
        // الفاتورة نخليها مربع تحت اجرة التوصيل والدفع افضل واضحة»)
        // ══════════════════════════════════════════════════════════════
        Spacer(Modifier.height(Rahal.space.md))
        TotalBox(order)
        Spacer(Modifier.height(Rahal.space.md))
        // **الأزرارُ في أسفل البطاقة بمتناول الإبهام: «رفض» بجانب «موافق».**
        // **والعدّادُ في الزرّ لا في زاوية البطاقة** — (قرارُ المالك
        // ٢٠٢٦-٠٨-١٤: «العدّادُ يكون داخل زرّ موافق بطريقةٍ احترافيّة،
        // ليكون واضحاً أنّه بمجرّد انتهاء الزمن سيختفي الطلب»).
        //
        // **ورقمٌ في الزاوية يُقرأ ثمّ يُنسى** — والزرُّ في مكانٍ آخر،
        // **فلا يربط الذهنُ بينهما.**
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (offer && onDecline != null) {
                com.rahalgo.ui.RahalOutlineButton(
                    onClick = onDecline,
                    enabled = enabled,
                    tone = com.rahalgo.ui.PriceConfirm.Reject,
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.order_decline), fontWeight = FontWeight.Bold) }
            }
            CountdownButton(
                expiresAt = if (offer) order.offerExpiresAt else null,
                onExpired = onExpired,
                onClick = onAccept,
                label = stringResource(actionLabel),
                icon = R.drawable.ic_check_circle,
                enabled = enabled,
                busy = busy,
                modifier = if (offer && onDecline != null) Modifier.weight(2f) else Modifier.fillMaxWidth(),
            )
        }
    }
}


/** **مربّعُ «ماذا»** — المطلوبُ أو ما يحمله. */
@Composable
private fun WhatBox(label: String, text: String) {
    com.rahalgo.ui.RahalCard(
        variant = com.rahalgo.ui.CardVariant.Tinted,
        tone = com.rahalgo.ui.StatusTone.Warning,
    ) {
        Text(text = label, color = Rahal.colors.inkMuted, style = Rahal.type.label)
        Spacer(Modifier.height(Rahal.space.xxs))
        Text(text = text, color = Rahal.colors.ink, style = Rahal.type.bodyStrong)
    }
}

/**
 * **مربّعُ المال** — من يدفع وكم، بالرقم الكبير. **أحمرُ يُقبض، وأخضرُ لا يُقبض.**
 */
@Composable
private fun TotalBox(order: DriverOrder) {
    val custom = order.kind == "custom"
    val fixed = com.rahalgo.driver.trip.fixedCustomFee(order)
    // **ما يُقبض بلون التنبيه الدافئ لا الأحمر** — الأحمرُ للخطأ (المالك ٢٠٢٦-١١-١٠).
    val collectTint = Rahal.colors.accent
    val standardCash = order.kind == "standard" && order.cashDue > 0
    if (standardCash) {
        // **ادفع للمتجر** منفصلٌ عن **تقبض من الزبون** — مالان مختلفان بجهتين مختلفتين.
        if (order.payMerchant > 0) {
            MoneyRow(stringResource(R.string.trip_pay_merchant), money(order.payMerchant), Rahal.colors.brand)
            Spacer(Modifier.height(Rahal.space.sm))
        }
        if (order.total != order.cashDue) {
            MoneyRow(stringResource(R.string.offer_invoice_total), money(order.total), Rahal.colors.inkMuted)
            Spacer(Modifier.height(Rahal.space.sm))
        }
        MoneyRow(stringResource(R.string.offer_collect_customer), money(order.cashDue), collectTint)
        return
    }
    val (label, amount, collect) = when {
        order.kind == "merchant_delivery" -> when (order.feePayer) {
            "recipient" -> Triple(stringResource(R.string.total_collect_recipient), money(order.cashDue), true)
            "merchant_cash" -> Triple(stringResource(R.string.total_collect_store), money(order.cashDue), true)
            else -> Triple(stringResource(R.string.total_paid_wallet), "", false)
        }
        custom && order.customMode == "amanah" && fixed != null ->
            Triple(stringResource(R.string.total_collect_customer), money(fixed), true)
        custom && fixed != null ->
            Triple(stringResource(R.string.total_purchase_fixed, money(fixed)), "", true)
        custom -> Triple(stringResource(R.string.card_custom_price), "", true)
        order.cashDue > 0 -> Triple(stringResource(R.string.card_invoice_total), money(order.total), true)
        else -> Triple(stringResource(R.string.total_paid_customer), money(order.total), false)
    }
    MoneyRow(label, amount, if (collect) collectTint else Rahal.colors.success)
}

/** **سطرُ مال** — وصفٌ وراءه رقمٌ كبير بنغمةٍ واحدة. */
@Composable
private fun MoneyRow(label: String, amount: String, tint: Color) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(tint.copy(alpha = 0.10f))
            .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(
            text = label,
            color = tint,
            style = Rahal.type.bodyStrong,
            modifier = Modifier.weight(1f, fill = false),
        )
        if (amount.isNotEmpty()) {
            Spacer(Modifier.size(Rahal.space.sm))
            Text(
                text = com.rahalgo.driver.Bidi.isolate(amount),
                color = tint,
                style = Rahal.type.priceLarge,
                maxLines = 1,
                softWrap = false,
            )
        }
    }
}

/**
 * **طرف الرحلة** — دبّوس وعنوان.
 *
 * **والعنوان تحت عنوانه لا بجانبه**: «شارع الكهربا جانب مغسلة أبو
 * الهيف» لا يسع سطرا فيه كلمة «إلى الزبون» قبله — **فينقطع بنقاط،
 * وأهمّ ما فيه آخره.**
 *
 * **وسطران حدّه**: ثلاثة تجعل البطاقة تطول فلا يُرى زرّها بلا تمرير.
 */
@Composable
private fun Leg(label: String, value: String, far: String, mins: Long, tint: Color) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = Rahal.space.xs),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(Rahal.icon.sm),
        )
        Spacer(Modifier.size(Rahal.space.sm))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // **والاسمُ يأخذ ما بقي ويُقصّ** — لا يأكل المسافةَ والمدّة.
                Text(
                    label,
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (far.isNotEmpty()) {
                    Spacer(Modifier.size(Rahal.space.sm))
                    // **والمسافة في سطر عنوانها** — رقم قصير لا يزاحم
                    // العنوان، **ويُقرأ معه لا بعد أن يبحث عنه.**
                    Text(
                        // **والرقم بين قوسين** — (طلب المالك): يُقرأ
                        // رقما مستقلّا لا امتدادا للاسم قبله.
                        text = far,
                        color = tint,
                        fontWeight = FontWeight.Bold,
                        style = Rahal.type.caption,
                    )
                }
                if (mins > 0) {
                    Spacer(Modifier.size(Rahal.space.sm))
                    // **والزمن باهت والمسافة ملوّنة** — خبران في سطر،
                    // **ولو تساويا في الصياح** لم يُقرأ أحدهما أوّلا.
                    Text(
                        text = minutesText(mins),
                        color = Rahal.colors.inkMuted,
                        style = Rahal.type.caption,
                    )
                }
            }
            // **وسطرٌ فارغٌ لا يُرسم** — لا فراغَ تحت العنوان ولا اسمٌ
            // مكرّر مكانه.
            if (value.isNotBlank()) {
                Text(
                    text = value,
                    fontWeight = FontWeight.Bold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    style = Rahal.type.body,
                )
            }
        }
    }
}

/** **رقم بعنوانه** — في صندوق صغير. */
@Composable
private fun Metric(
    icon: Int,
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    strong: Boolean = false,
    /** **لون الحال** — وفارغ يعني صندوقا محايدا. */
    tone: Color? = null,
    /** **سطرٌ صغيرٌ تحت القيمة** — كـ«مجانيّة للزبون». وفارغٌ لا يُرسم. */
    note: String? = null,
    noteTone: Color? = null,
) {
    val ink = tone ?: if (strong) Rahal.colors.brand else Rahal.colors.ink
    Column(
        modifier
            .clip(Rahal.shape.md)
            // **وخلفيّة باهتة لا صمّاء** — اللون يُقرأ ولا يصرخ.
            .background(tone?.copy(alpha = 0.10f) ?: Rahal.colors.surface)
            .padding(vertical = Rahal.space.md, horizontal = Rahal.space.sm),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = tone ?: if (strong) Rahal.colors.brand else Rahal.colors.inkMuted,
                modifier = Modifier.size(Rahal.icon.xs),
            )
            Spacer(Modifier.size(Rahal.space.xs))
            Text(label, color = Rahal.colors.inkMuted, style = Rahal.type.label)
        }
        Spacer(Modifier.height(Rahal.space.xs))
        // ══════════════════════════════════════════════════════════════
        // **ولا تُبتر القيمة في سطر** — (تصحيح المالك ٢٠٢٦-٠٨-١٢:
        // «كلمة عند الاستلام طلعت ناقصة، بس طلعت عند»)
        // ══════════════════════════════════════════════════════════════
        //
        // **و«عند» وحدها أسوأ من لا شيء**: تقرأ نصف الجملة **فتظنّ أنّك
        // قرأتها**، ولا شيء في الشاشة يقول إنّ بقيّتها سقطت.
        Text(
            text = value,
            color = ink,
            maxLines = 2,
            textAlign = TextAlign.Center,
            style = Rahal.type.bodyStrong,
        )
        if (note != null) {
            Text(
                text = note,
                color = noteTone ?: Rahal.colors.inkMuted,
                fontWeight = if (noteTone != null) FontWeight.Bold else null,
                textAlign = TextAlign.Center,
                style = Rahal.type.caption,
            )
        }
    }
}

/**
 * **المدّة المتوقّعة للرحلة كلّها** — إليه ثمّ إلى الزبون.
 *
 * **وصفر يعني لا تُعرض**: سرعة غير مضبوطة أو مسافة مجهولة، **ورقم مبنيّ
 * على صفر وعد كاذب.**
 */
private fun legEta(meters: Double, avgSpeedKmh: Long): Long {
    if (avgSpeedKmh <= 0 || meters <= 0) return 0
    return ((meters / 1000.0) / avgSpeedKmh * 60).toLong().coerceAtLeast(1)
}

/**
 * **الدقائق بأرقام غربيّة** — كالمسافة والمبلغ.
 *
 * **و`%d` في ملفّ النصوص يُكتب بأرقام هنديّة** في لغة عربيّة، **فيقع
 * «١ دقيقة» بجانب «56 م»** في السطر نفسه.
 */
private fun minutesText(mins: Long): String = minutes(mins)

/**
 * **المدّة المتوقّعة للرحلة كلّها** — إليه ثمّ إلى الزبون.
 */
private fun eta(toPickupM: Double, legM: Double, avgSpeedKmh: Long): Long {
    if (avgSpeedKmh <= 0) return 0
    val meters = (if (toPickupM > 0) toPickupM else 0.0) + (if (legM > 0) legM else 0.0)
    if (meters <= 0) return 0
    return ((meters / 1000.0) / avgSpeedKmh * 60).toLong().coerceAtLeast(1)
}

/**
 * **المسافة بالمتر أو بالكيلومتر** — لا «1400 م».
 *
 * **والسائق يقدّر بالكيلومتر فوق الألف** — ورقم بأربع خانات يُقرأ مرّتين.
 */
private fun distance(meters: Double): String = dist(meters)

/**
 * **مسافةُ ساقٍ في البطاقة، أو فارغٌ إن لم تُعرف أو كانت صفراً.**
 *
 * (ملاحظةُ المالك ٢٠٢٦-١١-١٠: «(٠ م)» بلا معنى.) **`leg_m` من الخادم: سالبٌ لا يُعرف،
 * وصفرٌ حين يتطابق موضعُ المتجر والزبون (نقطتان واحدة، أو بيانات تجربة)، ولا مشوارَ طوله
 * أقلّ من متر.** فلا يُكتب «(0 م)» — يُحذف القوسان معه.
 */
internal fun legDistanceText(meters: Double): String = if (meters >= 1.0) dist(meters) else ""

/**
 * **حال الطلب بالعربيّة.**
 *
 * **والمجهولُ «قيد المعالجة» لا رمزُه** (قرارُ المالك ٢٠٢٦-١٠-١٨: ولا رمزَ
 * إنكليزيٌّ أمام الناس) — **والحارسُ (`RawCodeTest`) يكشف حالاً زِيدت بلا اسم.**
 */
@Composable
private fun statusText(status: String): String = when (status) {
    "assigned" -> stringResource(R.string.status_assigned)
    // **وحالان كان المحرّك يقولهما والشاشةُ لا تعرفهما** — فيُعرض
    // الرمزُ خاما: «at_pickup» في بطاقةِ سائقٍ عربيّ. (قيس ٢٠٢٦-٠٨-١٢.)
    "at_pickup" -> stringResource(R.string.status_at_pickup)
    "at_dropoff" -> stringResource(R.string.status_at_dropoff)
    "picked_up" -> stringResource(R.string.status_picked_up)
    "on_the_way" -> stringResource(R.string.status_on_the_way)
    "arrived" -> stringResource(R.string.status_arrived)
    "dispatching" -> stringResource(R.string.ord_st_dispatching)
    // **وفي قائمته لا يظهر فاشلٌ إلّا مشوارَ إرجاع** (قرارُ المالك ٢٠٢٦-١٠-٠٣).
    "failed" -> stringResource(R.string.status_returning)
    else -> stringResource(com.rahalgo.ui.R.string.tik_progress)
}

/** ما تعرضه الشاشة — **ولا تملكه هي.** */
data class OrdersState(
    val offers: List<DriverOrder> = emptyList(),
    val mine: List<DriverOrder> = emptyList(),
    /** حال السائق كاملا — **منه تُبنى إجابة «لماذا لا تصلني طلبات».** */
    val me: DriverMe? = null,
    val locationOn: Boolean = true,
    val loading: Boolean = true,
    /** الطلب الذي يُقبل الآن — **وفارغ يعني لا شيء قيد القبول.** */
    val acceptingId: String? = null,
    val error: String = "",
    /**
     * ══════════════════════════════════════════════════════════════════
     * **خطأُ الفعل — لا يمحوه تحديثُ القائمة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (كشفته دورةٌ حقيقيّةٌ على المحاكي ٢٠٢٦-٠٨-١٣.)
     *
     * **ضُغط «موافق» فرُدَّ بـ«معك طلبات بعدد حدّك» — ولم يظهر شيء.**
     *
     * **والسببُ أنّ القبولَ بعد فشله يُعيد قراءةَ القائمة** — عمداً:
     * البطاقةُ قد تكون ذهبت لغيره. **والقراءةُ الناجحةُ تمسح `error`**،
     * فيُكتب الخبرُ ويُمحى في أقلّ من ثانية.
     *
     * **وزرٌّ يُضغط فلا يقع شيءٌ ولا يُقال لماذا يُقرأ عطباً** — ثمّ
     * يُعاد الضغطُ ويُعاد.
     *
     * **فحقلان لا واحد**: هذا لِما فشل من أفعاله، **وذاك لِما تعذّرت
     * قراءتُه** — ولا يمحو أحدُهما الآخر.
     */
    val actionError: String = "",
)

data class OrdersActions(
    val accept: (String) -> Unit,
    /** **يفتح رحلته** — يختار الطلب وينتقل إلى الخريطة. */
    val startTrip: (String) -> Unit,
    val refresh: () -> Unit,
    /** **رفضُ عرضٍ** — يُخفيه وينقل الدورَ للتالي. */
    val decline: (String) -> Unit = {},
    /** **يفتح «لوحتي»** — حيث تُفتح الوردية (رُئي على جوال المالك ٢٠٢٦-١١-١١: «افتحها من لوحتي» بلا زرّ). */
    val openBoard: () -> Unit = {},
)
