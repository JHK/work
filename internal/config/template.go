package config

import (
	"fmt"
	"strings"
	"text/template"
)

// tmpl is one [text/template] kept alongside the text it was written as, so that
// a refusal can quote it. Every setting that renders is parsed here, under
// missingkey=error and over the functions its key offers.
type tmpl struct {
	t    *template.Template
	text string
}

func parseTmpl(name, text string, funcs template.FuncMap) (tmpl, error) {
	t, err := template.New(name).Option("missingkey=error").Funcs(funcs).Parse(text)
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

// render is what the template comes to, empty where it cannot render.
func (t tmpl) render(data map[string]any) string {
	out, err := t.execute(data)
	if err != nil {
		return ""
	}
	return out
}

// must parses a compiled-in default, which cannot be at fault.
func must[T any](kind, text string, parse func(string) (T, error), check func(*T) error) T {
	v, err := parse(text)
	if err == nil {
		err = check(&v)
	}
	if err != nil {
		panic(fmt.Sprintf("config: default %s %q %v", kind, text, err))
	}
	return v
}
