package com.rahalgo.map

import android.content.Context
import kotlinx.coroutines.withContext
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CancellationException
import java.net.URL
import android.util.Log
import com.rahalgo.map.data.MapArchiveLease
import com.rahalgo.map.data.MapConfig
import com.rahalgo.map.data.MapManifest
import com.rahalgo.map.data.MapManifestClient
import com.rahalgo.map.data.MapPackageStore
import com.rahalgo.map.data.MapPaths
import com.rahalgo.map.data.MapRuntime
import com.rahalgo.map.data.MapSourceResolver
import com.rahalgo.map.data.MapStyleBinding
import com.rahalgo.map.data.MapSwitchPolicy
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **مستودعُ النمط — واحدٌ لكلّ التطبيقات الثلاثة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢٢.)
 *
 * **أمرُ المالك نصّاً**: «أزل الانقسام الحالي: MapCanvas → server
 * raster style · TripMap → asset raster style. واجعلهما يستخدمان نفس
 * MapRuntime / StyleRepository مع اختلاف overlays فقط».
 *
 * # **ولماذا كان الانقسامُ عطباً**
 *
 * **شاشتان ترسمان المدينةَ نفسَها بنمطين مختلفين.** فما يراه الزبونُ
 * وهو يختار عنوانَه **ليس ما يراه السائقُ وهو يقصده.** والاسمُ الذي
 * قرأه أحدُهما قد لا يظهر للآخر، **وذلك تشويشٌ في مهمّةٍ واحدة.**
 *
 * **والأسوأُ أنّ إحداهما كانت من الخادم والأخرى من ملفٍّ في الحزمة** —
 * **فتصحيحُ واحدةٍ لا يصحّح الأخرى.**
 */
object MapStyleRepository {

    @Volatile
    private var runtime: MapRuntime? = null

    @Volatile
    private var store: MapPackageStore? = null

    @Volatile
    private var client: MapManifestClient? = null

    @Volatile
    private var manifest: MapManifest? = null

    /**
     * ══════════════════════════════════════════════════════════════
     * **حجزُ الأرشيف — وهذا حدُّ الفصل**
     * ══════════════════════════════════════════════════════════════
     *
     * (إغلاقُ ٦ب الوظيفيّ، البند ١٣.)
     *
     * **أمرُ المالك نصّاً**: «MapStyleRepository هي المكان الطبيعي
     * لربط: resolved source · style loading · active archive lease.
     * لكن لا تجعل MapPackageStore يعرف MapLibre APIs».
     *
     * **فالمخزنُ يعرف الملفّاتِ ولا يعرف العارض**، **والعارضُ يعرف
     * `StyleLoaded` ولا يعرف المسارات** — **والمستودعُ بينهما.**
     */
    private val lease = MapArchiveLease()

    fun lease(): MapArchiveLease = lease

    /** **ما تفتحه الخريطةُ الآن** — يُقرأ لا يُخمَّن (البند ٨). */
    fun activeArchive(): File? = lease.activeArchive()

    /**
     * **يُهيَّأ مرّةً عند إقلاع التطبيق.**
     *
     * **والأصلُ من الإعداد** (البند ٣) — لا `maps.rahalgo.com` في
     * شيفرة. **والنمطُ من المورد الخامّ** — نسخةُ ٦أ حرفاً بحرف،
     * **يحرسها `check-map-style-single.mjs`.**
     */
    fun init(context: Context, config: MapConfig) {
        if (runtime != null) return
        synchronized(this) {
            if (runtime != null) return
            val app = context.applicationContext
            val canonical = app.resources
                .openRawResource(R.raw.rahalgo_style)
                .use { it.readBytes().toString(Charsets.UTF_8) }

            val pkgStore = MapPackageStore(app.filesDir).also { it.prepare() }
            store = pkgStore
            client = MapManifestClient(pkgStore, MapPaths.manifestCacheFile(app.filesDir))
            manifest = client?.cached()
            runtime = MapRuntime(pkgStore, canonical, config)
        }
    }

    fun isReady(): Boolean = runtime != null

    fun store(): MapPackageStore =
        store ?: error("MapStyleRepository لم تُهيَّأ — نادِ init في إقلاع التطبيق")

    fun manifestClient(): MapManifestClient =
        client ?: error("MapStyleRepository لم تُهيَّأ")

    fun manifest(): MapManifest? = manifest

    /**
     * ══════════════════════════════════════════════════════════════════
     * **الفهرسُ يُجلب حين يُحتاج — لا حين يضغط أحدٌ زرّاً**
     * ══════════════════════════════════════════════════════════════════
     *
     * (إغلاقُ `TD-CUSTOMER-MAP-NO-MANIFEST`، قِيس ٢٠٢٦-٠٨-٢٢.)
     *
     * # **العطبُ الذي أصلحته هذه**
     *
     * **`init` تقرأ الفهرسَ من ذاكرةٍ محلّيّةٍ وحدَها** (`client.cached()`)
     * — **وعلى تثبيتٍ جديدٍ لا ذاكرةَ ولا فهرس.** فيعيد `bindOnline(null)`
     * الحالةَ `Unavailable("لا فهرسَ بعد")`، **و`MapCanvas` تخرج صامتةً
     * فتبقى الشاشةُ بيضاء.**
     *
     * **وتطبيقُ السائق نجا بالصدفة**: فيه عاملٌ يجلب الفهرسَ حين يضغط
     * السائقُ «نزّل خريطة الرقة». **والزبونُ لا زرَّ له ولا عامل** —
     * فخريطتُه بيضاءُ من أوّل تثبيت. (شكا المالكُ ٢٠٢٦-٠٨-٢٢:
     * «الخريطة لاختيار عنوان لا تعمل بتطبيق الزبون».)
     *
     * # **ولماذا هنا لا في كلّ تطبيق**
     *
     * **كلُّ من يرسم خريطةً يحتاج فهرساً** — فلو وُضع الجلبُ في تطبيقٍ
     * تكرّر في الرابع ونُسي في الخامس. **وهنا يُكتب مرّةً ويعمل للجميع.**
     *
     * **ولا تُجلب مرّتين**: من عنده فهرسٌ يخرج فوراً، **والقفلُ يمنع
     * نداءين متزامنين من شاشتين.**
     *
     * @return **أصار عندنا فهرس؟** — تقرؤها الشاشةُ لتقول للمستخدم.
     */
    suspend fun ensureManifest(): Boolean = withContext(Dispatchers.IO) {
        manifest?.let { return@withContext true }
        val base = runtime?.origin ?: return@withContext false
        val url = base.trimEnd('/') + "/" + MANIFEST_FILE
        fetchLock.withLock {
            // **ويُعاد الفحصُ داخل القفل** — فقد جلبها من سبقنا.
            manifest?.let { return@withLock true }
            try {
                val text = URL(url).openStream()
                    .use { it.readBytes().toString(Charsets.UTF_8) }
                acceptManifest(text)
            } catch (e: CancellationException) {
                // **والإلغاءُ يُمرَّر لا يُبتلع** — يمسكه `check-cancellation`.
                throw e
            } catch (e: Exception) {
                // **وآخرُ صالحٍ خيرٌ من لا شيء** (البند ٤٣).
                manifestClient().offline(e.message ?: "تعذّر جلبُ الفهرس")
                Log.w(TAG, "تعذّر جلبُ الفهرس من $url: ${e.message}")
            }
            manifest != null
        }
    }

    /** **اسمُ ملفّ الفهرس** — واحدٌ للجميع، لا يُكرَّر في كلّ تطبيق. */
    const val MANIFEST_FILE = "manifest.json"

    private val fetchLock = Mutex()
    private const val TAG = "RahalGo/map"

    fun acceptManifest(text: String): MapManifestClient.Result {
        val result = manifestClient().accept(text)
        when (result) {
            is MapManifestClient.Result.Fresh -> manifest = result.manifest
            is MapManifestClient.Result.Cached -> manifest = result.manifest
            is MapManifestClient.Result.None -> Unit
        }
        return result
    }

    fun start(desired: MapSwitchPolicy.Desired) {
        runtime?.start(desired)
    }

    /**
     * **يعطي النمطَ المربوطَ لهذا الاستعمال.**
     *
     * **والقرارُ كلُّه هنا** — لا `if (network)` في أيّ شاشة (البند
     * ١٥).
     */
    fun bind(
        purpose: MapSourceResolver.Purpose,
        online: Boolean,
        lat: Double? = null,
        lng: Double? = null,
        routeBbox: List<Double>? = null,
        navigating: Boolean = false,
        nowMs: Long = System.currentTimeMillis(),
    ): MapRuntime.Binding {
        val rt = runtime ?: return MapRuntime.Binding.Unavailable("الخريطةُ لم تُهيَّأ")
        val pkgStore = store ?: return MapRuntime.Binding.Unavailable("لا مخزنَ حزم")

        val installed = pkgStore.installedRegions()
        val resourcesReady = installed
            .map { it.resourcesVersion }
            .filter { it.isNotBlank() }
            .toSet()
            .filter { pkgStore.installedResources(it) != null }
            .toSet()

        val binding = rt.resolve(
            MapSourceResolver.Request(
                purpose = purpose,
                online = online,
                lat = lat,
                lng = lng,
                routeBbox = routeBbox,
                installed = installed,
                installedResourcesVersions = resourcesReady,
                manifest = manifest,
            ),
            navigating = navigating,
            nowMs = nowMs,
        )

        /**
         * **ويُعلَن ما سيُحمَّل قبل أن يُحمَّل** — البند ٩.
         *
         * **فالحذفُ يعرف أنّ انتقالاً جارٍ** ولو لم يُنجَز، **ولا
         * يُسحب أرشيفٌ من تحت تحميلٍ في الطريق.**
         */
        when (val d = (binding as? MapRuntime.Binding.Ready)?.decision) {
            is MapSourceResolver.Decision.OfflineRegion -> lease.beginSwitch(
                MapArchiveLease.Ref(d.region.regionId, d.region.dataVersion, d.region.archive),
            )
            is MapSourceResolver.Decision.Online -> lease.beginSwitchToOnline()
            else -> Unit
        }

        return binding
    }

    /**
     * **يُنادى من الشاشة عند نجاح `StyleLoaded`.**
     *
     * **وهنا وحدَه يصير المطلوبُ محمَّلاً** (البند ١٢)، **ويُحرَّر
     * القديم** (البند ٩).
     */
    fun onStyleLoaded(binding: MapRuntime.Binding) {
        val decision = (binding as? MapRuntime.Binding.Ready)?.decision
        when (decision) {
            is MapSourceResolver.Decision.OfflineRegion -> {
                val r = decision.region
                lease.styleLoaded(MapArchiveLease.Ref(r.regionId, r.dataVersion, r.archive))
                store?.setActive(
                    MapPackageStore.Active(r.regionId, r.dataVersion, r.resourcesVersion),
                )
            }
            is MapSourceResolver.Decision.Online -> {
                lease.styleLoaded(null)
                store?.setActive(
                    MapPackageStore.Active(null, null, manifest?.resourcesVersion),
                )
            }
            else -> Unit
        }
    }

    /**
     * **وإن سقط التحميل** — يبقى القديمُ محجوزاً وفعّالاً.
     *
     * **ولا يُكتب `active.json`** — فلم يتغيّر ما هو محمَّل.
     */
    fun onStyleFailed() {
        lease.switchFailed()
    }

    /**
     * **نسخُ الموارد المحميّةُ من الحذف** — البند ٧.
     *
     * **ثلاثٌ**: التي يقرؤها المحمَّل، والتي يطلبها المطلوبُ في
     * انتقالٍ جارٍ، **ونسخةُ الفهرس الحاليّ** — فهي التي سيُبنى عليها
     * الربطُ التالي.
     */
    fun protectedResourceVersions(): Set<String> {
        val pkgStore = store ?: return emptySet()
        val snap = lease.snapshot()
        val out = mutableSetOf<String>()
        manifest?.resourcesVersion?.let { out += it }
        for (ref in listOfNotNull(snap.loaded, snap.desired)) {
            pkgStore.installedRegion(ref.regionId, ref.dataVersion)
                ?.resourcesVersion
                ?.takeIf { it.isNotBlank() }
                ?.let { out += it }
        }
        return out
    }

    /** **الرصّاتُ التي يطلبها النمط** — تُقرأ مرّةً وتُستعمل في التركيب. */
    fun fontstacks(context: Context): List<String> {
        val canonical = context.applicationContext.resources
            .openRawResource(R.raw.rahalgo_style)
            .use { it.readBytes().toString(Charsets.UTF_8) }
        return MapStyleBinding.fontstacksOf(canonical)
    }

    /** **للاختبار وحدَه.** */
    fun resetForTest() {
        synchronized(this) {
            lease.reset()
            runtime = null
            store = null
            client = null
            manifest = null
        }
    }
}
