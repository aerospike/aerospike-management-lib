package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/info"
	aero "github.com/ashishshinde/aerospike-client-go/v6"
	"github.com/go-logr/logr"
)

var ParsedData lib.Stats

var err error

var AsInfo *info.AsInfo

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
	_ = writeToFile(ParsedData, "as_info.json")
}

func TestMain(m *testing.M) {

	AsInfo, err = NewAsInfo()
	if err != nil {
		fmt.Println(err)
	}

	m.Run()

	fmt.Println("Run finished")
}

// Info return the asinfo connection to the host. This is pipelined
// asinfo connection object.
func NewAsInfo() (*info.AsInfo, error) {
	p := aero.NewClientPolicy()
	host := AerospikeHost()
	log := logr.Discard()
	return info.NewAsInfo(log, &host, p), nil
}

// AerospikeHost returns the aerospike host
func AerospikeHost() aero.Host {
	return aero.Host{
		Name: "127.0.0.1",
		Port: 3004,
	}
}

// TODO: REMOVE IT
func writeToFile(m interface{}, filename string) error {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	_ = enc.Encode(m)
	err := ioutil.WriteFile(filename, buf.Bytes(), 0644)
	return err
}

//SUD {{<nil> 0 [] [] []} sud [0xae59b0] 0xae5950 <nil>  [] 0s} {sud123 WithPasswordSudoMode}
