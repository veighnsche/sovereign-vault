Feature: SQL VM Lifecycle
  As a developer
  I want to manage the PostgreSQL VM
  So that I can run a database on my Android device

  # ==========================================================================
  # BUILD BEHAVIORS
  # Source: sql.go Build() and common.go Build()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @build
  Scenario: Build SQL VM with Docker
    Given Docker is available
    And the guest kernel Image exists at "vm/sql/Image"
    When I build the SQL VM
    Then the Docker image "sovereign-sql" should exist
    And the rootfs should be created at "vm/sql/rootfs.img"
    And the data disk should be created at "vm/sql/data.img"

  @build @dockerfile
  Scenario: Dockerfile uses verified Alpine version
    Given Docker is available
    When I build the SQL VM
    Then the Dockerfile should use Alpine version "3.23"
    # Note: Alpine 3.23 ships PostgreSQL 18, not 15

  @build @dockerfile @postgresql
  Scenario: Dockerfile installs correct PostgreSQL package
    Given Docker is available
    When I build the SQL VM
    Then the Dockerfile should install "postgresql" package
    And the Dockerfile should install "postgresql-contrib" package
    And the Dockerfile should install "icu-data-full" for ICU collation
    # Note: Package is "postgresql" not "postgresql15" in Alpine 3.23

  @build @dockerfile @tailscale
  Scenario: Dockerfile installs Tailscale static binary
    Given Docker is available
    When I build the SQL VM
    Then the Dockerfile should download Tailscale version "1.92.3"
    And the Tailscale binaries should be at "/usr/bin/tailscale" and "/usr/sbin/tailscaled"
    # Note: Static binary from pkgs.tailscale.com, not Alpine package

  @build @rootfs @init
  Scenario: Rootfs preparation creates init script
    Given the Docker build completed
    When the rootfs is prepared for AVF
    Then "/sbin/init.sh" should be created from "vm/sql/init.sh"
    And "/sbin/init.sh" should be executable
    And device nodes should be pre-created in "/dev"

  @build @rootfs @devices
  Scenario: Rootfs preparation creates device nodes
    Given the Docker build completed
    When the rootfs is prepared for AVF
    Then "/dev/console" should exist with major 5, minor 1
    And "/dev/null" should exist with major 1, minor 3
    And "/dev/tty" should exist with major 5, minor 0
    And "/dev/vsock" should exist with major 10, minor 121
    And "/dev/net/tun" should exist with major 10, minor 200

  @build @secrets
  Scenario: Build loads existing secrets
    Given Docker is available
    And a .secrets file exists
    When I build the SQL VM
    Then existing credentials should be used
    And I should see "Using existing credentials from .secrets"

  @build @secrets
  Scenario: Build prompts for new secrets
    Given Docker is available
    And no .secrets file exists
    When I build the SQL VM interactively
    Then I should be prompted for database password
    And credentials should be saved to .secrets

  @build @data-disk
  Scenario: Build creates data disk if not exists
    Given Docker is available
    And no data disk exists at "vm/sql/data.img"
    When I build the SQL VM
    Then a 4GB data disk should be created
    And the data disk should be formatted as ext4

  @build @data-disk @idempotent
  Scenario: Build preserves existing data disk
    Given Docker is available
    And a data disk already exists at "vm/sql/data.img"
    When I build the SQL VM
    Then the existing data disk should not be overwritten
    And I should see "Data disk already exists, skipping"

  @build @error
  Scenario: Build fails without Docker
    Given Docker is not available
    When I try to build the SQL VM
    Then the build should fail with error containing "docker not found"

  @build @error
  Scenario: Build fails without guest kernel
    Given Docker is available
    But the guest kernel Image does not exist at "vm/sql/Image"
    When I try to build the SQL VM
    Then the build should fail with error containing "kernel Image not found"

  # ==========================================================================
  # DEPLOY BEHAVIORS
  # Source: common.go Deploy() and CreateStartScript()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @deploy
  Scenario: Deploy SQL VM to device
    Given a device is connected
    And the SQL VM is built
    When I deploy the SQL VM
    Then the VM directory should exist on device at "/data/sovereign/vm/sql"
    And "rootfs.img" should be pushed to device
    And "Image" should be pushed to device
    And "data.img" should be pushed to device
    And the start script should exist at "/data/sovereign/vm/sql/start.sh"

  @deploy @tailscale-cleanup
  Scenario: Deploy checks for existing Tailscale registrations
    Given a device is connected
    And the SQL VM is built
    When I deploy the SQL VM
    Then old Tailscale registrations for "sovereign-sql" should be checked
    And I should see "Checking for existing Tailscale registrations"

  @deploy @env
  Scenario: Deploy pushes .env file if exists
    Given a device is connected
    And the SQL VM is built
    And a .env file exists with TAILSCALE_AUTHKEY
    When I deploy the SQL VM
    Then ".env" should be pushed to "/data/sovereign/.env"

  @deploy @start-script
  Scenario: Start script configures TAP networking
    Given a device is connected
    And the SQL VM is deployed
    Then the start script should configure TAP "vm_sql" with IP "192.168.100.1/24"
    And the start script should enable IP forwarding
    And the start script should add Android routing bypass rule
    And the start script should configure NAT masquerade to wlan0
    And the start script should add FORWARD rules for vm_sql

  @deploy @start-script @console
  Scenario: Start script uses ttyS0 for console
    Given a device is connected
    And the SQL VM is deployed
    Then the kernel params should contain "console=ttyS0"
    # Note: crosvm --serial captures ttyS0, NOT virtio hvc0

  # ==========================================================================
  # START BEHAVIORS
  # Source: common.go Start(), CheckTailscaleRegistration(), StreamBootAndWait()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @start
  Scenario: Start SQL VM successfully
    Given a device is connected
    And the SQL VM is deployed
    When I start the SQL VM
    Then the VM process should be running
    And the TAP interface "vm_sql" should be UP
    And the console log should contain "PostgreSQL started"
    And the console log should contain "INIT COMPLETE"

  @start @init @time
  Scenario: Init script sets correct system time
    Given a device is connected
    And the SQL VM is deployed
    When I start the SQL VM
    Then the VM system time should be set to a valid date
    And TLS certificate validation should succeed
    # Note: Time must be after cert "not before" dates (Dec 2025)

  @start @init @postgresql
  Scenario: Init script initializes PostgreSQL with ICU
    Given a device is connected
    And the SQL VM is deployed
    When I start the SQL VM
    Then PostgreSQL should be initialized with ICU locale provider
    And the default collation should be "en-US"
    And PostgreSQL version 18 should be running
    # Note: ICU prevents silent B-tree index corruption from musl libc

  @start @init @tailscale
  Scenario: Init script starts Tailscale with userspace networking
    Given a device is connected
    And the SQL VM is deployed
    And a valid TAILSCALE_AUTHKEY is configured
    When I start the SQL VM
    Then Tailscale should connect with hostname "sovereign-sql"
    And Tailscale should use userspace networking mode
    And "tailscale serve" should expose port 5432

  @start @idempotent
  Scenario: Start is idempotent when VM already running
    Given a device is connected
    And the SQL VM is running
    When I start the SQL VM
    Then the command should succeed
    And I should see "VM already running"
    And no new process should be started

  @start @tailscale
  Scenario: Start auto-cleans old Tailscale registrations
    Given a device is connected
    And the SQL VM is deployed
    And a Tailscale registration exists for "sovereign-sql"
    When I start the SQL VM
    Then the old registration should be removed
    And a new registration should be created
    And the VM should start

  @start @tailscale @force
  Scenario: Start with force skips Tailscale cleanup
    Given a device is connected
    And the SQL VM is deployed
    And a Tailscale registration exists for "sovereign-sql"
    When I start the SQL VM with force flag
    Then the Tailscale cleanup should be skipped
    And duplicate registrations may be created

  @start @timeout
  Scenario: Start times out after 90 seconds
    Given a device is connected
    And the SQL VM is deployed
    And the VM will not become ready
    When I start the SQL VM
    Then the start should fail with "timeout waiting for PostgreSQL (90s)"

  @start @error
  Scenario: Start fails on kernel panic
    Given a device is connected
    And the SQL VM is deployed
    And the init script will crash
    When I start the SQL VM
    Then the start should fail with "VM boot failed"
    And the console should contain "Kernel panic"

  @start @error
  Scenario: Start fails if VM process dies
    Given a device is connected
    And the SQL VM is deployed
    And the VM process will die during boot
    When I start the SQL VM
    Then the start should fail with "VM process died during boot"

  @start @supervision
  Scenario: Init script supervises PostgreSQL and Tailscale
    Given a device is connected
    And the SQL VM is running
    When PostgreSQL crashes
    Then the supervision loop should restart PostgreSQL
    And PostgreSQL should become available again

  @start @supervision @tailscale
  Scenario: Init script restarts Tailscale if it dies
    Given a device is connected
    And the SQL VM is running
    When Tailscaled process dies
    Then the supervision loop should restart Tailscaled
    And Tailscale should reconnect

  # ==========================================================================
  # STOP BEHAVIORS
  # Source: common.go Stop()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @stop
  Scenario: Stop running SQL VM
    Given a device is connected
    And the SQL VM is running
    When I stop the SQL VM
    Then the VM process should not be running
    And the TAP interface "vm_sql" should be removed
    And the socket file should be removed
    And the pid file should be removed
    And I should see "VM stopped"

  @stop @networking
  Scenario: Stop cleans up networking
    Given a device is connected
    And the SQL VM is running
    When I stop the SQL VM
    Then NAT rules should be removed
    And FORWARD rules should be removed
    And I should see "Cleaning up networking"

  @stop @idempotent
  Scenario: Stop is idempotent when VM not running
    Given a device is connected
    And the SQL VM is not running
    When I stop the SQL VM
    Then the command should succeed
    And I should see "VM not running"

  @stop @force-kill
  Scenario: Stop force kills unresponsive VM
    Given a device is connected
    And the SQL VM is running but unresponsive
    When I stop the SQL VM
    Then the VM should be force killed with SIGKILL
    And the VM process should not be running

  # ==========================================================================
  # TEST BEHAVIORS
  # Source: sql.go Test()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @test
  Scenario: Test healthy SQL VM
    Given a device is connected
    And the SQL VM is running
    And the TAP interface "vm_sql" is UP
    And PostgreSQL is responding on port 5432
    When I test the SQL VM
    Then I should see "VM process running: ✓ PASS"
    And I should see "TAP interface (vm_sql): ✓ PASS"
    And I should see "Tailscale connected: ✓ PASS"
    And I should see "PostgreSQL responding (via TAP): ✓ PASS"
    And I should see "Can execute query (via TAP): ✓ PASS"
    And I should see "ALL TESTS PASSED"

  @test @tailscale
  Scenario: Test includes Tailscale connectivity
    Given a device is connected
    And the SQL VM is running
    And Tailscale is connected
    When I test the SQL VM
    Then I should see "Tailscale connected: ✓ PASS"
    And the Tailscale IP should be displayed
    And the hostname "sovereign-sql" should be displayed

  @test @error
  Scenario: Test fails when VM not running
    Given a device is connected
    And the SQL VM is not running
    When I test the SQL VM
    Then I should see "VM process running: ✗ FAIL"
    And the test should fail

  @test @error
  Scenario: Test fails when TAP is down
    Given a device is connected
    And the SQL VM is running
    But the TAP interface "vm_sql" is DOWN
    When I test the SQL VM
    Then I should see "TAP interface (vm_sql): ✗ FAIL"
    And the test should fail

  @test @error
  Scenario: Test fails when PostgreSQL not responding
    Given a device is connected
    And the SQL VM is running
    And the TAP interface "vm_sql" is UP
    But PostgreSQL is not responding on port 5432
    When I test the SQL VM
    Then I should see "PostgreSQL responding (via TAP): ✗ FAIL"
    And the test should fail

  @test @error @tailscale
  Scenario: Test fails when Tailscale not connected
    Given a device is connected
    And the SQL VM is running
    But Tailscale is not connected
    When I test the SQL VM
    Then I should see "Tailscale connected: ✗ FAIL"
    And the test should fail

  # ==========================================================================
  # REMOVE BEHAVIORS
  # Source: common.go Remove()
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @remove
  Scenario: Remove deployed SQL VM
    Given a device is connected
    And the SQL VM is deployed
    And the SQL VM is not running
    When I remove the SQL VM
    Then the directory "/data/sovereign/vm/sql" should be deleted
    And the command should succeed

  @remove
  Scenario: Remove stops running VM first
    Given a device is connected
    And the SQL VM is running
    When I remove the SQL VM
    Then the VM should be stopped first
    And the directory "/data/sovereign/vm/sql" should be deleted

  @remove @data
  Scenario: Remove deletes all VM data
    Given a device is connected
    And the SQL VM is deployed
    When I remove the SQL VM
    Then "rootfs.img" should be deleted
    And "data.img" should be deleted
    And "Image" should be deleted
    And "start.sh" should be deleted
    And "console.log" should be deleted

  @remove @error-tolerant
  Scenario: Remove continues even if stop fails
    Given a device is connected
    And the SQL VM is in a bad state
    When I remove the SQL VM
    Then I should see a warning about stop failure
    But the VM directory should still be removed

  # ==========================================================================
  # STATUS BEHAVIORS
  # Source: cmd/sovereign/main.go status command
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @status
  Scenario: Status shows device connection
    Given a device is connected
    When I run sovereign status
    Then I should see "Device connected: ✓ Yes"

  @status @kernel
  Scenario: Status shows kernel information
    Given a device is connected
    When I run sovereign status
    Then I should see the kernel version
    And I should see KernelSU status

  # ==========================================================================
  # BACKUP BEHAVIORS
  # Source: cmd/sovereign/main.go backup command
  # Frozen: 2025-12-29 - TEAM_023
  # ==========================================================================

  @backup
  Scenario: Backup creates crash-consistent snapshot
    Given a device is connected
    And the SQL VM is running
    When I backup the SQL VM
    Then PostgreSQL should be fsfreeze'd
    And the VM should be suspended
    And a snapshot of data.img should be created
    And the VM should be resumed
    And PostgreSQL should be fsunfreeze'd

  @backup @error
  Scenario: Backup fails if VM not running
    Given a device is connected
    And the SQL VM is not running
    When I try to backup the SQL VM
    Then the backup should fail with "VM not running"
