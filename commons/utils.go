package commons

import (
	"regexp"
	"strings"

	"github.com/go-logr/logr"
)

var ReCurlyBraces = regexp.MustCompile(`^\{.*\}$`)

// SplitKey splits key by using sep
// it ignores sep inside sectionNameStartChar and sectionNameEndChar
func SplitKey(log logr.Logger, key, sep string) []string {
	sepRunes := []rune(sep)
	if len(sepRunes) > 1 {
		log.Info("Split expects single char as separator")
		return nil
	}

	openBracket := 0
	f := func(c rune) bool {
		if c == sepRunes[0] && openBracket == 0 {
			return true
		}

		if c == SectionNameStartChar {
			openBracket++
		} else if c == SectionNameEndChar {
			openBracket--
		}

		return false
	}

	return strings.FieldsFunc(key, f)
}
