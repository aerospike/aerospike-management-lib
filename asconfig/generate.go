package asconfig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

type ConfGetter interface {
	AllConfigs() (Conf, error)
	GetAsInfo(cmdList ...string) (Conf, error)
}

func GenerateConf(log *logr.Logger, confGetter ConfGetter) (Conf, error) {
	log.V(1).Info("Generating config")
	validConfig := Conf{}

	p := newPipeline(log, []pipelineStep{
		newGetConfigStep(log, confGetter),
		newServerVersionCheckStep(log, IsSupportedVersion),
	})

	err := p.execute(validConfig)

	return validConfig, err
}

type pipelineStep interface {
	execute(conf Conf) error
}

type pipeline struct {
	log   *logr.Logger
	steps []pipelineStep
}

func newPipeline(log *logr.Logger, steps []pipelineStep) *pipeline {
	return &pipeline{
		log:   log,
		steps: steps,
	}
}

func (p *pipeline) execute(conf Conf) error {
	for _, step := range p.steps {
		err := step.execute(conf)
		if err != nil {
			return err
		}
	}

	return nil
}

type GetConfigStep struct {
	log        *logr.Logger
	confGetter ConfGetter
}

func newGetConfigStep(log *logr.Logger, confGetter ConfGetter) *GetConfigStep {
	return &GetConfigStep{
		log:        log,
		confGetter: confGetter,
	}
}

func (s *GetConfigStep) execute(conf Conf) error {
	s.log.V(1).Info("Getting configs and metadata")

	configs, err := s.confGetter.AllConfigs()
	if err != nil {
		return err
	}

	conf["configs"] = configs["configs"]

	metadata, err := s.confGetter.GetAsInfo("metadata")
	if err != nil {
		return err
	}

	conf["metadata"] = metadata["metadata"]
	return nil
}

type ServerVersionCheckStep struct {
	log       *logr.Logger
	checkFunc func(string) (bool, error)
}

func newServerVersionCheckStep(log *logr.Logger, checkFunc func(string) (bool, error)) *ServerVersionCheckStep {
	return &ServerVersionCheckStep{
		log:       log,
		checkFunc: checkFunc,
	}
}

func (s *ServerVersionCheckStep) execute(conf Conf) error {
	s.log.V(1).Info("Checking server version")
	build := conf["metadata"].(map[string]interface{})["build"].(string)
	is_supported, err := s.checkFunc(build)

	if err != nil {
		return fmt.Errorf("error checking for supported server version: %s", err)
	}

	if !is_supported {
		return fmt.Errorf("unsupported version: %s", build)
	}

	return nil
}

type copyEffectiveRackIDStep struct {
	log *logr.Logger
}

func newCopyEffectiveRackIDStep(log *logr.Logger) *copyEffectiveRackIDStep {
	return &copyEffectiveRackIDStep{
		log: log,
	}
}

var rackRegex = regexp.MustCompile(`rack_(\d+)`)

func (s *copyEffectiveRackIDStep) execute(conf Conf) error {
	s.log.V(1).Info("Copying effective rack-id to rack-id")

	nsConfig := conf["configs"].(map[string]interface{})["namespaces"].(map[string]interface{})
	effectiveRacks := conf["config"].(map[string]interface{})["racks"].([]map[string]interface{})
	nodeID := conf["metadata"].(map[string]interface{})["node_id"].(string)

	for _, rackInfo := range effectiveRacks {
		ns := rackInfo["ns"].(string)

		// For this ns find which rack this node belongs to
		for rack, nodesStr := range rackInfo {
			if strings.Contains(nodesStr.(string), nodeID) {
				rackIDStr := rackRegex.FindString(rack)
				if rackIDStr == "" {
					return fmt.Errorf("unable to find rack id for rack %s", rack)
				}

				rackID, err := strconv.Atoi(rackIDStr)

				if err != nil {
					return fmt.Errorf("unable to convert rack id %s to int", rackIDStr)
				}

				// Copy effective rack-id over the ns config
				nsConfig[ns].(map[string]interface{})["rack-id"] = rackID
				break
			}
		}
	}
	// TODO: Consider adding a step that deletes invalid contexts and parameters
	delete(conf["config"].(map[string]interface{}), "racks")

	return nil
}
