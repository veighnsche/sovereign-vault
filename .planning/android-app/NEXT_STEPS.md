# Sovereign Android App - Next Steps

**TEAM_033** | Created: 2025-12-29

## Immediate Next Steps

### Step 1: Create Android Project Structure

```bash
# In sovereign/ directory
mkdir -p android/app/src/main/java/com/sovereign/vault
mkdir -p android/app/src/main/res
mkdir -p android/app/src/main/assets
```

Project files needed:
- `android/settings.gradle.kts`
- `android/build.gradle.kts`
- `android/app/build.gradle.kts`
- `android/app/src/main/AndroidManifest.xml`
- `android/gradle.properties`

### Step 2: Minimal Bootable VM Test

Create a minimal app that:
1. Has the required permissions
2. Loads a simple VM config
3. Boots a minimal Alpine Linux
4. Logs output to logcat

### Step 3: Port Existing Assets

Convert current VM build outputs to app-compatible format:
- `vm/sql/rootfs.img` → `assets/sql/rootfs.img` (or download on first launch)
- `vm/sql/vmlinuz` → `assets/sql/vmlinuz`
- `vm/sql/initrd.img` → `assets/sql/initrd.img`

### Step 4: Test Networking

With `useNetwork(true)`:
- Verify VM gets IP address automatically
- Test ping from VM to internet
- Test PostgreSQL connectivity

---

## Questions to Resolve

1. **Image bundling**: Should we bundle VM images in APK (large) or download on first launch?
   - APK size limit: ~150MB for Play Store, no limit for sideloading
   - Our rootfs: ~500MB+ 
   - **Recommendation**: Download on first launch, show progress

2. **Tailscale integration**: Run in VM or on Android host?
   - In VM: Current approach, has fwmark issues
   - On host: Cleaner, uses Android's Tailscale app
   - **Recommendation**: Explore host-side Tailscale with subnet routing

3. **Guest init system**: Keep custom init.sh or switch to systemd?
   - Custom init.sh: We control everything
   - Systemd: Standard, but heavier
   - **Recommendation**: Keep custom init for Alpine, minimal overhead

---

## File Checklist for MVP

### Android Project Files
- [ ] `android/settings.gradle.kts`
- [ ] `android/build.gradle.kts`
- [ ] `android/gradle.properties`
- [ ] `android/app/build.gradle.kts`
- [ ] `android/app/src/main/AndroidManifest.xml`

### Kotlin Source Files
- [ ] `SovereignApplication.kt` - Application class
- [ ] `MainActivity.kt` - Main UI
- [ ] `VmLauncherService.kt` - Foreground service
- [ ] `VmRunner.kt` - VM lifecycle management
- [ ] `VmConfig.kt` - Configuration parsing
- [ ] `VmImage.kt` - Disk image management

### VM Assets
- [ ] `vm_config.json` - VM configuration
- [ ] Kernel and initrd (bundled or downloaded)
- [ ] Rootfs (downloaded on first launch)

---

## Development Environment Requirements

1. **Android SDK**: API 34+ (Android 14)
2. **Kotlin**: 1.9+
3. **Gradle**: 8.0+
4. **Device**: Pixel 6+ with unlocked bootloader
5. **Permissions**: Grant via ADB during development

```bash
# Grant permissions after install
adb shell pm grant com.sovereign.vault android.permission.MANAGE_VIRTUAL_MACHINE
adb shell pm grant com.sovereign.vault android.permission.USE_CUSTOM_VIRTUAL_MACHINE
```

---

## Estimated Timeline

| Week | Milestone |
|------|-----------|
| 1 | Project setup, minimal VM boot |
| 2 | Port existing VM images, test networking |
| 3 | PostgreSQL integration, Tailscale exploration |
| 4 | Forgejo integration, UI polish |
| 5 | Testing, bug fixes, documentation |

---

## Risk Mitigation

### Risk: MANAGE_VIRTUAL_MACHINE permission denied
**Mitigation**: Use ADB grant for development; create Magisk module for production

### Risk: VM images too large for APK
**Mitigation**: Download on first launch with progress indicator

### Risk: Networking still doesn't work
**Mitigation**: `useNetwork(true)` should work; if not, fallback to vsock-only

### Risk: Performance issues
**Mitigation**: Start with 2GB RAM, adjust based on testing
