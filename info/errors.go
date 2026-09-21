package info

import "strings"

// infoErrPrefix is how the server prefixes a refused info command. The code is
// omitted when it is the generic "unknown" error, giving "ERROR::<message>".
const infoErrPrefix = "ERROR:"

// IsInfoErrorResponse reports whether an info response is a server-side rejection
// rather than a result.
//
// This is needed because a refused info command is NOT a Go error: the server answers
// with an ordinary "ERROR:<code>:<message>" payload, so RequestInfo and
// ASConn.RunInfo both return it with err == nil. Callers that only check err treat a
// rejection as success.
//
// The trailing colon is part of the match, and that matters: several info commands
// answer with a payload whose first field is a namespace name, and a namespace may
// legally be called "errors". Matching only the five characters "error" would read
// "errors:state=done:..." as a rejection.
func IsInfoErrorResponse(resp string) bool {
	return len(resp) >= len(infoErrPrefix) &&
		strings.EqualFold(resp[:len(infoErrPrefix)], infoErrPrefix)
}
