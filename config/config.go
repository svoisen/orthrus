package config

import "github.com/BurntSushi/toml"

// Stream represents configuration information for a stream of posts.
type Stream struct {
	Name       string
	Slug       string
	ContentDir string
}

type ContentConfig struct {
	Wikilinks  bool
	ContentDir string
}

type WebConfig struct {
	Enabled     bool
	Port        int
	TemplateDir string
	AssetsDir   string
	OutputDir   string
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
	Streams []Stream
}

func GetConfig(filename string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
