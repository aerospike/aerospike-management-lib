package asconfig

import (
	log "github.com/inconshreveable/log15"
)

// AsConfig is wrapper over Conf
type AsConfig struct {
	version  string
	baseConf Conf
}

// ValidationErr represents version validation error
type ValidationErr struct {
	ErrType     string
	Context     string
	Description string
	Field       string
	Value       interface{}
}

var pkglog = log.New(log.Ctx{"module": "lib.asconfig"})

// NewMapAsConfig creates AsConfig from map
func NewMapAsConfig(version string, configMap map[string]interface{}) (*AsConfig, error) {
	baseConf := newMap(configMap)

	return &AsConfig{
		baseConf: baseConf,
		version:  version}, nil
}

// newMap converts passed in map[string]interface{} into Conf
func newMap(configMap map[string]interface{}) Conf {
	return flattenConf(toConf(configMap), sep)
}

// IsValid checks validity of config
func (cfg *AsConfig) IsValid(version string) (bool, []*ValidationErr, error) {
	return confIsValid(cfg.baseConf, version)
}

// ToConfFile returns DotConf
func (cfg *AsConfig) ToConfFile() DotConf {
	conf := cfg.baseConf
	return confToDotConf(conf)
}

// IsSupportedVersion returns true if version supported else false
func IsSupportedVersion(ver string) (bool, error) {
	return isSupportedVersion(ver)
}

// BaseVersion returns baseversion for ver
func BaseVersion(ver string) (string, error) {
	return baseVersion(ver)
}
