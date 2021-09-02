package asconfig

import (
	"sync"

	"github.com/go-logr/logr"
)

// AsConfig is wrapper over Conf
type AsConfig struct {
	version  string
	baseConf Conf
}

var (
	doOnce sync.Once
	pkglog = logr.Discard()
)

func SetManagementLibLogger(logger logr.Logger) {
	doOnce.Do(func() {
		pkglog = logger
	})
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
