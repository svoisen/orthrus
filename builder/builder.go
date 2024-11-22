package builder

import (
	"bytes"
	"fmt"
	"ibeji/file"
	"ibeji/web"
	"log"
	"os"
	"path/filepath"
	"strings"

	gem "git.sr.ht/~kota/goldmark-gemtext"
	wiki "git.sr.ht/~kota/goldmark-wiki"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"go.abhg.dev/goldmark/wikilink"
)

type BuilderConfig struct {
	BuildWeb        bool
	BuildGemini     bool
	AssetsDir       string
	TemplateDir     string
	MarkdownDir     string
	WebOutputDir    string
	GeminiOutputDir string
	PrintAst        bool
}

type TemplateData struct {
	Title   string
	Content []byte
}

type builder struct {
	Config        BuilderConfig
	templateCache *web.TemplateCache
}

type Builder interface {
	BuildAll() error
	BuildFile(path string) error
}

func NewBuilder(c BuilderConfig) Builder {
	templateCacheCfg := web.TemplateCacheConfig{
		Development: true,
		TemplateDir: c.TemplateDir,
	}
	templateCache := web.NewTemplateCache(templateCacheCfg)
	err := templateCache.LoadTemplates()
	if err != nil {
		fmt.Printf("unable to load templates: %v", err)
		os.Exit(1)
	}

	builder := &builder{
		Config:        c,
		templateCache: templateCache,
	}

	return builder
}

// BuildAll walks the markdown directory and builds all markdown files
// after clearing the output directories.
func (b *builder) BuildAll() error {
	if b.Config.BuildWeb {
		b.prepareWebBuild()
	}

	if b.Config.BuildGemini {
		b.prepareGeminiBuild()
	}

	err := filepath.Walk(b.Config.MarkdownDir, b.createWalkFunc())
	if err != nil {
		fmt.Println("could not complete markdown conversion", err)
		os.Exit(1)
	}

	return nil
}

// BuildFile converts a single markdown file to HTML and gemtext.
func (b *builder) BuildFile(path string) error {
	ext := strings.ToLower(filepath.Ext(path))

	if ext != ".md" && ext != ".markdown" {
		log.Printf("skipping file %v, not a markdown file\n", path)
		return nil
	}

	fileContents, err := os.ReadFile(path)
	if err != nil {
		log.Printf("unable to read file %v: %v\n", path, err)
		return err
	}

	filename := file.Basename(path)

	if b.Config.BuildWeb {
		b.outputHTML(fileContents, filename)
	}

	if b.Config.BuildGemini {
		b.outputGemtext(fileContents, filename)
	}

	return nil
}

// prepareWebBuild prepares the output directory for web content.
func (b *builder) prepareWebBuild() {
	b.prepareOutputDir(b.Config.WebOutputDir)
}

// prepareGeminiBuild prepares the output directory for gemini content.
func (b *builder) prepareGeminiBuild() {
	b.prepareOutputDir(b.Config.GeminiOutputDir)
}

func (b *builder) prepareOutputDir(dir string) error {
	err := file.RemoveIfExists(dir)
	if err != nil {
		log.Fatalf("[Builder] unable to purge existing output dir: %v: %v", dir, err)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatalf("[Builder] could not create output directory: %v: %v\n", dir, err)
	}

	return nil
}

// createWalkFunc returns a function that is used to walk the markdown directory.
func (b *builder) createWalkFunc() func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("[Builder] error walking directory: %v\n", err)
		}

		if info.IsDir() {
			return nil
		}

		return b.BuildFile(path)
	}
}

func (b *builder) outputHTML(contents []byte, filename string) error {
	fmt.Println("converting markdown to HTML:", filename)

	md := goldmark.New(
		goldmark.WithExtensions(
			&wikilink.Extender{},
			extension.Linkify,
			extension.Strikethrough,
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(contents, &buf); err != nil {
		fmt.Println("could not convert markdown to HTML:", err)
		return err
	}

	data := TemplateData{
		Title:   "Fix Me",
		Content: buf.Bytes(),
	}
	renderedHTML, err := b.templateCache.Render("base", data)
	if err != nil {
		fmt.Println("could not render template:", err)
		return err
	}

	outputPath := b.Config.WebOutputDir + "/" + filename + ".html"
	fmt.Println("writing to file:", outputPath)
	os.WriteFile(outputPath, []byte(renderedHTML), 0644)

	return nil
}

func (b *builder) outputGemtext(contents []byte, filename string) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			wiki.Wiki,
			extension.Linkify,
			extension.Strikethrough,
		),
	)

	opts := []gem.Option{
		gem.WithHeadingLink(gem.HeadingLinkAuto),
		gem.WithParagraphLink(gem.ParagraphLinkOff),
		gem.WithListLink(gem.ListLinkAuto),
	}
	var buf bytes.Buffer
	md.SetRenderer(gem.New(opts...))
	fmt.Println("converting markdown to gemtext:", filename)
	if err := md.Convert(contents, &buf); err != nil {
		fmt.Println("failed to convert markdown to gemtext", err)
		return err
	}

	outputPath := b.Config.GeminiOutputDir + "/" + filename + ".gmi"
	fmt.Println("writing to file:", outputPath)
	os.WriteFile(outputPath, buf.Bytes(), 0644)

	return nil
}
