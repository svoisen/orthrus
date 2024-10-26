package build

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"polypub/internal/file"
	"strings"

	gem "git.sr.ht/~kota/goldmark-gemtext"
	wiki "git.sr.ht/~kota/goldmark-wiki"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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
	parser        *parser.Parser
	htmlRenderer  *html.Renderer
	templateCache *TemplateCache
}

type Builder interface {
	Build() error
}

func NewBuilder(c BuilderConfig) Builder {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	parser := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags
	htmlRendererOptions := html.RendererOptions{Flags: htmlFlags}
	htmlRenderer := html.NewRenderer(htmlRendererOptions)

	templateCacheCfg := TemplateCacheConfig{
		Development: true,
		TemplateDir: c.TemplateDir,
	}
	templateCache := NewTemplateCache(templateCacheCfg)

	builder := &builder{
		Config:        c,
		parser:        parser,
		htmlRenderer:  htmlRenderer,
		templateCache: templateCache,
	}

	return builder
}

func (b *builder) Build() error {
	if b.Config.BuildWeb {
		b.prepareWebBuild()
	}

	if b.Config.BuildGemini {
		b.prepareGeminiBuild()
	}

	err := filepath.Walk(b.Config.MarkdownDir, b.createWalkFunc())
	if err != nil {
		log.Fatalf("[Builder] could not complete markdown conversion: %v\n", b.Config.MarkdownDir, err)
	}

	return nil
}

func (b *builder) prepareWebBuild() {
	err := b.templateCache.LoadTemplates()
	if err != nil {
		log.Fatalf("[Builder] unable to load templates: %w", err)
	}

	b.prepareOutputDir(b.Config.WebOutputDir)
}

func (b *builder) prepareGeminiBuild() {
	b.prepareOutputDir(b.Config.GeminiOutputDir)
}

func (b *builder) prepareOutputDir(dir string) error {
	err := file.RemoveIfExists(dir)
	if err != nil {
		log.Fatalf("[Builder] unable to purge existing output dir: %v: %w", dir, err)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatalf("[Builder] could not create output directory: %v: %w\n", dir, err)
	}

	return nil
}

func (b *builder) createWalkFunc() func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("[Builder] error walking directory: %w\n", err)
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		fileContents, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[Builder] unable to read file %v, continuing: %w\n", path, err)
			return nil
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
}

func (b *builder) outputHTML(contents []byte, filename string) error {
	log.Printf("[Builder] converting markdown to HTML: %v\n", filename)
	renderedMarkdown, err := b.markdownToHTML(contents)
	if err != nil {
		log.Printf("[Builder] failed to convert markdown to HTML: %w\n", err)
		return err
	}

	data := TemplateData{
		Title:   "Fix Me",
		Content: renderedMarkdown,
	}
	renderedHTML, err := b.templateCache.Render("base", data)
	if err != nil {
		log.Printf("[Builder] error rendering template: %w\n", err)
		return err
	}

	outputPath := b.Config.WebOutputDir + "/" + filename + ".html"
	log.Printf("[Builder] writing to file %v\n", outputPath)
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
		gem.WithHeadingLink(gem.HeadingLinkBelow),
		gem.WithParagraphLink(gem.ParagraphLinkBelow),
	}
	var buf bytes.Buffer
	md.SetRenderer(gem.New(opts...))
	if err := md.Convert(contents, &buf); err != nil {
		log.Printf("[Builder] failed to convert markdown to gemtext: %v\n", err)
		return err
	}

	outputPath := b.Config.GeminiOutputDir + "/" + filename + ".gmi"
	log.Printf("[Builder] writing to file %v\n", outputPath)
	os.WriteFile(outputPath, buf.Bytes(), 0644)

	return nil
}

func (b *builder) markdownToHTML(input []byte) ([]byte, error) {
	doc := b.parser.Parse(input)

	if b.Config.PrintAst {
		ast.Print(os.Stdout, doc)
	}

	return markdown.Render(doc, b.htmlRenderer), nil
}
