// ══════════════════════════════════════════════════════════════════════
//  **جذر تطبيقات رحّال غو — مركز واحد وتطبيقات فوقه**
// ══════════════════════════════════════════════════════════════════════
//
// (قرار المالك ٢٠٢٦-٠٨-١١، و`docs/GROUND-RULES.md` §7.)
//
// **الوحدات تُضاف حين يوجد ما يوضع فيها** — لا مجلدات فارغة من اليوم
// الأول. **والبنية النهائية مقررة ومكتوبة** في القواعد، فلا يُنقل شيء
// لاحقا:
//
//   shared/        القلب — لا يعرف أندرويد ✓
//   design/        التوكنز والمكوّنات ✓
//   core-android/  ما يلمس النظام (يُضاف مع الموقع والإشعارات)
//   app-*/         شاشات فقط

pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    // **ولا مستودع يُعرَّف في وحدة** — مصدر واحد للتبعيات كلها، **ووحدة
    // تجلب من مستودع خاص بها تُدخل إلى المشروع ما لا يراه أحد.**
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "rahalgo"
include(":shared")
include(":design")
include(":app-driver")
