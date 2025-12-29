Feature: SQL VM Lifecycle
  As a developer
  I want to manage the PostgreSQL VM
  So that I can run a database on my Android device

  # ==========================================================================
  # BUILD BEHAVIORS
  # Source: sql.go Build() and common.go Build()
  # ==========================================================================

  @build
  Scenario: Build SQL VM with Docker
    Given Docker is available
    And the kernel Image exists
    When I build the SQL VM
    Then the Docker image "sovereign-sql" should exist
    And the rootfs should be created at "vm/sql/rootfs.img"
    And the data disk should be created at "vm/sql/data.img"

  @build @secrets
  Scenario: Build loads existing secrets
    Given Docker is available
    And a .secrets file exists
    When I build the SQL VM
    Then existing credentials should be used
    And no password prompt should appear

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
    Then a 4G data disk should be created
    And the data disk should be formatted as ext4

  @build @data-disk @idempotent
  Scenario: Build preserves existing data disk
    Given Docker is available
    And a data disk already exists at "vm/sql/data.img"
    When I build the SQL VM
    Then the existing data disk should not be overwritten

  @build @error
  Scenario: Build fails without Docker
    Given Docker is not available
    When I try to build the SQL VM
    Then the build should fail with error containing "docker not found"

  @build @error
  Scenario: Build fails without kernel
    Given Docker is available
    But the kernel Image does not exist
    When I try to build the SQL VM
    Then the build should fail with error containing "kernel Image not found"

  # ==========================================================================
  # DEPLOY BEHAVIORS
  # Source: common.go Deploy() and CreateStartScript()
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
    And the start script should add Android routing bypass
    And the start script should configure NAT masquerade
    And the start script should add FORWARD rules

  # ==========================================================================
  # START BEHAVIORS
  # Source: common.go Start(), CheckTailscaleRegistration(), StreamBootAndWait()
  # ==========================================================================

  @start
  Scenario: Start SQL VM successfully
    Given a device is connected
    And the SQL VM is deployed
    And no Tailscale registration exists for "sovereign-sql"
    When I start the SQL VM
    Then the VM process should be running
    And the TAP interface "vm_sql" should be UP
    And the console log should contain "PostgreSQL started"

  @start @idempotent
  Scenario: Start is idempotent when VM already running
    Given a device is connected
    And the SQL VM is running
    When I start the SQL VM
    Then the command should succeed
    And I should see "VM already running"
    And no new process should be started

  @start @tailscale
  Scenario: Start checks Tailscale registration
    Given a device is connected
    And the SQL VM is deployed
    And a Tailscale registration exists for "sovereign-sql"
    When I try to start the SQL VM
    Then the start should fail with "TAILSCALE IDEMPOTENCY CHECK FAILED"

  @start @tailscale @force
  Scenario: Start with force skips Tailscale check
    Given a device is connected
    And the SQL VM is deployed
    And a Tailscale registration exists for "sovereign-sql"
    When I start the SQL VM with force flag
    Then the Tailscale check should be skipped
    And the VM should start

  @start @timeout
  Scenario: Start times out after 90 seconds
    Given a device is connected
    And the SQL VM is deployed
    And the VM will not become ready
    When I start the SQL VM
    Then the start should fail with "timeout"

  @start @error
  Scenario: Start fails on kernel panic
    Given a device is connected
    And the SQL VM is deployed
    And the VM will kernel panic
    When I start the SQL VM
    Then the start should fail with "boot failed"

  @start @error
  Scenario: Start fails if VM process dies
    Given a device is connected
    And the SQL VM is deployed
    And the VM process will die during boot
    When I start the SQL VM
    Then the start should fail with "VM process died during boot"

  # ==========================================================================
  # STOP BEHAVIORS
  # Source: common.go Stop()
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
    And I should see "PostgreSQL responding (via TAP): ✓ PASS"
    And I should see "ALL TESTS PASSED"

  @test @error
  Scenario: Test fails when VM not running
    Given a device is connected
    And the SQL VM is not running
    When I test the SQL VM
    Then I should see "VM process running: ✗ FAIL"
    And the test should fail with "some tests failed"

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

  # ==========================================================================
  # REMOVE BEHAVIORS
  # Source: common.go Remove()
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

  @remove @error-tolerant
  Scenario: Remove continues even if stop fails
    Given a device is connected
    And the SQL VM is in a bad state
    When I remove the SQL VM
    Then I should see a warning about stop failure
    But the VM directory should still be removed
