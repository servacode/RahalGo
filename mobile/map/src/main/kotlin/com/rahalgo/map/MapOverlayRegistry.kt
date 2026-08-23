package com.rahalgo.map

/**
 * ══════════════════════════════════════════════════════════════════
 * **الطبقاتُ فوق النمط — تُعاد بعد كلّ تحميل**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ١٩ و٤١.)
 *
 * # **النقطةُ التي سمّاها المالكُ «شديدةَ الأهمّيّة»**
 *
 * **عند تحميل نمطٍ جديدٍ تُسقط MapLibre كلَّ ما فوقه**: مصدرَ المسار،
 * وطبقاتِه، ودبّوسَ السائق، ونقطتَي الاستلام والتسليم، **وطبقةَ
 * مناطق التوصيل.**
 *
 * **ولا تردّ خطأً** — الخريطةُ تُرسم سليمةً **وفوقها لا شيء.**
 * فالسائقُ يرى شوارعَ بلا مسار، **ولا شيءَ في سجلٍّ يقول لماذا.**
 *
 * **وذلك يقع في ٦ب أكثرَ من قبل**: كنّا لا نبدّل النمطَ أبداً،
 * **وصرنا نبدّله كلّما ذهبت الشبكةُ أو عادت.**
 *
 * # **فالحلُّ سجلٌّ لا نداءٌ في مكانه**
 *
 * **كلُّ من يرسم فوق النمط يسجّل كيف يُعيد رسمَ نفسِه.** وعند كلّ
 * `StyleLoaded` **يُنادى السجلُّ بالترتيب**:
 *
 *	نمطُ الأساس  →  طبقاتُ رحّال غو  →  الحالُ الراهنة
 *
 * **والترتيبُ يُحفظ بالأولويّة لا بترتيب التسجيل** — فالتسجيلُ يقع
 * بترتيب إنشاء الشاشة، **وهو ليس ترتيبَ الرسم المطلوب** (البند ٤٠).
 */
class MapOverlayRegistry {

    /**
     * **مُعيدُ تركيبٍ واحد.**
     *
     * **لا يرسم شيئاً بنفسه هنا** — الواجهةُ مُجرَّدةٌ عن MapLibre
     * ليُختبر السجلُّ بلا جهاز (البند ٤٩).
     */
    fun interface Restorer {
        fun restore()
    }

    /**
     * **والأولويّةُ هي ترتيبُ الرسم** — الأصغرُ أوّلاً.
     *
     * **المسارُ تحتَ الدبابيس** — وإلّا مرَّ الخطُّ فوق دبّوس التسليم
     * فأخفاه.
     */
    object Priority {
        const val ZONES = 100
        const val ROUTE = 200
        const val MARKERS = 300
    }

    private data class Entry(val id: String, val priority: Int, val restorer: Restorer)

    private val entries = LinkedHashMap<String, Entry>()

    var restoreCount: Int = 0
        private set

    /**
     * **يسجّل مُعيداً بمعرِّف.**
     *
     * **والمعرِّفُ يمنع التكرار** — إعادةُ تركيب Compose تُنادي هذا
     * كثيراً، **ولو أُضيف في كلّ مرّةٍ لتراكمت مئاتُ المُعيدين
     * لطبقةٍ واحدة.**
     */
    fun register(id: String, priority: Int, restorer: Restorer) {
        entries[id] = Entry(id, priority, restorer)
    }

    fun unregister(id: String) {
        entries.remove(id)
    }

    fun size(): Int = entries.size

    fun ids(): List<String> = ordered().map { it.id }

    /**
     * **يُنادى بعد كلّ `StyleLoaded`.**
     *
     * **ولا يُسقط الباقين إن سقط واحد** — طبقةُ مناطقِ توصيلٍ تعطّلت
     * **لا يجوز أن تُخفيَ خطَّ المسار.**
     */
    fun restoreAll(onError: ((String, Throwable) -> Unit)? = null) {
        restoreCount += 1
        for (e in ordered()) {
            try {
                e.restorer.restore()
            } catch (t: Throwable) {
                onError?.invoke(e.id, t)
            }
        }
    }

    private fun ordered(): List<Entry> =
        entries.values.sortedWith(compareBy({ it.priority }, { it.id }))

    fun clear() {
        entries.clear()
        restoreCount = 0
    }
}
