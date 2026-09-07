package deployment

import (
	"errors"
	"testing"
)

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
			name:     "the feature is not configured on this node",
			resp:     "ERROR:4:'index-checkpoint-path' is not configured",
			expected: CheckpointSaveNothingToDo,
		},
		{
			// A configured node whose namespaces have ALL opted out does not refuse: it
			// accepts, parks, and copies nothing. Classifying it as NothingToDo would
			// leave a node parked out of the cluster with nobody coming to reap it.
			name:     "configured but every namespace opted out - accepts and parks",
			resp:     "ok",
			expected: CheckpointSaveTriggered,
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
		expectedNSs         map[string]CheckpointNamespaceStatus
		name                string
		resp                string
		expectedMS          int64
		expectErr           bool
		expectNotConfigured bool
		expectedParked      bool
	}{
		{
			name: "two namespaces, one saved and one still copying",
			resp: "ns1:state=done:files_completed=42:files_total=42:is_parked=false:park_ms=0;" +
				"ns2:state=copying:files_completed=20:files_total=42:is_parked=false:park_ms=0",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"ns1": {State: CheckpointStateDone, FilesCompleted: 42, FilesTotal: 42},
				"ns2": {State: CheckpointStateCopying, FilesCompleted: 20, FilesTotal: 42},
			},
			expectedMS: 0,
		},
		{
			name: "two namespaces, parked once both finished",
			resp: "ns1:state=done:files_completed=42:files_total=42:is_parked=true:park_ms=1500;" +
				"ns2:state=failed:files_completed=3:files_total=42:is_parked=true:park_ms=1500",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"ns1": {State: CheckpointStateDone, FilesCompleted: 42, FilesTotal: 42},
				"ns2": {State: CheckpointStateFailed, FilesCompleted: 3, FilesTotal: 42},
			},
			expectedParked: true,
			expectedMS:     1500,
		},
		{
			// A configured node with NO checkpointing namespace still parks on a
			// checkpoint-save, and reports only this. It must parse as a successful,
			// parked, empty result — a caller that reads it as "nothing to do" leaves the
			// node parked out of the cluster until its park times out.
			name:           "every namespace opted out - parked, no namespace record",
			resp:           "is_parked=true:park_ms=1500",
			expectedNSs:    map[string]CheckpointNamespaceStatus{},
			expectedParked: true,
			expectedMS:     1500,
		},
		{
			// The same node before any save. Empty AND not parked — the one shape that
			// means "nothing to wait for and nothing to reap".
			name:        "every namespace opted out - not parked",
			resp:        "is_parked=false:park_ms=0",
			expectedNSs: map[string]CheckpointNamespaceStatus{},
			expectedMS:  0,
		},
		{
			name: "copying, not yet parked",
			resp: "test:state=copying:files_completed=7:files_total=42:is_parked=false:park_ms=0",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateCopying, FilesCompleted: 7, FilesTotal: 42},
			},
			expectedMS: 0,
		},
		{
			name: "unparseable park_ms does not discard the state",
			resp: "test:state=done:files_completed=1:files_total=1:is_parked=true:park_ms=soon",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateDone, FilesCompleted: 1, FilesTotal: 1},
			},
			expectedParked: true,
			expectedMS:     -1,
		},
		{
			name: "not yet triggered",
			resp: "test:state=none:files_completed=0:files_total=0",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateNone, FilesCompleted: 0, FilesTotal: 0},
			},
			expectedMS: -1,
		},
		{
			name: "failed namespace",
			resp: "test:state=failed:files_completed=3:files_total=42",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateFailed, FilesCompleted: 3, FilesTotal: 42},
			},
			expectedMS: -1,
		},
		{
			name: "trailing separator is tolerated",
			resp: "test:state=done:files_completed=1:files_total=1;",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateDone, FilesCompleted: 1, FilesTotal: 1},
			},
			expectedMS: -1,
		},
		{
			// A namespace legally named "errors" must not be mistaken for a rejection.
			name: "namespace named like an error",
			resp: "errors:state=done:files_completed=1:files_total=1",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"errors": {State: CheckpointStateDone, FilesCompleted: 1, FilesTotal: 1},
			},
			expectedMS: -1,
		},
		{
			name: "unrecognised state is preserved",
			resp: "test:state=verifying:files_completed=10:files_total=42",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: "verifying", FilesCompleted: 10, FilesTotal: 42},
			},
			expectedMS: -1,
		},
		{
			// One bad entry must not hide the good ones.
			name: "malformed entry is skipped, not fatal",
			resp: "garbage;ns2:state=done:files_completed=1:files_total=1",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"ns2": {State: CheckpointStateDone, FilesCompleted: 1, FilesTotal: 1},
			},
			expectedMS: -1,
		},
		{
			// Progress counters are advisory; a malformed one must not discard the
			// state reported alongside it.
			name: "unparseable counters do not discard the state",
			resp: "test:state=copying:files_completed=x:files_total=y",
			expectedNSs: map[string]CheckpointNamespaceStatus{
				"test": {State: CheckpointStateCopying, FilesCompleted: -1, FilesTotal: -1},
			},
			expectedMS: -1,
		},
		{
			// A settled answer, so it gets its own sentinel rather than a generic error:
			// the caller must stop polling, not retry.
			name:                "the feature is not configured on this node",
			resp:                "ERROR:4:'index-checkpoint-path' is not configured",
			expectErr:           true,
			expectNotConfigured: true,
		},
		{
			name:      "any other rejection is an error",
			resp:      "ERROR:22:checkpoint-save in progress - only checkpoint-status is available",
			expectErr: true,
		},
		{
			// Not the same as "nothing is checkpointing" — that node answers with its park
			// state. Empty means the reply was lost, so the caller must retry rather than
			// conclude the node checkpoints nothing.
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
					t.Fatalf("ParseCheckpointStatus(%q) expected an error, got %+v", tt.resp, got)
				}

				if gotNC := errors.Is(err, ErrCheckpointNotConfigured); gotNC != tt.expectNotConfigured {
					t.Errorf("ParseCheckpointStatus(%q) ErrCheckpointNotConfigured = %v, expected %v (err %v)",
						tt.resp, gotNC, tt.expectNotConfigured, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseCheckpointStatus(%q) unexpected error: %v", tt.resp, err)
			}

			if got.IsParked != tt.expectedParked || got.ParkMS != tt.expectedMS {
				t.Errorf("ParseCheckpointStatus(%q) park = {%v, %d}, expected {%v, %d}",
					tt.resp, got.IsParked, got.ParkMS, tt.expectedParked, tt.expectedMS)
			}

			if len(got.Namespaces) != len(tt.expectedNSs) {
				t.Fatalf("ParseCheckpointStatus(%q) namespaces = %v, expected %v",
					tt.resp, got.Namespaces, tt.expectedNSs)
			}

			for ns, want := range tt.expectedNSs {
				if got.Namespaces[ns] != want {
					t.Errorf("ParseCheckpointStatus(%q)[%q] = %+v, expected %+v",
						tt.resp, ns, got.Namespaces[ns], want)
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
