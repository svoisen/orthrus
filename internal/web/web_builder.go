package web

import (
	"log"
	"os"
	"path/filepath"
	"polypub/internal/file"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type WebBuilderConfig struct {
	AssetsDir   string
	TemplateDir string
	MarkdownDir string
	OutputDir   string
	PrintAst    bool
}

type TemplateData struct {
	Title   string
	Content []byte
}

type webBuilder struct {
	Config        WebBuilderConfig
	parser        *parser.Parser
	renderer      *html.Renderer
	templateCache *TemplateCache
}

type WebBuilder interface {
	Build() error
}

func NewWebBuilder(c WebBuilderConfig) WebBuilder {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	parser := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags
	rendererOptions := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(rendererOptions)

	templateCacheCfg := TemplateCacheConfig{
		Development: true,
		TemplateDir: c.TemplateDir,
	}
	templateCache := NewTemplateCache(templateCacheCfg)

	builder := &webBuilder{
		Config:        c,
		parser:        parser,
		renderer:      renderer,
		templateCache: templateCache,
	}

	return builder
}

func (b *webBuilder) Build() error {
	err := b.templateCache.LoadTemplates()
	if err != nil {
		log.Fatalf("[WebBuilder] unable to load templates: %w", err)
	}

	outputDir := b.Config.OutputDir

	err = file.RemoveIfExists(outputDir)
	if err != nil {
		log.Fatalf("[WebBuilder] unable to purge existing output dir: %w", err)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("[WebBuilder] could not create output directory: %v: %w\n", outputDir, err)
	}

	err = filepath.Walk(b.Config.MarkdownDir, b.createWalkFunc(outputDir))
	if err != nil {
		log.Fatalf("[WebBuilder] could not complete markdown conversion: %v\n", b.Config.MarkdownDir, err)
	}

	return nil
}

func (b *webBuilder) createWalkFunc(outputDir string) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("[WebBuilder] error walking directory: %w\n", err)
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		log.Printf("[WebBuilder] converting markdown to HTML: %v\n", path)
		html, err := b.markdownToHTML(path)
		if err != nil {
			log.Printf("[WebBuilder] error converting markdown to HTML, continuing\n")
			return nil
		}

		data := TemplateData{
			Title:   "Fix Me",
			Content: html,
		}
		rendered, err := b.templateCache.Render("base", data)
		if err != nil {
			log.Printf("[WebBuilder] error rendering template: %w\n", err)
			return err
		}

		outputPath := outputDir + "/" + file.Basename(path) + ".html"
		log.Printf("[WebBuilder] writing to file %v\n", outputPath)
		os.WriteFile(outputDir+"/"+file.Basename(path)+".html", []byte(rendered), 0644)
		return nil
	}
}

func (b *webBuilder) markdownToHTML(filepath string) ([]byte, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("[WebBuilder] unable to read file: %v\n", err)
		return nil, err
	}

	doc := b.parser.Parse(content)

	if b.Config.PrintAst {
		ast.Print(os.Stdout, doc)
	}

	return markdown.Render(doc, b.renderer), nil
}
