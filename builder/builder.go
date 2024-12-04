package builder

import (
	"bytes"
	"fmt"
	"html/template"
	"ibeji/config"
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

type TemplateData struct {
	Title    string
	Content  []byte
	Filename string
}

type builder struct {
	Config config.Config
}

type Builder interface {
	BuildAll() error
	BuildFile(path string) error
}

func NewBuilder(cfg config.Config) Builder {
	builder := &builder{
		Config: cfg,
	}

	return builder
}

func getFuncMap() template.FuncMap {
	return template.FuncMap{
		"bytesToHTML": func(b []byte) template.HTML {
			return template.HTML(string(b))
		},
	}
}

// BuildAll walks the markdown directory and builds all markdown files
// after clearing the output directories.
func (b *builder) BuildAll() error {
	if b.Config.Web.Enabled {
		if err := prepareOutputDir(b.Config.Web.OutputDir); err != nil {
			return err
		}

		b.copyWebAssets()
		b.createWebTemplate()
	}

	if b.Config.Gemini.Enabled {
		prepareOutputDir(b.Config.Gemini.OutputDir)
	}

	files, err := os.ReadDir(b.Config.Content.ContentDir)
	if err != nil {
		fmt.Println("could not read content directory:", err)
		return err
	}

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !file.IsDir() && (ext == ".md" || ext == ".markdown") {
			if err := b.BuildFile(b.Config.Content.ContentDir + "/" + file.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

// BuildFile converts a single markdown file to HTML and gemtext.
func (b *builder) BuildFile(path string) error {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("unable to read file:", path)
		return err
	}

	filename := file.Basename(path)

	if b.Config.Web.Enabled {
		if err := b.outputHTML(fileContents, filename); err != nil {
			fmt.Printf("could not output HTML for file %v: %v", filename, err)
			return err
		}
	}

	if b.Config.Gemini.Enabled {
		b.outputGemtext(fileContents, filename)
	}

	return nil
}

func (b *builder) createWebTemplate() error {
	_, err := template.New("template").Funcs(getFuncMap()).ParseFiles(b.Config.Web.TemplatePath)
	if err != nil {
		return err
	}

	return nil
}

func prepareOutputDir(dir string) error {
	err := file.RemoveIfExists(dir)
	if err != nil {
		fmt.Printf("unable to purge existing output dir: %v: %v\n", dir, err)
		return err
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("could not create output directory: %v: %v\n", dir, err)
		return err
	}

	return nil
}

func (b *builder) copyWebAssets() error {
	return file.CopyDir(b.Config.Web.AssetsDir, b.Config.Web.OutputDir)
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

	var mdBuf bytes.Buffer
	reader := text.NewReader(contents)
	doc := md.Parser().Parse(reader)
	title, err := getTitle(doc, contents)
	if err != nil {
		fmt.Println("[WARN] could not get title for doc:", filename)
	}

	if err := md.Renderer().Render(&mdBuf, contents, doc); err != nil {
		fmt.Println("could not convert markdown to HTML:", err)
		return err
	}

	outputFilename := normalizeFilename(filename)
	data := TemplateData{
		Title:    title,
		Content:  mdBuf.Bytes(),
		Filename: outputFilename,
	}

	var html strings.Builder
	template, err := template.New(filepath.Base(b.Config.Web.TemplatePath)).Funcs(getFuncMap()).ParseFiles(b.Config.Web.TemplatePath)
	if err != nil {
		fmt.Println("could not parse template:", err)
		return err
	}
	err = template.Execute(&html, data)
	if err != nil {
		fmt.Println("could not render template:", err)
		return err
	}

	outputPath := b.Config.Web.OutputDir + "/" + outputFilename + ".html"
	fmt.Println("writing file:", outputPath)
	os.WriteFile(outputPath, []byte(html.String()), 0644)

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

	var buf bytes.Buffer
	md.SetRenderer(gemtext.New(opts...))
	if err := md.Convert(contents, &buf); err != nil {
		fmt.Println("failed to convert markdown to gemtext", err)
		return err
	}

	outputPath := b.Config.Gemini.OutputDir + "/" + normalizeFilename(filename) + ".gmi"
	fmt.Println("writing file:", outputPath)
	os.WriteFile(outputPath, buf.Bytes(), 0644)

	return nil
}

func normalizeFilename(filename string) string {
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
