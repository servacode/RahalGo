package com.rahalgo.map.data

/**
 * **عنوانُ متاجرنا على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٢١) — يضعه التطبيقُ عند الإقلاع
 * (`/api/v1/public/map/stores?format=geojson`)، **وفارغُه: لا طبقةَ متاجر.**
 */
object MapStores {
    @Volatile
    var url: String = ""

    /**
     * **عنوانُ معالمِ الخريطة** (قرارُ المالك ٢٠٢٦-١١-٠٣) — مساجدُ ومستشفياتٌ ومدارسُ وما أضافه
     * السائقون (`/api/v1/public/map/landmarks?format=geojson`). **وفارغُه: لا معالم.**
     * **ويُضاف `&stores=0` حيث تُرسم المتاجرُ طبقةً مستقلّة.**
     */
    @Volatile
    var landmarksUrl: String = ""

    /**
     * **أسماءُ الأحياء والشوارع المُقرَّة من عناوين الزبائن** (قرارُ المالك ٢٠٢٦-١١-١٨: «تطلع عند الكل»)
     * — `/api/v1/public/map/areas`، **نفسُ ما تقرؤه خرائطُ الويب. وفارغُه: لا أسماء.**
     */
    @Volatile
    var areasUrl: String = ""
}
