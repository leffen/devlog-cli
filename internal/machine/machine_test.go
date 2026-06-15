package machine

import (
	"net"
	"runtime"
	"testing"
	"time"
)

func TestGatherWithoutPublicIP(t *testing.T) {
	info := Gather(false, time.Second)

	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.PublicIP != "" {
		t.Errorf("PublicIP should be empty when fetchPublic is false, got %q", info.PublicIP)
	}
	if info.Time.IsZero() {
		t.Error("Time should be set")
	}
	// LocalIP, when set, must be a valid IP.
	if info.LocalIP != "" && net.ParseIP(info.LocalIP) == nil {
		t.Errorf("LocalIP %q is not a valid IP", info.LocalIP)
	}
}

func TestUsernameNonEmpty(t *testing.T) {
	if Username() == "" {
		t.Error("expected a non-empty username in the test environment")
	}
}
