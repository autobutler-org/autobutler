import org.gradle.api.tasks.compile.JavaCompile

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
// Flutter's plugin installs its NDK by calling sdkmanager once while :app is
// configured, which is what evaluationDependsOn does below. build-android on
// #3311 died there: sdkmanager cached a truncated archive ("Error on ZipFile
// unknown archive") and that single call failed the build. Install it first,
// dropping the bad archive between tries. The plugin then calls sdkmanager
// again, which skips a package that is already installed.
val androidSdkRoot = findProperty("flutter.androidSdkRoot")?.toString()?.takeIf { it.isNotBlank() }
val sdkManagerPath = findProperty("flutter.sdkManagerPath")?.toString()?.takeIf { it.isNotBlank() }
// Flutter pins this in packages/flutter_tools/lib/src/android/gradle_utils.dart
// and copies it onto FlutterExtension.ndkVersion. The app module requests that
// same value, so the install has to use it too.
val flutterNdkVersion =
    rootProject.file("local.properties").takeIf { it.isFile }?.let { localProperties ->
        val properties = java.util.Properties()
        localProperties.inputStream().use { properties.load(it) }
        val flutterSdk = properties.getProperty("flutter.sdk")
        val gradleUtils =
            flutterSdk?.let {
                java.io.File(it, "packages/flutter_tools/lib/src/android/gradle_utils.dart")
            }
        val version =
            gradleUtils
                ?.takeIf { it.isFile }
                ?.readText()
                ?.let { Regex("""const ndkVersion = '([^']+)'""").find(it)?.groupValues?.get(1) }
        if (version == null && !flutterSdk.isNullOrBlank()) {
            logger.warn(
                "quark: could not read Flutter's NDK version from $flutterSdk; " +
                    "leaving the NDK install to the Flutter Gradle plugin",
            )
        }
        version
    }
if (androidSdkRoot != null && sdkManagerPath != null && flutterNdkVersion != null) {
    val installed =
        findProperty("flutter.installedNdkVersions")
            ?.toString()
            .orEmpty()
            .split(',')
            .map { it.trim() }
            .filter { it.isNotEmpty() }
    if (flutterNdkVersion !in installed) {
        val script = rootProject.projectDir.parentFile.resolve("scripts/ensure-android-ndk.bash")
        // providers.exec captures the process streams, and a non-zero exit only
        // surfaces once the result is read. Print both, then fail the build.
        val install =
            providers.exec {
                commandLine(
                    "bash",
                    script.absolutePath,
                    androidSdkRoot,
                    sdkManagerPath,
                    flutterNdkVersion,
                )
                isIgnoreExitValue = true
            }
        val installOut = install.standardOutput.asText.get()
        val installErr = install.standardError.asText.get()
        if (installOut.isNotBlank()) {
            logger.lifecycle(installOut.trimEnd())
        }
        if (installErr.isNotBlank()) {
            logger.lifecycle(installErr.trimEnd())
        }
        val exitValue = install.result.get().exitValue
        if (exitValue != 0) {
            throw GradleException(
                "Installing Android NDK $flutterNdkVersion failed with exit $exitValue",
            )
        }
    }
}

subprojects {
    project.evaluationDependsOn(":app")
}

subprojects {
    // Some third-party Android plugins still default to Java 8 and trigger
    // obsolete-source warnings with newer JDKs. Normalize all Java compiles.
    tasks.withType<JavaCompile>().configureEach {
        sourceCompatibility = JavaVersion.VERSION_17.toString()
        targetCompatibility = JavaVersion.VERSION_17.toString()
    }
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
