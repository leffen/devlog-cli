// Package machine gathers identifying information about the current host:
// hostname, username, local and public IP addresses, and OS/arch. It relies
// only on the Go standard library so it works the same on macOS, Linux, and
// Windows with no external tools to install.
package machine

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"
)

// Info describes the current machine at a point in time.
type Info struct {
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	LocalIP  string    `json:"local_ip"`
	PublicIP string    `json:"public_ip,omitempty"`
	OS       string    `json:"os"`
	Arch     string    `json:"arch"`
	Time     time.Time `json:"time"`
}

// LocalIP returns the preferred outbound IP of this machine. It opens a UDP
// socket to a public address (no packets are actually sent) to let the OS pick
// the source interface. Returns an empty string if it cannot be resolved.
func LocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}

// PublicIP queries an external service to discover the machine's public IP.
// It returns an empty string (no error) if the machine is offline or the
// lookup times out, so callers can degrade gracefully.
func PublicIP(timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "DevLog-CLI/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// Username returns the current user's login name, falling back to common
// environment variables if the user database is unavailable.
func Username() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	for _, key := range []string{"USER", "USERNAME", "LOGNAME"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

// Gather collects machine info for the current host. When fetchPublic is true
// it also attempts a public IP lookup, bounded by timeout.
func Gather(fetchPublic bool, timeout time.Duration) Info {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}

	info := Info{
		Hostname: host,
		Username: Username(),
		LocalIP:  LocalIP(),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Time:     time.Now(),
	}

	if fetchPublic {
		info.PublicIP = PublicIP(timeout)
	}

	return info
}
