package template

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"github.com/samber/lo"
)

//go:embed templates/*.html
var templateFS embed.FS

// truncateID truncates a Stellar account ID for display.
func truncateID(s string) string {
	if len(s) <= 15 {
		return s
	}
	return s[:6] + "..." + s[len(s)-6:]
}

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
	"truncateID": truncateID,
	"slice": func(s string, indices ...int) string {
		if len(s) == 0 {
			return ""
		}
		start := 0
		end := len(s)
		if len(indices) >= 1 {
			start = indices[0]
		}
		if len(indices) >= 2 {
			end = indices[1]
		}
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start >= end || start >= len(s) {
			return ""
		}
		return s[start:end]
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
	"isStellarID": func(s string) bool {
		return len(s) == 56 && (s[0] == 'G' || s[0] == 'M')
	},
	"accountDisplay": func(accountID string, names map[string]string) string {
		if names == nil {
			return truncateID(accountID)
		}
		if name, ok := names[accountID]; ok && name != "" {
			return name
		}
		return truncateID(accountID)
	},
	"containsTag": func(tags []string, tag string) bool {
		return lo.Contains(tags, tag)
	},
	"lower": strings.ToLower,
	"multiplyStrings": func(a, b string) string {
		af, _ := strconv.ParseFloat(a, 64)
		bf, _ := strconv.ParseFloat(b, 64)
		return strconv.FormatFloat(af*bf, 'f', 7, 64)
	},
	"tagURL": func(currentTags []string, tag string, add bool, currentQuery string) string {
		var newTags []string
		if add {
			if !lo.Contains(currentTags, tag) {
				newTags = append(currentTags, tag)
			} else {
				newTags = currentTags
			}
		} else {
			newTags = lo.Filter(currentTags, func(t string, _ int) bool {
				return t != tag
			})
		}

		params := url.Values{}
		if currentQuery != "" {
			params.Set("q", currentQuery)
		}
		for _, t := range newTags {
			params.Add("tag", t)
		}

		if len(params) == 0 {
			return "/search"
		}
		return "/search?" + params.Encode()
	},
	"markdown": func(s string) template.HTML {
		// Use GitHub Flavored Markdown extensions
		extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs | blackfriday.Autolink
		renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
			Flags: blackfriday.CommonHTMLFlags,
		})
		unsafe := blackfriday.Run([]byte(s), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
		// Sanitize HTML to prevent XSS attacks from untrusted blockchain data
		p := bluemonday.UGCPolicy()
		safe := p.SanitizeBytes(unsafe)
		return template.HTML(safe)
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
	pageNames := []string{"home.html", "account.html", "transaction.html", "search.html", "token.html", "reputation.html"}

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
