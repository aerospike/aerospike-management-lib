package deployment

import "testing"

// TestClassifyCheckpointSaveResponse covers every reply the server produces for
// checkpoint-save, plus the no-response case.
func TestClassifyCheckpointSaveResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     string
		expected CheckpointSaveVerdict
	}{
		{
			name:     "won the trigger",
			resp:     "ok",
			expected: CheckpointSaveTriggered,
		},
		{
			name:     "re-issue while copying",
			resp:     "checkpoint-save already in progress",
			expected: CheckpointSaveAccepted,
		},
		{
			name:     "re-issue after every namespace finished",
			resp:     "checkpoint-save already complete",
			expected: CheckpointSaveAccepted,
		},
		{
			// Non-fatal: a save ran. The caller polls, and checkpoint-status names the
			// failed namespaces — which is why this is not its own verdict.
			name:     "a namespace's save failed",
			resp:     "ERROR::checkpoint-save failed - see checkpoint-status",
			expected: CheckpointSaveAccepted,
		},
		{
			name:     "refused before startup completed",
			resp:     "ERROR:22:server is still starting; retry checkpoint-save once ready",
			expected: CheckpointSaveRejected,
		},
		{
			// Retrying cannot change this, so it must not classify as a rejection —
			// a caller that retries a rejection would spin forever.
			name:     "no namespace on this node is checkpointing",
			resp:     "ERROR:4:no namespace is checkpointing - the global 'index-checkpoint-path' is unset",
			expected: CheckpointSaveNothingToDo,
		},
		{
			name:     "malformed timeout parameter",
			resp:     "ERROR:4:checkpoint-save takes only an optional 'timeout' of 1..3600 seconds (got 'bogus')",
			expected: CheckpointSaveRejected,
		},
		{
			// No evidence a save started, so polling on it would spin.
			name:     "empty response",
			resp:     "",
			expected: CheckpointSaveRejected,
		},
		{
			// The trap this ordering exists for. A signal handler won the shutdown
			// claim, so NO checkpoint will be taken — but the text contains
			// "already in progress". Matching in-progress before the error branch
			// reads this as success and polls a node that will never produce one.
			name:     "lost the shutdown race - no checkpoint will be taken",
			resp:     "ERROR::checkpoint-save raced a shutdown already in progress - check checkpoint-status",
			expected: CheckpointSaveRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := ClassifyCheckpointSaveResponse(tt.resp); result != tt.expected {
				t.Errorf("ClassifyCheckpointSaveResponse(%q) = %v, expected %v",
					tt.resp, result, tt.expected)
			}
		})
	}
}

func TestParseCheckpointStatus(t *testing.T) {
	tests := []struct {
		expected  map[string]CheckpointNamespaceStatus
		name      string
		resp      string
		expectErr bool
	}{
		{
			name: "two namespaces, mixed states",
			resp: "ns1:state=done:files=42/42;ns2:state=copying:files=20/42",
			expected: map[string]CheckpointNamespaceStatus{
				"ns1": {State: CheckpointStateDone, FilesDone: 42, FilesTotal: 42},
				"ns2": {State: CheckpointStateCopying, FilesDone: 20, FilesTotal: 42},
			},
		},
		{
			name: "not yet triggered",
			resp: "test:state=none:files=0/0",
			expected: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateNone, FilesDone: 0, FilesTotal: 0},
			},
		},
		{
			name: "failed namespace",
			resp: "test:state=failed:files=3/42",
			expected: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateFailed, FilesDone: 3, FilesTotal: 42},
			},
		},
		{
			name: "trailing separator is tolerated",
			resp: "test:state=done:files=1/1;",
			expected: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateDone, FilesDone: 1, FilesTotal: 1},
			},
		},
		{
			// A namespace legally named "errors" must not be mistaken for a rejection.
			name: "namespace named like an error",
			resp: "errors:state=done:files=1/1",
			expected: map[string]CheckpointNamespaceStatus{
				"errors": {State: CheckpointStateDone, FilesDone: 1, FilesTotal: 1},
			},
		},
		{
			// Reported verbatim rather than coerced, so the caller can decide. A newer
			// server adding a state must not be silently read as one we know.
			name: "unrecognised state is preserved",
			resp: "test:state=verifying:files=10/42",
			expected: map[string]CheckpointNamespaceStatus{
				"test": {State: "verifying", FilesDone: 10, FilesTotal: 42},
			},
		},
		{
			// One bad entry must not hide the good ones.
			name: "malformed entry is skipped, not fatal",
			resp: "garbage;ns2:state=done:files=1/1",
			expected: map[string]CheckpointNamespaceStatus{
				"ns2": {State: CheckpointStateDone, FilesDone: 1, FilesTotal: 1},
			},
		},
		{
			// Progress counters are advisory; a malformed one must not discard the
			// state reported alongside it.
			name: "unparseable counters do not discard the state",
			resp: "test:state=copying:files=x/y",
			expected: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateCopying, FilesDone: -1, FilesTotal: -1},
			},
		},
		{
			name:     "nothing configured is an empty map, not an error",
			resp:     "ERROR:4:no namespace is checkpointing - the global 'index-checkpoint-path' is unset",
			expected: map[string]CheckpointNamespaceStatus{},
		},
		{
			name:      "any other rejection is an error",
			resp:      "ERROR:22:checkpoint-save in progress - only checkpoint-status is available",
			expectErr: true,
		},
		{
			// Not the same as "nothing is checkpointing" — the server answers that with
			// an error. Empty means the reply was lost, so the caller must retry rather
			// than conclude the node checkpoints nothing.
			name:      "empty response is an error, not an empty result",
			resp:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCheckpointStatus(tt.resp)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("ParseCheckpointStatus(%q) expected an error, got %v", tt.resp, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseCheckpointStatus(%q) unexpected error: %v", tt.resp, err)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("ParseCheckpointStatus(%q) = %v, expected %v", tt.resp, got, tt.expected)
			}

			for ns, want := range tt.expected {
				if got[ns] != want {
					t.Errorf("ParseCheckpointStatus(%q)[%q] = %+v, expected %+v",
						tt.resp, ns, got[ns], want)
				}
			}
		})
	}
}

// TestCheckpointNamespaceStatusIsTerminal pins that terminal is matched positively.
// Treating an unrecognised state as terminal would let a caller replace a node
// mid-copy.
func TestCheckpointNamespaceStatusIsTerminal(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{state: CheckpointStateDone, expected: true},
		{state: CheckpointStateFailed, expected: true},
		{state: CheckpointStateCopying, expected: false},
		{state: CheckpointStateNone, expected: false},
		{state: "verifying", expected: false},
		{state: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			s := CheckpointNamespaceStatus{State: tt.state}
			if got := s.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal(state=%q) = %v, expected %v", tt.state, got, tt.expected)
			}
		})
	}
}
