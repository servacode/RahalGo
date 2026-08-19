// ══════════════════════════════════════════════════════════════════════
//  **القلب المشترك — ولا يعرف أندرويد**
// ══════════════════════════════════════════════════════════════════════
//
// (`GROUND-RULES.md` §7.1 و§7.2.)
//
// **لا `android.*` ولا `Context` ولا Compose هنا.** وبابُ iOS يُغلق بهدوء
// بأول استيراد، **ولا يُكتشف إلا بعد سنة** حين يُراد فتحه فلا ينفتح.
//
// **ووحدة أندرويد لا وحدة KMP كاملة اليوم**: البنية جاهزة للانتقال —
// الشيفرة كلها Kotlin خالص بلا واجهة نظام — **والانتقال يوم iOS تبديل
// إضافة لا إعادة كتابة.**
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.rahalgo.shared"
    compileSdk = libs.versions.compileSdk.get().toInt()
    defaultConfig { minSdk = libs.versions.minSdk.get().toInt() }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

kotlin { compilerOptions { jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) } }

dependencies {
    // **حزمةُ الوحدة** — تُشغَّل بـ`./gradlew testDebugUnitTest`.
    testImplementation(libs.junit)
    testImplementation(libs.coroutines.test)
    api(libs.ktor.core)
    api(libs.kotlinx.serialization.json)
    implementation(libs.ktor.okhttp)
    implementation(libs.ktor.content.negotiation)
    implementation(libs.ktor.websockets)
    implementation(libs.ktor.json)
}
