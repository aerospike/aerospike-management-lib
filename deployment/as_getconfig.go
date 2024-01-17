package deployment

import (
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v6"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/asconfig"
	"github.com/aerospike/aerospike-management-lib/commons"
	"github.com/aerospike/aerospike-management-lib/info"
)

func GetASConfig(path string, conn *ASConn, aerospikePolicy *aero.ClientPolicy) (confToReturn interface{}, err error) {
	h := aero.Host{
		Name:    conn.AerospikeHostName,
		Port:    conn.AerospikePort,
		TLSName: conn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(conn.Log, &h, aerospikePolicy)
	ctx := asconfig.ContextKey(path)

	conf, err := asinfo.GetAsConfig(ctx)
	if err != nil {
		conn.Log.Error(err, "failed to get asconfig")
		return nil, err
	}

	namesInCurlyBraces := make([]string, 0)

	tokens := strings.Split(path, ".")
	for _, token := range tokens[1:] {
		if commons.ReCurlyBraces.MatchString(token) {
			namesInCurlyBraces = append(namesInCurlyBraces, strings.Trim(token, "{}"))
		}
	}

	switch ctx {
	case info.ConfigNamespaceContext:
		namespaces := conf[ctx]
		if namespaces != nil {
			if len(namesInCurlyBraces) > 0 {
				confToReturn = namespaces.(lib.Stats)[namesInCurlyBraces[0]]
			} else {
				confToReturn = namespaces
			}
		} else {
			conn.Log.Info("Namespaces are nil")
		}

	case info.ConfigXDRContext:
		xdr := conf[ctx]
		if xdr != nil {
			var dcs interface{}

			if len(namesInCurlyBraces) == 0 {
				confToReturn = xdr
			} else {
				dcs = xdr.(lib.Stats)[info.ConfigDCContext]
				if dcs != nil {
					confToReturn = dcs.(lib.Stats)[namesInCurlyBraces[0]]
				} else {
					conn.Log.Info("DCS is nil")
				}

				if len(namesInCurlyBraces) > 1 {
					if confToReturn != nil {
						namespaces := confToReturn.(lib.Stats)[info.ConfigNamespaceContext]
						if namespaces != nil {
							confToReturn = namespaces.(lib.Stats)[namesInCurlyBraces[1]]
						} else {
							conn.Log.Info("Namespaces are nil")
							confToReturn = nil
						}
					} else {
						conn.Log.Info("DC is nil", "DCName", namesInCurlyBraces[0])
					}
				}
			}
		} else {
			conn.Log.Info("XDR is nil")
		}

	case info.ConfigLoggingContext:
		logs := conf[ctx]
		if logs != nil {
			if len(namesInCurlyBraces) > 0 {
				confToReturn = logs.(lib.Stats)[namesInCurlyBraces[0]]
			} else {
				confToReturn = logs
			}
		} else {
			conn.Log.Info("Logs are nil")
		}
	case info.ConfigSetContext:
		namespaces := conf[info.ConfigNamespaceContext]
		if namespaces != nil {
			if len(namesInCurlyBraces) > 0 {
				confToReturn = namespaces.(lib.Stats)[namesInCurlyBraces[0]]
			} else {
				confToReturn = namespaces
			}
		} else {
			conn.Log.Info("Sets are nil")
		}
	default:
		confToReturn = conf
	}

	return confToReturn, nil
}
