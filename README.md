# Sovereign Vault CLI

Management CLI for the Sovereign Vault project - custom Android kernel with KernelSU and PostgreSQL VM.

## Structure

```
sovereign/
├── cmd/
│   └── sovereign/
│       └── main.go          # CLI entry point
├── internal/
│   ├── device/              # ADB/fastboot utilities
│   │   ├── device.go
│   │   └── device_test.go
│   ├── docker/              # Docker image export
│   │   ├── docker.go
│   │   └── docker_test.go
│   ├── kernel/              # Kernel build/deploy/test
│   │   ├── kernel.go
│   │   └── kernel_test.go
│   ├── rootfs/              # Rootfs AVF preparation
│   │   ├── rootfs.go
│   │   └── rootfs_test.go
│   ├── secrets/             # Secure credential management
│   │   └── secrets.go
│   └── vm/                  # VM interface and implementations
│       ├── vm.go
│       ├── vm_test.go
│       └── sql/             # PostgreSQL VM
│           ├── sql.go
│           └── sql_test.go
├── go.mod
├── .env                     # Tailscale auth key (gitignored)
├── .env.example             # Template for .env
├── .secrets                 # Database credentials (gitignored, created by build)
├── sovereign_vault.md       # Project documentation
├── docs/                    # Technical documentation
├── .teams/                  # Team logs (multi-agent workflow)
├── .questions/              # Open questions
└── .planning/               # Phase plans
```

## Build

```bash
cd sovereign/
go build -o sovereign ./cmd/sovereign
```

## Usage

```bash
# Kernel operations
./sovereign build --kernel
./sovereign deploy --kernel
./sovereign test --kernel

# PostgreSQL VM operations  
./sovereign build --sql     # Prompts for DB password on first run
./sovereign deploy --sql    # Idempotent - creates dirs if needed
./sovereign start --sql
./sovereign test --sql
./sovereign stop --sql
./sovereign remove --sql    # Clean removal from device

# Status
./sovereign status
./sovereign status --sql
```

## Testing

```bash
go test ./...
go test -short ./...  # Skip slow tests
```

## Setup

1. Copy `.env.example` to `.env`
2. Get a Tailscale auth key from https://login.tailscale.com/admin/settings/keys
3. Fill in `TAILSCALE_AUTHKEY` in `.env`
4. Run `./sovereign build --sql` - you'll be prompted for a database password

## Security

- **No default passwords**: The build process prompts for credentials interactively
- **Secrets file**: Credentials stored in `.secrets` (mode 0600, gitignored)
- **No shell history**: Password entry doesn't echo and isn't logged

## Path References

All paths in `main.go` use `../` to reference files in the parent repo:
- `../build_raviole.sh` - Kernel build script
- `../out/raviole/dist/` - Kernel build output
- `../vm/sql/` - PostgreSQL VM files (Dockerfile, rootfs, etc.)

## Why a Submodule?

The parent repo contains symlinked stock Android kernel files that are tracked by other git repos. The `sovereign/` directory is a standalone Go module that can be tracked independently without interfering with the stock files.
