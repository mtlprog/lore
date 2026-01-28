package template

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/*.html
var templateFS embed.FS

// Templates holds parsed HTML templates.
type Templates struct {
	pages map[string]*template.Template
}

// New parses and returns all templates.
func New() (*Templates, error) {
	pages := make(map[string]*template.Template)

	// Parse base template first
	base, err := template.ParseFS(templateFS, "templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("parsing base template: %w", err)
	}

	// Page templates to parse with base
	pageNames := []string{"home.html", "account.html"}

	for _, name := range pageNames {
		// Clone base template for each page
		pageTemplate, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning base for %s: %w", name, err)
		}

		// Parse page template into the clone
		_, err = pageTemplate.ParseFS(templateFS, "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}

		pages[name] = pageTemplate
	}

	return &Templates{pages: pages}, nil
}

// Render executes the named template with the given data.
func (t *Templates) Render(w io.Writer, name string, data any) error {
	tmpl, ok := t.pages[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}
