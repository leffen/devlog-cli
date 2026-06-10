// Package punch implements a lightweight local "punch clock" for timestamping
// work on a project. Records are stored as newline-delimited JSON (JSONL) in the
// config directory so they can be appended cheaply and read back in order.
package punch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/leffen/devlog-cli/internal/config"
)

// FileName is the name of the JSONL store inside the config directory.
const FileName = "punches.jsonl"

// Record is a single timestamped work punch.
type Record struct {
	Timestamp time.Time `json:"timestamp"`
	Machine   string    `json:"machine"`
	IP        string    `json:"ip"`
	Dir       string    `json:"dir"`
	Project   string    `json:"project"`
	Comment   string    `json:"comment,omitempty"`
}

// StorePath returns the full path to the punch store.
func StorePath() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// SanitizeProject derives a safe project slug from a directory name.
// It lowercases, replaces runs of non-alphanumeric characters with a single
// hyphen, and trims leading/trailing hyphens.
func SanitizeProject(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

// ProjectFromDir returns the sanitized project name for a directory path,
// based on its base name.
func ProjectFromDir(dir string) string {
	return SanitizeProject(filepath.Base(dir))
}

// outboundIP returns the preferred outbound IP of this machine. It opens a UDP
// socket to a public address (no packets are actually sent) to let the OS pick
// the source interface. Falls back to an empty string if it cannot be resolved.
func outboundIP() string {
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

// NewRecord builds a Record for the current machine and working directory.
// If project is empty it is derived from the current directory name. The
// timestamp is taken from now.
func NewRecord(project, comment string) (*Record, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	host, err := os.Hostname()
	if err != nil {
		host = ""
	}

	if project == "" {
		project = ProjectFromDir(dir)
	} else {
		project = SanitizeProject(project)
	}

	return &Record{
		Timestamp: time.Now(),
		Machine:   host,
		IP:        outboundIP(),
		Dir:       dir,
		Project:   project,
		Comment:   strings.TrimSpace(comment),
	}, nil
}

// Append writes a record to the store, creating the directory and file if
// needed.
func Append(rec *Record) error {
	path, err := StorePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open punch store: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to encode record: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	return nil
}

// Load reads all records from the store in chronological (file) order. A
// missing store is treated as empty, not an error. Malformed lines are skipped.
func Load() ([]Record, error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open punch store: %w", err)
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	// Allow long lines (large comments).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read punch store: %w", err)
	}

	return records, nil
}
