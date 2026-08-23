package com.rahalgo.map.data

import java.io.File
import java.io.IOException

/**
 * ══════════════════════════════════════════════════════════════════
 * **جالبُ الموارد — نسخةٌ كاملةٌ أو لا نسخة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ ٦ب الوظيفيّ، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ١ إلى ٦.)
 *
 * **أمرُ المالك نصّاً**: «أنشئ abstraction واحدة مسؤولة عن جلب: style
 * · glyphs · sprites · resource metadata/license. ولا تجعل OfflineMap
 * أو MapStyleRepository تنزّل ملفات منفردة بنفسها».
 *
 * # **الدَّينُ الذي يُغلق هنا**
 *
 * `ensureResources` **كانت تتحقّق ولا تجلب.** فجهازٌ جديدٌ لا يملك
 * نسخةَ الموارد **لا يستطيع تركيبَ حزمةِ منطقةٍ أصلاً** — يُنزّل مئةَ
 * ميغابايتٍ من البلاطات ثمّ لا يُفعّلها، **لأنّ اثنَي عشرَ ملفَّ حرفٍ
 * وزنُها ميغابايتٌ ونصفٌ غيرُ موجودة.**
 *
 * # **ولماذا مجلّدٌ مؤقّتٌ لا كتابةٌ في موضعها**
 *
 * (البند ٣.)
 *
 * **الأرشيفُ ملفٌّ واحدٌ فنقلتُه ذرّيّة.** **والموارُد ثمانيةَ عشرَ
 * ملفّاً**، فلو كُتبت في `resources/{v}/` مباشرةً **لظهرت النسخةُ
 * مركَّبةً وهي نصفُها** — **ولا شيءَ في السلوك يفرّق بين نصفٍ وكلّ
 * حتّى تُفتح الخريطةُ فتظهر مربّعات.**
 *
 * **فالكتابةُ في `.tmp-{v}/` ثمّ يُفحص العقدُ كلُّه ثمّ يُعاد تسميةُ
 * المجلّد.** **وإعادةُ تسمية المجلّد في نظام الملفّات نفسِه ذرّيّة**
 * كإعادة تسمية ملفّ.
 *
 * # **ولا استئنافَ لكلّ حرفٍ على حدة**
 *
 * (البند ٦: «لا تصنع Resume framework معقدًا لكل glyph PBF بلا
 *  قيمة… اختر الأبسط الصحيح».)
 *
 * **أكبرُ ملفٍّ ربعُ ميغابايت.** فاستئنافُه يوفّر ثوانيَ ويكلّف حالةً
 * على القرص. **والملفُّ يُنزَّل كاملاً ويُتحقَّق**، **والمنقطعُ
 * يُعاد** — وذلك أبسطُ وأصحّ.
 *
 * **لكنّ المجموعَ يُستأنف بمعنى آخر**: **ما نُزّل وطابق بصمتَه في
 * المؤقّت لا يُعاد** — فمحاولةٌ ثانيةٌ تكمل ولا تبدأ.
 */
class MapResourceFetcher(
    private val store: MapPackageStore,
    private val downloader: MapDownloader,
    private val http: HttpSource,
    private val config: MapConfig,
) {

    sealed interface Outcome {
        data class Installed(val contract: MapResourceContract, val files: Int) : Outcome
        data object AlreadyInstalled : Outcome
        data class Failed(val reason: MapFailure, val detail: String) : Outcome
        data object Cancelled : Outcome
    }

    /**
     * **يضمن نسخةَ مواردَ كاملةً مفعَّلة.**
     *
     * **والسلسلةُ عقدٌ** (البند ١):
     *
     *	عقدُ الموارد  →  تنزيلٌ إلى `.tmp-{v}`  →  بصمةُ كلّ ملفّ
     *	→  فحصُ العقد كاملاً  →  تفعيلٌ ذرّيّ
     *
     * **وأيُّ سقوطٍ يترك النسخةَ القديمةَ كما هي** (البند ٤).
     */
    fun ensure(
        manifest: MapManifest,
        requiredFontstacks: List<String>,
        progress: MapDownloader.Progress? = null,
        cancellation: MapDownloader.Cancellation? = null,
    ): Outcome {
        val version = manifest.resourcesVersion

        // ── ١ · أمركَّبةٌ سليمةٌ أصلاً؟ ─────────────────────────────
        if (store.resourcesComplete(version, requiredFontstacks)) return Outcome.AlreadyInstalled

        store.prepare()

        // ── ٢ · عقدُ الموارد ───────────────────────────────────────
        val baseUrl = try {
            MapUrls.resolveRemote(config.baseUrl, manifest.resources.url, config.allowLoopbackHttp)
        } catch (e: Exception) {
            return Outcome.Failed(MapFailure.SECURITY, e.message ?: "عنوانُ مواردَ مرفوض")
        }

        val contract = try {
            val text = readText(MapUrls.resolveRemote(baseUrl, CONTRACT_FILE, config.allowLoopbackHttp))
            MapResourceContract.parse(text, version)
        } catch (e: HttpSource.HttpException) {
            return Outcome.Failed(MapFailure.NETWORK, e.message ?: "تعذّر جلبُ عقد الموارد")
        } catch (e: IOException) {
            return Outcome.Failed(MapFailure.NETWORK, e.message ?: "تعذّر جلبُ عقد الموارد")
        } catch (e: IllegalArgumentException) {
            // **عقدٌ لا يُفهم لا يُعاد جلبُه خمسَ مرّات** (البند ١٥).
            return Outcome.Failed(MapFailure.INVALID_CONTRACT, e.message ?: "عقدُ مواردَ مرفوض")
        } catch (e: Exception) {
            return Outcome.Failed(MapFailure.INVALID_CONTRACT, e.message ?: "عقدُ مواردَ مرفوض")
        }

        // **وما يطلبه النمطُ يجب أن يكون في العقد** (البند ١٨).
        val missingStacks = requiredFontstacks.filter { it !in contract.fontstacks }
        if (missingStacks.isNotEmpty()) {
            return Outcome.Failed(
                MapFailure.INVALID_CONTRACT,
                "العقدُ لا يحوي رصّاتٍ يطلبها النمط: ${missingStacks.joinToString(" · ")}",
            )
        }

        // ── ٣ · المساحة ────────────────────────────────────────────
        val need = contract.downloadBytes * 2 + SAFETY_MARGIN_BYTES
        if (store.usableBytes() < need) {
            return Outcome.Failed(
                MapFailure.NO_SPACE,
                "الموارُد ${contract.downloadBytes} بايت والمتاح ${store.usableBytes()}",
            )
        }

        // ── ٤ · التنزيلُ إلى مجلّدٍ مؤقّت ──────────────────────────
        val staging = store.resourcesStagingDir(version)
        var done = 0L
        for (f in contract.files) {
            if (cancellation?.isCancelled() == true) return Outcome.Cancelled

            val target = try {
                store.resourceFileIn(staging, f.path)
            } catch (e: IllegalArgumentException) {
                store.discardStaging(version)
                return Outcome.Failed(MapFailure.SECURITY, e.message ?: "مسارٌ مرفوض")
            }

            /**
             * **وما نُزّل وطابق لا يُعاد** — فمحاولةٌ ثانيةٌ تكمل.
             *
             * **والمطابقةُ بالبصمة لا بالحجم** — ملفٌّ بحجمه الصحيح
             * ومحتوىً خاطئٍ يمرّ بالحجم ولا يمرّ بالبصمة.
             */
            if (target.isFile && target.length() == f.bytes &&
                MapDownloader.sha256(target).equals(f.sha256, ignoreCase = true)
            ) {
                done += f.bytes
                progress?.onBytes(done, contract.downloadBytes)
                continue
            }

            val url = try {
                MapUrls.resolveRemote(baseUrl, f.path, config.allowLoopbackHttp)
            } catch (e: Exception) {
                store.discardStaging(version)
                return Outcome.Failed(MapFailure.SECURITY, e.message ?: "عنوانٌ مرفوض")
            }

            when (
                val out = downloader.fetch(
                    url = url,
                    expectedBytes = f.bytes,
                    expectedSha256 = f.sha256,
                    target = target,
                    cancellation = cancellation,
                )
            ) {
                is MapDownloader.Outcome.Installed -> {
                    done += f.bytes
                    progress?.onBytes(done, contract.downloadBytes)
                }
                is MapDownloader.Outcome.Cancelled -> return Outcome.Cancelled
                is MapDownloader.Outcome.Failed -> {
                    store.discardStaging(version)
                    return Outcome.Failed(MapFailure.of(out.reason), "${f.path}: ${out.detail}")
                }
            }
        }

        // ── ٥ · فحصُ العقد كاملاً قبل التفعيل ──────────────────────
        // **البند ٤: «لا Partial Resources».**
        val problems = store.validateResourceSet(staging, requiredFontstacks)
        if (problems.isNotEmpty()) {
            store.discardStaging(version)
            return Outcome.Failed(
                MapFailure.INCOMPLETE,
                problems.take(4).joinToString(" · ") +
                    if (problems.size > 4) " (+${problems.size - 4})" else "",
            )
        }

        // ── ٦ · التفعيلُ الذرّيّ ───────────────────────────────────
        return try {
            store.activateResources(version, contract)
            Outcome.Installed(contract, contract.files.size)
        } catch (e: IOException) {
            store.discardStaging(version)
            Outcome.Failed(MapFailure.WRITE_FAILED, e.message ?: "تعذّر التفعيل")
        }
    }

    private fun readText(url: String): String =
        http.open(url, null).use { res ->
            if (res.status !in 200..299) {
                throw HttpSource.HttpException("الخادمُ ردَّ ${res.status}")
            }
            res.body.readBytes().toString(Charsets.UTF_8)
        }

    companion object {
        const val CONTRACT_FILE = "resources.json"

        /** **الموارُد صغيرةٌ** — فالهامشُ أصغرُ منه للأرشيف. */
        const val SAFETY_MARGIN_BYTES = 32L * 1024 * 1024
    }
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **تصنيفُ الإخفاق — والإعادةُ ليست لكلّ سقوط**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البند ١٥.)
 *
 * **أمرُ المالك نصّاً**: «لا نعيد تنزيل نفس Artifact خمس مرات إذا SHA
 * المعلن نفسه لا يطابق الملف باستمرار».
 *
 * # **والفرقُ ليس تجميليّاً**
 *
 * **شبكةٌ سقطت تُصلحها محاولةٌ ثانية.** **وبصمةٌ لا تطابق لا تُصلحها
 * خمسون**: الأثرُ في المنبع غيرُ الذي يعده العقد، **والإعادةُ تنزّل
 * ثلاثَ مئةِ ميغابايتٍ خمسَ مرّاتٍ من بياناتِ سائقٍ ثمّ تسقط.**
 *
 * **وخرقُ الأمان لا يُعاد أبداً** — فالإعادةُ محاولةُ استغلالٍ ثانية.
 */
enum class MapFailure {
    /** **عابرٌ** — يُعاد بالمهمّة نفسِها. */
    NETWORK,
    TIMEOUT,

    /** **دائمٌ لهذه المهمّة** — لا تُعاد. */
    NO_SPACE,
    CHECKSUM_MISMATCH,
    SIZE_MISMATCH,
    INVALID_CONTRACT,
    INCOMPATIBLE_SCHEMA,
    SECURITY,
    INCOMPLETE,
    WRITE_FAILED,
    ;

    /**
     * **أيُعاد بهذه المهمّة نفسِها؟**
     *
     * # **ونقصُ المساحة ليس عابراً** — قرارُ المالك ٢٠٢٦-٠٨-٢١
     *
     * **كان مصنَّفاً عابراً على أنّ السائقَ قد يحذف صوراً.** وذلك
     * **خلطٌ بين «قد يتغيّر يوماً» و«يتغيّر بالإعادة»** — والإعادةُ
     * هي المسألة.
     *
     * **و`WorkManager` لا تخلق مساحةً بالانتظار.** فخمسُ محاولاتٍ
     * بتراجعٍ أُسّيٍّ **تعطي النتيجةَ نفسَها خمسَ مرّات**، وتُبقي
     * المهمّةَ معلّقةً في الطابور بلا فائدة.
     *
     * **و`storageNotLow` لا تكفي حارساً**: النظامُ يرفعها عند عتبةٍ
     * عامّة، **وحاجتُنا أدقّ** — حجمُ الأثر زائدَ المؤقّت زائدَ
     * الهامش. **فقد تكون الشرطُ مستوفاةً والمساحةُ لا تكفينا.**
     *
     * **فالمهمّةُ تسقط، ويُعرض نقصُ التخزين.** والسائقُ يفرّغ ثمّ
     * يطلب من جديد، **والتحديثُ التلقائيُّ يحاول في دورةٍ قادمة —
     * لا في حلقةِ إعادةٍ لهذه المهمّة.**
     */
    val transient: Boolean
        get() = this == NETWORK || this == TIMEOUT

    companion object {
        fun of(reason: MapDownloader.Reason): MapFailure = when (reason) {
            MapDownloader.Reason.NETWORK -> NETWORK
            MapDownloader.Reason.NO_SPACE -> NO_SPACE
            MapDownloader.Reason.CHECKSUM_MISMATCH -> CHECKSUM_MISMATCH
            MapDownloader.Reason.SIZE_MISMATCH -> SIZE_MISMATCH
            MapDownloader.Reason.WRITE_FAILED -> WRITE_FAILED
            MapDownloader.Reason.BAD_URL -> SECURITY
        }
    }
}
