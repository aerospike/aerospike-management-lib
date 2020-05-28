package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"golang.org/x/crypto/ssh"

	aero "github.com/aerospike/aerospike-client-go"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/aerospike/aerospike-management-lib/system"
)

var runCmd = info.RunCmd

/*
var runCmdKeys = info.RunCmdKeys
var runCmdKeys = []string{"hostname", "top", "lsb", "meminfo", "interrupts", "iostat",
	"dmesg", "limits", "lscpu", "sysctlall", "iptables", "hdparm", "df", "free-m", "uname"}
*/
var runCmdKeys = []string{"hostname", "top", "lsb", "meminfo", "interrupts", "iostat",
	"dmesg", "limits", "lscpu", "sysctlall", "iptables", "hdparm", "df", "free-m", "uname"}

// all 8295 198
// -dmesg 7814 194
// -limits 7120 192
// -top 6607 179
// -lscpu 6038 177
// -sysctl 5134 177
// -meminfo 4488 178
// -hdparm 3983 87------------
// -interrupts 3523 87
// -iptables 3056 86
// -df 2422 86
// -freem 1939 84
// -iostat 1407 84
// -lsb 933 29------------------
// -uname 466 24
//

var ParsedData lib.Stats

var err error

var AsInfo *info.AsInfo
var SysInfo *info.SysInfo

func BenchmarkSysParser__map(b *testing.B) {

	b.ReportAllocs()
	b.ResetTimer()
	//b.N = 2
	for i := 0; i < b.N; i++ {
		ParsedData = SysInfo.GetSysInfo(runCmdKeys...)
	}
	writeToFile(ParsedData, "sys_info.json")
}

func BenchmarkAsParser__map(b *testing.B) {

	b.ReportAllocs()
	b.ResetTimer()
	//b.N = 9000
	for i := 0; i < b.N; i++ {
		ParsedData, err = AsInfo.GetAsInfo()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	writeToFile(ParsedData, "as_info.json")
}

func TestMain(m *testing.M) {

	AsInfo, err = NewAsInfo()
	if err != nil {
		fmt.Println(err)
	}

	SysInfo, err = getSysInfo()
	fmt.Printf("sud: sysinfo: %v, err: %v", SysInfo, err)

	m.Run()

	fmt.Println("Run finished")
}

// Info return the asinfo connection to the host. This is pipelined
// asinfo connection object.
func NewAsInfo() (*info.AsInfo, error) {
	p := aero.NewClientPolicy()
	host := AerospikeHost()
	return info.NewAsInfo(&host, p), nil
}

// AerospikeHost returns the aerospike host
func AerospikeHost() aero.Host {
	return aero.Host{
		Name: "127.0.0.1",
		Port: 3004,
	}
}

func getSysInfo() (*info.SysInfo, error) {
	config := &ssh.ClientConfig{
		User: "sud",
		Auth: []ssh.AuthMethod{
			ssh.Password("sud123"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sudo := system.NewSudoWithPassword("sud123")

	res, err := system.New("127.0.0.1", 22, sudo, config)
	if err != nil {
		return nil, err
	}
	return info.NewSysInfo(res)
}

// TODO: REMOVE IT
func writeToFile(m interface{}, fname string) error {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	_ = enc.Encode(m)
	err := ioutil.WriteFile(fname, buf.Bytes(), 0644)
	return err
}

//SUD {{<nil> 0 [] [] []} sud [0xae59b0] 0xae5950 <nil>  [] 0s} {sud123 WithPasswordSudoMode}
