// ══════════════════════════════════════════════════════════════════════
//  **وحدةُ الخرائط — منتقي نقطةٍ لثلاثة تطبيقات**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٤: نبدأ بالمندوب، **ويلزمه أن يضع نقطةَ متجرٍ
//  ليست نقطتَه هو.**)
//
// # ولماذا وحدةٌ رابعةٌ لا داخل `:ui`
//
// **`maplibre` مكتبةٌ أصليّةٌ ثقيلة** — تُضيف عدّةَ ميغاباتٍ لكلّ معماريّة.
// **ولو وُضعت في `:ui` لَحملها كلُّ تطبيقٍ ولو لم يرسم خريطةً قطّ.**
//
// **وتطبيقُ الزبون اليومَ لا يرسم خريطة** — يقرأ موضعَه بضغطة. **ويومَ
// يحتاجها لعناوينه المحفوظة يُعلنها**، ولا تُفرض عليه قبل ذلك.
//
// # وما فيها
//
// **منتقي نقطةٍ وبحثُ عنوان** — **ولا شيءَ يعرف دورا**: المندوبُ يضع
// نقطةَ متجر، والمتجرُ يصحّح نقطتَه، والزبونُ يحفظ بيتَ أمّه.
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "com.rahalgo.map"
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

kotlin { compilerOptions { jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) } }

dependencies {
    api(project(":ui"))
    // **و`api` لأنّ الشاشاتِ تُرجع `LatLng`** — ولو أُخفيت لَأعلنها كلُّ
    // تطبيقٍ من جديد.
    api(libs.maplibre)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
