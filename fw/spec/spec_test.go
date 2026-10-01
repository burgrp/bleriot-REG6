package spec

import (
	"testing"

	"github.com/burgrp/bleriot/lib/shared/inventory"
)

func TestTypeDefinesSixWritableIntegerChannels(t *testing.T) {
	deviceType := Type()
	if err := deviceType.Validate(); err != nil {
		t.Fatalf("Type().Validate() error: %v", err)
	}
	if got, want := len(deviceType.Registers), 6; got != want {
		t.Fatalf("register count = %d, want %d", got, want)
	}

	instance := inventory.Instance{Name: "ssr", Type: deviceType}
	for index, register := range deviceType.Registers {
		wantName := "ssr.channel." + string(rune('1'+index))
		if got := instance.RegistryName(register); got != wantName {
			t.Errorf("register %d Registry name = %q, want %q", index, got, wantName)
		}
		if got, want := register.Tag, uint16(index+1); got != want {
			t.Errorf("register %d tag = %d, want %d", index, got, want)
		}
		if register.Type != inventory.TypeInt {
			t.Errorf("register %d type = %q, want %q", index, register.Type, inventory.TypeInt)
		}
		if register.ReadOnly {
			t.Errorf("register %d is read-only", index)
		}
		if register.Conversion.Decode != nil || register.Conversion.Encode != nil {
			t.Errorf("register %d has an unexpected conversion", index)
		}
		if register.Metadata["min"] != "0" || register.Metadata["max"] != "100" {
			t.Errorf("register %d range metadata = %v, want 0..100", index, register.Metadata)
		}
	}
}
