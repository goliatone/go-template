package template_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/goliatone/go-template"
	"github.com/stretchr/testify/require"
)

func TestEngine_Render_MapData(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	out := &bytes.Buffer{}
	data := map[string]any{
		"name":  "Alice",
		"count": 3,
	}

	result, err := renderer.Render("hello", data, out)
	require.NoError(t, err, "should render template without error")

	expected := "Hello, Alice! You have 3 items.\n"
	require.Equal(t, expected, result)
	require.Equal(t, expected, out.String())
}

func TestEngine_Render_StructData(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	type Person struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	p := Person{
		Name:  "Bob",
		Count: 5,
	}

	out := &bytes.Buffer{}
	result, err := renderer.Render("hello", p, out)
	require.NoError(t, err)

	expected := "Hello, Bob! You have 5 items.\n"
	require.Equal(t, expected, result)
	require.Equal(t, expected, out.String())
}

func TestEngine_TemplateCaching(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	out1 := &bytes.Buffer{}
	_, err = renderer.Render("hello", map[string]any{
		"name":  "Eve",
		"count": 1,
	}, out1)
	require.NoError(t, err)

	out2 := &bytes.Buffer{}
	_, err = renderer.Render("hello", map[string]any{
		"name":  "Eve",
		"count": 2,
	}, out2)
	require.NoError(t, err)

	require.Contains(t, out2.String(), "You have 2 items")
}

func TestEngine_Render_MultipleWriters(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	outA := &bytes.Buffer{}
	outB := &bytes.Buffer{}
	_, err = renderer.Render("hello", map[string]any{
		"name":  "Charlie",
		"count": 2,
	}, outA, outB)
	require.NoError(t, err)

	expected := "Hello, Charlie! You have 2 items.\n"
	require.Equal(t, expected, outA.String(), "outA should get the full render")
	require.Equal(t, expected, outB.String(), "outB should get the full render")
}

func TestEngine_Render_FileNotFound(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	_, err = renderer.Render("does-not-exist", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load template does-not-exist.tpl")
}

// createTempTemplates creates a temporary directory with a "hello.tpl" file
// and returns the directory path plus a cleanup function.
func createTempTemplates(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "test-templates-")
	require.NoError(t, err)

	// Create a sample template file
	content := `Hello, {{ name }}! You have {{ count|integer }} items.
`
	err = os.WriteFile(filepath.Join(dir, "hello.tpl"), []byte(content), fs.ModePerm)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestDefaultFilters_Trim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim leading and trailing spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "trim tabs and newlines",
			input:    "\t\nhello\n\t",
			expected: "hello",
		},
		{
			name:     "trim mixed whitespace",
			input:    " \t\n hello world \n\t ",
			expected: "hello world",
		},
		{
			name:     "no whitespace to trim",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := createFilterTemplates(t, "{{ text|trim }}")
			defer cleanup()

			renderer, err := template.NewRenderer(template.WithBaseDir(dir))
			require.NoError(t, err)

			result, err := renderer.Render("filter", map[string]any{
				"text": tc.input,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultFilters_LowerFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase first letter of word",
			input:    "Hello",
			expected: "hello",
		},
		{
			name:     "already lowercase first letter",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "single uppercase letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: "a",
		},
		{
			name:     "mixed case sentence",
			input:    "Hello World",
			expected: "hello World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "unicode character",
			input:    "Ñice",
			expected: "ñice",
		},
		{
			name:     "number first",
			input:    "123abc",
			expected: "123abc",
		},
		{
			name:     "special character first",
			input:    "@Hello",
			expected: "@Hello",
		},
		{
			name:     "leading spaces before letter",
			input:    "  Hello",
			expected: "  hello",
		},
		{
			name:     "leading tabs and newlines before letter",
			input:    "\t\nHello",
			expected: "\t\nhello",
		},
		{
			name:     "mixed leading whitespace",
			input:    " \t\n Hello World",
			expected: " \t\n hello World",
		},
		{
			name:     "only whitespace characters",
			input:    "   \t\n   ",
			expected: "   \t\n   ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := createFilterTemplates(t, "{{ text|lowerfirst }}")
			defer cleanup()

			renderer, err := template.NewRenderer(template.WithBaseDir(dir))
			require.NoError(t, err)

			result, err := renderer.Render("filter", map[string]any{
				"text": tc.input,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultFilters_ChainedFilters(t *testing.T) {
	tests := []struct {
		name     string
		template string
		input    string
		expected string
	}{
		{
			name:     "trim then lowerfirst",
			template: "{{ text|trim|lowerfirst }}",
			input:    "  Hello World  ",
			expected: "hello World",
		},
		{
			name:     "lowerfirst then trim",
			template: "{{ text|lowerfirst|trim }}",
			input:    "  Hello World  ",
			expected: "hello World",
		},
		{
			name:     "multiple filters with empty result",
			template: "{{ text|trim|lowerfirst }}",
			input:    "   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := createFilterTemplates(t, tc.template)
			defer cleanup()

			renderer, err := template.NewRenderer(template.WithBaseDir(dir))
			require.NoError(t, err)

			result, err := renderer.Render("filter", map[string]any{
				"text": tc.input,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

// createFilterTemplates creates a temporary directory with a template file
// containing the specified template content for testing filters
func createFilterTemplates(t *testing.T, templateContent string) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "test-filter-templates-")
	require.NoError(t, err)

	// Create a filter test template file
	err = os.WriteFile(filepath.Join(dir, "filter.tpl"), []byte(templateContent), fs.ModePerm)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestEngine_GlobalData(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create a template that uses global data
	globalTemplate := `Global: {{ global_var }}, Local: {{ name }}`
	err := os.WriteFile(filepath.Join(dir, "global.tpl"), []byte(globalTemplate), fs.ModePerm)
	require.NoError(t, err)

	globalData := map[string]any{
		"global_var": "global_value",
		"app_name":   "MyApp",
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(globalData),
	)
	require.NoError(t, err)

	localData := map[string]any{
		"name": "Alice",
	}

	result, err := renderer.Render("global", localData)
	require.NoError(t, err)
	require.Equal(t, "Global: global_value, Local: Alice", result)
}

func TestEngine_GlobalData_Override(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create a template that tests local data overriding global data
	overrideTemplate := `Value: {{ shared_key }}`
	err := os.WriteFile(filepath.Join(dir, "override.tpl"), []byte(overrideTemplate), fs.ModePerm)
	require.NoError(t, err)

	globalData := map[string]any{
		"shared_key": "global_value",
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(globalData),
	)
	require.NoError(t, err)

	// Local data should override global data
	localData := map[string]any{
		"shared_key": "local_value",
	}

	result, err := renderer.Render("override", localData)
	require.NoError(t, err)
	require.Equal(t, "Value: local_value", result)
}

func TestEngine_GlobalData_EmptyLocal(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create a template that only uses global data
	globalOnlyTemplate := `App: {{ app_name }}, Version: {{ version }}`
	err := os.WriteFile(filepath.Join(dir, "global_only.tpl"), []byte(globalOnlyTemplate), fs.ModePerm)
	require.NoError(t, err)

	globalData := map[string]any{
		"app_name": "MyApp",
		"version":  "1.0.0",
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(globalData),
	)
	require.NoError(t, err)

	// Render with no local data
	result, err := renderer.Render("global_only", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "App: MyApp, Version: 1.0.0", result)
}

func TestEngine_WithoutGlobalData(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create a template that tries to use non-existent global data
	missingTemplate := `Global: {{ missing_var|default:"not_found" }}`
	err := os.WriteFile(filepath.Join(dir, "missing.tpl"), []byte(missingTemplate), fs.ModePerm)
	require.NoError(t, err)

	// Create renderer without global data
	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	result, err := renderer.Render("missing", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "Global: not_found", result)
}

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestConvertToContext(t *testing.T) {
	// Test our conversion logic directly
	testData := map[string]any{
		"user": User{Name: "TestUser", Age: 25},
	}

	// Manual conversion to see what should happen
	b, err := json.Marshal(testData)
	require.NoError(t, err)
	t.Logf("JSON: %s", string(b))

	var converted map[string]any
	err = json.Unmarshal(b, &converted)
	require.NoError(t, err)
	t.Logf("Manual conversion: %+v", converted)

	// This should show us the actual structure we expect
	if userMap, ok := converted["user"].(map[string]any); ok {
		t.Logf("User name from manual conversion: %v", userMap["name"])
	}
}

func TestEngine_GlobalData_DebugStruct(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Simple debug template
	debugTemplate := `{{ user.name }}`
	err := os.WriteFile(filepath.Join(dir, "debug.tpl"), []byte(debugTemplate), fs.ModePerm)
	require.NoError(t, err)

	// Test with struct - this should work with the JSON conversion
	globalData := map[string]any{
		"user": User{Name: "StructUser", Age: 30},
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(globalData),
	)
	require.NoError(t, err)

	result, err := renderer.Render("debug", map[string]any{})
	require.NoError(t, err)
	t.Logf("Debug result with struct: %q", result)
	require.Equal(t, "StructUser", result)
}

func TestEngine_GlobalData_ComplexTypes(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create a template that uses complex global data types
	complexTemplate := `User: {{ user.name }}, Items: {% for item in items %}{{ item }}{% if not forloop.last %}, {% endif %}{% endfor %}`
	err := os.WriteFile(filepath.Join(dir, "complex.tpl"), []byte(complexTemplate), fs.ModePerm)
	require.NoError(t, err)

	globalData := map[string]any{
		"user": User{
			Name: "GlobalUser",
			Age:  101,
		},
		"items": []string{"item1", "item2", "item3"},
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(globalData),
	)
	require.NoError(t, err)

	result, err := renderer.Render("complex", map[string]any{})
	require.NoError(t, err)
	t.Logf("Actual result: %q", result)
	require.Equal(t, "User: GlobalUser, Items: item1, item2, item3, ", result)
}
