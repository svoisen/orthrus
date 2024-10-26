package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"polypub/internal/build"
	"polypub/internal/config"
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
		log.Fatal(err)
	}

	switch args[0] {
	case "build":
		runBuild(cfg)
	default:
		fmt.Printf("unexpected subcommand %v\n", args[0])
	}

	// case "serve":
	// 	wg := new(sync.WaitGroup)
	// 	wg.Add(1)
	// 	go func() {
	// 		initGeminiServer()
	// 		wg.Done()
	// 	}()
	// 	wg.Wait()
	// }
}

func runBuild(cfg config.Config) {
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
	builder := build.NewBuilder(builderCfg)
	builder.Build()
}
