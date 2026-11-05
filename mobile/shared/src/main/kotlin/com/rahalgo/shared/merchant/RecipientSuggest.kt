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

    /**
     * **مفتاحُ الرقم مهما كُتب** — الأرقامُ الهنديّةُ لاتينيّة، والفواصلُ تُطرح، و`09…` و`+963…`
     * و`00963…` رقمٌ واحد (`9…`). **مقارنةٌ لا تطبيع** — الخادمُ يطبّع بـ`NormalizePhone`.
     */
    fun phoneKey(phone: String): String {
        val d = buildString {
            for (c in phone) {
                when (c) {
                    in '0'..'9' -> append(c)
                    in '\u0660'..'\u0669' -> append('0' + (c - '\u0660'))
                    in '\u06F0'..'\u06F9' -> append('0' + (c - '\u06F0'))
                }
            }
        }
        return when {
            d.startsWith("00963") -> d.drop(5)
            d.startsWith("963") -> d.drop(3)
            d.startsWith("0") -> d.drop(1)
            else -> d
        }
    }

    /** **أهو الرقمُ نفسُه؟** — وفارغٌ لا يساوي شيئاً. */
    fun samePhone(a: String, b: String): Boolean {
        val k = phoneKey(a)
        return k.isNotEmpty() && k == phoneKey(b)
    }

    /**
     * **أتبقى نقطةُ المستلِم المختار؟** (مراجعةُ ٢٠٢٦-١١-٠٦) — ما دام الرقمُ رقمَه والعنوانُ كما
     * مُلئ. **ومن بدّل الرقمَ أو كتب في العنوان فهو شخصٌ آخرُ أو مكانٌ آخر** — ونقطةٌ قديمةٌ تُرسل
     * السائقَ إلى غير مكانه، فتسقط ويعود الطلبُ بلا نقطةٍ كما اليوم.
     */
    fun keepsSavedPoint(picked: Fill, phone: String, address: String): Boolean =
        samePhone(picked.phone, phone) && address == picked.address

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
