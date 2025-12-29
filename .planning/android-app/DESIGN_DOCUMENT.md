# Sovereign Android App - Design Document

**TEAM_033** | Created: 2025-12-29

## Executive Summary

This document outlines the architecture for migrating the current Go + Bash VM management approach to a native Android app using Google's VirtualizationService API. This provides proper Android integration, eliminates routing hacks, and creates a robust, user-facing application.

---

## 1. Analysis of Google's Terminal App

### 1.1 Key Components

| Component | Purpose |
|-----------|---------|
| `VmLauncherService.kt` | Foreground service that manages VM lifecycle |
| `Runner.kt` | Creates and runs VirtualMachine with callback handling |
| `ConfigJson.kt` | Parses JSON config into `VirtualMachineConfig` |
| `InstalledImage.kt` | Manages disk images (resize, backup, truncate) |
| `DebianServiceImpl.kt` | gRPC service for VM-to-host communication |
| `PortNotifier.kt` | Notification-based port forwarding UI |

### 1.2 Key Android APIs Used

```kotlin
// Core VM APIs
import android.system.virtualmachine.VirtualMachine
import android.system.virtualmachine.VirtualMachineCallback
import android.system.virtualmachine.VirtualMachineConfig
import android.system.virtualmachine.VirtualMachineCustomImageConfig
import android.system.virtualmachine.VirtualMachineManager
```

### 1.3 Required Permissions

```xml
<uses-permission android:name="android.permission.MANAGE_VIRTUAL_MACHINE" />
<uses-permission android:name="android.permission.USE_CUSTOM_VIRTUAL_MACHINE" />
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_SPECIAL_USE"/>
<uses-permission android:name="android.permission.POST_NOTIFICATIONS"/>

<uses-feature android:name="android.software.virtualization_framework" android:required="true" />
```

### 1.4 How Google Creates Custom Linux VMs

```kotlin
// 1. Parse JSON config
val json = ConfigJson.from(context, image.configPath)
val configBuilder = json.toConfigBuilder(context)
val customImageConfigBuilder = json.toCustomImageConfigBuilder(context)

// 2. Build VirtualMachineConfig
customImageConfigBuilder
    .setName("debian")
    .setKernelPath(kernelPath)
    .setInitrdPath(initrdPath)
    .useNetwork(true)  // THIS IS KEY - enables proper networking
    .addDisk(VirtualMachineCustomImageConfig.Disk.RWDisk(rootPartPath))

configBuilder.setCustomImageConfig(customImageConfigBuilder.build())
val config = configBuilder.build()

// 3. Get VirtualMachineManager and create VM
val vmm = context.getSystemService(VirtualMachineManager::class.java)
val vm = vmm.getOrCreate("debian", config)

// 4. Set callback and run
vm.setCallback(executor, object : VirtualMachineCallback {
    override fun onStopped(vm: VirtualMachine, reason: Int) { ... }
    override fun onError(vm: VirtualMachine, errorCode: Int, message: String) { ... }
})
vm.run()
```

---

## 2. Sovereign App Architecture

### 2.1 Package Structure

```
com.sovereign.vault/
├── app/
│   ├── SovereignApplication.kt
│   └── MainActivity.kt
├── service/
│   ├── VmLauncherService.kt          # Foreground service for VM lifecycle
│   ├── PostgreSQLService.kt          # PostgreSQL-specific management
│   └── ForgejoService.kt             # Forgejo-specific management
├── vm/
│   ├── VmRunner.kt                   # Creates and runs VirtualMachine
│   ├── VmConfig.kt                   # VM configuration (JSON parsing)
│   ├── VmImage.kt                    # Disk image management
│   └── VmCallback.kt                 # VM lifecycle callbacks
├── network/
│   ├── TailscaleManager.kt           # Tailscale integration
│   └── PortForwarder.kt              # Port forwarding via vsock
├── data/
│   ├── VaultRepository.kt            # Password vault data access
│   └── SecureStorage.kt              # Encrypted credential storage
├── ui/
│   ├── dashboard/                    # VM status dashboard
│   ├── settings/                     # App settings
│   └── components/                   # Reusable UI components
└── util/
    ├── Logger.kt
    └── Extensions.kt
```

### 2.2 VM Configuration (JSON Format)

```json
{
    "name": "sovereign-sql",
    "kernel": "$APP_DATA_DIR/vm/sql/vmlinuz",
    "initrd": "$APP_DATA_DIR/vm/sql/initrd.img",
    "params": "root=/dev/vda1 console=ttyS0 init=/init.sh",
    "protected": false,
    "cpu_topology": "match_host",
    "memory_mib": 2048,
    "debuggable": true,
    "connect_console": true,
    "console_out": true,
    "network": true,
    "disks": [
        {
            "writable": true,
            "partitions": [
                {
                    "label": "ROOT",
                    "path": "$APP_DATA_DIR/vm/sql/rootfs.img",
                    "writable": true
                }
            ]
        },
        {
            "writable": true,
            "image": "$APP_DATA_DIR/vm/sql/data.img"
        }
    ]
}
```

### 2.3 Key Classes

#### VmRunner.kt
```kotlin
class VmRunner private constructor(val vm: VirtualMachine, callback: Callback) {
    val exitStatus: CompletableFuture<Boolean> = callback.finished

    private class Callback : VirtualMachineCallback {
        val finished = CompletableFuture<Boolean>()
        
        override fun onStopped(vm: VirtualMachine, reason: Int) {
            finished.complete(true)
        }
        
        override fun onError(vm: VirtualMachine, errorCode: Int, message: String) {
            finished.complete(false)
        }
    }

    companion object {
        fun create(context: Context, config: VirtualMachineConfig): VmRunner {
            val vmm = context.getSystemService(VirtualMachineManager::class.java)
            val name = config.customImageConfig?.name 
                ?: throw IllegalArgumentException("VM name required")
            
            var vm = vmm.getOrCreate(name, config)
            try {
                vm.config = config
            } catch (e: VirtualMachineException) {
                vmm.delete(name)
                vm = vmm.create(name, config)
            }
            
            val cb = Callback()
            vm.setCallback(ForkJoinPool.commonPool(), cb)
            vm.run()
            return VmRunner(vm, cb)
        }
    }
}
```

#### VmConfig.kt
```kotlin
data class VmConfig(
    val name: String,
    val kernel: String,
    val initrd: String,
    val params: String,
    val protected: Boolean = false,
    val cpuTopology: String = "match_host",
    val memoryMib: Int = 2048,
    val debuggable: Boolean = true,
    val network: Boolean = true,
    val disks: List<DiskConfig> = emptyList()
) {
    fun toVirtualMachineConfig(context: Context): VirtualMachineConfig {
        val customBuilder = VirtualMachineCustomImageConfig.Builder()
            .setName(name)
            .setKernelPath(resolvePath(kernel, context))
            .setInitrdPath(resolvePath(initrd, context))
            .useNetwork(network)
        
        params.split(" ").filter { it.isNotEmpty() }.forEach { 
            customBuilder.addParam(it) 
        }
        
        disks.forEach { disk ->
            customBuilder.addDisk(disk.toConfig(context))
        }
        
        return VirtualMachineConfig.Builder(context)
            .setProtectedVm(protected)
            .setMemoryBytes(memoryMib.toLong() * 1024 * 1024)
            .setCpuTopology(getCpuTopology())
            .setDebugLevel(if (debuggable) DEBUG_LEVEL_FULL else DEBUG_LEVEL_NONE)
            .setCustomImageConfig(customBuilder.build())
            .build()
    }
}
```

---

## 3. Migration Strategy

### Phase 1: Project Setup (Week 1)
- [ ] Create Android project with Kotlin
- [ ] Configure Gradle for API 34+ (Android 14)
- [ ] Add VirtualizationService dependencies
- [ ] Set up permission handling

### Phase 2: Core VM Management (Week 2)
- [ ] Implement VmConfig JSON parsing
- [ ] Implement VmRunner with lifecycle callbacks
- [ ] Implement VmImage disk management
- [ ] Create VmLauncherService foreground service

### Phase 3: Port Existing VM Images (Week 2-3)
- [ ] Convert current Alpine rootfs to app-compatible format
- [ ] Bundle kernel and initrd in app assets
- [ ] Implement image extraction on first launch
- [ ] Test basic VM boot

### Phase 4: Networking (Week 3)
- [ ] Leverage `useNetwork(true)` for automatic NetworkAgent
- [ ] Implement vsock-based port forwarding (if needed)
- [ ] Test PostgreSQL connectivity
- [ ] Integrate Tailscale (either in-VM or host-side)

### Phase 5: UI & Polish (Week 4)
- [ ] Create dashboard UI with Jetpack Compose
- [ ] Implement settings screens
- [ ] Add notification for VM status
- [ ] Port forwarding configuration UI

---

## 4. Key Differences from Current Approach

| Aspect | Current (Go + Bash) | New (Android App) |
|--------|---------------------|-------------------|
| VM Creation | `adb shell crosvm run ...` | `VirtualMachineManager.create()` |
| Networking | TAP + iptables + ip rules | `useNetwork(true)` → automatic |
| Lifecycle | PID files, shell scripts | `VirtualMachineCallback` |
| Init | Injected bash script | Standard init system |
| Storage | Manual disk management | `VmImage` class + Android APIs |
| Permissions | Root access required | `MANAGE_VIRTUAL_MACHINE` permission |

---

## 5. Permission Acquisition Strategy

Since `MANAGE_VIRTUAL_MACHINE` is a restricted permission:

### Option A: Development (Recommended for Now)
```bash
adb shell pm grant com.sovereign.vault android.permission.MANAGE_VIRTUAL_MACHINE
adb shell pm grant com.sovereign.vault android.permission.USE_CUSTOM_VIRTUAL_MACHINE
```

### Option B: Privileged App (Production)
- Sign APK with platform key
- Install as system app via Magisk module
- App placed in `/system/priv-app/`

### Option C: Request from Google (Long-term)
- Apply for allowlisting if app becomes public

---

## 6. File Locations

```
/data/data/com.sovereign.vault/
├── files/
│   ├── sql/
│   │   ├── vmlinuz              # Kernel
│   │   ├── initrd.img           # Initramfs
│   │   ├── rootfs.img           # Root filesystem
│   │   ├── data.img             # Persistent data
│   │   └── vm_config.json       # VM configuration
│   └── forge/
│       ├── vmlinuz
│       ├── initrd.img
│       ├── rootfs.img
│       ├── data.img
│       └── vm_config.json
├── vm/
│   ├── sql/
│   │   ├── config.xml           # VirtualizationService state
│   │   └── instance.img         # VM instance data
│   └── forge/
│       ├── config.xml
│       └── instance.img
└── shared_prefs/
    └── sovereign_prefs.xml      # App preferences
```

---

## 7. Build System

### Gradle Configuration
```kotlin
android {
    namespace = "com.sovereign.vault"
    compileSdk = 35  // Android 16
    
    defaultConfig {
        applicationId = "com.sovereign.vault"
        minSdk = 34  // Android 14 (API 34) minimum for VirtualizationService
        targetSdk = 35
        versionCode = 1
        versionName = "1.0"
    }
    
    buildFeatures {
        compose = true
    }
}

dependencies {
    // Jetpack Compose
    implementation(platform("androidx.compose:compose-bom:2024.01.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.material3:material3")
    
    // Lifecycle
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.7.0")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.7.0")
    
    // Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3")
    
    // JSON
    implementation("com.google.code.gson:gson:2.10.1")
}
```

---

## 8. Next Steps

1. **Create initial Android project** in `sovereign/android/`
2. **Implement minimal VmRunner** that can boot a test VM
3. **Port existing Alpine rootfs** to work with VirtualizationService
4. **Test networking** with `useNetwork(true)`
5. **Iterate on PostgreSQL/Forgejo integration**

---

## 9. References

- Google Terminal App source: `android.googlesource.com/platform/packages/modules/Virtualization/+/main/android/TerminalApp/`
- VirtualizationService docs: `source.android.com/docs/core/virtualization/virtualization-service`
- AVF Architecture: `source.android.com/docs/core/virtualization/architecture`
