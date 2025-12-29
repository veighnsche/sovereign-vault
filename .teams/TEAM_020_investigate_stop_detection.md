# TEAM_020: VM Detection & Boot Streaming

## Issues Fixed

### 1. Stop --sql "VM not running" Bug
**Root Cause**: `pgrep -f 'crosvm.*sql'` was matching itself (the grep process).

**Fix**: Use `pidof crosvm` for crosvm detection in `GetProcessPID()`:
```go
if strings.Contains(pattern, "crosvm") {
    out, _ := RunShellCommand("pidof crosvm | awk '{print $1}'")
    return out
}
```

### 2. Boot Streaming Implementation
Added `streamBootAndWaitForPostgres()` that:
- Streams console.log in real-time during boot
- Detects "PostgreSQL started" or "INIT COMPLETE" markers
- Falls back to PostgreSQL port check (192.168.100.2:5432) for old rootfs
- Tracks kernel boot completion via "Run /sbin/simple_init"

### 3. Init Script Console Output
Modified `simple_init_tap` to output to console instead of `/init.log`:
```diff
- exec > /init.log 2>&1
+ # Output to console for streaming
```
(Requires rootfs rebuild to take effect)

## Files Modified
- `internal/device/device.go` - Fixed GetProcessPID() self-matching
- `internal/vm/sql/sql.go` - Added boot streaming with PostgreSQL readiness detection
- `vm/sql/simple_init_tap` - Removed redirect to /init.log

## Status: COMPLETED
