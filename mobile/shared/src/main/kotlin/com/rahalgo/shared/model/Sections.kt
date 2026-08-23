package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ القائمة الجانبيّة — نماذجُها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ابدأ بملء الأقسام من الويب».)
 *
 * **وكلُّها نداءاتُ الويب نفسُها** — لا مسارَ جديدٌ في المحرّك ولا حقل:
 * **شاشتان تقرآن رقماً واحداً لا تفترقان.**
 */

// ══════════════════════════════════════════════════════════════════════
//  صندوقي — النقدُ الذي بذمّته
// ══════════════════════════════════════════════════════════════════════

/**
 * **سطرٌ في كشف الصندوق.**
 *
 * **والموجبُ قبضٌ يزيد ذمّتَه، والسالبُ تسليمٌ يُنقصها** — وهو ما تقوله
 * شاشةُ الويب بسهمَين.
 */
@Serializable
data class CashEntry(
    val kind: String = "",
    val amount: Long = 0,
    val note: String = "",
    @SerialName("order_number") val orderNumber: Long? = null,
    /**
     * **مِمَّن قبض** — باسمه لا بصفته.
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «مكتوبٌ قبضتُ من زبون، وهذا غلط —
     *  أساساً هو معروف».)
     *
     * **و«قبضتُ من زبون» ثلاثَ مرّاتٍ في يومٍ لا تُميّز واحدةً من
     * أخرى** — ومن اختلف على مبلغٍ لا يجد في كشفه ما يشير إلى أحد.
     *
     * **وفارغٌ لقيدٍ بلا طلب** — تسويةُ إدارةٍ لا صاحبَ لها.
     */
    @SerialName("customer_name") val customerName: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class CashPage(
    val entries: List<CashEntry> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    @SerialName("per_page") val perPage: Int = 25,
)

// ══════════════════════════════════════════════════════════════════════
//  أهدافي والمكافآت
// ══════════════════════════════════════════════════════════════════════

/**
 * **موقفُه من هدف الشهر.**
 *
 * **والبلوغُ يقوله المحرّك ولا يُستنتج هنا** — شرطٌ يُحسب في موضعين
 * يفترق يوماً، **فتُهنّئ الشاشةُ من لم يبلغ.**
 */
@Serializable
data class Standing(
    val done: Int = 0,
    val target: Int = 0,
    val reached: Boolean = false,
    val rewarded: Long = 0,
    val penalized: Long = 0,
)

/** **مكافأةٌ أو عقوبة — بسببها ومن قرّرها.** */
@Serializable
data class IncentiveEntry(
    val id: String = "",
    /** `reward` أو `penalty`. */
    val kind: String = "",
    val amount: Long = 0,
    val reason: String = "",
    @SerialName("for_target") val forTarget: Boolean = false,
    val by: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **ما يردّه المحرّك.**
 *
 * **و`standing` قد يغيب** — وردٌّ ناقصٌ كان يُبيّض صفحةَ الويب
 * (`Cannot read properties of undefined`). **فحقلٌ ناقصٌ يُقرأ صفراً
 * أهونُ من شاشةٍ تسقط.**
 */
@Serializable
data class IncentivesPayload(
    val standing: Standing = Standing(),
    val entries: List<IncentiveEntry> = emptyList(),
    /** **مكافأةُ بلوغ الهدف** — تُقال قبل أن يُبلَغ. صفرٌ يعني بلا وعد. */
    @SerialName("target_reward") val targetReward: Long = 0,
)

// ══════════════════════════════════════════════════════════════════════
//  صفحاتُ المنصّة — التعليمات ومن نحن والشروط والخصوصيّة
// ══════════════════════════════════════════════════════════════════════

/**
 * **هويّةُ المنصّة ونصوصُ صفحاتها** (`GET /api/v1/public/contact`).
 *
 * **والنصُّ نافذٌ لا افتراضيّ**: المحرّكُ يردّ ما حُرِّر من اللوحة،
 * **وإلّا أصلَه المكتوبَ فيه** (`settings/pagetext.go`). **فالوثيقةُ
 * التي يوافق عليها السائقُ في التطبيق هي التي يقرؤها الزبونُ في
 * الموقع.**
 */
@Serializable
data class SiteContact(
    @SerialName("legal_name") val legalName: String = "",
    @SerialName("support_phone") val supportPhone: String = "",
    val address: String = "",
    @SerialName("help_text") val helpText: String = "",
    @SerialName("terms_text") val termsText: String = "",
    @SerialName("privacy_text") val privacyText: String = "",
    @SerialName("about_text") val aboutText: String = "",
    @SerialName("driver_help_text") val driverHelpText: String = "",
    /** **وتعليماتُ المندوب ثالثة** — لا ورديّةَ له ولا سلّة. */
    @SerialName("rep_help_text") val repHelpText: String = "",
    /** **وتعليماتُ المتجر رابعة** — بُني تطبيقُه ٢٠٢٦-٠٨-٢٣. */
    @SerialName("merchant_help_text") val merchantHelpText: String = "",
)

/**
 * **كتلةُ نصٍّ — عنوانٌ وفقرات.**
 *
 * **والاتّفاقُ أبسطُ ما يمكن**: سطرٌ فارغٌ يفصل كتلةً عن كتلة، **وأوّلُ
 * سطرٍ عنوانٌ** وما بعده فقرات. **وكتلةٌ بسطرٍ واحدٍ فقرةٌ بلا عنوان** —
 * ولا يُجعل وحيدُها عنواناً فيبقى بلا متن.
 *
 * **وهو قارئُ الويب نفسُه** (`parseBlocks`) — نصٌّ واحدٌ يُقرأ قراءتين
 * متطابقتين.
 */
data class TextBlock(val head: String, val body: List<String>)

/**
 * **يقرأ النصَّ إلى كتل** — ويملأ قوالبَ الهويّة.
 *
 * **ولا يُترك قالبٌ ظاهراً لقارئ**: من رأى «{name} منصّةُ توصيل» قرأ
 * عطباً، **ومن رأى «{phone}» لم يجد رقماً يتّصل به.**
 */
fun parseBlocks(
    text: String,
    name: String = "",
    phone: String = "",
    address: String = "",
): List<TextBlock> {
    val filled = text
        .replace("{name}", name)
        .replace("{phone}", phone)
        .replace("{address}", address)
        .replace("\r\n", "\n")
    return filled.split(Regex("\\n\\s*\\n")).mapNotNull { chunk ->
        val lines = chunk.split("\n").map(String::trim).filter(String::isNotEmpty)
        when {
            lines.isEmpty() -> null
            lines.size == 1 -> TextBlock("", lines)
            else -> TextBlock(lines.first(), lines.drop(1))
        }
    }
}
