package system

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
)

var ip = flag.String("ip", "", "ip of machine to connect to")
var port = flag.Int("port", 22, "port to ssh")

var system *System

func TestMain(m *testing.M) {
	flag.Parse()

	if len(*ip) > 0 {
		config := &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.Password("root"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		sudo := NewSudoAsRoot()
		var err error
		logger := logr.Discard()
		system, err = New(logger, *ip, *port, sudo, config)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

	os.Exit(m.Run())
}

func TestSystem(t *testing.T) {
	os := system.OSName()
	if os != "centos" {
		t.Fatalf("os = %q, want %q", os, "ubuntu")
	}
}
