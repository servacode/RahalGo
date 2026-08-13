package com.rahalgo.driver.menu

import androidx.compose.foundation.background
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.InkMuted
import com.rahalgo.design.StateRed
import com.rahalgo.driver.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **القائمة — ثلاثةُ خطوطٍ يعرفها الناسُ بلا اسم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «شو رأيك ٣ خطوطٍ عائمة تفتح القائمة ويكون
 *  مكتوبٌ فيها هذه الأقسام؟ أحسن من «المزيد» ومن اسمٍ معيّن — فالكلّ
 *  يعرف أنّ لها معنىً واضحا».)
 *
 * # وهو أصوبُ من تبويبٍ خامس
 *
 * **الشريطُ السفليُّ أربعةٌ الآن**، وخامسٌ بأسماءٍ عربيّةٍ يضغطها حتّى
 * تُقصّ. **والخطوطُ الثلاثةُ لا تحتاج اسماً أصلاً** — ومن رآها في أيّ
 * تطبيقٍ عرف ما تفتح.
 *
 * # والأقسامُ ثلاثُ مجموعاتٍ لا قائمةٌ واحدة
 *
 * **ما ذكره المالكُ ثلاثةُ أنواع**: ما يخصّه · ما يخصّ المنصّة ·
 * والقانونيّ. **وقائمةٌ تخلطها تصير درجاً يُرمى فيه كلُّ شيء** — فلا
 * يجد فيها شيئاً بسرعة.
 *
 * # وفارغةٌ الآن بأمره
 *
 * («ضع الأقسام فارغةً بدون أيّ شيء، ثمّ إذا اتّفقنا عليها نملأ
 *  الأقسام».)
 *
 * **وكلُّ بندٍ يُملأ وحدَه في دفعته** — والنداءاتُ كلُّها موجودةٌ في
 * المحرّك (`incentives` · `my/tickets` · `my/chats`)، **فالعملُ ربطٌ
 * لا بناء.**
 */
@Composable
fun MenuDrawer(onPick: (MenuItem) -> Unit, onLogout: () -> Unit) {
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(vertical = 8.dp),
    ) {
        var lastGroup: Int? = null
        for (item in MenuItem.entries) {
            if (item.group != lastGroup) {
                if (lastGroup != null) {
                    Spacer(Modifier.height(6.dp))
                    HorizontalDivider()
                }
                Spacer(Modifier.height(12.dp))
                Text(
                    text = stringResource(item.group),
                    color = BrandOrange,
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier.padding(horizontal = 14.dp),
                )
                Spacer(Modifier.height(4.dp))
                lastGroup = item.group
            }
            Line(item, onPick)
        }

        // ══════════════════════════════════════════════════════════════
        // **والخروجُ في القاع — بعيداً عن طريق الإبهام**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «برأيك زرُّ تسجيل الخروج وين مكانه
        //  الصحيح؟ أيضاً بالقائمة الجانبيّة بالأسفل صحيح — هذا أفضل
        //  مكانٍ له».)
        //
        // **وثلاثةُ أسبابٍ تجعله صحيحا:**
        //
        // **١ · القاعُ آخرُ ما يبلغه الإبهام** — وفعلٌ يُخرجه من حسابه
        // لا يُوضع في طريق مرور. **ومن خرج سهواً يعود بكلمة مرورٍ قد
        // لا يحفظها.**
        //
        // **٢ · وهو حيث يتوقّعه** — كلُّ تطبيقٍ يضعه هناك، **فيُوجَد
        // بلا بحث.**
        //
        // **٣ · وموضعٌ واحدٌ لا موضعان** — كان في لوحة العمل، **ورُفع
        // منها**: فعلٌ في مكانين يُنسى أحدُهما فيبقى قديماً حين يتبدّل.
        //
        // **وأحمرُ بحدٍّ فوقه** — لا يُخلط بما قبله من أسماء أقسام.
        Spacer(Modifier.height(10.dp))
        HorizontalDivider()
        Row(
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onLogout)
                .padding(horizontal = 14.dp, vertical = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_logout),
                contentDescription = null,
                tint = StateRed,
                modifier = Modifier.size(20.dp),
            )
            Spacer(Modifier.size(12.dp))
            Text(
                stringResource(R.string.login_logout),
                color = StateRed,
                style = MaterialTheme.typography.bodyLarge,
            )
        }
        Spacer(Modifier.height(16.dp))
    }
}

@Composable
private fun Line(item: MenuItem, onPick: (MenuItem) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable { onPick(item) }
            .padding(horizontal = 14.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(item.icon),
            contentDescription = null,
            tint = InkMuted,
            modifier = Modifier.size(20.dp),
        )
        Spacer(Modifier.size(10.dp))
        Text(
            text = stringResource(item.label),
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
        )
    }
}

/**
 * **بنودُ القائمة** — بمجموعتها وأيقونتها.
 *
 * **والترتيبُ داخل «ما يخصّني» بحسب ما يُفتح**: السجلُّ يوميّاً،
 * والدردشاتُ عند خلاف، والهدايا آخرَ الشهر، **والشكاوى نادرا.**
 */
enum class MenuItem(val group: Int, val label: Int, val icon: Int) {
    History(R.string.menu_mine, R.string.menu_history, R.drawable.ic_history),
    // ══════════════════════════════════════════════════════════════════
    // **وصندوقي — النقدُ الذي بذمّته للمنصّة**
    // ══════════════════════════════════════════════════════════════════
    //
    // (شكوى المالك ٢٠٢٦-٠٨-١٣: «صندوقي غير موجودٍ أيضاً، اطّلع عليه من
    //  الويب لتعرف لماذا قسم صندوقي».)
    //
    // **يقبض نقداً من الزبائن فيتراكم عليه** حتّى يسلّمه للمكتب،
    // **وله سقفٌ يتوقّف عنده الطابور** فلا تصله طلباتٌ نقديّة.
    //
    // **ومالٌ في ذمّة إنسانٍ بلا كشفٍ يقرؤه خلافٌ ينتظر**: يقول
    // «سلّمتُ» وتقول المنصّةُ «لم يصل»، **ولا ورقةَ بينهما.**
    // (وهو نصُّ شاشة الويب نفسِه — `driver/cash`.)
    //
    // **وهو غيرُ المحفظة**: المحفظةُ ماله، **والصندوقُ مالُ غيره في
    // يده.**
    Cash(R.string.menu_mine, R.string.menu_cash, R.drawable.ic_cash),
    Chats(R.string.menu_mine, R.string.menu_chats, R.drawable.ic_chat),
    // **وأهدافي لا الهدايا** — (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «ليس الهدايا
    // والمكافآت إنّما أهدافي والمكافآت، لأنّ كلَّ سائقٍ لديه أهدافٌ إذا
    // حقّقها يأخذ مكافأة»).
    //
    // **والهديّةُ تُعطى والهدفُ يُبلَغ** — ومن قرأ «هدايا» انتظر عطاءً
    // لا يجيء، **ومن قرأ «أهدافي» عرف أنّ عليه عملا.**
    Rewards(R.string.menu_mine, R.string.menu_rewards, R.drawable.ic_star),
    Tickets(R.string.menu_mine, R.string.menu_tickets, R.drawable.ic_warning),

    Help(R.string.menu_platform, R.string.menu_help, R.drawable.ic_info),
    About(R.string.menu_platform, R.string.menu_about, R.drawable.ic_info),
    Contact(R.string.menu_platform, R.string.menu_contact, R.drawable.ic_phone),

    Terms(R.string.menu_legal, R.string.menu_terms, R.drawable.ic_info),
    Privacy(R.string.menu_legal, R.string.menu_privacy, R.drawable.ic_lock),
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ البند — كلُّ بندٍ يفتح ما يخصّه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ بملء الأقسام من الويب».)
 *
 * **وكانت تقول «هذا القسم لم يُملأ بعد»** — وهو أصدقُ من شاشةٍ بيضاء،
 * **ولا يبقى منه شيءٌ الآن.** فمن فتح بنداً وجد فيه ما وعده اسمُه.
 *
 * **والسجلُّ مبنيٌّ قبلها** فيُمرَّر كما هو — ولا يُبنى مرّتين.
 */
@Composable
fun MenuScreen(vm: SectionsViewModel, item: MenuItem) {
    // **ويُجلب ما يخصُّ هذا البندَ وحدَه** — عند فتحه لا عند إقلاع
    // التطبيق. **وسبعةُ نداءاتٍ لسبعة أقسامٍ لا يُفتح منها واحد**
    // تستنزف حزمةَ سائقٍ في الشارع.
    LaunchedEffect(item) { vm.open(item) }
    when (item) {
        MenuItem.Cash -> CashScreen(vm)
        MenuItem.Chats -> ChatsScreen(vm)
        MenuItem.Rewards -> IncentivesScreen(vm)
        MenuItem.Tickets -> ComplaintsScreen(vm)
        MenuItem.Contact -> ContactScreen(vm)
        MenuItem.Help, MenuItem.About, MenuItem.Terms, MenuItem.Privacy -> PageScreen(vm, item)
        // **والسجلُّ يُعرَض من نموذجه هو** — تُمرّره الشاشةُ الأمّ.
        MenuItem.History -> Unit
    }
}
