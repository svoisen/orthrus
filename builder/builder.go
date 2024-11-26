package builder

import (
	"bytes"
	"fmt"
	"ibeji/file"
	"ibeji/gemini"
	"ibeji/web"
	"os"
	"path/filepath"
	"strings"

	gemtext "git.sr.ht/~kota/goldmark-gemtext"
	wiki "git.sr.ht/~kota/goldmark-wiki"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
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
	GeminiFooter    string
}

type TemplateData struct {
	Title    string
	Content  []byte
	Filename string
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
		prepareOutputDir(b.Config.WebOutputDir)
	}

	if b.Config.BuildGemini {
		prepareOutputDir(b.Config.GeminiOutputDir)
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
		fmt.Printf("skipping file %v, not a markdown file\n", path)
		return nil
	}

	fileContents, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("unable to read file:", path)
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

func prepareOutputDir(dir string) error {
	err := file.RemoveIfExists(dir)
	if err != nil {
		fmt.Printf("unable to purge existing output dir: %v: %v\n", dir, err)
		os.Exit(1)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("could not create output directory: %v: %v\n", dir, err)
		os.Exit(1)
	}

	return nil
}

// createWalkFunc returns a function that is used to walk the markdown directory.
func (b *builder) createWalkFunc() func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("error walking directory:", err)
			os.Exit(1)
		}

		if info.IsDir() {
			return nil
		}

		return b.BuildFile(path)
	}
}

func (b *builder) outputHTML(contents []byte, filename string) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
			&wikilink.Extender{
				Resolver: web.WikilinkResolver{},
			},
			extension.Linkify,
			extension.Strikethrough,
			extension.Typographer,
		),
	)
	var buf bytes.Buffer
	reader := text.NewReader(contents)
	doc := md.Parser().Parse(reader)
	title, err := getTitle(doc, contents)
	if err != nil {
		fmt.Println("[WARN] could not get title for doc:", filename)
	}

	if err := md.Renderer().Render(&buf, contents, doc); err != nil {
		fmt.Println("could not convert markdown to HTML:", err)
		return err
	}

	outputFilename := transformFilename(filename)
	data := TemplateData{
		Title:    title,
		Content:  buf.Bytes(),
		Filename: outputFilename,
	}
	renderedHTML, err := b.templateCache.Render("base", data)
	if err != nil {
		fmt.Println("could not render template:", err)
		return err
	}

	outputPath := b.Config.WebOutputDir + "/" + outputFilename + ".html"
	fmt.Println("writing file:", outputPath)
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

	opts := []gemtext.Option{
		gemtext.WithHeadingLink(gemtext.HeadingLinkAuto),
		gemtext.WithHeadingSpace(gemtext.HeadingSpaceSingle),
		gemtext.WithParagraphLink(gemtext.ParagraphLinkOff),
		gemtext.WithListLink(gemtext.ListLinkAuto),
		gemtext.WithLinkReplacers([]gemtext.LinkReplacer{gemini.WikilinkReplacer}),
	}

	footer := []byte("\n\n" + b.Config.GeminiFooter)
	contents = append(contents, footer...)
	var buf bytes.Buffer
	md.SetRenderer(gemtext.New(opts...))
	if err := md.Convert(contents, &buf); err != nil {
		fmt.Println("failed to convert markdown to gemtext", err)
		return err
	}

	outputPath := b.Config.GeminiOutputDir + "/" + transformFilename(filename) + ".gmi"
	fmt.Println("writing file:", outputPath)
	os.WriteFile(outputPath, buf.Bytes(), 0644)

	return nil
}

func transformFilename(filename string) string {
	return strings.ReplaceAll(strings.ToLower(filename), " ", "-")
}

// getTitle returns the first heading in the document if found, or the empty
// string otherwise.
func getTitle(doc ast.Node, markdown []byte) (string, error) {
	var firstHeading string
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if heading, ok := n.(*ast.Heading); ok && entering {
			if heading.Level != 1 {
				return ast.WalkSkipChildren, nil
			}

			var buf bytes.Buffer
			for chld := heading.FirstChild(); chld != nil; chld = chld.NextSibling() {
				if text, ok := chld.(*ast.Text); ok {
					buf.Write(text.Segment.Value(markdown))
				}
			}
			firstHeading = buf.String()
			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})

	if firstHeading == "" {
		return "", fmt.Errorf("no heading found")
	}

	return firstHeading, nil
}
