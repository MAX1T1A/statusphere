plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

layout.buildDirectory = rootProject.layout.projectDirectory.dir("../unsynced/android/build/app")

android {
    namespace = "app.statusphere"
    compileSdk = 36
    ndkVersion = "30.0.16248370"

    val releaseVersionName = findProperty("releaseVersionName") as String?
    fun versionCodeFor(version: String): Int {
        val parts = version.substringBefore("-").split(".")
        val major = parts.getOrNull(0)?.toIntOrNull() ?: 0
        val minor = parts.getOrNull(1)?.toIntOrNull() ?: 0
        val patch = parts.getOrNull(2)?.toIntOrNull() ?: 0
        return major * 10000 + minor * 100 + patch
    }

    signingConfigs {
        create("release") {
            val keystoreFile = System.getenv("STATUSPHERE_KEYSTORE_FILE")
            if (keystoreFile != null) {
                storeFile = file(keystoreFile)
                storePassword = System.getenv("STATUSPHERE_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("STATUSPHERE_KEY_ALIAS")
                keyPassword = System.getenv("STATUSPHERE_KEY_PASSWORD")
            }
        }
    }

    defaultConfig {
        applicationId = "app.statusphere"
        minSdk = 26
        targetSdk = 36
        versionCode = releaseVersionName?.let { versionCodeFor(it) } ?: 1
        versionName = releaseVersionName ?: "0.1.0"
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("release")
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }
}

composeCompiler {
    stabilityConfigurationFiles.add(layout.projectDirectory.file("compose-stability.conf"))
}

val clientDir = rootProject.layout.projectDirectory.dir("../client")
val goBin = providers.exec { commandLine("go", "env", "GOPATH") }
    .standardOutput.asText.map { "${it.trim()}/bin" }
val sdkDir = androidComponents.sdkComponents.sdkDirectory
val ndkDir = androidComponents.sdkComponents.ndkDirectory

val bindCore = tasks.register<Exec>("bindCore") {
    val aar = layout.buildDirectory.file("core/statusphere-core.aar")
    inputs.files(fileTree(clientDir) { include("**/*.go", "go.mod", "go.sum") })
        .withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.file(aar)
    workingDir(clientDir)
    val bin = goBin.get()
    executable = "$bin/gomobile"
    environment("PATH", "$bin:${providers.environmentVariable("PATH").get()}")
    environment("ANDROID_HOME", sdkDir.get().asFile.path)
    environment("ANDROID_NDK_HOME", ndkDir.get().asFile.path)
    args(
        "bind",
        "-target=android/arm64,android/amd64",
        "-androidapi", "26",
        "-javapkg", "app.statusphere",
        "-o", aar.get().asFile.path,
        "./mobile",
    )
}

tasks.named("preBuild") { dependsOn(bindCore) }

dependencies {
    implementation(files(bindCore))
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.material3)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.core)
    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.glance.appwidget)
    implementation(libs.play.services.code.scanner)
}
