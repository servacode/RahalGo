package com.rahalgo.map.data

import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **حجزُ الأرشيف — والاستعمالُ يُعرف لا يُخمَّن**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ ٦ب الوظيفيّ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٨ إلى ١٢.)
 *
 * **أمرُ المالك نصّاً**: «لا تعتمد على `network online?` أو
 * `alternative exists?` لتحديد هل الملف مفتوح».
 *
 * # **الدَّينُ الذي يُغلق هنا**
 *
 * `OfflineMap.activeArchive()` **كانت تردّ `null`.** فالحمايةُ من حذفِ
 * حزمةٍ قيدَ الاستعمال كانت تستدلّ بالاتّصال وبوجود بديل — **وكلاهما
 * ظنٌّ لا خبر.**
 *
 * **وسائقٌ متّصلٌ قد تكون خريطتُه ما زالت على الحزمة المحلّيّة**
 * (سياسةُ التبديل تُبقيه عليها أثناء الملاحة، البند ١٨ من ٦ب).
 * **فيُحذف الملفُّ من تحتها وهي تقرأ منه.**
 *
 * # **والمطلوبُ سلوكٌ لا اسم**
 *
 *	قبل تحميل نمطٍ دونَ اتّصال   →  يُحجَز الأرشيف
 *	عند نجاح `StyleLoaded`       →  يصير هو الفعّال
 *	عند الانتقال                 →  لا يُحرَّر القديمُ قبل نجاح الجديد
 *	بعد نجاح الجديد              →  يُحرَّر القديم
 *	إن سقط الانتقال              →  القديمُ يبقى محجوزاً وفعّالاً
 *
 * # **والفرقُ بين المطلوب والمحمَّل** (البند ١٢)
 *
 * **`desired` ما طلبته الواجهة، و`loaded` ما نجح تحميلُه.** ولا
 * يُكتب `loaded = ب` **قبل `StyleLoaded(ب)`** — **وإلّا اعتقد الحذفُ
 * أنّ «أ» حُرّرت وهي ما زالت على الشاشة.**
 */
class MapArchiveLease {

    /** **ما تُشير إليه الحجزُ** — حزمةٌ بعينها بنسختها. */
    data class Ref(val regionId: String, val dataVersion: String, val archive: File) {
        override fun toString(): String = "$regionId@$dataVersion"
    }

    /** **حالُ زمنِ التشغيل** — تُقرأ ولا تُخمَّن. */
    data class Snapshot(
        /** **ما طُلب** — قد لا يكون قد حُمّل بعد. */
        val desired: Ref?,
        /** **ما نجح تحميلُه فعلاً** — وهو المفتوحُ الآن. */
        val loaded: Ref?,
        /** **أثمّة انتقالٌ جارٍ؟** */
        val switching: Boolean,
    ) {
        /**
         * **كلُّ ما لا يجوز حذفُه.**
         *
         * **والمطلوبُ محميٌّ كالمحمَّل** — فأثناء الانتقال **يكون
         * الملفّان كلاهما في الطريق**: القديمُ مفتوحٌ والجديدُ على
         * وشك.
         */
        val pinned: Set<String>
            get() = buildSet {
                loaded?.let { add(it.toString()) }
                desired?.let { add(it.toString()) }
            }
    }

    private var desired: Ref? = null
    private var loaded: Ref? = null

    @Synchronized
    fun snapshot(): Snapshot = Snapshot(desired, loaded, desired != null && desired != loaded)

    /** **ما تفتحه الخريطةُ الآن** — أو لا شيءَ إن كانت على الاتّصال. */
    @Synchronized
    fun activeArchive(): File? = loaded?.archive

    @Synchronized
    fun loadedRef(): Ref? = loaded

    /**
     * **يُحجَز قبل تحميل نمطٍ دونَ اتّصال.**
     *
     * **ولا يُحرَّر القديمُ هنا** — البند ٩: «لا تُحرر القديمة قبل
     * نجاح Style الجديدة».
     */
    @Synchronized
    fun beginSwitch(to: Ref?) {
        desired = to
    }

    /**
     * **يُثبَّت عند نجاح `StyleLoaded`.**
     *
     * **وهنا وحدَه يُحرَّر القديم.**
     */
    @Synchronized
    fun styleLoaded(ref: Ref?) {
        loaded = ref
        desired = ref
    }

    /**
     * **وإن سقط التحميل** — يُردُّ المطلوبُ إلى المحمَّل.
     *
     * **فالقديمُ يبقى محجوزاً وفعّالاً** (البند ٩)، **ولا تصير
     * الخريطةُ بلا مصدرٍ لأنّ محاولةً سقطت.**
     */
    @Synchronized
    fun switchFailed() {
        desired = loaded
    }

    /** **عند الذهاب إلى الاتّصال** — لا أرشيفَ محجوز. */
    @Synchronized
    fun beginSwitchToOnline() {
        desired = null
    }

    @Synchronized
    fun reset() {
        desired = null
        loaded = null
    }

    /**
     * **أيجوز حذفُ هذه الحزمة الآن؟**
     *
     * **ثلاثةُ موانع** (البندان ١٠ و١١): أن تكون المحمَّلةَ، أو أن
     * تكون المطلوبةَ في انتقالٍ جارٍ، **أو أن يكون ثمّة انتقالٌ جارٍ
     * أصلاً وهي طرفُه الآخر.**
     */
    @Synchronized
    fun deletionVerdict(regionId: String, dataVersion: String): Verdict {
        val key = "$regionId@$dataVersion"
        val snap = Snapshot(desired, loaded, desired != null && desired != loaded)

        if (loaded?.toString() == key) {
            return if (snap.switching) {
                Verdict.Wait("الخريطةُ تقرأ منها الآن وانتقالٌ جارٍ — يُنتظر نتيجتُه")
            } else {
                Verdict.Refused("الخريطةُ تقرأ منها الآن")
            }
        }
        if (desired?.toString() == key) {
            return Verdict.Wait("انتقالٌ إليها جارٍ — يُنتظر نتيجتُه")
        }
        return Verdict.Allowed
    }

    sealed interface Verdict {
        data object Allowed : Verdict
        data class Refused(val why: String) : Verdict
        data class Wait(val why: String) : Verdict
    }
}
