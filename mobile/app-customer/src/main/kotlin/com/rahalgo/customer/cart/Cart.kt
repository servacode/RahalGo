package com.rahalgo.customer.cart

import android.content.Context
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.model.Item
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السلّة — تبقى بعد إغلاق التطبيق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصحيحُ المالك ٢٠٢٦-٠٨-١٨: «عند إغلاق التطبيق وفتحه تُمسح الطلباتُ
 *  التي بالسلّة وهذا غلط — يجب أن تحتفظ به حتّى بعد الخروج أو إعادة
 *  تشغيل التطبيق».)
 *
 * # وكانت في الذاكرة عمدا
 *
 * **بحجّة أنّ الأسعارَ تتبدّل والمصادرَ تُغلق** — فسلّةُ أمسِ تحمل سعرَ
 * أمس. **والحجّةُ صحيحةٌ والدواءُ خطأ.**
 *
 * **ومن ملأ سلّتَه ثمّ ردّ على مكالمةٍ فقدها** — وهاتفُ اليوم يُخرج
 * التطبيقَ من الذاكرة لأتفهِ سبب: **مكالمةٌ، أو صورةٌ يلتقطها، أو
 * بطّاريّةٌ تُدار.** **وسلّةٌ تضيع بلا فعلٍ من صاحبها تُقرأ عطباً لا
 * سياسة.**
 *
 * # والسعرُ يُحمى بالتسعيرة لا بالنسيان
 *
 * **الشاشةُ تُسعّر عند فتحها والمحرّكُ يُعيد الحسابَ عند الإرسال** —
 * **فما يُدفع هو سعرُ اليوم لا سعرُ أمس.** وصنفٌ نفد يردّه المحرّكُ
 * برمزه (`item_unavailable`) وله عربيّةٌ تُقرأ.
 *
 * # ولماذا خارجَ الشاشة
 *
 * **يُضاف من السوق ويُقرأ في السلّة ويُعرض عدُّه في الشريط** — **وثلاثُ
 * شاشاتٍ تقرأ حالاً واحدة**: ولو مُلكت لإحداهنّ لَذهبت بذهابها.
 */
object Cart {

    var lines by mutableStateOf<List<Line>>(emptyList())
        private set

    val count: Int get() = lines.sumOf { it.qty }

    /** **ما يُقرأ قبل التسعيرة** — والمحرّكُ يُعيد الحسابَ عند الإرسال. */
    val subtotal: Long get() = lines.sumOf { it.item.price * it.qty }

    // ══════════════════════════════════════════════════════════════════
    // **والقرصُ يُلمس مرّةً عند الإقلاع ثمّ عند كلّ تبديل**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا تُقرأ في كلّ رسمة**: الحالُ في الذاكرة هي المصدر، **والقرصُ
    // نسخةٌ تُكتب** — **وقراءةُ ملفٍّ في كلّ إطارٍ تُبطئ التمرير.**
    private var store: android.content.SharedPreferences? = null

    /**
     * **تُنادى مرّةً عند الإقلاع** — قبل أوّل شاشة.
     *
     * **ومن نسي ندَاءها لا تُحفظ السلّة ولا يظهر خطأ** — **فتُقرأ
     * الحالُ فارغةً ويُظنّ أنّ الحفظَ لا يعمل.** ولذلك تُنادى بجانب
     * تركيب النواة.
     */
    fun install(context: Context) {
        if (store != null) return
        val prefs = context.applicationContext
            .getSharedPreferences(FILE, Context.MODE_PRIVATE)
        store = prefs
        val raw = prefs.getString(KEY, null) ?: return
        lines = runCatching {
            json.decodeFromString<List<Saved>>(raw).map { Line(it.item, it.qty) }
        }.getOrElse {
            // **وسلّةٌ لا تُقرأ تُمحى ولا تُسقط الإقلاع** — **صنفٌ تبدّل
            // حقلُه في المحرّك يكسر الفكَّ**، ولا يُترك التطبيقُ لا يفتح.
            Log.w("RahalGo/cart", "تعذّرت قراءةُ السلّة المحفوظة — تُمحى", it)
            prefs.edit().remove(KEY).apply()
            emptyList()
        }
    }

    /** **يُزاد صنفٌ أو يُرفع عدّه** — ولا يتكرّر السطرُ نفسُه مرّتين. */
    fun add(item: Item, qty: Int = 1) {
        val at = lines.indexOfFirst { it.item.id == item.id }
        lines = if (at >= 0) {
            lines.toMutableList().also { it[at] = it[at].copy(qty = it[at].qty + qty) }
        } else {
            lines + Line(item, qty)
        }
        save()
    }

    /** **يُنقص أو يُحذف** — والصفرُ يعني «ارفعه من السلّة». */
    fun setQty(itemId: String, qty: Int) {
        lines = if (qty <= 0) {
            lines.filter { it.item.id != itemId }
        } else {
            lines.map { if (it.item.id == itemId) it.copy(qty = qty) else it }
        }
        save()
    }

    fun clear() {
        lines = emptyList()
        save()
    }

    /**
     * **يكتب ما في الذاكرة على القرص.**
     *
     * **و`apply` لا `commit`** — الكتابةُ تقع في خيطٍ آخر، **و`commit`
     * توقف الخيطَ الرئيسَ عند كلّ ضغطةِ «زِد».**
     */
    private fun save() {
        val prefs = store ?: return
        val raw = runCatching {
            json.encodeToString(lines.map { Saved(it.item, it.qty) })
        }.getOrNull() ?: return
        prefs.edit().putString(KEY, raw).apply()
    }

    private val json = Json {
        // **وحقلٌ جديدٌ في المحرّك لا يُسقط سلّةً محفوظة.**
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    private const val FILE = "rahalgo_cart"
    private const val KEY = "lines"

    data class Line(val item: Item, val qty: Int)

    /** **صورةُ السطر على القرص** — والشاشةُ لا تعرفها. */
    @Serializable
    private data class Saved(val item: Item, val qty: Int)
}
