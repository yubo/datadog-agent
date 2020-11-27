package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type profileDefinitionMap map[string]profileDefinition

type profileDefinition struct {
	Metrics []metricsConfig `yaml:"metrics"`
}

func loadProfiles(pConfig profilesConfig) (profileDefinitionMap, error) {
	// TODO: Profiles
	//   - Load default profiles
	//   - Load config profiles

	confdPath := config.Datadog.GetString("confd_path")
	profiles := make(map[string]profileDefinition)

	for name, profile := range pConfig {
		// TODO: Support profiles locations
		// See https://github.com/DataDog/integrations-core/blob/d7f4d42a4721aea683056901cf2053395ff48173/snmp/datadog_checks/snmp/utils.py#L64-L73
		filePath := confdPath + "/snmp.d/profiles/" + profile.DefinitionFile

		buf, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		profileDef := &profileDefinition{}
		err = yaml.Unmarshal(buf, profileDef)
		if err != nil {
			return nil, fmt.Errorf("in file %q: %v", filePath, err)
		}
		profiles[name] = *profileDef
	}
	return profiles, nil
}
