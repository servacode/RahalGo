package com.rahalgo.ui

/**
 * **عزلُ الأرقام والرموز داخل الجملة العربيّة عند العرض** — لا يُغيَّر النصُّ نفسُه.
 *
 * (ملاحظةُ المالك ٢٠٢٦-١١-١٠: «طلب #1201 – +100 ل.س» تنقلب، و«الـ5» تلتصق.)
 * **كلُّ `#1201` وكلُّ مبلغٍ (`+100 ل.س`) وكلُّ رقمٍ يُحاط بعازلٍ (`FSI … PDI`)** فيبقى وحدةً
 * ولا يجرّ ما حولَه معه. **ونصوصُ الخادم تبقى كما وصلت** — العزلُ هنا في الرسم وحدَه.
 */
object BidiText {
    private const val FSI = "\u2068"
    private const val PDI = "\u2069"
    private val token = Regex("[+\\-−]?#?\\d[\\d.,:/]*(?:\\s?ل\\.س)?")

    fun isolateNumbers(text: String): String =
        if (text.isEmpty()) text else token.replace(text) { FSI + it.value + PDI }
}
