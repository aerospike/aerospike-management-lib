package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v6"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/info"
)

var ParsedData lib.Stats

var err error

var AsInfo *info.AsInfo

func BenchmarkAsParser__map(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

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
		Name: "172.17.0.3",
		Port: 3000,
	}
}

// TODO: REMOVE IT
func writeToFile(m interface{}, filename string) error {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	_ = enc.Encode(m)
	err := os.WriteFile(filename, buf.Bytes(), 0o600)

	return err
}
