package sql

import (
	"testing"

	"github.com/anthropics/sovereign/internal/vm"
)

func TestSQLVMRegistered(t *testing.T) {
	// The SQL VM should be auto-registered via init()
	v, ok := vm.Get("sql")
	if !ok {
		t.Fatal("SQL VM should be registered")
	}

	if v.Name() != "sql" {
		t.Errorf("Expected name 'sql', got '%s'", v.Name())
	}
}

func TestSQLVMImplementsInterface(t *testing.T) {
	// Verify VM struct implements vm.VM interface
	var _ vm.VM = (*VM)(nil)
}

func TestVMName(t *testing.T) {
	v := &VM{}
	if v.Name() != "sql" {
		t.Errorf("Expected name 'sql', got '%s'", v.Name())
	}
}
