// ══════════════════════════════════════════════════════════════════════
//  **تطبيق المندوب — شاشات فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكةَ ولا منطقَ هنا — كلُّها في
//  `shared` و`ui`.)
//
// **والإشعاراتُ مكتوبةٌ كاملةً** (`push/PushService.kt` والبيان)،
// **وتُفعَّل ساعةَ يُسجَّل `com.rahalgo.rep` في Firebase** ويُنزَّل ملفُّه
// الجديد فوق القديم.
//
// # ولماذا شرطٌ لا سطرٌ ثابت
//
// **`google-services.json` الحاضرُ نسخةٌ من ملفّ الزبون** — لا يذكر
// حزمةَ المندوب. **وإضافةُ غوغل تُسقط البناءَ إن لم تجد الاسمَ فيه**،
// فسطرٌ ثابتٌ يمنع بناءَ التطبيق اليومَ كلَّه.
//
// **والشرطُ يصرخ في كلّ بناء** — فلا يُنسى معطّلاً، وهو ما وقع سابقاً.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
}

// ══════════════════════════════════════════════════════════════════════
// **نقطةُ Firebase — تُفتح وحدَها متى صحّ الملفّ**
// ══════════════════════════════════════════════════════════════════════
//
// **و`crashlytics` لازمةٌ مع `google-services`** — و`:ui` تعلن مكتبتَها
// للأربعة، **فمن أضاف الأولى وحدَها سقط تطبيقُه عند أوّل إقلاع**
// (`The Crashlytics build ID is missing` — وقع في المتجر ٢٠٢٦-٠٨-٢٦).
val firebaseReady = file("google-services.json").let {
    it.exists() && it.readText().contains("com.rahalgo.rep")
}
if (firebaseReady) {
    apply(plugin = libs.plugins.google.services.get().pluginId)
    apply(plugin = libs.plugins.crashlytics.get().pluginId)
} else {
    logger.warn("═".repeat(58))
    logger.warn("  تطبيق المندوب يُبنى بلا إشعارات.")
    logger.warn("  السبب: com.rahalgo.rep غير مسجّل في Firebase.")
    logger.warn("  العلاج: سجّله في rahalgo-prod، ثمّ ضع ملفّه الجديد")
    logger.warn("          في mobile/app-rep/google-services.json")
    logger.warn("═".repeat(58))
}

// **مفتاحُ الرفع — من ملفٍّ خارج المستودع** (كما في تطبيق السائق).
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}

android {
    namespace = "com.rahalgo.rep"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.rahalgo.rep"
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
            // ══════════════════════════════════════════════════════
            // **وجدولُ الرموز — مضبوطٌ ولا يُنتج شيئاً اليوم**
            // ══════════════════════════════════════════════════════
            //
            // (تحذيرُ بلاي ٢٠٢٦-٠٨-٢٧ على الإصدار ٧: «يحتوي App Bundle
            //  على رموز برمجية أصلية، ولم يتم تحميل أي رموز لتصحيح
            //  الأخطاء».)
            //
            // **وقِيست المكتباتُ الثلاثُ في الحزمة** (٢٠٢٦-٠٨-٢٧):
            //
            //   libmaplibre.so                  ١٢٫٤ م.ب   .dynsym فقط
            //   libandroidx.graphics.path.so     ٩٫٩ ك.ب   .dynsym فقط
            //   libdatastore_shared_counter.so   ٦٫٩ ك.ب   .dynsym فقط
            //
            // **كلُّها مجرَّدةٌ عند من بناها** — لا `.symtab` ولا
            // `.debug_info`. **ونحن لا نبنيها**: تأتي جاهزةً في
            // `aar` من مبتدعيها.
            //
            // **فالمهمّةُ تعمل ولا تجد ما تستخرجه**، والتحذيرُ يبقى —
            // **وليس بيدنا رفعُه** ما لم يُصدر مبتدعُ المكتبة رموزَها.
            //
            // # ولماذا يبقى السطرُ إذاً
            //
            // **لأنّه يصير عاملاً يومَ تدخل مكتبةٌ نبنيها نحن** — ومن
            // حذفه اليومَ لن يتذكّر أن يُعيده غداً، **فيضيع أوّلُ
            // انهيارٍ أصليٍّ نملك أن نقرأه.**
            //
            // **و SYMBOL_TABLE لا FULL**: أسماءُ الدوالّ تكفي، وأرقامُ
            // الأسطر تضخّم ملفَّ الرفع بلا مقابل.
            ndk { debugSymbolLevel = "SYMBOL_TABLE" }
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
