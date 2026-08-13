package com.rahalgo.driver.menu

import androidx.compose.foundation.layout.Arrangement
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.Bar
import com.rahalgo.driver.ui.Card
import com.rahalgo.driver.ui.DayHead
import com.rahalgo.driver.ui.Empty
import com.rahalgo.driver.ui.LoadState
import com.rahalgo.driver.ui.Note
import com.rahalgo.driver.ui.Screen
import com.rahalgo.driver.ui.ScreenTitle
import com.rahalgo.driver.ui.dayText
import com.rahalgo.driver.ui.money
import com.rahalgo.driver.ui.timeText
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
        ScreenTitle(
            stringResource(R.string.menu_cash),
            stringResource(R.string.cash_hint),
        )

        val held = me.cashHeld
        val limit = me.cashLimit
        val left = limit - held
        val ratio = if (limit > 0) held.toFloat() / limit else 0f

        Card(tone = Rahal.colors.accent) {
            Text(stringResource(R.string.cash_held), color = Rahal.colors.inkMuted)
            Spacer(Modifier.height(4.dp))
            Text(
                text = money(held),
                color = Rahal.colors.accent,
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(12.dp))
            // **ولونُ الشريط يقول قربَه من الحافّة** قبل أن يُقرأ رقم.
            Bar(
                ratio = ratio,
                color = when {
                    ratio > 0.8f -> Rahal.colors.danger
                    ratio > 0.5f -> Rahal.colors.accent
                    else -> Rahal.colors.success
                },
            )
            Spacer(Modifier.height(6.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.cash_limit) + " " + money(limit),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Text(
                    text = if (left > 0) {
                        stringResource(R.string.cash_left) + " " + money(left)
                    } else {
                        stringResource(R.string.cash_over) + " " + money(-left)
                    },
                    color = if (left > 0) Rahal.colors.success else Rahal.colors.danger,
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold,
                )
            }
        }

        // **والتحذيرُ عند الحدّ لا بعده** — من بلغ السقفَ وقف طابورُه.
        Spacer(Modifier.height(10.dp))
        when {
            limit > 0 && ratio >= 1f -> Note(stringResource(R.string.cash_full), Rahal.colors.danger)
            ratio > 0.8f -> Note(stringResource(R.string.cash_near), Rahal.colors.accent)
        }

        if (page.entries.isEmpty()) {
            Empty(stringResource(R.string.cash_empty))
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
            Spacer(Modifier.height(12.dp))
            Text(
                text = stringResource(
                    R.string.cash_more,
                    page.entries.size.toString(),
                    page.total.toString(),
                ),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

/**
 * **صفُّ حركة** — جملةٌ تُقرأ: ما وقع · على أيّ طلب · بكم · ومتى.
 *
 * **والاتّجاهُ يُقرأ قبل الرقم**: قبضٌ يزيد ذمّتَك، **وتسليمٌ يُنقصها.**
 */
@Composable
private fun CashRow(e: CashEntry) {
    val inbound = e.amount > 0
    val color = if (inbound) Rahal.colors.accent else Rahal.colors.success
    Column(Modifier.fillMaxWidth().padding(vertical = 7.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                text = (if (inbound) "+" else "−") + money(kotlin.math.abs(e.amount)),
                color = color,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
            e.orderNumber?.let {
                Text("#$it", color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
            }
        }
        // **والاسمُ عربيٌّ دائماً** — ورمزٌ لا اسمَ له تقوله ملاحظتُه،
        // **ولا يُعرض رمزٌ إنكليزيٌّ على سائق.**
        val label = cashKind(e.kind)
        Text(label.ifEmpty { e.note }, style = MaterialTheme.typography.bodyMedium)
        // **والملاحظةُ لا تُكرّر الاسم** — تظهر إن قالت زيادة.
        if (e.note.isNotEmpty() && e.note != label) {
            Text(e.note, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        }
        Text(timeText(e.createdAt), color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.height(6.dp))
        HorizontalDivider()
    }
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
    else -> ""
}
