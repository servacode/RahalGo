// **جذر بلا شيفرة** — الإضافات تُعلن هنا ولا تُطبَّق، فتُطبَّق في الوحدات.
plugins {
    alias(libs.plugins.android.application) apply false
    // **وتُعلَن هنا وإن لم تُطبَّق** — إعلانُ الإضافة في وحدةٍ وحدَها
    // يجعل Gradle يحاول حلَّ إصدارها وهي أصلاً على المسار، **فيسقط
    // البناءُ برسالةٍ عن «إصدارٍ مجهول».**
    alias(libs.plugins.android.library) apply false
    alias(libs.plugins.compose.compiler) apply false
    alias(libs.plugins.kotlin.serialization) apply false
}
