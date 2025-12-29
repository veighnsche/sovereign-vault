# Implementation Patterns from Google Terminal App

**TEAM_033** | Created: 2025-12-29

## 1. VM Lifecycle Pattern

### VmLauncherService (Foreground Service)

Google runs the VM in a foreground service to prevent Android from killing it:

```kotlin
class VmLauncherService : Service() {
    private var runner: Runner? = null
    
    override fun onStartCommand(intent: Intent, flags: Int, startId: Int): Int {
        when (intent.action) {
            ACTION_START_VM -> {
                // Start foreground IMMEDIATELY to avoid ANR
                startForeground(hashCode(), notification)
                
                // Do actual work on background thread
                workerThread.execute { doStart(...) }
            }
            ACTION_SHUTDOWN_VM -> {
                workerThread.execute { doShutdown(...) }
            }
        }
        return START_NOT_STICKY
    }
    
    private fun doStart(...) {
        val config = buildConfig()
        runner = Runner.create(context, config)
        
        // Handle exit
        runner.exitStatus.thenAcceptAsync { success ->
            if (success) notifyStop() else notifyError()
            stopSelf()
        }
    }
}
```

**Key insight**: `START_NOT_STICKY` means the service won't restart automatically if killed. This is appropriate for VMs where we want explicit control.

---

## 2. VirtualMachineCallback Pattern

Google uses a simple callback wrapper:

```kotlin
private class Callback : VirtualMachineCallback {
    val finishedSuccessfully = CompletableFuture<Boolean>()

    override fun onPayloadStarted(vm: VirtualMachine) {
        // Only for Microdroid VMs - ignore for custom Linux
    }

    override fun onPayloadReady(vm: VirtualMachine) {
        // Only for Microdroid VMs - ignore for custom Linux
    }

    override fun onPayloadFinished(vm: VirtualMachine, exitCode: Int) {
        // Only for Microdroid VMs - ignore for custom Linux
    }

    override fun onError(vm: VirtualMachine, errorCode: Int, message: String) {
        Log.e(TAG, "Error from VM. code: $errorCode ($message)")
        finishedSuccessfully.complete(false)
    }

    override fun onStopped(vm: VirtualMachine, reason: Int) {
        Log.d(TAG, "VM stopped. Reason: $reason")
        finishedSuccessfully.complete(true)
    }
}
```

**Key insight**: For custom Linux VMs, only `onError` and `onStopped` matter. The payload callbacks are Microdroid-specific.

---

## 3. Config JSON Pattern

Google uses a JSON file with variable substitution:

```kotlin
companion object {
    private fun replaceKeywords(r: Reader, context: Context): String {
        val rules: Map<String, String> = mapOf(
            "\\\$PAYLOAD_DIR" to InstalledImage.getDefault(context).installDir.toString(),
            "\\\$USER_ID" to context.userId.toString(),
            "\\\$PACKAGE_NAME" to context.packageName,
            "\\\$APP_DATA_DIR" to context.dataDir.toString(),
        )

        return BufferedReader(r).useLines { lines ->
            lines.map { line ->
                rules.entries.fold(line) { acc, rule ->
                    acc.replace(rule.key.toRegex(), rule.value)
                }
            }.joinToString("\n")
        }
    }
}
```

**Key insight**: Using `$PAYLOAD_DIR`, `$APP_DATA_DIR` etc. in JSON allows portable configs.

---

## 4. Disk Management Pattern

```kotlin
class InstalledImage(val installDir: Path) {
    private val rootPartition: Path = installDir.resolve("root_part")
    
    fun resize(desiredSize: Long): Long {
        val roundedSize = roundUp(desiredSize)
        val curSize = getApparentSize()
        
        runE2fsck(rootPartition)  // Always check filesystem first
        
        if (roundedSize == curSize) return roundedSize
        
        if (roundedSize > curSize) {
            allocateSpace(roundedSize)
        }
        resizeFilesystem(rootPartition, roundedSize)
        return getApparentSize()
    }
    
    private fun allocateSpace(sizeInBytes: Long): Boolean {
        RandomAccessFile(rootPartition.toFile(), "rw").use { raf ->
            Os.posix_fallocate(raf.fd, 0, sizeInBytes)
        }
        return true
    }
    
    private fun resizeFilesystem(path: Path, sizeInBytes: Long) {
        val sizeArg = "${sizeInBytes / (1024 * 1024)}M"
        runCommand("/system/bin/resize2fs", path.toString(), sizeArg)
    }
}
```

**Key insight**: Use `Os.posix_fallocate()` for space allocation, `/system/bin/e2fsck` and `/system/bin/resize2fs` for filesystem operations.

---

## 5. Service Discovery Pattern (mDNS/NSD)

Google uses NSD to discover when the VM's ttyd service is ready:

```kotlin
private fun getTerminalServiceInfo(): CompletableFuture<NsdServiceInfo> {
    val nsdManager = getSystemService(NsdManager::class.java)
    val queryInfo = NsdServiceInfo().apply {
        serviceType = "_http._tcp"
        serviceName = "ttyd"
    }
    val resolved = CompletableFuture<NsdServiceInfo>()

    nsdManager.registerServiceInfoCallback(
        queryInfo,
        executor,
        object : NsdManager.ServiceInfoCallback {
            override fun onServiceUpdated(info: NsdServiceInfo) {
                nsdManager.unregisterServiceInfoCallback(this)
                resolved.complete(info)
            }
            // ... other callbacks
        }
    )

    return resolved.orTimeout(30, TimeUnit.SECONDS)
}
```

**Key insight**: mDNS/Avahi in the guest + NsdManager on host = elegant service discovery.

---

## 6. Port Forwarding Architecture

Google uses a native JNI library (`forwarder_host_jni`) for port forwarding:

```kotlin
companion object {
    init {
        System.loadLibrary("forwarder_host_jni")
    }

    @JvmStatic external fun runForwarderHost(cid: Int, callback: ForwarderHostCallback)
    @JvmStatic external fun terminateForwarderHost()
    @JvmStatic external fun updateListeningPorts(ports: IntArray?)
}
```

The forwarder uses **vsock** (CID-based) for VM communication, not TCP/IP.

**Key insight**: For production, we may need a native library for efficient port forwarding. However, for MVP, we can use the `"network": true` config which gives the VM a proper network interface.

---

## 7. gRPC Communication Pattern

Google uses gRPC between host and guest:

```kotlin
class DebianServiceImpl(context: Context) : DebianServiceImplBase() {
    override fun reportVmActivePorts(
        request: ReportVmActivePortsRequest,
        responseObserver: StreamObserver<ReportVmActivePortsResponse>
    ) {
        portsStateManager.updateActivePorts(request.portsList)
        responseObserver.onNext(
            ReportVmActivePortsResponse.newBuilder()
                .setSuccess(true)
                .build()
        )
        responseObserver.onCompleted()
    }
    
    override fun openShutdownRequestQueue(
        request: ShutdownQueueOpeningRequest?,
        responseObserver: StreamObserver<ShutdownRequestItem>
    ) {
        // Guest opens a stream, host sends shutdown when needed
        shutdownRunnable = Runnable {
            responseObserver.onNext(ShutdownRequestItem.newBuilder().build())
            responseObserver.onCompleted()
        }
    }
}
```

**Key insight**: gRPC streaming allows the host to send commands to the guest asynchronously.

---

## 8. Notification Pattern

```kotlin
private fun createNotification(): Notification {
    val stopIntent = Intent(this, VmLauncherService::class.java)
        .setAction(ACTION_SHUTDOWN_VM)
    val stopPendingIntent = PendingIntent.getService(
        this, 0, stopIntent,
        PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
    )
    
    return Notification.Builder(this, CHANNEL_ID)
        .setSmallIcon(R.drawable.ic_launcher)
        .setContentTitle("Sovereign VM Running")
        .setOngoing(true)
        .setSilent(true)
        .addAction(
            Notification.Action.Builder(icon, "Stop", stopPendingIntent).build()
        )
        .build()
}
```

---

## 9. Error Handling Pattern

```kotlin
runner.exitStatus
    .thenAcceptAsync { success ->
        resultReceiver.send(if (success) RESULT_STOP else RESULT_ERROR, null)
        stopSelf()
    }
    .exceptionallyAsync { e ->
        Log.e(TAG, "Failed to start VM", e)
        resultReceiver.send(RESULT_ERROR, null)
        stopSelf()
        null
    }
```

---

## 10. Threading Model

```kotlin
class VmLauncherService : Service() {
    // Thread pool for background work
    private lateinit var bgThreads: ExecutorService
    
    // Single thread for sequential VM operations
    private lateinit var mainWorkerThread: ExecutorService

    override fun onCreate() {
        bgThreads = Executors.newCachedThreadPool(threadFactory)
        mainWorkerThread = Executors.newSingleThreadExecutor(threadFactory)
    }
    
    override fun onDestroy() {
        bgThreads.shutdownNow()
        mainWorkerThread.shutdown()
    }
}
```

**Key insight**: Single-threaded executor for VM operations ensures no race conditions.

---

## Summary: What We Need for Sovereign

| Component | Google's Approach | Our Equivalent |
|-----------|-------------------|----------------|
| VM Config | JSON with variable substitution | Same |
| VM Lifecycle | VmLauncherService + Runner | Same pattern |
| Networking | `useNetwork(true)` | Same |
| Service Discovery | mDNS (NsdManager) | Same, or direct IP |
| Guest Communication | gRPC over vsock | gRPC or simple HTTP |
| Port Forwarding | Native JNI forwarder | Start with network mode |
| Disk Management | InstalledImage class | Same pattern |
| Notifications | Foreground service notification | Same |
