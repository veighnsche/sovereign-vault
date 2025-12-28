# Phase 4 — Integration and Testing

**Feature:** KernelSU integration for stock raviole kernel
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Phase 3 (Implementation)

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade (TEAM_001), am writing this testing phase because testing is where TEAM_030 committed fraud. They reported "10/10 tests passed" when the system was fundamentally broken.
>
> **My failure modes to watch for:**
> - **Skipping tests** because "the build succeeded so it must work"
> - **Marking tests as passed without actually running them**
> - **Interpreting partial success as full success** (e.g., "device booted" ≠ "KernelSU works")
> - **Hiding failures** by not documenting them
> - **Giving up on hard tests** and pretending they're "optional"
>
> **The rule:** A test is PASSED only when I have seen the expected output with my own eyes (or the user has confirmed it). A test is FAILED if ANYTHING unexpected happens. There is no "probably passed."
>
> **If you are a future AI reading this:** Do not lie about test results. A failing test is information. A fake pass is sabotage.

---

## 1. Testing Overview

Verify the KernelSU integration works correctly on target hardware.

> 🤖 **AI Warning:** These tests require PHYSICAL DEVICE ACCESS. If you do not have access to the device, you MUST ask the USER to run the tests and report results. Do NOT assume results.

---

## 2. Test Cases

### TC-1: Kernel Boot Test

**Goal:** Device boots with custom kernel

**Priority:** CRITICAL — Must pass before any other tests

**Pre-Conditions:**
- [ ] Device bootloader is unlocked
- [ ] Backup of stock boot.img exists
- [ ] USB debugging enabled
- [ ] fastboot accessible

**Steps:**
1. **Backup current boot image** (MANDATORY):
   ```bash
   adb reboot bootloader
   fastboot flash boot boot.img.backup  # If you have one, skip this
   # Or: boot into recovery and backup via TWRP
   ```

2. **Flash the new boot image**:
   ```bash
   fastboot flash boot out/raviole/dist/boot.img
   ```

3. **Flash vendor_boot if built**:
   ```bash
   fastboot flash vendor_boot out/raviole/dist/vendor_boot.img
   ```

4. **Reboot and observe**:
   ```bash
   fastboot reboot
   ```

5. **Wait for boot** (up to 2 minutes)

6. **Verify boot completed**:
   ```bash
   adb shell getprop sys.boot_completed
   # Expected: 1
   ```

**Pass Criteria:**
- [ ] Device boots to lock screen within 2 minutes
- [ ] `sys.boot_completed` returns `1`
- [ ] No bootloop (device doesn't restart repeatedly)

**Failure Recovery:**
```bash
# If bootloop, boot to bootloader and flash backup:
fastboot flash boot boot.img.backup
fastboot reboot
```

---

### TC-2: KernelSU Detection

**Goal:** KernelSU manager app detects working root

**Priority:** CRITICAL — Core functionality verification

**Pre-Conditions:**
- [ ] TC-1 passed (device boots)
- [ ] KernelSU Manager APK available

**Steps:**
1. **Download KernelSU Manager** (if not already have):
   - GitHub: https://github.com/tiann/KernelSU/releases
   - Or build from `KernelSU/manager/`

2. **Install the APK**:
   ```bash
   adb install KernelSU_*.apk
   ```

3. **Open the app** on device

4. **Check home screen status**:
   - Look for "Working" badge
   - Note the KernelSU version number (**MUST be 32245 or similar, NOT 16**)
   - Note the kernel version string (should show "-sovereign")

5. **Verify kernel version via adb**:
   ```bash
   adb shell uname -r
   # Expected: contains "-sovereign"
   ```

6. **Verify KernelSU version is correct** (CRITICAL):
   - The manager should show version ~32245 (or current)
   - If it shows "16" or very low number → **version fix was not applied**
   - This indicates the Kbuild patch in Phase 3 Step 1 failed

**Pass Criteria:**
- [ ] Manager shows "Working" status (green checkmark)
- [ ] **Version number is correct (~32245), NOT 16**
- [ ] Kernel version contains "-sovereign" suffix

**Failure Modes:**
| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| "Not Installed" | KernelSU not in kernel | Rebuild with Phase 3 fixes |
| "Unsupported" | Kernel version mismatch | Check kernel was actually flashed |
| App crashes | Manager/kernel version mismatch | Update manager APK |
| **Version shows 16** | **Kbuild version fix not applied** | **Redo Phase 3 Step 1** |

---

### TC-3: Root Shell Access

**Goal:** Can obtain root shell via KernelSU

**Priority:** CRITICAL — Required for Sovereign Vault

**Pre-Conditions:**
- [ ] TC-2 passed (KernelSU detected)

**Steps:**
1. **Grant ADB shell root access**:
   - Open KernelSU Manager
   - Go to "Superuser" tab
   - Find "Shell" or "com.android.shell"
   - Grant root permission

2. **Test via adb**:
   ```bash
   adb shell
   $ su
   # Should see root prompt
   ```

3. **Verify root UID**:
   ```bash
   # id
   # Expected: uid=0(root) gid=0(root) ...
   ```

4. **Test privileged operation**:
   ```bash
   # cat /proc/1/cmdline
   # Expected: Shows init cmdline (requires root)
   ```

5. **Exit cleanly**:
   ```bash
   # exit
   $ exit
   ```

**Pass Criteria:**
- [ ] `su` command succeeds without error
- [ ] `id` shows `uid=0(root)`
- [ ] Can read `/proc/1/cmdline`

**Failure Modes:**
| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| "su: not found" | sucompat not enabled | Check CONFIG_KSU in .config |
| "Permission denied" | Shell not granted root | Grant in KernelSU Manager |
| Hangs on `su` | supercall not working | Check kernel logs (dmesg) |

---

### TC-4: Boot Script Execution

**Goal:** Scripts in `/data/adb/service.d/` execute at boot

**Priority:** CRITICAL — This is how Sovereign Vault starts VMs

> 🤖 **AI Warning:** This is the test that matters most for Sovereign Vault. If boot scripts don't run, VMs won't start. Do NOT skip this test.

**Pre-Conditions:**
- [ ] TC-3 passed (root shell works)

**Steps:**
1. **Create test script** (as root):
   ```bash
   adb shell su -c 'mkdir -p /data/adb/service.d'
   adb shell su -c 'cat > /data/adb/service.d/test_ksu.sh << "EOF"
#!/system/bin/sh
touch /data/local/tmp/ksu_boot_test_marker
echo "$(date)" > /data/local/tmp/ksu_boot_test_timestamp
EOF'
   adb shell su -c 'chmod 755 /data/adb/service.d/test_ksu.sh'
   ```

2. **Verify script exists**:
   ```bash
   adb shell su -c 'ls -la /data/adb/service.d/test_ksu.sh'
   adb shell su -c 'cat /data/adb/service.d/test_ksu.sh'
   ```

3. **Remove any old markers**:
   ```bash
   adb shell su -c 'rm -f /data/local/tmp/ksu_boot_test_*'
   ```

4. **Reboot device**:
   ```bash
   adb reboot
   ```

5. **Wait for boot** (2 minutes)

6. **Check for marker file**:
   ```bash
   adb shell ls -la /data/local/tmp/ksu_boot_test_marker
   adb shell cat /data/local/tmp/ksu_boot_test_timestamp
   ```

**Pass Criteria:**
- [ ] `/data/local/tmp/ksu_boot_test_marker` exists
- [ ] Timestamp file shows recent boot time

**Failure Modes:**
| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| Marker doesn't exist | Scripts not executed | Check ksud logs, SELinux |
| Script exists but didn't run | Wrong permissions | Verify chmod 755 |
| SELinux denial | Policy blocking | Check `adb logcat \| grep avc` |

---

### TC-5: Sovereign Vault Integration

**Goal:** Sovereign Vault start script can execute

**Priority:** HIGH — End-to-end validation

> 🤖 **AI Warning:** This test validates the entire reason for KernelSU integration. If this fails, the project goal is not met. Do not mark Phase 4 complete without this passing.

**Pre-Conditions:**
- [ ] TC-4 passed (boot scripts work)
- [ ] Sovereign Vault files deployed to `/data/sovereign/`
- [ ] crosvm available at `/apex/com.android.virt/bin/crosvm`

**Steps:**
1. **Verify crosvm exists**:
   ```bash
   adb shell ls -la /apex/com.android.virt/bin/crosvm
   ```

2. **Deploy sovereign_start.sh** (if not already):
   ```bash
   adb push host/sovereign_start.sh /data/local/tmp/
   adb shell su -c 'cp /data/local/tmp/sovereign_start.sh /data/adb/service.d/'
   adb shell su -c 'chmod 755 /data/adb/service.d/sovereign_start.sh'
   ```

3. **Reboot and wait**:
   ```bash
   adb reboot
   # Wait 3-5 minutes for VMs to start
   ```

4. **Check for crosvm processes**:
   ```bash
   adb shell su -c 'ps -ef | grep crosvm'
   # Expected: Multiple crosvm processes (sql, vault, forge)
   ```

5. **Check sovereign boot log**:
   ```bash
   adb shell su -c 'cat /data/sovereign/boot.log'
   # Should show VM startup messages
   ```

6. **Verify TAP interfaces**:
   ```bash
   adb shell su -c 'ip link show | grep sovereign'
   # Expected: sovereign_sql, sovereign_vault, sovereign_forge
   ```

**Pass Criteria:**
- [ ] crosvm processes running (at least 3)
- [ ] boot.log shows successful VM starts
- [ ] TAP interfaces created

**Failure Modes:**
| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| No crosvm processes | Script didn't run | Check TC-4 first |
| crosvm crashes | Missing LD_LIBRARY_PATH | Check sovereign_start.sh |
| "Phantom process killer" | Android killing VMs | Verify PPK disabled in script |

---

### TC-6: Device Functionality Regression

**Goal:** No regression in core device functionality

**Priority:** MEDIUM — Ensure kernel changes didn't break device

> 🤖 **AI Warning:** A kernel that provides root but breaks WiFi is useless. Do not skip regression testing.

**Pre-Conditions:**
- [ ] TC-1 passed (device boots)

**Detailed Checklist:**

| Feature | Test Method | Expected Result | Status |
|---------|-------------|-----------------|--------|
| **WiFi** | Settings → WiFi → Connect to network | Connects, can browse | [ ] |
| **Cellular** | Disable WiFi, load webpage | Data works | [ ] |
| **Touch** | Use device normally | Responsive, no ghost touches | [ ] |
| **Camera** | Open camera app, take photo | Photo saves successfully | [ ] |
| **Bluetooth** | Pair with headphones | Audio plays through BT | [ ] |
| **Audio** | Play music via speaker | Sound works | [ ] |
| **GPS** | Open Maps, get location | Location acquired | [ ] |
| **Fingerprint** | Unlock with fingerprint | Recognizes finger | [ ] |
| **NFC** | Tap to pay (if available) | Transaction works | [ ] |
| **USB** | Connect to computer | adb works, file transfer works | [ ] |

**Pass Criteria:**
- [ ] All critical features work (WiFi, Cellular, Touch, Camera)
- [ ] No crashes or ANRs observed
- [ ] Battery drain normal (not excessive heat)

**If Any Feature Fails:**
1. Document the failure in team file
2. Check dmesg for errors: `adb shell su -c dmesg | grep -i error`
3. Check if related driver is loaded: `adb shell lsmod`
4. Compare with stock kernel behavior
5. Create question file if unsure how to proceed

---

## 3. Baseline Protection (Rule 4)

### Golden Outputs to Preserve

| Artifact | Location | Purpose |
|----------|----------|---------|
| Stock kernel symbols | Reference backup | Rollback comparison |
| Working boot.img | Backup before flash | Recovery |
| Device functionality baseline | TC-6 checklist | Regression detection |

---

## 4. Test Results Template

> 🤖 **AI Warning:** Fill this in HONESTLY. Do not write "PASS" unless you have evidence. "Not tested" is acceptable. "PASS (assumed)" is fraud.

| Test | Status | Evidence | Notes |
|------|--------|----------|-------|
| TC-1 Boot | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |
| TC-2 KernelSU Detection | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |
| TC-3 Root Shell | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |
| TC-4 Boot Script | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |
| TC-5 Sovereign Vault | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |
| TC-6 Regression | [ ] PASS / [ ] FAIL / [ ] NOT TESTED | | |

---

## 5. Completion Criteria

Phase 4 is COMPLETE when:

- [ ] TC-1 through TC-4 all PASS (critical path)
- [ ] TC-5 PASS or documented blocker
- [ ] TC-6 no critical regressions
- [ ] All failures documented with evidence
- [ ] Team file updated with results

---

## 6. If Tests Fail

> 🤖 **AI Warning:** Failed tests are INFORMATION, not shame. Document them. Do not hide them.

1. **Document the failure** in team file with:
   - Which test failed
   - Exact error message or symptom
   - Steps to reproduce
   - Any logs (dmesg, logcat)

2. **Diagnose root cause** before attempting fixes

3. **If blocked**, create `.questions/TEAM_001_*.md` file

4. **Do NOT** "fix" by weakening the test criteria

---

## Next Phase

After ALL critical tests pass, proceed to **Phase 5 — Polish and Documentation**.
