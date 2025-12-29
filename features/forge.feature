Feature: Forge VM Lifecycle
  As a developer
  I want to manage the Forgejo VM
  So that I can run git hosting on my Android device

  # Build scenarios
  @build
  Scenario: Build Forge VM with Docker
    Given Docker is available
    And the shared kernel exists at "vm/sql/Image"
    When I build the Forge VM
    Then the Docker image "sovereign-forge" should exist
    And the rootfs should be created at "vm/forgejo/rootfs.img"

  @build @error
  Scenario: Build fails without shared kernel
    Given Docker is available
    But the shared kernel does not exist
    When I try to build the Forge VM
    Then the build should fail with error containing "Build SQL VM first"

  # Deploy scenarios
  @deploy
  Scenario: Deploy Forge VM to device
    Given a device is connected
    And the Forge VM is built
    When I deploy the Forge VM
    Then the VM directory should exist on device at "/data/sovereign/forgejo"
    And the start script should exist at "/data/sovereign/forgejo/start.sh"

  @deploy
  Scenario: Forge uses unique subnet
    Given a device is connected
    And the Forge VM is deployed
    Then the Forge TAP IP should be "192.168.101.1"
    And the Forge TAP IP should differ from SQL TAP IP

  # Start scenarios
  @start
  Scenario: Start Forge VM
    Given a device is connected
    And the Forge VM is deployed
    When I start the Forge VM
    Then the VM process should be running
    And the TAP interface "vm_forge" should be UP

  @start @idempotent
  Scenario: Start is idempotent when VM already running
    Given a device is connected
    And the Forge VM is running
    When I start the Forge VM
    Then the command should succeed

  # Stop scenarios
  @stop
  Scenario: Stop running Forge VM
    Given a device is connected
    And the Forge VM is running
    When I stop the Forge VM
    Then the VM process should not be running

  @stop @idempotent
  Scenario: Stop is idempotent when VM not running
    Given a device is connected
    And the Forge VM is not running
    When I stop the Forge VM
    Then the command should succeed

  # Test scenarios
  @test
  Scenario: Test healthy Forge VM
    Given a device is connected
    And the Forge VM is running
    And Forgejo web UI is responding on port 3000
    When I test the Forge VM
    Then all tests should pass

  # Remove scenarios
  @remove
  Scenario: Remove Forge VM
    Given a device is connected
    And the Forge VM is deployed
    When I remove the Forge VM
    Then the VM directory should not exist on device
