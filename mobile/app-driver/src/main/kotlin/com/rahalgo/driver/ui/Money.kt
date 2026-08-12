package com.rahalgo.driver.ui

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المبالغ — تُكتب بشكل واحد في كلّ شاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والأرقام لاتينيّة (0-9) لا عربيّة-هنديّة (٠-٩)** — قرار المشروع كلّه
 * (`web/packages/i18n/format.ts`: «الأرقام إنجليزية في كلّ المشروع»).
 * **ومن استعمل تنسيق النظام على جهاز لغته العربيّة** أخرج «١٢٬٥٠٠» في
 * شاشة كلّ أرقامها لاتينيّة.
 *
 * **ولذلك تُكتب الفاصلة يدويّا** — لا `NumberFormat` يتبع لغة الجهاز.
 *
 * **والعملة «ل.س»** كما في الويب (`locales/ar.json`).
 */
fun money(amount: Long): String = "${grouped(amount)} ل.س"

/** رقم بفواصل آلاف: `12,500` — **وسالبه يبقى سالبا.** */
fun grouped(amount: Long): String {
    val negative = amount < 0
    // **والقيمة الصغرى لا تُقلب**: `-Long.MIN_VALUE` تفيض وتبقى سالبة.
    // **ونصّها يُؤخذ كما هو** بدل أن يخرج رقم بلا معنى.
    val digits = if (amount == Long.MIN_VALUE) {
        amount.toString().removePrefix("-")
    } else {
        (if (negative) -amount else amount).toString()
    }
    val out = StringBuilder()
    for ((i, c) in digits.withIndex()) {
        if (i > 0 && (digits.length - i) % 3 == 0) out.append(',')
        out.append(c)
    }
    return if (negative) "-$out" else out.toString()
}
