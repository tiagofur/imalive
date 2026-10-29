// Tranqui Android app (ADR-0002: native Kotlin + Compose).
// Dependency and plugin versions live exclusively in gradle/libs.versions.toml
// (docs/stack/android.md): pin updates land as version-bump PRs.
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.ksp) apply false
    alias(libs.plugins.hilt) apply false
    alias(libs.plugins.ktlint) apply false
    alias(libs.plugins.detekt) apply false
}
