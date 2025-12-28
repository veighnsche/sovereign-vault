# Phase 3B, Step 3 — Tailscale + Verify PostgreSQL

**Phase:** 3B (PostgreSQL)
**Step:** 3 of 3
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Step 2

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** This is the final verification of Phase 3B. I am prone to:
> - Declaring success because the VM booted without testing PostgreSQL
> - Claiming Tailscale works without actually connecting from another device
> - Using `localhost` tests that don't prove network connectivity
> - Skipping this step entirely because "the VM is running"
>
> **The rule:** Success means `psql` works FROM ANOTHER MACHINE via Tailscale. Nothing less.
>
> **TEAM_030's sin:** They faked this test. They reported PostgreSQL working when it was SQLite. I will not repeat this.

---

## 1. Goal

Verify PostgreSQL is accessible via Tailscale from another device on the Tailnet.

---

## 2. Pre-Conditions

- [ ] Step 2 complete (VM running)
- [ ] Tailscale auth key was provided at VM start
- [ ] You have another device on the same Tailscale network

---

## 3. Task 1: Verify VM is on Tailscale

**From Tailscale admin console:**
https://login.tailscale.com/admin/machines

Look for `sql-vm` in the machine list.

**Or from your dev machine:**
```bash
tailscale status | grep sql-vm
```

Expected: `sql-vm` appears with an IP like `100.x.x.x`

> 🤖 **AI Warning:** If sql-vm doesn't appear, Tailscale didn't connect. Check VM console log for errors.

---

## 4. Task 2: Get VM Tailscale IP

```bash
# From tailscale
tailscale ip sql-vm
# Returns: 100.x.x.x

# Or from admin console
# Note the IP address shown for sql-vm
```

Record the IP: `____________` (fill in)

---

## 5. Task 3: Test TCP Connectivity

From another machine on Tailscale:

```bash
# Test if port 5432 is open
nc -zv <sql-vm-tailscale-ip> 5432

# Expected output:
# Connection to 100.x.x.x 5432 port [tcp/postgresql] succeeded!
```

---

## 6. Task 4: Test PostgreSQL Connection

```bash
# Connect with psql
psql -h <sql-vm-tailscale-ip> -U postgres -c "SELECT version();"

# Password: sovereign (set in init script)

# Expected: PostgreSQL version string
```

**Or with connection string:**
```bash
psql "postgresql://postgres:sovereign@<sql-vm-tailscale-ip>:5432/postgres" -c "SELECT 1;"
```

---

## 7. Task 5: Run sovereign test

```bash
go run sovereign.go test --sql
```

Expected output:
```
=== Testing SQL VM ===

1. VM process running: ✓ PASS
2. TAP interface up: ✓ PASS
3. Tailscale connected: ✓ PASS (100.x.x.x)
4. PostgreSQL responding: ✓ PASS
5. Can execute query: ✓ PASS

=== ALL TESTS PASSED ===
Phase 3B complete! PostgreSQL accessible via Tailscale.
```

---

## 8. Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| sql-vm not in Tailscale | Auth key issue | Check VM logs, verify key is valid |
| Connection refused | PostgreSQL not listening | Check `listen_addresses = '*'` in config |
| Connection timeout | Firewall/routing | Check TAP interface, iptables rules |
| Auth failed | Wrong password | Password is `sovereign` per init script |
| "database does not exist" | Normal, expected | Connect to `postgres` database first |

---

## 9. Verification Checklist

```bash
echo "=== Phase 3B Step 3 Verification ==="

echo -n "1. sql-vm in Tailscale: "
tailscale status | grep -q sql-vm && echo "✓" || echo "✗ FAIL"

echo -n "2. Port 5432 open: "
nc -zv $(tailscale ip sql-vm) 5432 2>&1 | grep -q succeeded && echo "✓" || echo "✗ FAIL"

echo -n "3. PostgreSQL responds: "
psql -h $(tailscale ip sql-vm) -U postgres -c "SELECT 1;" 2>/dev/null | grep -q 1 && echo "✓" || echo "✗ FAIL"

echo "=== End Verification ==="
```

---

## 10. Checkpoint

- [ ] `sql-vm` appears in Tailscale admin console
- [ ] Can ping sql-vm from another Tailscale device
- [ ] Port 5432 is open (nc test passes)
- [ ] `psql` connects successfully
- [ ] Can execute SQL queries

---

## Phase 3B Complete!

If all tests pass, you have:
- PostgreSQL running on your phone in a pKVM VM
- Accessible from anywhere via Tailscale
- Ready to be used by Vaultwarden and Forgejo

**Summary:**
```
Phase 3A: KernelSU ✓
├── Kernel with KernelSU compiled in
├── "-sovereign" kernel suffix
└── Root access working

Phase 3B: PostgreSQL ✓
├── Alpine Linux VM under pKVM
├── PostgreSQL 15 running
├── Tailscale connected
└── Accessible from any Tailscale device
```

**Next:** Proceed to **[Phase 3C — Vaultwarden](phase-3c.md)**
