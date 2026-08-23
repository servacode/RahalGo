package com.rahalgo.map.data

/**
 * ══════════════════════════════════════════════════════════════════
 * **محلِّلُ المصدر — القرارُ في موضعٍ واحد**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ١٥ و١٦ و١٧.)
 *
 * **أمرُ المالك نصّاً**: «ولا تضع `if(network) …` داخل كل Map
 * composable».
 *
 * # **ولماذا القرارُ لا يُكرَّر**
 *
 * **ثلاثُ شاشاتٍ ترسم خرائط**: الزبونُ يلتقط نقطةً، والمندوبُ يلتقط،
 * **والسائقُ يلاحق مساراً.** ولو قرّرت كلُّ واحدةٍ بنفسها **لاختلفت
 * ثلاثتُها في الحالة نفسِها** — ولصار العطبُ يظهر في شاشةٍ ولا يظهر
 * في أختها، **وهو أصعبُ ما يُطارَد.**
 */
object MapSourceResolver {

    /** **ما يُطلب من الخريطة** — فليس كلُّ استعمالٍ سواء. */
    enum class Purpose {
        /** **التقاطُ نقطةٍ** — لحظيٌّ، والاتّصالُ متاحٌ غالباً. */
        PICK_POINT,

        /** **متابعةُ رحلة** — طويلٌ، وقد يمرّ بمناطقَ بلا تغطية. */
        TRIP,

        /** **الملاحة** — أطولُ، وانقطاعُ الخريطة فيه أثقل. */
        NAVIGATION,
    }

    data class Request(
        val purpose: Purpose,
        val online: Boolean,
        val lat: Double?,
        val lng: Double?,
        /** **حدودُ المسار الحاليّ** — البند ١٧. */
        val routeBbox: List<Double>? = null,
        val installed: List<MapPackageStore.Installed> = emptyList(),
        val installedResourcesVersions: Set<String> = emptySet(),
        val manifest: MapManifest? = null,
    )

    sealed interface Decision {
        /** **بلاطاتٌ من الشبكة.** */
        data class Online(val why: String) : Decision

        /** **بلاطاتٌ من حزمةٍ مركَّبة.** */
        data class OfflineRegion(
            val region: MapPackageStore.Installed,
            val coversRoute: Boolean,
            val why: String,
        ) : Decision

        /**
         * **ولا مصدرَ** — البند ٣٧.
         *
         * **لا انهيار، ولا عودةَ إلى راستر OSM.** الواجهةُ تبقى،
         * **وحالُ «بيانات الخريطة غير متاحة» تُعرض.**
         */
        data class Unavailable(val why: String) : Decision
    }

    /**
     * **يختار الحزمةَ التي تغطّي الموضع** — البند ١٦.
     *
     * **ولا تُختار حزمةٌ لا تغطّيه لمجرّد أنّها آخرُ ما رُكِّب** (البند
     * ١٧). **فسائقٌ في حلبَ وحزمةُ الرقّةِ مركَّبةٌ يرى فراغاً** —
     * وذلك أسوأُ من خريطةٍ لا تُحمَّل، **لأنّه يبدو خريطةً.**
     *
     * **وإن غطّت أكثرُ من واحدة**: الأصغرُ صندوقاً أوّلاً — **فالأخصُّ
     * أعلى تقريباً وأدقُّ تفصيلاً.** ثمّ الأحدثُ نسخةً عند التساوي.
     */
    fun bestCovering(
        installed: List<MapPackageStore.Installed>,
        lat: Double?,
        lng: Double?,
        resourcesReady: Set<String>,
    ): MapPackageStore.Installed? {
        if (lat == null || lng == null) return null
        return installed
            .filter { it.contains(lat, lng) }
            // **والحزمةُ بلا مواردَ متوافقةٍ ليست خياراً** (البند ١٣).
            .filter { it.resourcesVersion.isBlank() || it.resourcesVersion in resourcesReady }
            .minWithOrNull(
                compareBy<MapPackageStore.Installed> { it.areaDeg }
                    .thenByDescending { it.dataVersion },
            )
    }

    fun resolve(request: Request): Decision {
        val covering = bestCovering(
            request.installed,
            request.lat,
            request.lng,
            request.installedResourcesVersions,
        )

        val coversRoute = covering != null && routeCovered(covering, request.routeBbox)

        return when {
            // ── لا شبكةَ ولا حزمة ──────────────────────────────────
            !request.online && covering == null ->
                Decision.Unavailable(
                    if (request.installed.isEmpty()) {
                        "لا اتّصالَ ولا حزمةٌ مركَّبة"
                    } else {
                        "لا اتّصالَ ولا حزمةٌ تغطّي الموضع"
                    },
                )

            // ── لا شبكةَ وحزمةٌ تغطّي ──────────────────────────────
            !request.online && covering != null ->
                Decision.OfflineRegion(covering, coversRoute, "لا اتّصال — والحزمةُ تغطّي الموضع")

            /**
             * ── شبكةٌ متاحة ────────────────────────────────────────
             *
             * **والاتّصالُ يُفضَّل ما دام قائماً** — البند ١٧: «إذا
             * Network متوفرة، Online source أفضل لتغطية Route كاملة».
             *
             * **والحزمةُ صندوقٌ محدود**، والمسارُ قد يخرج عنه. **فلا
             * يُدَّعى أنّ الأساسَ كاملٌ خارجَ المنطقة.**
             */
            covering != null && !coversRoute ->
                Decision.Online("متّصل — والمسارُ يتجاوز صندوقَ الحزمة")

            else -> Decision.Online("متّصل")
        }
    }

    /**
     * **هل يقع المسارُ كلُّه داخلَ الحزمة؟**
     *
     * **ولا مسارَ يعني: لا سؤال** — فتُعدّ مغطّاةً.
     */
    fun routeCovered(region: MapPackageStore.Installed, routeBbox: List<Double>?): Boolean {
        val r = routeBbox ?: return true
        if (r.size != 4) return true
        val b = region.bbox
        if (b.size != 4) return false
        return r[0] >= b[0] && r[1] >= b[1] && r[2] <= b[2] && r[3] <= b[3]
    }
}
