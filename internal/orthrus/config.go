package orthrus

import "github.com/BurntSushi/toml"

type ContentConfig struct {
	Wikilinks  bool
	ContentDir string
}

type AssetsConfig struct {
	SourceDir string
	DestDir   string
}

type WebConfig struct {
	Enabled      bool
	Port         int
	TemplateDir  string
	PageTemplate string
	AssetsDir    string
	OutputDir    string
}

type GeminiConfig struct {
	Enabled      bool
	Hostname     string
	Port         int
	CertStore    string
	TemplateDir  string
	PageTemplate string
	OutputDir    string
}

type Config struct {
	SiteName string
	Content  ContentConfig
	Web      WebConfig
	Gemini   GeminiConfig
	Assets   []AssetsConfig
}

// GetConfig reads the configuration from a file and returns a Config struct.
func GetConfig(filename string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
