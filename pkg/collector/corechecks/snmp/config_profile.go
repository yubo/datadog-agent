package snmp

type profilesConfig map[string]profileConfig

type profileConfig struct {
	DefinitionFile string `yaml:"definition_file"`
}
