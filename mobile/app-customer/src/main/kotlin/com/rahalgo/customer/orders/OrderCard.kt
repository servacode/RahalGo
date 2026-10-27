package com.rahalgo.customer.orders

import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.RahalTextButton
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
import androidx.compose.foundation.background
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.padding
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
    /** **يرفض السعرَ الجديد** بعد الاستلام — و`null` لا رفضَ هنا. */
    onRejectQuote: (() -> Unit)? = null,
    /** **نصُّ زرّ الإلغاء بالعدّاد** — «إلغاء الطلب (باقي 0:45)»، و`null` النصُّ العاديّ. */
    cancelLabel: String? = null,
) {
    Card {
        // **الحالُ أبرزُ ما في الرأس ورقمُ الطلب صغيرٌ جنبها** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧)
        // — كان الرقمُ أبرزَ شيءٍ في البطاقة، **والزبونُ يسأل «وين صار طلبي» لا «شو رقمه».**
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            val tone = statusColor(order.status)
            Text(
                text = statusText(order.status),
                color = tone,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall,
                modifier = Modifier
                    .clip(Rahal.shape.pill)
                    .background(tone.copy(alpha = 0.12f))
                    .padding(horizontal = 12.dp, vertical = 5.dp),
            )
            Text(
                // **ورقمُه كما يعرفه الجميع** — لا ستّةُ أحرفٍ من
                // معرّفه: **رقمٌ لا يعرفه صاحبُه ولا المكتبُ لا يُستعمل
                // في شكوى.**
                text = "#" + order.number,
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
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
            Stages(order.stages.map { stageLabel(it, order.customMode == "amanah") }, order.stageAt, live = true)
        }

        // ══════════════════════════════════════════════════════════════
        // **وكم يأخذ تقريباً — من المحرّك** (تدقيقُ ٢٠٢٦-١٠-٢٧)
        // ══════════════════════════════════════════════════════════════
        //
        // **أكثرُ سؤالٍ عند الزبون «إيمتى بيوصل؟»** والمحرّكُ يرسل التقدير. **وبالطريق
        // زمنُ الطريق من المتجر إلى بابه** (`to_door_eta_sec`، يُلتقط عند الاستلام)،
        // **وقبله مدّةُ التوصيل المتوقّعة** (`delivery_estimate_min`). **ولا يُخترَع رقم.**
        if (!ended(order.status)) {
            val door = order.toDoorEtaSec?.takeIf {
                it > 0 && order.status in setOf("picked_up", "on_the_way")
            }
            val etaText = when {
                door != null -> stringResource(
                    R.string.ord_eta_door,
                    com.rahalgo.ui.minutes(((door + 59) / 60).toLong()),
                )
                order.deliveryEstimateMin > 0 && order.status != "at_dropoff" -> stringResource(
                    R.string.ord_eta_estimate,
                    com.rahalgo.ui.minutes(order.deliveryEstimateMin.toLong()),
                )
                else -> ""
            }
            if (etaText.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                Text(
                    text = etaText,
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }

        // **والسائقُ يُسمّى حين يُسنَد** — ومن يعرف اسمَ من يحمل طلبَه
        // يطمئنّ.
        // **والسائقُ يُسمّى حين يُسنَد** — **ويصل `null` قبل ذلك.**
        order.driverName?.takeIf { it.isNotEmpty() && !ended(order.status) }?.let {
            Spacer(Modifier.height(10.dp))
            KeyValue(stringResource(R.string.ord_driver), it)
        }

        // ══════════════════════════════════════════════════════════════
        // **وموقعُه يُشارَك الآن؟ فالإيقافُ هنا أيضاً** (قرارُ المالك ٢٠٢٦-١٠-١٦)
        // ══════════════════════════════════════════════════════════════
        //
        // **لا في الإشعار وحدَه** — من فتح «طلباتي» يرى أنّ موقعَه يُرسَل ويوقفه بضغطة.
        if (com.rahalgo.customer.live.LiveShare.activeOrderId == order.id) {
            val liveCtx = LocalContext.current
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.live_card_sharing),
                    color = Rahal.colors.success,
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                RahalTextButton(
                    onClick = { com.rahalgo.customer.live.LiveShare.stop(liveCtx) },
                    tone = Tone.Danger,
                ) { Text(stringResource(R.string.live_stop)) }
            }
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
            CustomQuoteSection(order, walletBalance, onConfirmQuote, confirming, onRejectQuote)
        } else {
            if (order.subtotal > 0) KeyValue(stringResource(R.string.ord_subtotal), money(order.subtotal))
            com.rahalgo.customer.DeliveryFeeRow(stringResource(R.string.ord_delivery), order.deliveryFee, order.freeWaived)
            if (order.discount > 0) {
                KeyValue(stringResource(R.string.ord_discount), "-" + money(order.discount))
            }
            // **والإجماليُّ برقمٍ كبيرٍ تحت خطّ** (تدقيقُ التصميم ٢٠٢٦-١٠-٢٧) — كان سطراً عاديّاً.
            Spacer(Modifier.height(4.dp))
            HorizontalDivider()
            Spacer(Modifier.height(4.dp))
            com.rahalgo.ui.SumLine(
                stringResource(R.string.ord_total),
                money(order.total),
                strong = true,
                color = Rahal.colors.brand,
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
        // **والعمرُ يمشي والشاشةُ مفتوحة** — انظر `rememberNow`.
        val now by com.rahalgo.ui.rememberNow()
        Since.text(ctx, order.createdAt, now).takeIf { it.isNotEmpty() }?.let {
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
                ) { Text(cancelLabel ?: stringResource(R.string.ord_cancel), maxLines = 1) }
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
    onRejectQuote: (() -> Unit)? = null,
) {
    val goods = order.customGoodsAmount
    // (B) لم يُتّفق على السعر بعد — بانتظار تحديد التكلفة.
    if (goods == null) {
        // **وأجرةُ المنصّة تُقال برقمها** (٢٠٢٦-١٠-١٤) — كانت الأمانةُ بأجرةٍ مفروضةٍ
        // تقول «يحدّدها السائق عند الاتفاق» و«بانتظار تحديد التكلفة»، **ولا اتّفاقَ فيها.**
        // والسياسةُ من إعدادات الخدمة نفسِها التي سعّرت شاشةَ الطلب (`Serving`).
        val fixed = com.rahalgo.customer.Serving.customFeeAdminDefined
        val fee = order.customFee ?: com.rahalgo.customer.Serving.customFeeFor(order.customMode)
        KeyValue(
            stringResource(R.string.ord_delivery),
            if (fixed) money(fee) else stringResource(R.string.ord_delivery_on_deal),
        )
        // **وعرضُ التوصيل المجاني يُذكَر بصيغةٍ صادقة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — شاشةُ الإنشاء
        // قالت «مجانية»، **والعرضُ يُحجز عند تحديد السعر لا قبله.**
        if (com.rahalgo.customer.FreeOffer.state.customOn(order.customMode)) {
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.free_custom_pending),
                color = Rahal.colors.success,
                style = MaterialTheme.typography.bodySmall,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(
                when {
                    fixed && order.customMode == "amanah" -> R.string.cst_amanah_pay
                    fixed -> R.string.cst_total_pending
                    else -> R.string.ord_awaiting_quote
                },
            ),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        return
    }

    val confirmedCurrent =
        order.quoteConfirmedAt != null && order.quoteConfirmedVersion == order.quoteVersion

    // (C) بطاقةُ المبلغ — تُعرَض دائماً حين يُعرَف.
    QuoteAmounts(order)

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

    // ══════════════════════════════════════════════════════════════════
    // **وبعد الاستلام: سعرٌ رفعه المكتبُ يُؤكَّد أو يُرفض** (قرارُ المالك ٢٠٢٦-١٠-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **تأكيدٌ قائمٌ بنسخةٍ سبقتها أخرى** — والتسليمُ موقوفٌ حتّى يؤكّد. **ومن رفض
    // يبقى طلبُه حيث هو** والمكتبُ يتّصل به؛ فيُقال له ذلك ولا يُعاد السؤال.
    // **والرفضُ من أوّل سعرٍ كذلك** (٢٠٢٦-١٠-١٦) — لا بعد التعديل وحده.
    if (order.quoteRejectedAt != null) {
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.ord_quote_rejected_note),
            color = Rahal.colors.inkMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        return
    }

    QuoteConfirmPanel(order, walletBalance, onConfirmQuote, confirming, onRejectQuote)
}

/**
 * **أينتظر هذا الطلبُ تأكيدَ سعره الآن؟** — شرطُ القسم في البطاقة نفسُه.
 *
 * **ونسختان من الشرط تفترقان يوما** — فتسأل النافذةُ عن طلبٍ لا تعرض بطاقتُه
 * زرّاً، أو تسكت عن طلبٍ ينتظر. **فالنافذةُ (`QuoteConfirmDialog`) تقرأ هذه.**
 */
internal fun quotePending(order: MyOrder): Boolean {
    if (order.kind != "custom" || order.customGoodsAmount == null) return false
    if (order.quoteConfirmedAt != null && order.quoteConfirmedVersion == order.quoteVersion) return false
    if (ended(order.status)) return false
    // **ومن رفض السعرَ الجديد لا يُسأل ثانية** — المكتبُ يتّصل به.
    // **ومن رفض لا يُسأل ثانيةً حتّى يعدّل المكتب** — والتعديلُ يمحو علامةَ الرفض.
    return order.quoteRejectedAt == null
}

/**
 * **مبالغُ العرض** — ثمنُ البضاعة والأجرةُ والمجموع، **في البطاقة والنافذة معاً.**
 *
 * **والأمانةُ بلا ثمنِ بضاعة، والمشترياتُ قبل الشراء «بعد الشراء»** (قرارُ المالك
 * ٢٠٢٦-١٠-٠٣: الأجرةُ أوّلاً ثمّ الثمن).
 */
@Composable
private fun QuoteAmounts(order: MyOrder) {
    val goods = order.customGoodsAmount ?: return
    if (order.customMode != "amanah") {
        KeyValue(
            stringResource(R.string.ord_goods),
            if (order.customGoodsPending) stringResource(R.string.ord_goods_after_purchase) else money(goods),
        )
    }
    // **والأجرةُ المُعفاةُ تُقال «مجاني 🎁» والأصلُ مشطوب** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كالطلب
    // العاديّ (`DeliveryFeeRow`)؛ **كانت «التوصيل 0 ل.س».** والمحرّكُ يردّ `custom_fee` بما يدفعه.
    com.rahalgo.customer.DeliveryFeeRow(
        stringResource(R.string.ord_delivery),
        order.customFee ?: 0L,
        order.freeWaived,
    )
    Spacer(Modifier.height(4.dp))
    HorizontalDivider()
    Spacer(Modifier.height(4.dp))
    com.rahalgo.ui.SumLine(
        stringResource(R.string.ord_total),
        money(order.total),
        strong = true,
        color = Rahal.colors.brand,
    )
}

/**
 * **تأكيدُ العرض: ملاحظةُ الرفع، واختيارُ الدفع، وزرّا القبول والرفض.**
 *
 * **في البطاقة والنافذة بشكلٍ واحد** — ونسختان تفترقان في عنوان زرٍّ أو في
 * بوّابة المحفظة. **و`onRejectQuote` فارغٌ لا رفضَ فيه.**
 */
@Composable
private fun QuoteConfirmPanel(
    order: MyOrder,
    walletBalance: Long,
    onConfirmQuote: (String) -> Unit,
    confirming: Boolean,
    onRejectQuote: (() -> Unit)?,
) {
    val amanah = order.customMode == "amanah"
    val total = order.total
    val reconfirm = order.quoteConfirmedVersion != null
    if (reconfirm) {
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.ord_quote_raised_note),
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.bodySmall,
        )
    }

    // (C/D) يحتاج تأكيداً: أكّد المبلغ واختر الدفع.
    // **وحجزُ هذا الطلب نفسِه يُحسب** — يُفكّ عند التأكيد قبل حجز الجديد، فرصيدٌ
    // متاحٌ ناقصٌ عن المجموع قد يكفيه مع ما حُجز له أصلاً.
    val ownReserved = if (order.paymentMethod == "wallet") order.customReservedAmount else 0L
    val walletOk = walletBalance + ownReserved >= total
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
    // **والقبولُ أخضرُ ممتلئ** (طلبُ المالك ٢٠٢٦-١٠-١٨) — كان أزرقَ يُقرأ «متابعة».
    RahalButton(
        onClick = { onConfirmQuote(method) },
        enabled = !confirming,
        tone = com.rahalgo.ui.PriceConfirm.Accept,
        modifier = Modifier.fillMaxWidth(),
    ) {
        if (confirming) {
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
        } else {
            Text(
                stringResource(
                    when {
                        amanah || order.customGoodsPending -> R.string.ord_confirm_fee
                        else -> R.string.ord_confirm_goods
                    },
                ),
            )
        }
    }
    // ══════════════════════════════════════════════════════════════════
    // **والرفضُ زرٌّ أحمرُ ممتلئ — ويُسأل قبله** (قرارُ المالك ٢٠٢٦-١٠-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان زرّاً نصّيّاً لا يكاد يُرى** — والرفضُ يوقف الطلبَ حتّى يتّصل المكتب،
    // **فضغطةٌ عابرةٌ عليه لا تُرَدّ.** فصار زرّاً أحمرَ بعرض البطاقة **وسؤالاً قبله.**
    if (onRejectQuote != null) {
        var askReject by rememberSaveable(order.id, order.quoteVersion) { mutableStateOf(false) }
        Spacer(Modifier.height(8.dp))
        RahalButton(
            onClick = { askReject = true },
            enabled = !confirming,
            tone = com.rahalgo.ui.PriceConfirm.Reject,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(R.string.ord_quote_reject))
        }
        if (askReject) {
            ConfirmDialog(
                title = stringResource(R.string.ord_quote_reject_title),
                body = stringResource(R.string.ord_quote_reject_body),
                confirm = stringResource(R.string.ord_quote_reject_yes),
                onConfirm = onRejectQuote,
                onDismiss = { askReject = false },
            )
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نافذةُ تأكيد السعر — تظهر من أيّ شاشة** (قرارُ المالك ٢٠٢٦-١٠-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان التأكيدُ قسماً في بطاقة الطلب وحدَها** — ومن كان في السوق حين وصل السعرُ
 * لا يعرف أنّ طلبَه موقوفٌ عليه. **فصار نافذةً يركّبها الغلاف** (`MainActivity`)
 * فوق أيّ شاشة، بالمبالغ واختيار الدفع وزرّي القبول والرفض نفسِها.
 *
 * **وإغلاقُها لا يُلغي شيئاً** — القسمُ يبقى في البطاقة، **وتعود النافذةُ في الفتح
 * التالي أو حين تتبدّل نسخةُ العرض** (`OrdersViewModel.dismissQuote`).
 */
@Composable
fun QuoteConfirmDialog(
    order: MyOrder,
    walletBalance: Long,
    confirming: Boolean,
    onConfirmQuote: (String) -> Unit,
    onRejectQuote: () -> Unit,
    onDismiss: () -> Unit,
) {
    // **ولا «لاحقاً» ولا إغلاقَ بلمسةٍ خارجها** (قرارُ المالك ٢٠٢٦-١٠-١٦: «لازم يرفض أو
    // يقبل مباشر») — **فالسائقُ ينتظر ثمنَ بضاعةٍ اشتراها.**
    AlertDialog(
        onDismissRequest = {},
        properties = androidx.compose.ui.window.DialogProperties(
            dismissOnBackPress = false,
            dismissOnClickOutside = false,
        ),
        title = {
            Text(
                stringResource(R.string.ord_quote_dialog_title, order.number),
                fontWeight = FontWeight.Bold,
            )
        },
        text = {
            Column(Modifier.verticalScroll(rememberScrollState())) {
                // **وما طلبه يُقال بلفظه** — ومن له طلبان يعرف أيَّهما يؤكّد.
                if (order.customRequest.isNotEmpty()) {
                    Text(
                        text = stringResource(R.string.ord_wanted) + " " + order.customRequest,
                        color = Rahal.colors.ink,
                        style = MaterialTheme.typography.bodyMedium,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Spacer(Modifier.height(8.dp))
                }
                QuoteAmounts(order)
                QuoteConfirmPanel(order, walletBalance, onConfirmQuote, confirming, onRejectQuote)
            }
        },
        confirmButton = {},
    )
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
 * **ومفتاحٌ لا نعرفه يُترك بلا اسمٍ لا برمزه** (٢٠٢٦-١٠-١٨) — ولا رمزَ
 * إنكليزيٌّ تحت عقدة الشريط؛ **والحارسُ (`RawCodeTest`) يكشف مرحلةً زِيدت بلا اسم.**
 */
@Composable
private fun stageLabel(key: String, amanah: Boolean = false): String = when (key) {
    // ══════════════════════════════════════════════════════════════════
    // **أسماءٌ قصيرةٌ للشريط وحدَه** (قرارُ المالك ٢٠٢٦-١٠-١٥)
    // ══════════════════════════════════════════════════════════════════
    //
    // **كلُّ اسمٍ سطرٌ واحدٌ تحت عقدته** — و«السائق في طريقه لشراء طلبك» لا تسع.
    // **والأسماءُ الطويلةُ باقيةٌ لغير الشريط** (`st_*` وحالُ الطلب في الشارة).
    // **والأمانةُ تُستلم لا تُشترى** — فلها لفظاها.
    "buying" -> stringResource(if (amanah) R.string.stg_buying_amanah else R.string.stg_buying)
    "bought" -> stringResource(if (amanah) R.string.stg_bought_amanah else R.string.stg_bought)
    "waiting" -> stringResource(R.string.stg_waiting)
    "accepted" -> stringResource(R.string.stg_accepted)
    "preparing" -> stringResource(R.string.stg_preparing)
    "seeking" -> stringResource(R.string.stg_seeking)
    "on_the_way" -> stringResource(R.string.stg_on_the_way)
    "arrived" -> stringResource(R.string.stg_arrived)
    "delivered" -> stringResource(R.string.stg_delivered)
    else -> ""
}
