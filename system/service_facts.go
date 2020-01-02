package system

import (
	"fmt"
	"path/filepath"
	"strings"
)

// serviceFacts captures the facts on the system about the service daemon.
type serviceFacts struct {
	name string
	err  error // error on running the command
}

// newServiceFacts returns a service facts.
func newServiceFacts() *serviceFacts {
	return &serviceFacts{}
}

// Sudo returns if the command requires sudo.
func (s *serviceFacts) Sudo() bool {
	return false
}

// Cmd returns the command to run to fetch the os facts
func (s *serviceFacts) Cmd() string {
	// https://linuxconfig.org/detecting-which-system-manager-is-running-on-linux-system
	return "ls -l /sbin/init"
}

// Parse parses the output returned by running the command.
func (s *serviceFacts) Parse(stdout, stderr string, err error) {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if e := parseErr(stderr, err); e != nil {
		s.err = fmt.Errorf("failed to parse error of system command: %v", e)
		return
	}

	// lrwxrwxrwx 1 root root 20 Oct 27 15:41 /sbin/init -> /lib/systemd/systemd
	w := strings.Split(stdout, " ")
	_, t := filepath.Split(w[len(w)-1])

	s.name = strings.TrimSpace(t)
}

// Facts returns the init system.
func (s *serviceFacts) Facts() string {
	return s.name
}

// Err returns any error encountered while running and processing the command.
func (s *serviceFacts) Err() error {
	return s.err
}
