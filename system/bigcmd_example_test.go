package system_test

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/aerospike/aerospike-management-lib/system"
)

type yes struct {
	nlines int
	err    error
}

func (y *yes) Cmd() string {
	return "yes"
}

func (y *yes) Sudo() bool {
	return false
}

func (y *yes) Nlines() int {
	return y.nlines
}

func (y *yes) Parse(stdin, stderr io.Reader, err <-chan error, cancel <-chan struct{}) {
	process := func() error {
		b := make([]byte, 64)
		n, err := stdin.Read(b)
		for i := 0; i < n; i++ {
			if b[i] == '\n' {
				y.nlines++
			}
		}

		return err
	}

	done := false
	for {
		select {
		case e := <-err:
			y.err = e
			return
		case <-cancel:
			return
		default:
			if done {
				continue
			}

			err := process()
			if err == io.EOF {
				done = true // wait for command to complete
			} else if err != nil {
				y.err = err
			}
		}
	}
}

const ip = "127.0.0.1"
const port = 22

var config *ssh.ClientConfig = &ssh.ClientConfig{
	User: "root",
	Auth: []ssh.AuthMethod{
		ssh.Password("root"),
	},
}

func ExampleSystem_RunBigCmd() {
	sudo := system.NewSudoDisallowed()
	s, err := system.New(ip, port, sudo, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	y := new(yes)
	job, err := s.RunBigCmd(y)

	time.Sleep(time.Second)
	job.Cancel()

	fmt.Println(y.Nlines(), y.err)
	time.Sleep(time.Second)
}
