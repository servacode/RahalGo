package com.rahalgo.map.data

import java.io.File
import java.net.URI
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════
 * **العناوين — ولا يُجمَع نصٌّ ليصير عنواناً**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٤ و٤٤.)
 *
 * **أمرُ المالك نصّاً**: «لا تبنِ URI بتجميع String هش».
 *
 * # **وفصلُ العالمين هو أصلُ الأمان**
 *
 * **عنوانٌ بعيدٌ من الفهرس** — يُقيَّد بـ`https` وبمضيفٍ من الإعداد،
 * **ولا يصير أبداً ملفّاً محلّيّاً.**
 *
 * **وعنوانٌ محلّيٌّ يبنيه التطبيق** — من ملفٍّ ركّبه بنفسه في مجلّده،
 * **ولا يأتي نصُّه من الشبكة.**
 *
 * **فلو خلطا لاستطاع فهرسٌ بعيدٌ أن يقول `file:///data/data/…`
 * فتفتح الخريطةُ ملفَّ التطبيق وترسله إلى عارضٍ لا يعرف.**
 *
 * # **والمسافةُ في المسار تكسر العنوان**
 *
 * `RahalGo Regular` **اسمُ مجلّدِ رصّةٍ فيه مسافة.** ولو جُمع نصّاً في
 * عنوانٍ **لصار العنوانُ مقطوعاً عند المسافة.** فالبناءُ بـ`File.toURI`
 * الذي يُرمّز، **لا بجمع نصوص.**
 */
object MapUrls {

    /** **المخطّطاتُ المسموحةُ للعناوين البعيدة** — ولا رابعَ. */
    val REMOTE_SCHEMES = setOf("https")

    /** **أهو `http` إلى هذا الجهاز نفسِه، وقد أُذن؟** */
    private fun isLoopback(uri: URI, scheme: String?, allowed: Boolean): Boolean {
        if (!allowed || scheme != "http") return false
        val host = uri.host?.lowercase(Locale.ROOT) ?: return false
        return host == "127.0.0.1" || host == "localhost" || host == "::1"
    }

    private val SHA256 = Regex("^[0-9a-f]{64}$")

    fun isSha256(s: String?): Boolean = s != null && SHA256.matches(s.lowercase(Locale.ROOT))

    /**
     * **مسارٌ نسبيٌّ داخلَ الفهرس** — لا مطلقٌ ولا صاعد.
     *
     * **الفهرسُ يقول `regions/2026-08-21/region-raqqa.pmtiles`** —
     * ويُضمّ إلى أصلٍ من الإعداد. **فلو قال `https://evil/x` أو
     * `../../` لخرج عن الأصل.**
     */
    fun isSafeRelative(url: String?): Boolean {
        val u = url ?: return false
        if (u.isBlank()) return false
        if (u.startsWith("/") || u.startsWith("\\")) return false
        if (u.contains("..")) return false
        if (u.contains("://")) return false
        // **والترميزُ لا يخفي الصعود** — `%2e%2e` نقطتان بعد الفكّ.
        val decoded = u.replace("%2e", ".", ignoreCase = true)
            .replace("%2f", "/", ignoreCase = true)
            .replace("%5c", "/", ignoreCase = true)
        if (decoded.contains("..")) return false
        /**
         * **والمسافةُ مقبولةٌ في مسارٍ ومرفوضةٌ في عنوان.**
         *
         * `glyphs/RahalGo Regular/0-255.pbf` **مسارٌ صحيحٌ في عقد
         * الموارد** — واسمُ الرصّة فيه مسافةٌ بقرار ٦أ. **والعنوانُ
         * يُرمّزها إلى `%20` عند البناء** لا يرفضها هنا.
         *
         * **لكنّ محارفَ التحكّم تُرفض** — لا تُرمَّز ولا تُقرأ،
         * **وسطرٌ جديدٌ في مسارٍ محاولةُ حقنٍ في ترويسة.**
         */
        if (u.any { it.code < 0x20 || it.code == 0x7f }) return false
        return true
    }

    /**
     * **يُرمّز مساراً نسبيّاً ليصير جزءاً من عنوان.**
     *
     * **`URI(null, null, path, null)` هي التي ترمّز** — مقطعاً
     * مقطعاً وبقواعد المخطّط. **ولو رُمّز النصُّ كلُّه بـ`URLEncoder`
     * لصارت الشرطةُ المائلةُ `%2F` فانهار المسار**، **ولصارت المسافةُ
     * `+` وهي ترميزُ نماذجَ لا مسارات.**
     */
    fun encodePath(relative: String): String =
        URI(null, null, relative, null).rawPath ?: relative

    /**
     * **يضمُّ مساراً نسبيّاً إلى أصلٍ بعيد.**
     *
     * **ويُتحقَّق من المخطّط بعد الضمّ لا قبله** — فالنتيجةُ هي ما
     * سيُفتح، **لا النيّة.**
     */
    /**
     * ══════════════════════════════════════════════════════════════════
     * **وهذا حارسٌ ثانٍ مستقلٌّ عن `MapConfig`**
     * ══════════════════════════════════════════════════════════════════
     *
     * **قِيس ٢٠٢٦-٠٨-٢٢**: فُتح ثقبُ الحلقيِّ في `MapConfig` فمرّ
     * الإعدادُ وسقط التنزيلُ **صامتاً** — لأنّ كلَّ عنوانٍ يُبنى يمرّ
     * من هنا، **وهنا `https` وحدَها.**
     *
     * **ولم يُسجَّل شيء**: `ensureResources` يعيد `Failed` بلا
     * `Log.w` (بخلاف `ensureRegion`)، **والواجهةُ عادت إلى «نزل
     * الآن» كأنّ شيئاً لم يقع.**
     *
     * **والدرسُ أنّ حارساً واحداً لا يكفي البحثُ عنه** — فُتح الأوّلُ
     * وبقي الثاني، **والأثرُ كان صمتاً لا خطأً.**
     *
     * **والمعطى صريحٌ افتراضُه مغلق** — فمن لم يمرّره لم يتغيّر عنده
     * شيء، **ومن مرّره لا يزال محكوماً بأنّ المضيفَ حلقيٌّ فعلاً.**
     */
    fun resolveRemote(
        baseUrl: String,
        relative: String,
        allowLoopbackHttp: Boolean = false,
    ): String {
        require(isSafeRelative(relative)) { "مسارٌ نسبيٌّ غيرُ مقبول: $relative" }
        val base = if (baseUrl.endsWith("/")) baseUrl else "$baseUrl/"
        // **والضمُّ على المُرمَّز** — فمسافةٌ خامٌّ تُسقط `URI` بخطأ
        // تركيبٍ يبدو خرقَ أمانٍ وهو اسمُ مجلّدٍ سليم.
        val uri = URI(base).resolve(encodePath(relative))
        val scheme = uri.scheme?.lowercase(Locale.ROOT)
        require(scheme in REMOTE_SCHEMES || isLoopback(uri, scheme, allowLoopbackHttp)) {
            "مخطّطٌ غيرُ مسموحٍ لعنوانٍ بعيد: $scheme — المسموح ${REMOTE_SCHEMES.joinToString()}"
        }
        require(!uri.host.isNullOrBlank()) { "عنوانٌ بعيدٌ بلا مضيف: $uri" }
        // **ولا يخرج الضمُّ عن الأصل** — `resolve` تقبل الصعودَ نظريّاً.
        val baseUri = URI(base)
        require(uri.host == baseUri.host && uri.scheme == baseUri.scheme) {
            "الضمُّ خرج عن الأصل: $uri"
        }
        return uri.toString()
    }

    // ══════════════════════════════════════════════════════════════
    // **عناوينُ MapLibre**
    // ══════════════════════════════════════════════════════════════
    //
    // **PMTiles البعيدُ**:   `pmtiles://https://host/path.pmtiles`
    // **PMTiles المحلّيّ**:  `pmtiles://file:///absolute/path.pmtiles`
    //
    // (ثبت الدعمُ بمسح `libmaplibre.so` في PREFLIGHT، وصحّحه المالك.)

    const val PMTILES = "pmtiles://"
    const val MBTILES = "mbtiles://"

    /** **بلاطاتٌ من الشبكة.** */
    fun onlinePmtiles(absoluteUrl: String, allowLoopbackHttp: Boolean = false): String {
        val uri = URI(absoluteUrl)
        val scheme = uri.scheme?.lowercase(Locale.ROOT)
        require(scheme in REMOTE_SCHEMES || isLoopback(uri, scheme, allowLoopbackHttp)) {
            "بلاطاتٌ بعيدةٌ بمخطّطٍ غيرِ مسموح: ${uri.scheme}"
        }
        require(!uri.host.isNullOrBlank()) { "بلاطاتٌ بعيدةٌ بلا مضيف" }
        return PMTILES + uri.toString()
    }

    /**
     * **وبلاطاتٌ من ملفٍّ ركّبه التطبيق.**
     *
     * **`File.toURI()` هي التي تُرمّز** — المسافةَ تصير `%20`،
     * **والعربيّةَ تصير بايتاتِ UTF-8 مرمَّزة.** ولو جُمع النصُّ
     * **لانقطع العنوانُ عند أوّل مسافة.**
     *
     * **والمسارُ يجب أن يكون مطلقاً** — MapLibre لا تعرف مجلَّدَ
     * تشغيلِ التطبيق.
     */
    fun offlinePmtiles(file: File): String {
        require(file.isAbsolute) { "أرشيفٌ محلّيٌّ بمسارٍ غيرِ مطلق: $file" }
        return PMTILES + file.toURI().toASCIIString()
    }

    /**
     * **وMBTiles تُترك بابا مفتوحاً لا طريقاً مسلوكاً** (البند ٥).
     *
     * **قدرةٌ لا مسارَ إنتاج** — `MBTILES DEVICE RENDER — DEFERRED`
     * باقٍ كما هو. **ووجودُها هنا يعني أنّ إضافتَها لاحقاً لا تُعيد
     * كتابة المحلِّل.**
     */
    fun offlineMbtiles(file: File): String {
        require(file.isAbsolute) { "أرشيفٌ محلّيٌّ بمسارٍ غيرِ مطلق: $file" }
        return MBTILES + file.absolutePath
    }

    /** **وعنوانُ مجلّدٍ محلّيٍّ للموارد** — الحروفُ والأيقونات. */
    fun localDirUri(dir: File): String {
        require(dir.isAbsolute) { "مجلّدٌ بمسارٍ غيرِ مطلق: $dir" }
        return dir.toURI().toASCIIString().trimEnd('/')
    }
}
