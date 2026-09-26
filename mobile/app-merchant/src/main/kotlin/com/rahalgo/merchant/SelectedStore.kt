package com.rahalgo.merchant

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.merchant.Store
import com.rahalgo.ui.AppCore

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المتجرُ المختار — لمالكٍ يملك أكثرَ من متجر** (B8، ٢٠٢٦-٠٩-٢٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك: «ادعم تعدّدَ المتاجر للمالك الواحد، ولا تفرض ١:١».)
 *
 * # لماذا حالٌ مشتركةٌ واحدة
 *
 * **كانت كلُّ شاشةٍ تأخذ `firstOrNull()`** — فمالكٌ له فرعان يرى الأوّلَ
 * في «طلباتي» وربّما الثاني في «متجري» لو اختلف الترتيب، **ويقبل طلبَ فرعٍ
 * ويعدّل قائمةَ آخر وهو لا يدري.** **فصار الاختيارُ واحداً** تقرؤه الشاشاتُ
 * كلُّها، **ونقرةٌ في المبدّل تُعيد بناءَها جميعاً** (`Refresh.bump`).
 *
 * # ويُحفَظ محلّيّاً
 *
 * **من اختار فرعاً ثمّ أغلق التطبيق يعود إليه** — لا إلى الأوّل. **والحفظُ
 * في الجهاز**: اختيارُ عرضٍ لا حالُ حساب، فلا يلزم الخادمَ.
 *
 * # والتوفيقُ عند كلّ جلب
 *
 * **متجرٌ مختارٌ اختفى** (بِيع أو أُوقفت ملكيّتُه) **يسقط إلى أوّل الموجود**
 * لا إلى شاشةٍ فارغة — **و`resolve` هو الحارس**: يردّ المختارَ إن بقي،
 * وإلّا الأوّلَ، ويُصحّح الاختيارَ المحفوظ.
 */
object SelectedStore {

    private const val KEY = "merchant_selected_store"

    /** **معرّفُ المتجر المختار** — حالٌ تُراقبها الشاشات. */
    var id by mutableStateOf(load())
        private set

    /** **يبدّل المختار ويحفظه** — والنبضةُ تُترك لمن ينادي (بعد التأكّد). */
    fun set(newId: String) {
        if (newId.isBlank() || newId == id) return
        id = newId
        persist(newId)
    }

    /**
     * **يردّ المتجرَ الذي تعمل عليه الشاشة** — المختارَ إن بقي في القائمة،
     * وإلّا الأوّلَ، **ويُصحّح الاختيارَ المحفوظ إن شاخ.** و`null` إن لا متجر.
     */
    fun resolve(stores: List<Store>): Store? {
        if (stores.isEmpty()) return null
        val chosen = stores.firstOrNull { it.id == id } ?: stores.first()
        if (chosen.id != id) {
            id = chosen.id
            persist(chosen.id)
        }
        return chosen
    }

    private fun prefs() = AppCore.get().app
        .getSharedPreferences("rahalgo", Context.MODE_PRIVATE)

    private fun load(): String =
        runCatching { prefs().getString(KEY, "").orEmpty() }.getOrDefault("")

    private fun persist(value: String) {
        runCatching { prefs().edit().putString(KEY, value).apply() }
    }

    /** **يُصفَّر في الفحص.** */
    fun resetForTest() {
        id = ""
        runCatching { prefs().edit().remove(KEY).apply() }
    }
}
