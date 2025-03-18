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

	"github.com/aerospike/aerospike-management-lib/utils"
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
			v[KeyName] = k
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
		} else {
			conf[cfgName] = sec
		}

		return nil
	}

	// All section starts with > 2 token are named
	// section with possible multiple entries.
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

		if len(seList) == 0 {
			return nil
		}

		// storage engine device or index-type flash
		if isTypedSection(cfgName) {
			// storage engine is a named section, but it is not list so use first entry.
			// the schema files expect index-type and storage-engine to have a type field, not name, so replace it
			seList[0][keyType] = seList[0][KeyName]
		}

		delete(seList[0], KeyName)
		conf[cfgName] = seList[0]
	}

	return nil
}

func addToStrList(conf Conf, cfgName, val string) {
	if _, ok := conf[cfgName]; !ok {
		conf[cfgName] = make([]string, 0)
	}

	if l, ok := conf[cfgName].([]string); ok {
		conf[cfgName] = append(l, val)
	}
}

func writeConf(log logr.Logger, tok []string, conf Conf) error {
	cfgName := tok[0]

	// Handle special case for tls-authentication-client which can be a list
	// or a string depending on its value
	if cfgName == keyTLSAuthenticateClient {
		if len(tok) < 2 {
			log.Error(ErrConfigParse, "tls-authenticate-client requires a value")
			return ErrConfigParse
		}

		v := strings.ToLower(tok[1])
		if v == keyFalse || v == keyAny {
			if _, ok := conf[cfgName]; ok {
				log.Error(ErrConfigParse, "tls-authenticate-client must only use 'any', 'false', or one or more subject names")
				return ErrConfigParse
			}

			conf[cfgName] = tok[1]

			return nil
		}
	}

	// Handle List Field that gets concatenated
	// Ex: node-address-port 10.20.10 tlsname 3000
	if ok, listSep := isListField(cfgName); ok {
		// we never want to concat list entries without a separator while parsing asconfig
		// because we loose the individual entries if we do
		if listSep == "" {
			listSep = " "
		}

		addToStrList(conf, cfgName, strings.Join(tok[1:], listSep))

		return nil
	}

	// Handle delimiter separated strings
	// Ex: node-address-port 10.20.10 tlsname 3000
	if ok, delim := isDelimitedStringField(cfgName); ok {
		// we never want to concat separated string entries without a separator while parsing asconfig
		// because we loose the individual entries if we do
		if delim == "" {
			delim = " "
		}

		conf[cfgName] = strings.Join(tok[1:], delim)

		return nil
	}

	// Handle human readable content
	if ok, humanizeFn := isSizeOrTime(cfgName); ok {
		conf[cfgName], _ = humanizeFn(tok[1])
		return nil
	}
	// More special Case handling
	switch cfgName {
	case "context":
		conf[tok[1]] = tok[2]

	case "xdr-digestlog-path":
		size, err := deHumanizeSize(tok[2])
		if err != nil {
			log.Error(err, "Found invalid xdr-digestlog-size value, while creating acc config struct")
			break
		}

		conf[cfgName] = fmt.Sprintf("%s %d", tok[1], size)

	default:
		if len(tok) > 2 {
			log.Error(ErrConfigParse,
				"Found > 2 tokens: Unknown format for config, "+
					"while creating acc config struct",
				"config", cfgName, "token", tok,
			)

			break
		}

		conf[cfgName] = parseValue(cfgName, tok[1])
	}

	return nil
}

func parseValue(k string, val interface{}) interface{} {
	valStr, ok := val.(string)
	if !ok {
		return val
	}

	if utils.IsStringField(k) {
		return val
	}

	if value, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return value
	} else if value, err := strconv.ParseUint(valStr, 10, 64); err == nil {
		return value
	} else if value, err := strconv.ParseFloat(valStr, 64); err == nil {
		return value
	} else if value, err := strconv.ParseBool(valStr); err == nil {
		return value
	}

	return valStr
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
			log.Error(ErrConfigParse, "Config file line has 0 tokens")
			return nil, ErrConfigParse
		}

		lastToken := tok[len(tok)-1]
		if lastToken != "{" && strings.HasSuffix(lastToken, "{") {
			log.Error(ErrConfigParse, "Config file items must have a space between them and '{' ", "token", lastToken)
			return nil, ErrConfigParse
		}

		// End of Section
		if tok[0] == "}" {
			return conf, nil
		}

		// Except end of section there should
		// be atleast 2 tokens
		if len(tok) < 2 {
			// if enable benchmark presence is
			// enable
			if isSpecialBoolField(tok[0]) || isSpecialOrNormalBoolField(tok[0]) {
				conf[tok[0]] = true
				continue
			}

			log.Error(ErrConfigParse, "Config file line has  < 2 tokens:", "token", tok)

			return nil, ErrConfigParse
		}

		// Start section
		if tok[len(tok)-1] == "{" {
			if err := processSection(log, tok, scanner, conf); err != nil {
				return nil, err
			}
		} else {
			if err := writeConf(log, tok, conf); err != nil {
				return nil, err
			}
		}
	}

	return conf, nil
}
