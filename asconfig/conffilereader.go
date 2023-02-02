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
			v["name"] = k
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
		// storage engine device
		seList := toList(tempConf)
		if len(seList) > 0 {
			// storage engine is named section, but it is not list so use first entry
			delete(seList[0], "name")
			conf[cfgName] = seList[0]
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

	// Handle List Field
	if ok, sep := isListField(cfgName); ok {
		addToStrList(conf, cfgName, strings.Join(tok[1:], sep))
		return
	}

	// Handle human readable content
	if ok, humanizeFn := isSizeOrTime(cfgName); ok {
		conf[cfgName], _ = humanizeFn(tok[1])
		return
	}
	// Special Case handling
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
