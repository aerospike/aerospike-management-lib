package system

import (
	"errors"
	"fmt"
	"strings"
)

// osFacts captures the facts on the system about the OS.
type osFacts struct {
	osname string
	osver  string

	err error // error on running the command
}

// newOSFacts returns a fact OS.
func newOSFacts() *osFacts {
	return &osFacts{}
}

// Sudo returns if the command requires sudo.
func (os *osFacts) Sudo() bool {
	return false
}

// Cmd returns the command to run to fetch the os facts
func (os *osFacts) Cmd() string {
	// https://unix.stackexchange.com/questions/92199/how-can-i-reliably-get-the-operating-systems-name/92218#92218
	// https://www.novell.com/coolsolutions/feature/11251.html

	return "OS=`uname -s` \n" + `
	if [ "{$OS}" = "Linux" ] ; then 
		>&2 echo "system not linux based"
		exit -1
	fi

	if type lsb_release 1>/dev/null ; then
		echo "lsb"
		lsb_release -ir
	elif [ -f /etc/centos-release ] ; then
		echo "centos"
		cat /etc/centos-release
	elif [ -f /etc/redhat-release ] ; then
		echo "redhat"
		cat /etc/redhat-release
	elif [ -f /etc/lsb-release ] ; then
		echo "ubuntu"
		cat /etc/lsb-release
	elif [ -f /etc/os-release ] ; then
		echo "debian"
		cat /etc/os-release
	fi`
}

// Parse parses the output returned by running the command.
func (os *osFacts) Parse(stdout, stderr string, err error) {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if e := parseErr(stderr, err); e != nil {
		os.err = fmt.Errorf("failed to parse error of system command: %v", e)
		return
	}

	lines := strings.Split(stdout, "\n")
	hint := lines[0]

	switch hint {
	case "ubuntu":
		os.processUbuntu(lines[1:])
	case "debian":
		os.processDebian(lines[1:])
	case "centos", "redhat":
		os.processRedhat(lines[1:])
	case "lsb":
		os.processLsb(lines[1:])
	default:
		os.err = fmt.Errorf("unknown os returned from system command %s", hint)
	}
}

// Facts returns the os name and version
func (os *osFacts) Facts() (name, version string) {
	name = os.osname
	version = os.osver

	return
}

// Err returns any error encountered while parsing the output.
func (os *osFacts) Err() error {
	return os.err
}

func (os *osFacts) set(name, ver string) {
	if len(name) == 0 || len(ver) == 0 {
		os.err = errors.New("failed to parse system command output")
	}

	f := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ToLower(s)
		return s
	}

	os.osname = f(name)
	os.osver = f(ver)
}

func (os *osFacts) processLsb(out []string) {
	var name, ver string
	// Distributor ID:	Ubuntu
	// Release:	16.04
	for _, s := range out {
		kv := strings.Split(s, ":")
		if len(kv) != 2 {
			continue
		}

		k := kv[0]
		v := strings.TrimSpace(kv[1])

		if strings.Contains(k, "Distributor") {
			name = v
		} else if strings.Contains(k, "Release") {
			ver = v
		}
	}

	os.set(name, ver)
}

func (os *osFacts) processUbuntu(out []string) {
	var name, ver string

	// DISTRIB_ID=Ubuntu
	// DISTRIB_RELEASE=14.04
	// DISTRIB_CODENAME=trusty
	// DISTRIB_DESCRIPTION="Ubuntu 14.04.5 LTS"
	for _, s := range out {
		kv := strings.Split(s, "=")
		if len(kv) != 2 {
			continue
		}

		k, v := kv[0], kv[1]
		switch k {
		case "DISTRIB_ID":
			name = v
		case "DISTRIB_RELEASE":
			ver = v
		}
	}

	os.set(name, ver)
}

func (os *osFacts) processDebian(out []string) {
	var ver string

	// PRETTY_NAME = "Debian GNU/Linux 7 (wheezy)"
	// NAME = "Debian GNU/Linux"
	// VERSION_ID = "7"
	// VERSION = "7 (wheezy)"
	// ID = debian
	// ANSI_COLOR = "1;31"
	// HOME_URL = "http://www.debian.org/"
	// SUPPORT_URL = "http://www.debian.org/support/"
	// BUG_REPORT_URL = "http://bugs.debian.org/"
	for _, s := range out {
		kv := strings.Split(s, "=")
		if len(kv) != 2 {
			continue
		}

		k, v := kv[0], kv[1]
		switch k {
		case "VERSION_ID":
			v = strings.TrimSpace(v)
			v = strings.Trim(v, "\"")
			ver = v
		}
	}

	os.set("debian", ver)
}

func (os *osFacts) processRedhat(out []string) {
	// CentOS release 6.9 (Final)
	// CentOS Linux release 7.4.1708 (Core)
	s := strings.Split(out[0], " ")

	name, ver := s[0], ""

	for i := range s {
		if s[i] == "release" {
			ver = s[i+1]
		}
	}

	os.set(name, ver)
}
