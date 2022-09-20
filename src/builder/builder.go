package builder

type Builder struct {
	Config *Configuration
}

type Configuration struct {
	Title       string            `yaml:"Title" validate:"required"`
	Description string            `yaml:"Description"`
	Footer      string            `yaml:"Footer"`
	Components  *ComponentsConfig `yaml:"Components" validate:"required"`
}

type ComponentsConfig struct {
	Backward string `yaml:"Backward" validate:"required"`
	Forward  string `yaml:"Forward" validate:"required"`
	Pause    string `yaml:"Pause" validate:"required"`
	Skip     string `yaml:"Skip" validate:"required"`
	Previous string `yaml:"Previous" validate:"required"`
	Replay   string `yaml:"Replay" validate:"required"`
	AddSongs string `yaml:"AddSongs" validate:"required"`
	Loop     string `yaml:"Loop" validate:"required"`
	Join     string `yaml:"Join" validate:"required"`
}

// NewBuilder constructs an object that handles building
// the queue's embed, components, ... based on it's current state
func NewBuilder(config *Configuration) *Builder {
	return &Builder{config}
}
