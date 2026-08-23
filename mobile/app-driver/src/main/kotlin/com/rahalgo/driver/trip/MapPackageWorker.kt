package com.rahalgo.driver.trip

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.rahalgo.driver.BuildConfig
import com.rahalgo.map.data.MapManifestClient
import com.rahalgo.driver.data.Backend
import com.rahalgo.map.MapStyleRepository
import com.rahalgo.map.data.MapConfig
import com.rahalgo.map.data.MapDownloader
import com.rahalgo.map.data.MapFailure
import com.rahalgo.map.data.MapPackageInstaller
import com.rahalgo.map.data.MapResourceFetcher
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL

/**
 * ══════════════════════════════════════════════════════════════════
 * **عاملُ تنزيل الحزم — يبقى بعد الشاشة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١٢.)
 *
 * **أمرُ المالك نصّاً**: «لا أريد Coroutine مرتبطة بشاشة فقط إذا كان
 * إغلاق الشاشة سيضيع تنزيلًا كبيرًا… إذا لا يوجد mechanism مناسب،
 * استخدم أقل حل Android صحيح مثل WorkManager».
 *
 * # **وفُحص ما في المشروع أوّلاً**
 *
 * **`LocationService` خدمةٌ أماميّةٌ للموقع** — لا تصلح لهذا: عمرُها
 * عمرُ المناوبة، **والتنزيلُ قد يمتدّ بعدها.** ولا `WorkManager` في
 * المشروع. **فأُدخلت، وهي الأقلُّ الصحيحة.**
 *
 * # **وما يحمي عند موت العمليّة**
 *
 * **`WorkManager` تعيد جدولةَ العمل**، **و`.part` على القرص يُستأنف
 * منه** (البند ١١) — فالاثنان معاً يعبران قتلَ النظام.
 *
 * # **وطلبُ السائق ليس تحديثاً تلقائيّاً**
 *
 * (إغلاقُ ٦ب الوظيفيّ، البند ١٤.)
 *
 * **كان `UNMETERED` شرطاً لكلّ تنزيل.** وذلك **يمنع سائقاً طلب
 * الخريطةَ صراحةً وهو على بيانات هاتفه** — **فيُتّخذ القرارُ نيابةً
 * عنه في أمرٍ هو صاحبُه.**
 *
 * **فصارتا سياستين**:
 *
 *	طلبٌ صريحٌ من السائق  →  `CONNECTED`      · يبدأ الآن
 *	تحديثٌ تلقائيٌّ       →  `UNMETERED` + مساحةٌ كافية
 *
 * **والواجهةُ تُحذّر من الحجم لاحقاً** — ولا واجهةَ جديدةً الآن.
 */
class MapPackageWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result = withContext(Dispatchers.IO) {
        val regionId = inputData.getString(KEY_REGION) ?: OfflineMap.DEFAULT_REGION
        val ctx = applicationContext

        MapStyleRepository.init(ctx, mapConfig(ctx))

        // **والفهرسُ يُجلب أوّلاً** — فلا تُنزَّل نسخةٌ يعرف الخادمُ
        // أنّها شاخت. **وإن سقط الجلبُ يُستعمل آخرُ صالح** (البند ٤٣).
        val manifestFailure = refreshManifest(ctx)

        val store = MapStyleRepository.store()
        val http = UrlHttpSource()
        val downloader = MapDownloader(store, http)
        val installer = MapPackageInstaller(
            store,
            downloader,
            mapConfig(ctx),
            MapResourceFetcher(store, downloader, http, mapConfig(ctx)),
        )

        OfflineMap.install(ctx, installer, regionId, manifestFailure)

        when (OfflineMap.status.state) {
            OfflineMap.State.INSTALLED -> Result.success()
            OfflineMap.State.FAILED -> retryOrFail(OfflineMap.status.failure)
            else -> Result.success()
        }
    }

    /**
     * **الإعادةُ للعابر لا للدائم** — البند ١٥.
     *
     * **أمرُ المالك نصّاً**: «لا نعيد تنزيل نفس Artifact خمس مرات إذا
     * SHA المعلن نفسه لا يطابق الملف باستمرار».
     *
     * **فبصمةٌ لا تطابق لا تُصلحها خمسون محاولة**: الأثرُ في المنبع
     * غيرُ الذي يعده العقد. **والإعادةُ تنزّل مئةَ ميغابايتٍ خمسَ
     * مرّاتٍ من بيانات سائقٍ ثمّ تسقط.**
     *
     * **وخرقُ الأمان لا يُعاد أبداً** — فالإعادةُ محاولةٌ ثانية.
     */
    private fun retryOrFail(failure: MapFailure?): Result = when {
        failure == null -> Result.failure()
        !failure.transient -> Result.failure()
        runAttemptCount < MAX_ATTEMPTS -> Result.retry()
        else -> Result.failure()
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ويعيد تصنيفَ ما سقط — لا يبتلعه**
     * ══════════════════════════════════════════════════════════════════
     *
     * (إغلاقُ `TD-MAP-SILENT-RESOURCE-FAIL`، ٢٠٢٦-٠٨-٢٢.)
     *
     * **كان يبتلع السببَ ويمضي**: يُخزَّن آخرُ فهرسٍ صالحٍ إن وُجد،
     * **وإن لم يوجد مضى إلى `install` بفهرسٍ فارغ** — فتخرج من هناك
     * برسالةٍ عامّةٍ لا تعرف أشبكةٌ سقطت أم عقدٌ لا يُفهم.
     *
     * **وقِيس على `SM-A525F` ٢٠٢٦-٠٨-٢٢**: قُطع مضيفُ الخرائط فسقط
     * التنزيل، **وظهرت للسائق عبارةُ «حاول مرّة أخرى» العامّة** بدل
     * عبارةِ الاتّصال — **وهو يستطيع إصلاحَ الاتّصال ولا يستطيع
     * إصلاحَ عقد.**
     *
     * **و`IOException` شبكةٌ، وما سواه عقدٌ لا يُفهم** — وهو التقسيمُ
     * الذي يُغيّر ما يفعله السائق.
     */
    private fun refreshManifest(ctx: Context): MapFailure? {
        val url = "${mapConfig(ctx).baseUrl.trimEnd('/')}/$MANIFEST_PATH"
        try {
            val text = URL(url).openStream().use { it.readBytes().toString(Charsets.UTF_8) }
            return when (MapStyleRepository.acceptManifest(text)) {
                is MapManifestClient.Result.Fresh -> null
                is MapManifestClient.Result.Cached -> MapFailure.INVALID_CONTRACT
                is MapManifestClient.Result.None -> MapFailure.INVALID_CONTRACT
            }
        } catch (e: CancellationException) {
            /**
             * **والإلغاءُ يُمرَّر لا يُبتلع.**
             *
             * **`CancellationException` هي إشارةُ الإلغاء في
             * الكوروتين**، فالتقاطُها بـ`Exception` **يجعل العملَ
             * الملغى يبدو ساقطاً** — فيُعاد جدولتُه، والسائقُ يرى عطباً
             * حيث ألغى بنفسه. (أمسكه `check-cancellation.mjs`.)
             */
            throw e
        } catch (e: java.io.IOException) {
            MapStyleRepository.manifestClient().offline(e.message ?: "تعذّر جلبُ الفهرس")
            return MapFailure.NETWORK
        } catch (e: Exception) {
            MapStyleRepository.manifestClient().offline(e.message ?: "تعذّر جلبُ الفهرس")
            return MapFailure.INVALID_CONTRACT
        }
    }

    companion object {
        const val KEY_REGION = "regionId"
        const val MANIFEST_PATH = "manifest.json"
        const val WORK_NAME = "rahalgo-map-package"
        const val MAX_ATTEMPTS = 5

        /**
         * **إعدادُ الخرائط** — من `Backend` لا من ثابتٍ في شيفرة
         * (البند ٣).
         */
        fun mapConfig(context: Context): MapConfig =
            MapConfig(
                baseUrl = Backend.of(context).mapsBaseUrl,
                allowLoopbackHttp = BuildConfig.DEBUG,
            )

        /** **من طلب التنزيل** — البند ١٤. */
        enum class Trigger {
            /** **السائقُ ضغط الزرّ** — قرارُه، فلا تُشترط واي فاي. */
            USER,

            /** **تحديثٌ تلقائيّ** — لم يطلبه أحد، فيُنتظر واي فاي. */
            AUTOMATIC,
        }

        /**
         * **قيودُ الشبكة بحسب من طلب.**
         *
         * **ومساحةٌ كافيةٌ شرطٌ في الحالين** — فتنزيلٌ يملأ القرص
         * يُفقد بياناتِ تطبيقاتٍ أخرى.
         */
        fun constraints(trigger: Trigger): Constraints = Constraints.Builder()
            .setRequiredNetworkType(
                when (trigger) {
                    Trigger.USER -> NetworkType.CONNECTED
                    Trigger.AUTOMATIC -> NetworkType.UNMETERED
                },
            )
            .setRequiresStorageNotLow(true)
            .build()

        fun enqueue(
            context: Context,
            regionId: String = OfflineMap.DEFAULT_REGION,
            trigger: Trigger = Trigger.USER,
        ) {
            val request = OneTimeWorkRequestBuilder<MapPackageWorker>()
                .setInputData(Data.Builder().putString(KEY_REGION, regionId).build())
                .setConstraints(constraints(trigger))
                .build()
            WorkManager.getInstance(context.applicationContext).enqueueUniqueWork(
                WORK_NAME,
                // ══════════════════════════════════════════════════════
                // **ولا يُبدأ تنزيلان — وضغطةُ السائق تسبق الآليّ**
                // ══════════════════════════════════════════════════════
                //
                // **كانت `KEEP` للاثنين**، فضغطةُ السائق تُهمَل ما دامت
                // إعادةٌ آليّةٌ معلّقةً في تراجعِها. **وقِيس على
                // `SM-A525F` ٢٠٢٦-٠٨-٢٢**: ظهرت رسالةُ الإخفاق وزرُّ
                // «حاول مرّة أخرى»، **فضُغط فلم يقع شيء** — والسائقُ
                // يرى زرّاً لا يفعل.
                //
                // **وهذه ليست حلقةَ إعادة**: واحدةٌ لكلّ ضغطة، **تحلّ
                // محلَّ المعلَّقة ولا تُضاف إليها.** والآليُّ يبقى
                // `KEEP` كما كان — **فلا يُقاطع تنزيلاً جارياً بتحديثٍ
                // لم يطلبه أحد.**
                if (trigger == Trigger.USER) ExistingWorkPolicy.REPLACE
                else ExistingWorkPolicy.KEEP,
                request,
            )
        }

        fun cancel(context: Context) {
            OfflineMap.cancel()
            WorkManager.getInstance(context.applicationContext).cancelUniqueWork(WORK_NAME)
        }
    }
}

/**
 * **منفذُ الشبكة الحقيقيّ** — `HttpURLConnection` بلا مكتبة.
 *
 * **والاستئنافُ عقدٌ يُقرأ من الردّ لا يُفترض** (البند ١١): حالةُ
 * `206` و`Content-Range` **هما وحدَهما ما يسمح باللصق.**
 */
class UrlHttpSource : com.rahalgo.map.data.HttpSource {

    override fun open(url: String, rangeStart: Long?): com.rahalgo.map.data.HttpSource.Response {
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            connectTimeout = 20_000
            readTimeout = 60_000
            requestMethod = "GET"
            if (rangeStart != null && rangeStart > 0) {
                setRequestProperty("Range", "bytes=$rangeStart-")
            }
        }

        val status = connection.responseCode
        if (status !in 200..299) {
            connection.disconnect()
            throw com.rahalgo.map.data.HttpSource.HttpException("الخادمُ ردَّ $status")
        }

        /**
         * **بدايةُ المدى تُقرأ من `Content-Range` لا من الطلب.**
         *
         * `Content-Range: bytes 1024-9999/10000` — **والرقمُ الأوّلُ
         * هو ما بدأ الخادمُ منه فعلاً**، وقد يخالف ما طُلب.
         */
        val contentRange = connection.getHeaderField("Content-Range")
        val start = contentRange
            ?.substringAfter("bytes ", "")
            ?.substringBefore('-', "")
            ?.toLongOrNull()
            ?: 0L

        val length = connection.getHeaderFieldLong("Content-Length", -1L)

        return object : com.rahalgo.map.data.HttpSource.Response {
            override val status: Int = status
            override val rangeStart: Long = start
            override val remainingBytes: Long = length
            override val body: InputStream = connection.inputStream
            override fun close() {
                try {
                    connection.inputStream.close()
                } catch (e: Exception) {
                    // **الإغلاقُ لا يُسقط شيئاً** — الاتّصالُ انتهى.
                }
                connection.disconnect()
            }
        }
    }
}
