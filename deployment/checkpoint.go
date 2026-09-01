package deployment

import (
	"fmt"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/info"
)

// Index-checkpoint info commands.
//
// checkpoint-save copies a namespace's volatile shared-memory state (primary index,
// secondary index, and for an in-memory namespace its data stripes) to a durable
// directory, then PARKS the node so an orchestrator can confirm the copy finished before
// replacing the pod. It is a shutdown command: the node leaves the cluster when it fires
// and never serves again.
//
// While parked, the server answers only these two commands on the info port; everything
// else is refused. So any other info work must happen BEFORE checkpoint-save is sent.
const (
	checkpointSaveCmd   = "checkpoint-save"
	checkpointStatusCmd = "checkpoint-status"
)

// Substrings that identify a checkpoint-save response. Matched as substrings rather than
// compared whole because the server wraps some of them in an "ERROR:<code>:" envelope.
const (
	// The idempotent re-issue replies. These are PLAIN responses, not errors — which is
	// why the error branch is classified first (see ClassifyCheckpointSaveResponse).
	// Matched exactly rather than on a bare "in progress": that broader form also
	// appears in the raced-shutdown error below, which means the opposite.
	checkpointInProgressResp = "already in progress"
	checkpointCompleteResp   = "already complete"

	// A save that started and then failed for at least one namespace. The node still
	// parks, so checkpoint-status can be read for the per-namespace detail.
	checkpointFailedResp = "checkpoint-save failed"

	// Both commands answer with this when no namespace resolved a checkpoint path —
	// the feature is off, or every namespace is fully durable or opted out.
	checkpointNothingConfiguredResp = "no namespace is checkpointing"

	// A signal handler won the shutdown claim before checkpoint-save could, so an
	// ORDINARY shutdown is running and NO checkpoint will be taken. Its full text is
	// "checkpoint-save raced a shutdown already in progress - check checkpoint-status",
	// which contains "already in progress" — so it must be tested before any
	// in-progress match, or it reads as success and the caller polls a node that is
	// never going to produce a checkpoint.
	checkpointRacedShutdownResp = "raced a shutdown"
)

// Per-namespace checkpoint states reported by checkpoint-status.
const (
	CheckpointStateNone    = "none"
	CheckpointStateCopying = "copying"
	CheckpointStateDone    = "done"
	CheckpointStateFailed  = "failed"
)

// CheckpointSaveVerdict classifies a checkpoint-save response.
//
// The caller's real question is "poll, skip, or abort?" — Triggered and Accepted both
// mean poll, and differ only so a caller can avoid announcing a save it did not start.
//
// String-backed so it logs as itself and needs no String() method to keep in sync, and
// so the zero value is the empty string rather than a valid outcome — an unset verdict
// must not read as success.
type CheckpointSaveVerdict string

const (
	// CheckpointSaveTriggered — this call won the trigger; the save has started.
	CheckpointSaveTriggered CheckpointSaveVerdict = "triggered"

	// CheckpointSaveAccepted — a save already exists on this node: running, finished, or
	// FAILED. All three are one verdict because the caller does the same thing with them
	// — read checkpoint-status, which reports the outcome per namespace and is strictly
	// more informative than this reply. The server says as much, answering a failed save
	// with "see checkpoint-status".
	CheckpointSaveAccepted CheckpointSaveVerdict = "accepted"

	// CheckpointSaveNothingToDo — no namespace on this node is checkpointing at all.
	// Distinct from CheckpointSaveRejected because retrying cannot change it. Strictly a
	// fast path: checkpoint-status answers with the same error.
	CheckpointSaveNothingToDo CheckpointSaveVerdict = "nothing-to-do"

	// CheckpointSaveRejected — the command was refused and nothing started (the server is
	// still starting up, a parameter was malformed, or an ordinary shutdown won the race).
	//
	// This is the verdict that cannot be recovered from checkpoint-status. A refused save
	// leaves every namespace at state=none, which is exactly what an ACCEPTED save reads
	// as until the shutdown sequence reaches the copy — so a caller that polled instead
	// would wait out its whole window on every refusal.
	CheckpointSaveRejected CheckpointSaveVerdict = "rejected"
)

// CheckpointNamespaceStatus is one namespace's entry in a checkpoint-status response.
type CheckpointNamespaceStatus struct {
	// State is one of the CheckpointState* constants. It is reported verbatim, so a
	// state added by a newer server arrives unrecognised rather than being coerced —
	// callers should treat an unknown state as non-terminal.
	State string

	// FilesDone/FilesTotal track copy progress, or are -1 when the server did not
	// report a parseable counter. 0/0 is a legitimate value — a namespace with nothing
	// to copy, or one whose save has not begun — so it cannot double as "unknown".
	//
	// Once checkpoint-save has fired this is the only progress signal the node offers,
	// since the info gate refuses statistics.
	FilesDone  int
	FilesTotal int
}

// IsTerminal reports whether the save has finished for this namespace, successfully or not.
// An unrecognised state is treated as still running, which is self-correcting, since a non-terminal state is transient.
func (s CheckpointNamespaceStatus) IsTerminal() bool {
	return s.State == CheckpointStateDone || s.State == CheckpointStateFailed
}

// CheckpointSave sends checkpoint-save to one host and classifies the reply.
// The node leaves the cluster and parks when this succeeds, so send it only after all
// other info work for this node is complete. timeoutSec sets the park bound; pass 0 to
// leave the server on its default. The call is idempotent — re-issuing it against a
// parked node reports the current state rather than restarting the save.
func (asc *ASConn) CheckpointSave(
	policy *aero.ClientPolicy, timeoutSec int,
) (CheckpointSaveVerdict, error) {
	cmd := checkpointSaveCmd
	if timeoutSec > 0 {
		cmd = fmt.Sprintf("%s:timeout=%d", checkpointSaveCmd, timeoutSec)
	}

	res, err := asc.RunInfo(policy, cmd)
	if err != nil {
		return CheckpointSaveRejected, err
	}

	verdict := ClassifyCheckpointSaveResponse(res[cmd])
	asc.Log.V(1).Info("CheckpointSave", "verdict", verdict, "res", res[cmd])

	return verdict, nil
}

// CheckpointStatus reports per-namespace checkpoint progress for one host.
//
// Only namespaces the node is actually checkpointing appear. A namespace absent from a
// successful response is one this node is NOT checkpointing — see ParseCheckpointStatus.
// It returns an empty map, and no error, when no namespace on the node is checkpointing.
func (asc *ASConn) CheckpointStatus(
	policy *aero.ClientPolicy,
) (map[string]CheckpointNamespaceStatus, error) {
	res, err := asc.RunInfo(policy, checkpointStatusCmd)
	if err != nil {
		return nil, err
	}

	return ParseCheckpointStatus(res[checkpointStatusCmd])
}

// ClassifyCheckpointSaveResponse maps a checkpoint-save response to a verdict.
//
// An empty response classifies as rejected: it is no evidence that a save started, and
// polling on that assumption spins until the caller's window expires.
func ClassifyCheckpointSaveResponse(resp string) CheckpointSaveVerdict {
	// Errors are classified FIRST. The two idempotent "a save is already running" replies
	// are plain responses, so nothing is lost by checking errors first — and it is what
	// keeps the raced-shutdown error, whose text also contains "already in progress",
	// from being read as success.
	if info.IsInfoErrorResponse(resp) {
		switch {
		case strings.Contains(resp, checkpointRacedShutdownResp):
			// Ordered ahead of the in-progress case below: this text contains
			// "already in progress" but means no checkpoint will be taken at all.
			return CheckpointSaveRejected
		case strings.Contains(resp, checkpointFailedResp):
			// A save ran and failed for at least one namespace. Non-fatal: the node
			// still parks, so the caller polls and checkpoint-status names them.
			return CheckpointSaveAccepted
		case strings.Contains(resp, checkpointNothingConfiguredResp):
			return CheckpointSaveNothingToDo
		default:
			// Includes the server's info-gate refusal, should a future server ever
			// stop allowlisting checkpoint-save during the park. That refusal confirms
			// a save IS running, so it would belong above rather than here — but no
			// server that has this feature can produce it for checkpoint-save, and the
			// error text surfaces to the caller either way.
			return CheckpointSaveRejected
		}
	}

	if strings.Contains(resp, checkpointInProgressResp) ||
		strings.Contains(resp, checkpointCompleteResp) {
		return CheckpointSaveAccepted
	}

	// No evidence a save started; polling on it would spin until the caller gives up.
	if resp == "" {
		return CheckpointSaveRejected
	}

	return CheckpointSaveTriggered
}

// ParseCheckpointStatus parses a checkpoint-status payload:
//
//	ns1:state=done:files=42/42;ns2:state=copying:files=20/42
//
// A namespace missing from the returned map is one the node is NOT checkpointing.
// An empty map with a nil error means no namespace on this node is checkpointing.
// Malformed entries are skipped rather than failing the whole parse, so one bad entry
// cannot hide the rest; a payload that is a server rejection returns an error.
func ParseCheckpointStatus(resp string) (map[string]CheckpointNamespaceStatus, error) {
	statuses := make(map[string]CheckpointNamespaceStatus)

	if strings.Contains(resp, checkpointNothingConfiguredResp) {
		return statuses, nil
	}

	if info.IsInfoErrorResponse(resp) {
		return nil, fmt.Errorf("checkpoint-status rejected: %s", resp)
	}

	// An empty payload is not "no namespace is checkpointing" — the server answers that
	// with the error handled above. Empty means the reply was lost or truncated, so it
	// must not be reported as an authoritative empty result: callers read an empty map
	// as "this node checkpoints nothing", which is a very different conclusion.
	if strings.TrimSpace(resp) == "" {
		return nil, fmt.Errorf("empty checkpoint-status response")
	}

	for _, entry := range strings.Split(resp, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		fields := strings.Split(entry, ":")
		if len(fields) < 2 || fields[0] == "" {
			continue
		}

		status := CheckpointNamespaceStatus{FilesDone: -1, FilesTotal: -1}

		for _, field := range fields[1:] {
			key, value, found := strings.Cut(field, "=")
			if !found {
				continue
			}

			switch key {
			case "state":
				status.State = value
			case "files":
				done, total, ok := strings.Cut(value, "/")
				if !ok {
					continue
				}

				status.FilesDone = atoiOrNegative(done)
				status.FilesTotal = atoiOrNegative(total)
			}
		}

		if status.State == "" {
			continue // not a status entry
		}

		statuses[fields[0]] = status
	}

	return statuses, nil
}

// atoiOrNegative returns -1 rather than an error for an unparseable counter: progress
// counters are advisory, and a malformed one must not discard the state alongside it.
func atoiOrNegative(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return -1
	}

	return n
}
