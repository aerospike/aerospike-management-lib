package asconfig

import (
	"github.com/go-logr/logr"
)

// AsConfig is wrapper over Conf
type AsConfig struct {
	version  string
	baseConf *Conf
	log      logr.Logger
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
	ErrType     string
	Context     string
	Description string
	Field       string
	Value       interface{}
}

// NewMapAsConfig creates AsConfig from map
func NewMapAsConfig(log logr.Logger, version string, configMap map[string]interface{}) (*AsConfig, error) {
	baseConf := newMap(log, configMap)

	return &AsConfig{
		log:      log,
		baseConf: &baseConf,
		version:  version}, nil
}

// newMap converts passed in map[string]interface{} into Conf
func newMap(log logr.Logger, configMap map[string]interface{}) Conf {
	return flattenConf(log, toConf(log, configMap), sep)
}

// IsValid checks validity of config
func (cfg *AsConfig) IsValid(log logr.Logger, version string) (bool, []*ValidationErr, error) {
	return confIsValid(log, cfg.baseConf, version)
}

// ToConfFile returns DotConf
func (cfg *AsConfig) ToConfFile() DotConf {
	conf := cfg.baseConf
	return confToDotConf(cfg.log, conf)
}

// IsSupportedVersion returns true if version supported else false
func IsSupportedVersion(ver string) (bool, error) {
	return isSupportedVersion(ver)
}

// BaseVersion returns baseversion for ver
func BaseVersion(ver string) (string, error) {
	return baseVersion(ver)
}
