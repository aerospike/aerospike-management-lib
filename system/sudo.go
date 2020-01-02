package system

import (
	"fmt"
	"io"
	"strings"
)

// SudoMode is the mode of the sudo
type SudoMode string

const (
	RootSudoMode         SudoMode = "RootSudoMode"
	NoPasswdSudoMode     SudoMode = "NoPasswdSudoMode"
	DisallowedSudoMode   SudoMode = "DisallowedSudoMode"
	WithPasswordSudoMode SudoMode = "WithPasswordSudoMode"
)

// Sudo captures the privileged execution permissions of the user.
type Sudo struct {
	passwd   string
	sudoMode SudoMode
}

// NewRawSudo creates a sudo with password and mode.
func NewRawSudo(passwd string, sudoMode SudoMode) *Sudo {
	return &Sudo{passwd: passwd, sudoMode: sudoMode}
}

// NewSudoWithPassword creates a sudo with password based access.
func NewSudoWithPassword(passwd string) *Sudo {
	return &Sudo{passwd: passwd, sudoMode: WithPasswordSudoMode}
}

// NewSudoAsRoot creates a sudo which doesn't need to prepend sudo.
func NewSudoAsRoot() *Sudo {
	return &Sudo{sudoMode: RootSudoMode}
}

// NewSudoPasswordLess creates a sudo with passwordless access.
func NewSudoPasswordLess() *Sudo {
	return &Sudo{sudoMode: NoPasswdSudoMode}
}

// NewSudoDisallowed creates a sudo with no sudo privileges.
func NewSudoDisallowed() *Sudo {
	return &Sudo{sudoMode: DisallowedSudoMode}
}

// ToSudoMode convert to the sudo mode
func ToSudoMode(sudo string) (SudoMode, error) {
	switch sudo {
	case "password":
		return WithPasswordSudoMode, nil
	case "root":
		return RootSudoMode, nil
	case "nopasswd":
		return NoPasswdSudoMode, nil
	case "disallowed":
		return DisallowedSudoMode, nil
	default:
		return "", fmt.Errorf("invalid value of sudo %s", sudo)
	}
}

func (s SudoMode) String() string {
	switch s {
	case WithPasswordSudoMode:
		return "password"
	case RootSudoMode:
		return "root"
	case NoPasswdSudoMode:
		return "nopasswd"
	case DisallowedSudoMode:
		return "disallowed"
	default:
		return ""
	}
}

func (s *Sudo) disallowed() bool {
	return s.sudoMode == DisallowedSudoMode
}

func (s *Sudo) isRoot() bool {
	return s.sudoMode == RootSudoMode
}

func (s *Sudo) noPasswd() bool {
	return s.sudoMode == NoPasswdSudoMode
}

// canSudo returns true iff sudo privileges exist.
func (s *Sudo) canSudo() bool {
	return !s.disallowed()
}

// wrap wraps a given command with the sudo privileges.
// Check canSudo before making a call to Wrap.
func (s *Sudo) wrap(cmd string) string {
	if s.disallowed() {
		return cmd
	} else if s.isRoot() {
		return cmd
	} else if s.noPasswd() {
		return "sudo " + cmd
	}

	return fmt.Sprintf("echo %s | sudo -S %s", s.passwd, cmd)
}

func (s *Sudo) passwdPrompt() bool {
	return !s.disallowed() && !s.isRoot() && !s.noPasswd()
}

// removePrompt removes the sudo prompt from out.
// see -S option in man sudo
func (s *Sudo) removePrompt(out string) string {
	if !s.passwdPrompt() {
		return out
	}

	i := strings.Index(out, ":")
	if i == -1 {
		return out
	}

	prefix := strings.ToLower(out[:i])
	if strings.Contains(prefix, "sudo") && strings.Contains(prefix, "password") {
		i++ // exclude colon
		return out[i:]
	}

	return out
}

// removePromptInReader removes the sudo prompt from the out reader.
// see -S option in man sudo
func (s *Sudo) removePromptInReader(out io.Reader) io.Reader {
	if !s.passwdPrompt() {
		return out
	}
	return newSudoFilter(out, s)
}

// sudoFilter removes the sudo password prompt from the stderr reader
type sudoFilter struct {
	in       io.Reader
	consumed bool
	unread   []byte
	sudo     *Sudo
}

func newSudoFilter(in io.Reader, sudo *Sudo) *sudoFilter {
	return &sudoFilter{
		in:   in,
		sudo: sudo,
	}
}

// copyUnread copies the unread bytes to b and returns the number of bytes copied.
func (f *sudoFilter) copyUnread(b []byte) int {
	if len(f.unread) == 0 {
		return 0
	}

	i := 0
	for i = 0; i < len(f.unread) && i < len(b); i++ {
		b[i] = f.unread[i]
	}
	f.unread = f.unread[i:]
	return i
}

// consumePrompt consumes the sudo prompt from the bytes b.
func (f *sudoFilter) consumePrompt(b []byte) {
	f.consumed = true

	s := string(b)
	s = f.sudo.removePrompt(s)
	f.unread = []byte(s)
}

// implements the reader interface
func (f *sudoFilter) Read(b []byte) (int, error) {
	if !f.consumed {
		nread := 0
		buf := make([]byte, 128) // sufficently large to accomodate long user names

		for nread < cap(buf) {
			n, err := f.in.Read(buf[nread:])
			nread += n

			if err != nil {
				f.consumePrompt(buf[:nread])
				i := f.copyUnread(b)
				return i, err
			}
		}

		f.consumePrompt(buf[:nread])
	}

	n := f.copyUnread(b)
	if len(b) == n {
		return n, nil
	}

	m, err := f.in.Read(b[n:])
	return n + m, err
}
