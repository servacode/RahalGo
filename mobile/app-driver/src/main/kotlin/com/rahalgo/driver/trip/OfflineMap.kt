package com.rahalgo.driver.trip

import android.content.Context
import com.rahalgo.driver.R
import android.util.Log
import com.rahalgo.driver.data.Backend
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.offline.OfflineManager
import org.maplibre.android.offline.OfflineRegion
import org.maplibre.android.offline.OfflineRegionError
import org.maplibre.android.offline.OfflineRegionStatus
import org.maplibre.android.offline.OfflineTilePyramidRegionDefinition

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطة المدينة في الجهاز — تُنزَّل مرّة وتبقى**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند الرابع في قائمة المالك ٢٠٢٦-٠٨-١٢: «الخريطة تشتغل بلا إنترنت …
 *  حتّى لو انقطع الإنترنت بالطريق تضلّ تشوف وين رايح».)
 *
 * # لماذا هي شرط لا تحسين
 *
 * **الانقطاع في الرقّة أمر يوميّ**: حيّ بلا تغطية، أو حزمة انتهت، أو
 * شبكة ثقيلة وقت الذروة. **وسائق أمام خريطة بيضاء لا يعرف أين يذهب** —
 * ومعه بضاعة ونقد وزبون ينتظر.
 *
 * # وما الذي يعمل بلا شبكة وما الذي لا يعمل
 *
 * **تعمل**: الخريطة والشوارع والدبابيس وموقعك (القمر لا يحتاج إنترنت).
 * **لا تعمل**: تحديث حال الطلب، والملاحة الصوتيّة في التطبيق الخارجيّ.
 *
 * # والحدود صندوق حول المدينة
 *
 * **من التقريب ١٠ إلى ١٦**: العاشر يُري المدينة كلَّها، **والسادس عشر
 * يُري أسماء الشوارع** — وهو ما يحتاجه من يبحث عن باب. **وما فوقه
 * يضاعف الحجم بلا أن يزيد معرفة.**
 */
object OfflineMap {

    /** **صندوق يحيط بالرقّة** — ومن خرج منه رجع إلى الشبكة. */
    private val RAQQA_BOUNDS = LatLngBounds.Builder()
        .include(LatLng(36.03, 39.12))
        .include(LatLng(35.88, 38.92))
        .build()

    private const val MIN_ZOOM = 10.0
    private const val MAX_ZOOM = 16.0
    private const val TAG = "RahalGo/offline"

    /** حال التنزيل — **تقرؤه الشاشة.** */
    var progress by mutableStateOf(-1)
        private set

    var ready by mutableStateOf(false)
        private set

    /**
     * **أيجري التنزيل الآن؟**
     *
     * **وغيرُ التقدّم**: منطقةٌ وقفت على ٣٠٪ تقدّمُها ٣٠ **ولا شيء
     * يجري** — ومن عرض شريطاً متحرّكا عليها **ترك صاحبَه ينتظر ما لا
     * يأتي**، ولا زرّ يستأنف به.
     */
    var downloading by mutableStateOf(false)
        private set

    /** المنطقة القائمة — **تُستأنف ولا تُنشأ ثانيةً.** */
    private var existing: OfflineRegion? = null

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يفحص إن كانت المنطقة مكتملة — لا إن كانت موجودة**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وقع ٢٠٢٦-٠٨-١٢**: كُتب الفحصُ «هل توجد منطقة؟» — **فاختفت بطاقة
     * التنزيل بعد محاولةٍ فاشلة** أنشأت منطقةً فارغة، **وظنّ صاحبه أنّ
     * الخريطة معه** حتّى ينقطع الإنترنت فيجدها بيضاء.
     *
     * **فالوجود ليس اكتمالا**: يُسأل كلُّ منطقةٍ عن حالها.
     */
    fun check(context: Context) {
        manager(context).listOfflineRegions(
            object : OfflineManager.ListOfflineRegionsCallback {
                override fun onList(offlineRegions: Array<OfflineRegion>?) {
                    val region = offlineRegions?.firstOrNull()
                    if (region == null) {
                        ready = false
                        return
                    }
                    existing = region
                    region.getStatus(
                        object : OfflineRegion.OfflineRegionStatusCallback {
                            override fun onStatus(status: OfflineRegionStatus?) {
                                if (status == null) return
                                ready = status.isComplete
                                if (!status.isComplete && status.requiredResourceCount > 0) {
                                    progress = (
                                        status.completedResourceCount * 100 /
                                            status.requiredResourceCount
                                        ).toInt().coerceIn(0, 99)
                                }
                            }

                            override fun onError(error: String?) {
                                Log.w(TAG, "تعذّرت قراءة حال المنطقة: $error")
                            }
                        },
                    )
                }

                override fun onError(error: String) {
                    Log.w(TAG, "تعذّرت قراءة المناطق: $error")
                }
            },
        )
    }

    /**
     * **ينزّل خريطة المدينة — أو يستأنف ما وقف.**
     *
     * **ولا تُنشأ منطقةٌ ثانيةٌ فوق الأولى**: نسختان من المدينة في
     * الجهاز **تضاعفان الحجم بلا فائدة**، وتجعلان الفحص لا يعرف أيّهما
     * الحقّ.
     */
    fun download(context: Context) {
        if (downloading) return
        downloading = true

        // ══════════════════════════════════════════════════════════════
        // **ومنطقةٌ بأسلوبٍ قديمٍ تُحذف لا تُستأنف**
        // ══════════════════════════════════════════════════════════════
        //
        // **تعريفُ المنطقة يحفظ عنوانَ الأسلوب يوم أُنشئت** — فاستئنافُها
        // يعيد طلبَ العنوان القديم. **ووقع ٢٠٢٦-٠٨-١٢**: أُنشئت بـ
        // `asset://` (وهو ما لا يقرؤه المنزِّل)، **فبقي الاستئناف يسقط
        // بالخطأ نفسِه** بعد أن صُلّح العنوان.
        val region = existing
        if (region != null) {
            val sameStyle = region.definition.styleURL == Backend.of(context).styleUrl
            Log.i(TAG, "منطقة قائمة — أسلوبها ${region.definition.styleURL} · مطابق=$sameStyle")
            if (sameStyle) {
                region.setObserver(observer(region))
                region.setDownloadState(OfflineRegion.STATE_ACTIVE)
                return
            }
            region.delete(
                object : OfflineRegion.OfflineRegionDeleteCallback {
                    override fun onDelete() {
                        Log.i(TAG, "حُذفت المنطقة القديمة")
                        existing = null
                        downloading = false
                        download(context)
                    }

                    override fun onError(error: String) {
                        Log.w(TAG, "تعذّر حذف المنطقة القديمة: $error")
                        downloading = false
                    }
                },
            )
            return
        }

        val definition = OfflineTilePyramidRegionDefinition(
            Backend.of(context).styleUrl,
            RAQQA_BOUNDS,
            MIN_ZOOM,
            MAX_ZOOM,
            context.resources.displayMetrics.density,
        )
        manager(context).createOfflineRegion(
            definition,
            // **وبيانات المنطقة لا تُترك فارغة** — MapLibre يشترطها،
            // **ومنها يُعرف ما هذه المنطقة** يوم تصير مناطق.
            context.getString(R.string.map_region).toByteArray(),
            object : OfflineManager.CreateOfflineRegionCallback {
                override fun onCreate(region: OfflineRegion) {
                    Log.i(TAG, "أُنشئت المنطقة — يبدأ التنزيل")
                    existing = region
                    region.setObserver(observer(region))
                    region.setDownloadState(OfflineRegion.STATE_ACTIVE)
                }

                override fun onError(error: String) {
                    Log.w(TAG, "تعذّر إنشاء المنطقة: $error")
                    downloading = false
                }
            },
        )
    }

    private fun observer(region: OfflineRegion) = object : OfflineRegion.OfflineRegionObserver {
        override fun onStatusChanged(status: OfflineRegionStatus) {
            Log.i(
                TAG,
                "حال: ${status.completedResourceCount}/${status.requiredResourceCount} " +
                    "مكتمل=${status.isComplete}",
            )
            val total = status.requiredResourceCount
            progress = if (total > 0) {
                (status.completedResourceCount * 100 / total).toInt().coerceIn(0, 100)
            } else {
                0
            }
            if (status.isComplete) {
                ready = true
                progress = 100
                downloading = false
                // **وتُترك خاملة بعد الاكتمال** — نشِطة تعني «تابع
                // التنزيل»، **وهي تستيقظ مع كلّ شبكة** بلا سبب.
                region.setDownloadState(OfflineRegion.STATE_INACTIVE)
                Log.i(TAG, "اكتملت خريطة المدينة")
            }
        }

        override fun onError(error: OfflineRegionError) {
            Log.w(TAG, "خطأ تنزيل: ${error.reason} — ${error.message}")
            // **وخطأٌ واحدٌ لا يوقف التنزيل** — المكتبة تعيد المحاولة،
            // **لكنّ الشاشة تعود إلى الزرّ** فلا يبقى شريطٌ يدور أبدا.
            downloading = false
        }

        // **واسم الدالّة ورثته المكتبة عن أصلها** (Mapbox) — يُكتب كما
        // هو لا كما نتمنّى، **والمترجم وحدَه يكشف الفرق.**
        override fun mapboxTileCountLimitExceeded(limit: Long) {
            // **وسقف MapLibre ستّة آلاف بلاطة** — والصندوق أصغر منه
            // بكثير، **فبلوغه يعني أنّ الحدود اتّسعت بلا انتباه.**
            Log.w(TAG, "تجاوز سقف البلاطات: $limit")
        }
    }

    private fun manager(context: Context): OfflineManager {
        // **والمكتبة تُهيَّأ أوّلا** — وإلّا سقط التطبيق بـ
        // `MapLibreConfigurationException` (وقع على جهاز حقيقيّ
        // ٢٠٢٦-٠٨-١٢ حين أُخفي تبويب الرحلة).
        ensureMapLibre(context)
        return OfflineManager.getInstance(context.applicationContext).also {
            // ══════════════════════════════════════════════════════════
            // **ورفعُ السقف — والرقّة تجاوزته**
            // ══════════════════════════════════════════════════════════
            //
            // **سقف MapLibre الافتراضي ستّة آلاف مورد**، **والرقّة من
            // التقريب ١٠ إلى ١٦ تحتاج ٦٨٨٠** (قيس على الجهاز
            // ٢٠٢٦-٠٨-١٢). **فيقف التنزيل قبل أن يكتمل** ويبقى صاحبه
            // يظنّ الخريطة معه.
            //
            // **والسقف حارسٌ لا حدّ تقنيّ** — غرضه ألّا ينزّل تطبيقٌ
            // نصفَ الكوكب بلا انتباه. **ومدينةٌ واحدةٌ ليست ذاك.**
            it.setOfflineMapboxTileCountLimit(MAX_TILES)
        }
    }

    /** **ضعف ما تحتاجه المدينة** — يتّسع لتوسيع الحدود لاحقا. */
    private const val MAX_TILES = 15_000L
}
