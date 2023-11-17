package asconfig

import (
	"fmt"

	"github.com/go-logr/logr"
)

type ConfGetter interface {
	AllConfigs() (Conf, error)
	GetAsInfo(cmdList ...string) (Conf, error)
}

func GenerateConf(log logr.Logger, confGetter ConfGetter) (Conf, error) {
	configs, err := confGetter.AllConfigs()
	if err != nil {
		return nil, err
	}

	metadata, err := confGetter.GetAsInfo("metadata")
	if err != nil {
		return nil, err
	}

	fmt.Printf("%v\n", configs)
	fmt.Printf("%v\n", metadata)

	return Conf{}, nil
}

type pipelineStep interface {
	Execute(log logr.Logger, conf Conf) error
}

type pipeline interface {
	pipelineStep
	Steps() []pipelineStep
}
