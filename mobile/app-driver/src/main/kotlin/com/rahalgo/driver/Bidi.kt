package com.rahalgo.driver

/**
 * **عزلُ الاتّجاه** — نصٌّ لاتينيٌّ أو رقميٌّ داخل جملةٍ عربيّة يُعزَل فلا يقلب ما حوله.
 *
 * (ملاحظةُ المالك ٢٠٢٦-١١-١٠: «مدين للمكتب بـ … ل.س» تتفكّك، و«#1201 طلب» تنعكس،
 * واسمُ متجرٍ إنكليزيٌّ يخلط ترتيبَ السطر.) **`FSI … PDI`: الاتّجاهُ يُستنتج من المحتوى
 * ويبقى المعزولُ وحدةً واحدة.**
 */
object Bidi {
    private const val FSI = '\u2068'
    private const val PDI = '\u2069'

    /** **يعزل `text`** — وفارغٌ يبقى فارغاً. */
    fun isolate(text: String): String = if (text.isEmpty()) text else "$FSI$text$PDI"
}
