package com.rahalgo.customer.cart

import android.app.Application
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import kotlinx.coroutines.launch

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

    // **والتسعيرةُ تُطلب متى تبدّلت السلّةُ أو النقطة** — لا عند الضغط
    // وحدَه: **من رأى الإجماليَّ لحظةَ الدفع فوجده أكبرَ تردّد.**
    LaunchedEffect(Cart.lines, here) { vm.quote(here?.lat, here?.lng) }

    if (Cart.lines.isEmpty()) {
        Screen {
            ScreenTitle(stringResource(R.string.cart_title), "")
            com.rahalgo.ui.Empty(stringResource(R.string.cart_empty))
        }
        return
    }

    Screen {
        ScreenTitle(stringResource(R.string.cart_title), stringResource(R.string.cart_hint))

        if (vm.error.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            Note(vm.error, Rahal.colors.danger)
        }

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
                        Text(
                            money(line.item.price * line.qty),
                            color = Rahal.colors.brand,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    // **والعدُّ يُبدَّل هنا** — ومن أراد صنفين لا يعود
                    // إلى السوق ليضغط مرّتين.
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        TextButton(onClick = { Cart.setQty(line.item.id, line.qty - 1) }) {
                            Text("−", style = MaterialTheme.typography.titleLarge)
                        }
                        Text(line.qty.toString(), fontWeight = FontWeight.Bold)
                        TextButton(onClick = { Cart.setQty(line.item.id, line.qty + 1) }) {
                            Text("+", style = MaterialTheme.typography.titleLarge)
                        }
                    }
                }
                HorizontalDivider()
            }
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
            Button(
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
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(
                selected = !vm.wallet,
                onClick = { vm.wallet = false },
                label = { Text(stringResource(R.string.cart_cash)) },
            )
            FilterChip(
                selected = vm.wallet,
                onClick = { vm.wallet = true },
                label = { Text(stringResource(R.string.cart_wallet)) },
            )
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
                KeyValue(stringResource(R.string.ord_delivery), money(q.deliveryFee))
                KeyValue(
                    stringResource(R.string.ord_total),
                    money(q.total),
                    valueColor = Rahal.colors.brand,
                )
            }
        }

        // **وخارجَ النطاق يُقال قبل الضغط** — لا بعد أن يملأ كلَّ شيء.
        if (vm.priced?.outOfZone == true) {
            Spacer(Modifier.height(8.dp))
            Note(stringResource(R.string.cart_out_of_zone), Rahal.colors.danger)
        }

        Spacer(Modifier.height(14.dp))
        Button(
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

    fun locate() = Here.refresh(getApplication())

    fun quote(lat: Double?, lng: Double?) {
        if (lat == null || lng == null || Cart.lines.isEmpty()) {
            priced = null
            return
        }
        viewModelScope.launch {
            runCatching {
                api.quote(Cart.lines.map { CartLine(it.item.id, it.qty) }, lat, lng)
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
        viewModelScope.launch {
            try {
                api.createOrder(
                    NewOrder(
                        items = Cart.lines.map { CartLine(it.item.id, it.qty) },
                        addressText = address,
                        lat = lat,
                        lng = lng,
                        paymentMethod = payment,
                        promoCode = promo,
                    ),
                )
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
