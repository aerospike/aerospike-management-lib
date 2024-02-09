package commons

import (
	"regexp"
	"strings"

	"github.com/go-logr/logr"
)

var ReCurlyBraces = regexp.MustCompile(`^\{.*\}$`)

// DynamicConfigMap is a map of config flatten keys and their operations and values
// for eg: "xdr.dcs.{DC3}.node-address-ports": {commons.Remove: []string{"1.1.2.1 3000"}}
type DynamicConfigMap map[string]map[Operation]interface{}

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
