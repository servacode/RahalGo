// ══════════════════════════════════════════════════════════════════════
//  **وحدة التصميم — الألوان والخط والحركة، لأربعة تطبيقات**
// ══════════════════════════════════════════════════════════════════════
//
// (قرار المالك ٢٠٢٦-٠٨-١١، و`GROUND-RULES.md` §7.1.)
//
// **ولماذا وحدةٌ من اليوم الأول** — وشاشةُ الافتتاح أول ساكنيها:
//
// الحركةُ نفسُها ستلزم تطبيقَ الزبون والمتجر والمندوب. **ولو كُتبت في
// `app-driver` لَنُسخت ثلاثَ مرات** — وهذا بعينه ما وقع في الويب: خمسة
// تطبيقات متشابهة اضطررنا لدمجها.
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "com.rahalgo.design"
    compileSdk = libs.versions.compileSdk.get().toInt()
    defaultConfig { minSdk = libs.versions.minSdk.get().toInt() }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    buildFeatures { compose = true }
}

kotlin { compilerOptions { jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) } }

dependencies {
    // **و`api` لا `implementation`** — التطبيقاتُ تستعمل أنواعَ Compose
    // نفسَها، **ولو أُخفيت لَاضطر كلُّ تطبيقٍ أن يُعلنها من جديد.**
    api(platform(libs.compose.bom))
    api(libs.compose.ui)
    api(libs.compose.ui.graphics)
    api(libs.compose.material3)
    implementation(libs.androidx.core.ktx)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
