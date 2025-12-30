# Vaultwarden Credentials

> **TEAM_035**: Generated credentials for Vaultwarden VM

## Database Credentials

| Field | Value |
|-------|-------|
| **Host** | `192.168.100.2` (SQL VM via TAP) |
| **Port** | `5432` |
| **Database** | `vaultwarden` |
| **Username** | `vaultwarden` |
| **Password** | `PCc5zNNG6v8gwguclMQWMPjk4DUvg5F5` |

### Connection String
```
postgresql://vaultwarden:PCc5zNNG6v8gwguclMQWMPjk4DUvg5F5@192.168.100.2:5432/vaultwarden
```

## How to Change Password

1. **Update SQL VM init.sh** (`vm/sql/init.sh:259`):
   ```bash
   su postgres -c "psql -c \"CREATE USER vaultwarden WITH PASSWORD 'NEW_PASSWORD';\""
   ```

2. **Update Vault VM init.sh** (`vm/vault/init.sh:201`):
   ```bash
   export DATABASE_URL="postgresql://vaultwarden:NEW_PASSWORD@192.168.100.2:5432/vaultwarden"
   ```

3. **Rebuild both VMs**:
   ```bash
   ./sovereign build --sql
   ./sovereign build --vault
   ./sovereign deploy --sql --fresh-data
   ./sovereign deploy --vault --fresh-data
   ```

## Security Notes

- Password is stored in source control (same pattern as Forgejo)
- Database is only accessible via internal bridge network (192.168.100.0/24)
- Traffic never leaves the Android device
- For production, consider using the secrets system (`internal/secrets/secrets.go`)
