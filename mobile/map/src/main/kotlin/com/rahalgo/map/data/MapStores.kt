package com.rahalgo.map.data

/**
 * **عنوانُ متاجرنا على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٢١) — يضعه التطبيقُ عند الإقلاع
 * (`/api/v1/public/map/stores?format=geojson`)، **وفارغُه: لا طبقةَ متاجر.**
 */
object MapStores {
    @Volatile
    var url: String = ""
}
