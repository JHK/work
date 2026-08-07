package config

import (
	"strings"
	"text/template"
)

// tmpl is one [text/template] kept alongside the text it was written as, so that
// a refusal can quote it. Every setting that renders is parsed here, so
// missingkey=error, which is what refuses a value the key does not have, is
// settled for all of them at once.
type tmpl struct {
	t    *template.Template
	text string
}

func parseTmpl(name, text string) (tmpl, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(text)
	if err != nil {
		return tmpl{}, err
	}
	return tmpl{t: t, text: text}, nil
}

func (t tmpl) execute(data map[string]any) (string, error) {
	var b strings.Builder
	err := t.t.Execute(&b, data)
	return b.String(), err
}
