package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"polypub/internal/config"
	"polypub/internal/web"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'serve' or 'build' subcommand")
		os.Exit(1)
	}

	var err error
	c, err := config.GetConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	switch args[0] {
	case "build":
		// buildGeminiCapsule(config.ContentDir)
		buildWeb(c)
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

func buildWeb(c config.Config) {
	webConfig := web.WebBuilderConfig{
		AssetsDir:   c.WebAssetsDir,
		TemplateDir: c.WebTemplateDir,
		MarkdownDir: c.MarkdownDir,
		OutputDir:   c.WebOutputDir,
		PrintAst:    false,
	}
	builder := web.NewWebBuilder(webConfig)
	builder.Build()
}
