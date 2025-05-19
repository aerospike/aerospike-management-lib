package asconfig

// This is a placeholder package to satisfy imports from asconfig CLI.

import (
	"bufio"
	"io"

	"github.com/go-logr/logr"
)

// AsConfig is wrapper over Conf
type AsConfig struct {
	baseConf *Conf
	log      logr.Logger
}

func New(log logr.Logger, bconf *Conf) *AsConfig {
	return &AsConfig{
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

// NewMapAsConfig creates AsConfig. Typically, an unmarshalled yaml file is passed in
func NewMapAsConfig(
	log logr.Logger, configMap map[string]interface{},
) (*AsConfig, error) {
	baseConf := newMap(log, configMap)

	return &AsConfig{
		log:      log,
		baseConf: &baseConf,
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

// ToMap returns a pointer to the
// expanded map form of AsConfig
func (cfg *AsConfig) ToMap() *Conf {
	cpy := cfg.baseConf.DeepClone()
	res := expandConf(cfg.log, &cpy, sep)

	return &res
}

// GetFlatMap returns a pointer to the copy
// of the flattened config stored in cfg
func (cfg *AsConfig) GetFlatMap() *Conf {
	res := cfg.baseConf.DeepClone()
	return &res
}

// FromConfFile unmarshales the aerospike config text in "in" into a new *AsConfig
func FromConfFile(log logr.Logger, in io.Reader) (*AsConfig, error) {
	scanner := bufio.NewScanner(in)

	configMap, err := process(log, scanner, Conf{})
	if err != nil {
		return nil, err
	}

	return NewMapAsConfig(log, configMap)
}

// IsSupportedVersion returns true if version supported else false
func IsSupportedVersion(ver string) (bool, error) {
	return isSupportedVersion(ver)
}

// BaseVersion returns base-version for ver
func BaseVersion(ver string) (string, error) {
	return baseVersion(ver)
}
