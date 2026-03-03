package email

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

// Engine loads and renders HTML email templates.
type Engine struct {
	tmpl *template.Template
}

// NewEngine loads all *.html files from dir into a single template set.
func NewEngine(dir string) (*Engine, error) {
	tmpl, err := template.ParseGlob(dir + "/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse email templates: %w", err)
	}
	return &Engine{tmpl: tmpl}, nil
}

// Render executes the named template with data and returns HTML + plain text.
func (e *Engine) Render(name string, data any) (html, text string, err error) {
	var buf bytes.Buffer
	if err = e.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", "", fmt.Errorf("render template %q: %w", name, err)
	}
	html = buf.String()
	text = stripHTML(html)
	return html, text, nil
}

var (
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	spaceRe = regexp.MustCompile(`\s{2,}`)
)

func stripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// ---- Template data structs ----

type WelcomeData struct {
	DisplayName string
	AppName     string
	AppURL      string
}

type VerifyEmailData struct {
	DisplayName string
	VerifyURL   string
	AppName     string
	ExpiresIn   string
}

type PasswordResetData struct {
	DisplayName string
	ResetURL    string
	AppName     string
	ExpiresIn   string
}

type TwoFABackupData struct {
	DisplayName string
	AppName     string
	Codes       []string
}

type SecurityAlertData struct {
	DisplayName string
	AppName     string
	IP          string
	Device      string
	Time        string
}

type AccountDeletionData struct {
	DisplayName string
	AppName     string
}
