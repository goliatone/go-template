# go-template

A Go template engine wrapper around pongo2 that provides simplified configuration and dynamic runtime updates.

## Features

- Template rendering with automatic struct-to-JSON conversion
- Global data context shared across all templates
- Dynamic filter registration at runtime
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

All methods are safe for concurrent use. Template caching uses read-write mutex for optimal performance.

## Template Syntax

Uses pongo2 syntax. See [pongo2 documentation](https://github.com/flosch/pongo2) for complete syntax reference.
