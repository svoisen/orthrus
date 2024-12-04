package config

import "github.com/BurntSushi/toml"

// StreamConfig represents configuration information for a stream of posts.
type StreamConfig struct {
	Name        string
	Slug        string
	ContentPath string
}

type ContentConfig struct {
	Wikilinks  bool
	ContentDir string
}

type WebConfig struct {
	Enabled      bool
	Port         int
	TemplatePath string
	AssetsDir    string
	OutputDir    string
}

type GeminiConfig struct {
	Enabled   bool
	Hostname  string
	Port      int
	CertStore string
	OutputDir string
}

type Config struct {
	Content ContentConfig
	Web     WebConfig
	Gemini  GeminiConfig
	Streams []StreamConfig
}

func GetConfig(filename string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
