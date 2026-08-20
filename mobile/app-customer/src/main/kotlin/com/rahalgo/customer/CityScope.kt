package com.rahalgo.customer

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.shared.customer.BrowseScope
import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.City
import com.rahalgo.ui.LastPoint

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من أيّ مدينةٍ يتسوّق — يُعرف بلا سؤالٍ ويُبدَّل بضغطة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «يختار مدينةً من أوّل ما يفتح، أو بطريقةٍ
 *  احترافيّةٍ نحدّد عنوانَه تلقائيّا».)
 *
 * # ولا يُسأل من لا سؤالَ له
 *
 * **والمنصّةُ اليومَ مدينةٌ واحدة** — **وسؤالُ «أيَّ مدينةٍ تريد؟» عن
 * قائمةٍ فيها سطرٌ واحدٌ ليس اختياراً بل عقبة.** فإن كانت واحدةً
 * أُخذت بلا كلام.
 *
 * # وترتيبُ ما يُصدَّق
 *
 *     ما اختاره بيده        — **واختيارُه يسبق كلَّ حَدْس**
 *       ↓ عنوانُه الافتراضيّ  — حيث يستقبل طلباته فعلاً
 *         ↓ موضعُ جهازه       — إن كان الإذنُ مأخوذاً أصلاً
 *           ↓ المدينةُ الوحيدة إن لم يكن غيرُها
 *             ↓ ولا شيء      — **فيرى السوقَ كلَّه**
 *
 * **ولا يُطلب إذنُ موقعٍ من أجل هذا**: يُقرأ إن كان مأخوذاً، **وإذنٌ
 * يُطلب في أوّل شاشةٍ قبل أن يُعرف سببُه يُرفض** — ومن رفضه مرّتين أُغلق
 * البابُ في أندرويد.
 *
 * # ولا شيءَ تعني «أرِني كلَّ شيء»
 *
 * **من فتح التطبيقَ أوّلَ مرّةٍ يجب أن يرى سوقاً** — **وشاشةٌ فارغةٌ
 * تُقرأ عطباً فيُحذف التطبيق.**
 */
object CityScope {

    private const val FILE = "rahalgo_city"
    private const val KEY_ID = "id"
    private const val KEY_LAT = "lat"
    private const val KEY_LNG = "lng"
    private const val KEY_NAME = "name"

    /** **ما اختاره بيده** — يبقى بين الجلسات. */
    var chosen by mutableStateOf<City?>(null)
        private set

    fun load(context: Context) {
        val p = context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)
        val id = p.getString(KEY_ID, null) ?: return
        // **والإحداثيّ يُحفظ نصّاً لا `Float`** — `Float` يفقد منازلَ
        // تُزحزح النقطةَ مئاتِ الأمتار، **و`SharedPreferences` لا تحفظ
        // `Double` بلا حيلة.**
        val lat = p.getString(KEY_LAT, null)?.toDoubleOrNull() ?: return
        val lng = p.getString(KEY_LNG, null)?.toDoubleOrNull() ?: return
        chosen = City(id = id, name = p.getString(KEY_NAME, "").orEmpty(), lat = lat, lng = lng)
    }

    fun choose(context: Context, city: City?) {
        chosen = city
        val p = context.applicationContext
            .getSharedPreferences(FILE, Context.MODE_PRIVATE).edit()
        if (city == null) {
            p.clear()
        } else {
            p.putString(KEY_ID, city.id)
                .putString(KEY_LAT, city.lat.toString())
                .putString(KEY_LNG, city.lng.toString())
                .putString(KEY_NAME, city.name)
        }
        p.apply()
    }

    /**
     * **يحسب نقطةَ التصفّح من كلّ ما يُعرف** — ويردّ `null` حين لا
     * يُعرف شيء.
     */
    fun resolve(address: Address?, cities: List<City>): BrowseScope.Point? {
        chosen?.let { return BrowseScope.Point(it.lat, it.lng) }
        address?.takeIf { it.lat != 0.0 || it.lng != 0.0 }
            ?.let { return BrowseScope.Point(it.lat, it.lng) }
        LastPoint.value?.let { return BrowseScope.Point(it.lat, it.lng) }
        // **ومدينةٌ واحدةٌ لا تُسأل** — انظر أعلاه.
        cities.singleOrNull()?.let { return BrowseScope.Point(it.lat, it.lng) }
        return null
    }

    /** **اسمُ المدينة التي يتسوّق منها** — للعرض وحدَه، وفارغٌ إن لم تُعرف. */
    fun nameOf(cities: List<City>): String {
        chosen?.let { return it.name }
        val p = BrowseScope.point.value ?: return ""
        // **وأقربُ مدينةٍ تحويه** — القاعدةُ نفسُها التي في المحرّك
        // (`city_filter.go`)، **وقاعدتان تفترقان تجعلان الشاشةَ تقول
        // «الرقّة» والخادمَ يُرشِّح بدمشق.**
        return cities
            .filter { withinKm(p.lat, p.lng, it.lat, it.lng) <= it.radiusM / 1000.0 }
            .minByOrNull { withinKm(p.lat, p.lng, it.lat, it.lng) }
            ?.name.orEmpty()
    }

    /**
     * **المسافةُ بالكيلومترات** — تقريبٌ مستوٍ يكفي لمدنٍ متباعدة.
     *
     * **ولا يُستعمل هذا في قرارِ ترشيح**: القرارُ في المحرّك بـPostGIS،
     * **وهذا يُسمّي ما رشّحه هو.**
     */
    private fun withinKm(aLat: Double, aLng: Double, bLat: Double, bLng: Double): Double {
        val dLat = (aLat - bLat) * 111.0
        val dLng = (aLng - bLng) * 111.0 * Math.cos(Math.toRadians((aLat + bLat) / 2))
        return Math.sqrt(dLat * dLat + dLng * dLng)
    }
}

/**
 * **يُبقي نقطةَ التصفّح موافقةً لما يُعرف** — ويُعيد التحميلَ حين
 * تتغيّر.
 *
 * **والإعادةُ عند التغيّر وحدَه**: `BrowseScope.set` تردّ «أتغيّرت»،
 * **ونداءٌ يُعيد التحميلَ في كلّ دورانِ جهازٍ يستنزف شبكةً في الرقّة.**
 */
@Composable
fun CityGate(address: Address?, cities: List<City>, onChanged: () -> Unit) {
    val context = LocalContext.current
    var loaded by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        CityScope.load(context)
        loaded = true
    }
    val here = LastPoint.value
    LaunchedEffect(loaded, CityScope.chosen, address, cities, here) {
        if (!loaded) return@LaunchedEffect
        val p = CityScope.resolve(address, cities)
        if (BrowseScope.set(p?.lat, p?.lng)) onChanged()
    }
}
