package main

import (
	"flag"
	"fmt"
	"ibeji/builder"
	"ibeji/config"
	"ibeji/gemini"
	"ibeji/web"
	"os"
	"path/filepath"
	"strings"
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
		if stat, _ := os.Stat("config.toml"); stat == nil {
			fmt.Println("no config file found nor provided with config flag")
			os.Exit(1)
		}

		*configPath = "config.toml"
	}

	// Ensure a valid argument is provided
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'serve' or 'build' as an argument")
		os.Exit(1)
	}

	// Get the config
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
		// @TODO: Is there a better way to do this?
		wg := new(sync.WaitGroup)
		wg.Add(3)
		go func() {
			watchDirs(cfg)
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

func watchDirs(cfg config.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("could not create watcher:", err)
		os.Exit(1)
	}

	done := make(chan bool)

	defer watcher.Close()

	go func() {
		builder := builder.NewBuilder(cfg)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("detected modified file:", event.Name)
					ext := strings.ToLower(filepath.Ext(event.Name))
					switch ext {
					case ".tmpl":
						builder.BuildAll()
					case ".md", ".markdown":
						builder.BuildFile(event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("watcher error:", err)
			}
		}
	}()

	dirs := []string{cfg.Content.ContentDir, filepath.Dir(cfg.Web.TemplatePath)}
	for _, dir := range dirs {
		fmt.Println("watching directory:", dir)
		err = watcher.Add(dir)
		if err != nil {
			fmt.Println("could not watch directory", err)
			os.Exit(1)
		}
	}
	<-done
}

func runGeminiServer(cfg config.Config) {
	server := gemini.NewGeminiServer(cfg.Gemini)
	if err := server.Start(); err != nil {
		os.Exit(1)
	}
}

func runWebServer(cfg config.Config) {
	server := web.NewWebServer(cfg.Web)
	if err := server.Start(); err != nil {
		os.Exit(1)
	}
}

func runBuild(cfg config.Config) {
	builder := builder.NewBuilder(cfg)
	err := builder.BuildAll()
	if err != nil {
		os.Exit(1)
	}
}
