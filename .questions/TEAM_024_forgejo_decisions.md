# Forgejo Implementation Decisions

**Created:** 2025-12-29 by TEAM_024
**Source:** Extracted from `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` Section 11

---

## Question 1: Database Connection Method

**Question:** Should Forgejo use the SQL VM's Tailscale IP for database connection, or TAP network?

**Options:**
- A) Tailscale: `sovereign-sql.tail*.ts.net:5432`
- B) TAP network: `192.168.100.2:5432`

**Recommendation:** Use Tailscale for reliability. Tailscale handles reconnection automatically if either VM restarts.

**Current State:** `app.ini` uses `sql-vm:5432` which doesn't match either option.

**Status:** DECIDED - Use Tailscale (Option A)

---

## Question 2: Tailscale Port Exposure

**Question:** What ports should Forgejo expose via Tailscale?

**Options:**
- Port 3000: Web UI (direct or via HTTPS funnel)
- Port 22: SSH for git operations

**Recommendation:** 
- `tailscale serve --bg --tcp 3000 3000` for web
- `tailscale serve --bg --tcp 22 22` for SSH

**Status:** DECIDED - Both ports

---

## Question 3: Shared Kernel

**Question:** Should we create a shared kernel for all VMs?

**Options:**
- A) Current: Each VM copies kernel from `vm/sql/Image`
- B) Future: Shared `vm/common/Image` referenced by all VMs

**Recommendation:** Defer. Current approach works fine. Only optimize if kernel size becomes a problem.

**Status:** DECIDED - Defer (keep current approach)
