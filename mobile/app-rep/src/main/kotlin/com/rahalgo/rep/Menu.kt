package com.rahalgo.rep

import com.rahalgo.ui.DrawerItem

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ المندوب الجانبيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وما ليس فيها عمدا**: لوحتُه وعملاؤه وحسابُه **تبويباتٌ في الأسفل**
 * — **وفعلٌ في موضعين يُنسى أحدُهما فيبقى قديماً حين يتبدّل.**
 *
 * **والترتيبُ بحسب ما يُفتح**: رابطُ الدعوة كلَّما لقي صاحبَ متجر،
 * **والأهدافُ آخرَ الشهر.**
 *
 * # ولا دردشاتِ له
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-١٤: «المندوبُ لا يملك دردشات، والدردشاتُ فقط
 *  بين السائق والزبون حاليّاً».)
 *
 * **وبندٌ يُعرض لمن لا يستعمله يُفتح مرّةً فيُوجد فارغا** — ثمّ يبقى في
 * القائمة يشغل موضعاً ويُقرأ عطبا.
 */
object RepItems {
    const val LINK = "Link"
    const val REWARDS = "Rewards"
}

val REP_ITEMS: List<DrawerItem> = listOf(
    DrawerItem(
        RepItems.LINK,
        com.rahalgo.ui.R.string.menu_mine,
        R.string.menu_link,
        com.rahalgo.ui.R.drawable.ic_invite,
    ),
    DrawerItem(
        RepItems.REWARDS,
        com.rahalgo.ui.R.string.menu_mine,
        com.rahalgo.ui.R.string.menu_rewards,
        com.rahalgo.ui.R.drawable.ic_star,
    ),
)
