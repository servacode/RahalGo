// **تطبيق السائق — شاشات فقط.**
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكة ولا منطق هنا — كلها في `shared`.)
plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "com.rahalgo.driver"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.rahalgo.driver"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = 1
        versionName = "0.1.0"
    }

    buildTypes {
        release {
            // **الضغط والتشويش في الإصدار وحده** — البناء التجريبي يبقى
            // مقروءا في تقارير الانهيار.
            isMinifyEnabled = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
        debug {
            // **ولاحقة على المعرّف** — فيجلس التجريبي والإصدار على الجهاز
            // نفسه، **ولا يُحذف أحدهما ليُثبَّت الآخر.**
            applicationIdSuffix = ".debug"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures { compose = true }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    // **وحدةُ التصميم — الثيمُ والحركةُ لأربعة تطبيقات.**
    implementation(project(":design"))
    // **شاشةُ النظام عند الإقلاع** — الطريقةُ الرسميّة، انظر `themes.xml`.
    implementation(libs.androidx.splashscreen)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.ktx)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
