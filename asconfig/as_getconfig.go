package asconfig

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v8"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/deployment"
	"github.com/aerospike/aerospike-management-lib/info"
)

// GetASConfig returns the value of the given path from the aerospike config from given host.
func GetASConfig(path *string, conn *deployment.ASConn, aerospikePolicy *aero.ClientPolicy) (
	confToReturn interface{}, err error) {
	h := aero.Host{
		Name:    conn.AerospikeHostName,
		Port:    conn.AerospikePort,
		TLSName: conn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(conn.Log, &h, aerospikePolicy)

	var ctxs []string
	if path != nil {
		// Get the corresponding sets info also if context is namespace.
		ctxs = []string{ContextKey(*path)}
		if ctxs[0] == info.ConfigNamespaceContext {
			ctxs = append(ctxs, info.ConfigSetContext)
		}
	}

	conf, err := asinfo.GetAsConfig(ctxs...)
	if err != nil {
		conn.Log.Error(err, "failed to get asconfig")
		return nil, err
	}

	if path == nil {
		return conf, nil
	}

	confToReturn = conf[ctxs[0]]
	if confToReturn == nil {
		conn.Log.Info("Config is nil", "context", ctxs[0])
		return nil, nil
	}

	confToReturn, err = traverseConfig(conn.Log, confToReturn, *path, ctxs[0])
	if err != nil {
		conn.Log.Error(err, "failed to traverse config")
		return nil, err
	}

	return confToReturn, nil
}

// traverseConfig recursively traverses the configuration based on the given path.
func traverseConfig(logger logr.Logger, conf interface{}, path, context string) (interface{}, error) {
	tokens := strings.Split(path, sep)
	for idx, token := range tokens[1:] {
		if ReCurlyBraces.MatchString(token) {
			name := strings.Trim(token, "{}")
			if context == info.ConfigLoggingContext && name == constLoggingConsole {
				name = constLoggingStderr
			}

			stats, ok := conf.(lib.Stats)
			if !ok {
				return nil, fmt.Errorf("invalid configuration type")
			}

			conf = stats[name]
		} else {
			stats, ok := conf.(lib.Stats)
			if !ok {
				return nil, fmt.Errorf("invalid configuration type")
			}

			conf = stats[token]
		}

		if conf == nil {
			logger.Info("Config is nil", "path", strings.Join(tokens[:idx+2], sep))
			return nil, nil
		}
	}

	return conf, nil
}
