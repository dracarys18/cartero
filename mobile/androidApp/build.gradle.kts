plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.baselineprofile)
}

val releaseKeystore = providers.environmentVariable("CARTERO_KEYSTORE").orNull

android {
    namespace = "app.cartero"
    compileSdk = 37

    defaultConfig {
        applicationId = "app.cartero"
        minSdk = 33
        targetSdk = 37
        versionCode = providers.gradleProperty("versionCode").get().toInt()
        versionName = providers.gradleProperty("versionName").get()
        ndk {
            abiFilters += "arm64-v8a"
        }
    }

    androidResources {
        localeFilters += "en"
        ignoreAssetsPatterns += "PublicSuffixDatabase.list"
    }

    signingConfigs {
        if (releaseKeystore != null) {
            create("release") {
                storeFile = file(releaseKeystore)
                storePassword = providers.environmentVariable("CARTERO_KEYSTORE_PASSWORD").get()
                keyAlias = providers.environmentVariable("CARTERO_KEY_ALIAS").get()
                keyPassword = providers.environmentVariable("CARTERO_KEY_PASSWORD").get()
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            signingConfig = signingConfigs.findByName("release")
        }
    }

    buildFeatures {
        compose = true
    }

    packaging {
        resources.excludes += listOf(
            "/META-INF/androidx/**",
            "/META-INF/*.version",
            "/META-INF/*.kotlin_module",
            "/META-INF/**/LICENSE*",
            "/META-INF/{AL2.0,LGPL2.1}",
            "/kotlin/**",
            "DebugProbesKt.bin",
        )
    }

    dependenciesInfo {
        includeInApk = false
        includeInBundle = false
    }
}

baselineProfile {
    dexLayoutOptimization = false
}

dependencies {
    implementation(project(":shared"))
    implementation(libs.activity.compose)
    implementation(libs.profileinstaller)
    baselineProfile(project(":baselineprofile"))
}
