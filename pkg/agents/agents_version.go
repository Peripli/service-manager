package agents

type Settings struct {
	Versions map[string][]string `mapstructure:"versions"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Versions: map[string][]string{},
	}
}
