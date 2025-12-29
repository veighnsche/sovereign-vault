# TEAM_023: Field Guide Compliance Audit

## Mission
Systematically audit whether Sovereign is defensive against ALL edge cases documented in "A Field Guide to Deploying Self-Hosted Services on Android 16 with AVF".

## EXECUTIVE SUMMARY

| Category | Score | Critical Gaps |
|----------|-------|---------------|
| Host Process Management | 2/2 | ✅ Complete |
| Guest Environment (musl) | 3/3 | ✅ Complete |
| Kernel Configuration | 6/6 | ✅ Complete |
| PostgreSQL Hardening | 2/3 | ⚠️ SWIOTLB is hardware limitation |
| Networking | 3/3 | ✅ Complete |
| Operational Resilience | 3/3 | ✅ Complete |
| Security | 1/2 | ⚠️ Rollback is pKVM limitation |

**Overall: 20/22 items addressed (91%)**
**Remaining 2 items are hardware/hypervisor limitations, not code gaps**

### TEAM_023 Fixes Applied
1. ✅ `vm/sql/start.sh` - Added Phantom Process Killer defense (`device_config` commands)
2. ✅ `internal/rootfs/rootfs.go` - Added PostgreSQL supervision loop (auto-restart on death)
3. ✅ `vm/sql/build-guest-kernel.sh` - Full VIRTIO stack (PCI, BLK, NET, VSOCK, RNG) + disabled BINDER
4. ✅ `vm/sql/Dockerfile` - Added `icu-data-full` + `PTHREAD_STACK_MIN` env var
5. ✅ `internal/rootfs/rootfs.go` - PostgreSQL initdb uses `--locale-provider=icu --icu-locale=en-US`
6. ✅ `cmd/sovereign/main.go` - Added `backup` command with fsfreeze protocol

---

## Detailed Checklist

### 1. Host-Level Process Management
| Item | Status | Evidence |
|------|--------|----------|
| Phantom Process Killer disabled | ✅ **FIXED** | `vm/sql/start.sh:22-23` - `device_config` commands added |
| LMK protection (oom_score_adj -1000) | ✅ | `vm/sql/start.sh:85` - `echo -1000 > /proc/${VM_PID}/oom_score_adj` |

**RESOLVED by TEAM_023**: Added Phantom Process Killer defense to start.sh

### 2. Guest Environment (Alpine/musl)
| Item | Status | Evidence |
|------|--------|----------|
| Thread stack size configured | ✅ **FIXED** | `vm/sql/Dockerfile:50` - `ENV PTHREAD_STACK_MIN=2097152` |
| ICU collation for PostgreSQL | ✅ **FIXED** | `internal/rootfs/rootfs.go:229` - `--locale-provider=icu --icu-locale=en-US` |
| icu-data-full installed | ✅ **FIXED** | `vm/sql/Dockerfile:13` - `icu-data-full` package added |

**RESOLVED by TEAM_023**: All musl libc mitigations in place

### 3. Custom Kernel Configuration
| Item | Status | Evidence |
|------|--------|----------|
| CONFIG_VIRTIO_PCI | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:35` - `--enable VIRTIO_PCI` |
| CONFIG_VIRTIO_VSOCKETS | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:38` - `--enable VIRTIO_VSOCKETS` |
| CONFIG_VIRTIO_BLK | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:37` - `--enable VIRTIO_BLK` |
| CONFIG_VIRTIO_NET | ✅ | `vm/sql/build-guest-kernel.sh:36` - `--enable VIRTIO_NET` |
| CONFIG_VIRTIO_RNG | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:39-40` - `--enable HW_RANDOM_VIRTIO` |
| CONFIG_ANDROID_BINDER_IPC disabled | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:41` - `--disable ANDROID_BINDER_IPC` |

**RESOLVED by TEAM_023**: Full VIRTIO stack + disabled Android-specific features

### 4. PostgreSQL Hardening
| Item | Status | Evidence |
|------|--------|----------|
| I/O performance awareness (SWIOTLB) | ⚠️ DOCUMENTED | Mentioned in docs but no mitigation |
| ICU collation configured | ✅ **FIXED** | `internal/rootfs/rootfs.go:229` - initdb with `--locale-provider=icu` |
| dynamic_shared_memory_type = mmap | ✅ | `internal/rootfs/rootfs.go:233` |

### 5. Networking
| Item | Status | Evidence |
|------|--------|----------|
| Unprivileged ports (>1024) | ✅ | PostgreSQL on 5432, Forgejo on 3000 |
| Tailscale integration | ✅ | Full integration via `tailscale serve` |
| userspace-networking fallback | ✅ | `internal/rootfs/rootfs.go:194` - `--tun=userspace-networking` |

### 6. Operational Resilience
| Item | Status | Evidence |
|------|--------|----------|
| fsfreeze backup protocol | ✅ **FIXED** | `cmd/sovereign/main.go:206-259` - `sovereign backup --sql` command |
| Process supervision in VM | ✅ **FIXED** | `internal/rootfs/rootfs.go:249-258` - supervision loop added |
| Crash recovery | ✅ **FIXED** | PostgreSQL auto-restarts via pg_isready check every 30s |

**RESOLVED by TEAM_023**: Full operational resilience - backup + supervision + recovery

### 7. Security
| Item | Status | Evidence |
|------|--------|----------|
| Block-level rollback awareness | ⚠️ **LIMITATION** | pKVM architecture limitation - cannot be fixed in software |
| Entropy source (VIRTIO_RNG) | ✅ **FIXED** | `vm/sql/build-guest-kernel.sh:39-40` - HW_RANDOM_VIRTIO enabled |

---

## CRITICAL GAPS RANKED BY SEVERITY

### ✅ ALL CODE GAPS RESOLVED by TEAM_023
1. ~~**Phantom Process Killer not disabled**~~ → FIXED in `vm/sql/start.sh`
2. ~~**No process supervision**~~ → FIXED in `internal/rootfs/rootfs.go`
3. ~~**No VIRTIO_RNG**~~ → FIXED in `vm/sql/build-guest-kernel.sh`
4. ~~**ICU collation not configured**~~ → FIXED in `vm/sql/Dockerfile` + `internal/rootfs/rootfs.go`
5. ~~**Thread stack size**~~ → FIXED in `vm/sql/Dockerfile`
6. ~~**Missing VIRTIO configs**~~ → FIXED in `vm/sql/build-guest-kernel.sh`
7. ~~**No backup protocol**~~ → FIXED in `cmd/sovereign/main.go`

### ⚠️ Hardware/Hypervisor Limitations (Cannot Fix in Code)
1. **SWIOTLB overhead** - ~25% I/O penalty is inherent to pKVM bounce buffering
2. **Block-level rollback** - pKVM architecture does not provide rollback protection

---

## FINAL STATUS: 91% COMPLETE

All fixable items have been addressed. Remaining gaps are hardware limitations.

