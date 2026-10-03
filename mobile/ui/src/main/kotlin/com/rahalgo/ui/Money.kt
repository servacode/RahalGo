package com.rahalgo.ui

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

// ══════════════════════════════════════════════════════════════════════
// **والمسافةُ والزمنُ هنا أيضا — لا في كلّ شاشة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم أصلحها».)
//
// **وكانت `distance` مكتوبةً في شاشتين بالنصّ نفسِه** — في الطلبات
// وفي الرحلة. **ومن صحّح واحدةً ترك الأخرى**، وهو عينُ ما تمنعه
// المركزيّة.
//
// **وهذا الملفُّ هو مصدرُ الوحدات** — يعرفه حارسُ النصوص
// (`scripts/check-strings.py`) فلا يشكو منه، **ويشكو من غيره.**

/**
 * **مسافةٌ كما تُقرأ** — «٥٦٠ م» أو «١٫٢ كم».
 *
 * **وبأرقام غربيّة**: `format` بلا لغةٍ يكتب «١٫٠ كم» في جهازٍ عربيّ،
 * **فيقع رقمان بخطّين في السطر نفسِه.**
 *
 * **وسالبٌ يعني «لا يُعرف»** — فيُردّ فارغاً ولا يُكتب «‎-١ م».
 */
fun dist(meters: Double): String {
    if (meters < 0) return ""
    val m = meters.toLong()
    return if (m < 1000) {
        "$m م"
    } else {
        "%.1f كم".format(java.util.Locale.US, m / 1000.0)
    }
}

/** **دقائقُ كما تُقرأ** — «٧ دقيقة». */
fun minutes(mins: Long): String = "$mins دقيقة"

/** **دقائقُ مختصرة** — «٧ د»، لِما يقع في خانةٍ ضيّقة. */
fun minutesShort(mins: Long): String = "$mins د"

/**
 * **زمنُ الوصول من مسافةٍ وسرعة** — «‏ · ~٧ د»، وفارغٌ إن لم يُعرف.
 *
 * **والسرعةُ سرعةُ السائق نفسِه** (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الوقتُ
 * والسرعةُ تتحدّد من سرعة الموتور الحقيقيّ لا من إعدادات الأدمن»).
 */
fun etaText(meters: Double, avgSpeedKmh: Long): String {
    if (avgSpeedKmh <= 0 || meters < 0) return ""
    val mins = ((meters / 1000.0) / avgSpeedKmh * 60).toLong().coerceAtLeast(1)
    return " · ~" + minutesShort(mins)
}

// ══════════════════════════════════════════════════════════════════════
// **والتاريخُ هنا أيضا — صيغةٌ واحدةٌ في كلّ كشف**
// ══════════════════════════════════════════════════════════════════════
//
// (قاعدةُ المالك ٢٠٢٦-٠٨-١٣: «ركّز جيّداً على المركزيّة بكلّ شيء».)
//
// **وكانت `fmtWhen` مكتوبةً في المحفظة وحدَها** — ثمّ يحتاجها الصندوقُ
// والمكافآتُ والدردشاتُ والشكاوى. **وخمسُ نسخٍ من قصِّ نصٍّ تفترق يومَ
// يتبدّل شكلُ التاريخ في واحدةٍ منها.**

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وقتُ سوريا — منطقةٌ واحدةٌ لكلّ ما يُعرض**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: الشكوى قالت «فُتحت 16:51» والساعةُ 19:51،
 *  والمحفظةُ «18:09» والساعةُ 21:09 — **ثلاثُ ساعاتٍ متأخّرة**، والدردشةُ
 *  وحدَها صحيحة.)
 *
 * **وكان النصُّ يُقصّ بلا تحويل** على ظنِّ أنّ الطابعَ يحمل إزاحةَ دمشق —
 * **والمحرّكُ يرسله بتوقيت غرينتش (`…Z`)**، فتُقرأ ساعةُ غرينتش ساعةَ دمشق.
 *
 * **فيُحلَّل الطابعُ ويُحوَّل إلى دمشق** — وهي منطقةُ الدردشة نفسُها
 * (`ChatBubble`) التي كانت صحيحة. **وطابعٌ بإزاحة دمشق أصلاً لا يتزحزح:
 * التحويلُ من الشيء إلى نفسه لا يغيّره.**
 */
val SYRIA_ZONE: java.time.ZoneId = java.time.ZoneId.of("Asia/Damascus")

/**
 * **الطابعُ بتوقيت سوريا** — و`null` إن لم يُفهم.
 *
 * **ويُقبل بإزاحةٍ أو بـ`Z`** — وهما ما يرسله المحرّك.
 */
fun syriaTime(iso: String): java.time.ZonedDateTime? =
    runCatching { java.time.OffsetDateTime.parse(iso.trim()).atZoneSameInstant(SYRIA_ZONE) }.getOrNull()

private val DATE_FMT = java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd", java.util.Locale.US)
private val TIME_FMT = java.time.format.DateTimeFormatter.ofPattern("HH:mm", java.util.Locale.US)

/**
 * **تاريخُ الحركة ووقتُها** — كما يقرؤها صاحبُها: «2026-08-13 · 03:12».
 *
 * **والمحرّكُ يرسله بصيغة ISO** — وهي صيغةُ آلةٍ لا تُعرض. **ويُحوَّل إلى
 * توقيت سوريا** (`syriaTime`). **وما لا يُحلَّل يُقصّ كما كان** — ولا يُخترَع.
 */
fun whenText(iso: String): String {
    syriaTime(iso)?.let { return it.format(DATE_FMT) + " · " + it.format(TIME_FMT) }
    if (iso.length < 16) return iso
    return iso.substring(0, 10) + " · " + iso.substring(11, 16)
}

/** **الوقتُ وحدَه** — «03:12»، لِما يقع تحت عنوان يومٍ يقول تاريخَه. */
fun timeText(iso: String): String {
    syriaTime(iso)?.let { return it.format(TIME_FMT) }
    return if (iso.length < 16) iso else iso.substring(11, 16)
}

/**
 * **اسمُ اليوم** — «اليوم» و«أمس» ثمّ التاريخ.
 *
 * **ومن راجع ورديّتَه مساءً يريد أن يرى «اليوم» وحدَه** — وكشفٌ متّصلٌ
 * من مئة سطرٍ لا يُراجَع. **وهو نصُّ شاشة الويب نفسُه.**
 */
fun dayText(iso: String, today: java.time.LocalDate = java.time.LocalDate.now(SYRIA_ZONE)): String {
    // **واليومُ يومُ دمشق لا يومُ غرينتش** — حركةُ الواحدة ليلاً بتوقيت
    // دمشق هي العاشرةُ مساءَ أمسِ بغرينتش، **فتُعدّ في يومٍ ليس يومَها.**
    val at = syriaTime(iso)?.toLocalDate()
        ?: runCatching { java.time.LocalDate.parse(iso.take(10)) }.getOrNull()
        ?: return iso
    val date = at.format(DATE_FMT)
    return when (java.time.temporal.ChronoUnit.DAYS.between(at, today)) {
        in Long.MIN_VALUE..0L -> "اليوم"
        1L -> "أمس"
        else -> date
    }
}
