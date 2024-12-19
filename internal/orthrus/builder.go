package orthrus

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gemtext "git.sr.ht/~kota/goldmark-gemtext"
	wiki "git.sr.ht/~kota/goldmark-wiki"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"
)

// In Ibeji, we normalize filenames and paths to lowercase and replace spaces
// with dashes.
func normalizePath(filepath string) string {
	return strings.ReplaceAll(strings.ToLower(filepath), " ", "-")
}

// GemtextWikilinkReplacer is a LinkReplacer that replaces wiki links with links
// to the normalized path for that link in any gemtext files.
var GemtextWikilinkReplacer = gemtext.LinkReplacer{
	Function: func(input string) string {
		return normalizePath(input) + ".gmi"
	},
	Type: gemtext.LinkWiki,
}

var _html = []byte(".html")
var _hash = []byte("#")

// WikilinkResolver resolves wikilinks to their normalized paths in any HTML
// files.
type WikilinkResolver struct{}

func (WikilinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	newTarget := []byte(normalizePath(string(n.Target)))
	dest := make([]byte, len(newTarget)+len(_html)+len(_hash)+len(n.Fragment))
	var i int
	if len(n.Target) > 0 {
		i += copy(dest, newTarget)
		if filepath.Ext(string(n.Target)) == "" {
			i += copy(dest[i:], _html)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	return dest[:i], nil
}

type WebTemplateData struct {
	Title    string
	Content  []byte
	Filename string
}

type builder struct {
	Config              Config
	WebTemplateCache    *TemplateCache
	GeminiTemplateCache *TemplateCache
}

type Builder interface {
	BuildAll() error
	BuildFile(path string) error
}

// NewBuilder creates a new Builder with the provided configuration.
func NewBuilder(cfg Config) Builder {
	builder := &builder{
		Config:              cfg,
		WebTemplateCache:    NewTemplateCache(),
		GeminiTemplateCache: NewTemplateCache(),
	}

	return builder
}

// BuildAll walks the markdown directory and builds all markdown files
// after clearing the output directories.
func (b *builder) BuildAll() error {
	// Prepare the output directory for web output
	if err := PurgeDir(b.Config.Web.OutputDir); err != nil {
		fmt.Println("could not purge web output directory:", err)
		return err
	}

	if err := CopyDir(b.Config.Web.AssetsDir, b.Config.Web.OutputDir); err != nil {
		fmt.Println("could not copy web assets:", err)
		return err
	}

	// Prepare the output directory for gemini output
	if err := PurgeDir(b.Config.Gemini.OutputDir); err != nil {
		fmt.Println("could not purge gemini output directory:", err)
		return err
	}

	// Load templates
	if err := b.WebTemplateCache.LoadTemplates(b.Config.Web.TemplateDir); err != nil {
		fmt.Println("could not load web templates:", err)
		return err
	}

	if err := b.GeminiTemplateCache.LoadTemplates(b.Config.Gemini.TemplateDir); err != nil {
		fmt.Println("could not load gemini templates:", err)
		return err
	}

	files, err := os.ReadDir(b.Config.Content.ContentDir)
	if err != nil {
		fmt.Println("could not read content directory:", err)
		return err
	}

	// We intentionally do not walk subdirectories of the content directory.
	// The builder assumes all markdown is in a flat directory structure.
	for _, file := range files {
		if IsMarkdownFile(file) {
			if err := b.BuildFile(filepath.Join(b.Config.Content.ContentDir, file.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

// BuildFile converts a single markdown file to HTML and gemtext and writes both
// to disk. path should be the full path to the file.)
func (b *builder) BuildFile(path string) error {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("unable to read file:", path)
		return err
	}

	if err := b.outputHTML(fileContents, path); err != nil {
		fmt.Printf("could not output HTML for file %v: %v\n", path, err)
		return err
	}

	if err := b.outputGemtext(fileContents, path); err != nil {
		fmt.Printf("could not output gemtext for file %v: %v\n", path, err)
		return err
	}

	return nil
}

// outputHTML converts markdown to HTML and writes the HTML file to disk.
// contents should be the markdown file contents
// path should be the full path to the input file
func (b *builder) outputHTML(contents []byte, path string) error {
	md := goldmark.New(
		// @Todo: Make this configurable
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
			&wikilink.Extender{
				Resolver: WikilinkResolver{},
			},
			&frontmatter.Extender{},
			extension.Linkify,
			extension.Strikethrough,
			extension.Typographer,
		),
	)

	outputFilename := normalizePath(Basename(path)) + ".html"

	var mdBuf bytes.Buffer
	reader := text.NewReader(contents)
	doc := md.Parser().Parse(reader)
	title, err := GetMarkdownTitle(doc, contents)
	if err != nil {
		fmt.Println("[WARN] could not get title from markdown for file:", path)
		title = ""
	}

	if err := md.Renderer().Render(&mdBuf, contents, doc); err != nil {
		fmt.Printf("could not convert markdown to HTML for file %v: %v\n", path, err)
		return err
	}

	data := WebTemplateData{
		Title:    title,
		Content:  mdBuf.Bytes(),
		Filename: Basename(outputFilename),
	}

	tmpl, ok := b.WebTemplateCache.GetTemplate(b.Config.Web.PageTemplate)
	if !ok {
		fmt.Println("could not find template:", b.Config.Web.PageTemplate)
		return fmt.Errorf("could not find template: %s", b.Config.Web.PageTemplate)
	}

	var html strings.Builder
	if err := tmpl.Execute(&html, data); err != nil {
		fmt.Println("could not render template:", err)
		return err
	}

	outputPath := filepath.Join(b.Config.Web.OutputDir, outputFilename)
	fmt.Println("writing file:", outputPath)
	if err := os.WriteFile(outputPath, []byte(html.String()), 0644); err != nil {
		fmt.Println("could not write file:", err)
		return err
	}

	return nil
}

func (b *builder) outputGemtext(contents []byte, path string) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			wiki.Wiki,
			extension.Linkify,
			extension.Strikethrough,
			&frontmatter.Extender{},
		),
	)

	opts := []gemtext.Option{
		gemtext.WithHeadingLink(gemtext.HeadingLinkAuto),
		gemtext.WithHeadingSpace(gemtext.HeadingSpaceSingle),
		gemtext.WithParagraphLink(gemtext.ParagraphLinkOff),
		gemtext.WithListLink(gemtext.ListLinkAuto),
		gemtext.WithLinkReplacers([]gemtext.LinkReplacer{GemtextWikilinkReplacer}),
	}

	var mdBuf bytes.Buffer
	md.SetRenderer(gemtext.New(opts...))
	if err := md.Convert(contents, &mdBuf); err != nil {
		fmt.Println("failed to convert markdown to gemtext", err)
		return err
	}

	tmpl, ok := b.GeminiTemplateCache.GetTemplate(b.Config.Gemini.PageTemplate)
	if !ok {
		fmt.Println("could not find template:", b.Config.Gemini.PageTemplate)
		return fmt.Errorf("could not find template: %s", b.Config.Gemini.PageTemplate)
	}

	data := map[string][]byte{
		"Content": mdBuf.Bytes(),
	}

	var gemtext strings.Builder
	if err := tmpl.Execute(&gemtext, data); err != nil {
		fmt.Println("could not render template:", err)
		return err
	}

	outputPath := filepath.Join(b.Config.Gemini.OutputDir, normalizePath(Basename(path))+".gmi")
	fmt.Println("writing file:", outputPath)
	if err := os.WriteFile(outputPath, []byte(gemtext.String()), 0644); err != nil {
		fmt.Println("could not write file:", err)
		return err
	}

	return nil
}
