package com.rahalgo.merchant.orders

import com.rahalgo.shared.merchant.MerchantOrder

/**
 * **عددُ المحلّات في الطلب المشترَك** — يُعرض «قسمك من طلب فيه ٣ محلات».
 *
 * **و`null` لا تسمية**: الحقلُ غائبٌ (طلبٌ عاديّ) أو أقلُّ من محلّين.
 */
internal fun sharedStopsCount(order: MerchantOrder): Int? =
    order.sharedStops?.takeIf { it >= 2 }
