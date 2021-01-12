package agents

type Settings struct {
	SupportedVersions map[string][]string `mapstructure:"supported_versions"`
	VersionsEnv       string              `mapstructure:"versions"`
}

func DefaultSettings() *Settings {
	return &Settings{
		SupportedVersions: map[string][]string{},
	}
}
