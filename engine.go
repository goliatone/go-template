package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/flosch/pongo2/v6"
)

func init() {
	pongo2.RegisterFilter("trim", filterTrim)
	pongo2.RegisterFilter("lowerfirst", filterLowerFirst)
}

type Renderer interface {
	// Render finds a template by `name` and executes if with
	// the given `data`.
	// The output is written to `out` and also returned as a string for
	// conveniene.
	Render(name string, data any, out ...io.Writer) (string, error)
}

type Engine struct {
	mu          sync.RWMutex
	templateSet *pongo2.TemplateSet
	templates   map[string]*pongo2.Template
	tplExt      string
	fs          fs.FS
	baseDir     string
	funcMap     map[string]any
}

type Option func(*Engine)

func WithFS(fs fs.FS) Option {
	return func(e *Engine) {
		e.fs = fs
	}
}

func WithBaseDir(dir string) Option {
	return func(e *Engine) {
		e.baseDir = dir
	}
}

func WithTemplateFunc(funcs map[string]any) Option {
	return func(e *Engine) {
		maps.Copy(e.funcMap, funcs)
	}
}

func WithExtension(ext string) Option {
	return func(e *Engine) {
		if !strings.HasPrefix(ext, ".") {
			e.tplExt = "." + ext
		} else {
			e.tplExt = ext
		}
	}
}

func NewRenderer(opts ...Option) (*Engine, error) {
	e := &Engine{
		templates: make(map[string]*pongo2.Template),
		tplExt:    ".tpl",
		funcMap:   defaultFuncMaps(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, e.Load()
}

func (r *Engine) Load() error {
	if r.baseDir == "" && r.fs == nil {
		return fmt.Errorf("need to provide either baseDir or fs.FS")
	}

	var err error
	var loader pongo2.TemplateLoader
	var loaders []pongo2.TemplateLoader

	if r.baseDir != "" {
		if loader, err = pongo2.NewLocalFileSystemLoader(r.baseDir); err != nil {
			return fmt.Errorf("failed to create loader: %w", err)
		}
		loaders = append(loaders, loader)
	}

	if r.fs != nil {
		if loader = pongo2.NewFSLoader(r.fs); err != nil {
			return fmt.Errorf("failed to create loader: %w", err)
		}
		loaders = append(loaders, loader)
	}

	ts := pongo2.NewSet("default", loaders...)

	ts.Globals.Update(r.funcMap)

	// for n, fn := range r.funcMap {
	// 	if !pongo2.FilterExists(n) {
	// 		pongo2.RegisterFilter(n)
	// 	}
	// }

	r.templateSet = ts

	return nil
}

// Render finds a template by `name` and executes it with the given `data`.
// The output is written to any provided `io.Writer`s and is also returned as a string.
//
// If the provided `data` is not a map[string]any, it will be converted to one
// by marshaling it to JSON and then unmarshaling. Be aware of the performance
// implications and that this respects `json` struct tags.
func (r *Engine) Render(name string, data any, out ...io.Writer) (string, error) {
	templatePath := name
	if !strings.HasSuffix(templatePath, r.tplExt) {
		templatePath += r.tplExt
	}

	tmpl, err := r.getTemplate(templatePath)
	if err != nil {
		return "", err
	}

	viewContext := make(pongo2.Context)
	switch val := data.(type) {
	case map[string]any:
		maps.Copy(viewContext, val)
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		m := map[string]any{}
		err = json.Unmarshal(b, &m)
		if err != nil {
			return "", err
		}
		maps.Copy(viewContext, m)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteWriter(viewContext, &buf); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	renderedStr := buf.String()

	if len(out) > 0 {
		for _, w := range out {
			if _, err := w.Write([]byte(renderedStr)); err != nil {
				return "", err
			}
		}
	}
	return renderedStr, nil
}

func (r *Engine) getTemplate(path string) (*pongo2.Template, error) {
	r.mu.RLock()
	if tmpl, ok := r.templates[path]; ok {
		r.mu.RUnlock()
		return tmpl, nil
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()

	if tmpl, ok := r.templates[path]; ok {
		return tmpl, nil
	}

	compiled, err := r.templateSet.FromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", path, err)
	}
	r.templates[path] = compiled
	return compiled, nil
}

func defaultFuncMaps() map[string]any {
	out := map[string]any{}
	// out["lowerfirst"] = filterLowerFirst
	return out
}

func filterLowerFirst(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	r, size := utf8.DecodeRuneInString(t)
	return pongo2.AsValue(strings.ToLower(string(r)) + t[size:]), nil
}

func filterTrim(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	return pongo2.AsValue(strings.TrimSpace(t)), nil
}
