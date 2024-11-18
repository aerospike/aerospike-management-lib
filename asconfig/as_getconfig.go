package asconfig

import (
	"fmt"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v7"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/deployment"
	"github.com/aerospike/aerospike-management-lib/info"
)

// GetASConfig returns the value of the given path from the aerospike config from given host.
func GetASConfig(paths []string, conn *deployment.ASConn, aerospikePolicy *aero.ClientPolicy) (
	confToReturn map[string]interface{}, err error) {
	h := aero.Host{
		Name:    conn.AerospikeHostName,
		Port:    conn.AerospikePort,
		TLSName: conn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(conn.Log, &h, aerospikePolicy)
	ctxs := sets.NewSet[string]()

	for _, path := range paths {
		ctx := ContextKey(path)
		ctxs.Add(ctx)

		// Get the corresponding sets info also if context is namespace.
		if ctx == info.ConfigNamespaceContext {
			ctxs.Add(info.ConfigSetContext)
		}
	}

	conf, err := asinfo.GetAsConfig(ctxs.ToSlice()...)
	if err != nil {
		conn.Log.Error(err, "failed to get asconfig")
		return nil, err
	}

	if len(paths) == 0 {
		return conf, nil
	}

	confToReturn = make(map[string]interface{})

	for _, path := range paths {
		ctx := ContextKey(path)

		ctxConf := conf[ctx]
		if ctxConf == nil {
			conn.Log.Info("Config is nil", "context", ctxConf)
			return nil, nil
		}

		ctxConf, err = traverseConfig(conn.Log, ctxConf, path, ctx)
		if err != nil {
			conn.Log.Error(err, "failed to traverse config")
			return nil, err
		}

		confToReturn[path] = ctxConf
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
