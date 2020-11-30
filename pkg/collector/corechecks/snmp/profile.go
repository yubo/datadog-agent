package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type profileDefinitionMap map[string]profileDefinition

type profileDefinition struct {
	Metrics    []metricsConfig   `yaml:"metrics"`
	MetricTags []metricTagConfig `yaml:"metric_tags"`
	Extends    []string          `yaml:"extends"`
}

func loadProfiles(pConfig profilesConfig) (profileDefinitionMap, error) {
	// TODO: Profiles
	//   - Load default profiles
	//   - Load config profiles
	profiles := make(map[string]profileDefinition)

	for name, profile := range pConfig {
		definitionFile := profile.DefinitionFile

		profileDefinition, err := readProfileDefinition(definitionFile)
		if err != nil {
			return nil, err
		}

		err = recursivelyExpandBaseProfiles(profileDefinition)
		if err != nil {
			return nil, err
		}

		profiles[name] = *profileDefinition
	}
	return profiles, nil
}

func readProfileDefinition(definitionFile string) (*profileDefinition, error) {
	filePath := resolveProfileDefinitionPath(definitionFile)
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	profileDefinition := &profileDefinition{}
	err = yaml.Unmarshal(buf, profileDefinition)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filePath, err)
	}
	return profileDefinition, nil
}

func resolveProfileDefinitionPath(definitionFile string) string {
	// TODO: Support profiles locations
	// See https://github.com/DataDog/integrations-core/blob/d7f4d42a4721aea683056901cf2053395ff48173/snmp/datadog_checks/snmp/utils.py#L64-L73
	confdPath := config.Datadog.GetString("confd_path")
	profilesPath := confdPath + "/snmp.d/profiles/"
	return profilesPath + definitionFile
}

func recursivelyExpandBaseProfiles(definition *profileDefinition) error {
	for _, basePath := range definition.Extends {
		baseDefinition, err := readProfileDefinition(basePath)
		if err != nil {
			return err
		}

		definition.Metrics = append(definition.Metrics, baseDefinition.Metrics...)
		definition.MetricTags = append(definition.MetricTags, baseDefinition.MetricTags...)
	}
	return nil
}
