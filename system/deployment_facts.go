package system

import (
	"fmt"
	"strings"
)

// netInterfaceFacts captures the facts on the system about the network interfaces.
type netInterfaceFacts struct {
	netInterfaces []string

	err error // error on running the command
}

// newNetInterfaceFacts returns a fact netInterfaceFacts.
func newNetInterfaceFacts() *netInterfaceFacts {
	return &netInterfaceFacts{}
}

// Sudo returns if the command requires sudo.
func (ni *netInterfaceFacts) Sudo() bool {
	return false
}

// Cmd returns the command to run to fetch the network interface facts
func (ni *netInterfaceFacts) Cmd() string {
	// https://www.cyberciti.biz/faq/linux-list-network-cards-command/
	// Discussed with Sunil: /proc/net/dev is good source for all OS. We should try to avoid tool output parsing.
	// But as per Thomas suggestion, /proc/net/dev won't show virtual interfaces.
	// There are not much examples of use of virtual interfaces in production or staging environment.
	// But still in future we might need to consider those cases.
	// FIXME : Try to find out good way to get virtual interfaces

	return "cat /proc/net/dev | grep ':' | cut -d':' -f 1 | tr -d '[:blank:]'"
}

// Parse parses the output returned by running the command.
func (ni *netInterfaceFacts) Parse(stdout, stderr string, err error) {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if e := parseErr(stderr, err); e != nil {
		ni.err = fmt.Errorf("failed to parse error of system command: %v", e)
		return
	}

	ni.netInterfaces = strings.Split(stdout, "\n")
}

// Facts returns the os name and version
func (ni *netInterfaceFacts) Facts() []string {
	return ni.netInterfaces
}

// Err returns any error encountered while parsing the output.
func (ni *netInterfaceFacts) Err() error {
	return ni.err
}
