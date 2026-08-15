package com.rahalgo.customer

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import com.rahalgo.ui.ContactPage
import com.rahalgo.ui.DrawerItem
import com.rahalgo.ui.PagesViewModel
import com.rahalgo.ui.PlatformPage
import com.rahalgo.ui.PlatformPages
import com.rahalgo.ui.R

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ الزبون الجانبيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «على القائمة الجانبيّة نضيف: المفضّلة ·
 *  العروض · ادعُ صديقاً · الشكاوى والبلاغات».)
 *
 * # وأبوابُها كلُّها موجودةٌ في المحرّك
 *
 * `my/favorites` · `public/offers` · `auth/referral` · `my/tickets` —
 * **يقرؤها الويبُ اليوم**، فالعملُ ربطٌ لا بناء.
 *
 * **وكلُّ بندٍ يُملأ وحدَه في دفعته** — (قاعدةُ `CLAUDE.md`: دفعةٌ
 * واحدةٌ في كلّ مرّة).
 */
object CustomerItems {
    const val HISTORY = "History"
    const val CHATS = "Chats"
    const val FAVORITES = "Favorites"
    const val OFFERS = "Offers"
    const val INVITE = "Invite"
    const val TICKETS = "Tickets"
}

/**
 * **بنودُ «ما يخصّني» — أربعةٌ بأمر المالك** (٢٠٢٦-٠٨-١٤).
 *
 * **وما ليس فيها عمدا**: طلباتي والطلبُ الخاصُّ وحسابي **تبويباتٌ في
 * الأسفل** — **وفعلٌ في موضعين يُنسى أحدُهما فيبقى قديماً حين يتبدّل.**
 *
 * **والترتيبُ بحسب ما يُفتح**: السجلُّ أوّلا (أين فاتورةُ طلبِ أمس؟)،
 * ثمّ المفضّلةُ عند كلّ طلب، والعروضُ حين يبحث عن سعر، والدردشاتُ عند
 * خلاف، والدعوةُ مرّةً أو مرّتين، **والشكوى نادرا** — ومن يشتكي كثيراً
 * فالعلّةُ في الخدمة لا في موضع الزرّ.
 *
 * # وسجلُّ الطلبات غيرُ تبويب «طلباتي»
 *
 * **التبويبُ لما يجري الآن** — أين وصل طلبُه. **والسجلُّ لما انتهى** —
 * ما سُلّم وما تعذّر بفواتيره.
 *
 * **ولو جُمعا في موضعٍ واحد** لَبحث عن طلبٍ يجري وسط مئةٍ منتهية،
 * **أو فتح «طلباتي» وهو فارغٌ فظنّ أنّ طلبَه ضاع.**
 */
val CUSTOMER_ITEMS: List<DrawerItem> = listOf(
    DrawerItem(CustomerItems.HISTORY, R.string.menu_mine, R.string.menu_history, R.drawable.ic_history),
    DrawerItem(CustomerItems.FAVORITES, R.string.menu_mine, R.string.menu_favorites, R.drawable.ic_heart),
    DrawerItem(CustomerItems.OFFERS, R.string.menu_mine, R.string.menu_offers, R.drawable.ic_offer),
    DrawerItem(CustomerItems.CHATS, R.string.menu_mine, R.string.menu_chats, R.drawable.ic_chat),
    DrawerItem(CustomerItems.INVITE, R.string.menu_mine, R.string.menu_invite, R.drawable.ic_invite),
    DrawerItem(CustomerItems.TICKETS, R.string.menu_mine, R.string.menu_tickets, R.drawable.ic_warning),
)


