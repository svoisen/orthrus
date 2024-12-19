package orthrus

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// funcMap returns the template function map
func funcMap() template.FuncMap {
	return template.FuncMap{
		"bytesToHTML": func(b []byte) template.HTML {
			return template.HTML(string(b))
		},
	}
}

type TemplateCache struct {
	templates map[string]*template.Template
	cacheLock sync.RWMutex
}

func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*template.Template),
	}
}

// LoadTemplates loads all templates in the provided directory into the cache.
func (t *TemplateCache) LoadTemplates(templateDir string) error {
	t.cacheLock.Lock()
	defer t.cacheLock.Unlock()

	partialsPaths, err := filepath.Glob(filepath.Join(templateDir, "_*.tmpl"))
	if err != nil {
		return fmt.Errorf("could not load partials: %w", err)
	}

	t.templates = make(map[string]*template.Template)
	files, err := os.ReadDir(templateDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		// Ignore files that are NOT templates or are partials
		if !IsTemplateFile(file) || strings.HasPrefix(file.Name(), "_") {
			continue
		}

		name := strings.TrimSuffix(file.Name(), ".tmpl")

		filesToParse := append([]string{}, partialsPaths...)
		filesToParse = append(filesToParse, filepath.Join(templateDir, file.Name()))
		tmpl, err := template.New(name).Funcs(funcMap()).ParseFiles(filesToParse...)
		if err != nil {
			return err
		}

		// Store the template in the cache
		t.templates[name] = tmpl
	}

	return nil
}

func (t *TemplateCache) GetTemplate(name string) (*template.Template, bool) {
	t.cacheLock.RLock()
	tmpl, ok := t.templates[name]
	t.cacheLock.RUnlock()

	if !ok {
		return nil, false
	}

	return tmpl, ok
}
