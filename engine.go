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
	globalData  map[string]any
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

func WithGlobalData(data map[string]any) Option {
	return func(e *Engine) {
		maps.Copy(e.globalData, data)
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
		templates:  make(map[string]*pongo2.Template),
		tplExt:     ".tpl",
		funcMap:    defaultFuncMaps(),
		globalData: make(map[string]any),
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

	// we have to set the template set first
	r.templateSet = ts

	// then we apply global data
	if err := r.GlobalContext(r.globalData); err != nil {
		return fmt.Errorf("failed to convert global data to context: %w", err)
	}

	for n, fn := range r.funcMap {
		if !pongo2.FilterExists(n) {
			if pfn, ok := fn.(func(*pongo2.Value, *pongo2.Value) (*pongo2.Value, *pongo2.Error)); ok {
				pongo2.RegisterFilter(n, pfn)
			}
		}
	}

	return nil
}

func (r *Engine) GlobalContext(data any) error {
	globalContext, err := convertToContext(data)
	if err != nil {
		return fmt.Errorf("failed to convert global data to context: %w", err)
	}

	// store the global data for later use
	maps.Copy(r.globalData, globalContext)

	// if templateSet is available we update it immediately
	if r.templateSet != nil {
		r.templateSet.Globals.Update(globalContext)
	}

	return nil
}

func (r *Engine) RegisterFilter(name string, fn func(input any, param any) (any, error)) error {
	pongo2Filter := func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		var inputVal any = in.Interface()
		var paramVal any
		if param != nil {
			paramVal = param.Interface()
		}

		result, err := fn(inputVal, paramVal)
		if err != nil {
			return nil, &pongo2.Error{Sender: "custom_filter", OrigError: err}
		}
		return pongo2.AsValue(result), nil
	}

	if !pongo2.FilterExists(name) {
		pongo2.RegisterFilter(name, pongo2Filter)
		return nil
	}

	return fmt.Errorf("filter %s already exists", name)
}

// RenderString renders a template from a string content with the given `data`.
// The output is written to any provided `io.Writer`s and is also returned as a string.
//
// Unlike RenderTemplate, this method takes template content directly instead of a filename.
// The template is parsed each time (no caching) but benefits from global data and filters.
//
// If the provided `data` is not a map[string]any, it will be converted to one
// by marshaling it to JSON and then unmarshaling. Be aware of the performance
// implications and that this respects `json` struct tags.
func (r *Engine) RenderString(templateContent string, data any, out ...io.Writer) (string, error) {
	// Create template from string content
	tmpl, err := r.templateSet.FromString(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template string: %w", err)
	}

	viewContext, err := convertToContext(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert data to context: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteWriter(viewContext, &buf); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	renderedStr := buf.String()

	// Write to provided writers
	if len(out) > 0 {
		for _, w := range out {
			if _, err := w.Write([]byte(renderedStr)); err != nil {
				return "", err
			}
		}
	}

	return renderedStr, nil
}

// Render intelligently renders either a template file or template string content.
// It auto-detects whether the `name` parameter contains template syntax ({{ or {%)
// and calls either RenderString for template content or RenderTemplate for filenames.
//
// Template detection logic:
// - If `name` contains "{{" or "{%" it's treated as template content (calls RenderString)
// - Otherwise it's treated as a template filename (calls RenderTemplate)
//
// This method provides backward compatibility while enabling both use cases with a single API.
func (r *Engine) Render(name string, data any, out ...io.Writer) (string, error) {
	// detect if this is template content or a filename
	if isTemplateContent(name) {
		return r.RenderString(name, data, out...)
	}
	return r.RenderTemplate(name, data, out...)
}

// isTemplateContent detects if a string contains template syntax
func isTemplateContent(s string) bool {
	return strings.Contains(s, "{{") || strings.Contains(s, "{%")
}

// RenderTemplate finds a template by `name` and executes it with the given `data`.
// The output is written to any provided `io.Writer`s and is also returned as a string.
//
// If the provided `data` is not a map[string]any, it will be converted to one
// by marshaling it to JSON and then unmarshaling. Be aware of the performance
// implications and that this respects `json` struct tags.
func (r *Engine) RenderTemplate(name string, data any, out ...io.Writer) (string, error) {
	templatePath := name
	if !strings.HasSuffix(templatePath, r.tplExt) {
		templatePath += r.tplExt
	}

	tmpl, err := r.getTemplate(templatePath)
	if err != nil {
		return "", err
	}

	viewContext, err := convertToContext(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert data to context: %w", err)
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
	out["trim"] = filterTrim
	out["lowerfirst"] = filterLowerFirst
	return out
}

// convertToContext converts any data to a pongo2.Context map.
// It always uses JSON marshaling/unmarshaling to ensure consistent behavior
// and proper handling of structs with json tags.
func convertToContext(data any) (pongo2.Context, error) {
	viewContext := make(pongo2.Context)
	switch data.(type) {
	case nil:
		// Return empty context for nil data
		return viewContext, nil
	default:
		// Always use JSON conversion to handle structs properly
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		m := map[string]any{}
		err = json.Unmarshal(b, &m)
		if err != nil {
			return nil, err
		}
		maps.Copy(viewContext, m)
	}
	return viewContext, nil
}

func filterLowerFirst(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()

	// find the first non whitespace character
	var firstNonWhitespaceIndex int
	var firstRune rune
	var firstRuneSize int

	for i, r := range t {
		if !strings.ContainsRune(" \t\n\r", r) {
			firstNonWhitespaceIndex = i
			firstRune = r
			firstRuneSize = utf8.RuneLen(r)
			break
		}
	}

	// not whitespace character, return original string
	if firstRune == 0 {
		return pongo2.AsValue(t), nil
	}

	// build result = prefix + lowercased first letter + rest
	prefix := t[:firstNonWhitespaceIndex]
	loweredRune := strings.ToLower(string(firstRune))
	rest := t[firstNonWhitespaceIndex+firstRuneSize:]

	return pongo2.AsValue(prefix + loweredRune + rest), nil
}

func filterTrim(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() <= 0 {
		return pongo2.AsValue(""), nil
	}
	t := in.String()
	return pongo2.AsValue(strings.TrimSpace(t)), nil
}
