// **جذر بلا شيفرة** — الإضافات تُعلن هنا ولا تُطبَّق، فتُطبَّق في الوحدات.
plugins {
    alias(libs.plugins.android.application) apply false
    // **وتُعلَن هنا وإن لم تُطبَّق** — إعلانُ الإضافة في وحدةٍ وحدَها
    // يجعل Gradle يحاول حلَّ إصدارها وهي أصلاً على المسار، **فيسقط
    // البناءُ برسالةٍ عن «إصدارٍ مجهول».**
    alias(libs.plugins.android.library) apply false
    alias(libs.plugins.compose.compiler) apply false
    alias(libs.plugins.kotlin.serialization) apply false
    alias(libs.plugins.google.services) apply false
    alias(libs.plugins.crashlytics) apply false
}

// ══════════════════════════════════════════════════════════════════════
// **حارسُ البنية يقرأ شيفرةَ غيره — فتُعلَن مدخلاتُه** (`TI`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # العطبُ الذي كُشف
//
// **وفحوصُ البنية تفتح ملفّاتِ وحداتٍ أخرى من القرص** — **وغرادل لا
// يعلم بها**: **فيقول «المهمّةُ محدّثة» بعد تعديلٍ في وحدةٍ أخرى،
// ولا تعمل.**
//
// **وقِيس في إغلاق الدفعة السادسة**: **ثلاثةُ شواهدَ سلبيّةٍ بدت
// خضراءَ** — **رُفعت المرحلةُ الأولى من قارئ الموضع، ورُفع سؤالُ
// خدمة النظام، وكُتب الموضعُ قبل فحص دقّته** — **ولم يسقط فحصٌ
// واحد.** **ثمّ بِيست كلُّها بـ`--rerun`.**
//
// **وحارسٌ لا يعمل إلّا إذا تذكّر أحدٌ أن يجبره ليس حارساً** —
// **وخضرةٌ كاذبةٌ أسوأُ من لا فحص**: **الأولى تُطمئن.**
//
// # ولمَ هنا لا في كلّ وحدة
//
// **وسطرٌ يُكتب في أربعة ملفّاتِ بناءٍ يُنسى في الخامس** — **وهو
// عينُ ما وقع في قارئ الموضع**: **ثلاثُ نسخٍ متطابقةٍ افترقت.**
//
// # ولا يُطفأ الفحصُ التزايديّ
//
// **ولا `--rerun` دائمٌ ولا تعطيلٌ عامّ** — **فبناءٌ يُعيد كلَّ شيءٍ
// في كلّ مرّةٍ يُستَبدَل بعد أسبوع.** **بل تُعلَن المدخلاتُ الحقيقيّة
// فيُبطلها تبدّلُها وحدَه.**

// **وما يقرؤه حرّاسُ كلّ وحدةٍ من خارجها** — **بالاسم، ليُقرأ.**
val guardedSources: Map<String, List<String>> = mapOf(
    // `CleartextPolicyTest` · `DrawerCentralPolicyTest` · `MyLocationWiringTest`
    //
    // **والمجلَّدُ كلُّه لا ملفّاتُه** — **لأنّ بعضَ الحراسة نفيٌ**:
    // **«لا إعدادَ أمنِ شبكةٍ في موارد الإصدار»** و**«لا نسخةَ ثانيةً
    // من قارئ الموضع»**. **وملفٌّ يُنشأ لا يُبطل بصمةَ ملفٍّ لم
    // يتبدّل** — **وبصمةُ المجلَّد تتبدّل بإنشائه.**
    "ui" to listOf(
        "app-customer/src",
        "app-driver/src",
        "app-merchant/src",
        "app-rep/src",
        "map/src/main",
    ),
    // `BusinessArrivalTest` · `RouteChoiceHealthTest` — **تقرآن نموذجَ
    // طلبات السائق.** **ولا تعتمد وحدةُ الملاحة على تطبيق السائق
    // أصلاً** (بل العكس): **فلا شيءَ كان يُبطلهما البتّة.**
    "driver-navigation" to listOf("app-driver/src/main/kotlin"),
)

subprojects {
    tasks.withType<Test>().configureEach {
        // **ونصُّ بناءِ الوحدة** — **يقرؤه `ProductionEndpointGuardTest`
        // ليمنع أن يُبنى إصدارُ إنتاجٍ يشير إلى التجهيز.**
        //
        // **وهو أخطرُها**: **تبديلُ عنوانِ الإصدار لا يمسّ `BuildConfig`
        // في التصحيح** — **فلا يُعاد بناءٌ ولا يعمل الحارس.**
        val script = project.file("build.gradle.kts")
        if (script.exists()) {
            inputs.file(script)
                .withPropertyName("moduleBuildScript")
                .withPathSensitivity(PathSensitivity.RELATIVE)
        }

        // **وشيفرةُ الوحدة نفسِها** — **وأكثرُها يُبطَل بإعادة الترجمة**،
        // **إلّا تعديلاً لا يغيّر شيفرةَ الآلة** (تعليقاً أو ترتيباً):
        // **وحارسُ نصٍّ يقرأ النصَّ لا الآلة.**
        val main = project.file("src/main")
        if (main.exists()) {
            inputs.dir(main)
                .withPropertyName("moduleMainSources")
                .withPathSensitivity(PathSensitivity.RELATIVE)
        }

        guardedSources[project.name].orEmpty().forEachIndexed { i, rel ->
            val dir = rootProject.file(rel)
            if (dir.exists()) {
                inputs.dir(dir)
                    .withPropertyName("guardedSource-" + i)
                    .withPathSensitivity(PathSensitivity.RELATIVE)
            }
        }
    }
}
