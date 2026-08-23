package com.rahalgo.map.data

import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **زمنُ التشغيل — الفهرسُ يصير نمطاً مربوطاً**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١.)
 *
 *	الفهرس
 *	   ↓
 *	MapRuntime
 *	 ┌──┴───┐
 *	أونلاين  حزمةٌ مركَّبة
 *	 └──┬───┘
 *	النمطُ القانونيّ
 *	   ↓
 *	MapLibre
 *
 * # **ولا عنوانَ في منطق الواجهة** (البند ٣)
 *
 * `maps.rahalgo.com` **ليست في شيفرةٍ.** تأتي من الإعداد، **والفهرسُ
 * يعطي المساراتِ النسبيّة.** فتبديلُ المزوّد إعدادٌ لا إصدار.
 */
class MapRuntime(
    private val store: MapPackageStore,
    private val canonicalStyle: String,
    private val config: MapConfig,
    private val policy: MapSwitchPolicy = MapSwitchPolicy(),
) {

    /** **أصلُ الآثار** — يقرؤه المستودعُ ليجلب فهرسَه. */
    val origin: String get() = config.baseUrl

    /** **ما ينتهي إليه القرار** — نمطٌ جاهزٌ أو سببُ غيابه. */
    sealed interface Binding {
        data class Ready(
            val bound: MapStyleBinding.Bound,
            val decision: MapSourceResolver.Decision,
        ) : Binding

        data class Unavailable(val why: String) : Binding
    }

    /**
     * **آخرُ ربطٍ صالح** — البند ٢.
     *
     * **أمرُ المالك نصّاً**: «الـfallback الحقيقي: last known valid
     * vector configuration/resources — وليس العودة إلى OSM public
     * raster».
     *
     * **فإن سقط كلُّ شيءٍ نُعيد ما كان يعمل**، **ولا نستدعي راستراً
     * عامّاً بحال.**
     */
    var lastGood: MapStyleBinding.Bound? = null
        private set

    fun state(): MapSwitchPolicy.State = policy.state()

    /**
     * **يحلّ المصدرَ ثمّ يربط.**
     *
     * **والسياسةُ بين الحلّ والربط** — فالمحلِّلُ يقول ما ينبغي،
     * **والسياسةُ تقول متى يُنفَّذ** (البند ١٨).
     */
    fun resolve(
        request: MapSourceResolver.Request,
        navigating: Boolean,
        nowMs: Long,
    ): Binding {
        val decision = MapSourceResolver.resolve(request)

        val desired = when (decision) {
            is MapSourceResolver.Decision.Online -> MapSwitchPolicy.Desired.ONLINE
            is MapSourceResolver.Decision.OfflineRegion -> MapSwitchPolicy.Desired.OFFLINE
            is MapSourceResolver.Decision.Unavailable -> MapSwitchPolicy.Desired.UNAVAILABLE
        }

        val out = policy.update(MapSwitchPolicy.Input(desired, navigating, nowMs))

        /**
         * **وأثناء المهلة يبقى القائمُ قائماً.**
         *
         * **لا نمطَ جديدٌ يُحمَّل** — وهو أصلُ منع الوميض.
         */
        if (out.active != desired) {
            return lastGood?.let { Binding.Ready(it, decision) }
                ?: bindFor(decision, request)
        }

        return bindFor(decision, request)
    }

    /** **ويُبتدأ من حالٍ معلوم** — عند إنشاء الخريطة. */
    fun start(desired: MapSwitchPolicy.Desired) = policy.start(desired)

    private fun bindFor(
        decision: MapSourceResolver.Decision,
        request: MapSourceResolver.Request,
    ): Binding = when (decision) {
        is MapSourceResolver.Decision.Online -> bindOnline(request.manifest)
        is MapSourceResolver.Decision.OfflineRegion -> bindOffline(decision.region)
        is MapSourceResolver.Decision.Unavailable ->
            lastGood?.let { Binding.Ready(it, decision) }
                ?: Binding.Unavailable(decision.why)
    }

    fun bindOnline(manifest: MapManifest?): Binding {
        val m = manifest ?: return lastGood?.let {
            Binding.Ready(it, MapSourceResolver.Decision.Online("آخرُ ربطٍ صالح"))
        } ?: Binding.Unavailable("لا فهرسَ بعد")

        return try {
            val tiles = MapUrls.resolveRemote(config.baseUrl, m.base.url, config.allowLoopbackHttp)
            val resources = MapUrls.resolveRemote(config.baseUrl, m.resources.url, config.allowLoopbackHttp)
            val bound = MapStyleBinding.online(
                canonical = canonicalStyle,
                tileUrl = tiles,
                resourcesBaseUrl = resources,
                attribution = m.attribution,
                allowLoopbackHttp = config.allowLoopbackHttp,
            )
            lastGood = bound
            Binding.Ready(bound, MapSourceResolver.Decision.Online("متّصل"))
        } catch (e: Exception) {
            lastGood?.let {
                Binding.Ready(it, MapSourceResolver.Decision.Online("آخرُ ربطٍ صالح"))
            } ?: Binding.Unavailable(e.message ?: "تعذّر ربطُ الأونلاين")
        }
    }

    /**
     * **ولا تُربط حزمةٌ بمواردَ ناقصة** — البندان ١٣ و٢٥.
     *
     * **يُفحص كلُّ ملفّ حرفٍ وأيقونةٍ قبل التبديل** — فلا «404 محلّيٌّ
     * صامت» يظهر بعد أن يصير السائقُ في الطريق.
     */
    fun bindOffline(region: MapPackageStore.Installed): Binding {
        val version = region.resourcesVersion.ifBlank { config.fallbackResourcesVersion }
            ?: return Binding.Unavailable("حزمةُ ${region.regionId} بلا نسخةِ موارد")

        val resources = store.installedResources(version)
            ?: return Binding.Unavailable("مواردُ $version غيرُ مركَّبة")

        val wanted = MapStyleBinding.fontstacksOf(canonicalStyle)
        val missingGlyphs = store.missingGlyphs(version, wanted)
        if (missingGlyphs.isNotEmpty()) {
            return Binding.Unavailable(
                "حروفٌ ناقصةٌ في $version: ${missingGlyphs.take(3).joinToString(" · ")}" +
                    if (missingGlyphs.size > 3) " (+${missingGlyphs.size - 3})" else "",
            )
        }

        val missingSprites = store.missingSprites(version)
        if (missingSprites.isNotEmpty()) {
            return Binding.Unavailable("أيقوناتٌ ناقصة: ${missingSprites.joinToString(" · ")}")
        }

        if (!region.archive.isFile) {
            return Binding.Unavailable("أرشيفُ ${region.regionId} غيرُ موجود")
        }

        return try {
            val bound = MapStyleBinding.offline(
                canonical = canonicalStyle,
                archive = region.archive,
                resourcesDir = resources.dir,
                attribution = config.attribution,
            )
            lastGood = bound
            Binding.Ready(
                bound,
                MapSourceResolver.Decision.OfflineRegion(region, true, "حزمةٌ مركَّبة"),
            )
        } catch (e: Exception) {
            Binding.Unavailable(e.message ?: "تعذّر ربطُ الحزمة")
        }
    }
}

/**
 * **إعدادُ الخريطة — ولا مضيفَ في شيفرة** (البندان ٣ و٩ من ٦أ).
 *
 * **أمرُ المالك**: «لا hardcode `maps.rahalgo.com` داخل
 * Navigation/UI logic».
 */
data class MapConfig(
    val baseUrl: String,
    /** **الإسنادُ حين لا فهرسَ بعد** — البند ٢٧، ولا خريطةَ بلا إسناد. */
    val attribution: String = DEFAULT_ATTRIBUTION,
    val fallbackResourcesVersion: String? = null,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **ثقبُ القبول — للمضيف المحلّيّ وحدَه، وبطلبٍ صريح**
     * ══════════════════════════════════════════════════════════════════
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٢، بند ٤: «Release: HTTPS ONLY · Debug:
     *  HTTPS أو HTTP فقط إذا host محلي معتمد».)
     *
     * **وافتراضُه `false`** — فلا يُفتح بالسهو. **ومن أراده كتبه**،
     * وتطبيقُ السائق وحدَه يكتبه وفي بناءِ التصحيح وحدَه.
     *
     * **ولا يكفي أنّ البناءَ تجريبيّ**: العنوانُ نفسُه يجب أن يكون
     * محلّيّاً — **فـ`http://example.com` مرفوضٌ في التصحيح كما في
     * الإصدار.** والفرقُ بين «تجريبيٌّ فليمرّ كلُّ شيء» و«تجريبيٌّ
     * فليمرّ الحلقيُّ وحدَه» هو الفرقُ بين ثقبٍ وباب.
     */
    val allowLoopbackHttp: Boolean = false,
) {
    init {
        require(baseUrl.startsWith("https://") || (allowLoopbackHttp && isLoopbackHttp(baseUrl))) {
            "أصلُ الخرائط يجب أن يكون https: $baseUrl"
        }
    }

    companion object {
        /**
         * **أهو `http` إلى هذا الجهاز نفسِه؟**
         *
         * **والمضيفُ يُقتطع بين `//` وأوّلِ `/` أو `:`** — فلا يخدع
         * `http://127.0.0.1.attacker.com` ولا `http://x/127.0.0.1`.
         */
        fun isLoopbackHttp(url: String): Boolean {
            if (!url.startsWith("http://")) return false
            val authority = url.removePrefix("http://").substringBefore('/')
            // **والعنوانُ السادسُ بين قوسين وفيه نقطتان أصلاً** — فمن
            // قطع عند أوّل `:` أخرج `[` من `[::1]:8791`. (أسقطه
            // الحارسُ عند أوّل تشغيل، ٢٠٢٦-٠٨-٢٢.)
            val host = if (authority.startsWith("[")) {
                authority.substringBefore(']', "") + "]"
            } else {
                authority.substringBefore(':')
            }.lowercase()
            return host == "127.0.0.1" || host == "localhost" || host == "[::1]"
        }

        /**
         * **نصُّ الإسناد** — شرطُ ترخيصِ OpenStreetMap وOpenMapTiles.
         *
         * **ويُستبدل بما في الفهرس متى وصل** — فالمصدرُ يُعلنه بنفسه.
         */
        const val DEFAULT_ATTRIBUTION =
            "© مساهمو OpenStreetMap · OpenMapTiles"
    }
}
