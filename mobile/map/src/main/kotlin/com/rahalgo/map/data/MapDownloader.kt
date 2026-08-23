package com.rahalgo.map.data

import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.RandomAccessFile
import java.security.MessageDigest
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════
 * **التنزيل — والملفُّ الجزئيُّ ليس دليلَ صحّة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٩ و١٠ و١١.)
 *
 * **أمرُ المالك نصّاً**: «لا تفترض أن `.part` صالح لمجرد وجوده».
 *
 * # **ولماذا الاستئنافُ خطرٌ إن أُحسن الظنّ**
 *
 * `.part` **بايتاتٌ على القرص لا هويّةَ لها.** فلو استُؤنف عليه
 * بلا تحقّقٍ وقعت ثلاثُ كوارثَ صامتة:
 *
 * **الأولى** — بقايا نسخةٍ أقدمَ من الأثر نفسِه. **فيُلصق أوّلُ ملفٍّ
 * بآخرِ ملفٍّ آخر**، والبصمةُ تكشفه بعد تنزيل ثلاثِ مئةِ ميغابايت.
 *
 * **الثانية** — الخادمُ لا يفهم `Range` **فيردّ 200 بالملفّ كلِّه**.
 * ولو كُتب في آخر الجزئيّ **لصار الملفُّ أطولَ من الأصل ونصفُه
 * مكرَّر.** **فالردُّ 200 يعني: ابدأ من الصفر.**
 *
 * **الثالثة** — الأثرُ نفسُه تبدّل في المنبع. **فالهويّةُ تُثبَّت في
 * اسم المؤقّت** (بصمةُ الأثر المنتظَرة)، **فلا يُستأنف على غيره.**
 */
class MapDownloader(
    private val store: MapPackageStore,
    private val http: HttpSource,
    private val tuning: Tuning = Tuning(),
) {

    /**
     * **الثوابتُ في مكانٍ واحد** — أمرُ المالك: «لا تثبت Safety margin
     * اعتباطيًا؛ اجعلها Tuning/central constant».
     */
    data class Tuning(
        /**
         * **هامشُ الأمان** — لا يُبدأ تنزيلٌ يملأ القرص.
         *
         * **ونظامُ أندرويد يحذف بيانات التطبيقات حين يمتلئ التخزين**،
         * **فحزمةٌ تُركَّب على آخر بايتٍ قد تُمحى في الليلة نفسِها.**
         */
        val safetyMarginBytes: Long = 200L * 1024 * 1024,

        /**
         * **وأثناء التركيب يوجد الملفّان معاً** — المؤقّتُ والهدفُ
         * القديم. **فالحاجةُ ضِعفٌ لحظةَ التبديل لا مرّةً واحدة.**
         *
         * **لكنّ النقلةَ إعادةُ تسميةٍ لا نسخ**، فالزيادةُ الحقيقيّةُ
         * حجمُ الجديد وحدَه ما دام القديمُ يُحذف بعدَه.
         */
        val tempOverheadBytes: Long = 8L * 1024 * 1024,

        /** **وعمرُ المؤقّت** — بعده لا يُستأنف ويُنظَّف (البند ٩). */
        val staleTempMs: Long = 7L * 24 * 60 * 60 * 1000,

        val bufferBytes: Int = 64 * 1024,
    )

    /** **ما يُبلَّغ به المتابعُ** — بايتاتٌ لا نِسَبٌ مئويّة. */
    fun interface Progress {
        fun onBytes(downloaded: Long, total: Long)
    }

    /** **ونافذةُ الإلغاء** — تُسأل بين الحُزم لا مرّةً في البداية. */
    fun interface Cancellation {
        fun isCancelled(): Boolean
    }

    sealed interface Outcome {
        data class Installed(val file: File, val bytes: Long) : Outcome
        data class Failed(val reason: Reason, val detail: String) : Outcome
        data object Cancelled : Outcome
    }

    enum class Reason {
        NO_SPACE,
        NETWORK,
        SIZE_MISMATCH,
        CHECKSUM_MISMATCH,
        WRITE_FAILED,
        BAD_URL,
    }

    /**
     * **ينزّل أثراً ويركّبه** — أو لا يترك أثراً لتركيبٍ ناقص.
     *
     * **الترتيبُ عقدٌ** (البند ٩):
     *
     *	تنزيل → مؤقّت → حجم → بصمة → نقلةٌ ذرّيّة → علامةُ تركيب
     *
     * **وأيُّ سقوطٍ يترك القديمَ الصالحَ كما هو.**
     */
    fun fetch(
        url: String,
        expectedBytes: Long,
        expectedSha256: String,
        target: File,
        progress: Progress? = null,
        cancellation: Cancellation? = null,
    ): Outcome {
        if (!MapUrls.isSha256(expectedSha256)) {
            return Outcome.Failed(Reason.CHECKSUM_MISMATCH, "بصمةٌ منتظَرةٌ غيرُ صحيحة")
        }
        store.prepare()
        store.cleanStaleTemp(tuning.staleTempMs, System.currentTimeMillis())

        // ── ١ · المساحة ────────────────────────────────────────────
        val need = expectedBytes + tuning.tempOverheadBytes + tuning.safetyMarginBytes
        val have = store.usableBytes()
        if (have < need) {
            return Outcome.Failed(
                Reason.NO_SPACE,
                "المتاح ${mb(have)} والمطلوب ${mb(need)} " +
                    "(${mb(expectedBytes)} + مؤقّت ${mb(tuning.tempOverheadBytes)} " +
                    "+ هامش ${mb(tuning.safetyMarginBytes)})",
            )
        }

        // ── ٢ · الهويّةُ في اسم المؤقّت ────────────────────────────
        // **فلا يُستأنف على بقايا أثرٍ آخر.**
        val part = store.partFile(expectedSha256)

        val already = if (part.isFile) part.length() else 0L
        if (already > expectedBytes) {
            // **أطولُ من المنتظَر ⇒ ليس هذا** — يُطرح ويُبدأ.
            part.delete()
        }

        return try {
            val downloaded = transfer(url, part, expectedBytes, progress, cancellation)
                ?: return Outcome.Cancelled

            if (downloaded != expectedBytes) {
                part.delete()
                return Outcome.Failed(
                    Reason.SIZE_MISMATCH,
                    "نُزّل $downloaded والمنتظَر $expectedBytes",
                )
            }

            // ── ٣ · البصمة ─────────────────────────────────────────
            // **مرّةً واحدةً عند التركيب** (البند ٤٧) — لا كلَّ فتحة.
            val got = sha256(part)
            if (!got.equals(expectedSha256, ignoreCase = true)) {
                part.delete()
                return Outcome.Failed(
                    Reason.CHECKSUM_MISMATCH,
                    "المقروء $got والمنتظَر $expectedSha256",
                )
            }

            // ── ٤ · النقلةُ الذرّيّة ───────────────────────────────
            store.installFile(part, target)
            Outcome.Installed(target, expectedBytes)
        } catch (e: HttpSource.HttpException) {
            Outcome.Failed(Reason.NETWORK, e.message ?: "تعذّر الاتّصال")
        } catch (e: IOException) {
            /**
             * **وامتلاءُ القرص أثناء الكتابة نقصُ مساحةٍ لا خطأَ
             * كتابة** — قرارُ المالك ٢٠٢٦-٠٨-٢١.
             *
             * **الفحصُ قبل البدء لا يمنع كلَّ حالة**: تطبيقٌ آخرُ
             * يملأ القرصَ بينما ننزّل، **فتسقط الكتابةُ في المنتصف.**
             *
             * **والفرقُ يُعرض للسائق**: «لا مساحة» يعرف ما يفعله بها،
             * **و«تعذّرت الكتابة» لا تقول شيئاً.**
             */
            val full = e.message?.let { m ->
                DISK_FULL.any { m.contains(it, ignoreCase = true) }
            } == true
            part.delete()
            if (full) {
                Outcome.Failed(Reason.NO_SPACE, e.message ?: "امتلأ القرصُ أثناء التنزيل")
            } else {
                Outcome.Failed(Reason.WRITE_FAILED, e.message ?: "تعذّرت الكتابة")
            }
        } catch (e: IllegalArgumentException) {
            Outcome.Failed(Reason.BAD_URL, e.message ?: "عنوانٌ مرفوض")
        }
    }

    /**
     * **النقلُ نفسُه — واستئنافٌ لا يُصدَّق حتّى يُثبت.**
     *
     * **يردّ `null` إن أُلغي.**
     */
    private fun transfer(
        url: String,
        part: File,
        expectedBytes: Long,
        progress: Progress?,
        cancellation: Cancellation?,
    ): Long? {
        val from = if (part.isFile) part.length() else 0L
        val response = http.open(url, if (from > 0) from else null)

        response.use { res ->
            /**
             * **وهنا يُحسم الاستئناف.**
             *
             * **206 مع `Content-Range` مطابقٍ** ⇒ نكمل من حيث وقفنا.
             * **200** ⇒ الخادمُ تجاهل الطلبَ وأرسل الكلَّ، **فنبدأ من
             * الصفر** — أمرُ المالك نصّاً.
             */
            val append = when {
                from <= 0L -> false
                res.status == 206 && res.rangeStart == from -> true
                res.status == 206 -> {
                    // **206 بمدىً غيرِ الذي طُلب** — لا يُلصق.
                    false
                }
                else -> false
            }

            if (!append && part.exists() && !part.delete()) {
                throw IOException("تعذّر إخلاءُ المؤقّت: $part")
            }

            val startAt = if (append) from else 0L
            val total = if (append) startAt + res.remainingBytes else res.remainingBytes
            if (res.remainingBytes > 0 && total != expectedBytes) {
                // **والحجمُ المُعلَنُ من الخادم يُقارَن بالفهرس** — فلو
                // اختلف فالأثرُ ليس الذي وُعدنا به.
                throw IOException("الخادمُ يُعلن $total والفهرس $expectedBytes")
            }

            RandomAccessFile(part, "rw").use { out ->
                out.seek(startAt)
                if (!append) out.setLength(0)

                val buf = ByteArray(tuning.bufferBytes)
                var written = startAt
                val input: InputStream = res.body
                while (true) {
                    if (cancellation?.isCancelled() == true) return null
                    val n = input.read(buf)
                    if (n < 0) break
                    out.write(buf, 0, n)
                    written += n
                    progress?.onBytes(written, expectedBytes)
                }
                return written
            }
        }
    }

    private fun mb(b: Long): String =
        String.format(Locale.US, "%.1f م.ب", b / (1024.0 * 1024.0))

    companion object {
        /**
         * **علاماتُ امتلاء القرص في رسائل النظام.**
         *
         * **ولا استثناءَ مخصَّصٌ لها في جافا** — `IOException` عامّةٌ،
         * **والنصُّ هو ما يفرّق.** فيُقرأ بالإنكليزيّة لأنّ الرسالةَ
         * من النواة لا من تطبيقنا.
         */
        private val DISK_FULL = listOf(
            "ENOSPC",
            "No space left",
            "Not enough space",
            "disk full",
        )

        fun sha256(file: File): String {
            val digest = MessageDigest.getInstance("SHA-256")
            file.inputStream().use { input ->
                val buf = ByteArray(64 * 1024)
                while (true) {
                    val n = input.read(buf)
                    if (n < 0) break
                    digest.update(buf, 0, n)
                }
            }
            return digest.digest().joinToString("") { "%02x".format(it) }
        }
    }
}

/**
 * **منفذُ الشبكة — مُجرَّدٌ ليُختبر بلا خادم.**
 *
 * **البند ٤٩ يطلب اختبارَ سلوك الاستئناف و«200 بعد Range»** — ولا
 * يُختبر ذلك بشبكةٍ حقيقيّة. **فالواجهةُ هنا هي موضعُ الحقن.**
 */
interface HttpSource {

    class HttpException(message: String) : IOException(message)

    /**
     * @param rangeStart **إن لم يكن `null` طُلب `Range: bytes=N-`.**
     */
    fun open(url: String, rangeStart: Long?): Response

    interface Response : AutoCloseable {
        val status: Int

        /** **بدايةُ المدى الذي ردَّه الخادمُ فعلاً** — من `Content-Range`. */
        val rangeStart: Long

        /** **وما تبقّى في هذا الردّ** — من `Content-Length`. */
        val remainingBytes: Long

        val body: InputStream
    }
}
