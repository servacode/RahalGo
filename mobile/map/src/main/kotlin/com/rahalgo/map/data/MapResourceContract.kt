package com.rahalgo.map.data

import org.json.JSONObject

/**
 * ══════════════════════════════════════════════════════════════════
 * **عقدُ الموارد — كما يكتبه ٦أ ويقرؤه الهاتف**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ ٦ب الوظيفيّ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢.)
 *
 * **أمرُ المالك نصّاً**: «استخدم عقد 6A الحقيقي. إذا resources-
 * manifest يحتوي hashes/sizes لكل ملف: استخدمها».
 *
 * # **وكان لا يحتويها**
 *
 * **العقدُ كان يصف مجموعاتٍ لا ملفّات**: عددَ النطاقات ومجموعَ حجمها،
 * وبصمةَ الأيقونات والنمط **دونَ ملفّ حرفٍ واحد.** فالهاتفُ يعلم أنّ
 * ثمّة اثنَي عشرَ ملفَّ حرفٍ **ولا يعلم عنوانَ واحدٍ منها ولا بصمتَه**
 * — **فلا سبيلَ إلى جلبٍ يتحقّق.**
 *
 * **فأُضيف `files[]` إلى `resources-manifest.mjs`** — مسارٌ نسبيٌّ
 * وحجمٌ وبصمةٌ لكلّ ملفّ. **ثمانيةَ عشرَ ملفّاً · ١٫٣ م.ب** لرصّتين.
 *
 * # **والمسارُ نصٌّ من الشبكة**
 *
 * **فلا يصير ملفّاً إلّا بعد فحص.** `../` و`%2e%2e` والمطلقُ كلُّها
 * مرفوضة — **ولا يستطيع عقدٌ بعيدٌ أن يكتب خارجَ مجلّد نسخته.**
 */
data class MapResourceContract(
    val version: String,
    val styleVersion: String,
    val glyphVersion: String,
    val spriteVersion: String,
    val glyphUrlTemplate: String,
    val fontstacks: List<String>,
    val files: List<ResourceFile>,
    val totalBytes: Long,
    val licenseSpdx: String?,
) {
    /** **ملفٌّ واحدٌ يُنزَّل.** */
    data class ResourceFile(val path: String, val bytes: Long, val sha256: String)

    /** **مجموعُ ما سيُنزَّل** — تُسأل قبل بدء أيّ تنزيل (البند ١٠ من ٦ب). */
    val downloadBytes: Long get() = files.sumOf { it.bytes }

    companion object {
        const val EXPECTED_GLYPH_TEMPLATE = "{fontstack}/{range}.pbf"

        fun parse(text: String, version: String): MapResourceContract =
            fromJson(JSONObject(text), version)

        fun fromJson(o: JSONObject, version: String): MapResourceContract {
            val declared = o.optString("resourcesVersion", "")
            require(declared.isBlank() || declared == version) {
                "عقدُ الموارد يقول نسخةَ $declared والفهرسُ يطلب $version"
            }

            val template = o.optString("glyphUrlTemplate", EXPECTED_GLYPH_TEMPLATE)
            require(template == EXPECTED_GLYPH_TEMPLATE) {
                "عقدُ عنوان الحروف غيرُ متوقَّع: $template"
            }

            val stacksArr = o.optJSONArray("fontstacks")
                ?: throw IllegalArgumentException("عقدُ الموارد: لا رصّاتِ خطوط")
            val stacks = (0 until stacksArr.length()).map { i ->
                when (val v = stacksArr.get(i)) {
                    is String -> v
                    is JSONObject -> v.optString("name", "")
                    else -> ""
                }
            }.filter { it.isNotBlank() }
            require(stacks.isNotEmpty()) { "عقدُ الموارد: لا رصّةَ خطٍّ معلنة" }

            val filesArr = o.optJSONArray("files")
                ?: throw IllegalArgumentException(
                    "عقدُ الموارد: لا `files` — عقدٌ من قبل إغلاق ٦ب الوظيفيّ",
                )
            val files = (0 until filesArr.length()).map { i ->
                val f = filesArr.getJSONObject(i)
                val path = f.optString("path", "")
                require(MapUrls.isSafeRelative(path)) {
                    "عقدُ الموارد: مسارٌ غيرُ مقبول: $path"
                }
                // **ولا يُقبل مسارٌ خارجَ ما نتوقّعه** — القائمةُ بيضاءُ
                // لا تنظيف. **فلو أُضيف نوعٌ جديدٌ يوماً يُضاف هنا
                // بقصد**، ولا يمرّ لأنّه بدا بريئاً.
                require(isKnownResourcePath(path)) {
                    "عقدُ الموارد: نوعُ ملفٍّ غيرُ معروف: $path"
                }
                val bytes = f.optLong("bytes", -1L)
                require(bytes > 0) { "عقدُ الموارد: حجمٌ غيرُ صحيحٍ لـ$path" }
                val sha = f.optString("sha256", "")
                require(MapUrls.isSha256(sha)) { "عقدُ الموارد: بصمةٌ غيرُ صحيحةٍ لـ$path" }
                ResourceFile(path, bytes, sha)
            }
            require(files.isNotEmpty()) { "عقدُ الموارد: `files` فارغة" }

            val license = o.optJSONObject("fontLicense")?.optString("spdx")?.ifBlank { null }

            return MapResourceContract(
                version = version,
                styleVersion = o.optString("styleVersion", ""),
                glyphVersion = o.optString("glyphVersion", ""),
                spriteVersion = o.optString("spriteVersion", ""),
                glyphUrlTemplate = template,
                fontstacks = stacks,
                files = files,
                totalBytes = o.optLong("totalBytes", files.sumOf { it.bytes }),
                licenseSpdx = license,
            )
        }

        /**
         * **الأشكالُ المعروفة** — وما عداها يُرفض.
         *
         *	glyphs/<رصّة>/<نطاق>.pbf
         *	sprite.json · sprite.png · sprite@2x.json · sprite@2x.png
         *	style.offline.json
         *	fonts/LICENSE.txt
         */
        fun isKnownResourcePath(path: String): Boolean {
            if (path in MapSpriteFiles.ALL) return true
            if (path == MapPaths.STYLE_FILE) return true
            if (path == LICENSE_PATH) return true
            val m = GLYPH_PATH.matchEntire(path) ?: return false
            val stack = m.groupValues[1]
            return !stack.contains('/') && !stack.contains("..") && stack.isNotBlank()
        }

        const val LICENSE_PATH = "fonts/LICENSE.txt"
        private val GLYPH_PATH = Regex("^glyphs/([^/]+)/\\d+-\\d+\\.pbf$")
    }
}
