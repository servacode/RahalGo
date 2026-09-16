// ══════════════════════════════════════════════════════════════════════
//  **تطبيق المتجر — شاشات فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قاعدة `GROUND-RULES.md` §7.1: لا شبكةَ ولا منطقَ هنا — كلُّها في
//  `shared` و`ui`.)
//
// **والإشعاراتُ وصلت ٢٠٢٦-٠٨-٢٥**: سُجّلت الحزمتان في `rahalgo-prod`
// ونُزّل `google-services.json`. **ومنطقُ الإشعار كلُّه في
// `:ui/push/RahalPushService`** — وهنا خمسةُ أسطرٍ تخصّ المتجر.
//
// **وكان المتجرُ لا يعلم بطلبٍ إلّا إن كان التطبيقُ مفتوحاً** — وطلبٌ
// لا يُنبَّه به يبرد ويُلغى.
import java.util.Properties
import org.gradle.api.tasks.PathSensitivity

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
    // **وإضافةُ غوغل تقرأ `google-services.json`** — وتُسقط البناءَ إن
    // لم تجد اسمَ الحزمة فيه، **فلا يمرّ بناءٌ بإعدادٍ ناقص.**
    alias(libs.plugins.google.services)
    // ══════════════════════════════════════════════════════════════════
    // **وإضافةُ Crashlytics لازمةٌ متى وُجدت مكتبتُها**
    // ══════════════════════════════════════════════════════════════════
    //
    // **و`:ui` تُعلن `firebase-crashlytics` للأربعة** — فمن أضاف
    // `google-services` بلا هذه **سقط تطبيقُه عند أوّل إقلاع**:
    //
    //   The Crashlytics build ID is missing.
    //
    // **وقِيس على الجهاز 2026-08-26**: أُضيفت `google-services` وحدَها
    // فانهار المتجرُ قبل أن تُرسم شاشة. **والانهيارُ في مزوّد محتوى،
    // فلا شاشةَ خطأٍ ولا سطرَ سبب — يُغلق التطبيقُ صامتاً.**
    alias(libs.plugins.crashlytics)
}

// **مفتاحُ الرفع — من ملفٍّ خارج المستودع** (كما في تطبيق السائق).
val keystoreProps = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}

// ══════════════════════════════════════════════════════════════════════
// **عنوانُ المحرّك: الإصدارُ حرفيٌّ والتجريبيُّ يقبل تجاوزاً** (`P-8`)
// ══════════════════════════════════════════════════════════════════════
//
// **والنمطُ منقولٌ عن تطبيق السائق حرفاً** — **وهو القائمُ والمُختبَر**،
// **ولا تُبتكَر آليّةٌ ثانيةٌ لتفترقا.**
//
// **والإصدارُ لا يقبل تجاوزاً البتّة**: قيمتاه مكتوبتان حرفاً بحرف في
// `release` أدناه، **ولا تقرآن خاصّيّةً ولا متغيّرَ بيئة.** فمن مرّر
// `-Prahalgo.apiBaseUrl` وبنى إصداراً **لم يتغيّر عنده شيء.**
//
// **ويحرسه `ProductionEndpointGuardTest`** في التطبيقات الأربعة.
private fun overrideOrNull(project: org.gradle.api.Project, key: String): String? =
    (project.findProperty(key) as String?)?.trim()?.takeIf { it.isNotEmpty() }

private fun quoted(value: String) = "\"" + value + "\""

android {
    namespace = "com.rahalgo.merchant"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.rahalgo.merchant"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        // ══════════════════════════════════════════════════════════════
        // **مرشَّحُ الإطلاق الأوّل** (`P-8`، ٢٠٢٦-٠٩-١٤)
        // ══════════════════════════════════════════════════════════════
        //
        // **ولا يُعاد رقمٌ وُزّع**: نسخةٌ بُنيت بالرقم 4 كانت تُثبَّت
        // على أجهزة التجربة — **والتحديثُ فوقها لا يُقاس إن لم يتبدّل
        // الرقم.**
        //
        // **والاسمُ رفعةٌ صغرى**: هذه النسخةُ تحمل سلوكاً جديداً يراه
        // الناس — **هويّةَ هاتفٍ سوريّةً · وأبوابَ وضعِ إطلاقٍ · وحالاتٍ
        // فارغةً مكتوبةً · وأحوالَ حدِّ توصيلٍ مفترقةً · وقيدَ كلمةٍ
        // مؤقّتة.**
        versionCode = 5
        versionName = "0.3.0"
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
            // **العنوانان حرفيّان — ولا مدخلَ لتجاوزٍ هنا** (`P-8`).
            buildConfigField("String", "API_BASE_URL", quoted("https://api.rahalgo.com"))
            buildConfigField("String", "MAPS_BASE_URL", quoted("https://maps.rahalgo.com"))
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
            // **وهنا وحدَه يُقبل التجاوز** — وافتراضُه الإنتاجُ نفسُه،
            // **فمن بنى تجريبيّاً بلا خاصّيّةٍ لم يتغيّر عنده شيء.**
            buildConfigField(
                "String", "API_BASE_URL",
                // ══════════════════════════════════════════════════════════
                // **وافتراضُ التصحيح مضيفُ التجهيز لا الإنتاج** (٢٠٢٦-٠٩-١٦)
                // ══════════════════════════════════════════════════════════
                //
                // **وكان الافتراضُ `api.rahalgo.com`** — **فمن بنى أثرَ
                // قبولٍ ونسي `-P` بنى أثراً يكلّم الإنتاج وهو يظنّه
                // تجهيزاً.** **والنسيانُ لا يُحرَس بالتذكير.**
                //
                // **فصار الافتراضُ هو الآمن**: **بناءُ تصحيحٍ بلا تجاوزٍ
                // يكلّم التجهيز**، **والتجاوزُ يبقى للحلقيّ في التطوير.**
                //
                // **والإصدارُ لم يُمَسّ** — **يبقى على الإنتاج حرفاً.**
                quoted(overrideOrNull(project, "rahalgo.apiBaseUrl") ?: "https://staging-api.rahalgo.com"),
            )
            buildConfigField(
                "String", "MAPS_BASE_URL",
                quoted(overrideOrNull(project, "rahalgo.mapsBaseUrl") ?: "https://maps.rahalgo.com"),
            )
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
    // ══════════════════════════════════════════════════════════════════
    // **وحارسُ عزل الإنتاج يحتاج مُشغّلاً** (`P-8`، ٢٠٢٦-٠٩-١٤)
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولم يكن لهذا التطبيق فحصُ وحدةٍ واحد** — **ولا حارسَ عنوانٍ**،
    // لأنّ عنوانَه كان ثابتاً مشتركاً لا يُبدَّل. **ويومَ فُتح مدخلُ
    // التجاوز للتجربة لزم الحارسُ.**
    testImplementation(libs.junit)

}

