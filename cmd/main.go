package main

import (
	"flag"
	"fmt"
	"ibeji/internal/build"
	"ibeji/internal/config"
	"ibeji/internal/gemini"
	"ibeji/internal/web"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

func main() {
	// Define flags
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.StringVar(configPath, "c", "config.toml", "path to config file (shorthand)")

	// Parse flags
	flag.Parse()

	// Ensure a config path is provided
	if *configPath == "" {
		fmt.Println("config file path is required")
		os.Exit(1)
	}

	// Ensure a valid subcommand is provided
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'serve' or 'build' subcommand")
		os.Exit(1)
	}

	var err error
	cfg, err := config.GetConfig(*configPath)
	if err != nil {
		fmt.Println("could not load config file")
		os.Exit(1)
	}

	switch args[0] {
	case "build":
		runBuild(cfg)
	case "serve":
		runBuild(cfg)
		wg := new(sync.WaitGroup)
		wg.Add(3)
		go func() {
			watchMarkdownDir(cfg)
			wg.Done()
		}()
		go func() {
			runWebServer(cfg)
			wg.Done()
		}()
		go func() {
			runGeminiServer(cfg)
			wg.Done()
		}()
		wg.Wait()
	default:
		fmt.Println("unexpected subcommand:", args[0])
		fmt.Println("valid commands are 'build' or 'serve'")
		os.Exit(1)
	}
}

func watchMarkdownDir(cfg config.Config) {
	log.Println("watching markdown directory:", cfg.MarkdownDir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("could not create watcher: %v", err)
	}

	done := make(chan bool)

	defer watcher.Close()

	go func() {
		builder := createBuilder(cfg)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					builder.BuildFile(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(cfg.MarkdownDir)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func createBuilder(cfg config.Config) build.Builder {
	builderCfg := build.BuilderConfig{
		AssetsDir:       cfg.WebAssetsDir,
		TemplateDir:     cfg.WebTemplateDir,
		MarkdownDir:     cfg.MarkdownDir,
		WebOutputDir:    cfg.WebOutputDir,
		GeminiOutputDir: cfg.GeminiOutputDir,
		BuildWeb:        true,
		BuildGemini:     true,
		PrintAst:        false,
	}
	return build.NewBuilder(builderCfg)
}

func runGeminiServer(cfg config.Config) {
	log.Println("starting gemini server ...")
	serverCfg := gemini.GeminiServerConfig{
		ContentDir: cfg.GeminiOutputDir,
		HostName:   cfg.HostName,
		CertStore:  cfg.GeminiCertStore,
		Port:       1965,
	}

	server := gemini.NewGeminiServer(serverCfg)
	server.Start()
}

func runWebServer(cfg config.Config) {
	serverCfg := web.WebServerConfig{
		ContentDir: cfg.WebOutputDir,
		Port:       8080,
	}
	server := web.NewWebServer(serverCfg)
	server.Start()
}

func runBuild(cfg config.Config) {
	builder := createBuilder(cfg)
	builder.BuildAll()
}
