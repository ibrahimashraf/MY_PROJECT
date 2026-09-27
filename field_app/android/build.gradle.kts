allprojects {
    repositories {
        google()
        mavenCentral()
    }
    configurations.all {
        resolutionStrategy {
            force("org.jetbrains.kotlin:kotlin-stdlib:2.2.10")
            force("org.jetbrains.kotlin:kotlin-reflect:2.2.10")
        }
    }
    buildscript {
        configurations.all {
            resolutionStrategy {
                force("org.jetbrains.kotlin:kotlin-stdlib:2.2.10")
                force("org.jetbrains.kotlin:kotlin-reflect:2.2.10")
            }
        }
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
subprojects {
    project.evaluationDependsOn(":app")
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
