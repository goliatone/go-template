# go-template

A Go template engine wrapper around pongo2 that provides simplified configuration and dynamic runtime updates.

## Features

- Template rendering with automatic struct-to-JSON conversion
- Global data context shared across all templates
- Dynamic filter registration at runtime
- Composable pre/post hook system with priority scheduling
- Pluggable hook helpers via `templatehooks` (timestamps, headers, validation, …)
- File system and embedded FS support
- Template caching with concurrent access safety
- Built-in filters: `trim`, `lowerfirst`

## Installation

```bash
go get github.com/goliatone/go-template
```

## Usage

### Basic Setup

```go
import "github.com/goliatone/go-template"

// Initialize with local file system
renderer, err := template.NewRenderer(template.WithBaseDir("./templates"))
if err != nil {
    log.Fatal(err)
}

// Unified Render method - auto-detects template files vs template strings
result, err := renderer.Render("hello", map[string]any{
    "name": "World",
}) // Renders from file: hello.tpl

result, err = renderer.Render("Hello, {{ name }}!", map[string]any{
    "name": "World",
}) // Renders template string directly
```

### Explicit Methods

```go
// Render from file explicitly
result, err := renderer.RenderTemplate("hello", map[string]any{
    "name": "World",
})

// Render from string explicitly
result, err = renderer.RenderString("Hello, {{ name }}!", map[string]any{
    "name": "World",
})
```

### Configuration Options

```go
renderer, err := template.NewRenderer(
    template.WithBaseDir("./templates"),           // Local file system
    template.WithFS(embedFS),                     // Embedded FS
    template.WithExtension(".html"),              // Template extension (default: .tpl)
    template.WithGlobalData(map[string]any{       // Global template data
        "app_name": "MyApp",
        "version":  "1.0.0",
    }),
    template.WithTemplateFunc(map[string]any{     // Custom template functions
        "myFilter": myFilterFunc,
    }),
)
```

### Hooks Overview

The engine exposes **pre** and **post** hooks that run before data is rendered and after output is produced. Hooks work on a shared `HookContext`, so they can adjust data, metadata, template names/content, or final output.

```go
renderer.RegisterPreHook(func(ctx *template.HookContext) error {
    // Stamp metadata other hooks or templates can reuse
    if ctx.Metadata == nil {
        ctx.Metadata = make(map[string]any)
    }
    ctx.Metadata["started_at"] = time.Now()
    return nil
})

renderer.RegisterPostHook(func(ctx *template.HookContext) (string, error) {
    // append diagnostic metadata to every rendered file
    return fmt.Sprintf("%s\n// rendered at %s", ctx.Output, time.Now().Format(time.RFC3339)), nil
})
```

#### Hook Priorities

`HookManager` keeps pre/post hooks in priority buckets. The smallest priority runs first. A `HookManager` is handy when you want to assemble hooks elsewhere and apply them in priority order later.

```go
manager := template.NewHooksManager()

var executionOrder []string

manager.AddPreHook(func(ctx *template.HookContext) error {
    executionOrder = append(executionOrder, "defaults")
    return nil
}, -10) // run before priority 0 hooks

manager.AddPreHook(func(ctx *template.HookContext) error {
    executionOrder = append(executionOrder, "validation")
    return nil
}) // defaults to priority 0

for _, hook := range manager.PreHooks() {
    renderer.RegisterPreHook(hook)
}
```

#### Hook Chains

Use `template.NewHookChain` to compose multiple hooks into a single unit you can register once. Chains are useful when you want to bundle reusable behaviors together.

```go
chain := template.NewHookChain(
    template.WithPreHooksChain(
        ensureDefaultsHook,
        validateInputHook,
    ),
    template.WithPostHooksChain(
        addGeneratedHeaderHook,
        stripTrailingWhitespaceHook,
    ),
)

renderer.RegisterPreHook(chain.AsPreHook())
renderer.RegisterPostHook(chain.AsPostHook())
```

`ExecutePostHooks` automatically propagates the last hook output, so empty chains fall back to the renderer output without modifications.

### Common Hook Helpers

For batteries-included behavior, import `github.com/goliatone/go-template/templatehooks`. It ships configurable helpers for timestamps, copyright/license headers, generated warnings, validation, defaults, and more.

```go
import "github.com/goliatone/go-template/templatehooks"

hooks := templatehooks.NewCommonHooks()

renderer.RegisterPostHook(hooks.AddTimestampHook(
    templatehooks.WithTimestampCommentPrefix("# "),
    templatehooks.WithTimestampFormat(time.RFC822),
    templatehooks.WithTimestampCondition(func(ctx *template.HookContext) bool {
        return strings.HasSuffix(ctx.TemplateName, ".yaml")
    }),
))

renderer.RegisterPostHook(hooks.AddLicenseHook(licenseText,
    templatehooks.WithLicenseCommentStyle(templatehooks.CommentBlockStyle{
        Start:      "/*",
        LinePrefix: " * ",
        End:        " */",
    }),
))

renderer.RegisterPreHook(hooks.ValidateDataHook([]string{"name", "version"}))
renderer.RegisterPreHook(hooks.SetDefaultsHook(map[string]any{"version": "0.0.1"}))
```

Each helper accepts functional options so you can adjust comment prefixes, timestamp layouts, execution conditions, or metadata without writing new hooks from scratch.

### Global Data Management

```go
// Set global data after initialization
err := renderer.GlobalContext(map[string]any{
    "user": User{Name: "John", Role: "admin"},
    "config": appConfig,
})

// Global data is available in all templates
// Local render data takes precedence over global data
result, err := renderer.RenderTemplate("dashboard", localData)
```

### Custom Filters

```go
// Register filters at runtime
err := renderer.RegisterFilter("currency", func(input any, param any) (any, error) {
    amount, ok := input.(float64)
    if !ok {
        return input, nil
    }
    return fmt.Sprintf("$%.2f", amount), nil
})

// Use in templates: {{ price|currency }}
```

### Template Example

```html
<!-- templates/user.tpl -->
<h1>{{ app_name }}</h1>
<p>Hello, {{ user.name|lowerfirst }}!</p>
<p>Total: {{ total|currency }}</p>
<p>Note: {{ message|trim }}</p>
```

### Data Conversion

The engine automatically converts structs to JSON-compatible maps:

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// This struct becomes accessible as {{ user.name }} and {{ user.age }} in templates
renderer.RenderTemplate("profile", map[string]any{
    "user": User{Name: "Alice", Age: 30},
})
```

### Multiple Writers

```go
var buf1, buf2 bytes.Buffer
result, err := renderer.RenderTemplate("template", data, &buf1, &buf2)
// Result written to both buffers and returned as string

// Works with RenderString too
result, err = renderer.RenderString("{{ message }}", data, &buf1, &buf2)
```

### Template Auto-Detection

The `Render` method automatically detects whether you're passing a filename or template content:

```go
// Filename detection (no {{ or {% syntax) - loads from file system
result, err := renderer.Render("user-profile", userData)
result, err = renderer.Render("emails/welcome.html", userData)

// Template content detection (contains {{ or {% syntax) - renders directly
result, err = renderer.Render("Hello, {{ name }}!", userData)
result, err = renderer.Render("{% if user %}Welcome {{ user.name }}!{% endif %}", userData)

// Mixed content with template syntax - treated as template content
result, err = renderer.Render("Loading config.json: {{ status }}", userData)
```

### Template Caching Behavior

```go
// File templates are cached after first load
renderer.RenderTemplate("cached-template", data)  // Loaded and cached
renderer.RenderTemplate("cached-template", data)  // Uses cache

// String templates are parsed each time (no caching)
renderer.RenderString(templateContent, data)      // Parsed fresh each time
```

### Error Handling

```go
result, err := renderer.RenderTemplate("template", data)
if err != nil {
    // Handle template not found, parsing errors, execution errors
    log.Printf("Template error: %v", err)
}
```

## Built-in Filters

- `trim`: Remove leading/trailing whitespace
- `lowerfirst`: Lowercase first non-whitespace character

## Thread Safety

All methods are safe for concurrent use. Template caching uses read write mutex for optimal performance.

## Template Syntax

Uses pongo2 syntax. See [pongo2 documentation](https://github.com/flosch/pongo2) for complete syntax reference.

### Outputting Go Template Syntax

When you need to output Go template syntax (like `{{.PackageName}}`) in your rendered templates, use the `verbatim` tag to prevent pongo2 from trying to parse it:

```html
<!-- Output literal Go template syntax -->
name: {% verbatim %}{{.PackageName}}{% endverbatim %}
config: {% verbatim %}{{.Config.Value}}{% endverbatim %}

<!-- For larger blocks -->
{% verbatim %}
package {{.PackageName}}

type Config struct {
    Name  string `json:"{{.FieldName}}"`
    Value string `json:"{{.FieldValue}}"`
}
{% endverbatim %}
```

This is particularly useful when generating Go code, configuration files, or other templates that use `{{` and `}}` syntax.
