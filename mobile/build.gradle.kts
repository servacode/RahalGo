// **جذر بلا شيفرة** — الإضافات تُعلن هنا ولا تُطبَّق، فتُطبَّق في الوحدات.
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.compose.compiler) apply false
}
