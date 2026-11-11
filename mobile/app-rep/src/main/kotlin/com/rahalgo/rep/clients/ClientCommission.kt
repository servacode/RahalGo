package com.rahalgo.rep.clients

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import com.rahalgo.design.Rahal
import com.rahalgo.rep.R
import com.rahalgo.shared.rep.RepMerchantDetail
import com.rahalgo.shared.rep.RepMerchantHead
import com.rahalgo.shared.rep.RepStoreCommission
import com.rahalgo.ui.IconTile
import com.rahalgo.ui.LabeledField
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalDialog
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.WesternDigits
import java.math.BigDecimal

/**
 * «7.5» و«10» و«0» — بلا أصفارٍ زائدة.
 *
 * **والصفرُ يُكتب «0» نصّاً لا حساباً**: `stripTrailingZeros()` على الصفر تُبقيه «0.0» في `java.math`
 * القديمة (أندرويد قبل OpenJDK)، **والحدُّ الأدنى الافتراضيُّ صفر** — فيقرأ المندوبُ «من 0.0٪»
 * على هاتفٍ قديم، ومتجرٌ نسبتُه صفرٌ يظهر «0.0٪».
 */
internal fun pctText(v: Double): String =
    if (v == 0.0) "0" else BigDecimal.valueOf(v).stripTrailingZeros().toPlainString()

/**
 * **ما كتبه المندوبُ لاتينيّاً ونقطةً** — قبل أن يُفحص وقبل أن يُرسَل.
 *
 * **كيبوردُ الأرقام العربيّة يكتب «٧٫٥»، والمحرّكُ (`pricing.ParsePercent2`) يرفض كلَّ حرفٍ ليس
 * من 0 إلى 9 ونقطة** — فالأرقامُ العربيّةُ والفارسيّةُ تصير 0–9 (`WesternDigits`)، وفاصلتُها «٫»
 * وفاصلةُ الإنكليزيّ «,» تصيران «.».
 */
internal fun normalizePctInput(raw: String): String = WesternDigits.text(raw).replace(',', '.')

/** **شكلُ ما يقبله المحرّكُ**: أرقامٌ، أو نقطةٌ بعدها رقمٌ أو رقمان (قبلها أرقامٌ أو لا). */
private val PCT_SHAPE = Regex("[0-9]+|[0-9]*\\.[0-9]{1,2}")

/**
 * **النسبةُ كما تُرسَل إلى المحرّك، أو `null` إن كان سيردّها.**
 *
 * **والزرُّ لا يُفعَّل إلّا لما يقبله المحرّك** — شكلاً (`pricing.ParsePercent2`: نقطةٌ يتبعها رقمٌ
 * على الأقلّ، ومنزلتان عشريّتان على الأكثر) **وحدّاً** (حدّا المندوب من الإعدادات). **وما فُحص هو ما
 * يُرسَل**: الدالةُ نفسُها تُفعّل الزرَّ وتُخرج النصَّ، فلا يتفرّقان.
 *
 * **و«7.» تُقرأ «7»** — تُحذف نقطتُها الأخيرةُ فلا تُرسَل؛ **والمحرّكُ يردّ نقطةً بلا كسر بعدها.**
 */
internal fun commissionPayload(raw: String, lo: Double, hi: Double): String? {
    var s = normalizePctInput(raw.trim())
    if (s.length > 1 && s.endsWith('.')) s = s.dropLast(1)
    if (!PCT_SHAPE.matches(s)) return null
    val v = BigDecimal(s)
    return if (v >= BigDecimal.valueOf(lo) && v <= BigDecimal.valueOf(hi)) s else null
}

/**
 * **التفصيلُ بنسبةٍ حُفظت** — نسبةُ المتجر وحدّاها من ردّ المحرّك، وكلُّ ما عداها كما هو.
 */
internal fun RepMerchantDetail.withStoreCommission(saved: RepStoreCommission): RepMerchantDetail =
    copy(
        merchant = merchant.copy(
            storeCommissionPercent = saved.percent,
            storeCommissionMin = saved.min,
            storeCommissionMax = saved.max,
        ),
    )

/**
 * **التفصيلُ بعد ردِّ حفظٍ — للعميل المفتوحِ نفسِه وحدَه.**
 *
 * **والحفظُ يعيش في `ClientsViewModel` فقد يصل ردُّه بعد أن يغلق المندوبُ العميلَ ويفتح غيرَه**
 * — فنسبةُ المتجر «أ» لا تُكتب في تفصيل «ب» ولا في فراغٍ؛ **يبقى كلٌّ كما هو**، والعميلُ يُفتح
 * من جديدٍ فيُجلب من المحرّك.
 */
internal fun detailAfterCommissionSaved(
    current: RepMerchantDetail?,
    openId: String,
    savedFor: String,
    saved: RepStoreCommission,
): RepMerchantDetail? =
    if (current == null || openId != savedFor) current else current.withStoreCommission(saved)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نسبةُ عمولة متجر العميل — يضبطها المندوب** (قرارُ المالك ٢٠٢٦-١١-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بين ال0 وال10 بالمية» — **والحدّان من المحرّك** يُعرضان تحت الخانة، وتسري فوراً
 * بلا موافقة المكتب. **وما يراه نسبةُ المتجر وحدَها** لا ربحَ المنصّة.
 *
 * # ولماذا لا يحفظ السطرُ بنفسه
 *
 * **حالُ السطر هنا محلّيّةٌ تُمحى كلّما غطّت شاشةٌ التفصيلَ** («ساعات الدوام» · «الأقسام» ·
 * «الأصناف» · «العروض») ثمّ يُبنى من التفصيل المحفوظ في `ClientsViewModel`. **ونداءُ الحفظ لو بقي
 * في نطاقٍ من نطاقات السطر لَمات بخروجه**: يضغط المندوبُ «حفظ» ثمّ «ساعات الدوام» قبل الردّ،
 * فيُلغى النداءُ والمحرّكُ قد حفظ، ويعود السطرُ يقول القديمَ.
 *
 * **فالحفظُ في الـvm** (`saveStoreCommission`): يصل ردُّه `detail` ولو غادر المندوبُ الصفحةَ.
 * **والسطرُ يقرأ نسبتَه وحدّيها من `head` وحدَه**، ويُسلِّم الـvm ما يُرسَل ومعه ما يُفعَل عند
 * النجاح (`onSave`) وهو إغلاقُ الخانة، ويأخذ منه `saving` فلا يُرسَل طلبان معاً.
 */
@Composable
fun StoreCommissionRow(
    merchantId: String,
    head: RepMerchantHead,
    saving: Boolean,
    onSave: (String, () -> Unit) -> Unit,
    modifier: Modifier = Modifier,
) {
    val current = head.storeCommissionPercent
    val lo = head.storeCommissionMin
    val hi = head.storeCommissionMax
    var editing by remember(merchantId) { mutableStateOf(false) }
    var text by remember(merchantId, current) { mutableStateOf(pctText(current)) }

    // **مربّعٌ بأيقونةٍ وقيمةٍ يُضغط فتُفتح نافذةُ التعديل** (طلبُ المالك ٢٠٢٦-١١-١١) — لا سطرٌ وزرٌّ مؤطَّر.
    IconTile(
        icon = R.drawable.ic_percent,
        label = stringResource(R.string.cd_commission_tile),
        value = pctLabel(current),
        valueColor = Rahal.colors.brand,
        modifier = modifier,
        onClick = { editing = true },
    )
    if (!editing) return

    val close = { editing = false; text = pctText(current) }
    RahalDialog(
        onDismissRequest = close,
        title = { Text(stringResource(R.string.cd_commission_title), style = Rahal.type.sectionTitle) },
        text = {
            // **والحدّان سطرُ عونٍ تحت الخانة** — ويصير أحمرَ إن خرجت النسبةُ عنهما.
            // **ما يُفحص هو ما يُرسَل** — لاتينيّاً وبما يقبله المحرّك وحدَه، والفارغُ `null` (`commissionPayload`).
            val payload = commissionPayload(text, lo, hi)
            val range = stringResource(R.string.cd_commission_range, pctLabel(lo), pctLabel(hi))
            LabeledField(
                label = stringResource(R.string.cd_commission_field),
                value = text,
                // **والخانةُ تُظهر ما سيُرسَل**: الأرقامُ العربيّةُ تصير لاتينيّةً عند الكتابة (قاعدةُ الأرقام).
                onValueChange = { v -> text = normalizePctInput(v).filter { c -> c in '0'..'9' || c == '.' }.take(6) },
                supporting = range,
                error = if (text.isNotBlank() && payload == null) range else "",
                keyboardType = KeyboardType.Decimal,
            )
        },
        confirmButton = {
            val payload = commissionPayload(text, lo, hi)
            RahalButton(
                // **الحفظُ في الـvm لا هنا** (انظر أعلاه) — ونجاحُه يُغلق النافذةَ، وفشلُه يتركها مفتوحةً.
                onClick = { if (payload != null) onSave(payload) { editing = false } },
                enabled = payload != null && !saving,
            ) { Text(stringResource(com.rahalgo.ui.R.string.act_save)) }
        },
        dismissButton = {
            RahalTextButton(onClick = close) {
                Text(stringResource(R.string.cd_commission_cancel), color = Rahal.colors.inkMuted)
            }
        },
    )
}

/**
 * **«10٪» بأيّ اتّجاه** — الرقمُ ثمّ ٪ بعده، **معزولاً يساريّاً**: داخل جملةٍ عربيّةٍ كان الإشارةُ تنزاح
 * فتُقرأ «٪10». (طلبُ المالك ٢٠٢٦-١١-١١.)
 */
internal fun pctLabel(v: Double): String = "⁦" + pctText(v) + "٪" + "⁩"
