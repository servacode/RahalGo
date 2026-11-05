package com.rahalgo.shared.merchant

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اقتراحُ المستلِم وقتَ الكتابة — قواعدُه بلا شاشة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بمجرد ما يكتب اسم الزبون يتعبى باقي المعلومات بشكل تلقائي عنوان موقع رقم وكلشي».
 *
 * **متى يُسأل الخادم** (حرفان فأكثر، بعد ٣٠٠ مللي ثانية من آخر حرف) **وما يملأ النموذجَ**
 * عند الاختيار — هنا لا في الشاشة، فيُقاس بلا جهاز (`RecipientSuggestTest`).
 */
object RecipientSuggest {

    /** **انتظارُ ما بعد آخر حرف** — لا نداءَ مع كلّ ضغطة. */
    const val DEBOUNCE_MS = 300L

    /** **أقلُّ ما يُبحث به** — حرفٌ واحدٌ يطابق نصفَ الدفتر. */
    const val MIN_CHARS = 2

    /** **أيُسأل الخادمُ عن هذا النصّ؟** — حرفان غيرُ الفراغ فأكثر. */
    fun shouldSearch(text: String): Boolean = text.count { !it.isWhitespace() } >= MIN_CHARS

    /**
     * **الرقمُ كما يكتبه الناس** — `+9639XXXXXXXX` ⇒ `09XXXXXXXX`، وغيرُه كما هو.
     * **والخادمُ يقبل الصيغتين** (`identity.NormalizePhone`).
     */
    fun localPhone(phone: String): String =
        if (phone.startsWith("+963") && phone.length == 13) "0" + phone.substring(4) else phone

    /** **ما يُملأ في النموذج** — والنقطةُ `null` إن أُرسلت آخرُ توصيلةٍ بلا نقطة. */
    data class Fill(
        val name: String,
        val phone: String,
        val address: String,
        val point: Pair<Double, Double>?,
    )

    fun fill(r: DeliveryRecipient): Fill {
        val lat = r.lat
        val lng = r.lng
        return Fill(
            name = r.name,
            phone = localPhone(r.phone),
            address = r.addressText,
            point = if (lat != null && lng != null) lat to lng else null,
        )
    }
}
