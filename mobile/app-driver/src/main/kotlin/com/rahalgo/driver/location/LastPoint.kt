package com.rahalgo.driver.location

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **آخر موضع عُرف — تكتبه الخدمة وتقرؤه الشاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والخدمة والشاشة لا تلتقيان**: الأولى تعمل والتطبيق مغلق، والثانية
 * تُبنى وتُهدم مع كلّ دوران جهاز. **فالموضع يُترك في مكان يعرفه
 * الاثنان.**
 *
 * **ولا يُحفظ على القرص**: قيمة عمرها ثوانٍ، **وموضع الأمس على خريطة
 * اليوم كذب.** والخدمة تكتبه ثانيةً بعد ثوانٍ من كلّ إقلاع.
 *
 * **وحالة Compose لا متغيّر عاديّ** — فالخريطة تتحرّك معه بلا أن يسأل
 * أحد.
 */
object LastPoint {

    var value by mutableStateOf<Point?>(null)
        private set

    fun set(lat: Double, lng: Double) {
        value = Point(lat, lng)
    }

    data class Point(val lat: Double, val lng: Double)
}
