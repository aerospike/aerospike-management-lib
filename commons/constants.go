package commons

// Operation is a custom type representing different operations.
type Operation string

const (
	// start and end character for section names
	SectionNameStartChar = '{'
	SectionNameEndChar   = '}'
	// Enum values for Operation
	Add    Operation = "add"
	Remove Operation = "remove"
	Update Operation = "update"
)
