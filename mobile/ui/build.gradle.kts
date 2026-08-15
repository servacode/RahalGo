// ══════════════════════════════════════════════════════════════════════
//  **وحدةُ الواجهة — قطعُ الشاشات لأربعة تطبيقات**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٤: «هناك الكثيرُ من الأمور المشتركة بين كلّ
//  التطبيقات، يجب أن تكون مركزيّةً فلا نبنيها بكلّ تطبيق».)
//
// # ولماذا وحدةٌ ثالثةٌ لا ثانية
//
// **`design` تعرف الألوانَ ولا تعرف الشبكة** — والفقاعةُ تحتاج رسالةً،
// وصندوقُ الإشعارات يحتاج إشعاراً. **ولو حُقنت النماذجُ في `design`
// لَصار الثيمُ يعرف المحرّك.**
//
// **و`shared` لا تعرف أندرويد** (`GROUND-RULES.md` §7.2) — والجلسةُ
// تحتاج `Context`، والانهياراتُ تحتاج `Application`.
//
// **فبينهما موضعٌ ثالث**: يعرف الاثنين ولا يعرف أيَّ تطبيق.
//
// # وما لا يدخلها
//
// **ما فيه اسمُ دورٍ واحد** — الرحلةُ والورديّةُ والطابور. **وقطعةٌ
// تعرف «السائق» لا تُستعمل في تطبيق الزبون** ولو نُقلت إلى هنا.
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "com.rahalgo.ui"
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
    // **و`api` لا `implementation`** — التطبيقاتُ تستعمل الألوانَ
    // والنماذجَ نفسَها، **ولو أُخفيت لَأعلنها كلُّ تطبيقٍ من جديد.**
    api(project(":design"))
    api(project(":shared"))
    implementation(libs.androidx.core.ktx)
    // ══════════════════════════════════════════════════════════════════
    // **صورُ السوق — `api` لا `implementation`**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وتُعلَن مرّةً هنا** — لا في كلّ تطبيق: **أربعُ إعلاناتٍ بأربع
    // نسخٍ تفترق يومَ تُرفع واحدةٌ منها.**
    api(libs.coil.compose)
    api(libs.coil.network)
    // **ومُلتقِطُ الرجوع** — شاشةٌ تغطّي تعود خطوةً لا تُغلق التطبيق.
    implementation(libs.androidx.activity.compose)
    // **وخزنُ الجلسة مشفَّرٌ** — والمفتاحُ في مخزن المفاتيح لا في ملفّ.
    implementation(libs.androidx.security.crypto)
    // **وتقاريرُ الانهيار** — غلافُها هنا، والتطبيقُ يُشغّلها بإقلاعه.
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.crashlytics)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    debugImplementation(libs.compose.ui.tooling)
    implementation(libs.compose.ui.tooling.preview)
}
