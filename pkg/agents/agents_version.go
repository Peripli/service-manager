package agents

type Settings struct {
	Versions string `mapstructure:"versions"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Versions: "",
	}
}
