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
        // **ونصُّ الجذر** — **يقرؤه `StagingFirebaseGuardTest` في `:ui`.**
        //
        // **وغرادل لا يعلم أنّ فحصاً في وحدةٍ يفتح نصَّ الجذر** —
        // **فيقول «محدّثة» بعد أن يُنزَع الحارسُ منه، ولا تعمل.**
        // **وهي عينُ العلّة المشروحة أعلاه.**
        inputs.file(rootProject.file("build.gradle.kts"))
            .withPropertyName("rootBuildScript")
            .withPathSensitivity(PathSensitivity.RELATIVE)

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

// ======================================================================
//  **حارسُ فايربيس لبناء `P-8` — يُسقط البناءَ ولا يحذّر** (٢٠٢٦-٠٩-١٦)
// ======================================================================
//
// # العطبُ المقيس
//
// **وكلُّ تطبيقٍ من الأربعة يُطبّق إضافةَ فايربيس بشرط**:
//
//     val firebaseReady = file("google-services.json").let {
//         it.exists() && it.readText().contains("com.rahalgo.<س>")
//     }
//
// **فإن لم يصحّ الشرطُ لم تُطبَّق الإضافةُ ولا `crashlytics`** —
// **ويُطبَع تحذيرٌ في السجلّ ويمضي البناءُ ناجحاً.**
//
// **وقِيس ٢٠٢٦-٠٩-١٦ بتنحية ملفّ المندوب**: **البناءُ لم يسقط**،
// **وتبخّرت مهمّةُ `processDebugGoogleServices` كلُّها** — «task
// not found in project».
//
// **فينتج أثرٌ بلا دفعٍ أصلاً** — **يُنصَّب ويعمل وتُجرَّب الإشعاراتُ
// فلا يصل شيء**، **ويُقرأ ذلك عيباً في المنصّة وهو نقصُ إعداد.**
// **والتحذيرُ في سجلّ بناءٍ لا يراه أحد** — **فصار سقوطاً.**
//
// # ولمَ هنا لا في ملفّات الأربعة
//
// **وسطرٌ يُكتب في أربعة ملفّاتِ بناءٍ يُنسى في الخامس** — **وهي عينُ
// الحجّة التي جمعت مدخلاتِ الحرّاس أعلاه في هذا الملفّ.**
//
// # ولا يُختلَق معرّفُ مشروع
//
// **ومشروعُ التجهيز لم يُنشأ بعد** (٢٠٢٦-٠٩-١٦) — **فالمعرّفُ يُمرَّر
// خاصّيّةً ولا يُكتب ثابتاً هنا.** **وبناءُ `P-8` بلا معرّفٍ يقف**:
// **وقوفٌ صريحٌ خيرٌ من أثرٍ صامتٍ بلا دفع.**
//
// **ولا يُعاد استعمالُ `rahalgo-prod`** — **ومشروعٌ واحدٌ للبيئتين
// يجعل العزلَ محفوظاً بقاعدةِ بياناتٍ لا بالاستحالة.**
//
//     ./gradlew -Prahalgo.p8=true \
//               -Prahalgo.stagingFirebaseProject=<معرّفُ مشروع التجهيز> ...

/** **حزمةُ التصحيح لكلّ تطبيق** — **ولا خامسَ لها.** */
val p8DebugPackages: Map<String, String> = mapOf(
    "app-customer" to "com.rahalgo.customer.debug",
    "app-driver" to "com.rahalgo.driver.debug",
    "app-merchant" to "com.rahalgo.merchant.debug",
    "app-rep" to "com.rahalgo.rep.debug",
)

val p8Build: Boolean =
    ((findProperty("rahalgo.p8") as String?) ?: "false").trim().toBoolean()

val stagingFirebaseProject: String? =
    (findProperty("rahalgo.stagingFirebaseProject") as String?)?.trim()?.takeIf { it.isNotEmpty() }

fun p8Halt(why: String): Nothing = throw GradleException(
    "بناءُ P-8 موقوف — " + why + "\n" +
        "ولا يُبنى أثرُ قبولٍ بلا دفعٍ: يُنصَّب ويعمل ولا يصل إشعارٌ أبداً.",
)

if (p8Build) {
    val wanted = stagingFirebaseProject
        ?: p8Halt("معرّفُ مشروع التجهيز غيرُ مُمرَّر (rahalgo.stagingFirebaseProject).")
    if (wanted == "rahalgo-prod") {
        p8Halt("مشروعُ الإنتاج لا يصلح للتجهيز — والعزلُ يسقط بمشروعٍ واحد.")
    }
    p8DebugPackages.forEach { (module, debugPackage) ->
        val cfg = file(module + "/src/debug/google-services.json")
        if (!cfg.isFile) {
            p8Halt("لا ملفَّ تجهيزٍ لـ" + module + ": " + cfg.path)
        }
        val root = runCatching { groovy.json.JsonSlurper().parse(cfg) as Map<*, *> }
            .getOrElse { p8Halt("ملفُّ " + module + " ليس JSON صالحاً: " + it.message) }

        val info = root["project_info"] as? Map<*, *>
            ?: p8Halt("لا project_info في ملفّ " + module)
        val projectId = (info["project_id"] as? String)?.trim().orEmpty()
        if (projectId != wanted) {
            p8Halt("مشروعُ " + module + " = " + projectId + " والمنتظَرُ " + wanted)
        }
        if ((info["project_number"] as? String)?.trim().isNullOrEmpty()) {
            p8Halt("لا project_number في ملفّ " + module)
        }

        val clients = root["client"] as? List<*>
            ?: p8Halt("لا تسجيلاتِ تطبيقٍ في ملفّ " + module)
        fun androidInfo(c: Any?): Map<*, *>? =
            ((c as? Map<*, *>)?.get("client_info") as? Map<*, *>)
                ?.get("android_client_info") as? Map<*, *>
        val packages = clients.mapNotNull { androidInfo(it)?.get("package_name") as? String }
        if (!packages.contains(debugPackage)) {
            p8Halt(
                "حزمةُ " + debugPackage + " غيرُ مسجّلةٍ في ملفّ " + module +
                    " (الموجودُ: " + packages.joinToString(" · ") + ")",
            )
        }
        // **ولا حزمةَ إنتاجٍ في ملفّ تجهيز** — **وملفٌّ يعرف الحزمتَين
        // بابٌ لخلطٍ لا داعيَ له.**
        val production = packages.filter { it.startsWith("com.rahalgo.") && !it.endsWith(".debug") }
        if (production.isNotEmpty()) {
            p8Halt("ملفُّ تجهيزِ " + module + " يذكر حزمةَ إنتاج: " + production.joinToString(" · "))
        }

        val appId = clients.firstNotNullOfOrNull { c ->
            if (androidInfo(c)?.get("package_name") as? String == debugPackage) {
                ((c as? Map<*, *>)?.get("client_info") as? Map<*, *>)?.get("mobilesdk_app_id") as? String
            } else {
                null
            }
        }?.trim().orEmpty()
        if (appId.isEmpty()) {
            p8Halt("لا mobilesdk_app_id لحزمة " + debugPackage + " في ملفّ " + module)
        }
        logger.lifecycle("فايربيس P-8: " + module + " ⇐ " + projectId + " · " + debugPackage)
    }
}
