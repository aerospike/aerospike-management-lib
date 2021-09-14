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

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/go-logr/logr"
)

func indentString(indent int) string {
	return strings.Repeat(" ", indent*4)
}

func beginSection(log logr.Logger, buf *bytes.Buffer, indent int, name ...string) {
	buf.WriteString("\n" + indentString(indent) + strings.Join(name[:], " ") + " {\n")
}

func endSection(buf *bytes.Buffer, indent int) {
	buf.WriteString(strings.Repeat(" ", indent*4) + "}\n")
}

func writeSimpleSection(log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int) {
	beginSection(log, buf, indent, section)
	writeDotConf(log, buf, conf, indent+1, nil)
	endSection(buf, indent)
}

func writeLogContext(buf *bytes.Buffer, conf Conf, indent int) {
	var keys []string
	for k := range conf {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, context := range keys {
		if context == "name" {
			// ignore generated field
			continue
		}
		writeField(buf, "context "+context, conf[context].(string), indent)
	}
}

func writeLogSection(log logr.Logger, buf *bytes.Buffer, section string, confs []Conf, indent int) {
	beginSection(log, buf, indent, section)
	for i := range confs {
		conf := confs[i]

		name, ok := conf["name"].(string)
		if !ok {
			continue
		}
		key := name
		if name != "console" {
			key = "file " + name
		}

		beginSection(log, buf, indent+1, key)
		writeLogContext(buf, conf, indent+2)
		endSection(buf, indent+1)
	}
	endSection(buf, indent)
}

func writeTypedSection(log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int) {
	typeStr := conf["type"].(string)
	delete(conf, "type")

	if len(conf) > 0 {
		beginSection(log, buf, indent, fmt.Sprintf("%s %v", section, typeStr))
		writeDotConf(log, buf, conf, indent+1, nil)
		endSection(buf, indent)
	} else {
		// Section with just the type like storage-engine memory
		writeField(buf, section, typeStr, indent)
	}
}

func writeSpecialListSection(log logr.Logger, buf *bytes.Buffer, section string, confList []Conf, indent int) {
	section = SingularOf(section)
	switch section {
	case "logging":
		writeLogSection(log, buf, section, confList, indent)
		return
	}
}

func writeListSection(log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int) {
	name, ok := conf["name"].(string)
	if !ok || len(name) == 0 {
		return
	}

	delete(conf, "name")
	section = SingularOf(section)
	beginSection(log, buf, indent, section+" "+name)
	writeDotConf(log, buf, conf, indent+1, nil)
	endSection(buf, indent)
	conf["name"] = name
}

func writeSection(log logr.Logger, buf *bytes.Buffer, section string, conf Conf, indent int) {

	m, ok := conf[section].(Conf)
	if !ok {
		log.V(1).Info("section is not a config", "section", section)
		return
	}

	// Skip if no entry for the list.
	if len(m) == 0 {
		return
	}

	switch {
	case strings.EqualFold(section, "storage-engine"):
		writeTypedSection(log, buf, section, m, indent)
		return

	case strings.EqualFold(section, "index-type"):
		writeTypedSection(log, buf, section, m, indent)
		return

	default:
		writeSimpleSection(log, buf, section, m, indent)
	}

}

func writeListField(buf *bytes.Buffer, key string, value string, indent int, sep string) {
	key = SingularOf(key)
	if sep != "" {
		buf.WriteString(indentString(indent) + string(key) + "    " + strings.Replace(value, sep, " ", -1) + "\n")
	} else {
		buf.WriteString(indentString(indent) + string(key) + "    " + value + "\n")
	}
}

func writeSpecialBoolField(buf *bytes.Buffer, key string, indent int) {
	buf.WriteString(indentString(indent) + string(key) + "\n")
}

func writeField(buf *bytes.Buffer, key string, value string, indent int) {

	switch {
	case isFormField(key):
		return

	case isEmptyField(key, value):
		return

	case isSpecialBoolField(key):
		if strings.EqualFold(value, "true") {
			writeSpecialBoolField(buf, key, indent)
		}
		return
	}

	buf.WriteString(indentString(indent) + string(key) + "    " + value + "\n")
}

func writeKeys(log logr.Logger, buf *bytes.Buffer, keys *[]string, conf Conf, isSimple bool, indent int) {

	for _, k := range *keys {

		v := conf[k]
		if v == nil {
			continue
		}

		switch v := v.(type) {
		case string, bool, int, int64, uint64, float64:
			if isSimple {
				sv, _ := lib.ToString(v)
				writeField(buf, k, sv, indent)
			}

		case []string:
			if isSimple {
				ok, sep := isListField(k)
				if !ok {
					log.V(1).Info("list found in non list field", "key", k)
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
					log.V(1).Info("list found in non list section", "key", k)
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
					log.V(1).Info("list found in non list section", "key", k)
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
				"unknown config value type",
				"type", reflect.TypeOf(v), "key", k, "value", v)

		}
	}
}

func writeDotConf(log logr.Logger, buf *bytes.Buffer, conf Conf, indent int, onlyKeys *[]string) {

	var keys = onlyKeys

	// Asthetics, print conf in sorted manner in config
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
