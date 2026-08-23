package com.rahalgo.map.data

import org.json.JSONArray
import org.json.JSONObject

/**
 * ══════════════════════════════════════════════════════════════════
 * **الفهرس — عقدُ ٦أ كما يقرؤه الهاتف**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٨ و٤٤.)
 *
 * **أمرُ المالك نصّاً**: «ولا تثق بقيمة حجم أو path دون validation».
 *
 * # **والفهرسُ نصٌّ من الشبكة لا حقيقةٌ**
 *
 * **كلُّ حقلٍ هنا يمكن أن يكذب**: حجمٌ سالب، وبصمةٌ ليست ستّاً وأربعين
 * محرفاً، وصندوقٌ مقلوب، ومستوياتٌ عكسيّة، **وعنوانٌ `file:///` يفتح
 * ملفَّ الهاتف.**
 *
 * **فالقراءةُ تحقُّقٌ لا نسخ.** وما لم يمرّ **يُرفض كلُّه** — لا
 * «يُصلَح» ولا «يُتجاهل الحقلُ المعطوب»، **لأنّ فهرساً نصفَ صحيحٍ
 * يُنتج حزمةً نصفَ صحيحة.**
 */
data class MapManifest(
    val schemaVersion: Int,
    val dataVersion: String,
    val tileSchema: String,
    val resourcesVersion: String,
    val base: MapArtifact,
    val regions: List<MapRegion>,
    val resources: MapResources,
    val attribution: String,
) {
    fun region(id: String): MapRegion? = regions.firstOrNull { it.id == id }

    companion object {
        /**
         * **نسخةُ المخطّط التي يفهمها هذا التطبيق.**
         *
         * **وأكبرُ منها يُرفض لا يُحاول** — فهرسٌ من مستقبلٍ قد يحمل
         * معنًى مختلفاً لحقلٍ نعرف اسمَه، **والقراءةُ المتفائلةُ تركّب
         * حزمةً لا تُفهم.**
         */
        const val SUPPORTED_SCHEMA = 1

        fun parse(text: String): MapManifest = fromJson(JSONObject(text))

        fun fromJson(o: JSONObject): MapManifest {
            val schema = o.optInt("schemaVersion", -1)
            require(schema > 0) { "الفهرس: schemaVersion غائبٌ أو غيرُ صحيح" }
            require(schema <= SUPPORTED_SCHEMA) {
                "الفهرس: schemaVersion=$schema أحدثُ ممّا يفهمه التطبيق ($SUPPORTED_SCHEMA)"
            }

            val dataVersion = MapPaths.requireSafeId(o.optString("dataVersion", ""), "نسخةِ البيانات")
            val resourcesVersion =
                MapPaths.requireSafeId(o.optString("resourcesVersion", ""), "نسخةِ الموارد")
            val tileSchema = o.optString("tileSchema", "")
            require(tileSchema.isNotBlank()) { "الفهرس: tileSchema غائب" }

            val base = MapArtifact.fromJson(o.getJSONObject("base"), "base")

            val regionsJson: JSONArray = o.optJSONArray("regions") ?: JSONArray()
            val regions = (0 until regionsJson.length())
                .map { MapRegion.fromJson(regionsJson.getJSONObject(it)) }
            // **ولا معرِّفَ مكرَّر** — وإلّا صار الاختيارُ بينهما بالحظّ.
            require(regions.map { it.id }.toSet().size == regions.size) {
                "الفهرس: معرِّفُ منطقةٍ مكرَّر"
            }

            val attribution = o.optString("attribution", "")
            require(attribution.isNotBlank()) {
                // **الإسنادُ شرطُ ترخيص** (البند ٢٧) — لا حقلٌ تجميليّ.
                "الفهرس: attribution غائب — والإسنادُ شرطُ ترخيصٍ لا زينة"
            }

            return MapManifest(
                schemaVersion = schema,
                dataVersion = dataVersion,
                tileSchema = tileSchema,
                resourcesVersion = resourcesVersion,
                base = base,
                regions = regions,
                resources = MapResources.fromJson(o.getJSONObject("resources"), resourcesVersion),
                attribution = attribution,
            )
        }
    }
}

/** **أثرٌ يُنزَّل** — عنوانٌ وحجمٌ وبصمة. */
data class MapArtifact(
    val url: String,
    val bytes: Long,
    val sha256: String,
    val minZoom: Int,
    val maxZoom: Int,
    val bbox: List<Double>,
) {
    companion object {
        fun fromJson(o: JSONObject, what: String): MapArtifact {
            val url = o.optString("url", "")
            require(url.isNotBlank()) { "$what: url غائب" }
            require(MapUrls.isSafeRelative(url)) { "$what: url غيرُ مقبول: $url" }

            val bytes = o.optLong("bytes", -1L)
            require(bytes > 0) { "$what: bytes غيرُ صحيح ($bytes)" }

            val sha = o.optString("sha256", "")
            require(MapUrls.isSha256(sha)) { "$what: sha256 غيرُ صحيح" }

            val minZoom = o.optInt("minZoom", -1)
            val maxZoom = o.optInt("maxZoom", -1)
            require(minZoom in 0..24 && maxZoom in 0..24 && minZoom <= maxZoom) {
                "$what: مدى التقريب غيرُ صحيح ($minZoom–$maxZoom)"
            }

            return MapArtifact(url, bytes, sha, minZoom, maxZoom, MapBbox.fromJson(o, what))
        }
    }
}

/** **حزمةُ منطقة** — أثرٌ باسمٍ ومعرِّف. */
data class MapRegion(
    val id: String,
    val name: String,
    val dataVersion: String,
    val artifact: MapArtifact,
) {
    val bbox: List<Double> get() = artifact.bbox

    /**
     * **هل تغطّي الحزمةُ هذه النقطة؟**
     *
     * **الصندوقُ من الفهرس لا من اسم المدينة** (البند ١٦) — فالاسمُ
     * نصٌّ للعرض، **والتغطيةُ هندسة.**
     */
    fun contains(lat: Double, lng: Double): Boolean = MapBbox.contains(bbox, lat, lng)

    /** **ومساحتُه** — بها يُرجَّح الأخصُّ حين تتداخل حزمتان. */
    val areaDeg: Double get() = MapBbox.area(bbox)

    companion object {
        fun fromJson(o: JSONObject): MapRegion {
            val id = MapPaths.requireSafeId(o.optString("id", ""), "المنطقة")
            val dataVersion =
                MapPaths.requireSafeId(o.optString("dataVersion", ""), "نسخةِ بيانات $id")
            val name = o.optString("name", "").ifBlank { id }
            return MapRegion(id, name, dataVersion, MapArtifact.fromJson(o, "region $id"))
        }
    }
}

/** **الموارُد المشتركة** — نسخةٌ واحدةٌ لكلّ المناطق (البند ١٤). */
data class MapResources(
    val version: String,
    val url: String,
    val styleVersion: String,
    val glyphVersion: String,
    val spriteVersion: String,
    val fontstacks: List<String>,
) {
    companion object {
        fun fromJson(o: JSONObject, version: String): MapResources {
            val url = o.optString("url", "")
            require(url.isNotBlank()) { "الموارد: url غائب" }
            require(MapUrls.isSafeRelative(url)) { "الموارد: url غيرُ مقبول: $url" }

            /**
             * **ورصّاتُ الخطوط تُقرأ بشكلين.**
             *
             * **كانت أسماءً في أوّل ٦أ**، **وصارت سجلّاتٍ في إغلاق
             * الأدوات** (فيها العائلةُ والترخيصُ والبصمة). **فيُقبل
             * الشكلان ويُؤخذ الاسم** — وإلّا سقط فهرسٌ سليمٌ لأنّ
             * حقلاً اغتنى.
             */
            val arr = o.optJSONArray("fontstacks") ?: JSONArray()
            val stacks = (0 until arr.length()).map { i ->
                when (val v = arr.get(i)) {
                    is String -> v
                    is JSONObject -> v.optString("name", "")
                    else -> ""
                }
            }.filter { it.isNotBlank() }
            require(stacks.isNotEmpty()) { "الموارد: لا رصّةَ خطٍّ معلنة" }

            return MapResources(
                version = version,
                url = url,
                styleVersion = o.optString("styleVersion", ""),
                glyphVersion = o.optString("glyphVersion", ""),
                spriteVersion = o.optString("spriteVersion", ""),
                fontstacks = stacks,
            )
        }
    }
}

/** **الصندوق** — `[غرب, جنوب, شرق, شمال]` كما يكتبه ٦أ. */
object MapBbox {
    fun fromJson(o: JSONObject, what: String): List<Double> {
        val arr = o.optJSONArray("bbox") ?: throw IllegalArgumentException("$what: bbox غائب")
        require(arr.length() == 4) { "$what: bbox ليس أربعةَ أعداد" }
        val v = (0 until 4).map { arr.getDouble(it) }
        require(v.all { it.isFinite() }) { "$what: bbox فيه قيمةٌ غيرُ منتهية" }
        require(v[0] in -180.0..180.0 && v[2] in -180.0..180.0) { "$what: خطُّ طولٍ خارجَ المدى" }
        require(v[1] in -90.0..90.0 && v[3] in -90.0..90.0) { "$what: خطُّ عرضٍ خارجَ المدى" }
        // **والمقلوبُ يُرفض** — صندوقٌ غربُه شرقَ شرقِه يغطّي كلَّ شيءٍ أو لا شيء.
        require(v[0] < v[2] && v[1] < v[3]) { "$what: bbox مقلوب" }
        return v
    }

    fun contains(bbox: List<Double>, lat: Double, lng: Double): Boolean {
        if (bbox.size != 4) return false
        if (!lat.isFinite() || !lng.isFinite()) return false
        return lng >= bbox[0] && lng <= bbox[2] && lat >= bbox[1] && lat <= bbox[3]
    }

    fun area(bbox: List<Double>): Double =
        if (bbox.size != 4) Double.MAX_VALUE else (bbox[2] - bbox[0]) * (bbox[3] - bbox[1])
}
