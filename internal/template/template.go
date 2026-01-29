package template

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"strconv"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

// funcMap provides custom template functions.
var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"addFloat": func(a float64, b int) float64 {
		return a + float64(b)
	},
	"truncate": func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n]
	},
	"truncateID": func(s string) string {
		if len(s) <= 15 {
			return s
		}
		return s[:6] + "..." + s[len(s)-6:]
	},
	"slice": func(s string, start int) string {
		if start < 0 || start >= len(s) {
			return ""
		}
		return s[start:]
	},
	"formatNumber": func(f float64) string {
		rounded := int64(math.Round(f))
		str := strconv.FormatInt(rounded, 10)
		var result strings.Builder
		for i, c := range str {
			if i > 0 && (len(str)-i)%3 == 0 {
				result.WriteRune(' ')
			}
			result.WriteRune(c)
		}
		return result.String()
	},
	"votePower": func(totalVotes float64) int {
		if totalVotes <= 0 {
			return 0
		}
		if totalVotes <= 10 {
			return 1
		}
		// Logarithmic scaling: 11-100 → 2, 101-1000 → 3, etc.
		return int(math.Ceil(math.Log10(totalVotes)))
	},
	"trustBarWidth": func(percent int) string {
		return fmt.Sprintf("%d%%", percent)
	},
	"relationArrow": func(direction string) string {
		if direction == "outgoing" {
			return "→"
		}
		return "←"
	},
}

// Templates holds parsed HTML templates.
type Templates struct {
	pages map[string]*template.Template
}

// New parses and returns all templates.
func New() (*Templates, error) {
	pages := make(map[string]*template.Template)

	// Parse base template first with functions
	base, err := template.New("base.html").Funcs(funcMap).ParseFS(templateFS, "templates/base.html")
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
