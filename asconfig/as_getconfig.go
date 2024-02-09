package asconfig

import (
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v6"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/commons"
	"github.com/aerospike/aerospike-management-lib/deployment"
	"github.com/aerospike/aerospike-management-lib/info"
)

func GetASConfig(path string, conn *deployment.ASConn, aerospikePolicy *aero.ClientPolicy) (
	confToReturn interface{}, err error) {
	h := aero.Host{
		Name:    conn.AerospikeHostName,
		Port:    conn.AerospikePort,
		TLSName: conn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(conn.Log, &h, aerospikePolicy)

	ctxs := []string{ContextKey(path)}
	if ctxs[0] == info.ConfigNamespaceContext {
		ctxs = append(ctxs, info.ConfigSetContext)
	}

	conf, err := asinfo.GetAsConfig(ctxs...)
	if err != nil {
		conn.Log.Error(err, "failed to get asconfig")
		return nil, err
	}

	confToReturn = conf[ctxs[0]]
	if confToReturn == nil {
		conn.Log.Info("Config is nil", "context", ctxs[0])
		return nil, nil
	}

	tokens := strings.Split(path, ".")
	for idx, token := range tokens[1:] {
		if commons.ReCurlyBraces.MatchString(token) {
			name := strings.Trim(token, "{}")

			if ctxs[0] == info.ConfigLoggingContext && name == constLoggingConsole {
				name = constLoggingStderr
			}

			confToReturn = confToReturn.(lib.Stats)[name]
		} else {
			confToReturn = confToReturn.(lib.Stats)[token]
		}

		if confToReturn == nil {
			conn.Log.Info("Config is nil", strings.Join(tokens[:idx+2], sep))
			break
		}
	}

	return confToReturn, nil
}
