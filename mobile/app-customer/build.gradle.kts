// ══════════════════════════════════════════════════════════════════════
//  **تطبيق الزبون — شاشات فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكةَ ولا منطقَ هنا — كلُّها في
//  `shared` و`ui`.)
//
// **والإشعاراتُ وُصلت ٢٠٢٦-٠٨-٢٠**: سُجّل التطبيقُ في `rahalgo-prod`
// بجانب السائق — **مشروعٌ واحدٌ لا مشروعان**: مفتاحُ خادمٍ واحدٌ يخدم
// الثلاثة، **ومشروعان يعنيان مفتاحين ينسى أحدُهما.**
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
    // **وإضافةُ غوغل تقرأ `google-services.json`** — وتُسقط البناءَ إن
    // لم تجد اسمَ الحزمة فيه، **فلا يمرّ بناءٌ بإعدادٍ ناقص.**
    alias(libs.plugins.google.services)
    // **والسلّةُ تُحفظ على القرص** — (تصحيحُ المالك ٢٠٢٦-٠٨-١٨).
    // **وتُسلسَل بالمولّد لا بيد** — وكاتبُ JSON بيده ينسى حقلاً يُضاف.
    alias(libs.plugins.kotlin.serialization)
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
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        applicationId = "com.rahalgo.customer"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        // ══════════════════════════════════════════════════════════════
        // **ورقمُ النسخة يُزاد قبل كلّ رفعة**
        // ══════════════════════════════════════════════════════════════
        //
        // (تدقيقُ الجاهزيّة ٢٠٢٦-٠٨-١٩.)
        //
        // **وبلاي يرفض رفعةً برقمٍ سبق أن رُفع** — ولا رسالةَ تقول
        // «زدِ الرقم»، **بل رفضٌ عند الباب.**
        //
        // **والاسمُ يُقرأ ويُذكر في المتجر** (`versionName`)، **والرقمُ
        // لا يراه أحدٌ وهو ما يحكم.**
        versionCode = 2
        versionName = "1.0.0"
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
            // **واللاحقةُ على المعرّف وحدَها لا تُرى**: أيقونتان
            // باسمٍ واحدٍ في شاشة الجهاز، **ولا شيء يقول أيُّهما
            // الجديدة.**
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
    // ══════════════════════════════════════════════════════════════════
    // **والدفعُ إلى الجهاز — لِما يصل والتطبيقُ مغلق**
    // ══════════════════════════════════════════════════════════════════
    //
    // (سؤالُ المالك ٢٠٢٦-٠٨-١٩: «المفروض يصل إشعارٌ لو التطبيق مغلق
    //  صحيح؟».)
    //
    // **والمقبسُ الحيُّ يعمل والتطبيقُ مفتوحٌ وحدَه** — ومن أغلقه لا
    // يعرف أنّ سائقَه يسأله «أيّ طابق؟»، **فيقف على الباب ينتظر جواباً
    // لا يأتي.**
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)
    // ══════════════════════════════════════════════════════════════════
    // **حزمةُ الواجهة — تُشغَّل على جهازٍ متّصل**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٩: «أريد اختباراتٍ تقود التطبيقَ فعلاً،
    //  لا تعتمد فقط على ViewModel tests».)
    //
    // **و`ui-test-manifest` في `debug` لا في `androidTest`** — هي التي
    // تُعلن `ComponentActivity` الفارغةَ التي يُركَّب فيها المكوّن،
    // **وبدونها تسقط الحزمةُ بلا نشاطٍ يستضيفها.**
    androidTestImplementation(platform(libs.compose.bom))
    androidTestImplementation(libs.compose.ui.test)
    androidTestImplementation(libs.androidx.test.runner)
    androidTestImplementation(libs.androidx.test.rules)
    androidTestImplementation(libs.junit)
    debugImplementation(libs.compose.ui.test.manifest)
    // **حزمةُ الوحدة** — تُشغَّل بـ`./gradlew testDebugUnitTest`.
    testImplementation(libs.junit)
    testImplementation(libs.coroutines.test)
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
