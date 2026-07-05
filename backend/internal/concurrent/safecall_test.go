package concurrent

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeCall_Success(t *testing.T) {
	if err := SafeCall(func() error { return nil }); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSafeCall_Error(t *testing.T) {
	wantErr := errors.New("fetch failed")
	if err := SafeCall(func() error { return wantErr }); !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestSafeCall_Panic(t *testing.T) {
	err := SafeCall(func() error { panic("boom") })
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want message containing %q", err, "boom")
	}
}
