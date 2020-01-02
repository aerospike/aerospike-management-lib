package system

import (
	"bytes"
	"strings"
	"testing"
)

type pstruct struct {
	p     string
	valid bool
}

var suffixes = []string{"good day", "", "password:", "  sudo   :"}
var prompts = []pstruct{
	{"[sudo] password for root: ", true},
	{"sudo password for long long long long long long long long name:", true},
	{"not a prompt", false},
	{"whatevs", false},
	{"", false},
}

type processPrompt func(string, *Sudo) string

func testSudoPrompt(fn processPrompt, t *testing.T) {
	sudo := NewSudoWithPassword("root")

	for _, prompt := range prompts {
		for _, suffix := range suffixes {
			p := prompt.p + suffix
			s := fn(p, sudo)

			var ss string
			if prompt.valid {
				ss = strings.TrimSpace(suffix)
			} else {
				ss = strings.TrimSpace(p)
			}

			if s != ss {
				t.Fatalf("failed to remove prompt from %q: want %q, got %q", p, ss, s)
			}
		}
	}
}
func TestSudoPrompt(t *testing.T) {
	fn := func(s string, sudo *Sudo) string {
		s = sudo.removePrompt(s)
		s = strings.TrimSpace(s)
		return s
	}
	testSudoPrompt(fn, t)
}

func TestSudoPromptReader(t *testing.T) {
	fn := func(s string, sudo *Sudo) string {
		r := sudo.removePromptInReader(strings.NewReader(s))
		buf := new(bytes.Buffer)
		buf.ReadFrom(r)

		s = strings.TrimSpace(buf.String())
		return s
	}
	testSudoPrompt(fn, t)
}
