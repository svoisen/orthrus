package build

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TemplateCache struct {
	templates    map[string]*template.Template
	lastModTime  map[string]time.Time
	development  bool
	cacheLock    sync.RWMutex
	templateRoot string
}

// Config holds template configuration
type TemplateCacheConfig struct {
	Development bool // If true, templates will be reloaded on each request
	TemplateDir string
}

func NewTemplateCache(config TemplateCacheConfig) *TemplateCache {
	return &TemplateCache{
		templates:    make(map[string]*template.Template),
		lastModTime:  make(map[string]time.Time),
		development:  config.Development,
		templateRoot: config.TemplateDir,
	}
}

// getFuncMap returns the template function map
func getFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"bytesToHTML": func(b []byte) template.HTML {
			return template.HTML(string(b))
		},
	}
}

func (t *TemplateCache) LoadTemplates() error {
	t.cacheLock.Lock()
	defer t.cacheLock.Unlock()

	return t.loadTemplatesInternal()
}

func (t *TemplateCache) loadTemplatesInternal() error {
	// Clear existing templates
	t.templates = make(map[string]*template.Template)
	t.lastModTime = make(map[string]time.Time)

	// Walk through the templates directory
	err := filepath.WalkDir(t.templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory or not a .tmpl file
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Get file modification time for cache invalidation
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("error getting file info: %w", err)
		}
		t.lastModTime[path] = info.ModTime()

		// Get the template name without extension and templates/ prefix
		name := strings.TrimPrefix(path, t.templateRoot+"/")
		name = strings.TrimSuffix(name, ".tmpl")

		// Parse the base template first
		tmpl := template.New("base.tmpl").Funcs(getFuncMap())

		// Parse all remaining templates
		tmpl, err = tmpl.ParseGlob(filepath.Join(t.templateRoot, "*.tmpl"))
		if err != nil {
			return fmt.Errorf("error parsing templates: %w", err)
		}

		// Store the template in our map
		t.templates[name] = tmpl

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking templates directory: %w", err)
	}

	return nil
}

// checkTemplateChanges checks if any templates have been modified
func (t *TemplateCache) checkTemplateChanges() bool {
	changed := false

	err := filepath.WalkDir(t.templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		lastMod, exists := t.lastModTime[path]
		if !exists || info.ModTime().After(lastMod) {
			changed = true
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		// Log error but continue with existing templates
		fmt.Printf("Error checking template changes: %v\n", err)
	}

	return changed
}

// getTemplate returns the template, reloading if necessary in development mode
func (t *TemplateCache) getTemplate(name string) (*template.Template, error) {
	t.cacheLock.RLock()
	tmpl, exists := t.templates[name]
	t.cacheLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}

	if t.development {
		if t.checkTemplateChanges() {
			t.cacheLock.Lock()
			err := t.loadTemplatesInternal()
			t.cacheLock.Unlock()
			if err != nil {
				return nil, fmt.Errorf("error reloading templates: %w", err)
			}

			t.cacheLock.RLock()
			tmpl = t.templates[name]
			t.cacheLock.RUnlock()
		}
	}

	return tmpl, nil
}

// Render executes a template with the given name and data
func (t *TemplateCache) Render(name string, data interface{}) (string, error) {
	tmpl, err := t.getTemplate(name)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = tmpl.ExecuteTemplate(&buf, "base.tmpl", data)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

// RenderPartial renders a specific template block without the base template
func (t *TemplateCache) RenderPartial(name, block string, data interface{}) (string, error) {
	tmpl, err := t.getTemplate(name)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = tmpl.ExecuteTemplate(&buf, block, data)
	if err != nil {
		return "", fmt.Errorf("error executing partial template: %w", err)
	}

	return buf.String(), nil
}
