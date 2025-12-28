package vm

import (
	"testing"
)

// MockVM is a test implementation of the VM interface
type MockVM struct {
	name         string
	buildCalled  bool
	deployCalled bool
	startCalled  bool
	stopCalled   bool
	testCalled   bool
	removeCalled bool
}

func (m *MockVM) Name() string  { return m.name }
func (m *MockVM) Build() error  { m.buildCalled = true; return nil }
func (m *MockVM) Deploy() error { m.deployCalled = true; return nil }
func (m *MockVM) Start() error  { m.startCalled = true; return nil }
func (m *MockVM) Stop() error   { m.stopCalled = true; return nil }
func (m *MockVM) Test() error   { m.testCalled = true; return nil }
func (m *MockVM) Remove() error { m.removeCalled = true; return nil }

func TestRegisterAndGet(t *testing.T) {
	// Create a mock VM
	mock := &MockVM{name: "test-vm"}

	// Register it
	Register("test", mock)

	// Get it back
	vm, ok := Get("test")
	if !ok {
		t.Fatal("Expected to find registered VM")
	}

	if vm.Name() != "test-vm" {
		t.Errorf("Expected name 'test-vm', got '%s'", vm.Name())
	}
}

func TestGetNotFound(t *testing.T) {
	_, ok := Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent VM")
	}
}

func TestList(t *testing.T) {
	// Register a test VM
	Register("list-test", &MockVM{name: "list-test"})

	names := List()
	if len(names) == 0 {
		t.Error("Expected at least one VM in list")
	}

	// Check that our test VM is in the list
	found := false
	for _, name := range names {
		if name == "list-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'list-test' in VM list")
	}
}

func TestVMInterface(t *testing.T) {
	mock := &MockVM{name: "interface-test"}
	Register("interface-test", mock)

	vm, _ := Get("interface-test")

	// Test all interface methods
	if err := vm.Build(); err != nil {
		t.Errorf("Build failed: %v", err)
	}
	if !mock.buildCalled {
		t.Error("Build was not called")
	}

	if err := vm.Deploy(); err != nil {
		t.Errorf("Deploy failed: %v", err)
	}
	if !mock.deployCalled {
		t.Error("Deploy was not called")
	}

	if err := vm.Start(); err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if !mock.startCalled {
		t.Error("Start was not called")
	}

	if err := vm.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if !mock.stopCalled {
		t.Error("Stop was not called")
	}

	if err := vm.Test(); err != nil {
		t.Errorf("Test failed: %v", err)
	}
	if !mock.testCalled {
		t.Error("Test was not called")
	}

}

func TestConcurrentAccess(t *testing.T) {
	// Test that concurrent access to registry is safe
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			Register("concurrent-test", &MockVM{name: "concurrent"})
			Get("concurrent-test")
			List()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
