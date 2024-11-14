package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"polypub/internal/build"
	"polypub/internal/config"
	"polypub/internal/gemini"
	"sync"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'serve' or 'build' subcommand")
		os.Exit(1)
	}

	var err error
	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		log.Fatalf("could not load config: %w", err)
	}

	switch args[0] {
	case "build":
		fmt.Println("building ...")
		runBuild(cfg)
	case "serve":
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			runGeminiServer(cfg)
			wg.Done()
		}()
		wg.Wait()
	default:
		fmt.Printf("unexpected subcommand %v\n", args[0])
	}
}

func runGeminiServer(cfg config.Config) {
	fmt.Println("starting gemini server ...")
	serverCfg := gemini.GeminiServerConfig{
		ContentDir: cfg.GeminiOutputDir,
		HostName:   cfg.HostName,
		CertStore:  cfg.GeminiCertStore,
		Port:       1965,
	}

	server := gemini.NewGeminiServer(serverCfg)
	server.Start()
}

func runBuild(cfg config.Config) {
	builderCfg := build.BuilderConfig{
		AssetsDir:       cfg.WebAssetsDir,
		TemplateDir:     cfg.WebTemplateDir,
		MarkdownDir:     cfg.MarkdownDir,
		WebOutputDir:    cfg.WebOutputDir,
		GeminiOutputDir: cfg.GeminiOutputDir,
		BuildWeb:        false,
		BuildGemini:     true,
		PrintAst:        false,
	}
	builder := build.NewBuilder(builderCfg)
	builder.Build()
}
