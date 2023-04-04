package asconfig

import (
	"bufio"
	"io"

	"github.com/go-logr/logr"
)

// AsConfig is wrapper over Conf
type AsConfig struct {
	log      logr.Logger
	baseConf *Conf
	version  string
}

func New(log logr.Logger, version string, bconf *Conf) *AsConfig {
	return &AsConfig{
		version:  version,
		baseConf: bconf,
		log:      log,
	}
}

// ValidationErr represents version validation error
type ValidationErr struct {
	Value       interface{}
	ErrType     string
	Context     string
	Description string
	Field       string
}

// NewMapAsConfig creates AsConfig from map
func NewMapAsConfig(
	log logr.Logger, version string, configMap map[string]interface{},
) (*AsConfig, error) {
	baseConf := newMap(log, configMap)

	return &AsConfig{
		log:      log,
		baseConf: &baseConf,
		version:  version,
	}, nil
}

// newMap converts passed in map[string]interface{} into Conf
func newMap(log logr.Logger, configMap map[string]interface{}) Conf {
	return flattenConf(log, toConf(log, configMap), sep)
}

// IsValid checks validity of config
func (cfg *AsConfig) IsValid(log logr.Logger, version string) (
	bool, []*ValidationErr, error,
) {
	return confIsValid(log, cfg.baseConf, version)
}

// ToConfFile returns DotConf
func (cfg *AsConfig) ToConfFile() DotConf {
	conf := cfg.baseConf
	return confToDotConf(cfg.log, conf)
}

// ToMap returns a pointer to the expanded map form of AsConfig
func (cfg *AsConfig) ToMap() *Conf {
	res := Conf(expandConf(cfg.log, cfg.baseConf, sep))
	return &res
}

// FromConfFile unmarshales the aerospike config text in "in" into a new *Asconfig
func FromConfFile(log logr.Logger, version string, in io.Reader) (*AsConfig, error) {
	scanner := bufio.NewScanner(in)

	configMap, err := process(log, scanner, Conf{})
	if err != nil {
		return nil, err
	}

	return NewMapAsConfig(log, version, configMap)
}

// IsSupportedVersion returns true if version supported else false
func IsSupportedVersion(ver string) (bool, error) {
	return isSupportedVersion(ver)
}

// BaseVersion returns base-version for ver
func BaseVersion(ver string) (string, error) {
	return baseVersion(ver)
}
