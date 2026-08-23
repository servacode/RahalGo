// ══════════════════════════════════════════════════════════════════════
//  **تطبيق المتجر — شاشات فقط**
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
    namespace = "com.rahalgo.merchant"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.rahalgo.merchant"
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

            // ══════════════════════════════════════════════════════════
            // **واسمٌ يقول إنّه تجريبيّ**
            // ══════════════════════════════════════════════════════════
            //
            // **واللاحقةُ على المعرّف وحدَها لا تُرى**: أيقونتان باسمٍ
            // واحدٍ في شاشة الجهاز، **ولا شيء يقول أيُّهما الجديدة.**
            //
            // **وقع ٢٠٢٦-٠٨-١٨**: بُني الإصلاحُ وثُبّت وأُبلغ المالكُ
            // «تمّ» — **ففتح النسخةَ الأخرى فوجد العطبَ قائما.**
            //
            // **وأسوأُ من ضياع الوقت أن يُشكَّ في الإصلاح**: أُبلغ أنّ
            // العطبَ زال وهو يراه بعينه.
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
    implementation(project(":shared"))
    // ══════════════════════════════════════════════════════════════════
    // **والخريطةُ لزمت** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣).
    // ══════════════════════════════════════════════════════════════════
    //
    // **وكان هنا سطرٌ يقول إنّها لا تلزم**: «المتجرُ لا يوصّل ولا
    // يسجّل نفسَه — نقطتُه تُوضع مرّةً عند التسجيل». **وذاك كان صحيحاً
    // يومَ كُتب.**
    //
    // **ثمّ صار له أن يصحّح دبّوسَه**: ينتقل، أو يجد نفسَه في الشارع
    // المجاور، **والسائقُ هو من يدفع الثمن — يقف على بابٍ ليس بابَه.**
    //
    // **والنصُّ وحدَه لا يكفي**: الأجرةُ تُحسب من الدبّوس لا من
    // العنوان المكتوب، **فمتجرٌ يصحّح عنوانَه نصّاً ودبّوسُه مخطئٌ
    // بمئة مترٍ يبقى أجرُه مخطئاً.**
    implementation(project(":map"))
    implementation(libs.androidx.security.crypto)
    implementation(libs.androidx.core.ktx)
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
