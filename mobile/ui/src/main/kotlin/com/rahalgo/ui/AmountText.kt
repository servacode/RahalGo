package com.rahalgo.ui

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مبلغُ «لدي توصيلة» و«إحضار» بعملته** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المتجرُ يختار ليرة أو دولار** للمبلغ يلي بيقبضه السائق أو بيدفعه — **والخادمُ يحفظ
 * الدولارَ بالسنتات** (١٢٥٠ = ١٢٫٥٠$) **والليرةَ صحيحةً كما هي.**
 *
 * **فلا يُعرض المبلغُ بـ`money` وحدَه** — دولارٌ بسنتاته يصير «1,250 ل.س». **وهذا المكانُ
 * الوحيد** الذي يعرف القسمةَ على مئة، وهو نصُّ إشعار الخادم نفسُه (`orders.AmountText`).
 */
fun amountText(units: Long, currency: String?): String {
    if (!isUsd(currency)) return money(units)
    val negative = units < 0
    val abs = if (negative) -units else units
    val whole = grouped(abs / 100)
    val cents = abs % 100
    val text = if (cents == 0L) "$whole$" else "$whole.${cents.toString().padStart(2, '0')}$"
    return if (negative) "-$text" else text
}

/** **أدولارٌ؟** — وفراغُ العملة ليرة (تطبيقٌ قديمٌ لا يرسلها). */
fun isUsd(currency: String?): Boolean = currency == "USD"

/**
 * **مبلغٌ مكتوبٌ باليد كما يُرسَل** — الليرةُ أرقامٌ فقط، **والدولارُ بفاصلةٍ واحدةٍ ومنزلتين.**
 *
 * **والفاصلةُ تُقبل بأشكالها الثلاثة** (`.` و`,` و«٫» العربيّة) وتصير نقطة — لوحةُ مفاتيحٍ عربيّةٌ
 * تكتب «٫» والإنكليزيّةُ «.». **وما بعد المنزلتين يُترك**، ولا تُقبل فاصلةٌ ثانية.
 */
fun amountInput(raw: String, currency: String?, maxDigits: Int = 12): String {
    if (!isUsd(currency)) return raw.filter { it.isDigit() }.take(maxDigits)
    val out = StringBuilder()
    var dot = -1
    for (c in raw) {
        when {
            c in "0123456789" -> {
                if (dot >= 0 && out.length - dot > 2) continue
                if (dot < 0 && out.length >= maxDigits) continue
                out.append(c)
            }
            (c == '.' || c == ',' || c == '٫') && dot < 0 -> {
                if (out.isEmpty()) out.append('0')
                dot = out.length
                out.append('.')
            }
        }
    }
    return out.toString()
}

/** **قيمةُ المكتوب** بالوحدة الكبرى (١٢٫٥ لا ١٢٥٠) — و`null` لفراغٍ أو صفر. */
fun amountValue(text: String): Double? = text.toDoubleOrNull()?.takeIf { it > 0 }
