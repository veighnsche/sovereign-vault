# TEAM_019: Tailscale Registration Idempotency

## Problem
Every `sovereign start --sql` creates a NEW Tailscale machine registration instead of reconnecting to existing one. Result: 7 duplicate `sovereign-sql` machines with different IPs, breaking dependency stability for PostgreSQL clients.

## Root Cause
1. `start.sh` passes `tailscale.authkey=$TAILSCALE_AUTHKEY` via kernel params
2. Guest init calls `tailscale up --authkey=...` unconditionally
3. No preflight check to detect existing registration

## Solution
1. Add preflight check in `Deploy()` that queries Tailscale API for existing `sovereign-sql` machines
2. If found online → FAIL with "machine already registered, delete old one first"
3. If found offline → WARN but allow (machine may have been cleaned up)
4. Only proceed if no matching machine exists OR user explicitly cleans up first

## Files Modified
- `internal/vm/sql/sql.go` - Add `checkTailscaleRegistration()` preflight

## Status: COMPLETED

## Test Output
```
=== Deploying PostgreSQL VM ===
Error: TAILSCALE IDEMPOTENCY CHECK FAILED

  Found 2 existing sovereign-sql machine(s):
    sovereign-sql-5 (100.80.40.12)
    sovereign-sql-6 (100.87.183.44)

  Deploying again will create ANOTHER registration, breaking IP stability.
```

The deploy now fails fast instead of creating duplicate registrations.
