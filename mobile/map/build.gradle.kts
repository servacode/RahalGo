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
    // ══════════════════════════════════════════════════════════════════
    // **وقراءةُ موضع الجهاز هنا لا في كلّ تطبيق**
    // ══════════════════════════════════════════════════════════════════
    //
    // **كان `Here` منسوخاً في الزبون وفي المندوب** — ملفّان متطابقان.
    // **وثالثٌ في تطبيق المتجر يجعلها ثلاثةً تفترق يومَ يُصلَح أحدُها.**
    //
    // **ولا حجمَ يُضاف**: الثلاثةُ يعلنون `play-services-location`
    // لأنفسهم أصلاً.
    api(libs.play.services.location)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    // **ورجوعُ النظام يُلتقط هنا** — حُذف زرُّ «إلغاء» من شاشة الخريطة
    // (قرارُ المالك ٢٠٢٦-٠٨-١٨)، **فلو لم يُلتقط الرجوعُ لَخرج صاحبُه
    // من التطبيق كلِّه ليغلق خريطة.**
    implementation(libs.androidx.activity.compose)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)

    // ══════════════════════════════════════════════════════════════
    // **اختباراتُ طبقة البيانات** — المرحلة ٦ب، البند ٤٩
    // ══════════════════════════════════════════════════════════════
    //
    // **أمرُ المالك**: «حتى بدون جهاز أريد اختبار: PMTiles local URI
    // generation · package storage · download · range resume · SHA ·
    // atomic install · manifest cache · source resolution…».
    //
    // **وكلُّها في `map/data` بلا تبعيّةِ أندرويد** — إلّا `org.json`
    // وهي في `android.jar` **فارغةَ التنفيذ في اختبارات JVM**
    // (`method not mocked`). **فتُجلب النسخةُ الحقيقيّة للاختبار.**
    testImplementation(libs.junit)
    testImplementation(libs.json)
}
