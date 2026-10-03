package com.rahalgo.driver.orders

/**
 * **أيمحو نجاحُ القراءة هذا الخطأ؟** (تجربةُ القبول ٢٠٢٦-١٠-٠٣) — خطأُ الشبكة وحدَه.
 *
 * **«سبقك غيرُك» يبقى حتّى يقرأه** — نجاحُ قراءةٍ لا يجيب عنه. **و«لا اتصال بالإنترنت»
 * يجيب عنه أيُّ نداءٍ نجح** — فبقاؤه بعدها كذبٌ على الشاشة.
 */
internal object NetworkNotice {
    fun clears(error: String, networkText: String): Boolean =
        error.isNotEmpty() && error == networkText
}
