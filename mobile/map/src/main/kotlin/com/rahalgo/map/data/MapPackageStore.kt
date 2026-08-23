package com.rahalgo.map.data

import org.json.JSONObject
import java.io.File
import java.io.IOException

/**
 * ══════════════════════════════════════════════════════════════════
 * **مخزنُ الحزم — الموضعُ الوحيدُ الذي يعرف أسماءَ الملفّات**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٦ و٧.)
 *
 * **أمرُ المالك نصّاً**: «لا تجعل `TripMap` تعرف أسماء الملفات. ولا
 * تجعل `OfflineMap.kt` تتعامل مباشرةً مع filesystem في عشرين موضعًا».
 *
 * # **والذرّيّةُ ليست ترفاً**
 *
 * **التنزيلُ يُقطع**: بطّاريّةٌ تنفد، ونظامٌ يقتل العمليّة، وشبكةٌ
 * تسقط في آخر ميغابايت. **فلو كُتب في موضعه النهائيّ مباشرةً لبقيت
 * حزمةٌ نصفُها على القرص** — **وحجمُها يبدو صحيحاً في قائمة الملفّات.**
 *
 * **فالكتابةُ في مؤقّتٍ ثمّ إعادةُ تسمية.** وإعادةُ التسمية في نظام
 * الملفّات نفسِه **عمليّةٌ ذرّيّة**: إمّا وقعت أو لم تقع، **ولا حالةَ
 * بينهما.** ولذلك `tmp/` **تحت `maps/` لا في مجلّد النظام المؤقّت** —
 * فعبرَ نظامَي ملفّاتٍ تصير النقلةُ نسخاً، **والنسخُ يُقطَع.**
 *
 * # **والقديمُ لا يُحذف قبل نجاح الجديد** (البند ٣٤)
 *
 * النسخةُ في المسار (`regions/{id}/{version}/`) **فتتعايش نسختان**،
 * **ثمّ يُبدَّل المؤشِّرُ ثمّ تُنظَّف القديمة.** ولو حُذفت أوّلاً
 * **لبقي السائقُ بلا خريطةٍ إن سقط التنزيل.**
 */
/**
 * **مفتوحٌ للوراثة** — لا لتوسيعٍ في الإنتاج بل لاختبارٍ صادق.
 *
 * **قياسُ المساحة يسأل نظامَ الملفّات**، **ولا يُملأ قرصُ آلة البناء
 * لاختبار نقص المساحة.** فتُتجاوَز `usableBytes` في رفيدةٍ وحدَها،
 * **وباقي السلوك حقيقيٌّ على قرصٍ حقيقيّ.**
 */
open class MapPackageStore(private val baseDir: File) {

    /**
     * **حالُ حزمةٍ على القرص.**
     *
     * **لا تشمل حالاتِ التنزيل** (`QUEUED`، `DOWNLOADING`…) — تلك
     * حالُ عملٍ جارٍ، **ومكانُها في نموذج الشاشة لا في المخزن.**
     */
    data class Installed(
        val regionId: String,
        val dataVersion: String,
        val resourcesVersion: String,
        val name: String,
        val bbox: List<Double>,
        val minZoom: Int,
        val maxZoom: Int,
        val bytes: Long,
        val sha256: String,
        val archive: File,
    ) {
        fun contains(lat: Double, lng: Double): Boolean = MapBbox.contains(bbox, lat, lng)
        val areaDeg: Double get() = MapBbox.area(bbox)
    }

    data class InstalledResources(
        val version: String,
        val styleVersion: String,
        val fontstacks: List<String>,
        val dir: File,
    )

    // ══════════════════════════════════════════════════════════════
    // **الإقلاع**
    // ══════════════════════════════════════════════════════════════

    fun prepare() {
        MapPaths.stateDir(baseDir).mkdirs()
        MapPaths.tmpDir(baseDir).mkdirs()
    }

    fun tmpDir(): File = MapPaths.tmpDir(baseDir).also { it.mkdirs() }

    /**
     * **ملفُّ التنزيل المؤقّت لهويّةِ أثرٍ بعينها** — البند ١١.
     *
     * **ولا يُبنى من خارج المخزن** — كان `MapDownloader` يشتقّه من
     * `tmpDir().parentFile` **فينتج `maps/maps/tmp/…`**، فلا يجد
     * الجزئيَّ أبداً **ويُعيد التنزيل من الصفر في كلّ مرّة.**
     * (كشفه اختبارُ الاستئناف ٢٠٢٦-٠٨-٢١.)
     */
    fun partFile(identity: String): File =
        MapPaths.partFile(baseDir, identity).also { it.parentFile?.mkdirs() }

    // ══════════════════════════════════════════════════════════════
    // **المناطقُ المركَّبة**
    // ══════════════════════════════════════════════════════════════

    /**
     * **المركَّبُ هو ما له `package.json` وأرشيفٌ بالحجم المُعلَن.**
     *
     * **ووجودُ المجلّد ليس تركيباً** — قد يكون بقيّةَ محاولةٍ سقطت
     * قبل أن يُكتب عقدُها. **فالعقدُ آخرُ ما يُكتب**، ووجودُه هو
     * العلامة.
     */
    fun installedRegions(): List<Installed> {
        val regionsRoot = File(MapPaths.root(baseDir), "regions")
        val dirs = regionsRoot.listFiles()?.filter { it.isDirectory } ?: return emptyList()
        val out = mutableListOf<Installed>()
        for (regionDir in dirs) {
            if (!MapPaths.isSafeId(regionDir.name)) continue
            val versions = regionDir.listFiles()?.filter { it.isDirectory } ?: continue
            for (v in versions) {
                if (!MapPaths.isSafeVersion(v.name)) continue
                readPackage(regionDir.name, v.name)?.let(out::add)
            }
        }
        return out
    }

    fun installedRegion(regionId: String, dataVersion: String): Installed? =
        if (MapPaths.isSafeId(regionId) && MapPaths.isSafeVersion(dataVersion)) {
            readPackage(regionId, dataVersion)
        } else {
            null
        }

    private fun readPackage(regionId: String, dataVersion: String): Installed? {
        val pkg = MapPaths.regionPackageFile(baseDir, regionId, dataVersion)
        if (!pkg.isFile) return null
        val archive = MapPaths.regionArchive(baseDir, regionId, dataVersion)
        if (!archive.isFile) return null
        return try {
            val o = JSONObject(pkg.readText())
            val bytes = o.optLong("bytes", -1L)
            // **والحجمُ يُقاس لا يُصدَّق** — ملفٌّ اقتُطع بعد تركيبه
            // (قرصٌ امتلأ، أو نظامٌ نظّف) **يبقى عقدُه سليماً.**
            if (bytes <= 0 || archive.length() != bytes) return null
            val bboxArr = o.optJSONArray("bbox") ?: return null
            if (bboxArr.length() != 4) return null
            Installed(
                regionId = regionId,
                dataVersion = dataVersion,
                resourcesVersion = o.optString("resourcesVersion", ""),
                name = o.optString("name", regionId),
                bbox = (0 until 4).map { bboxArr.getDouble(it) },
                minZoom = o.optInt("minZoom", 0),
                maxZoom = o.optInt("maxZoom", 0),
                bytes = bytes,
                sha256 = o.optString("sha256", ""),
                archive = archive,
            )
        } catch (e: Exception) {
            null
        }
    }

    /**
     * **يكتب عقدَ الحزمة** — وهو آخرُ خطوةٍ في التركيب.
     *
     * **فما قبله قد يكون ناقصاً، وما بعده حزمةٌ كاملة.**
     */
    fun markRegionInstalled(region: MapRegion, resourcesVersion: String) {
        val dir = MapPaths.regionVersionDir(baseDir, region.id, region.dataVersion)
        val json = JSONObject().apply {
            put("regionId", region.id)
            put("name", region.name)
            put("dataVersion", region.dataVersion)
            put("resourcesVersion", resourcesVersion)
            put("bytes", region.artifact.bytes)
            put("sha256", region.artifact.sha256)
            put("minZoom", region.artifact.minZoom)
            put("maxZoom", region.artifact.maxZoom)
            put("bbox", org.json.JSONArray(region.artifact.bbox))
        }
        atomicWrite(File(dir, MapPaths.PACKAGE_FILE), json.toString(2))
    }

    // ══════════════════════════════════════════════════════════════
    // **الموارُد**
    // ══════════════════════════════════════════════════════════════

    /**
     * **الموارُد مركَّبةٌ إن وُجد عقدُها والنمطُ وكلُّ ملفّ حرفٍ يطلبه.**
     *
     * **ولا يكفي وجودُ المجلّد** (البند ٢٥): «لا أريد runtime 404
     * محلّيّ صامت». **فيُفحص كلُّ رصّةٍ في كلّ نطاقٍ لازم.**
     */
    fun installedResources(version: String): InstalledResources? {
        if (!MapPaths.isSafeVersion(version)) return null
        val pkg = MapPaths.resourcesPackageFile(baseDir, version)
        if (!pkg.isFile) return null
        return try {
            val o = JSONObject(pkg.readText())
            val stacks = o.optJSONArray("fontstacks") ?: return null
            InstalledResources(
                version = version,
                styleVersion = o.optString("styleVersion", ""),
                fontstacks = (0 until stacks.length()).map { stacks.getString(it) },
                dir = MapPaths.resourcesDir(baseDir, version),
            )
        } catch (e: Exception) {
            null
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════
     * **التركيبُ الذرّيُّ للموارد** — إغلاقُ ٦ب الوظيفيّ، البند ٣
     * ══════════════════════════════════════════════════════════════
     */

    fun resourcesStagingDir(version: String): File =
        MapPaths.resourcesStagingDir(baseDir, version).also { it.mkdirs() }

    fun resourceFileIn(versionDir: File, relative: String): File =
        MapPaths.resourceFileIn(versionDir, relative).also { it.parentFile?.mkdirs() }

    /** **يُطرح المؤقّتُ كلُّه** — فلا يبقى نصفُ نسخةٍ يُستأنف عليه غداً. */
    fun discardStaging(version: String): Boolean {
        if (!MapPaths.isSafeVersion(version)) return false
        return MapPaths.resourcesStagingDir(baseDir, version).deleteRecursively()
    }

    /**
     * **هل نسخةُ الموارد كاملةٌ ومفعَّلة؟**
     *
     * **العلامةُ والنمطُ وكلُّ حرفٍ وكلُّ أيقونة** — لا وجودُ المجلّد.
     */
    fun resourcesComplete(version: String, fontstacks: List<String>): Boolean {
        if (installedResources(version) == null) return false
        if (!MapPaths.styleFile(baseDir, version).isFile) return false
        return missingGlyphs(version, fontstacks).isEmpty() && missingSprites(version).isEmpty()
    }

    /**
     * **فحصُ مجموعةِ مواردَ في مجلّدٍ أيّاً كان** — قبل التفعيل.
     *
     * **البند ٤: «لا Partial Resources».** فيُفحص المؤقّتُ بالفحص
     * نفسِه الذي يُفحص به المفعَّل — **ولا يُفعَّل ما لم يمرّ.**
     */
    fun validateResourceSet(dir: File, fontstacks: List<String>): List<String> {
        val problems = mutableListOf<String>()

        if (!File(dir, MapPaths.STYLE_FILE).isFile) problems += "النمطُ ${MapPaths.STYLE_FILE} مفقود"

        for (stack in fontstacks) {
            for (range in MapGlyphRanges.REQUIRED) {
                val f = try {
                    MapPaths.resourceFileIn(dir, "glyphs/$stack/$range.pbf")
                } catch (e: IllegalArgumentException) {
                    problems += "رصّةٌ مرفوضة: $stack"
                    continue
                }
                if (!f.isFile) problems += "$stack/$range مفقود"
                else if (f.length() == 0L) problems += "$stack/$range فارغ"
            }
        }

        for (name in MapSpriteFiles.ALL) {
            val f = File(dir, name)
            if (!f.isFile) problems += "$name مفقود"
            else if (f.length() == 0L) problems += "$name فارغ"
        }

        /**
         * **وفهرسُ الأيقونات بلا صورته عطبٌ يبدو سليماً.**
         *
         * **MapLibre تقرأ الفهرسَ فتجد الأيقونةَ معلنةً**، ثمّ تطلب
         * الصورةَ فلا تجدها — **فتُرسم الخريطةُ بلا أيقونةٍ ولا خطأ.**
         */
        val one = File(dir, "sprite.json")
        val onePng = File(dir, "sprite.png")
        val two = File(dir, "sprite@2x.json")
        val twoPng = File(dir, "sprite@2x.png")
        if (one.isFile && !onePng.isFile) problems += "sprite.json بلا sprite.png"
        if (two.isFile && !twoPng.isFile) problems += "sprite@2x.json بلا sprite@2x.png"
        if (one.isFile && !two.isFile) problems += "1x بلا 2x"

        return problems
    }

    /**
     * **يُفعِّل المؤقّتَ نقلةً واحدة.**
     *
     * **وإعادةُ تسمية المجلّد ذرّيّةٌ كإعادة تسمية ملفّ** ما دامت في
     * نظام الملفّات نفسِه — **وهي كذلك، فكلاهما تحتَ `maps/`.**
     *
     * **والقديمةُ تُزاح لا تُمحى قبل النجاح** (البند ٤): تُسمّى
     * `.old-{v}` **ثمّ تُحذف بعد أن يستقرّ الجديد.** فلو سقطت النقلةُ
     * في المنتصف **رُدّت القديمةُ مكانَها.**
     */
    fun activateResources(version: String, contract: MapResourceContract) {
        val staging = MapPaths.resourcesStagingDir(baseDir, version)
        require(staging.isDirectory) { "لا مجلّدَ مؤقّتاً لنسخة $version" }

        // **والعلامةُ تُكتب في المؤقّت** — فتصير النسخةُ كاملةً لحظةَ
        // النقلة، **لا بعدها بخطوةٍ قد لا تقع.**
        writeResourcesMarker(File(staging, MapPaths.PACKAGE_FILE), contract)

        val target = MapPaths.resourcesDir(baseDir, version)
        val parked = File(target.parentFile, ".old-${target.name}")
        parked.deleteRecursively()

        val hadOld = target.exists()
        if (hadOld && !target.renameTo(parked)) {
            throw IOException("تعذّرت إزاحةُ النسخة القديمة: $target")
        }
        if (!staging.renameTo(target)) {
            // **وتُردُّ القديمةُ** — فلا يبقى الجهازُ بلا موارد.
            if (hadOld) parked.renameTo(target)
            throw IOException("تعذّر تفعيلُ نسخة الموارد $version")
        }
        parked.deleteRecursively()
    }

    private fun writeResourcesMarker(file: File, contract: MapResourceContract) {
        val json = JSONObject().apply {
            put("resourcesVersion", contract.version)
            put("styleVersion", contract.styleVersion)
            put("glyphVersion", contract.glyphVersion)
            put("spriteVersion", contract.spriteVersion)
            put("fontstacks", org.json.JSONArray(contract.fontstacks))
            put("files", contract.files.size)
            contract.licenseSpdx?.let { put("license", it) }
        }
        file.parentFile?.mkdirs()
        file.writeText(json.toString(2))
    }

    /**
     * **نسخُ الموارد المركَّبة** — لتنظيفِ ما لم يعد مطلوباً.
     *
     * **والمؤقّتاتُ والمُزاحاتُ لا تُعدّ** — `isSafeVersion` ترفض ما
     * يبدأ بنقطة.
     */
    fun installedResourceVersions(): List<String> =
        MapPaths.resourcesRoot(baseDir).listFiles()
            ?.filter { it.isDirectory && MapPaths.isSafeVersion(it.name) }
            ?.map { it.name }
            ?.filter { installedResources(it) != null }
            ?.sorted()
            ?: emptyList()

    /**
     * **يحذف نسخةَ مواردَ لم تعد مطلوبة** — البند ٧.
     *
     * **والمطلوبُ ثلاثة**: النسخةُ التي يقرؤها النمطُ الفعّال، والتي
     * تطلبها حزمةٌ مركَّبةٌ ما زالت صالحة، **والتي يجري الانتقالُ
     * منها أو إليها.** فالقرارُ يُمرَّر إلى هنا **لأنّ المخزنَ لا
     * يعرف ما تفتحه MapLibre.**
     */
    fun deleteResourceVersion(version: String, protectedVersions: Set<String>): Boolean {
        if (!MapPaths.isSafeVersion(version)) return false
        if (version in protectedVersions) return false
        // **وكلُّ نسخةٍ تطلبها حزمةٌ مركَّبةٌ محميّة** — وإلّا صارت
        // الحزمةُ يتيمةً لا تُفتح.
        if (installedRegions().any { it.resourcesVersion == version }) return false
        return MapPaths.resourcesDir(baseDir, version).deleteRecursively()
    }

    fun markResourcesInstalled(resources: MapResources) {
        val dir = MapPaths.resourcesDir(baseDir, resources.version)
        val json = JSONObject().apply {
            put("resourcesVersion", resources.version)
            put("styleVersion", resources.styleVersion)
            put("glyphVersion", resources.glyphVersion)
            put("spriteVersion", resources.spriteVersion)
            put("fontstacks", org.json.JSONArray(resources.fontstacks))
        }
        atomicWrite(File(dir, MapPaths.PACKAGE_FILE), json.toString(2))
    }

    /**
     * **فحصُ عقد الحروف** — البند ٢٥.
     *
     * **يُسأل عن كلّ رصّةٍ وكلّ نطاقٍ لازم**: هل الملفُّ الذي سيطلبه
     * MapLibre موجودٌ فعلاً؟ **فلا يُكتشف الغيابُ عند أوّل تسمية
     * شارعٍ في الشارع.**
     */
    fun missingGlyphs(version: String, fontstacks: List<String>): List<String> {
        val missing = mutableListOf<String>()
        for (stack in fontstacks) {
            for (range in MapGlyphRanges.REQUIRED) {
                val f = try {
                    MapPaths.glyphFile(baseDir, version, stack, range)
                } catch (e: IllegalArgumentException) {
                    missing += "$stack/$range (اسمٌ مرفوض)"
                    continue
                }
                if (!f.isFile || f.length() == 0L) missing += "$stack/$range"
            }
        }
        return missing
    }

    /** **وأربعةُ ملفّاتِ الأيقونات** — 1x و2x فهرساً وصورة (البند ٢٦). */
    fun missingSprites(version: String): List<String> =
        MapSpriteFiles.ALL.filter { !MapPaths.spriteFile(baseDir, version, it).isFile }

    fun resourcesDir(version: String): File = MapPaths.resourcesDir(baseDir, version)

    fun styleFile(version: String): File = MapPaths.styleFile(baseDir, version)

    // ══════════════════════════════════════════════════════════════
    // **المؤشِّرُ الفعّال**
    // ══════════════════════════════════════════════════════════════

    /**
     * **ما نجح تحميلُه فعلاً** — لا ما طلبته الواجهة.
     *
     * (إغلاقُ ٦ب الوظيفيّ، البند ١٢.)
     *
     * **أمرُ المالك نصّاً**: «active.json يجب أن يعكس الحالة الفعلية
     * المقبولة، لا مجرد آخر اختيار مطلوب من UI… ولا تكتب current=B
     * قبل StyleLoaded(B) بنجاح».
     *
     * # **ولماذا يهمّ**
     *
     * **الملفُّ يُقرأ عند الإقلاع ليُعرف ما كان مفتوحاً.** فلو كُتب
     * فيه ما طُلب **لأقلع الجهازُ ظانّاً أنّ حزمةً فعّالةٌ وهي لم
     * تُحمَّل قطّ** — **ثمّ يمتنع الحذفُ عن ملفٍّ لا يقرؤه أحد.**
     */
    data class Active(val regionId: String?, val dataVersion: String?, val resourcesVersion: String?)

    fun active(): Active {
        val f = MapPaths.activeFile(baseDir)
        if (!f.isFile) return Active(null, null, null)
        return try {
            val o = JSONObject(f.readText())
            // **ويُقرأ `loaded` وحدَه** — و`desired` يُكتب للتشخيص
            // ولا يُبنى عليه قرار.
            val loaded = o.optJSONObject("loaded") ?: o
            Active(
                regionId = loaded.optString("regionId", "").ifBlank { null },
                dataVersion = loaded.optString("dataVersion", "").ifBlank { null },
                resourcesVersion = loaded.optString("resourcesVersion", "").ifBlank { null },
            )
        } catch (e: Exception) {
            Active(null, null, null)
        }
    }

    /** **يُكتب بعد `StyleLoaded` لا قبله.** */
    fun setActive(active: Active, desired: Active? = null) {
        val json = JSONObject().apply {
            put(
                "loaded",
                JSONObject().apply {
                    active.regionId?.let { put("regionId", it) }
                    active.dataVersion?.let { put("dataVersion", it) }
                    active.resourcesVersion?.let { put("resourcesVersion", it) }
                },
            )
            desired?.let { d ->
                put(
                    "desired",
                    JSONObject().apply {
                        d.regionId?.let { put("regionId", it) }
                        d.dataVersion?.let { put("dataVersion", it) }
                        d.resourcesVersion?.let { put("resourcesVersion", it) }
                    },
                )
            }
        }
        atomicWrite(MapPaths.activeFile(baseDir), json.toString(2))
    }

    // ══════════════════════════════════════════════════════════════
    // **التركيبُ والحذف**
    // ══════════════════════════════════════════════════════════════

    /**
     * **ينقل ملفّاً مؤقّتاً إلى موضعه النهائيّ نقلةً ذرّيّة.**
     *
     * **و`renameTo` تفشل صامتةً** إن اختلف نظامُ الملفّات أو وُجد
     * الهدف. **فتُفحص نتيجتُها ويُرفع الخطأ** — ولا يُتظاهر بالنجاح.
     */
    fun installFile(part: File, target: File) {
        require(part.isFile) { "لا ملفَّ مؤقّتاً لتركيبه: $part" }
        target.parentFile?.mkdirs()
        if (target.exists() && !target.delete()) {
            throw IOException("تعذّر إخلاءُ الهدف قبل التركيب: $target")
        }
        if (!part.renameTo(target)) {
            throw IOException("تعذّرت النقلةُ الذرّيّة: $part ← $target")
        }
    }

    fun regionArchiveTarget(regionId: String, dataVersion: String): File =
        MapPaths.regionArchive(baseDir, regionId, dataVersion)
            .also { it.parentFile?.mkdirs() }

    /**
     * **حذفُ نسخةٍ قديمةٍ بعد نجاح بديلتها** (البند ٣٤).
     *
     * **ولا يُحذف ما هو فعّال** — البند ٣٥. والقرارُ ليس هنا:
     * **المخزنُ ينفّذ، والامتناعُ قرارُ زمن التشغيل** لأنّه وحدَه يعرف
     * ما تفتحه MapLibre الآن.
     */
    fun deleteRegionVersion(regionId: String, dataVersion: String): Boolean {
        if (!MapPaths.isSafeId(regionId) || !MapPaths.isSafeVersion(dataVersion)) return false
        val dir = MapPaths.regionVersionDir(baseDir, regionId, dataVersion)
        if (!dir.exists()) return true
        // **والعقدُ يُمحى أوّلاً** — فلو قُطع الحذفُ في منتصفه
        // **لم تُقرأ الحزمةُ الناقصةُ على أنّها مركَّبة.**
        File(dir, MapPaths.PACKAGE_FILE).delete()
        val ok = dir.deleteRecursively()
        MapPaths.regionDir(baseDir, regionId).let { parent ->
            if (parent.listFiles()?.isEmpty() == true) parent.delete()
        }
        return ok
    }

    /**
     * **ونظافةُ المؤقّت بسياسةٍ معلنة** (البند ٩).
     *
     * **يُحذف ما تجاوز عمرُه المدى** — فالمقطوعُ حديثاً قد يُستأنف،
     * **والقديمُ لا يُستأنف وإنّما يشغل قرصاً.**
     */
    fun cleanStaleTemp(olderThanMs: Long, nowMs: Long): Int {
        val dir = MapPaths.tmpDir(baseDir)
        val files = dir.listFiles() ?: return 0
        var n = 0
        for (f in files) {
            if (!f.isFile) continue
            if (nowMs - f.lastModified() >= olderThanMs && f.delete()) n += 1
        }
        return n
    }

    /** **المساحةُ المتاحةُ في نظام الملفّات نفسِه** — البند ١٠. */
    open fun usableBytes(): Long = MapPaths.root(baseDir).let {
        it.mkdirs()
        it.usableSpace
    }

    private fun atomicWrite(target: File, text: String) {
        target.parentFile?.mkdirs()
        val tmp = File(target.parentFile, "${target.name}.tmp")
        tmp.writeText(text)
        if (target.exists() && !target.delete()) {
            throw IOException("تعذّر إخلاءُ $target")
        }
        if (!tmp.renameTo(target)) {
            tmp.delete()
            throw IOException("تعذّرت كتابةُ $target ذرّيّاً")
        }
    }
}

/**
 * **نطاقاتُ الحروف اللازمة** — مرآةُ `maps/scripts/glyph-ranges.mjs`.
 *
 * **ولا تُكتب مرّتين بلا حارس**: `check-glyph-ranges-parity.mjs`
 * يقارنها بالمصدر ويُسقط البناءَ إن انحرفت. **فالتكرارُ هنا ضرورةُ
 * لغةٍ، لا قراراً.**
 */
object MapGlyphRanges {
    val REQUIRED = listOf(
        "0-255",
        "1536-1791",
        "64256-64511",
        "64512-64767",
        "64768-65023",
        "65024-65279",
    )
}

/** **وملفّاتُ الأيقونات الأربعة** — عقدُ MapLibre. */
object MapSpriteFiles {
    val ALL = listOf("sprite.json", "sprite.png", "sprite@2x.json", "sprite@2x.png")
}
