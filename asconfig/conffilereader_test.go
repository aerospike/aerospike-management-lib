package asconfig

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

func TestProcessLoggingContext(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Conf
		wantErr  bool
	}{
		{
			name:     "context with level",
			input:    "context any info",
			expected: Conf{"any": "info"},
		},
		{
			name:    "context without level",
			input:   "context any",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tc.input))

			conf, err := process(logr.Discard(), scanner, Conf{})
			if tc.wantErr {
				if !errors.Is(err, ErrConfigParse) {
					t.Fatalf("expected %v, got %v", ErrConfigParse, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, v := range tc.expected {
				if conf[k] != v {
					t.Errorf("expected %s=%v, got %v", k, v, conf[k])
				}
			}
		})
	}
}
