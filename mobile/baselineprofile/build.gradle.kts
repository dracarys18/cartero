plugins {
    alias(libs.plugins.android.test)
    alias(libs.plugins.baselineprofile)
}

android {
    namespace = "app.cartero.baselineprofile"
    compileSdk = 37

    defaultConfig {
        minSdk = 33
        targetSdk = 37
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    targetProjectPath = ":androidApp"
}

baselineProfile {
    useConnectedDevices = true
}

dependencies {
    implementation(libs.junit.ext)
    implementation(libs.uiautomator)
    implementation(libs.benchmark.macro)
}
