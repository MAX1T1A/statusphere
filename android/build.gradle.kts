plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
}

layout.buildDirectory = layout.projectDirectory.dir("../unsynced/android/build/root")
