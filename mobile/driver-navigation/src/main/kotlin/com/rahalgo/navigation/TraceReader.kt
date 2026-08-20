package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قارئُ الرحلة — يُعيدها إلى السلسلة بلا هاتف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك ٢٠٢٦-٠٨-٢٠: «نستطيع اختبارَ مئات التغييرات في الملاحة
 *  على رحلاتٍ حقيقيّةٍ مسجَّلةٍ دون إعادة القيادة كلَّ مرّة».)
 *
 * **وهذا نصفُ المختبَر الثاني**: المسجّلُ يكتب، وهذا يقرأ، **وبينهما
 * تُعاد الرحلةُ في اختبارِ وحدةٍ يعمل في ثوانٍ.**
 *
 * # ويقرأ النسخةَ قبل المحتوى
 *
 * **ورفيدةٌ من صيغةٍ لا نعرفها تُقرأ أرقاماً خاطئةً بصمت** — **وصمتٌ
 * كهذا يجعل اختباراً أخضرَ يحرس شيئاً لم يقع.**
 *
 * # وقارئٌ باليد لا بمكتبة
 *
 * **الحقولُ مسطّحةٌ وقيمُها أرقامٌ ونصوصٌ قصيرة** — **ومكتبةٌ تُضاف
 * لسطرٍ كهذا تحمل الوحدةَ تبعيّةً لا تحتاجها.**
 */
object TraceReader {

    /** **سطرٌ مقروءٌ من رفيدة** — بقراءته وحكمِه كما سُجّل. */
    data class Row(
        val fix: NavFix,
        val grade: FixGrade,
        val reason: RejectReason,
        val schema: Int,
        val sessionId: String,
    )

    /** **ما تعذّر قراءتُه** — يُقال ولا يُبتلع. */
    class BadTrace(message: String) : IllegalArgumentException(message)

    /**
     * **يقرأ رفيدةً كاملة.**
     *
     * **والسطرُ الفارغُ يُتخطّى** — ملفٌّ ينتهي بسطرٍ فارغٍ أمرٌ عاديّ.
     *
     * **والنسخةُ غيرُ المدعومة تُوقف** ولا تُقرأ جزئيّاً.
     */
    fun parse(lines: Iterable<String>): List<Row> {
        val out = ArrayList<Row>()
        for (raw in lines) {
            val line = raw.trim()
            if (line.isEmpty()) continue
            out.add(parseLine(line))
        }
        return out
    }

    /** **ويردّ القراءاتِ وحدَها** — لمن يريد أن يغذّي السلسلة. */
    fun fixesOf(rows: List<Row>): List<NavFix> = rows.map { it.fix }

    fun parseLine(line: String): Row {
        val schema = num(line, "schema_version")?.toInt()
            ?: throw BadTrace("سطرٌ بلا نسخةِ صيغة")
        if (schema < TraceRecorder.MIN_SCHEMA || schema > TraceRecorder.SCHEMA) {
            throw BadTrace(
                "نسخةُ صيغةٍ غيرُ مدعومة: $schema " +
                    "(المدعوم ${TraceRecorder.MIN_SCHEMA}..${TraceRecorder.SCHEMA})",
            )
        }
        val grade = str(line, "quality_result")?.let { runCatching { FixGrade.valueOf(it) }.getOrNull() }
            ?: throw BadTrace("حكمٌ غيرُ مفهوم")
        val reason = str(line, "rejection_reason")
            ?.let { runCatching { RejectReason.valueOf(it) }.getOrNull() }
            ?: RejectReason.NONE

        val fix = NavFix(
            lat = num(line, "latitude") ?: throw BadTrace("لا خطَّ عرض"),
            lng = num(line, "longitude") ?: throw BadTrace("لا خطَّ طول"),
            accuracyM = (num(line, "accuracy_m") ?: throw BadTrace("لا دقّة")).toFloat(),
            speedMps = num(line, "speed_mps")?.toFloat(),
            // **والغائبُ يبقى غائباً** — `null` لا صفر.
            bearingDeg = num(line, "bearing_deg")?.toFloat(),
            atMs = num(line, "at_ms")?.toLong() ?: 0L,
            elapsedNs = num(line, "elapsed_realtime_nanos")?.toLong() ?: 0L,
            wallMs = num(line, "timestamp")?.toLong() ?: 0L,
            provider = str(line, "provider"),
        )
        return Row(fix, grade, reason, schema, str(line, "session_id") ?: "")
    }

    /**
     * **رقمٌ باسمه** — أو فارغٌ إن كان `null` أو غائباً.
     *
     * **و`null` تُميَّز عن الغياب في المعنى لا في القراءة**: كلاهما
     * «لا قيمة».
     */
    fun num(line: String, key: String): Double? {
        val v = rawValue(line, key) ?: return null
        if (v == "null") return null
        return v.toDoubleOrNull()
    }

    /** **نصٌّ باسمه** — بلا علامتَي الاقتباس. */
    fun str(line: String, key: String): String? {
        val v = rawValue(line, key) ?: return null
        if (v == "null") return null
        if (v.length >= 2 && v.startsWith("\"") && v.endsWith("\"")) {
            return unquote(v.substring(1, v.length - 1))
        }
        return v
    }

    /**
     * **القيمةُ الخامّةُ كما كُتبت.**
     *
     * **ويُبحث عن `"key":` لا عن `key`** — **واسمٌ يقع داخل قيمةٍ
     * نصّيّةٍ يخدع البحثَ الساذج.**
     */
    private fun rawValue(line: String, key: String): String? {
        val needle = "\"$key\":"
        val at = line.indexOf(needle)
        if (at < 0) return null
        var i = at + needle.length
        while (i < line.length && line[i] == ' ') i++
        if (i >= line.length) return null
        if (line[i] == '"') {
            val sb = StringBuilder()
            sb.append('"')
            i++
            while (i < line.length) {
                val c = line[i]
                if (c == '\\' && i + 1 < line.length) {
                    sb.append(c).append(line[i + 1]); i += 2; continue
                }
                sb.append(c)
                i++
                if (c == '"') break
            }
            return sb.toString()
        }
        val end = generateSequence(i) { it + 1 }
            .first { it >= line.length || line[it] == ',' || line[it] == '}' }
        return line.substring(i, end).trim()
    }

    private fun unquote(s: String): String =
        s.replace("\\\"", "\"").replace("\\n", "\n").replace("\\r", "\r")
            .replace("\\t", "\t").replace("\\\\", "\\")
}
