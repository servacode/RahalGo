package com.rahalgo.customer.orders

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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.ui.Since
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.MyOrder
import com.rahalgo.ui.Card
import com.rahalgo.ui.Chip
import com.rahalgo.ui.KeyValue
import com.rahalgo.ui.Stages
import com.rahalgo.ui.money
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.RahalOutlineButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ طلبٍ — كلُّ شيءٍ فيها ولا صفحةَ ثانية**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك في الويب ٢٠٢٦-٠٨-١٠: «الفاتورة تصير بنفس الكرت، بالكرت
 *  كلُّ صنفٍ وسعره في حقلٍ خاصٍّ مرتّب».)
 *
 * **وصفحةٌ ثانيةٌ لكلّ طلبٍ تعني خروجاً وعودة** — ومن يتابع طلبَه يفتح
 * الشاشةَ عشرَ مرّاتٍ في نصف ساعة.
 */
@Composable
fun OrderCard(
    order: MyOrder,
    onCancel: (() -> Unit)?,
    onComplain: (() -> Unit)?,
    onRate: (() -> Unit)?,
    /**
     * **حديثُ الطلب مع السائق** — وفارغُه لا سائقَ بعد.
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٨: «أيقونةُ الدردشة لم تظهر عند الزبون،
     *  تأكّد منها أنّ السائق والزبون يستطيعون التحدّث فيها».)
     *
     * **والمحرّكُ يسمح للطرفين منذ بُني** (`/orders/{id}/messages`)
     * — **والزبونُ لم يكن له بابٌ إليه**: «دردشاتي السابقة» في القائمة
     * **تُبنى من الرسائل**، فلا تعرض طلباً لم يُكتب فيه شيءٌ بعد.
     *
     * **فمن أراد أن يبدأ لا يجد من أين** — والسائقُ عنده الحديثُ داخل
     * شاشة رحلته.
     */
    onChat: (() -> Unit)? = null,
    /**
     * **رصيدُ المحفظة** — لبوّابة أهليّة الدفع من المحفظة في تأكيد عرض
     * الطلب المخصَّص (Batch 2c). **يُمرَّر من `ShellViewModel`.**
     */
    walletBalance: Long = 0,
    /** **يؤكّد عرضَ الطلب المخصَّص بالطريقة المختارة** — و`null` لا تأكيدَ هنا. */
    onConfirmQuote: ((String) -> Unit)? = null,
    /** **جارٍ تأكيدُ عرضٍ الآن** — يُعطّل الزرَّ ويُظهر دوّارة. */
    confirming: Boolean = false,
) {
    Card {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                // **ورقمُه كما يعرفه الجميع** — لا ستّةُ أحرفٍ من
                // معرّفه: **رقمٌ لا يعرفه صاحبُه ولا المكتبُ لا يُستعمل
                // في شكوى.**
                text = "#" + order.number,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            Chip(statusText(order.status), statusColor(order.status))
        }

        // **وما ليس في المنصّة يُقال بلفظ صاحبه** — لا «طلب #12».
        if (order.kind == "custom" && order.customRequest.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(
                text = stringResource(R.string.ord_wanted) + " " + order.customRequest,
                color = Rahal.colors.ink,
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **وشريطُ المراحل يأتي من المحرّك كاملا**
        // ══════════════════════════════════════════════════════════════
        //
        // **والخاصُّ لا متجرَ له** — فلا «قيد التحضير»: يُقبل ثمّ
        // يُسنَد لسائقٍ ثمّ **يشتريه** ثمّ يمشي ثمّ يصل ثمّ يُسلّم.
        // **والمحرّكُ يعرف الفرقَ ويرسل مسارَ كلٍّ منهما.**
        //
        // **و`stage_at = -1` تعني انتهى قبل أن يصل** — وشريطٌ يقف في
        // منتصفه يُقرأ «عالق» لا «انتهى».
        if (order.stageAt >= 0 && order.stages.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            // **ودرّاجتُه تمشي ما دام جاريا** — والمنتهي لا شريطَ له.
            Stages(order.stages.map { stageLabel(it) }, order.stageAt, live = true)
        }

        // **والسائقُ يُسمّى حين يُسنَد** — ومن يعرف اسمَ من يحمل طلبَه
        // يطمئنّ.
        // **والسائقُ يُسمّى حين يُسنَد** — **ويصل `null` قبل ذلك.**
        order.driverName?.takeIf { it.isNotEmpty() && !ended(order.status) }?.let {
            Spacer(Modifier.height(10.dp))
            KeyValue(stringResource(R.string.ord_driver), it)
        }

        // ══════════════════════════════════════════════════════════════
        // **وما فيه باختصارٍ كما يرسله المحرّك**
        // ══════════════════════════════════════════════════════════════
        //
        // **والقائمةُ لا تحمل الأصنافَ صنفاً صنفا** — تحمل عدَّها
        // ومختصرَها (`items_count` و`items_preview`). **وتفصيلُها في
        // صفحة الطلب.**
        //
        // **ونداءٌ لكلّ بطاقةٍ ليُعرض تفصيلُها** يعني عشرةَ نداءاتٍ في
        // شاشةٍ واحدة.
        if (order.itemsPreview.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            HorizontalDivider()
            Spacer(Modifier.height(8.dp))
            KeyValue(
                pluralStringResource(R.plurals.ord_items_n, order.itemsCount, order.itemsCount),
                order.itemsPreview,
            )
        }

        Spacer(Modifier.height(8.dp))
        HorizontalDivider()
        Spacer(Modifier.height(8.dp))
        // ══════════════════════════════════════════════════════════════
        // **والطلبُ الخاصُّ عقدُ عرضٍ يُؤكَّد** (Batch 2c) — لا فاتورةٌ ثابتة
        // ══════════════════════════════════════════════════════════════
        //
        // **قبل الاتفاق: بانتظار التكلفة. وحين يصل العرضُ: بطاقتُه وزرُّ
        // التأكيد واختيارُ الدفع. وبعد التأكيد: الحجزُ إن كان محفظة.**
        if (order.kind == "custom") {
            CustomQuoteSection(order, walletBalance, onConfirmQuote, confirming)
        } else {
            if (order.subtotal > 0) KeyValue(stringResource(R.string.ord_subtotal), money(order.subtotal))
            KeyValue(stringResource(R.string.ord_delivery), money(order.deliveryFee))
            if (order.discount > 0) {
                KeyValue(stringResource(R.string.ord_discount), "-" + money(order.discount))
            }
            KeyValue(
                stringResource(R.string.ord_total),
                money(order.total),
                valueColor = Rahal.colors.brand,
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والبطاقةُ هي سطحُ الطلبِ الأوّل — فتحمل ما يُعرِّفه** (`CUST-14-021`،
        // قرارُ المالك §40.15-4): وقتُ الطلبِ، وطريقةُ الدفع، والعنوان.
        // ══════════════════════════════════════════════════════════════
        //
        // **بطاقةٌ بلا وقتٍ ولا دفعٍ ولا عنوانٍ تُقرأ إيصالاً ناقصاً** — ومن
        // شكا لم يعرف متى طلب ولا أين يُوصَّل ولا بمَ يدفع. **والبياناتُ في
        // النموذج أصلاً** (`createdAt`/`paymentMethod`/`addressText`) —
        // كانت تُجلَب ولا تُعرَض.
        Spacer(Modifier.height(8.dp))
        HorizontalDivider()
        Spacer(Modifier.height(8.dp))
        val ctx = LocalContext.current
        Since.text(ctx, order.createdAt).takeIf { it.isNotEmpty() }?.let {
            KeyValue(stringResource(R.string.ord_placed), it)
        }
        KeyValue(
            stringResource(R.string.ord_payment),
            stringResource(
                if (order.paymentMethod == "wallet") R.string.cart_wallet else R.string.cart_cash,
            ),
        )
        order.addressText.takeIf { it.isNotEmpty() }?.let {
            KeyValue(stringResource(R.string.ord_address), it)
        }

        // **وسببُ النهاية يُقال** — (قاعدةُ المحرّك: `cancel_reason` عند
        // كلّ نهايةٍ غير التسليم). **ومن أُلغي طلبُه بلا سببٍ يظنّ العطبَ
        // في المنصّة.**
        if (order.cancelReason.isNotEmpty()) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = order.cancelReason,
                color = Rahal.colors.danger,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        val actions = listOfNotNull(onCancel, onComplain, onRate, onChat)
        if (actions.isEmpty()) return@Card

        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            // **والإلغاءُ أحمرُ لأنّه لا يُتراجع عنه.**
            onCancel?.let {
                RahalButton(
                    onClick = it,
                    tone = Tone.Danger,
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.ord_cancel)) }
            }
            onRate?.let {
                RahalButton(onClick = it, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.ord_rate))
                }
            }
            // **وحديثُ السائق قبل الشكوى** — **ومن يستطيع أن يسأل لا
            // يشكو**: أكثرُ الشكاوى سوءُ فهمٍ يحلّه سطر.
            onChat?.let {
                RahalOutlineButton(
                    onClick = it,
                    tone = Tone.Accent,
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.ord_chat), maxLines = 1) }
            }
            onComplain?.let {
                RahalOutlineButton(onClick = it, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.ord_complain))
                }
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقدُ عرضِ الطلب المخصَّص في البطاقة** (Batch 2c)
 * ══════════════════════════════════════════════════════════════════════
 *
 * ثلاثُ حالات: **بانتظار التكلفة** (لم يُتّفق بعد) · **عرضٌ يُؤكَّد**
 * (وصل المبلغُ ولم يؤكّده الزبون، أو تغيّر فبطل تأكيدُه) · **مؤكَّد**
 * (يُظهر الحجزَ إن كان محفظة). **والمحفظةُ لا تُعرَض إلّا إن غطّى رصيدُها
 * المبلغَ** — والمحرّكُ سلطانٌ على كلّ حال.
 */
@Composable
private fun CustomQuoteSection(
    order: MyOrder,
    walletBalance: Long,
    onConfirmQuote: ((String) -> Unit)?,
    confirming: Boolean,
) {
    val goods = order.customGoodsAmount
    // (B) لم يُتّفق على السعر بعد — بانتظار تحديد التكلفة.
    if (goods == null) {
        KeyValue(
            stringResource(R.string.ord_delivery),
            stringResource(R.string.ord_delivery_on_deal),
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.ord_awaiting_quote),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        return
    }

    val fee = order.customFee ?: 0L
    val total = order.total
    val confirmedCurrent =
        order.quoteConfirmedAt != null && order.quoteConfirmedVersion == order.quoteVersion

    // (C) بطاقةُ المبلغ — تُعرَض دائماً حين يُعرَف.
    KeyValue(stringResource(R.string.ord_goods), money(goods))
    KeyValue(stringResource(R.string.ord_delivery), money(fee))
    KeyValue(stringResource(R.string.ord_total), money(total), valueColor = Rahal.colors.brand)

    // (K) بعد التأكيد — يُظهر الحجزَ من المحفظة إن وُجد، ولا زرَّ.
    if (confirmedCurrent) {
        if (order.customReservedAmount > 0) {
            Spacer(Modifier.height(6.dp))
            Text(
                stringResource(R.string.ord_reserved_note, money(order.customReservedAmount)),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        return
    }

    // **ولا تأكيدَ بعد نهايةٍ ولا في السجلّ.**
    if (onConfirmQuote == null || ended(order.status)) return

    // (C/D) يحتاج تأكيداً: أكّد المبلغ واختر الدفع.
    val walletOk = walletBalance >= total
    var method by rememberSaveable(order.id, order.quoteVersion) { mutableStateOf("cash") }
    // (J) المحفظةُ لا تبقى مختارةً إن لم تعُد كافية قبل التأكيد.
    if (method == "wallet" && !walletOk) method = "cash"

    Spacer(Modifier.height(10.dp))
    HorizontalDivider()
    Spacer(Modifier.height(8.dp))
    Text(
        stringResource(R.string.ord_confirm_prompt),
        color = Rahal.colors.ink,
        style = MaterialTheme.typography.bodyMedium,
        fontWeight = FontWeight.Bold,
    )
    Spacer(Modifier.height(6.dp))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        FilterChip(
            selected = method == "cash",
            onClick = { method = "cash" },
            label = { Text(stringResource(R.string.cart_cash)) },
        )
        // **والمحفظةُ خيارٌ فقط إن غطّى رصيدُها المبلغَ** — وإلّا نصٌّ لا زرّ.
        if (walletOk) {
            FilterChip(
                selected = method == "wallet",
                onClick = { method = "wallet" },
                label = { Text(stringResource(R.string.cart_wallet)) },
            )
        }
    }
    if (!walletOk) {
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.ord_wallet_short),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
    }
    Spacer(Modifier.height(10.dp))
    RahalButton(
        onClick = { onConfirmQuote(method) },
        enabled = !confirming,
        modifier = Modifier.fillMaxWidth(),
    ) {
        if (confirming) {
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
        } else {
            Text(stringResource(R.string.ord_confirm_action))
        }
    }
}

private fun ended(status: String): Boolean =
    status in setOf("delivered", "cancelled", "failed", "rejected")

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اسمُ المرحلة بالعربيّة — ولا شيءَ غيرُه في الجهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المحرّكُ يقول أيَّ المراحلِ وأينَ هو منها** (`stages/stage_at`)،
 * **والجهازُ يترجم المفتاح** — لا يزيد ولا ينقص ولا يرتّب.
 *
 * **وكانت المراحلُ مكتوبةً هنا بيد** فسقطت منها «وصل إليك»
 * (٢٠٢٦-٠٨-١٥): **قائمةٌ تُكتب مرّتين تفترق مرّة.**
 *
 * **ومفتاحٌ لا نعرفه يُكتب كما جاء** — لا يُطوى: **مرحلةٌ تُزاد في
 * المحرّك تظهر بلا اسمٍ خيرٌ من أن تختفي.**
 */
@Composable
private fun stageLabel(key: String): String = when (key) {
    "waiting" -> stringResource(R.string.st_pending)
    "accepted" -> stringResource(R.string.st_accepted)
    "preparing" -> stringResource(R.string.st_preparing)
    "seeking" -> stringResource(R.string.st_driver)
    "bought" -> stringResource(R.string.st_bought)
    "on_the_way" -> stringResource(R.string.ord_st_onway)
    "arrived" -> stringResource(R.string.st_arrived)
    "delivered" -> stringResource(R.string.ord_st_delivered)
    else -> key
}
