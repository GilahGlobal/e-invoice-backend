package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed emails
var emailFS embed.FS

var emailTemplates *template.Template

func init() {
	funcMap := template.FuncMap{
		"currentYear": func() int {
			return time.Now().UTC().Year()
		},
		"formatAmount": func(amount float64) string {
			return fmt.Sprintf("NGN %.2f", amount)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"toInt": func(v any) int {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			default:
				return 0
			}
		},
		"lte": func(a, b any) bool {
			toInt := func(v any) int {
				switch n := v.(type) {
				case int:
					return n
				case int64:
					return int(n)
				case float64:
					return int(n)
				default:
					return 0
				}
			}
			return toInt(a) <= toInt(b)
		},
		"gte": func(a, b any) bool {
			toInt := func(v any) int {
				switch n := v.(type) {
				case int:
					return n
				case int64:
					return int(n)
				case float64:
					return int(n)
				default:
					return 0
				}
			}
			return toInt(a) >= toInt(b)
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call: odd number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings, got %T", values[i])
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
	}

	var err error
	emailTemplates, err = template.New("").Funcs(funcMap).ParseFS(emailFS, "emails/*.html", "emails/components/*.html")
	if err != nil {
		panic(fmt.Sprintf("failed to parse email templates: %v", err))
	}
}

// RenderEmail renders the named email template with the provided data.
func RenderEmail(templateName string, data any) (string, error) {
	var buf bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to execute email template %q: %w", templateName, err)
	}
	return buf.String(), nil
}
