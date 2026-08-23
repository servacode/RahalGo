package com.rahalgo.map.data

import org.json.JSONObject
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **ربطُ النمط — ثلاثةُ عناوينَ لا أكثر**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٢٣ و٢٥ و٢٦.)
 *
 * **أمرُ المالك نصّاً**: «Online/Offline binding فقط يبدل: tile source
 * URI · glyph URI · sprite URI. أما layers · paint · layout ·
 * source-layer فتبقى canonical».
 *
 * # **وهذه مرآةُ `maps/scripts/bind-style.mjs`**
 *
 * **الوظيفةُ واحدةٌ في لغتين**: النمطُ الدلاليُّ واحدٌ في الويب
 * والهاتف، **والاختلافُ عنوانٌ.** فما يبدّله هذا الملفُّ هو نفسُه ما
 * يبدّله ذاك، **وحارسٌ يقارنهما** (`check-style-binding-parity.mjs`).
 *
 * **ولو أضاف أحدُهما طبقةً لانحرف النمطان** — وهو بعينه ما حذّر منه
 * `CLAUDE.md`: «ووثيقتان تتناقضان صامتتين أسوأُ من لا وثيقة».
 */
object MapStyleBinding {

    const val BINDING_ONLINE = "online"
    const val BINDING_OFFLINE = "offline"

    /** **مراسي الطبقات من ٦أ** — البند ٤٠. */
    const val ANCHOR_ROUTE = "rahalgo-route-anchor"
    const val ANCHOR_MARKERS = "rahalgo-marker-anchor"
    const val ANCHOR_BELOW_LABELS = "belowLabels"

    /**
     * **النمطُ مربوطاً** — نصُّه، وما يحتاجه من يفحصه.
     */
    data class Bound(
        val json: String,
        val binding: String,
        val tileUri: String,
        val glyphsUri: String,
        val spriteUri: String,
        val fontstacks: List<String>,
        val attribution: String,
    )

    /**
     * **يربط النمطَ القانونيَّ بمصدرٍ بعيد.**
     *
     * `pmtiles://https://…` — والعنوانُ من الفهرس لا من الشيفرة
     * (البند ٣).
     */
    fun online(
        canonical: String,
        tileUrl: String,
        resourcesBaseUrl: String,
        attribution: String,
        allowLoopbackHttp: Boolean = false,
    ): Bound = bind(
        canonical = canonical,
        binding = BINDING_ONLINE,
        tileUri = MapUrls.onlinePmtiles(tileUrl, allowLoopbackHttp),
        resourcesUri = resourcesBaseUrl.trimEnd('/'),
        attribution = attribution,
    )

    /**
     * **ويربطه بحزمةٍ ومواردَ على القرص.**
     *
     * **العنوانان يُبنيان من `File`** — فالمسافةُ في `RahalGo Regular`
     * تُرمَّز، **ولا يُقطع العنوان.**
     */
    fun offline(
        canonical: String,
        archive: File,
        resourcesDir: File,
        attribution: String,
    ): Bound = bind(
        canonical = canonical,
        binding = BINDING_OFFLINE,
        tileUri = MapUrls.offlinePmtiles(archive),
        resourcesUri = MapUrls.localDirUri(resourcesDir),
        attribution = attribution,
    )

    private fun bind(
        canonical: String,
        binding: String,
        tileUri: String,
        resourcesUri: String,
        attribution: String,
    ): Bound {
        val style = JSONObject(canonical)

        // **وحقولُ الشرح لا تُنقل** — تُقرأ في المستودع لا في جهاز.
        style.remove("_note")
        style.remove("_contract")

        val sources = style.optJSONObject("sources")
            ?: throw IllegalArgumentException("النمط: لا `sources`")
        val base = sources.optJSONObject("base")
            ?: throw IllegalArgumentException("النمط: لا مصدرَ `base`")
        base.put("url", tileUri)

        /**
         * **والإسنادُ يُحقن في المصدر** — البند ٢٧.
         *
         * **MapLibre تعرض `attribution` المصدر**، فلا تُبنى واجهةٌ
         * جديدةٌ لعرضه. **وهو شرطُ ترخيصٍ لا زينة**، فلا يُترك
         * لبياناتٍ داخلَ PMTiles قد لا تُقرأ.
         */
        if (attribution.isNotBlank()) base.put("attribution", attribution)

        val glyphs = "$resourcesUri/glyphs/{fontstack}/{range}.pbf"
        val sprite = "$resourcesUri/sprite"
        style.put("glyphs", glyphs)
        style.put("sprite", sprite)

        val metadata = style.optJSONObject("metadata") ?: JSONObject()
        metadata.put("rahalgo:binding", binding)
        style.put("metadata", metadata)

        return Bound(
            json = style.toString(),
            binding = binding,
            tileUri = tileUri,
            glyphsUri = glyphs,
            spriteUri = sprite,
            fontstacks = fontstacksOf(style),
            attribution = attribution,
        )
    }

    /**
     * **رصّاتُ الخطوط التي يطلبها النمطُ فعلاً.**
     *
     * **تُقرأ من الطبقات لا من الإعداد** — فالنمطُ هو الذي يطلب،
     * **وما لا يطلبه لا يلزم تركيبُه** (البند ٢٥).
     */
    fun fontstacksOf(style: JSONObject): List<String> {
        val out = LinkedHashSet<String>()
        val layers = style.optJSONArray("layers") ?: return emptyList()
        for (i in 0 until layers.length()) {
            val layout = layers.optJSONObject(i)?.optJSONObject("layout") ?: continue
            val fonts = layout.optJSONArray("text-font") ?: continue
            for (j in 0 until fonts.length()) out += fonts.getString(j)
        }
        return out.toList()
    }

    fun fontstacksOf(canonical: String): List<String> = fontstacksOf(JSONObject(canonical))

    /**
     * **ومراسي الطبقات تُقرأ من النمط** — البند ٤٠.
     *
     * **وغيابُ مِرساةٍ يُسقط في الفحص لا يُتجاوَز** — أمرُ المالك:
     * «Fail fast في QA، لا fallback عشوائي إلى آخر Layer».
     */
    fun anchors(canonical: String): Map<String, String> {
        val style = JSONObject(canonical)
        val a = style.optJSONObject("_anchors") ?: return emptyMap()
        return a.keys().asSequence().associateWith { a.getString(it) }
    }

    /** **وترتيبُ الطبقات** — يُقرأ ليُختبر. */
    fun layerIds(styleJson: String): List<String> {
        val layers = JSONObject(styleJson).optJSONArray("layers") ?: return emptyList()
        return (0 until layers.length()).map { layers.getJSONObject(it).getString("id") }
    }

    /**
     * **والتسميةُ العربيّةُ عقدٌ في النمط لا في الشيفرة** — البند ٢٤.
     *
     * `["coalesce", ["get","name:ar"], ["get","name"]]` — **فما لا
     * اسمَ عربيَّ له يُعرض باسمه، وما لا اسمَ له أصلاً لا يُعرض.**
     *
     * **ويُفحص أنّه لم ينحرف** — فلو صار `name` أوّلاً **لظهرت
     * الخريطةُ لاتينيّةً في مدينةٍ عربيّة.**
     */
    fun labelExpressionsAreArabicFirst(canonical: String): Boolean {
        val layers = JSONObject(canonical).optJSONArray("layers") ?: return false
        var seen = 0
        for (i in 0 until layers.length()) {
            val layout = layers.optJSONObject(i)?.optJSONObject("layout") ?: continue
            val field = layout.opt("text-field") ?: continue
            val text = field.toString()
            if (!text.contains("name:ar")) return false
            if (text.indexOf("name:ar") > text.indexOf("\"name\"").let {
                    if (it < 0) Int.MAX_VALUE else it
                }
            ) {
                return false
            }
            seen += 1
        }
        return seen > 0
    }
}
