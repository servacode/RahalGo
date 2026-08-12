package com.rahalgo.driver.orders

import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateGreen
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.grouped
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder

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
            CircularProgressIndicator()
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
            start = 20.dp,
            end = 20.dp,
            bottom = 28.dp,
        ),
    ) {
        if (state.error.isNotEmpty()) {
            item {
                Column(Modifier.fillMaxWidth().padding(vertical = 12.dp)) {
                    Text(state.error, color = BrandOrange)
                    TextButton(onClick = actions.refresh) {
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
            item { WhyNoOrders(state) }
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
                    onOpen = null,
                    actionLabel = R.string.order_agree,
                    // ══════════════════════════════════════════════════
                    // **والرفض في الحالين** — (قرار المالك ٢٠٢٦-٠٨-١٢)
                    // ══════════════════════════════════════════════════
                    //
                    // **ومعناه يختلف**: في «بالدور» ينقل الدور فورا إلى
                    // من بعده، **وفي «للجميع» يخفيه عن شاشته ويبقى
                    // لغيره** — فلا يقرأ في كلّ تحديث طلبا لا يريده.
                    onDecline = { actions.decline(order.id) },
                    // **وانقضاء المهلة يعيد القراءة** — البطاقة لم تعد
                    // له، **ومن أبقاها** جعله يضغطها فيُردّ «سبقك غيرك».
                    onExpired = actions.refresh,
                )
            }
        }

        item {
            SectionTitle(stringResource(R.string.orders_mine), Modifier.padding(top = 24.dp))
        }
        if (state.mine.isEmpty()) {
            item { Empty(stringResource(R.string.orders_no_mine)) }
        } else {
            items(state.mine, key = { it.id }) { order ->
                OrderCard(
                    order = order,
                    offer = false,
                    avgSpeedKmh = state.me?.avgSpeedKmh ?: 0,
                    busy = false,
                    enabled = true,
                    // **وزرّ «ابدأ الرحلة» هو الفعل الوحيد هنا** — لا
                    // «تفاصيل» ولا صفحة سطور: **الرحلة هي التفاصيل.**
                    // (مواصفة المالك ٢٠٢٦-٠٨-١٢.)
                    onAccept = { actions.startTrip(order.id) },
                    onOpen = { actions.startTrip(order.id) },
                    actionLabel = R.string.trip_start,
                )
            }
        }
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
private fun WhyNoOrders(state: OrdersState) {
    val me = state.me
    val reason: Pair<Int, Boolean>? = when {
        me == null -> null
        !me.onShift -> R.string.why_shift_closed to true
        !state.locationOn -> R.string.why_location_off to true
        me.cashLimit > 0 && me.cashHeld >= me.cashLimit -> R.string.why_cash_full to true
        me.maxActiveOrders > 0 && me.activeOrders >= me.maxActiveOrders ->
            R.string.why_too_many to true

        else -> R.string.why_all_good to false
    }

    if (reason == null) {
        Empty(stringResource(R.string.orders_no_offers))
        return
    }

    val (text, blocking) = reason
    Box(
        Modifier
            .fillMaxWidth()
            .padding(top = 12.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(if (blocking) Color(0xFFFFF4E5) else Color(0xFFF5F7F8))
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(text),
            color = if (blocking) BrandOrange else InkMuted,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun SectionTitle(text: String, modifier: Modifier = Modifier) {
    Text(
        text = text,
        modifier = modifier.fillMaxWidth(),
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Bold,
    )
}

@Composable
private fun Empty(text: String) {
    Text(
        text = text,
        color = InkMuted,
        modifier = Modifier.fillMaxWidth().padding(vertical = 14.dp),
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
    onDecline: (() -> Unit)? = null,
    onExpired: () -> Unit = {},
) {
    Column(
        Modifier
            .fillMaxWidth()
            .padding(top = 10.dp)
            .clip(RoundedCornerShape(18.dp))
            .border(1.dp, Color(0xFFE3E8EB), RoundedCornerShape(18.dp))
            .background(Color.White)
            .then(if (onOpen != null) Modifier.clickable(onClick = onOpen) else Modifier)
            .padding(16.dp),
    ) {
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
                    Chip(R.drawable.ic_bell, stringResource(R.string.card_new_order))
                } else {
                    Text(
                        text = statusText(order.status),
                        color = BrandTeal,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                val expiresAt = order.offerExpiresAt
                if (offer && expiresAt != null) {
                    Spacer(Modifier.height(4.dp))
                    Countdown(expiresAt, onExpired = onExpired)
                }
            }

            // ── وسط: المتجر ──
            //
            // **والأيقونةُ يمينَ الاسم** — (تصحيح المالك ٢٠٢٦-٠٨-١٢).
            // **وأوّلُ الصفّ في العربيّة يمينُه**، فتُكتب قبله.
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    painter = painterResource(R.drawable.ic_store),
                    // **ويُسمّى للقارئ الصوتيّ** — ومن يقود ويسمع لا يرى.
                    contentDescription = stringResource(R.string.card_store_cd),
                    tint = BrandOrange,
                    modifier = Modifier.size(22.dp),
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = order.merchantName.ifBlank { stringResource(R.string.card_custom) },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
            }

            // ── يسار ──
            Column(horizontalAlignment = Alignment.End) {
                Text(
                    text = "#" + order.number,
                    color = InkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.height(2.dp))
                // ══════════════════════════════════════════════════════
                // **ورقم بلا اسمه يُقرأ خطأ**
                // ══════════════════════════════════════════════════════
                //
                // (قرار المالك ٢٠٢٦-٠٨-١٢: «جنب السعر الإجماليّ يجب أن
                //  نكتب إجماليّ الفاتورة».)
                //
                // **ثلاثة أرقام في البطاقة**: الإجماليّ وأجرتُه وما
                // يقبضه. **ورقم عارٍ في زاوية** يُقرأ أجرةً — **فيفرح
                // بمئتين وهي للمتجر**، أو يقبض مئةً والفاتورة ثلاثمئة.
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = stringResource(R.string.card_invoice_total),
                        color = InkMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Spacer(Modifier.size(6.dp))
                    Text(
                        text = money(order.total),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                        // **ولونه يقول أمقبوضٌ أم لا** — أحمر: اقبض،
                        // أخضر: مدفوع بالمحفظة.
                        color = if (order.cashDue > 0) StateRed else StateGreen,
                    )
                }
            }
        }

        Spacer(Modifier.height(12.dp))
        Spacer(Modifier.height(12.dp))
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
        Leg(
            label = stringResource(
                R.string.card_dist_to,
                order.merchantName.ifBlank { stringResource(R.string.card_custom) },
            ),
            value = order.pickupAddress,
            far = if (order.toPickupM >= 0) distance(order.toPickupM) else "",
            // **والزمن بجانب المسافة** — (قرار المالك ٢٠٢٦-٠٨-١٢).
            //
            // **والمسافة وحدها لا تقرّر**: «٣ كم» تعني ربع ساعة في زحمة
            // ودقيقتين على طريق فارغ، **ومن يوازن بين طلبين يوازن
            // بالوقت** لا بالمتر.
            mins = legEta(order.toPickupM, avgSpeedKmh),
            tint = BrandTeal,
        )
        Leg(
            // **واسمُ الزبون بين القوسين** — (تصحيح المالك ٢٠٢٦-٠٨-١٢).
            // **وطلبُ الزائر لا اسمَ له**، فيُكتب «الزبون» ولا يُترك
            // قوسان فارغان يظنّهما صاحبُهما عطبا.
            label = stringResource(
                R.string.card_dist_to,
                order.customerName.ifBlank { stringResource(R.string.card_customer) },
            ),
            value = order.addressText,
            far = if (order.legM >= 0) distance(order.legM) else "",
            mins = legEta(order.legM, avgSpeedKmh),
            tint = BrandOrange,
        )

        Spacer(Modifier.height(10.dp))
        // **والصناديق متساوية الارتفاع** — `IntrinsicSize.Min` تقيسها
        // بأطولها، **ولولاها لطال صندوقٌ سطراه** وقصر جاراه.
        Row(
            Modifier.fillMaxWidth().height(IntrinsicSize.Min),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Metric(
                // **وأيقونةُ نقدٍ لا علامةُ دولار** — (تصحيح المالك
                // ٢٠٢٦-٠٨-١٢): **رمز عملة أجنبيّة على مبلغ بالليرة**
                // يقرؤه صاحبه لحظةً بغير ما هو.
                icon = R.drawable.ic_cash,
                label = stringResource(R.string.card_fee),
                value = money(order.deliveryFee),
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
                icon = if (cash) R.drawable.ic_cash else R.drawable.ic_wallet,
                label = stringResource(R.string.card_payment),
                value = stringResource(if (cash) R.string.card_cash else R.string.card_wallet),
                tone = if (cash) StateRed else StateGreen,
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
        // **زرّان متساويان — أخضر وأحمر**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرار المالك ٢٠٢٦-٠٨-١٢: «موافق أخضر ورفض أحمر، وبنفس الحجم
        //  والشكل، ورفض أيضا أيقونة بداخله».)
        //
        // **ومتساويان لأنّ القرارين متساويان**: زرّ أكبر يقول «اضغطني»،
        // **ورفضٌ باهت يجعل من لا يريد الطلب يقبله** ليمضي.
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Button(
                onClick = onAccept,
                enabled = enabled && !busy,
                colors = ButtonDefaults.buttonColors(containerColor = StateGreen),
                modifier = Modifier.weight(1f),
            ) {
                if (busy) {
                    CircularProgressIndicator(
                        Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = Color.White,
                    )
                } else {
                    Icon(
                        painter = painterResource(R.drawable.ic_check_circle),
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.size(6.dp))
                    Text(stringResource(actionLabel))
                }
            }
            if (onDecline != null) {
                Button(
                    onClick = onDecline,
                    enabled = enabled && !busy,
                    colors = ButtonDefaults.buttonColors(containerColor = StateRed),
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_close_circle),
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.size(6.dp))
                    Text(stringResource(R.string.order_decline))
                }
            }
        }
    }
}

/**
 * **ما بقي من مهلة العرض** — يُعدّ كلّ ثانية.
 *
 * **ويحمرّ في آخر عشر ثوان** — القرار صار عاجلا، **ولون واحد طوال
 * المهلة لا يقول ذلك.**
 *
 * **وتاريخ لا يُقرأ لا يُعرض** — المحرّك يرسل `ISO-8601`، **ومن سقط
 * تحليله** يُترك السطر فارغا بدل «صفر ثانية» كاذبة.
 */
@Composable
private fun Countdown(expiresAt: String, onExpired: () -> Unit) {
    val end = remember(expiresAt) {
        runCatching { java.time.Instant.parse(expiresAt).toEpochMilli() }.getOrNull()
    } ?: return

    var left by remember(expiresAt) { mutableStateOf(end - System.currentTimeMillis()) }
    LaunchedEffect(expiresAt) {
        while (left > 0) {
            kotlinx.coroutines.delay(1000)
            left = end - System.currentTimeMillis()
        }
        onExpired()
    }

    if (left <= 0) {
        Text(stringResource(R.string.offer_expired), color = InkMuted)
        return
    }

    val seconds = (left / 1000).toInt()
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(R.drawable.ic_time),
            contentDescription = null,
            tint = if (seconds <= 10) BrandOrange else BrandTeal,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = stringResource(R.string.offer_left, "%d:%02d".format(seconds / 60, seconds % 60)),
            color = if (seconds <= 10) BrandOrange else BrandTeal,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

/** **شارة صغيرة** — أيقونة وكلمة. */
@Composable
private fun Chip(icon: Int, text: String) {
    Row(
        Modifier
            .clip(RoundedCornerShape(20.dp))
            .background(BrandTeal)
            .padding(horizontal = 10.dp, vertical = 5.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.size(5.dp))
        Text(
            text = text,
            color = Color.White,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Bold,
        )
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
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.size(8.dp))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
                if (far.isNotEmpty()) {
                    Spacer(Modifier.size(8.dp))
                    // **والمسافة في سطر عنوانها** — رقم قصير لا يزاحم
                    // العنوان، **ويُقرأ معه لا بعد أن يبحث عنه.**
                    Text(
                        // **والرقم بين قوسين** — (طلب المالك): يُقرأ
                        // رقما مستقلّا لا امتدادا للاسم قبله.
                        text = "(" + far + ")",
                        color = tint,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (mins > 0) {
                    Spacer(Modifier.size(6.dp))
                    // **والزمن باهت والمسافة ملوّنة** — خبران في سطر،
                    // **ولو تساويا في الصياح** لم يُقرأ أحدهما أوّلا.
                    Text(
                        text = "(" + minutesText(mins) + ")",
                        color = InkMuted,
                        style = MaterialTheme.typography.bodySmall,
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
                    style = MaterialTheme.typography.bodyMedium,
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
) {
    val ink = tone ?: if (strong) BrandTeal else MaterialTheme.colorScheme.onSurface
    Column(
        modifier
            .clip(RoundedCornerShape(12.dp))
            // **وخلفيّة باهتة لا صمّاء** — اللون يُقرأ ولا يصرخ.
            .background(tone?.copy(alpha = 0.10f) ?: Color(0xFFF5F7F8))
            .padding(vertical = 10.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = tone ?: if (strong) BrandTeal else InkMuted,
                modifier = Modifier.size(14.dp),
            )
            Spacer(Modifier.size(4.dp))
            Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(2.dp))
        // ══════════════════════════════════════════════════════════════
        // **ولا تُبتر القيمة في سطر** — (تصحيح المالك ٢٠٢٦-٠٨-١٢:
        // «كلمة عند الاستلام طلعت ناقصة، بس طلعت عند»)
        // ══════════════════════════════════════════════════════════════
        //
        // **و«عند» وحدها أسوأ من لا شيء**: تقرأ نصف الجملة **فتظنّ أنّك
        // قرأتها**، ولا شيء في الشاشة يقول إنّ بقيّتها سقطت.
        Text(
            text = value,
            fontWeight = FontWeight.Bold,
            color = ink,
            maxLines = 2,
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.bodyMedium,
        )
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
private fun minutesText(mins: Long): String = "$mins دقيقة"

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
private fun distance(meters: Double): String {
    val m = meters.toLong()
    return if (m < 1000) "$m م" else "${"%.1f".format(m / 1000.0)} كم"
}

/**
 * **حال الطلب بالعربيّة.**
 *
 * **ومجهول يُعرض رمزه** — لا يُبتلع في «قيد التنفيذ»: **حال جديد في
 * المحرّك يظهر هنا فيُعرف ويُترجم**، بدل أن يختفي تحت كلمة عامّة.
 */
@Composable
private fun statusText(status: String): String = when (status) {
    "assigned" -> stringResource(R.string.status_assigned)
    "picked_up" -> stringResource(R.string.status_picked_up)
    "on_the_way" -> stringResource(R.string.status_on_the_way)
    "arrived" -> stringResource(R.string.status_arrived)
    "dispatching" -> stringResource(R.string.status_dispatching)
    else -> status
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
)

data class OrdersActions(
    val accept: (String) -> Unit,
    val decline: (String) -> Unit,
    /** **يفتح رحلته** — يختار الطلب وينتقل إلى الخريطة. */
    val startTrip: (String) -> Unit,
    val refresh: () -> Unit,
)
