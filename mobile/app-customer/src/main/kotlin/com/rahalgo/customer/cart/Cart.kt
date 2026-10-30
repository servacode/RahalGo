package com.rahalgo.customer.cart

import android.content.Context
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.customer.CartLine
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ModifierOption
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
    val subtotal: Long get() = lines.sumOf { it.total }

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
            json.decodeFromString<List<Saved>>(raw).map { it.toLine() }
        }.getOrElse {
            // **وسلّةٌ لا تُقرأ تُمحى ولا تُسقط الإقلاع** — **صنفٌ تبدّل
            // حقلُه في المحرّك يكسر الفكَّ**، ولا يُترك التطبيقُ لا يفتح.
            Log.w("RahalGo/cart", "تعذّرت قراءةُ السلّة المحفوظة — تُمحى", it)
            prefs.edit().remove(KEY).apply()
            emptyList()
        }
    }

    /**
     * **يُزاد صنفٌ أو يُرفع عدّه** — ولا يتكرّر السطرُ نفسُه مرّتين.
     *
     * # **والاختلافُ في الخيارات اختلافٌ في السطر**
     *
     * «برغر كبير بجبنة» و«برغر صغير بلا جبنة» **صنفٌ واحدٌ ومعرّفٌ
     * واحد** — **فجمعُهما في سطرٍ يعني أن يختفي أحدُ الطلبين**، ويقرأ
     * المتجرُ اثنين متطابقين ويصنعهما متطابقين.
     *
     * **فالمفتاحُ هو المعرّفُ وخياراتُه معاً** لا المعرّفَ وحدَه.
     */
    /** **يُرجع `true` إن زاد كميّةَ صنفٍ قائمٍ** — ليختار المنادي رسالةَ التأكيد. */
    fun add(
        item: Item,
        qty: Int = 1,
        options: List<ModifierOption> = emptyList(),
        /** **المشروباتُ بأعدادها** (٢٠٢٦-١٠-٣٠) — لا تُضرب بعدد الصنف. */
        drinks: List<DrinkPick> = emptyList(),
    ): Boolean {
        val line = Line(item, qty, options, drinks.filter { it.qty > 0 })
        val at = lines.indexOfFirst { it.key == line.key }
        val increased = at >= 0
        lines = if (increased) {
            // **والمشروباتُ تُجمع كما يُجمع العدد** — من أضاف «شاورما وعيران» مرّتين أراد عيرانين.
            lines.toMutableList().also { it[at] = it[at].copy(qty = it[at].qty + qty, drinks = it[at].drinks.plusDrinks(line.drinks)) }
        } else {
            lines + line
        }
        save()
        return increased
    }

    /**
     * **سطورُ نافذةِ الصنف دفعةً واحدة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «٢ عادي + ١ دبل»
     * سطران في السلّة. **و`true` إن زاد أيٌّ منها سطراً قائماً.**
     */
    fun addAll(item: Item, picks: List<Pick>): Boolean {
        var increased = false
        picks.filter { it.qty > 0 }.forEach { p ->
            if (add(item, p.qty, p.options, p.drinks)) increased = true
        }
        return increased
    }

    /** **سطرٌ تُخرجه نافذةُ الصنف** — خياراتُه وعددُه ومشروباتُه. */
    data class Pick(
        val options: List<ModifierOption>,
        val qty: Int,
        val drinks: List<DrinkPick> = emptyList(),
    )

    /**
     * **يُنقص أو يُحذف** — والصفرُ يعني «ارفعه من السلّة».
     *
     * **وبمفتاح السطر لا بمعرّف الصنف** — **ولو كان بالمعرّف لَحذف
     * «برغر بجبنة» حين يُنقص «برغر بلا جبنة».**
     */
    fun setQty(key: String, qty: Int) {
        lines = if (qty <= 0) {
            lines.filter { it.key != key }
        } else {
            lines.map { if (it.key == key) it.copy(qty = qty) else it }
        }
        save()
    }

    /**
     * **عددُ مشروبٍ في سطره** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — يُبدَّل من «سلّتي» مباشرةً،
     * **والصفرُ يرفع المشروبَ وحدَه** لا الصنف. ويُرسَل بعدها في `option_qty` للسطر.
     * **وسطرٌ صار مطابقاً لآخر يُدمج فيه** — فلا يظهر سطران متطابقان.
     */
    fun setDrinkQty(key: String, optionId: String, qty: Int) {
        val at = lines.indexOfFirst { it.key == key }
        if (at < 0) return
        val line = lines[at]
        val drinks = line.drinks.mapNotNull { d ->
            when {
                d.option.id != optionId -> d
                qty <= 0 -> null
                else -> d.copy(qty = minOf(MAX_DRINK, qty))
            }
        }
        val updated = line.copy(drinks = drinks)
        val rest = lines.toMutableList().also { it.removeAt(at) }
        val same = rest.indexOfFirst { it.key == updated.key }
        if (same >= 0) {
            rest[same] = rest[same].copy(qty = rest[same].qty + updated.qty)
        } else {
            rest.add(at, updated)
        }
        lines = rest
        save()
    }

    // ══════════════════════════════════════════════════════════════════
    // **المشروباتُ تحت كلّ الأصناف — سطرٌ لكلّ مشروب** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
    // ══════════════════════════════════════════════════════════════════
    //
    // «شاورما عادي» ثمّ «شاورما دبل» ثمّ المشروباتُ تحتهما — **وعيرانٌ مع العادي وعيرانٌ
    // مع الدبل سطرٌ واحدٌ بعددهما معاً.** والمشروبُ يبقى محفوظاً في سطر صنفه
    // (`option_qty` لسطره)، **والجمعُ للعرض وحدَه.**

    /** **مشروبٌ في سطرٍ من السلّة** — مفتاحُ السطر ومعرّفُ الخيار وعددُه. */
    data class DrinkEntry(val lineKey: String, val optionId: String, val qty: Int)

    /** **سطرُ مشروبٍ مجموعٌ** — باسمه، بعدد كلّ سطوره ومجموعِ ثمنها. */
    data class DrinkRow(
        val name: String,
        val thumb: String?,
        val qty: Int,
        val total: Long,
        val entries: List<DrinkEntry>,
    )

    /** **المشروباتُ مجموعةً بالاسم** بترتيب أوّل ظهورها في السلّة. */
    fun drinkRows(from: List<Line> = lines): List<DrinkRow> {
        val out = LinkedHashMap<String, DrinkRow>()
        from.forEach { line ->
            line.drinks.forEach { d ->
                val name = d.option.name.trim()
                val e = DrinkEntry(line.key, d.option.id, d.qty)
                val was = out[name]
                out[name] = if (was == null) {
                    DrinkRow(name, d.option.imageThumbUrl, d.qty, d.option.priceDelta * d.qty, listOf(e))
                } else {
                    was.copy(
                        thumb = was.thumb ?: d.option.imageThumbUrl,
                        qty = was.qty + d.qty,
                        total = was.total + d.option.priceDelta * d.qty,
                        entries = was.entries + e,
                    )
                }
            }
        }
        return out.values.toList()
    }

    /** **«+» على سطر المشروب** — يزيد أوّلَ سطرٍ لم يبلغ السقف. */
    fun moreDrink(name: String) {
        val row = drinkRows().firstOrNull { it.name == name.trim() } ?: return
        val e = row.entries.firstOrNull { it.qty < MAX_DRINK } ?: return
        setDrinkQty(e.lineKey, e.optionId, e.qty + 1)
    }

    /** **أيُزاد** — ما دام في سطوره ما لم يبلغ السقف. */
    fun canMoreDrink(row: DrinkRow): Boolean = row.entries.any { it.qty < MAX_DRINK }

    /** **«−» على سطر المشروب** — ينقص أوّلَ سطر، وإن صار صفراً انتقل التالي إلى أوّلٍ. */
    fun lessDrink(name: String) {
        val row = drinkRows().firstOrNull { it.name == name.trim() } ?: return
        val e = row.entries.firstOrNull() ?: return
        setDrinkQty(e.lineKey, e.optionId, e.qty - 1)
    }

    /** **سلّةُ الحذف على سطر المشروب** — يرفعه من كلّ الأصناف، **والأصنافُ تبقى.** */
    fun removeDrink(name: String) {
        val n = name.trim()
        val out = ArrayList<Line>(lines.size)
        lines.forEach { l ->
            val updated = l.copy(drinks = l.drinks.filter { it.option.name.trim() != n })
            val same = out.indexOfFirst { it.key == updated.key }
            if (same >= 0) out[same] = out[same].copy(qty = out[same].qty + updated.qty) else out += updated
        }
        lines = out
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
            json.encodeToString(lines.map { Saved(it.item, it.qty, it.options, if (it.legacy) null else it.drinks) })
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

    data class Line(
        val item: Item,
        val qty: Int,
        /** **ما اختاره** — حجمٌ وإضافات، فارغةٌ لصنفٍ بلا خيارات. */
        val options: List<ModifierOption> = emptyList(),
        /** **المشروباتُ بأعدادها** — كلٌّ يُحسب بعدده هو لا بعدد الصنف. */
        val drinks: List<DrinkPick> = emptyList(),
        /**
         * **سطرٌ من سلّةٍ حُفظت قبل عدّاد المشروبات** — مشروبُه (إن كان) داخل
         * `options` ويُضرب بعدد الصنف. **فلا يُرسَل له `option_qty`** ليبقى سعرُه
         * كما رآه، ولا يضيع مشروبُه بصمت.
         */
        val legacy: Boolean = false,
    ) {
        /**
         * **هويّةُ السطر** — المعرّفُ وخياراتُه مرتَّبةً.
         *
         * **والترتيبُ مقصود**: «جبنة ثمّ ثوم» و«ثوم ثمّ جبنة» اختيارٌ
         * واحد، **ولو لم يُرتَّب لَصارا سطرين في السلّة.**
         */
        val key: String
            get() = item.id + "|" + options.map { it.id }.sorted().joinToString(",") +
                (if (drinks.isEmpty()) "" else "|" + drinks.map { it.option.id + "x" + it.qty }.sorted().joinToString(","))

        /** **ثمنُ الواحد بخياراته** — والفرقُ يُضاف كما يُضاف في المحرّك. */
        val unitPrice: Long get() = item.price + options.sumOf { it.priceDelta }

        /** **ثمنُ المشروبات** — كلٌّ بسعره وعدده، ولا يُضرب بعدد الصنف. */
        val drinksTotal: Long get() = drinks.sumOf { it.option.priceDelta * it.qty }

        /** **ثمنُ السطر كاملاً** — الصنفُ بعدده وخياراته، ثمّ المشروبات. */
        val total: Long get() = unitPrice * qty + drinksTotal

        /**
         * **صورةُ السطر كما يقرؤها المحرّك.**
         *
         * **وموضعٌ واحدٌ للتحويل**: التسعيرةُ والإرسالُ يبنيان الحمولةَ
         * نفسَها، **ونسختان تفترقان يوماً** فيُسعَّر بخياراتٍ ويُرسَل
         * بغيرها — **ويدفع الزبونُ غيرَ ما رأى.**
         */
        fun toPayload(): CartLine =
            CartLine(
                menuItemId = item.id,
                qty = qty,
                optionIds = options.map { it.id },
                optionQty = if (legacy) null else drinks.associate { it.option.id to it.qty },
            )
    }

    /** **مشروبٌ وعددُه** — من مجموعة «مشروبات» في نافذة الصنف. */
    @Serializable
    data class DrinkPick(val option: ModifierOption, val qty: Int)

    /** **يجمع مشروبين بالمعرّف نفسِه** — والسقفُ ٢٠ كما في المحرّك. */
    private fun List<DrinkPick>.plusDrinks(more: List<DrinkPick>): List<DrinkPick> {
        val out = LinkedHashMap<String, DrinkPick>()
        (this + more).forEach { d ->
            val was = out[d.option.id]
            out[d.option.id] = if (was == null) d else was.copy(qty = minOf(MAX_DRINK, was.qty + d.qty))
        }
        return out.values.toList()
    }

    /** **سقفُ المشروب الواحد في السطر** — مرآةُ `maxDrinkQty` في المحرّك. */
    const val MAX_DRINK = 20

    /** **صورةُ السطر على القرص** — والشاشةُ لا تعرفها. */
    @Serializable
    private data class Saved(
        val item: Item,
        val qty: Int,
        val options: List<ModifierOption> = emptyList(),
        /** **غيابُه = سلّةٌ قديمة** — انظر `Line.legacy`. */
        val drinks: List<DrinkPick>? = null,
    ) {
        fun toLine(): Line = Line(item, qty, options, drinks.orEmpty(), legacy = drinks == null)
    }
}
