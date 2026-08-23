package com.rahalgo.customer

import android.app.Application
import android.content.ComponentCallbacks2
import com.rahalgo.customer.Backend
import com.rahalgo.map.MapStyleRepository
import com.rahalgo.map.data.MapConfig
import com.rahalgo.map.dispatchMapLowMemory

/**
 * ══════════════════════════════════════════════════════════════════
 * **إقلاعُ التطبيق — تهيئةُ الخريطة وتوصيلُ ضغطِ الذاكرة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٣ و٢٩.)
 *
 * # **ولماذا صنفُ `Application` الآن**
 *
 * **لم يكن للمشروع صنفٌ للتطبيق** — كلُّ شيءٍ يبدأ في `MainActivity`.
 * **وذلك يكفي لواجهةٍ ولا يكفي لخريطة**:
 *
 * **الأوّل** — `onLowMemory` **يصل التطبيقَ لا النشاط** (دَينُ
 * `TD-MAP-LOWMEM`). فما لم يكن صنفٌ يسمعه **لم تسمعه خريطةٌ أبداً**،
 * **فيُقتل التطبيقُ بدل أن يتقلّص.**
 *
 * **الثاني** — عاملُ التنزيل يعمل **بلا نشاطٍ مفتوح** (البند ١٢).
 * **فتهيئةُ المستودع لا يجوز أن تكون في شاشة.**
 *
 * # **و`onTrimMemory` أهمُّ من `onLowMemory`**
 *
 * **`onLowMemory` تكاد لا تُنادى في أندرويد الحديث** — النظامُ يقتل
 * قبلها. **و`onTrimMemory` تصل مبكّراً وبدرجات**، وهي المسموعةُ فعلاً.
 */
class CustomerApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        // ══════════════════════════════════════════════════════════
        // **وقنواتُ الإشعار قبل أن تصل رسالة**
        // ══════════════════════════════════════════════════════════
        //
        // (قِيس ٢٠٢٦-٠٨-٢٣: إشعارٌ يصل بلا صوتٍ على قناة فايربيس
        //  الاحتياطيّة.)
        //
        // **وكانت تُنشأ داخل `show()`** — أي بعد وصول رسالة.
        // **ورسالةُ `notification` لا تُسلَّم إلى التطبيق وهو في
        // الخلفيّة** — يرسمها النظام، **فلا تُنشأ القناةُ أبداً في
        // الحالة التي نحتاجها.**
        com.rahalgo.ui.PushChannels.ensure(this)

        MapStyleRepository.init(this, MapConfig(baseUrl = Backend.MAPS_BASE_URL))
    }

    /**
     * **يوصل ضغطَ الذاكرة إلى كلّ خريطةٍ حيّة** — البند ٢٩.
     *
     * **ولا مرجعَ عامٌّ إلى `MapView`** — السجلُّ في
     * `MapLifecycleBridge` **يُنزع منه عند الهدم**، فلا يبقى نشاطٌ
     * محبوساً في الذاكرة.
     */
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
