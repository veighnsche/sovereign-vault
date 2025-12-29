# Phase 3, Step 4: Go Code

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** [Phase 3: Implementation](phase-3.md)

---

## Goal

Refactor `internal/vm/forge/` to use correct patterns and paths.

---

## Reference Implementation

**Copy patterns from:** `internal/vm/sql/` (sql.go, lifecycle.go, verify.go)

---

## Current State (WRONG)

| Issue | Current | Should Be |
|-------|---------|-----------|
| Device path | `/data/sovereign/forgejo/` | `/data/sovereign/vm/forgejo/` |
| Deploy | Pushes gvproxy binaries | Should NOT push gvproxy |
| Start | Runs start.sh blocking | Should stream boot and wait |
| Process grep | `crosvm.*forgejo` | `[c]rosvm.*forge` |

---

## Tasks

### Task 1: Fix device path constant

In `forge.go`, change all occurrences:

```go
// WRONG
"/data/sovereign/forgejo/"

// CORRECT
"/data/sovereign/vm/forgejo/"
```

### Task 2: Remove gvproxy from Deploy

Remove these lines from Deploy():

```go
// REMOVE THESE
if _, err := os.Stat("vm/sql/bin/gvproxy"); err == nil {
    fmt.Println("Pushing gvproxy...")
    device.PushFile("vm/sql/bin/gvproxy", "/data/sovereign/forgejo/bin/gvproxy")
    fmt.Println("Pushing gvforwarder...")
    device.PushFile("vm/sql/bin/gvforwarder", "/data/sovereign/forgejo/bin/gvforwarder")
}
```

### Task 3: Split into lifecycle.go

Create `internal/vm/forge/lifecycle.go` with Start(), Stop(), Remove():

```go
// lifecycle.go - Forge VM lifecycle operations
package forge

import (
    "fmt"
    "os/exec"
    "strings"
    "time"
    
    "github.com/anthropics/sovereign/internal/device"
)

func (v *VM) Start() error {
    fmt.Println("=== Starting Forgejo VM ===")
    
    // Check if already running
    runningPid := device.GetProcessPID("[c]rosvm.*forge")
    if runningPid != "" {
        fmt.Printf("⚠ VM already running (PID: %s)\n", runningPid)
        return nil
    }
    
    fmt.Println("Note: Forgejo requires SQL VM for database")
    fmt.Println("Tailscale: Using persistent machine identity")
    
    // Check start script
    if !device.FileExists("/data/sovereign/vm/forgejo/start.sh") {
        return fmt.Errorf("start script not found - run 'sovereign deploy --forge' first")
    }
    
    // Clear old log
    device.RunShellCommand("rm -f /data/sovereign/vm/forgejo/console.log")
    
    // Start VM
    fmt.Println("Starting VM...")
    cmd := exec.Command("adb", "shell", "su", "-c", "/data/sovereign/vm/forgejo/start.sh")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("start script failed: %w", err)
    }
    
    // Stream boot and wait for Forgejo
    fmt.Println("\n--- Boot Sequence ---")
    return streamBootAndWaitForForgejo()
}

func streamBootAndWaitForForgejo() error {
    const maxWaitSeconds = 120  // Forgejo takes longer to start
    
    var lastLineCount int
    startTime := time.Now()
    
    for {
        elapsed := time.Since(startTime)
        if elapsed > maxWaitSeconds*time.Second {
            return fmt.Errorf("timeout waiting for Forgejo (%.0fs)", elapsed.Seconds())
        }
        
        out, _ := device.RunShellCommand(
            fmt.Sprintf("cat /data/sovereign/vm/forgejo/console.log 2>/dev/null | tail -n +%d", lastLineCount+1))
        if out != "" {
            lines := strings.Split(out, "\n")
            for _, line := range lines {
                if line != "" {
                    fmt.Println(line)
                    lastLineCount++
                    
                    if strings.Contains(line, "INIT COMPLETE") {
                        time.Sleep(2 * time.Second)
                        fmt.Println("\n✓ Forgejo VM started")
                        return nil
                    }
                    
                    if strings.Contains(line, "Kernel panic") {
                        return fmt.Errorf("VM boot failed - see output above")
                    }
                }
            }
        }
        
        if device.GetProcessPID("[c]rosvm.*forge") == "" {
            return fmt.Errorf("VM process died during boot")
        }
        
        time.Sleep(500 * time.Millisecond)
    }
}

func (v *VM) Stop() error {
    fmt.Println("=== Stopping Forgejo VM ===")
    
    pid := device.GetProcessPID("[c]rosvm.*forge")
    if pid != "" {
        fmt.Printf("Stopping VM (PID: %s)...\n", pid)
        device.KillProcess(pid)
    } else {
        fmt.Println("VM not running")
    }
    
    // Clean up networking
    fmt.Println("Cleaning up networking...")
    device.RunShellCommand("ip link del vm_forge 2>/dev/null")
    device.RunShellCommand("iptables -t nat -D POSTROUTING -s 192.168.101.0/24 -o wlan0 -j MASQUERADE 2>/dev/null")
    device.RunShellCommand("iptables -D FORWARD -i vm_forge -o wlan0 -j ACCEPT 2>/dev/null")
    device.RunShellCommand("iptables -D FORWARD -i wlan0 -o vm_forge -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null")
    device.RunShellCommand("rm -f /data/sovereign/vm/forgejo/vm.pid 2>/dev/null")
    
    fmt.Println("✓ VM stopped")
    return nil
}

func (v *VM) Remove() error {
    fmt.Println("=== Removing Forgejo VM from device ===")
    
    v.Stop()
    
    fmt.Println("Removing Tailscale registration...")
    RemoveTailscaleRegistrations()
    
    fmt.Println("Removing VM files...")
    device.RemoveDir("/data/sovereign/vm/forgejo")
    
    if device.DirExists("/data/sovereign/vm/forgejo") {
        return fmt.Errorf("failed to remove /data/sovereign/vm/forgejo")
    }
    
    fmt.Println("✓ Forgejo VM removed")
    return nil
}
```

### Task 4: Create verify.go

```go
// verify.go - Forge VM verification and Tailscale cleanup
package forge

import (
    "fmt"
    "os/exec"
    "strings"
    
    "github.com/anthropics/sovereign/internal/device"
)

func (v *VM) Test() error {
    fmt.Println("=== Testing Forgejo VM ===")
    
    // Test 1: VM process
    fmt.Println("\n[Test 1/4] Checking VM process...")
    if device.GetProcessPID("[c]rosvm.*forge") == "" {
        return fmt.Errorf("VM process not running")
    }
    fmt.Println("  ✓ VM process running")
    
    // Test 2: TAP interface
    fmt.Println("\n[Test 2/4] Checking TAP interface...")
    out, _ := device.RunShellCommand("ip link show vm_forge 2>/dev/null | grep UP")
    if out == "" {
        return fmt.Errorf("TAP interface vm_forge not UP")
    }
    fmt.Println("  ✓ TAP interface UP")
    
    // Test 3: Tailscale
    fmt.Println("\n[Test 3/4] Checking Tailscale...")
    cmd := exec.Command("tailscale", "ping", "-c", "1", "sovereign-forge")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("sovereign-forge not reachable via Tailscale")
    }
    fmt.Println("  ✓ Tailscale connected")
    
    // Test 4: Forgejo web UI
    fmt.Println("\n[Test 4/4] Checking Forgejo web UI...")
    cmd = exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", 
        "--connect-timeout", "5", "http://sovereign-forge:3000")
    output, _ := cmd.Output()
    if string(output) != "200" {
        fmt.Printf("  ⚠ Web UI returned: %s\n", string(output))
    } else {
        fmt.Println("  ✓ Forgejo web UI responding")
    }
    
    fmt.Println("\n✓ Forgejo VM tests complete")
    return nil
}

func RemoveTailscaleRegistrations() {
    // Remove old registrations via Tailscale API
    cmd := exec.Command("tailscale", "status", "--json")
    // ... similar to SQL's RemoveTailscaleRegistrations
}
```

---

## Expected Output

- Refactored: `internal/vm/forge/forge.go`
- New: `internal/vm/forge/lifecycle.go`
- New: `internal/vm/forge/verify.go`
- All paths use `/data/sovereign/vm/forgejo/`
- No gvproxy references

---

## Verification

1. `grep -r "sovereign/forgejo" internal/vm/forge/` returns no results
2. `grep -r "gvproxy" internal/vm/forge/` returns no results
3. `grep -r "vm_forge" internal/vm/forge/` shows TAP interface usage
4. All three files exist and compile

---

## Next Phase

→ [Phase 4: Integration & Testing](phase-4.md)
