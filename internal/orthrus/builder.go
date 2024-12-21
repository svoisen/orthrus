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

var GemTextExt = ".gmi"
var HTMLExt = ".html"

// In Orthrus, we normalize filenames and paths to lowercase and replace spaces
// with dashes.
func normalizePath(filepath string) string {
	return strings.ReplaceAll(strings.ToLower(filepath), " ", "-")
}

// GemtextWikilinkReplacer is a LinkReplacer that replaces wiki links with links
// to the normalized path for that link in any gemtext files.
var GemtextWikilinkReplacer = gemtext.LinkReplacer{
	Function: func(input string) string {
		return normalizePath(input) + GemTextExt
	},
	Type: gemtext.LinkWiki,
}

// HTMLWikilinkResolver resolves wikilinks to their normalized paths in any HTML
// files.
type HTMLWikilinkResolver struct{}

func (HTMLWikilinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	_html := []byte(HTMLExt)
	_hash := []byte("#")

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

type TemplateData struct {
	SiteName string
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
	BuildFile(srcPath string, webDestDir string, geminiDestDir string) error
	LoadTemplates() error
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
	// Load all templates
	if err := b.LoadTemplates(); err != nil {
		fmt.Println("could not load templates:", err)
		return err
	}

	// Prepare the output directory for web output
	if err := PurgeDir(b.Config.Web.OutputDir); err != nil {
		fmt.Println("could not purge web output directory:", err)
		return err
	}

	// Prepare the output directory for gemini output
	if err := PurgeDir(b.Config.Gemini.OutputDir); err != nil {
		fmt.Println("could not purge gemini output directory:", err)
		return err
	}

	// Copy all assets
	for _, assetSet := range b.Config.Assets {
		fmt.Println("copying assets:", assetSet.SourceDir, "to", assetSet.DestDir)

		if err := PurgeDir(assetSet.DestDir); err != nil {
			fmt.Println("could not purge asset directory:", err)
			return err
		}

		if err := CopyDir(assetSet.SourceDir, assetSet.DestDir); err != nil {
			fmt.Println("could not copy assets:", err)
			return err
		}
	}

	// Build the content for each stream
	for _, stream := range b.Config.Streams {
		webDestDir := normalizePath(filepath.Join(b.Config.Web.OutputDir, stream.Name))
		geminiDestDir := normalizePath(filepath.Join(b.Config.Gemini.OutputDir, stream.Name))

		if err := PurgeDir(webDestDir); err != nil {
			fmt.Println("could not purge stream directory:", err)
			return err
		}

		if err := PurgeDir(geminiDestDir); err != nil {
			fmt.Println("could not purge stream directory:", err)
			return err
		}

		if err := b.buildContent(stream.ContentDir, webDestDir, geminiDestDir); err != nil {
			return err
		}
	}

	// Build the main content
	if err := b.buildContent(b.Config.Content.ContentDir, b.Config.Web.OutputDir, b.Config.Gemini.OutputDir); err != nil {
		return err
	}

	return nil
}

func (b *builder) LoadTemplates() error {
	if err := b.WebTemplateCache.LoadTemplates(b.Config.Web.TemplateDir); err != nil {
		fmt.Println("could not load web templates:", err)
		return err
	}

	if err := b.GeminiTemplateCache.LoadTemplates(b.Config.Gemini.TemplateDir); err != nil {
		fmt.Println("could not load gemini templates:", err)
		return err
	}

	return nil
}

// BuildFile converts a single markdown file to HTML and gemtext and writes both
// to disk. path should be the full path to the file.)
func (b *builder) BuildFile(srcPath string, webDestDir string, geminiDestDir string) error {
	fileContents, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Println("unable to read file:", srcPath)
		return err
	}

	outputFilename := normalizePath(Basename(srcPath))

	destPath := filepath.Join(webDestDir, outputFilename+HTMLExt)
	if err := b.outputHTML(fileContents, destPath); err != nil {
		fmt.Printf("could not output HTML for file %v: %v\n", srcPath, err)
		return err
	}

	destPath = filepath.Join(geminiDestDir, outputFilename+GemTextExt)
	if err := b.outputGemtext(fileContents, destPath); err != nil {
		fmt.Printf("could not output gemtext for file %v: %v\n", srcPath, err)
		return err
	}

	return nil
}

func (b *builder) buildContent(srcDir string, webDestDir string, geminiDestDir string) error {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Println("could not read content directory:", err)
		return err
	}

	// We intentionally do not walk subdirectories of the content directory.
	// The builder assumes all markdown is in a flat directory structure.
	for _, file := range files {
		if IsMarkdownFile(file) {
			srcPath := filepath.Join(srcDir, file.Name())
			if err := b.BuildFile(srcPath, webDestDir, geminiDestDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// outputHTML converts markdown to HTML and writes the HTML file to disk.
// markdown should be the markdown file contents
// destPath should be the full path to the location of the output
func (b *builder) outputHTML(markdown []byte, destPath string) error {
	md := goldmark.New(
		// @Todo: Make this configurable
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
			&wikilink.Extender{
				Resolver: HTMLWikilinkResolver{},
			},
			&frontmatter.Extender{},
			extension.Linkify,
			extension.Strikethrough,
			extension.Typographer,
		),
	)

	var mdBuf bytes.Buffer
	reader := text.NewReader(markdown)
	doc := md.Parser().Parse(reader)
	title, err := GetMarkdownTitle(doc, markdown)
	if err != nil {
		fmt.Println("[WARN] could not get title from markdown for file:", destPath)
		title = ""
	}

	if err := md.Renderer().Render(&mdBuf, markdown, doc); err != nil {
		fmt.Printf("could not convert markdown to HTML for file %v: %v\n", destPath, err)
		return err
	}

	data := TemplateData{
		SiteName: b.Config.SiteName,
		Title:    title,
		Content:  mdBuf.Bytes(),
		Filename: Basename(destPath),
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

	fmt.Println("writing file:", destPath)
	if err := os.WriteFile(destPath, []byte(html.String()), 0644); err != nil {
		fmt.Println("could not write file:", err)
		return err
	}

	return nil
}

func (b *builder) outputGemtext(contents []byte, destPath string) error {
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

	fmt.Println("writing file:", destPath)
	if err := os.WriteFile(destPath, []byte(gemtext.String()), 0644); err != nil {
		fmt.Println("could not write file:", err)
		return err
	}

	return nil
}
