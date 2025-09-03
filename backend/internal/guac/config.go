package guac

// Config represents the configuration for a Guacamole connection
type Config struct {
	ConnectionID          string
	Protocol              string
	Parameters            map[string]string
	OptimalScreenWidth    int
	OptimalScreenHeight   int
	OptimalResolution     int
	AudioMimetypes        []string
	VideoMimetypes        []string
	ImageMimetypes        []string
}

// NewGuacamoleConfiguration creates a new Guacamole configuration with default values
func NewGuacamoleConfiguration() *Config {
	return &Config{
		Parameters:            make(map[string]string),
		OptimalScreenWidth:    1920,
		OptimalScreenHeight:   1080,
		OptimalResolution:     96,
		AudioMimetypes:        []string{"audio/L16", "rate=44100", "channels=2"},
		VideoMimetypes:        []string{},
		ImageMimetypes:        []string{"image/jpeg", "image/png"},
	}
}
