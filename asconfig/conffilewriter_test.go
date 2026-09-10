package asconfig

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteFieldBenchmarkFlags(t *testing.T) {
	for _, key := range BenchmarkConfigs {
		t.Run(key+"/false", func(t *testing.T) {
			buf := &bytes.Buffer{}
			writeField(buf, key, "false", 0)

			if buf.Len() != 0 {
				t.Errorf("expected %s=false to be omitted, got %q", key, buf.String())
			}
		})

		t.Run(key+"/true", func(t *testing.T) {
			buf := &bytes.Buffer{}
			writeField(buf, key, "true", 0)

			if !strings.Contains(buf.String(), key) {
				t.Errorf("expected %s=true to be written, got %q", key, buf.String())
			}
		})
	}
}
