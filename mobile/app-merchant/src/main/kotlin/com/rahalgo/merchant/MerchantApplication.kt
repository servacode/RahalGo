package com.rahalgo.merchant

import android.app.Application
import android.content.ComponentCallbacks2
import com.rahalgo.map.MapStyleRepository
import com.rahalgo.map.data.MapConfig
import com.rahalgo.map.dispatchMapLowMemory

/**
 * ══════════════════════════════════════════════════════════════════
 * **إقلاعُ تطبيق المتجر — تهيئةُ الخريطة وتوصيلُ ضغطِ الذاكرة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «طبعاً يجب أن تكون خريطة يحدّد موقعه
 *  بدقّة المتجر».)
 *
 * # وكُتب أنّه لا يلزم — ثمّ لزم
 *
 * **كان في هذا التطبيق سطرٌ يقول: «لا صنفَ `Application`، وتطبيقُ
 * المتجر لا خريطةَ فيه».** **وذاك كان صحيحاً يومَ كُتب**: لم يكن
 * لصاحب المتجر أن يحرّك دبّوسَه.
 *
 * **فلمّا صار له ذلك لزم الصنف** — لا نسخاً عن أخيه:
 *
 * **الأوّل** — `onLowMemory` **تصل التطبيقَ لا النشاط** (دَينُ
 * `TD-MAP-LOWMEM`). فما لم يكن صنفٌ يسمعه **لم تسمعه خريطةٌ أبداً**،
 * **فيُقتل التطبيقُ بدل أن يتقلّص.**
 *
 * **الثاني** — **مستودعُ النمط يُهيّأ قبل أن تُرسم خريطة**، ولا
 * يجوز أن يكون ذلك في شاشة.
 *
 * # وضغطُ الذاكرة يخصّ هذا التطبيقَ خصوصاً
 *
 * **صاحبُ المطعم لا يُغلق تطبيقَه** — يبقى مفتوحاً ساعاتٍ في المطبخ
 * لينتظر رنّةَ طلب. **وخريطةٌ تُفتح مرّةً ثمّ تبقى في الذاكرة تجعل
 * النظامَ يقتل التطبيقَ** — **فتضيع الرنّة، وهي كلُّ سببِ وجوده.**
 */
class MerchantApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        MapStyleRepository.init(this, MapConfig(baseUrl = Backend.MAPS_BASE_URL))
    }

    @Deprecated("يبقى للأجهزة القديمة — والمسموعُ فعلاً onTrimMemory")
    override fun onLowMemory() {
        super.onLowMemory()
        dispatchMapLowMemory()
    }

    override fun onTrimMemory(level: Int) {
        super.onTrimMemory(level)
        // **ولا يُنادى عند كلّ درجة** — إخفاءُ الواجهة ليس ضغطَ ذاكرة.
        if (level >= ComponentCallbacks2.TRIM_MEMORY_RUNNING_LOW) {
            dispatchMapLowMemory()
        }
    }
}
