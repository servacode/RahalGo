// ══════════════════════════════════════════════════════════════════════
//  **وحدةُ ملاحة السائق — المنطقُ لا الرسم**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٠ من خطّة الملاحة، بأمر المالك ٢٠٢٦-٠٨-٢٠.)
//
// # لماذا وحدةٌ مستقلّةٌ لا حزمةٌ داخل التطبيق
//
// **الحزمةُ لا تمنع شيئاً** — يستوردها من شاء. **والوحدةُ تمنع بالبناء
// نفسِه**: `map-core` لا تستطيع استيرادَ هذه ولو أراد كاتبُها، **وجرادل
// يرفض الدورة.**
//
// **وهو ما طلبه المالك صراحةً**: ألّا تصير `:map` وحدةً إلهاً.
//
// # وما يأتيها لاحقاً — ولا شيءَ منه في المرحلة ٠
//
// `NavigationSession` · `LocationEngine` · `RouteProgress` ·
// `ManeuverEngine` · `OffRouteDetector` · `WrongWayDetector` ·
// `RerouteEngine` · `VoiceGuide`
//
// **والمرحلةُ ٠ تفتح البابَ ولا تدخله** — تنقل ما هو قائمٌ كما هو.
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "com.rahalgo.navigation"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        minSdk = libs.versions.minSdk.get().toInt()
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures { compose = true }
}

dependencies {
    // **وترسم بـ`:map` ولا تُعيد بناءَ أساسِها** — التهيئةُ واللوحُ
    // ودورةُ الحياة هناك.
    implementation(project(":map"))
    implementation(project(":ui"))
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.material3)
    implementation(libs.androidx.core.ktx)
    implementation(libs.maplibre)
}
