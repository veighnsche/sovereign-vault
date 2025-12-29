Feature: Forge VM Lifecycle
  As a developer
  I want to manage the Forgejo VM
  So that I can run git hosting on my Android device

  # ==========================================================================
  # BUILD BEHAVIORS
  # Source: forge.go Build()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @build
  Scenario: Build Forge VM with Docker
    Given Docker is available
    And the shared kernel Image exists at "vm/sql/Image"
    When I build the Forge VM
    Then the Docker image "sovereign-forge" should exist
    And the rootfs should be created at "vm/forgejo/rootfs.img"
    And the data disk should be created at "vm/forgejo/data.img"

  @build @dockerfile
  Scenario: Dockerfile uses verified Alpine version
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should use Alpine version "3.21"
    And the community repository should be enabled

  @build @dockerfile @forgejo
  Scenario: Dockerfile installs Forgejo from community repo
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should install "forgejo" package
    And the Dockerfile should install "git" package
    And the Dockerfile should install "openssh-server" package
    And the Dockerfile should NOT install "openrc" package
    # Note: OpenRC hangs in AVF - we use custom init.sh

  @build @dockerfile @tailscale
  Scenario: Dockerfile installs Tailscale static binary
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should download Tailscale version "1.78.3"
    And the Tailscale binaries should be at "/usr/bin/tailscale" and "/usr/sbin/tailscaled"
    # Note: Static binary from pkgs.tailscale.com, not Alpine package

  @build @rootfs @init
  Scenario: Rootfs preparation creates init script
    Given the Docker build completed
    When the rootfs is prepared for AVF
    Then "/sbin/init.sh" should be created from "vm/forgejo/init.sh"
    And "/sbin/init.sh" should be executable
    And device nodes should be pre-created in "/dev"

  @build @data-disk
  Scenario: Build creates data disk if not exists
    Given Docker is available
    And no data disk exists at "vm/forgejo/data.img"
    When I build the Forge VM
    Then a 4GB data disk should be created
    And the data disk should be formatted as ext4

  @build @data-disk @idempotent
  Scenario: Build preserves existing data disk
    Given Docker is available
    And a data disk already exists at "vm/forgejo/data.img"
    When I build the Forge VM
    Then the existing data disk should not be overwritten
    And I should see "Data disk already exists, skipping"

  @build @error
  Scenario: Build fails without Docker
    Given Docker is not available
    When I try to build the Forge VM
    Then the build should fail with error containing "docker not found"

  @build @error
  Scenario: Build fails without shared kernel
    Given Docker is available
    But the shared kernel Image does not exist at "vm/sql/Image"
    When I try to build the Forge VM
    Then the build should fail with error containing "Build SQL VM first"

  # ==========================================================================
  # DEPLOY BEHAVIORS
  # Source: forge.go Deploy()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @deploy
  Scenario: Deploy Forge VM to device
    Given a device is connected
    And the Forge VM is built
    When I deploy the Forge VM
    Then the VM directory should exist on device at "/data/sovereign/vm/forgejo"
    And "rootfs.img" should be pushed to device
    And "Image" should be pushed to device
    And "data.img" should be pushed to device
    And the start script should exist at "/data/sovereign/vm/forgejo/start.sh"

  @deploy @networking
  Scenario: Deploy creates TAP networking script
    Given a device is connected
    And the Forge VM is built
    When I deploy the Forge VM
    Then the start script should configure TAP "vm_forge" with IP "192.168.101.1/24"
    And the start script should NOT use gvproxy or vsock

  @deploy @start-script
  Scenario: Start script configures TAP networking
    Given a device is connected
    And the Forge VM is deployed
    Then the start script should configure TAP "vm_forge" with IP "192.168.101.1/24"
    And the start script should enable IP forwarding
    And the start script should add Android routing bypass rule
    And the start script should configure NAT masquerade to wlan0
    And the start script should add FORWARD rules for vm_forge

  @deploy @start-script @console
  Scenario: Start script uses ttyS0 for console
    Given a device is connected
    And the Forge VM is deployed
    Then the kernel params should contain "console=ttyS0"
    And the kernel params should contain "init=/sbin/init.sh"
    # Note: crosvm --serial captures ttyS0, NOT virtio hvc0

  @deploy @env
  Scenario: Deploy pushes .env file if exists
    Given a device is connected
    And the Forge VM is built
    And a .env file exists with TAILSCALE_AUTHKEY
    When I deploy the Forge VM
    Then ".env" should be pushed to "/data/sovereign/.env"

  # ==========================================================================
  # START BEHAVIORS
  # Source: lifecycle.go Start(), streamBootAndWaitForForgejo()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @start
  Scenario: Start Forge VM successfully
    Given a device is connected
    And the Forge VM is deployed
    When I start the Forge VM
    Then the VM process should be running
    And the TAP interface "vm_forge" should be UP
    And the console log should contain "INIT COMPLETE"

  @start @init @networking
  Scenario: Init script configures guest networking
    Given a device is connected
    And the Forge VM is deployed
    When I start the Forge VM
    Then the guest should have IP "192.168.101.2"
    And the guest should be able to ping "8.8.8.8"

  @start @init @tailscale
  Scenario: Init script starts Tailscale with userspace networking
    Given a device is connected
    And the Forge VM is deployed
    And a valid TAILSCALE_AUTHKEY is configured
    When I start the Forge VM
    Then Tailscale should connect with hostname "sovereign-forge"
    And Tailscale should use userspace networking mode
    And "tailscale serve" should expose port 3000
    And "tailscale serve" should expose port 22

  @start @tailscale @persistent
  Scenario: Start uses persistent Tailscale identity
    Given a device is connected
    And the Forge VM is deployed
    And Tailscale state exists on the data disk
    When I start the Forge VM
    Then Tailscale should reconnect using saved state
    And no new registration should be created
    And I should see "Found persistent state, reconnecting"

  @start @tailscale @idempotent
  Scenario: Restart does not create duplicate Tailscale registration
    Given a device is connected
    And the Forge VM is running
    And Tailscale is connected as "sovereign-forge"
    When I stop the Forge VM
    And I start the Forge VM
    Then only one "sovereign-forge" registration should exist
    And the Tailscale IP should be the same as before

  @start @database
  Scenario: Start waits for PostgreSQL database
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is deployed
    When I start the Forge VM
    Then the init should wait for PostgreSQL on port 5432
    And the init should show "PostgreSQL is ready"
    # Note: Forgejo connects to sovereign-sql for its database

  @start @forgejo
  Scenario: Start launches Forgejo web service
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is deployed
    When I start the Forge VM
    Then Forgejo should be listening on port 3000
    And Forgejo should be exposed via Tailscale

  @start @idempotent
  Scenario: Start is idempotent when VM already running
    Given a device is connected
    And the Forge VM is running
    When I start the Forge VM
    Then the command should succeed
    And I should see "VM already running"
    And no new process should be started

  @start @timeout
  Scenario: Start times out after 120 seconds
    Given a device is connected
    And the Forge VM is deployed
    And the VM will not become ready
    When I start the Forge VM
    Then the start should fail with "timeout waiting for Forgejo (120s)"

  @start @error
  Scenario: Start fails on kernel panic
    Given a device is connected
    And the Forge VM is deployed
    And the init script will crash
    When I start the Forge VM
    Then the start should fail with "VM boot failed"
    And the console should contain "Kernel panic"

  @start @error
  Scenario: Start fails if VM process dies
    Given a device is connected
    And the Forge VM is deployed
    And the VM process will die during boot
    When I start the Forge VM
    Then the start should fail with "VM process died during boot"

  @start @supervision
  Scenario: Init script supervises Forgejo and Tailscale
    Given a device is connected
    And the Forge VM is running
    When Forgejo crashes
    Then the supervision loop should restart Forgejo
    And Forgejo should become available again

  @start @supervision @tailscale
  Scenario: Init script restarts Tailscale if it dies
    Given a device is connected
    And the Forge VM is running
    When Tailscaled process dies
    Then the supervision loop should restart Tailscaled
    And Tailscale should reconnect
    And port 3000 and 22 should be re-exposed

  # ==========================================================================
  # STOP BEHAVIORS
  # Source: lifecycle.go Stop()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @stop
  Scenario: Stop running Forge VM
    Given a device is connected
    And the Forge VM is running
    When I stop the Forge VM
    Then the VM process should not be running
    And the TAP interface "vm_forge" should be removed
    And the socket file should be removed
    And the pid file should be removed
    And I should see "VM stopped"

  @stop @networking
  Scenario: Stop cleans up networking
    Given a device is connected
    And the Forge VM is running
    When I stop the Forge VM
    Then NAT rules for 192.168.101.0/24 should be removed
    And FORWARD rules for vm_forge should be removed
    And I should see "Cleaning up networking"

  @stop @idempotent
  Scenario: Stop is idempotent when VM not running
    Given a device is connected
    And the Forge VM is not running
    When I stop the Forge VM
    Then the command should succeed
    And I should see "VM not running"

  @stop @force-kill
  Scenario: Stop force kills unresponsive VM
    Given a device is connected
    And the Forge VM is running but unresponsive
    When I stop the Forge VM
    Then the VM should be force killed with SIGKILL
    And the VM process should not be running

  # ==========================================================================
  # TEST BEHAVIORS
  # Source: verify.go Test()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @test
  Scenario: Test healthy Forge VM
    Given a device is connected
    And the Forge VM is running
    And the TAP interface "vm_forge" is UP
    And Forgejo web UI is responding on port 3000
    When I test the Forge VM
    Then I should see "VM process running: ✓ PASS"
    And I should see "TAP interface (vm_forge): ✓ PASS"
    And I should see "Tailscale connected: ✓ PASS"
    And I should see "Forgejo web UI (via Tailscale): ✓ PASS"
    And I should see "ALL TESTS PASSED"

  @test @tailscale
  Scenario: Test includes Tailscale connectivity
    Given a device is connected
    And the Forge VM is running
    And Tailscale is connected
    When I test the Forge VM
    Then I should see "Tailscale connected: ✓ PASS"
    And the Tailscale IP should be displayed
    And the hostname "sovereign-forge" should be displayed

  @test @web-ui
  Scenario: Test checks Forgejo web UI
    Given a device is connected
    And the Forge VM is running
    And Forgejo is responding on port 3000
    When I test the Forge VM
    Then I should see "Forgejo web UI (via Tailscale): ✓ PASS"
    # Note: HTTP 302/303 redirects are acceptable (needs initial setup)

  @test @ssh
  Scenario: Test checks SSH port
    Given a device is connected
    And the Forge VM is running
    And SSH is listening on port 22
    When I test the Forge VM
    Then I should see "SSH port (via Tailscale): ✓ PASS"

  @test @error
  Scenario: Test fails when VM not running
    Given a device is connected
    And the Forge VM is not running
    When I test the Forge VM
    Then I should see "VM process running: ✗ FAIL"
    And the test should fail

  @test @error
  Scenario: Test fails when TAP is down
    Given a device is connected
    And the Forge VM is running
    But the TAP interface "vm_forge" is DOWN
    When I test the Forge VM
    Then I should see "TAP interface (vm_forge): ✗ FAIL"
    And the test should fail

  @test @error @tailscale
  Scenario: Test fails when Tailscale not connected
    Given a device is connected
    And the Forge VM is running
    But Tailscale is not connected
    When I test the Forge VM
    Then I should see "Tailscale connected: ✗ FAIL"
    And the test should fail

  # ==========================================================================
  # REMOVE BEHAVIORS
  # Source: lifecycle.go Remove()
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @remove
  Scenario: Remove deployed Forge VM
    Given a device is connected
    And the Forge VM is deployed
    And the Forge VM is not running
    When I remove the Forge VM
    Then the directory "/data/sovereign/vm/forgejo" should be deleted
    And the command should succeed

  @remove
  Scenario: Remove stops running VM first
    Given a device is connected
    And the Forge VM is running
    When I remove the Forge VM
    Then the VM should be stopped first
    And the directory "/data/sovereign/vm/forgejo" should be deleted

  @remove @tailscale
  Scenario: Remove cleans up Tailscale registration
    Given a device is connected
    And the Forge VM is deployed
    And Tailscale is registered as "sovereign-forge"
    When I remove the Forge VM
    Then the Tailscale registration should be removed
    And I should see "Removing Tailscale registration"

  @remove @data
  Scenario: Remove deletes all VM data
    Given a device is connected
    And the Forge VM is deployed
    When I remove the Forge VM
    Then "rootfs.img" should be deleted
    And "data.img" should be deleted
    And "Image" should be deleted
    And "start.sh" should be deleted
    And "console.log" should be deleted

  @remove @idempotent
  Scenario: Remove is idempotent when VM not deployed
    Given a device is connected
    And the Forge VM is not deployed
    When I remove the Forge VM
    Then the command should succeed

  # ==========================================================================
  # MULTI-VM SCENARIOS
  # Source: Integration tests
  # TEAM_025: Re-enabled and updated 2025-12-29
  # ==========================================================================

  @multi-vm
  Scenario: Forge and SQL VMs run simultaneously
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is running
    Then both VMs should have separate TAP interfaces
    And vm_sql should use 192.168.100.x
    And vm_forge should use 192.168.101.x
    And both VMs should have separate Tailscale IPs

  @multi-vm @database
  Scenario: Forge connects to SQL via Tailscale
    Given a device is connected
    And the SQL VM is running as "sovereign-sql"
    And the Forge VM is running
    Then Forgejo should connect to PostgreSQL at "sovereign-sql:5432"
    And database operations should succeed

  @multi-vm @restart
  Scenario: Forge reconnects to SQL after SQL restart
    Given a device is connected
    And both SQL and Forge VMs are running
    When I restart the SQL VM
    Then Forge should reconnect to PostgreSQL
    And Forgejo should continue working
