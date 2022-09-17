package builder

type Builder struct {
	components *ComponentsConfig
}

type ComponentsConfig struct {
	Backward string `yaml:"Forward" validate:"required"`
	Forward  string `yaml:"Backward" validate:"required"`
	Pause    string `yaml:"Pause" validate:"required"`
	Skip     string `yaml:"Skip" validate:"required"`
	Previous string `yaml:"Previous" validate:"required"`
	Replay   string `yaml:"Replay" validate:"required"`
	AddSongs string `yaml:"AddSongs" validate:"required"`
	Loop     string `yaml:"Loop" validate:"required"`
}

// NewBuilder constructs an object that handles building
// the queue's embed, components, ... based on it's current
// state
func NewBuilder(components *ComponentsConfig) *Builder {
	return &Builder{components}
}
