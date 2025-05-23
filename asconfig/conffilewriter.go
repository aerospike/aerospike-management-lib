// Copyright (C) 2018 Aerospike, Inc.
//
// All rights reserved.
//
// THIS IS UNPUBLISHED PROPRIETARY SOURCE CODE. THE COPYRIGHT NOTICE ABOVE DOES
// NOT EVIDENCE ANY ACTUAL OR INTENDED PUBLICATION.

package asconfig

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/info"
)

const (
	constLoggingConsole = "console"
	constLoggingStderr  = "stderr"
	constLoggingSyslog  = "syslog"
)

func indentString(indent int) string {
	return strings.Repeat(" ", indent*4)
}

func beginSection(
	_ logr.Logger, buf *bytes.Buffer, indent int, name ...string,
) {
	buf.WriteString(
		"\n" + indentString(indent) + strings.Join(
			name, " ",
		) + " {\n",
	)
}

func endSection(buf *bytes.Buffer, indent int) {
	buf.WriteString(strings.Repeat(" ", indent*4) + "}\n")
}

func writeSimpleSection(
	log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int,
) {
	beginSection(log, buf, indent, section)
	writeDotConf(log, buf, conf, indent+1, nil)
	endSection(buf, indent)
}

func writeLogContext(buf *bytes.Buffer, conf Conf, indent int) {
	keys := make([]string, 0, len(conf))

	for k := range conf {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	syslogParamsSets := sets.NewSet("facility", "path", "tag")

	for _, key := range keys {
		if key == KeyName {
			// ignore generated field
			continue
		}

		if syslogParamsSets.Contains(key) {
			// This is non-context syslog specific key
			writeField(buf, key, conf[key].(string), indent)
		} else {
			writeField(buf, "context "+key, conf[key].(string), indent)
		}
	}
}

func writeLogSection(
	log logr.Logger, buf *bytes.Buffer, section string, confs []Conf, indent int,
) {
	beginSection(log, buf, indent, section)

	for i := range confs {
		conf := confs[i]

		name, ok := conf[KeyName].(string)
		if !ok {
			continue
		}

		key := name
		if name != constLoggingConsole && name != constLoggingSyslog {
			key = "file " + name
		}

		beginSection(log, buf, indent+1, key)
		writeLogContext(buf, conf, indent+2)
		endSection(buf, indent+1)
	}

	endSection(buf, indent)
}

func writeTypedSection(
	log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int,
) {
	typeStr := conf[keyType].(string)
	delete(conf, keyType)

	if len(conf) > 0 {
		beginSection(log, buf, indent, fmt.Sprintf("%s %v", section, typeStr))
		writeDotConf(log, buf, conf, indent+1, nil)
		endSection(buf, indent)
	} else {
		// Section with just the type like storage-engine memory
		writeField(buf, section, typeStr, indent)
	}
}

func writeSpecialListSection(
	log logr.Logger, buf *bytes.Buffer, section string, confList []Conf,
	indent int,
) {
	section = SingularOf(section)
	if section == info.ConfigLoggingContext {
		writeLogSection(log, buf, section, confList, indent)
		return
	}
}

func writeListSection(
	log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int,
) {
	name, ok := conf[KeyName].(string)
	if !ok || name == "" {
		return
	}

	delete(conf, KeyName)

	section = SingularOf(section)

	beginSection(log, buf, indent, section+" "+name)
	writeDotConf(log, buf, conf, indent+1, nil)
	endSection(buf, indent)

	conf[KeyName] = name
}

func writeSection(
	log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int,
) {
	m, ok := conf[section].(Conf)
	if !ok {
		log.V(1).Info("Section is not a config", "section", section)
		return
	}

	// Skip if no entry for the list.
	if len(m) == 0 && !strings.EqualFold(section, "security") {
		return
	}

	switch {
	case strings.EqualFold(section, keyStorageEngine):
		writeTypedSection(log, buf, section, m, indent)
		return

	case strings.EqualFold(section, "index-type"):
		writeTypedSection(log, buf, section, m, indent)
		return

	case strings.EqualFold(section, "sindex-type"):
		writeTypedSection(log, buf, section, m, indent)
		return

	default:
		writeSimpleSection(log, buf, section, m, indent)
	}
}

func writeListField(
	buf *bytes.Buffer, key string, value string, indent int, sep string,
) {
	key = SingularOf(key)
	if sep != "" {
		buf.WriteString(
			indentString(indent) + key + "    " + strings.ReplaceAll(
				value, sep, " ",
			) + "\n",
		)
	} else {
		buf.WriteString(indentString(indent) + key + "    " + value + "\n")
	}
}

func writeField(buf *bytes.Buffer, key, value string, indent int) {
	switch {
	case isFormField(key):
		return
	case isEmptyField(key, value):
		return
	// Skipping the writing of benchmark configurations when their corresponding value is false.
	// In server versions without the fix for AER-6767 (https://aerospike.atlassian.net/browse/AER-6767),
	// the presence of these fields implied that the corresponding benchmark was enabled,
	// even if the value was explicitly set to false.
	// To prevent such configurations from being misinterpreted as enabled,
	// benchmark configurations with a value of false are now omitted entirely.
	case isSpecialBoolField(key):
		if strings.EqualFold(value, "false") {
			return
		}
	}

	buf.WriteString(indentString(indent) + key + "    " + value + "\n")
}

func writeKeys(
	log logr.Logger, buf *bytes.Buffer, keys *[]string, conf Conf,
	isSimple bool, indent int,
) {
	for _, k := range *keys {
		v := conf[k]
		if v == nil {
			continue
		}

		switch v := v.(type) {
		case string, bool, int, int64, uint64, float64:
			if isSimple {
				sv, _ := lib.ToString(v)

				if ok, sep := isDelimitedStringField(k); ok {
					writeListField(buf, k, sv, indent, sep)
					break
				}

				writeField(buf, k, sv, indent)
			}

		case []string:
			if isSimple {
				ok, sep := isListField(k)
				if !ok {
					log.V(1).Info("List found in non list field", "key", k)
					break
				}

				if len(v) == 0 {
					break
				}

				for _, str := range v {
					writeListField(buf, k, str, indent, sep)
				}
			}

		case []interface{}:
			if !isSimple {
				if !isListSection(k) && !isSpecialListSection(k) {
					log.V(1).Info("List found in non list section", "key", k)
					break
				}

				if len(v) == 0 {
					continue
				}

				if isSpecialListSection(k) {
					vList := make([]Conf, 0)

					for indx := range v {
						if vM, ok := v[indx].(Conf); ok {
							vList = append(vList, vM)
						}
					}

					writeSpecialListSection(log, buf, k, vList, indent)

					break
				}

				for _, confI := range v {
					if confM, ok := confI.(Conf); ok {
						writeListSection(log, buf, k, confM, indent)
					}
				}
			}

		case []Conf:
			if !isSimple {
				if !isListSection(k) && !isSpecialListSection(k) {
					log.V(1).Info("List found in non list section", "key", k)
					break
				}

				if len(v) == 0 {
					continue
				}

				if isSpecialListSection(k) {
					writeSpecialListSection(log, buf, k, v, indent)
					break
				}

				for _, confM := range v {
					writeListSection(log, buf, k, confM, indent)
				}
			}

		case Conf:
			if !isSimple {
				writeSection(log, buf, k, conf, indent)
			}

		default:
			log.V(1).Info(
				"Unknown config value type",
				"type", reflect.TypeOf(v), "key", k, "value", v,
			)
		}
	}
}

//nolint:unparam // kept "onlyKeys" param for future ref
func writeDotConf(log logr.Logger, buf *bytes.Buffer, conf Conf, indent int, onlyKeys *[]string) {
	var keys = onlyKeys

	// Aesthetics, print conf in sorted manner in config
	// file.
	if keys == nil {
		allKeys := make([]string, 0, len(conf))
		for k := range conf {
			allKeys = append(allKeys, k)
		}

		keys = &allKeys
	}

	sort.Strings(*keys)
	writeKeys(log, buf, keys, conf, true, indent)
	writeKeys(log, buf, keys, conf, false, indent)
}
