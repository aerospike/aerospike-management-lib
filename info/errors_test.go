package info

import "testing"

func TestIsInfoErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     string
		expected bool
	}{
		{
			name:     "error with a numeric code",
			resp:     "ERROR:4:no namespace is checkpointing",
			expected: true,
		},
		{
			// The generic "unknown" error omits the code, leaving an empty field.
			name:     "error with an omitted code",
			resp:     "ERROR::checkpoint-save failed - see checkpoint-status",
			expected: true,
		},
		{
			name:     "lower-case error prefix",
			resp:     "error:22:server is still starting",
			expected: true,
		},
		{
			name:     "ok response",
			resp:     "ok",
			expected: false,
		},
		{
			name:     "empty response",
			resp:     "",
			expected: false,
		},
		{
			name:     "a payload that merely contains colons",
			resp:     "test:state=done:files=42/42",
			expected: false,
		},
		{
			// Short enough that slicing without a length check would panic.
			name:     "response shorter than the prefix",
			resp:     "err",
			expected: false,
		},
		{
			// The reason the trailing colon is part of the match: a namespace may
			// legally be named "errors", and several payloads lead with a namespace
			// name. Matching only "error" would read this as a rejection.
			name:     "namespace merely named like an error",
			resp:     "errors:state=done:files=1/1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := IsInfoErrorResponse(tt.resp); result != tt.expected {
				t.Errorf("IsInfoErrorResponse(%q) = %v, expected %v", tt.resp, result, tt.expected)
			}
		})
	}
}
