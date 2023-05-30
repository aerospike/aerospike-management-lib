// Copyright (C) 2018 Aerospike, Inc.
//
// All rights reserved.
//
// THIS IS UNPUBLISHED PROPRIETARY SOURCE CODE. THE COPYRIGHT NOTICE ABOVE DOES
// NOT EVIDENCE ANY ACTUAL OR INTENDED PUBLICATION.

package asconfig

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

var leadcloseWhtspRegex = regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
var insideWhtspRegex = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

func parseLine(line string) string {
	input := strings.Split(line, "#")[0]
	final := leadcloseWhtspRegex.ReplaceAllString(input, "")

	final = insideWhtspRegex.ReplaceAllString(final, " ")
	if final == "" || final == " " {
		return ""
	}

	return final
}

func toList(conf Conf) []Conf {
	if len(conf) == 0 {
		return nil
	}

	confList := make([]Conf, 0)

	for k := range conf {
		v := conf[k]
		switch v := v.(type) {
		case Conf:
			v[keyName] = k
			confList = append(confList, v)

		case []Conf:
			confList = append(confList, v...)

		default:
			continue
		}
	}

	return confList
}

func processSection(
	log logr.Logger, tok []string, scanner *bufio.Scanner, conf Conf,
) error {
	cfgName := tok[0]

	// Unnamed Sections are simply processed as Map except special sections like logging
	if len(tok) == 2 {
		if _, ok := conf[cfgName]; !ok {
			conf[cfgName] = make(Conf)
		}

		sec, err := process(log, scanner, conf[cfgName].(Conf))
		if err != nil {
			return err
		}

		if isSpecialListSection(cfgName) {
			conf[cfgName] = toList(sec)
			// } else if cfgName == "tls-authentication-client" {
			// 	if
		} else {
			conf[cfgName] = sec
		}

		return nil
	}

	// All section starts with > 2 token are named
	// section with possible multiple entries.
	// NOTE this means tls-authenticate-client will always be a slice.
	// The schema expects tls-authenticate-client to not have "false" or "any"
	// entries when it is an array so converting this back into yaml causes problems
	if _, ok := conf[cfgName]; !ok {
		conf[cfgName] = make([]Conf, 0)
	}

	tempConf := make(Conf)
	if err := processSection(log, tok[1:], scanner, tempConf); err != nil {
		return err
	}

	if isListSection(cfgName) {
		conf[cfgName] = append(conf[cfgName].([]Conf), toList(tempConf)...)
	} else {
		// process a non list named section (typed section)
		seList := toList(tempConf)

		if len(seList) < 1 {
			return nil
		}

		// storage engine device or index-type flash
		if isTypedSection(cfgName) {
			// storage engine is a named section, but it is not list so use first entry.
			// the schema files expect index-type and storage-engine to have a type field, not name, so replace it
			seList[0][keyType] = seList[0][keyName]
			delete(seList[0], keyName)
			conf[cfgName] = seList[0]
		} else { // TODO maybe error out in this else instead
			delete(seList[0], keyName)
		}
	}

	return nil
}

func addToStrList(conf Conf, cfgName, val string) {
	if _, ok := conf[cfgName]; !ok {
		conf[cfgName] = make([]string, 0)
	}

	conf[cfgName] = append(conf[cfgName].([]string), val)
}

func writeConf(log logr.Logger, tok []string, conf Conf) {
	cfgName := tok[0]

	// Handle special case for tls-authentication-client which can be a list
	// or a string depending on its value
	if cfgName == keyTLSAuthenticateClient {
		if len(tok) < 2 {
			log.V(1).Info("tls-authenticate-client requires a value")
			return
		}

		v := strings.ToLower(tok[1])
		if v == "false" || v == "any" {
			if _, ok := conf[cfgName]; ok {
				log.V(1).Info("tls-authenticate-client must only use 'any', 'false', or one or more subject names")
				return
			}

			conf[cfgName] = tok[1]
			return
		}
	}

	// // Handle single line list field
	// // Ex: file <path1> <path2> ...
	// if ok, _ := isSingleLineListField(cfgName); ok {
	// 	if _, ok := conf[cfgName]; !ok {
	// 		tmp := Conf{}
	// 		for i, item := range tok[1:] {
	// 			key := fmt.Sprintf("placeholder%d", i)
	// 			tmp[key] = item
	// 		}
	// 		tmp[keyName] = cfgName
	// 		conf[cfgName] = []Conf{tmp}
	// 	}

	// 	return
	// }

	// Handle List Field that gets concatenated
	// Ex: node-address-port 10.20.10 tlsname 3000
	if ok, sep := isListField(cfgName); ok {
		// we never want to concat list entries without a separator while parsing asconfig
		// because we loose the individual entries if we do
		if sep == "" {
			sep = " "
		}
		addToStrList(conf, cfgName, strings.Join(tok[1:], sep))
		return
	}

	// Handle human readable content
	if ok, humanizeFn := isSizeOrTime(cfgName); ok {
		conf[cfgName], _ = humanizeFn(tok[1])
		return
	}
	// More special Case handling
	switch cfgName {
	case "context":
		conf[tok[1]] = tok[2]

	case "xdr-digestlog-path":
		size, err := deHumanizeSize(tok[2])
		if err != nil {
			log.V(1).Info("Found invalid xdr-digestlog-size value, while creating acc config struct",
				"err", err)
			break
		}

		conf[cfgName] = fmt.Sprintf("%s %d", tok[1], size)

	default:
		if len(tok) > 2 {
			log.V(1).Info(
				"Found > 2 tokens: Unknown format for config, "+
					"while creating acc config struct",
				"config", cfgName, "token", tok,
			)

			break
		}

		if isStringField(cfgName) {
			conf[cfgName] = tok[1]
			break
		}

		// Convert string into Uint if possible
		n, err := strconv.ParseUint(tok[1], 10, 64)
		if err != nil {
			conf[cfgName] = tok[1]
		} else {
			conf[cfgName] = n
		}
	}
}

func process(log logr.Logger, scanner *bufio.Scanner, conf Conf) (Conf, error) {
	for scanner.Scan() {
		line := parseLine(scanner.Text())
		if line == "" {
			continue
		}

		tok := strings.Split(line, " ")

		// Zero tokens
		if len(tok) == 0 {
			log.V(1).Info("Config file line has 0 tokens")
			return nil, ErrConfigParse
		}
		// End of Section
		if tok[0] == "}" {
			return conf.ToParsedValues(), nil
		}

		// Except end of section there should
		// be atleast 2 tokens
		if len(tok) < 2 {
			// if enable benchmark presence is
			// enable
			if isSpecialBoolField(tok[0]) {
				conf[tok[0]] = true
				continue
			}

			log.V(1).Info("Config file line has  < 2 tokens:", "token", tok)

			return nil, ErrConfigParse
		}

		// Start section
		if tok[len(tok)-1] == "{" {
			if err := processSection(log, tok, scanner, conf); err != nil {
				return nil, err
			}
		} else {
			writeConf(log, tok, conf)
		}
	}

	return conf, nil
}

// // isSingleLineListField identifies aerospike config list fields that can contain
// // multiple elements on the same line without a repeated key. Ex: file <path1> <path2> ...
// // it returns true if the key is a single line list field, and the separator used between
// // its elements
// func isSingleLineListField(key string) (exists bool, separator string) {
// 	switch key {
// 	case keyFile: // TODO identify other single line list fields
// 		exists = true
// 		separator = " "
// 	default:
// 		exists = false
// 	}

// 	return
// }
