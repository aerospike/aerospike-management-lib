package info

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"

	lib "github.com/aerospike/aerospike-management-lib"
	"golang.org/x/crypto/ssh"
)

//**************************************************************************
// For Testing
//

// // API variant having nodeIP inputs
// func GetSysInfo(nodeIPs ...string) lib.Stats {
// 	cmdList := make([]string, len(RunCmd))
// 	i := 0
// 	for cmd := range RunCmd {
// 		cmdList[i] = cmd
// 		i++
// 	}
// 	return GetSysCmdInfoForNodes(cmdList, nodeIPs)
// }

// func GetSysCmdInfoForNodes(cmdList []string, nodeIPs []string) lib.Stats {
// 	nodesSysMap := make(lib.Stats)
// 	for _, n := range nodeIPs {
// 		sysMap := s.GetSysCmdInfo(cmdList)
// 		nodesSysMap[n] = sysMap
// 	}
// 	writeToFile(nodesSysMap)
// 	fmt.Println(nodesSysMap)
// 	return nodesSysMap
// }

// GetSysCmdInfo fetch and parse info for given commands
func GetSysCmdInfo(ip string, cmdList ...string) NodeSysStats {
	var wg sync.WaitGroup
	var lock = sync.RWMutex{}
	sysMap := make(NodeSysStats)

	wg.Add(len(cmdList))
	for _, cmd := range cmdList {
		go func(cmd string) {
			defer wg.Done()
			cmdOutput := runSysCmd(RunCmd[cmd][0], ip)

			var m lib.Stats
			if cmd == "uname" {
				m = parseUnameInfo(cmdOutput)
			} else if cmd == "meminfo" {
				m = parseMemInfo(cmdOutput)
			} else if cmd == "df" {
				m = parseDfInfo(cmdOutput)
			} else if cmd == "free-m" {
				m = parseFreeMInfo(cmdOutput)
			} else if cmd == "hostname" {
				m = parseHostnameInfo(cmdOutput)
			} else if cmd == "dmesg" {
				m = parseDmesgInfo(cmdOutput)
			} else if cmd == "lscpu" {
				m = parseLscpuInfo(cmdOutput)
			} else if cmd == "iptables" {
				m = parseIptablesInfo(cmdOutput)
			} else if cmd == "sysctlall" {
				m = parseSysctlallInfo(cmdOutput)
			} else if cmd == "hdparm" {
				m = parseHdparmInfo(cmdOutput)
			} else if cmd == "limits" {
				m = parseLimitsInfo(cmdOutput)
			} else if cmd == "interrupts" {
				m = parseInterruptsInfo(cmdOutput)
			} else if cmd == "top" {
				m = parseTopInfo(cmdOutput)
			} else if cmd == "lsb" {
				m = parseLsbInfo(cmdOutput)
			} else if cmd == "iostat" {
				m = parseIostatInfo(cmdOutput)
			} else {
				return
			}
			lock.Lock()
			sysMap[cmd] = m
			lock.Unlock()
		}(cmd)
	}
	wg.Wait()
	return sysMap
}

func isLocalAddr(ip string) bool {
	iAddrs, _ := net.InterfaceAddrs()
	// handle err
	for _, i := range iAddrs {
		if strings.Split(i.String(), "/")[0] == ip {
			//fmt.Println(i.String())
			return true
		}
	}
	return false
}

func runSysCmd(cmd string, nodeIP string) string {
	if nodeIP == "" || isLocalAddr(nodeIP) {
		return runSysCmdLocal(cmd, nodeIP)
	}

	return runSysCmdSSH(cmd, nodeIP)
}

func runSysCmdLocal(cmd string, nodeIP string) string {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Sprintf("Failed to execute command `%s`", cmd)
	}
	return string(out)
}

func runSysCmdSSH(cmd string, nodeIP string) string {
	// An SSH client is represented with a ClientConn.
	//
	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig,
	// and provide a HostKeyCallback.
	config := &ssh.ClientConfig{
		User: "citrusleaf",
		Auth: []ssh.AuthMethod{
			ssh.Password("citrusleaf"),
		},
	}
	client, err := ssh.Dial("tcp", nodeIP+":22", config)
	if err != nil {
		//log.Fatal("Failed to dial: ", err)
		fmt.Println(err)
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		//log.Fatal("Failed to create session: ", err)
		fmt.Println(err)
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		//log.Println("Failed to run: " + err.Error())
		fmt.Println("Failed to run: " + err.Error())
	}
	return b.String()
}
