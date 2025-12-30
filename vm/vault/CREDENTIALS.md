# Vaultwarden Credentials

> **TEAM_035**: Credentials centralized in `.env` (not in source control)

## Database Credentials

| Field | Value |
|-------|-------|
| **Host** | `192.168.100.2` (SQL VM via TAP) |
| **Port** | `5432` |
| **Database** | `vaultwarden` |
| **Username** | `vaultwarden` |
| **Password** | See `POSTGRES_VAULTWARDEN_PASSWORD` in `.env` |

### Connection String
```
postgresql://vaultwarden:${POSTGRES_VAULTWARDEN_PASSWORD}@192.168.100.2:5432/vaultwarden
```

## How Secrets Work

All secrets are centralized in `.env` (gitignored) and passed to VMs via kernel cmdline:

1. `.env` contains `POSTGRES_VAULTWARDEN_PASSWORD` and `VAULTWARDEN_ADMIN_TOKEN`
2. `start.sh` reads `.env` and passes secrets via kernel cmdline
3. `init.sh` reads secrets from `/proc/cmdline` at boot
4. No secrets in source control!

## How to Change Password

1. **Update `.env`**:
   ```bash
   POSTGRES_VAULTWARDEN_PASSWORD=your_new_password
   ```

2. **Redeploy with fresh data** (recreates database users):
   ```bash
   ./sovereign deploy --sql --fresh-data
   ./sovereign deploy --vault --fresh-data
   ./sovereign start --sql
   ./sovereign start --vault
   ```

## Security Notes

- Secrets stored in `.env` (gitignored, NOT in source control)
- Database only accessible via internal bridge network (192.168.100.0/24)
- Traffic never leaves the Android device
- Admin panel enabled if `VAULTWARDEN_ADMIN_TOKEN` is set
