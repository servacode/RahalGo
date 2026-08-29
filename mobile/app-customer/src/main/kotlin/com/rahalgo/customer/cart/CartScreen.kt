package com.rahalgo.customer.cart

import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.border
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.material3.RadioButton
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.draw.clip
import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.Here
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CartLine
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.NewOrder
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.Quote
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Card
import com.rahalgo.ui.KeyValue
import com.rahalgo.shared.model.Address
import com.rahalgo.shared.customer.PromoPreview
import com.rahalgo.ui.Flash
import com.rahalgo.ui.AddressCard
import com.rahalgo.ui.DeliveryAddress
import com.rahalgo.ui.LastPoint
import com.rahalgo.ui.Note
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.apiError
import com.rahalgo.ui.money
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import kotlinx.coroutines.launch
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.ui.res.painterResource
import com.rahalgo.ui.Tone
import androidx.compose.foundation.layout.Box

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السلّة والدفع — والسعرُ من الخادم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وحسبةٌ في الجهاز تفترق عمّا يُقيَّد في الطلب** — فيرى سعراً
 * ويُحاسَب بآخر، **وهي أسرعُ طريقةٍ لكسر الثقة.**
 *
 * **فالمجموعُ يُعرض تقريباً حتّى تصل التسعيرة**، **والتوصيلُ والإجماليُّ
 * لا يُعرضان إلّا منها**: **رقمٌ مخمَّنٌ للتوصيل أسوأُ من لا رقم.**
 */
@Composable
fun CartScreen(
    vm: CartViewModel,
    /**
     * **عنوانُ التوصيل المختار** — الافتراضيُّ في حسابه.
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نغيّر زرَّ العنوان أيضاً بصفحة سلّتي
     *  بنفس الطريقة».)
     */
    address: Address?,
    onDone: () -> Unit,
) {
    val here = address?.let { LastPoint.Point(it.lat, it.lng, it.text) }
    val context = androidx.compose.ui.platform.LocalContext.current
    val media = { path: String? -> com.rahalgo.customer.Backend.of(context).media(path) }

    // **والتسعيرةُ تُطلب متى تبدّلت السلّةُ أو النقطة** — لا عند الضغط
    // وحدَه: **من رأى الإجماليَّ لحظةَ الدفع فوجده أكبرَ تردّد.**
    LaunchedEffect(Cart.lines, here) { vm.quote(here?.lat, here?.lng) }
    LaunchedEffect(Cart.lines) { vm.loadSuggestions() }

    if (Cart.lines.isEmpty()) {
        Screen {
            ScreenTitle(stringResource(R.string.cart_title), "")
            com.rahalgo.ui.Empty(stringResource(R.string.cart_empty))
        }
        return
    }

    Screen {
        ScreenTitle(stringResource(R.string.cart_title), stringResource(R.string.cart_hint))

        // **والخطأُ لا يُرسم هنا** — يُرسم فوق زرّ الطلب. انظر أدناه.
        Spacer(Modifier.height(10.dp))
        Card {
            Cart.lines.forEach { line ->
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(line.item.name, style = MaterialTheme.typography.bodyMedium)
                        // **وما اختاره يُقال تحت اسمه** — **وسطران
                        // متطابقان في السلّة بلا فرقٍ مكتوبٍ يُقرآن
                        // تكراراً**، فيُحذف أحدُهما وهو المقصود.
                        if (line.options.isNotEmpty()) {
                            Text(
                                line.options.joinToString("، ") { it.name },
                                color = Rahal.colors.inkMuted,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Text(
                            money(line.unitPrice * line.qty),
                            color = Rahal.colors.brand,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    // **والعدُّ يُبدَّل هنا** — ومن أراد صنفين لا يعود
                    // إلى السوق ليضغط مرّتين.
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        RahalTextButton(onClick = { Cart.setQty(line.key, line.qty - 1) }) {
                            Text("−", style = MaterialTheme.typography.titleLarge)
                        }
                        Text(line.qty.toString(), fontWeight = FontWeight.Bold)
                        RahalTextButton(onClick = { Cart.setQty(line.key, line.qty + 1) }) {
                            Text("+", style = MaterialTheme.typography.titleLarge)
                        }

                        // ══════════════════════════════════════════════
                        // **وسلّةٌ يُحذف منها بضغطةٍ لا بالإنقاص**
                        // ══════════════════════════════════════════════
                        //
                        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «بسلّتي أضف أيقونةَ
                        //  حذفٍ لحذف عناصر السلّة وإفراغها».)
                        //
                        // **وكان الحذفُ بالإنقاص حتّى الصفر** — ومن
                        // أضاف خمسةً يضغط خمساً ليُلغيها، **وضغطةٌ
                        // تُعاد خمساً تُقرأ عناداً في التطبيق.**
                        Spacer(Modifier.size(4.dp))
                        Box(
                            Modifier
                                .clip(CircleShape)
                                .clickable { Cart.setQty(line.key, 0) }
                                .padding(6.dp),
                        ) {
                            Icon(
                                painter = painterResource(
                                    com.rahalgo.ui.R.drawable.ic_trash,
                                ),
                                contentDescription = stringResource(R.string.cart_remove),
                                tint = Rahal.colors.danger,
                                modifier = Modifier.size(18.dp),
                            )
                        }
                    }
                }
                HorizontalDivider()
            }
        }

        // **«يُطلب معه» تحت السطور مباشرةً** — انظر `SuggestRow`.
        //
        // **وقبل العنوانِ والدفع**: من بلغ زرَّ الإرسال قرّر، **واقتراحٌ
        // تحت الزرّ لا يُرى.**
        if (vm.suggested.isNotEmpty()) {
            Spacer(Modifier.height(14.dp))
            SuggestRow(items = vm.suggested, media = media, onAdd = { Cart.add(it) })
        }

        // ══════════════════════════════════════════════════════════════
        // **والعنوانُ من حسابه لا من حقلٍ يُملأ كلَّ مرّة**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨.) **والبابُ هو بابُ الشريط نفسُه** —
        // **ولوحتان تفترقان يومَ تُزاد فيهما ميزة.**
        Spacer(Modifier.height(12.dp))
        AddressCard(address) { DeliveryAddress.open() }

        // ══════════════════════════════════════════════════════════════
        // **وكودُ الخصم يُطبَّق بزرٍّ ويُرى أثرُه فورا**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «كودُ الخصم أضف إليه زرَّ تطبيق…
        //  بحيث النتيجةُ تظهر بشكلٍ فوريٍّ للمستخدم».)
        //
        // **وكان يُرسَل مع الطلب وحدَه** — **فيعرف أثرَه بعد أن يطلب**،
        // ومن كتب كوداً منتهياً دفع ثمناً ظنّه أقلّ.
        Spacer(Modifier.height(10.dp))
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedTextField(
                value = vm.promo,
                onValueChange = vm::typePromo,
                label = { Text(stringResource(R.string.cart_promo)) },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            RahalButton(
                onClick = vm::applyPromo,
                enabled = !vm.promoBusy && vm.promo.isNotBlank(),
            ) { Text(stringResource(R.string.cart_promo_apply)) }
        }

        // **والنتيجةُ تحت الحقل** — **ورسالةٌ تطفو تنصرف بعد ثوانٍ**،
        // وهذه تبقى ما دام الكودُ مكتوبا.
        vm.promoResult?.let { r ->
            Spacer(Modifier.height(6.dp))
            Text(
                text = if (r.valid) {
                    stringResource(R.string.cart_promo_ok, money(r.discount))
                } else {
                    stringResource(R.string.cart_promo_bad)
                },
                color = if (r.valid) Rahal.colors.success else Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والدفعُ نقدٌ أو محفظة — ولا خليط**
        // ══════════════════════════════════════════════════════════════
        //
        // **وخليطٌ يعني رقمين يُتابَعان في دفترين** — قرارُ المالك
        // بحذفه قائم.
        // **وطريقةُ الدفع تُقرأ اختياراً** — (طلبُ المالك
        // ٢٠٢٦-٠٨-١٨: «نقداً عند التسليم ومن محفظتي، اجعلها واضحةً
        // أنّها أزرارُ اختيار»).
        //
        // **والرقاقةُ تُقرأ وسماً لا زرّا**: صغيرةٌ رماديّةٌ بلا دائرة،
        // **فلا يُعرف أنّ فيها خياراً حتّى تُضغط بالصدفة.**
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            PayChoice(
                text = stringResource(R.string.cart_cash),
                on = !vm.wallet,
                modifier = Modifier.weight(1f),
            ) { vm.wallet = false }
            PayChoice(
                text = stringResource(R.string.cart_wallet),
                on = vm.wallet,
                modifier = Modifier.weight(1f),
            ) { vm.wallet = true }
        }

        // **وإفراغُ السلّة كلِّها** — (طلبُ المالك ٢٠٢٦-٠٨-١٨).
        //
        // **ونصٌّ لا زرٌّ مملوء**: فعلٌ يُندَم عليه لا يُوضع في زرٍّ
        // يلمع، **ومن أفرغها بالخطأ يعيد بناءها صنفاً صنفا.**
        if (Cart.lines.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            RahalTextButton(onClick = { Cart.clear() }, tone = Tone.Danger) {
                Text(stringResource(R.string.cart_clear))
            }
        }

        Spacer(Modifier.height(14.dp))
        Card {
            KeyValue(stringResource(R.string.ord_subtotal), money(Cart.subtotal))
            val q = vm.priced
            if (q == null) {
                // **ولا يُخمَّن التوصيل** — رقمٌ مخمَّنٌ أسوأُ من لا رقم.
                //
                // **والنصُّ يقول ما ينقص** — (شكوى المالك ٢٠٢٦-٠٨-١٨:
                // «تفاجأتُ بشيءٍ يقول حدّد موقعك ليُحسب التوصيل»):
                // **والموقعُ لم يعد يُلتقط هنا** — العنوانُ المحفوظ يحمل
                // نقطتَه، **فالناقصُ عنوانٌ لا موقع.**
                Text(
                    text = stringResource(R.string.cart_need_address),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            } else {
                // ══════════════════════════════════════════════════════
                // **والحسمُ يُطرح من الإجماليّ فورَ تطبيقه**
                // ══════════════════════════════════════════════════════
                //
                // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «الحسمُ وقتَ الكود يجب أن
                //  يظهر مباشرةً على السعر بالسلّة أيضاً».)
                //
                // **وكان يُعرض سطراً تحت الحقل ولا يمسّ الإجماليّ** —
                // **فيقرأ الزبونُ «خصم ٥٠٠» ثمّ يرى الإجماليَّ كما هو**،
                // فلا يعرف أوقع الخصمُ أم لا.
                //
                // **والتوصيلُ يتبع الكودَ أيضاً**: كودٌ يُلغي الأجرة
                // يردّها صفراً في `deliveryFee` — **وقراءةُ الخصم
                // وحدَه تُخفي أثرَه.**
                val cut = vm.promoResult?.takeIf { it.valid }
                val fee = cut?.deliveryFee ?: q.deliveryFee
                KeyValue(stringResource(R.string.ord_delivery), money(fee))
                if (cut != null && cut.discount > 0) {
                    KeyValue(
                        stringResource(R.string.ord_discount),
                        "− " + money(cut.discount),
                        valueColor = Rahal.colors.success,
                    )
                }
                KeyValue(
                    stringResource(R.string.ord_total),
                    money(Cart.subtotal + fee - (cut?.discount ?: 0)),
                    valueColor = Rahal.colors.brand,
                )
            }
        }

        // **وخارجَ النطاق يُقال قبل الضغط** — لا بعد أن يملأ كلَّ شيء.
        if (vm.priced?.outOfZone == true) {
            Spacer(Modifier.height(8.dp))
            Note(stringResource(R.string.cart_out_of_zone), Rahal.colors.danger)
        }

        // ══════════════════════════════════════════════════════════════
        // **والخطأُ يُقال حيث تقع العين — فوق الزرّ**
        // ══════════════════════════════════════════════════════════════
        //
        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٥: «رسالة رصيدك لا يكفي عند الدفع
        //  بالمحفظة تظهر بمكانٍ لا يراه المستخدم، فلا يعرف سببَ عدم
        //  اكتمال الطلب».)
        //
        // **وكان يُرسم تحت عنوان الشاشة** — والزرُّ في أسفلها بعد
        // الأصناف والعنوان والدفع. **فيضغط الزرَّ فتظهر الرسالةُ فوق
        // شاشتين، ولا يرى إلّا زرّاً لم يفعل شيئا.**
        //
        // **وخطأٌ لا يُرى كأنّه لم يقع** — والمستخدمُ يعيد الضغطَ ثمّ
        // يخرج.
        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Note(vm.error, Rahal.colors.danger)
        }

        Spacer(Modifier.height(14.dp))
        RahalButton(
            onClick = {
                address?.let {
                    vm.send(
                        it.text, it.lat, it.lng,
                        if (vm.wallet) "wallet" else "cash", vm.promo.trim(), onDone,
                    )
                }
            },
            enabled = !vm.busy && address != null &&
                vm.priced != null && vm.priced?.outOfZone != true,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (vm.busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(R.string.cart_send))
            }
        }
        Spacer(Modifier.height(28.dp))
    }
}

class CartViewModel(app: Application) : AndroidViewModel(app) {

    private val api = CustomerApi(AppCore.get().api)

    // ══════════════════════════════════════════════════════════════════
    // **وما كُتب في السلّة يبقى فيها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «وقت تفوت على سلّتك وتضغط على أيّ
    //  عنصرٍ آخر يبقى بسلّتك ولا يخرج — وقبلُ عانينا من هذا الموضوع
    //  بباقي الأقسام».)
    //
    // **والسلّةُ نفسُها كانت تبقى** — `Cart` شيءٌ عامٌّ في الذاكرة.
    // **وما كُتب فيها لا**: العنوانُ والرمزُ وطريقةُ الدفع كانت
    // `rememberSaveable` **داخل شاشةٍ تُهدَم حين يُضغط تبويبٌ آخر** —
    // **ورمزُ حفظها يُطرح معها.**
    //
    // **فمن كتب عنوانَه ثمّ خرج لينظر في صنفٍ رجع إلى حقلٍ فارغ.**
    //
    // **وموضعُها النموذجُ لا الشاشة** — يعيش ما دام التطبيقُ حيّا.
    var address by mutableStateOf("")
    var promo by mutableStateOf("")

    /**
     * **أثرُ الكود بعد تطبيقه** — وفارغٌ يعني «لم يُطبَّق بعد».
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨.)
     *
     * **ولا يُطبَّق مع كلّ حرفٍ يُكتب** — **ونداءٌ في كلّ ضغطةِ لوحةٍ
     * يُغرق الخادمَ ويومض النتيجةَ**: يكتب أربعةَ أحرفٍ فيرى «غير صالح»
     * أربعَ مرّات قبل أن يُتمّ.
     */
    var promoResult by mutableStateOf<PromoPreview?>(null)
        private set

    var promoBusy by mutableStateOf(false)
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **مفتاحُ المحاولة — يبقى ما دامت لم تنجح**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تدقيقُ الإطلاق ٢٠٢٦-٠٨-١٩ — BUG-001.)
     *
     * **والخطرُ ليس الضغطتين المتتاليتين** — تلك يمنعها `busy`.
     * **الخطرُ أن يُنشأ الطلبُ في الخادم ثمّ تنقطع الشبكةُ قبل الردّ**:
     * يرى «تعذّر» فيضغط ثانيةً — **فطلبان وسائقان وخصمان.**
     *
     * **فيُولَّد مرّةً ويبقى حتّى ينجح** — فإعادةُ المحاولة تحمله نفسَه
     * فيردّ الخادمُ الطلبَ الأوّلَ بعينه (`idempotency.go`).
     *
     * **ويُمحى بعد النجاح** — **ومفتاحٌ يبقى يجعل الطلبَ التالي يردّ
     * جوابَ الذي قبله.**
     */
    private var attemptKey: String? = null


    /** **ويُنسى الأثرُ حين يُبدَّل الكود** — نتيجةُ كودٍ على كودٍ آخرَ كذب. */
    fun typePromo(v: String) {
        promo = v
        promoResult = null
    }

    fun applyPromo() {
        val code = promo.trim()
        if (code.isEmpty() || promoBusy) return
        promoBusy = true
        viewModelScope.launch {
            runCatching {
                api.previewPromo(code, Cart.subtotal, priced?.deliveryFee ?: 0)
            }.onSuccess { promoResult = it }
                .onFailure { Flash.fail(apiError(getApplication(), it as Exception)) }
            promoBusy = false
        }
    }
    var wallet by mutableStateOf(false)

    var priced by mutableStateOf<Quote?>(null)
        private set

    var busy by mutableStateOf(false)
        private set

    var error by mutableStateOf("")
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **«يُطلب معه» — مشروبٌ ومقبّلاتٌ وحلوى**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-٢٢: «حركةٌ ذكيّةٌ بالأكل تعرض عليه كولا أو
     *  عيران، هي أكثرُ شيءٍ تنطلب».)
     *
     * **وصنفٌ مستقلٌّ لا خيارٌ ملصوق**: من طلب ثلاثةَ ساندويشات وكولتين
     * **لا يستطيع قولَها بالخيارات** — إمّا كولا لكلٍّ أو لا شيء.
     */
    var suggested by mutableStateOf<List<Item>>(emptyList())
        private set

    /**
     * **تُنادى متى تبدّلت السلّة** — **وما فيها يُستثنى**: اقتراحُ ما
     * اشتراه يقول له إنّا لا نقرأ سلّته.
     *
     * **وإخفاقُها لا يُقال ولا يُسجَّل خطأً في الشاشة** — **هي زيادةٌ
     * لا ركن**، ورسالةُ عطبٍ لأجل اقتراحٍ تُقلق بلا سبب.
     */
    fun loadSuggestions() {
        if (Cart.lines.isEmpty()) {
            suggested = emptyList()
            return
        }
        viewModelScope.launch {
            runCatching { api.suggest(Cart.lines.map { it.item.id }.distinct()) }
                .onSuccess { suggested = it.items.filter { i -> i.available && !i.sourceClosed } }
                .onFailure { suggested = emptyList() }
        }
    }

    fun locate() = Here.refresh(getApplication())

    fun quote(lat: Double?, lng: Double?) {
        if (lat == null || lng == null || Cart.lines.isEmpty()) {
            priced = null
            return
        }
        viewModelScope.launch {
            runCatching {
                api.quote(Cart.lines.map { it.toPayload() }, lat, lng)
            }.onSuccess { priced = it; error = "" }
                .onFailure { error = apiError(getApplication(), it as Exception) }
        }
    }

    fun send(
        address: String,
        lat: Double,
        lng: Double,
        payment: String,
        promo: String,
        onDone: () -> Unit,
    ) {
        if (busy) return
        busy = true
        error = ""
        // **ولا يُولَّد إن كان قائماً** — محاولةٌ ثانيةٌ لطلبٍ واحد.
        val key = attemptKey ?: java.util.UUID.randomUUID().toString().also { attemptKey = it }
        viewModelScope.launch {
            try {
                api.createOrder(
                    NewOrder(
                        items = Cart.lines.map { it.toPayload() },
                        addressText = address,
                        lat = lat,
                        lng = lng,
                        paymentMethod = payment,
                        promoCode = promo,
                    ),
                    attemptKey = key,
                )
                // **والمفتاحُ يُمحى بعد النجاح** — انظر `attemptKey`.
                attemptKey = null
                // **والسلّةُ تُفرَغ بعد أن يُقيَّد الطلبُ لا قبله** —
                // **ومن فرّغها قبل الجواب خسر سلّةَ من سقط نداؤه.**
                Cart.clear()
                onDone()
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}

/**
 * **خيارُ دفعٍ — دائرةٌ ونصٌّ في إطار.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «اجعلها واضحةً أنّها أزرارُ اختيار».)
 *
 * **والدائرةُ هي التي تقول «اختر واحدا»** — إطارٌ ملوّنٌ وحدَه يُقرأ
 * زرَّ فعلٍ فيُضغط ظنّاً أنّه يدفع.
 *
 * **وكلاهما بعرضٍ واحد** — **وخيارٌ أعرضُ من أخيه يُقرأ الموصى به.**
 */
@Composable
private fun PayChoice(
    text: String,
    on: Boolean,
    modifier: Modifier = Modifier,
    onPick: () -> Unit,
) {
    val tint = if (on) Rahal.colors.brand else Rahal.colors.line
    Row(
        modifier
            .clip(Rahal.shape.md)
            .border(if (on) 2.dp else 1.dp, tint, Rahal.shape.md)
            .background(if (on) Rahal.colors.brand.copy(alpha = 0.06f) else Color.Transparent)
            .clickable(onClick = onPick)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = on, onClick = onPick)
        Text(
            text = text,
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (on) FontWeight.Bold else FontWeight.Normal,
        )
    }
}
