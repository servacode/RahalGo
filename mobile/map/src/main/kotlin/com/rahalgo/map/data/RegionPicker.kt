package com.rahalgo.map.data

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أيُّ منطقةٍ تخصّ هذا السائق — تُشتقّ ولا تُكتب في الكود**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «لنرتاح لاحقاً، نفعل كلَّ خرائط سوريا —
 *  كي لا نضطرَّ لاحقاً لتعديل البرنامج والكود».)
 *
 * # وخوفُه كان في محلّه — نصفَ محلّه
 *
 * **الفهرسُ يُقرأ حيّاً منذ بُني** (`MapManifest.regions`): إضافةُ
 * مدينةٍ إليه لا تحتاج سطرَ كودٍ ولا تحديثَ تطبيق.
 *
 * **لكنّ تطبيقَ السائق كان يثبّت «الرقّة» في الكود** — معرِّفاً
 * واسماً. **فلو بُنيت ستّون حزمةً لَما نزّل إلّا واحدة**، وبقيت
 * التسعُ والخمسون على الخادم لا يراها أحد.
 *
 * **فهذا الملفُّ هو التعديلُ الذي لا يُعاد**: بعده تُفتح المدنُ
 * بإضافة صفٍّ في `manifest.json` — **بياناتٌ لا شيفرة.**
 *
 * # وكيف تُشتقّ
 *
 * **من موضعه على الأرض لا من اسم مدينته.** الاسمُ يُكتب بصيغٍ
 * («الرقة» · «الرقّة» · «Raqqa») **ومقارنةُ نصوصٍ عربيّةٍ بالحروف
 * تخطئ في الهمزة والشدّة والألف المقصورة.** **والإحداثيّةُ لا
 * تختلف في إملائها.**
 *
 * # وحين يقع في تداخُلٍ بين منطقتين
 *
 * **تُؤخذ الأصغرُ مساحة** — **والكبرى تحتوي الصغرى غالباً**، ومن
 * أخذ الكبرى نزّل خريطةَ محافظةٍ ليقود في حيّ.
 *
 * # وحين لا يقع في شيء
 *
 * **يُردّ فراغٌ ولا تُخمَّن منطقة** — **وخريطةُ مدينةٍ أخرى أسوأُ من
 * لا خريطة**: تُنزَّل ميغاباتٌ على حزمته ثمّ لا يجد شارعَه فيها.
 */
object RegionPicker {

    /**
     * **المنطقةُ التي تحوي هذه النقطة** — أو فارغٌ.
     *
     * @param regions ما في الفهرس، كما جاء من الشبكة.
     */
    fun of(regions: List<MapRegion>, lat: Double, lng: Double): MapRegion? {
        // **الحدودُ أوّلاً** — المحافظةُ التي تقع النقطةُ داخلها فعلاً. **وريفُ
        // دمشقَ يحيط بدمشق** فتقع فيهما معاً، **فالأصغرُ يغلب** كما كان.
        val inside = regions.filter { r -> r.polygon.any { inRing(it, lat, lng) } }
        if (inside.isNotEmpty()) return inside.minByOrNull { it.areaDeg }
        // **والمستطيلُ لما خرج عن كلّ حدّ** — هامشٌ عند الحدود، أو فهرسٌ بلا حدود.
        return regions
            .filter { contains(it.bbox, lat, lng) }
            .minByOrNull { it.areaDeg }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **أيُّ أرشيفٍ يُقرأ أونلاين — حزمةُ المدينة أم سوريا كلُّها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (طلبُ المالك ٢٠٢٦-١٠-٠٩: أوّلُ فتحةٍ للخريطة بطيئة.)
     *
     * **قِيس**: الأونلاين يقرأ أرشيفَ سوريا كلَّها (٥٠٢ م.ب) بطلباتٍ
     * متتابعة — الترويسة ثمّ الفهرس الجذر ثمّ الأوراق ثمّ البلاطات،
     * **كلُّ واحدٍ نحوَ ٠٫٦ ثانية.** **وأرشيفُ مدينةٍ فهرسُه صغير**،
     * فتقلّ الطلباتُ وتصغر.
     *
     * **فإن وقع الموضعُ — والمسارُ كلُّه إن وُجد — داخلَ صندوقِ حزمةٍ
     * في الفهرس قُرئت هي.** وإلّا `null`، **فيُقرأ الأساسُ كما كان.**
     *
     * # وما لا يُختار أبداً
     *
     * - **ما عنوانُه عنوانُ الأساس** — «سوريا كلّها» ليست أصغرَ من نفسها.
     * - **ما مداه في التقريب أضيقُ من الأساس** — وإلّا ابيضّت الخريطةُ
     *   عند تقريبٍ يعرفه الأساس.
     * - **ما قيل إنّه غيرُ صالح** (`usable` يردّ `false`) — **فالملفُّ
     *   الغائبُ عن الخادم (404) يعني خريطةً بيضاء**، والأساسُ لا يغيب.
     *
     * **والصندوقُ وحدَه يحكم هنا لا حدودُ المحافظة** — فالبلاطاتُ في
     * الحزمة لا تتجاوز صندوقَها، **ونقطةٌ داخلَ المحافظة خارجَ الصندوق
     * تقع على فراغ.**
     */
    fun forOnline(
        manifest: MapManifest,
        lat: Double?,
        lng: Double?,
        routeBbox: List<Double>? = null,
        usable: (MapRegion) -> Boolean = { true },
    ): MapRegion? {
        if (lat == null || lng == null) return null
        val base = manifest.base
        return manifest.regions
            .asSequence()
            .filter { it.artifact.url != base.url }
            .filter { it.artifact.minZoom <= base.minZoom && it.artifact.maxZoom >= base.maxZoom }
            .filter { contains(it.bbox, lat, lng) }
            .filter { r -> routeBbox == null || routeBbox.size != 4 || routeInside(r.bbox, routeBbox) }
            .filter(usable)
            .minByOrNull { it.areaDeg }
    }

    /**
     * **حكمُ الفحص على حزمةٍ في الخادم** — من رمزِ الردّ وأوّلِ بايتاته.
     *
     * - `true`: ردٌّ ناجحٌ يبدأ بتوقيع `PMTiles` — **تُقرأ.**
     * - `false`: 404 أو 410 أو ملفٌّ ليس أرشيفاً — **لا تُقرأ في هذه الجلسة.**
     * - `null`: غيرُ ذلك (انقطاعٌ، 5xx) — **لا حكم**، فيُقرأ الأساسُ ويُعاد
     *   الفحصُ مع الفهرس التالي. **فعطلٌ عابرٌ لا يحرم المدينةَ حزمتَها.**
     */
    fun verdictOf(httpCode: Int, head: ByteArray?): Boolean? = when {
        httpCode == 404 || httpCode == 410 -> false
        httpCode == 200 || httpCode == 206 ->
            head != null && head.size >= PMTILES_MAGIC.size &&
                PMTILES_MAGIC.indices.all { head[it] == PMTILES_MAGIC[it] }
        else -> null
    }

    /** **توقيعُ أرشيف PMTiles** — أوّلُ سبعِ بايتاتٍ في كلّ ملفّ. */
    val PMTILES_MAGIC: ByteArray = "PMTiles".toByteArray(Charsets.US_ASCII)

    /** **أيقع المسارُ كلُّه في الصندوق؟** — وإلّا خرج السائقُ إلى فراغ. */
    private fun routeInside(bbox: List<Double>, route: List<Double>): Boolean =
        bbox.size >= 4 &&
            route[0] >= bbox[0] && route[1] >= bbox[1] &&
            route[2] <= bbox[2] && route[3] <= bbox[3]

    /** **أداخل الحلقة؟** — عدُّ التقاطعات، والحلقةُ `[طول، عرض]`. */
    private fun inRing(ring: List<List<Double>>, lat: Double, lng: Double): Boolean {
        var inside = false
        var j = ring.size - 1
        for (i in ring.indices) {
            val xi = ring[i][0]; val yi = ring[i][1]
            val xj = ring[j][0]; val yj = ring[j][1]
            if ((yi > lat) != (yj > lat) && lng < (xj - xi) * (lat - yi) / (yj - yi) + xi) {
                inside = !inside
            }
            j = i
        }
        return inside
    }

    /**
     * **أفي هذا الإطار؟**
     *
     * **والإطارُ `[غرب، جنوب، شرق، شمال]`** كما يكتبه الفهرس —
     * **وترتيبٌ يُخمَّن يقلب الطولَ بالعرض فيضع الرقّةَ في المحيط.**
     */
    private fun contains(bbox: List<Double>, lat: Double, lng: Double): Boolean {
        if (bbox.size < 4) return false
        val (w, s, e, n) = listOf(bbox[0], bbox[1], bbox[2], bbox[3])
        return lng in w..e && lat in s..n
    }
}
