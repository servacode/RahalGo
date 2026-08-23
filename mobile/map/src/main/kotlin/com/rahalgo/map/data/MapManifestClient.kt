package com.rahalgo.map.data

import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **عميلُ الفهرس — وآخرُ صالحٍ يبقى**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٤٣.)
 *
 * **أمرُ المالك نصّاً**: «احتفظ بآخر Manifest صالح محليًا. إذا fetch
 * جديد فشل: استخدم Last Known Good».
 *
 * # **ولماذا لا يُخزَّن ما لم يُفحص**
 *
 * **المخبأُ يُقرأ عند كلّ إقلاع.** فلو خُزّن نصٌّ لم يُفحص **لصار
 * فهرسٌ معطوبٌ يُقرأ إلى الأبد** — والشبكةُ التي جلبته لا تُسأل ثانيةً
 * ما دام المخبأُ موجوداً.
 *
 * **فالفحصُ قبل الخزن، والفحصُ بعد القراءة.** مرّتان — **لأنّ ملفَّ
 * القرص يمكن أن يفسد بعد كتابته.**
 */
class MapManifestClient(
    private val store: MapPackageStore,
    private val cacheFile: File,
) {

    sealed interface Result {
        /** **من الشبكة، مفحوصٌ ومخزَّن.** */
        data class Fresh(val manifest: MapManifest) : Result

        /** **من المخبأ — والشبكةُ لم تُسعف.** */
        data class Cached(val manifest: MapManifest, val why: String) : Result

        /** **ولا فهرسَ أصلاً.** */
        data class None(val why: String) : Result
    }

    /**
     * **يقرأ آخرَ فهرسٍ صالحٍ من القرص.**
     *
     * **ويُفحص كما يُفحص القادمُ من الشبكة** — فملفٌّ فسد أو نسخةُ
     * مخطّطٍ لم تعد مفهومةً **لا يُقرأ لمجرّد أنّه محلّيّ.**
     */
    fun cached(): MapManifest? {
        if (!cacheFile.isFile) return null
        return try {
            MapManifest.parse(cacheFile.readText())
        } catch (e: Exception) {
            null
        }
    }

    /**
     * **يقبل نصّاً من الشبكة.**
     *
     * **ولا يُخزَّن إلّا بعد أن يُفحص** — والفحصُ هنا هو `parse`
     * نفسُها: **ترمي عند أوّل حقلٍ لا يصحّ.**
     */
    fun accept(text: String): Result = try {
        val manifest = MapManifest.parse(text)
        store.prepare()
        cacheFile.parentFile?.mkdirs()
        val tmp = File(cacheFile.parentFile, "${cacheFile.name}.tmp")
        tmp.writeText(text)
        cacheFile.delete()
        if (!tmp.renameTo(cacheFile)) tmp.delete()
        Result.Fresh(manifest)
    } catch (e: Exception) {
        /**
         * **وفهرسٌ جديدٌ معطوبٌ لا يمحو صالحاً قديماً.**
         *
         * **البند ٤٣**: «لا تنزل Version مجهولة». فالقديمُ الصالحُ
         * أفضلُ من جديدٍ لا يُفهم.
         */
        val fallback = cached()
        if (fallback != null) {
            Result.Cached(fallback, "الفهرسُ الجديد مرفوض: ${e.message}")
        } else {
            Result.None(e.message ?: "فهرسٌ غيرُ مقبول")
        }
    }

    /** **وحين تسقط الشبكةُ قبل أن تردّ شيئاً.** */
    fun offline(why: String): Result =
        cached()?.let { Result.Cached(it, why) } ?: Result.None(why)
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **مركّبُ الحزم — الترتيبُ عقدٌ**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البند ١٣.)
 *
 *	ضمانُ الموارد  →  ضمانُ المنطقة  →  تفعيلُ الربط
 *
 * **ولا تُستعمل منطقةٌ مع نمطٍ أو حروفٍ غيرِ متوافقة** — أمرُ المالك.
 *
 * # **ولماذا الموارُد أوّلاً**
 *
 * **الحزمةُ بلاطاتٌ وحدَها** (البند ١٤): لا نمطَ فيها ولا حروف. فلو
 * رُكِّبت وحدَها **لفُتحت خريطةٌ بلا اسمِ شارعٍ واحد** — وهو العطبُ
 * الذي وقع في ٦أ باسم مجلّدِ الرصّة، **وقد كلّف ما كلّف.**
 */
class MapPackageInstaller(
    private val store: MapPackageStore,
    private val downloader: MapDownloader,
    private val config: MapConfig,
    /**
     * **جالبُ الموارد** — إغلاقُ ٦ب الوظيفيّ، البند ١.
     *
     * **كانت `ensureResources` تتحقّق ولا تجلب**، **فجهازٌ جديدٌ لا
     * يستطيع تركيبَ حزمةٍ أصلاً.** وهو دَينُ `TD-RESOURCE-FETCH`.
     */
    private val resources: MapResourceFetcher,
) {

    sealed interface Step {
        data object Ok : Step
        data class Failed(val what: String, val detail: String, val failure: MapFailure) : Step
        data object Cancelled : Step
    }

    /**
     * **يضمن مواردَ كاملةً** — جلباً وتحقُّقاً وتفعيلاً ذرّيّاً.
     *
     * **ولا يُعلَّم بشيءٍ هنا** — الجالبُ يكتب العلامةَ داخلَ المؤقّت
     * **قبل النقلة**، **فتصير النسخةُ كاملةً لحظةَ صيرورتها.**
     */
    fun ensureResources(
        manifest: MapManifest,
        fontstacks: List<String>,
        progress: MapDownloader.Progress? = null,
        cancellation: MapDownloader.Cancellation? = null,
    ): Step = when (val out = resources.ensure(manifest, fontstacks, progress, cancellation)) {
        is MapResourceFetcher.Outcome.Installed -> Step.Ok
        is MapResourceFetcher.Outcome.AlreadyInstalled -> Step.Ok
        is MapResourceFetcher.Outcome.Cancelled -> Step.Cancelled
        is MapResourceFetcher.Outcome.Failed ->
            Step.Failed("الموارد", out.detail, out.reason)
    }

    /**
     * **ويركّب حزمةَ منطقة.**
     *
     * **والقديمةُ تبقى حتّى ينجح الجديد** (البند ٣٤) — النسخةُ في
     * المسار **فلا تتصادمان.**
     */
    fun ensureRegion(
        manifest: MapManifest,
        regionId: String,
        requiredFontstacks: List<String>,
        progress: MapDownloader.Progress? = null,
        cancellation: MapDownloader.Cancellation? = null,
    ): Step {
        val region = manifest.region(regionId)
            ?: return Step.Failed(
                "المنطقة",
                "لا منطقةَ بالمعرِّف $regionId في الفهرس",
                MapFailure.INVALID_CONTRACT,
            )

        /**
         * **ولا تُعدُّ حزمةٌ مركَّبةً ومواردُها ليست كذلك** — البند ١.
         *
         * **أمرُ المالك نصّاً**: «لا يجوز اعتبار Region
         * INSTALLED/READY إذا كانت الموارد المطلوبة غير مثبتة».
         */
        if (store.installedRegion(region.id, region.dataVersion) != null) {
            return if (store.resourcesComplete(manifest.resourcesVersion, requiredFontstacks)) {
                Step.Ok
            } else {
                Step.Failed(
                    "المنطقة",
                    "الأرشيفُ مركَّبٌ ومواردُ ${manifest.resourcesVersion} ناقصة",
                    MapFailure.INCOMPLETE,
                )
            }
        }

        val url = try {
            MapUrls.resolveRemote(config.baseUrl, region.artifact.url, config.allowLoopbackHttp)
        } catch (e: Exception) {
            return Step.Failed("المنطقة", e.message ?: "عنوانٌ مرفوض", MapFailure.SECURITY)
        }

        val target = store.regionArchiveTarget(region.id, region.dataVersion)
        return when (
            val out = downloader.fetch(
                url = url,
                expectedBytes = region.artifact.bytes,
                expectedSha256 = region.artifact.sha256,
                target = target,
                progress = progress,
                cancellation = cancellation,
            )
        ) {
            is MapDownloader.Outcome.Installed -> {
                store.markRegionInstalled(region, manifest.resourcesVersion)
                Step.Ok
            }
            is MapDownloader.Outcome.Cancelled -> Step.Cancelled
            is MapDownloader.Outcome.Failed ->
                Step.Failed("المنطقة", out.detail, MapFailure.of(out.reason))
        }
    }

    /**
     * **وينظّف نسخاً قديمةً بعد نجاح بديلتها** — البند ٣٤.
     *
     * **ولا يُحذف ما هو فعّال** — البند ٣٥. **والقرارُ يُمرَّر إليه**
     * لأنّ المركّبَ لا يعرف ما تفتحه MapLibre.
     */
    fun cleanupOldVersions(
        regionId: String,
        keepVersion: String,
        lease: MapArchiveLease,
    ): Int {
        var n = 0
        for (installed in store.installedRegions()) {
            if (installed.regionId != regionId) continue
            if (installed.dataVersion == keepVersion) continue
            // **ولا يُحذف محجوزٌ ولو كان قديماً** — البند ١٠.
            if (lease.deletionVerdict(installed.regionId, installed.dataVersion)
                    !is MapArchiveLease.Verdict.Allowed
            ) {
                continue
            }
            if (store.deleteRegionVersion(installed.regionId, installed.dataVersion)) n += 1
        }
        return n
    }

    /**
     * **وتنظيفُ نسخِ الموارد** — البند ٧.
     *
     * **والمحميّاتُ تُمرَّر** — فالمركّبُ لا يعرف ما يقرؤه العارض.
     */
    fun cleanupOldResources(keepVersion: String, protectedVersions: Set<String>): Int {
        var n = 0
        for (v in store.installedResourceVersions()) {
            if (v == keepVersion) continue
            if (store.deleteResourceVersion(v, protectedVersions + keepVersion)) n += 1
        }
        return n
    }
}
