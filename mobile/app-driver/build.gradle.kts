// **تطبيق السائق — شاشات فقط.**
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكة ولا منطق هنا — كلها في `shared`.)
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
    alias(libs.plugins.google.services)
    alias(libs.plugins.crashlytics)
}

// ══════════════════════════════════════════════════════════════════════
// **مفتاحُ الرفع — من ملفٍّ خارج المستودع**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نبدأ بالإغلاق واحدةً تلو الأخرى».)
//
// **وكانت نسخةُ الإصدار بلا توقيع** — فتُوقَّع بمفتاح التجربة لتُجرَّب،
// **وغوغل يرفض مفتاحَ التجربة.**
//
// **ولا يُكتب المفتاحُ ولا كلمتُه في المستودع**: من قرأ الشيفرة يستطيع
// أن يبني نسخةً موقّعةً باسم المنصّة، **فيُثبّتها الناسُ ظنّاً أنّها
// منها.**
//
// **وغيابُ الملفّ لا يُسقط البناء** — من نسخ المستودعَ يبني تجريبيّاً
// بلا مفتاح، **وشرطٌ يمنعه يجعل الشيفرةَ لا تُبنى إلّا على جهازٍ واحد.**
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
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
            // **ويُوقَّع إن وُجد المفتاح** — وإلّا خرج بلا توقيعٍ كما كان.
            if (keystoreProps.getProperty("storeFile") != null) {
                signingConfig = signingConfigs.getByName("upload")
            }
            // **الضغط والتشويش في الإصدار وحده** — البناء التجريبي يبقى
            // مقروءا في تقارير الانهيار.
            isMinifyEnabled = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            // ══════════════════════════════════════════════════════════
            // **وخريطةُ فكّ التشويش تُرفع مع كلّ إصدار**
            // ══════════════════════════════════════════════════════════
            //
            // **بلاها يصل التقريرُ بأسماءٍ من حرفين**: `a.b(c.java:12)` —
            // **فيُعرف أنّ التطبيقَ سقط ولا يُعرف أين.** والانهيارُ الذي
            // لا يُقرأ لا يُصلَح.
        }
        debug {
            // ══════════════════════════════════════════════════════════
            // **ولا تُرفع انهياراتُ البناء التجريبيّ**
            // ══════════════════════════════════════════════════════════
            //
            // **وهي انهياراتي أنا لا انهياراتُ سائق** — تُختلط بها
            // فتُقرأ الأرقامُ كاذبة: **عشرةُ سقوطٍ في التطوير تُقرأ
            // عشرةَ سائقين.**
            //
            // **ويُبدَّل بسطرٍ حين يُراد فحصُها** (`-PcrashDebug`).
            configure<com.google.firebase.crashlytics.buildtools.gradle.CrashlyticsExtension> {
                mappingFileUploadEnabled = false
            }
            // **ولاحقة على المعرّف** — فيجلس التجريبي والإصدار على الجهاز
            // نفسه، **ولا يُحذف أحدهما ليُثبَّت الآخر.**
            applicationIdSuffix = ".debug"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    // **و`BuildConfig` يُولَّد** — منه تُعرف السمةُ التجريبيّةُ من
    // الإصدار، **فلا تُرسَل انهياراتُ التطوير مع انهيارات السائقين.**
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
    // **وحدةُ التصميم — الثيمُ والحركةُ لأربعة تطبيقات.**
    implementation(project(":design"))
    implementation(project(":shared"))
    implementation(libs.androidx.security.crypto)
    // **شاشةُ النظام عند الإقلاع** — الطريقةُ الرسميّة، انظر `themes.xml`.
    implementation(libs.androidx.splashscreen)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.play.services.location)
    implementation(libs.maplibre)
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)
    implementation(libs.firebase.crashlytics)
    implementation(libs.coroutines.play.services)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
