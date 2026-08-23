package com.rahalgo.driver.trip

import android.content.Context
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.map.MapStyleRepository
import com.rahalgo.map.data.MapDownloader
import com.rahalgo.map.data.MapPackageStore
import com.rahalgo.map.data.MapArchiveLease
import com.rahalgo.map.data.MapPackageInstaller
import com.rahalgo.map.data.MapRegion

/**
 * ══════════════════════════════════════════════════════════════════
 * **خريطةُ المدينة دونَ اتّصال — محرّكٌ جديدٌ تحتَ الشاشة نفسِها**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٣٢ و٣٣ و٣٤ و٣٥.)
 *
 * # **ما أُزيل**
 *
 * **`OfflineManager` من MapLibre** — كانت تنزّل بلاطاتٍ نقطيّةً
 * `z10–z16` من `tile.openstreetmap.org` بسقفِ **١٥٠٠٠ بلاطة.**
 *
 * **وثلاثةُ عيوبٍ فيها**:
 *
 * **الأوّل** — سياسةُ OpenStreetMap تمنع التنزيلَ الكثيف. **فكنّا
 * نبني ميزةً على خدمةٍ لا تسمح بها**، وتوقُّفُها مسألةُ وقت.
 *
 * **الثاني** — السقفُ رقمٌ لا معنى له. **إن جاوزته المدينةُ نقصت
 * الخريطةُ صامتةً**، ولا يعرف السائقُ أيَّ حيٍّ سقط.
 *
 * **الثالث** — نقطيّةٌ لا متّجهة: **لا تدوير، ولا أسماءَ عربيّة،
 * وحجمٌ أكبرُ بمراتب.**
 *
 * # **وما صار**
 *
 * **أرشيفُ PMTiles واحدٌ للمنطقة** بُني بـPlanetiler على حدودها (٦أ).
 * **يُنزَّل ببصمةٍ ويُركَّب ذرّيّاً** (البند ٩)، **وموارُده مشتركةٌ مع
 * كلّ المناطق** (البند ١٤).
 *
 * **والشاشةُ لم تتغيّر** — أمرُ المالك: «حافظ على شكلها قدر الإمكان
 * وغيّر المحرك تحتها». **فـ`progress` و`ready` و`downloading` كما
 * كانت**، وزادت `state` و`error` لما تحتاجه الحالاتُ السبع.
 */
object OfflineMap {

    private const val TAG = "RahalGo/offline"

    /** **حالاتُ الحزمة** — البند ٣٣. */
    enum class State {
        NOT_INSTALLED,
        QUEUED,
        DOWNLOADING,
        VERIFYING,
        INSTALLED,
        UPDATE_AVAILABLE,
        FAILED,
    }

    /** **وما يُعرض عنها** — البند ٣٣. */
    data class Status(
        val regionId: String,
        val name: String,
        val version: String?,
        val state: State,
        val downloadedBytes: Long,
        val totalBytes: Long,
        val error: String? = null,
        /** **تصنيفُ الإخفاق** — يقرؤه المُجدوِل ليقرّر الإعادة (البند ١٥). */
        val failure: com.rahalgo.map.data.MapFailure? = null,
    ) {
        val progress: Int
            get() = when {
                state == State.INSTALLED -> 100
                totalBytes <= 0 -> 0
                else -> ((downloadedBytes * 100) / totalBytes).toInt().coerceIn(0, 99)
            }
    }

    /** **المنطقةُ الافتراضيّة** — الرقّة، وهي مدينةُ الإطلاق. */
    const val DEFAULT_REGION = "raqqa"

    var status by mutableStateOf(
        Status(DEFAULT_REGION, "الرقّة", null, State.NOT_INSTALLED, 0, 0),
    )
        private set

    /** **وما تقرؤه الشاشةُ القائمة** — لم يتغيّر شكلُه. */
    val progress: Int get() = if (status.state == State.NOT_INSTALLED) -1 else status.progress
    val ready: Boolean get() = status.state == State.INSTALLED

    /**
     * **أسقط التنزيلُ؟** — البند ٦ من قرار ٢٠٢٦-٠٨-٢٢.
     *
     * **وكانت الشاشةُ لا تسأل** — تقرأ `progress` و`downloading` وحدَهما،
     * **والساقطُ يبدو كمن لم يبدأ.**
     */
    val failed: Boolean get() = status.state == State.FAILED

    /** **وتصنيفُ السبب** — تختار به الشاشةُ عبارتَها لا نصَّ العطب. */
    val failure: com.rahalgo.map.data.MapFailure? get() = status.failure
    val downloading: Boolean
        get() = status.state == State.DOWNLOADING || status.state == State.VERIFYING

    @Volatile
    private var cancelled = false

    /**
     * **يقرأ ما هو مركَّبٌ فعلاً** — ولا يفترض.
     *
     * **ويُقارَن بالفهرس**: فإن كانت نسخةٌ أحدثُ **تُعلَن
     * `UPDATE_AVAILABLE` ولا تُحذف القائمة** (البند ٣٤).
     */
    fun check(context: Context, regionId: String = DEFAULT_REGION) {
        if (!MapStyleRepository.isReady()) {
            Log.w(TAG, "المستودعُ لم يُهيَّأ بعد")
            return
        }
        val store = MapStyleRepository.store()
        val manifest = MapStyleRepository.manifest()
        val installed = store.installedRegions().filter { it.regionId == regionId }
        val latest = installed.maxByOrNull { it.dataVersion }
        val remote = manifest?.region(regionId)

        status = when {
            latest == null -> Status(
                regionId,
                remote?.name ?: status.name,
                null,
                State.NOT_INSTALLED,
                0,
                remote?.artifact?.bytes ?: 0,
            )

            remote != null && remote.dataVersion > latest.dataVersion -> Status(
                regionId,
                remote.name,
                latest.dataVersion,
                State.UPDATE_AVAILABLE,
                latest.bytes,
                remote.artifact.bytes,
            )

            else -> Status(
                regionId,
                latest.name,
                latest.dataVersion,
                State.INSTALLED,
                latest.bytes,
                latest.bytes,
            )
        }
        Log.i(TAG, "حالُ $regionId: ${status.state} · نسخة ${status.version}")
    }

    /**
     * **ينزّل ويركّب** — والترتيبُ عقدٌ (البند ١٣).
     *
     *	ضمانُ الموارد  →  ضمانُ المنطقة  →  تنظيفُ القديم
     *
     * **ويُنادى من عملٍ يبقى بعد الشاشة** — البند ١٢. **فلو رُبط
     * بـ`viewModelScope` لضاع تنزيلُ مئةِ ميغابايت بإغلاق شاشة.**
     */
    /**
     * ══════════════════════════════════════════════════════════════════
     * **سطرُ إخفاقٍ واحدٌ لكلّ خطوة — بلا سرٍّ ولا عنوان**
     * ══════════════════════════════════════════════════════════════════
     *
     * (البند ٤ من قرار ٢٠٢٦-٠٨-٢٢.)
     *
     * **ويحمل ما يُبحَث به**: أيُّ خطوةٍ، وأيُّ سببٍ باسمه المستقرّ،
     * وأيُّ منطقةٍ ونسخةِ موارد، **وأعابرٌ هو أم دائم** — فمن قرأ
     * السطرَ عرف أيُعاد أم لا يُعاد.
     *
     * **ولا عنوانَ فيه ولا ترويسة**: العنوانُ قد يحمل رمزاً، **والسجلُّ
     * يُقرأ من جهازِ سائقٍ لا من جهازي.** و`detail` نصُّنا نحن —
     * يُكتب في `MapResourceFetcher` ولا يأتي من الشبكة.
     */
    private fun logFailure(
        step: String,
        regionId: String,
        resourcesVersion: String,
        r: com.rahalgo.map.data.MapPackageInstaller.Step.Failed,
    ) {
        Log.w(
            TAG,
            "إخفاقُ $step · السبب=${r.failure?.name ?: "غيرُ مصنَّف"}" +
                " · عابر=${r.failure?.transient ?: false}" +
                " · المنطقة=$regionId · نسخةُ الموارد=$resourcesVersion" +
                " · ${r.detail}",
        )
    }

    fun install(
        context: Context,
        installer: com.rahalgo.map.data.MapPackageInstaller,
        regionId: String = DEFAULT_REGION,
        /**
         * **سببُ سقوط الفهرس إن سقط** — يمرّره العامل.
         *
         * **وبلاه تخرج رسالةٌ عامّةٌ من انقطاعِ شبكة** — والسائقُ يصلح
         * الشبكةَ ولا يصلح عقداً.
         */
        manifestFailure: com.rahalgo.map.data.MapFailure? = null,
    ) {
        val manifest = MapStyleRepository.manifest()
        if (manifest == null) {
            // **وهذا الفرعُ كان صامتاً هو الآخر** — قِيس على الجهاز
            // ٢٠٢٦-٠٨-٢٢: قُطع المضيفُ فخرج التنزيلُ من هنا **بلا
            // سطرٍ في السجلّ**، فبدا كأنّ العاملَ لم يعمل.
            Log.w(
                TAG,
                "إخفاقُ الفهرس · السبب=${manifestFailure?.name ?: "غيرُ مصنَّف"}" +
                    " · عابر=${manifestFailure?.transient ?: false}" +
                    " · المنطقة=$regionId · لا فهرسَ مخزَّنٌ يُرجَع إليه",
            )
            status = status.copy(
                state = State.FAILED,
                error = "لا فهرسَ خرائطَ بعد",
                failure = manifestFailure,
            )
            return
        }
        val region = manifest.region(regionId)
        if (region == null) {
            Log.w(
                TAG,
                "إخفاقُ المنطقة · السبب=INVALID_CONTRACT · عابر=false" +
                    " · المنطقة=$regionId · ليست في فهرسِ ${manifest.dataVersion}",
            )
            status = status.copy(
                state = State.FAILED,
                error = "لا حزمةَ باسم $regionId",
                failure = com.rahalgo.map.data.MapFailure.INVALID_CONTRACT,
            )
            return
        }

        cancelled = false
        status = Status(
            regionId, region.name, region.dataVersion,
            State.QUEUED, 0, region.artifact.bytes,
        )

        val progress = MapDownloader.Progress { done, total ->
            if (status.state != State.DOWNLOADING) {
                status = status.copy(state = State.DOWNLOADING)
            }
            status = status.copy(downloadedBytes = done, totalBytes = total)
        }
        val cancellation = MapDownloader.Cancellation { cancelled }

        val fontstacks = MapStyleRepository.fontstacks(context)

        /**
         * **الموارُد أوّلاً** — إغلاقُ ٦ب الوظيفيّ، البند ١.
         *
         * **وهي تُجلب الآن لا تُتحقَّق وحدَها.** فجهازٌ جديدٌ يبدأ
         * بميغابايتٍ ونصفٍ من الحروف والأيقونات **قبل مئةِ ميغابايتٍ
         * من البلاطات** — **فلو سقطت لم يُهدر التنزيلُ الكبير.**
         */
        when (val r = installer.ensureResources(manifest, fontstacks, progress, cancellation)) {
            is MapPackageInstaller.Step.Failed -> {
                // ══════════════════════════════════════════════════════
                // **وسقوطُ الموارد يُسجَّل كما يُسجَّل سقوطُ المنطقة**
                // ══════════════════════════════════════════════════════
                //
                // (إغلاقُ `TD-MAP-SILENT-RESOURCE-FAIL`، قرارُ المالك
                //  ٢٠٢٦-٠٨-٢٢ البند ٤: «اجعل ensureResources وensureRegion
                //  متسقين في الرصد».)
                //
                // **كان هذا الفرعُ وحدَه بلا سطرِ سجلّ** — وأخوه أدناه
                // يسجّل. **فسقط التنزيلُ على الجهاز المرجعيّ ولم يُعرف
                // أين** (قِيس ٢٠٢٦-٠٨-٢٢): لا سطرَ في `logcat`، ولا
                // رسالةَ في الشاشة، **والزرُّ عاد إلى «نزّل الآن» كأنّ
                // شيئاً لم يقع.**
                //
                // **والسببُ يُطبع باسمه** (`MapFailure`) لا بنصّه وحدَه —
                // فالنصُّ يتبدّل والاسمُ يُبحَث عنه.
                logFailure("الموارد", regionId, manifest.resourcesVersion, r)
                status = status.copy(
                    state = State.FAILED,
                    error = "${r.what}: ${r.detail}",
                    failure = r.failure,
                )
                return
            }
            is MapPackageInstaller.Step.Cancelled -> {
                check(context, regionId)
                return
            }
            else -> Unit
        }

        when (
            val r = installer.ensureRegion(manifest, regionId, fontstacks, progress, cancellation)
        ) {
            is MapPackageInstaller.Step.Failed -> {
                // **والقديمُ الصالحُ يبقى** — فتُعاد قراءةُ الحال.
                logFailure("المنطقة", regionId, manifest.resourcesVersion, r)
                check(context, regionId)
                status = status.copy(error = "${r.what}: ${r.detail}", failure = r.failure)
                return
            }
            is MapPackageInstaller.Step.Cancelled -> {
                check(context, regionId)
                return
            }
            else -> Unit
        }

        status = status.copy(state = State.VERIFYING)

        // **والتنظيفُ بعد النجاح لا قبله** — البند ٣٤ من ٦ب.
        // **ويسأل الحجزَ** — البند ١٠ من الإغلاق الوظيفيّ.
        val lease = MapStyleRepository.lease()
        val removed = installer.cleanupOldVersions(regionId, region.dataVersion, lease)
        val removedRes = installer.cleanupOldResources(
            manifest.resourcesVersion,
            MapStyleRepository.protectedResourceVersions(),
        )
        if (removed + removedRes > 0) {
            Log.i(TAG, "نُظّفت $removed نسخةَ حزمةٍ و$removedRes نسخةَ موارد")
        }

        check(context, regionId)
    }

    fun cancel() {
        cancelled = true
    }

    /**
     * **الحذف — ولا يُسحب أرشيفٌ من تحت خريطةٍ تقرؤه** (البند ٣٥).
     *
     * **MapLibre تفتح الملفَّ وتقرأ منه عشوائيّاً** — فحذفُه أثناء
     * الرسم **يُنتج انهياراً أو خريطةً فارغة.**
     *
     * **فيُرفض الحذفُ إن كانت هي الفعّالةَ ولا بديل**، ويُترك القرارُ
     * للسائق. **ولا يُتظاهر بالنجاح.**
     */
    sealed interface DeleteResult {
        data object Deleted : DeleteResult
        data class Refused(val why: String) : DeleteResult
        /** **انتقالٌ جارٍ** — يُعاد الطلبُ بعد استقراره. */
        data class Wait(val why: String) : DeleteResult
    }

    /**
     * **الحذف — والاستعمالُ يُعرف لا يُستدلُّ عليه.**
     *
     * (إغلاقُ ٦ب الوظيفيّ، البندان ٨ و١٠.)
     *
     * **كانت تسأل: أمتّصلٌ؟ أثمّة بديل؟** — وكلاهما ظنّ. **وسائقٌ
     * متّصلٌ قد تكون خريطتُه ما زالت على الحزمة المحلّيّة**، فيُحذف
     * الملفُّ من تحتها وهي تقرأ منه.
     *
     * **والآن تُسأل الحجزُ**: ما المحمَّلُ؟ وما المطلوبُ؟ وأثمّة انتقال؟
     */
    fun delete(context: Context, regionId: String = DEFAULT_REGION): DeleteResult {
        val store = MapStyleRepository.store()
        val lease = MapStyleRepository.lease()
        val installed = store.installedRegions().filter { it.regionId == regionId }
        if (installed.isEmpty()) return DeleteResult.Deleted

        for (i in installed) {
            when (val v = lease.deletionVerdict(i.regionId, i.dataVersion)) {
                is MapArchiveLease.Verdict.Refused -> return DeleteResult.Refused(v.why)
                is MapArchiveLease.Verdict.Wait -> return DeleteResult.Wait(v.why)
                is MapArchiveLease.Verdict.Allowed -> Unit
            }
        }

        var ok = true
        for (i in installed) {
            if (!store.deleteRegionVersion(i.regionId, i.dataVersion)) ok = false
        }
        check(context, regionId)
        return if (ok) {
            DeleteResult.Deleted
        } else {
            DeleteResult.Refused("تعذّر حذفُ بعض الملفّات")
        }
    }

    /** **ما تفتحه الخريطةُ الآن** — من الحجز لا من ظنّ. */
    fun activeArchive(): java.io.File? = MapStyleRepository.activeArchive()

    /** **للاختبار.** */
    fun resetForTest() {
        status = Status(DEFAULT_REGION, "الرقّة", null, State.NOT_INSTALLED, 0, 0)
        cancelled = false
    }
}
