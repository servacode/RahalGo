package com.rahalgo.driver.menu

import androidx.compose.foundation.layout.Arrangement
import com.rahalgo.shared.model.MyReport
import com.rahalgo.ui.SectionTitle
import androidx.compose.foundation.layout.Column
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.ui.Card
import com.rahalgo.ui.ticketStatusColor
import com.rahalgo.ui.ticketStatusText
import com.rahalgo.ui.Chip
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.TicketRow
import com.rahalgo.ui.TicketsScreen
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle
import com.rahalgo.ui.whenText
import com.rahalgo.shared.model.ComplaintBrief

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الشكاوى والبلاغات — ما رُفع عليّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # لماذا يراها
 *
 * **كانت تصل غرفةَ العمليات وحدَها.** ومن اشتُكي عليه ولا يعلم **لا
 * يُصلح شيئاً** — يُخصم منه يوماً فيُفاجأ، **ويظنّ الظلمَ حيث كان خبر.**
 *
 * **والشكوى تُقرأ لا تُبلَّغ فحسب**: «تأخّر ساعةً» و«لم يردّ على الهاتف»
 * سببان مختلفان، **وأحدُهما يُصلَح بترتيب اليوم والآخر بشحن الهاتف.**
 *
 * # ولا اسمَ لصاحبها
 *
 * **تلك خصومةٌ تُحقَّق، وكشفُ اسمِه يفتح باباً للردّ عليه** — والذي
 * يخصُّ السائقَ ما وقع لا من قاله.
 *
 * # وبلاغاتُه هو ليست هنا
 *
 * **بلاغُ السائق على متجرٍ يُفتح باسم زبون الطلب** (`support/driver_report.go`)
 * — **فلا بابَ في المحرّك يردّ ما رفعه هو.** وهي ثغرةٌ مقيسة، مكتوبةٌ في
 * تقرير الجرد، **ولا تُخترع لها شاشةٌ تعرض فراغا.**
 */
@Composable
fun ComplaintsScreen(vm: SectionsViewModel) {
    val rep = vm.reputation
    if (rep == null) {
        LoadState(vm.busy, vm.error) { vm.open(MenuItem.Tickets, force = true) }
        return
    }

    // ══════════════════════════════════════════════════════════════════
    // **والشاشةُ من `:ui` — وكانت قسمين فوق بعضهما**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٣١: «أفضلُ شكلٍ هو شكلُ الشكاوى في المتجر ·
    // وأساساً يجب أن يكون بشكلٍ مركزيّ».)
    //
    // **والقسمان يجعلانه يمرّر ليعرف أيَّهما يقرأ** — **والتبويبُ يفصل
    // قبل أن يقرأ سطراً.** (وأصلُ الفصل بلاغُ المالك ٢٠٢٦-٠٨-١٣ بعد أن
    // قرأ بلاغَه شكوى عليه — **والفصلُ باقٍ، وشكلُه هو الذي تبدّل.**)
    TicketsScreen(
        title = stringResource(com.rahalgo.ui.R.string.menu_tickets),
        hint = stringResource(R.string.tik_hint),
        mine = rep.reports.map { r ->
            TicketRow(
                key = "#" + r.number,
                title = reportReason(r.reason),
                status = r.status,
                orderNumber = r.orderNumber?.toString().orEmpty(),
                resolution = r.resolution,
            )
        },
        mineEmpty = stringResource(R.string.tik_mine_empty),
        // **والشكوى عليه لا سببَ لها بل موضوع** — المحرّكُ يكتبه نصّاً،
        // **ولا رمزَ يُترجَم.**
        againstMe = rep.complaints.map { c ->
            TicketRow(
                key = "#" + c.number,
                title = c.subject,
                status = c.status,
                orderNumber = c.orderNumber?.toString().orEmpty(),
            )
        },
        // **وفراغُها خبرٌ سارٌّ يُقال** — لا قائمةٌ فارغةٌ صامتة.
        againstMeEmpty = stringResource(R.string.tik_none) + " — " +
            stringResource(R.string.tik_none_hint),
    )
}
/** **سببُ البلاغ بعربيّته** — والمجهولُ يُعرض برمزه ليُبلَّغ عنه. */
@Composable
private fun reportReason(code: String): String = when (code) {
    "merchant_slow" -> stringResource(R.string.reason_merchant_slow)
    "merchant_refused" -> stringResource(R.string.reason_merchant_refused)
    "merchant_wrong_goods" -> stringResource(R.string.reason_merchant_wrong_goods)
    "merchant_conduct" -> stringResource(R.string.reason_merchant_conduct)
    "customer_absent" -> stringResource(R.string.reason_customer_absent)
    "customer_address" -> stringResource(R.string.reason_customer_address)
    "customer_refused" -> stringResource(R.string.reason_customer_refused)
    "customer_conduct" -> stringResource(R.string.reason_customer_conduct)
    "other" -> stringResource(R.string.rs_other)
    else -> code
}
/** **حالُ الشكوى بعربيّة** — والمجهولُ يُعرض برمزه ليُبلَّغ عنه. */

