package config

import "github.com/BurntSushi/toml"

type Config struct {
	MarkdownDir     string
	HostName        string
	GeminiCertStore string
	GeminiOutputDir string
	WebAssetsDir    string
	WebTemplateDir  string
	WebOutputDir    string
}

func GetConfig(filename string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
