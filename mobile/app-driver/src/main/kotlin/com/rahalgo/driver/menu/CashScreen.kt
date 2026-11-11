package com.rahalgo.driver.menu

import com.rahalgo.ui.RahalDivider
import androidx.compose.foundation.layout.Arrangement
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.driver.R
import com.rahalgo.ui.Bar
import com.rahalgo.ui.Card
import com.rahalgo.ui.DayHead
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.Note
import com.rahalgo.ui.Screen
import com.rahalgo.ui.dayText
import com.rahalgo.ui.money
import com.rahalgo.ui.timeText
import com.rahalgo.shared.model.CashEntry

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صندوقي — مالُ غيري في يدي**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٣: «صندوقي غير موجودٍ أيضاً، اطّلع عليه من
 *  الويب» — ثمّ أمرُه: «ابدأ بملء الأقسام من الويب».)
 *
 * # وهو غيرُ المحفظة
 *
 * **المحفظةُ مالُه هو** — أجورُ ما وصّل. **والصندوقُ مالُ غيره في يده**:
 * ما قبضه نقداً من الزبائن حتّى يسلّمه للمكتب.
 *
 * # ومالٌ في ذمّة إنسانٍ بلا كشفٍ يقرؤه خلافٌ ينتظر
 *
 * يقول «سلّمتُ» وتقول المنصّةُ «لم يصل»، **ولا ورقةَ بينهما.** (وهو نصُّ
 * شاشة الويب نفسِه — والنداءُ نفسُه.)
 *
 * # والسقفُ يُقال بما يبقى لا بما مضى
 *
 * **والسائقُ يسأل سؤالاً واحداً: كم أقدر أن أقبض بعد؟** — فيُقال له
 * بالرقم. **و«٢٠٤٬٠٠٠ / ٥٬٠٠٠٬٠٠٠» كسرٌ يُحسب في الرأس.**
 *
 * **والسقفُ ليس زينة**: من بلغه لا يُعرض عليه طلبٌ نقديٌّ جديد، **فيقف
 * عملُه ولا يعرف لماذا.**
 */
@Composable
fun CashScreen(vm: SectionsViewModel) {
    val page = vm.cash
    val me = vm.me
    if (page == null || me == null) {
        LoadState(vm.busy, vm.error) { vm.open(MenuItem.Cash, force = true) }
        return
    }

    Screen {
        // **والعنوانُ رأسٌ بسهمٍ فوق الصفحة** — ولا سطرَ شرح (مراجعةُ المالك ٢٠٢٦-١٠-٣٠).

        val held = me.cashHeld
        val limit = me.cashLimit
        val left = limit - held
        val ratio = if (limit > 0) held.toFloat() / limit else 0f

        // **بطاقةُ الرأس بالنظام الموحّد** (إعادةُ التصميم ٢٠٢٦-١١-١١) — الرقمُ بأكبر دورٍ في السلّم،
        // وبالبرتقاليّ الغامق للكتابة (`accentInk`) لا البرتقاليّ الذي لا يُقرأ على الفاتح.
        com.rahalgo.ui.RahalCard(
            variant = com.rahalgo.ui.CardVariant.Tinted,
            tone = com.rahalgo.ui.StatusTone.Warning,
            padding = Rahal.space.lg,
        ) {
            Text(stringResource(R.string.cash_held), color = Rahal.colors.inkMuted, style = Rahal.type.label)
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                text = com.rahalgo.driver.Bidi.isolate(money(held)),
                color = Rahal.colors.accentInk,
                style = Rahal.type.display,
            )
            Spacer(Modifier.height(Rahal.space.md))
            // **ولونُ الشريط يقول قربَه من الحافّة** قبل أن يُقرأ رقم.
            Bar(ratio = ratio, color = cashBarColor(ratio))
            Spacer(Modifier.height(Rahal.space.sm))
            // **ينزل الثاني سطراً إن ضاق** (الفحصُ البصريّ ٢٠٢٦-١١-١١: على ٣٦٠ بخطٍّ ١٫٣ تلاصق
            // «السقف ١٠٬٠٠٠ ل.س» و«يبقى قبل السقف» فقُرئا جملةً واحدة).
            @OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
            androidx.compose.foundation.layout.FlowRow(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalArrangement = Arrangement.spacedBy(Rahal.space.xs),
                itemVerticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.cash_limit) + " " + com.rahalgo.driver.Bidi.isolate(money(limit)),
                    color = Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                )
                Text(
                    text = if (left > 0) {
                        stringResource(R.string.cash_left) + " " + com.rahalgo.driver.Bidi.isolate(money(left))
                    } else {
                        stringResource(R.string.cash_over) + " " + com.rahalgo.driver.Bidi.isolate(money(-left))
                    },
                    color = if (left > 0) Rahal.colors.success else Rahal.colors.danger,
                    style = Rahal.type.label,
                )
            }
            // **ومن معه نقدٌ يسلّمه من هنا** (٢٠٢٦-١١-١١) — ورقةٌ بالمبلغ وعنوان المكتب ورقمه، بلا حركة مال.
            if (held > 0) {
                Spacer(Modifier.height(Rahal.space.md))
                com.rahalgo.driver.home.HandInButton(held = held)
            }
        }

        // **والتحذيرُ عند الحدّ لا بعده** — من بلغ السقفَ وقف طابورُه.
        Spacer(Modifier.height(Rahal.space.md))
        when {
            limit > 0 && ratio >= 1f -> Note(stringResource(R.string.cash_full), Rahal.colors.danger)
            ratio > 0.8f -> Note(stringResource(R.string.cash_near), Rahal.colors.accent)
        }

        if (page.entries.isEmpty()) {
            com.rahalgo.ui.EmptyState(title = stringResource(R.string.cash_empty), icon = R.drawable.ic_cash)
            return@Screen
        }

        // **والحركاتُ مجموعةٌ بالأيّام**: من سلّم صندوقَه مساءً يريد أن
        // يرى «اليوم» وحدَه، **وكشفٌ متّصلٌ من مئة سطرٍ لا يُراجَع.**
        //
        // **ولا يُفترض ترتيبٌ لم يُطلب** — ردٌّ غيرُ مرتّبٍ يُنتج يوماً
        // يتكرّر مرّتين في كشف مال.
        var day = ""
        page.entries.sortedByDescending { it.createdAt }.forEach { e ->
            val d = dayText(e.createdAt)
            if (d != day) {
                day = d
                DayHead(d)
            }
            CashRow(e)
        }

        // **وناقصٌ يقول إنّه ناقص** — كشفٌ قُصَّ عند الصفحة ولا يقول
        // **يُقرأ كاملا**، فيُجمع فلا يساوي ما في يده.
        if (page.total > page.entries.size) {
            Spacer(Modifier.height(Rahal.space.md))
            Text(
                text = stringResource(
                    R.string.cash_more,
                    page.entries.size.toString(),
                    page.total.toString(),
                ),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
            Spacer(Modifier.height(Rahal.space.sm))
            RahalOutlineButton(
                onClick = vm::loadMoreCash,
                enabled = !vm.cashMoreBusy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (vm.cashMoreBusy) {
                    CircularProgressIndicator(Modifier.size(Rahal.icon.sm), strokeWidth = Rahal.stroke.focus)
                } else {
                    Text(stringResource(R.string.cash_load_older), style = Rahal.type.button)
                }
            }
        }
    }
}

/**
 * **لونُ شريط الصندوق — واحدٌ للشاشتين** (الرئيسيّة وصندوقي).
 *
 * كانت الرئيسيّةُ تقلبه برتقاليّاً فوق ٨٠٪ وصندوقي فوق ٥٠٪، **فنفسُ المبلغ أخضرُ هنا
 * وبرتقاليٌّ هناك.** والحدودُ حدودُ تنبيه «اقتربت من السقف» نفسِه.
 */
@Composable
internal fun cashBarColor(ratio: Float): Color = when {
    ratio >= 1f -> Rahal.colors.danger
    ratio > 0.8f -> Rahal.colors.accent
    else -> Rahal.colors.success
}

/**
 * **صفُّ حركة** — جملةٌ تُقرأ: ما وقع · على أيّ طلب · بكم · ومتى.
 *
 * **والاتّجاهُ يُقرأ قبل الرقم**: قبضٌ يزيد ذمّتَك، **وتسليمٌ يُنقصها.**
 */
@Composable
private fun CashRow(e: CashEntry) {
    val inbound = e.amount > 0
    // **البرتقاليُّ الغامقُ للكتابة** — البرتقاليُّ العاديُّ لا يُقرأ رقماً على الأبيض.
    val color = if (inbound) Rahal.colors.accentInk else Rahal.colors.success
    // **صفُّ حركةٍ بأيقونة اتّجاهه** (إعادةُ التصميم ٢٠٢٦-١١-١١): مربّعٌ ملوّنٌ يميناً، والجملةُ
    // في الوسط، **والمبلغُ في الطرف** — كأيّ كشف حسابٍ رآه في حياته.
    Row(Modifier.fillMaxWidth().padding(vertical = Rahal.space.md), verticalAlignment = Alignment.Top) {
        com.rahalgo.ui.IconBox(
            icon = if (inbound) R.drawable.ic_cash else R.drawable.ic_check_circle,
            tint = color,
        )
        Spacer(Modifier.size(Rahal.space.md))
    Column(Modifier.weight(1f)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                text = com.rahalgo.driver.Bidi.isolate((if (inbound) "+" else "−") + money(kotlin.math.abs(e.amount))),
                color = color,
                style = Rahal.type.price,
            )
            e.orderNumber?.let {
                Text(com.rahalgo.driver.Bidi.isolate("#$it"), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
            }
        }
        // **والاسمُ عربيٌّ دائماً** — ورمزٌ لا اسمَ له تقوله ملاحظتُه،
        // **ولا يُعرض رمزٌ إنكليزيٌّ على سائق.**
        //
        // **ومِمَّن قبض باسمه** — (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «مكتوبٌ
        // قبضتُ من زبون وهذا غلط، أساساً هو معروف»). **و«قبضتُ من
        // زبون» ثلاثَ مرّاتٍ في يومٍ لا تُميّز واحدةً من أخرى.**
        //
        // **وقيدٌ بلا طلبٍ يبقى بصفته** — تسويةُ إدارةٍ لا صاحبَ لها،
        // **واسمٌ يُخترع لها أسوأُ من صفة.**
        val label = if (inbound && e.customerName.isNotEmpty()) {
            stringResource(R.string.cash_from, e.customerName)
        } else {
            cashKind(e.kind)
        }
        Text(label.ifEmpty { e.note }, style = Rahal.type.body, color = Rahal.colors.ink)
        // **وتفصيلُ التحصيل** (يرسله الخادمُ منذ ٢٠٢٦-١٠-٢٦) — كم منها أجرةٌ وكم عمولة،
        // ومن أيّ متجر، **وأكان التوصيلُ مجانيّاً للزبون.**
        //
        // **والتفصيلُ لسطر التحصيل وحدَه** — الخادمُ يُلحق بيانات الطلب بسطرَي
        // «انمسحت العمولة» و«رجعت النقد» أيضاً، **وأجرةٌ وعمولةٌ تحتهما تُقرأ
        // قبضاً ثانياً.** فهناك اسمُ المتجر فقط.
        //
        // **وأجرةٌ صفرٌ لا تُكتب** — توصيلٌ مجانيٌّ تقوله كلمتُه لا «أجرة ٠».
        val collection = e.kind == "order_collection"
        val parts = buildList {
            if (e.merchantName.isNotBlank()) add(stringResource(R.string.cash_d_store, e.merchantName))
            if (collection) {
                e.feePart?.takeIf { it > 0 }?.let { add(stringResource(R.string.cash_d_fee, money(it))) }
                e.commissionPart?.takeIf { it > 0 }?.let { add(stringResource(R.string.cash_d_commission, money(it))) }
                if ((e.commissionWaived ?: 0) > 0) add(stringResource(R.string.cash_d_commission_waived, money(e.commissionWaived ?: 0)))
                if ((e.freeWaived ?: 0) > 0) add(stringResource(R.string.cash_d_free))
            }
        }
        if (parts.isNotEmpty()) {
            // **والمبلغُ لا ينكسر عن «ل.س»** (رُئي على جوال المالك ٢٠٢٦-١١-١١: «ل.س» وحدَها أوّلَ السطر) —
            // مسافاتُ كلّ جزءٍ لا تنكسر، والسطرُ ينكسر بين الأجزاء عند «·» وحدَها.
            Text(
                parts.joinToString(" · ") { it.replace(' ', NBSP) },
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
            )
        }
        // **والملاحظةُ لا تُكرّر الاسم** — تظهر إن قالت زيادة.
        if (e.note.isNotEmpty() && e.note != label) {
            Text(e.note, color = Rahal.colors.inkMuted, style = Rahal.type.caption)
        }
        Text(timeText(e.createdAt), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
    }
    }
    RahalDivider(inset = Rahal.size.iconBox + Rahal.space.md)
}

/**
 * **اسمُ نوع الحركة** — والمجهولُ يُترك ليقوله نصُّ الملاحظة.
 *
 * **والأسماءُ هي التي يكتبها المحرّك** (`cashbox.go`) — لا ما يشبهها:
 * **كان المعجمُ يسمّي `collect` والمحرّكُ يكتب `order_collection`**،
 * فيفشل البحثُ دائماً **فتُعرض إنكليزيّةٌ على سائقٍ في الرقّة.**
 *
 * **وحارسٌ يمنع عودتَها** (`cashbox/kinds_test.go`) — يقرأ الأنواعَ من
 * المحرّك ويفتّش عنها في نصوص التطبيق ومعجم الويب معا.
 */
@Composable
private fun cashKind(kind: String): String = when (kind) {
    "order_collection" -> stringResource(R.string.cash_k_order_collection)
    "settlement" -> stringResource(R.string.cash_k_settlement)
    "commission_waived" -> stringResource(R.string.cash_k_commission_waived)
    "recipient_cash_returned" -> stringResource(R.string.cash_k_recipient_cash_returned)
    "collection_reversed" -> stringResource(R.string.cash_k_collection_reversed)
    else -> ""
}

/** **مسافةٌ لا تنكسر** — تربط المبلغَ بعملته في سطرٍ واحد. */
private const val NBSP = '\u00A0'
