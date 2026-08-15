// ══════════════════════════════════════════════════════════════════════
//  **تطبيق الزبون — شاشات فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكةَ ولا منطقَ هنا — كلُّها في
//  `shared` و`ui`.)
//
// **ولا إشعاراتٍ بعد**: نقطةُ Firebase تحتاج `google-services.json` باسم
// هذا التطبيق، **وتُضاف في خطوتها** (٦ من خطوات الزبون) لا قبلها:
// **مفتاحٌ يُضاف بلا أن يُجرَّب يُنسى معطّلا.**
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
}

// **مفتاحُ الرفع — من ملفٍّ خارج المستودع** (كما في تطبيق السائق).
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}

android {
    namespace = "com.rahalgo.customer"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.rahalgo.customer"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = 1
        versionName = "0.1.0"
    }

    signingConfigs {
        create("upload") {
            val path = keystoreProps.getProperty("storeFile")
            if (path != null) {
                storeFile = file(path)
                storePassword = keystoreProps.getProperty("storePassword")
                keyAlias = keystoreProps.getProperty("keyAlias")
                keyPassword = keystoreProps.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            if (keystoreProps.getProperty("storeFile") != null) {
                signingConfig = signingConfigs.getByName("upload")
            }
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
        debug {
            // **ولاحقةٌ على المعرّف** — فيجلس التجريبيُّ والإصدارُ على
            // الجهاز نفسِه، **ولا يُحذف أحدُهما ليُثبَّت الآخر.**
            applicationIdSuffix = ".debug"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    implementation(project(":design"))
    implementation(project(":ui"))
    // **وخريطةٌ لعناوينه** — من أراد أن يحفظ بيتَ أمّه لا يحفظ موضعَه هو.
    implementation(project(":map"))
    implementation(project(":shared"))
    implementation(libs.androidx.security.crypto)
    implementation(libs.androidx.core.ktx)
    // **قراءةُ الموضع مرّةً عند حفظ عنوان** — لا خدمةَ تتبّع.
    implementation(libs.play.services.location)
    // **الوجهةُ المؤجَّلة** — من نزّله من المتجر يُنسب لمن دعاه.
    implementation(libs.install.referrer)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
