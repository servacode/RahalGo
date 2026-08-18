package com.rahalgo.map

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.geo.GeoApi
import com.rahalgo.shared.geo.Place
import com.rahalgo.ui.AppCore
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ منتقي النقطة — نقطةٌ واسمُها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا يُنادى المزوّدُ مع كلّ حركة
 *
 * **تحريكةٌ واحدةٌ بالإصبع تُطلق عشرين استقرارا** — **وعشرون نداءً على
 * حزمةِ من في الشارع.** فتُؤخَّر القراءةُ قليلاً، **وما وصل قبل الأخيرة
 * يُلغى.**
 *
 * # ونقطةٌ تُحفظ قبل اسمِها
 *
 * **الاسمُ قد لا يجيء** — مزوّدٌ نائمٌ أو حيٌّ بلا اسم. **والنقطةُ
 * صحيحةٌ على كلّ حال**: **ومن انتظر الاسمَ ليُثبّت لا يُثبّت أبدا في
 * أطراف المدينة.**
 */
class PickPointViewModel(app: Application) : AndroidViewModel(app) {

    private val geo = GeoApi(AppCore.get().api)

    /** **ما تحت الدبّوس الآن** — وفارغٌ يعني «لم تستقرّ بعد». */
    var point by mutableStateOf<LatLng?>(null)
        private set

    var label by mutableStateOf("")
        private set

    var reading by mutableStateOf(false)
        private set

    var results by mutableStateOf<List<Place>>(emptyList())
        private set

    /** **تقفز الخريطةُ إليها** — تقرؤها `MapCanvas` من هذه الحال. */
    var jumpTo by mutableStateOf<LatLng?>(null)
        private set

    private var reader: Job? = null
    private var typer: Job? = null

    /** **استقرّت الخريطةُ فيُقرأ ما تحتها.** */
    fun readAddress(at: LatLng) {
        point = at
        reader?.cancel()
        reader = viewModelScope.launch {
            delay(QUIET_MS)
            reading = true
            // **واسمٌ لا يجيء لا يمنع التثبيت** — النقطةُ محفوظةٌ سلفا.
            runCatching { geo.reverse(at.latitude, at.longitude) }
                .onSuccess { label = it.label }
                .onFailure { label = "" }
            reading = false
        }
    }

    /** **يبحث بعد أن يسكت** — كبحث السوق حرفا. */
    fun search(query: String) {
        typer?.cancel()
        val q = query.trim()
        if (q.length < MIN_CHARS) {
            results = emptyList()
            return
        }
        typer = viewModelScope.launch {
            delay(QUIET_MS)
            results = runCatching { geo.search(q) }.getOrDefault(emptyList())
        }
    }

    fun goTo(place: Place) {
        results = emptyList()
        label = place.label
        jumpTo = LatLng(place.lat, place.lng)
    }

    /**
     * **يطلب القفزَ إلى موضعه الحاليّ** — ويرفع رايةَ الانتظار.
     *
     * **والموضعُ يأتي من التطبيق لا من هنا**: وحدةُ الخرائط لا تعرف
     * جهازَ التموضع — **ولو عرفته لَحملته كلُّ شاشةٍ ترسم خريطة.**
     */
    fun wantHere() {
        wantingHere = true
    }

    /** **ينتظر موضعاً** — يُطفأ حين يجيء أو حين يُلغى. */
    var wantingHere by mutableStateOf(false)
        private set

    /** **يقفز إلى نقطةٍ صريحة** — الموضعُ الذي جاء من الجهاز. */
    fun jumpToPoint(lat: Double, lng: Double) {
        wantingHere = false
        jumpTo = LatLng(lat, lng)
    }

    fun jumped() {
        jumpTo = null
    }

    private companion object {
        const val MIN_CHARS = 2
        const val QUIET_MS = 400L
    }
}
