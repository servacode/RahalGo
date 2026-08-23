package com.rahalgo.map.data

import java.io.File
import java.util.Locale

/**
 * ══════════════════════════════════════════════════════════════════
 * **المسارات — والفهرسُ البعيدُ لا يكتب على قرصنا حيث يشاء**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٤٤ و٤٥.)
 *
 * # **لماذا لا يُشتقّ مسارٌ من نصٍّ بعيد**
 *
 * **الفهرسُ ملفٌّ يُنزَّل من الشبكة.** ومعرِّفُ المنطقةِ فيه نصٌّ، ولو
 * صار مسارَ ملفٍّ بالجمع **لكتب من ملك الخادمَ حيث شاء من قرص
 * الهاتف**:
 *
 *	id = "../../../databases"      ←  خارجَ مجلّد التطبيق
 *	id = "/data/data/other/files"  ←  مسارٌ مطلق
 *	id = "raqqa%2f..%2fx"          ←  بعد فكّ الترميز
 *
 * **فلا يُنظَّف المعرِّفُ ولا يُهرَّب** — بل **يُرفض ما ليس على الشكل
 * المسموح.** والتنظيفُ يترك ثقوباً؛ **والقائمةُ البيضاءُ لا تتركها.**
 *
 * **والشكلُ المسموح**: حروفٌ لاتينيّةٌ صغيرةٌ وأرقامٌ وشرطة، **ولا نقطةَ
 * ولا شرطةَ مائلة.** فلا `..` ولا مسارٌ مطلقٌ ولا امتداد.
 */
object MapPaths {

    /** **حدُّ الطول** — فلا اسمُ ملفٍّ يتجاوز ما يقبله نظامُ الملفّات. */
    const val MAX_ID_LENGTH = 64

    private val SAFE = Regex("^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$")

    /**
     * **هل المعرِّفُ صالحٌ ليصير جزءاً من مسار؟**
     *
     * **يُفحص النصُّ الخام** — لا بعد فكِّ ترميزٍ ولا بعد تطبيع.
     * **ولو فُكّ الترميزُ أوّلاً لصار `%2e%2e` نقطتين ثمّ مرّ.**
     */
    fun isSafeId(raw: String?): Boolean {
        val id = raw ?: return false
        if (id.length > MAX_ID_LENGTH) return false
        if (!SAFE.matches(id)) return false
        // **والحذرُ مضاعف** — لو تبدّل التعبيرُ يوماً بقيت هذه.
        if (id.contains("..") || id.contains('/') || id.contains('\\')) return false
        return true
    }

    /**
     * **ونسخةُ البيانات معرِّفٌ أيضاً.**
     *
     * **تأتي من الفهرس** (`dataVersion` من ترويسة الأرشيف) **فتُفحص
     * كما يُفحص المعرِّف.**
     */
    fun isSafeVersion(raw: String?): Boolean = isSafeId(raw)

    /** **ويُرفض بصوتٍ لا يُنظَّف بصمت.** */
    fun requireSafeId(raw: String?, what: String): String {
        require(isSafeId(raw)) { "معرِّفٌ غيرُ مقبول لـ$what: ${describe(raw)}" }
        return raw!!
    }

    private fun describe(raw: String?): String = when {
        raw == null -> "غائب"
        raw.isEmpty() -> "فارغ"
        raw.length > MAX_ID_LENGTH -> "أطولُ من $MAX_ID_LENGTH محرفاً"
        else -> raw.take(MAX_ID_LENGTH).replace(Regex("[\\r\\n\\t]"), " ")
    }

    // ══════════════════════════════════════════════════════════════
    // **بنيةُ المجلّدات** — البند ٦
    // ══════════════════════════════════════════════════════════════
    //
    //	maps/
    //	  state/active.json
    //	  tmp/
    //	  resources/{resourcesVersion}/{style,glyphs,sprite}
    //	  regions/{regionId}/{dataVersion}/{region.pmtiles,package.json}
    //
    // **والنسخةُ في المسار لا في اسم الملفّ** — فتُركَّب الجديدةُ
    // بجانب القديمة، **ولا تُحذف صالحةٌ قبل نجاح بديلها** (البند ٣٤).

    fun root(base: File): File = File(base, "maps")

    fun stateDir(base: File): File = File(root(base), "state")

    fun activeFile(base: File): File = File(stateDir(base), "active.json")

    fun manifestCacheFile(base: File): File = File(stateDir(base), "manifest.json")

    /** **والمؤقّتُ في نظام الملفّات نفسِه** — وإلّا لم يكن التبديلُ ذرّيّاً. */
    fun tmpDir(base: File): File = File(root(base), "tmp")

    fun resourcesRoot(base: File): File = File(root(base), "resources")

    fun resourcesDir(base: File, resourcesVersion: String): File =
        File(resourcesRoot(base), requireSafeId(resourcesVersion, "نسخةِ الموارد"))

    /**
     * **مجلّدُ التركيب المؤقّت** — إغلاقُ ٦ب الوظيفيّ، البند ٣.
     *
     * **الموارُد ثمانيةَ عشرَ ملفّاً لا ملفّاً واحداً.** فلو كُتبت في
     * موضعها النهائيّ **لظهرت النسخةُ مركَّبةً وهي نصفُها**، **ولا شيءَ
     * يفرّق بين نصفٍ وكلٍّ حتّى تُفتح الخريطةُ فتظهر مربّعات.**
     *
     * **والبادئةُ نقطةٌ** — فلا يخلطها ماسحُ النسخ بنسخةٍ مركَّبة:
     * `isSafeId` ترفض ما يبدأ بنقطة.
     */
    fun resourcesStagingDir(base: File, resourcesVersion: String): File =
        File(resourcesRoot(base), ".tmp-${requireSafeId(resourcesVersion, "نسخةِ الموارد")}")

    fun regionDir(base: File, regionId: String): File =
        File(File(root(base), "regions"), requireSafeId(regionId, "المنطقة"))

    fun regionVersionDir(base: File, regionId: String, dataVersion: String): File =
        File(regionDir(base, regionId), requireSafeId(dataVersion, "نسخةِ البيانات"))

    fun regionArchive(base: File, regionId: String, dataVersion: String): File =
        File(regionVersionDir(base, regionId, dataVersion), REGION_ARCHIVE)

    fun regionPackageFile(base: File, regionId: String, dataVersion: String): File =
        File(regionVersionDir(base, regionId, dataVersion), PACKAGE_FILE)

    fun resourcesPackageFile(base: File, resourcesVersion: String): File =
        File(resourcesDir(base, resourcesVersion), PACKAGE_FILE)

    /**
     * **ومسارُ رصّة الحروف** — عقدُ ٦أ: `{fontstack}/{range}.pbf`.
     *
     * **واسمُ الرصّة فيه مسافة** (`RahalGo Regular`) — وهو اسمُ مجلّدٍ
     * على القرص لا معرِّفٌ من فهرس، **فلا يمرّ على `isSafeId`.**
     * **لكنّه لا يُترك بلا حارس**: يأتي من النمط القانونيّ المُثبَّت في
     * التطبيق، **ويُرفض إن حوى فاصلَ مسار.**
     */
    fun glyphFile(
        base: File,
        resourcesVersion: String,
        fontstack: String,
        range: String,
    ): File {
        require(!fontstack.contains('/') && !fontstack.contains('\\') &&
            !fontstack.contains("..") && fontstack.isNotBlank()) {
            "رصّةٌ غيرُ مقبولة: $fontstack"
        }
        require(Regex("^\\d+-\\d+$").matches(range)) { "نطاقٌ غيرُ مقبول: $range" }
        return File(
            File(resourcesDir(base, resourcesVersion), "glyphs"),
            "$fontstack${File.separator}$range.pbf",
        )
    }

    fun spriteFile(base: File, resourcesVersion: String, name: String): File =
        File(resourcesDir(base, resourcesVersion), name)

    fun styleFile(base: File, resourcesVersion: String): File =
        File(resourcesDir(base, resourcesVersion), STYLE_FILE)

    /**
     * **مسارُ مورِدٍ داخلَ مجلّد نسخة** — ولا يخرج عنه.
     *
     * **يُبنى من مقاطعَ مفحوصةٍ واحداً واحدا**، **ثمّ يُتحقَّق أنّ
     * الناتجَ المُقنَّن ما زال تحتَ الجذر** — فلو نجا مقطعٌ من الفحص
     * لسببٍ لم نتوقّعه **أمسكه الفحصُ الثاني.**
     */
    fun resourceFileIn(versionDir: File, relative: String): File {
        require(relative.isNotBlank()) { "مسارُ مورِدٍ فارغ" }
        require(!relative.startsWith("/") && !relative.startsWith("\\")) {
            "مسارُ مورِدٍ مطلق: $relative"
        }
        val parts = relative.split('/')
        var f = versionDir
        for (part in parts) {
            require(part.isNotBlank() && part != "." && part != "..") {
                "مقطعُ مسارٍ غيرُ مقبول في: $relative"
            }
            require(!part.contains('\\')) { "فاصلٌ خلفيٌّ في: $relative" }
            f = File(f, part)
        }
        val root = versionDir.canonicalPath
        val leaf = f.canonicalPath
        require(leaf == root || leaf.startsWith(root + File.separator)) {
            "مسارُ مورِدٍ يخرج عن مجلّد نسخته: $relative"
        }
        return f
    }

    const val REGION_ARCHIVE = "region.pmtiles"
    const val PACKAGE_FILE = "package.json"
    const val STYLE_FILE = "style.offline.json"

    /**
     * **ولا يُبنى اسمُ ملفٍّ مؤقّتٍ بالتاريخ.**
     *
     * **الاسمُ مشتقٌّ من هويّة الأثر** — فيُعرف عند الاستئناف **أنّ
     * `.part` الموجودَ لهذا الأثر بعينه لا لغيره** (البند ١١).
     */
    fun partFile(base: File, identity: String): File =
        File(tmpDir(base), "${identity.lowercase(Locale.ROOT).replace(Regex("[^a-z0-9-]"), "_")}.part")
}
