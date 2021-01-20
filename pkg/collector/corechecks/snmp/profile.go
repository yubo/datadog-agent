package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

type profileDefinitionMap map[string]profileDefinition

type deviceMeta struct {
	Vendor string `yaml:"vendor"`
}

type profileDefinition struct {
	Metrics      []metricsConfig   `yaml:"metrics"`
	MetricTags   []metricTagConfig `yaml:"metric_tags"`
	Extends      []string          `yaml:"extends"`
	Device       deviceMeta        `yaml:"device"`
	SysObjectIds StringArray       `yaml:"sysobjectid"`
}

func getDefaultProfilesDefinitionFiles() profileConfigMap {
	profilesRoot := getProfileConfdRoot()
	files, err := ioutil.ReadDir(profilesRoot)
	if err != nil {
		log.Fatal(err)
	}

	profiles := make(profileConfigMap)
	for _, f := range files {
		fName := f.Name()
		// Skip partial profiles
		if strings.HasPrefix(fName, "_") {
			continue
		}
		// Skip non yaml profiles
		if !strings.HasSuffix(fName, ".yaml") {
			continue
		}
		profiles[fName[:len(fName)-5]] = profileConfig{filepath.Join(profilesRoot, fName)}
	}
	return profiles
}

func loadProfiles(pConfig profileConfigMap) (profileDefinitionMap, error) {
	profiles := make(map[string]profileDefinition)

	for name, profile := range pConfig {
		definitionFile := profile.DefinitionFile

		profileDefinition, err := readProfileDefinition(definitionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read profile definition `%s`: %s", name, err)
		}

		err = recursivelyExpandBaseProfiles(profileDefinition, profileDefinition.Extends)
		if err != nil {
			return nil, fmt.Errorf("failed to expand profile `%s`: %s", name, err)
		}

		profiles[name] = *profileDefinition
	}
	return profiles, nil
}

func readProfileDefinition(definitionFile string) (*profileDefinition, error) {
	filePath := resolveProfileDefinitionPath(definitionFile)
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file `%s`: %s", filePath, err)
	}

	profileDefinition := &profileDefinition{}
	err = yaml.Unmarshal(buf, profileDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall `%q`: %v", filePath, err)
	}
	return profileDefinition, nil
}

func resolveProfileDefinitionPath(definitionFile string) string {
	if filepath.IsAbs(definitionFile) {
		return definitionFile
	}
	return filepath.Join(getProfileConfdRoot(), definitionFile)
}

func getProfileConfdRoot() string {
	confdPath := config.Datadog.GetString("confd_path")
	return filepath.Join(confdPath, "snmp.d", "profiles")
}

func recursivelyExpandBaseProfiles(definition *profileDefinition, extends []string) error {
	for _, basePath := range extends {
		baseDefinition, err := readProfileDefinition(basePath)
		if err != nil {
			return err
		}
		definition.Metrics = append(definition.Metrics, baseDefinition.Metrics...)
		definition.MetricTags = append(definition.MetricTags, baseDefinition.MetricTags...)

		// TODO: Protect against infinite extend loop
		err = recursivelyExpandBaseProfiles(definition, baseDefinition.Extends)
		if err != nil {
			// TODO: Test me
			return err
		}
	}
	return nil
}

func getOidPatternSpecificity(pattern string) ([]int, error) {
	wildcardKey := -1
	var parts []int
	for _, part := range strings.Split(strings.TrimLeft(pattern, "."), ".") {
		if part == "*" {
			parts = append(parts, wildcardKey)
		} else {
			intPart, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("error parsing part `%s` for pattern `%s`: %v", part, pattern, err)
			}
			parts = append(parts, intPart)
		}
	}
	return parts, nil
}

func getMostSpecificOid(oids []string) (string, error) {
	var mostSpecificParts []int
	var mostSpecificOid string

	if len(oids) == 0 {
		return "", fmt.Errorf("cannot get most specific oid from empty list of oids")
	}

	for _, oid := range oids {
		parts, err := getOidPatternSpecificity(oid)
		if err != nil {
			return "", err
		}
		if len(parts) > len(mostSpecificParts) {
			mostSpecificParts = parts
			mostSpecificOid = oid
			continue
		}
		if len(parts) == len(mostSpecificParts) {
			for i := range mostSpecificParts {
				if parts[i] > mostSpecificParts[i] {
					mostSpecificParts = parts
					mostSpecificOid = oid
				}
			}
		}
	}
	return mostSpecificOid, nil
}