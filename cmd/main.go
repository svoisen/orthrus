package main

import (
	"flag"
	"fmt"
	"ibeji/internal/build"
	"ibeji/internal/config"
	"ibeji/internal/gemini"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
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
		log.Fatalf("could not load config file")
	}

	switch args[0] {
	case "build":
		runBuild(cfg)
	case "serve":
		runBuild(cfg)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			watchMarkdownDir(cfg)
		}()
		go func() {
			runGeminiServer(cfg)
			wg.Done()
		}()
		wg.Wait()
	default:
		log.Println("unexpected subcommand:", args[0])
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

func runBuild(cfg config.Config) {
	log.Println("running build ...")
	builder := createBuilder(cfg)
	builder.BuildAll()
}
