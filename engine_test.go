package template_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flosch/pongo2/v6"
	"github.com/goliatone/go-template"
	"github.com/stretchr/testify/require"
)

func TestEngine_RenderTemplate_MapData(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	out := &bytes.Buffer{}
	data := map[string]any{
		"name":  "Alice",
		"count": 3,
	}

	result, err := renderer.RenderTemplate("hello", data, out)
	require.NoError(t, err, "should render template without error")

	expected := "Hello, Alice! You have 3 items.\n"
	require.Equal(t, expected, result)
	require.Equal(t, expected, out.String())
}

func TestEngine_RenderTemplate_StructData(t *testing.T) {
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
	result, err := renderer.RenderTemplate("hello", p, out)
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
	_, err = renderer.RenderTemplate("hello", map[string]any{
		"name":  "Eve",
		"count": 1,
	}, out1)
	require.NoError(t, err)

	out2 := &bytes.Buffer{}
	_, err = renderer.RenderTemplate("hello", map[string]any{
		"name":  "Eve",
		"count": 2,
	}, out2)
	require.NoError(t, err)

	require.Contains(t, out2.String(), "You have 2 items")
}

func TestEngine_RenderTemplate_MultipleWriters(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	outA := &bytes.Buffer{}
	outB := &bytes.Buffer{}
	_, err = renderer.RenderTemplate("hello", map[string]any{
		"name":  "Charlie",
		"count": 2,
	}, outA, outB)
	require.NoError(t, err)

	expected := "Hello, Charlie! You have 2 items.\n"
	require.Equal(t, expected, outA.String(), "outA should get the full render")
	require.Equal(t, expected, outB.String(), "outB should get the full render")
}

func TestEngine_RenderTemplate_FileNotFound(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	_, err = renderer.RenderTemplate("does-not-exist", nil)
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

			result, err := renderer.RenderTemplate("filter", map[string]any{
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

			result, err := renderer.RenderTemplate("filter", map[string]any{
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

			result, err := renderer.RenderTemplate("filter", map[string]any{
				"text": tc.input,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestWithTemplateFuncRegistersPlainHelper(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	content := `{{ is_even(value) }}`
	writeTemplate(t, dir, "helper", content)

	isEven := func(v any) bool {
		switch val := v.(type) {
		case int:
			return val%2 == 0
		case int64:
			return val%2 == 0
		case float64:
			return int(val)%2 == 0
		default:
			return false
		}
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithTemplateFunc(map[string]any{
			"is_even": isEven,
		}),
	)
	require.NoError(t, err)

	result, err := renderer.RenderTemplate("helper", map[string]any{"value": 4})
	require.NoError(t, err)
	require.Equal(t, "true", strings.ToLower(strings.TrimSpace(result)))
}

func TestWithTemplateFuncRegistersFilterHelpers(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	content := `{{ value|times_two }}`
	writeTemplate(t, dir, "filter_helper", content)

	timesTwo := func(in *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		num := in.Interface()
		var f float64
		switch v := num.(type) {
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		case float64:
			f = v
		default:
			return pongo2.AsValue(0), nil
		}
		if f == math.Trunc(f) {
			return pongo2.AsValue(int64(f * 2)), nil
		}
		return pongo2.AsValue(f * 2), nil
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithTemplateFunc(map[string]any{
			"times_two": pongo2.FilterFunction(timesTwo),
		}),
	)
	require.NoError(t, err)

	result, err := renderer.RenderTemplate("filter_helper", map[string]any{"value": 5})
	require.NoError(t, err)
	require.Equal(t, "10", strings.TrimSpace(result))
}

func TestWithTemplateFuncMultipleInvocationsSurviveReload(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	comboContent := `double: {{ value|reload_double }}, positive: {{ is_positive(value) }}`
	writeTemplate(t, dir, "combo", comboContent)

	reloadDouble := func(in *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		num := in.Interface()
		var f float64
		switch v := num.(type) {
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		case float64:
			f = v
		default:
			return pongo2.AsValue(0), nil
		}
		if f == math.Trunc(f) {
			return pongo2.AsValue(int64(f * 2)), nil
		}
		return pongo2.AsValue(f * 2), nil
	}

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithTemplateFunc(map[string]any{
			"reload_double": pongo2.FilterFunction(reloadDouble),
		}),
	)
	require.NoError(t, err)

	isPositive := func(v any) bool {
		switch val := v.(type) {
		case int:
			return val >= 0
		case int64:
			return val >= 0
		case float64:
			return val >= 0
		default:
			return false
		}
	}
	template.WithTemplateFunc(map[string]any{"is_positive": isPositive})(renderer)

	result, err := renderer.RenderTemplate("combo", map[string]any{"value": 3})
	require.NoError(t, err)
	require.Equal(t, "double: 6, positive: true", strings.ToLower(strings.TrimSpace(result)))

	// Add another helper after the engine is live and ensure reload keeps all helpers
	writeTemplate(t, dir, "extended", `double: {{ value|reload_double }}, negative: {{ is_negative(value) }}`)
	isNegative := func(v any) bool {
		switch val := v.(type) {
		case int:
			return val < 0
		case int64:
			return val < 0
		case float64:
			return val < 0
		default:
			return false
		}
	}
	template.WithTemplateFunc(map[string]any{"is_negative": isNegative})(renderer)

	require.NoError(t, renderer.Load())

	negResult, err := renderer.RenderTemplate("extended", map[string]any{"value": -4})
	require.NoError(t, err)
	require.Equal(t, "double: -8, negative: true", strings.ToLower(strings.TrimSpace(negResult)))

	// Ensure the initial template still works post-reload
	comboResult, err := renderer.RenderTemplate("combo", map[string]any{"value": 2})
	require.NoError(t, err)
	require.Equal(t, "double: 4, positive: true", strings.ToLower(strings.TrimSpace(comboResult)))
}

func TestGlobalContextCallableUpdatesTemplateSet(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	writeTemplate(t, dir, "global_helper", `value: {{ call_me() }}`)

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	callMe := func() string { return "global" }
	require.NoError(t, renderer.GlobalContext(map[string]any{"call_me": callMe}))

	result, err := renderer.RenderTemplate("global_helper", nil)
	require.NoError(t, err)
	require.Equal(t, "value: global", strings.TrimSpace(result))

	writeTemplate(t, dir, "global_helper_args", `shout: {{ shout(name) }}`)
	shout := func(v any) string { return strings.ToUpper(fmt.Sprint(v)) }
	require.NoError(t, renderer.GlobalContext(map[string]any{"shout": shout}))

	require.NoError(t, renderer.Load())

	shoutResult, err := renderer.RenderTemplate("global_helper_args", map[string]any{"name": "go"})
	require.NoError(t, err)
	require.Equal(t, "shout: GO", strings.TrimSpace(shoutResult))

	// Original callable should still be available after reload
	result, err = renderer.RenderTemplate("global_helper", nil)
	require.NoError(t, err)
	require.Equal(t, "value: global", strings.TrimSpace(result))
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

func writeTemplate(t *testing.T, dir, name, content string) {
	t.Helper()

	filename := filepath.Join(dir, name+".tpl")
	require.NoError(t, os.WriteFile(filename, []byte(content), fs.ModePerm))
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

	result, err := renderer.RenderTemplate("global", localData)
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

	result, err := renderer.RenderTemplate("override", localData)
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

	// RenderTemplate with no local data
	result, err := renderer.RenderTemplate("global_only", map[string]any{})
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

	result, err := renderer.RenderTemplate("missing", map[string]any{})
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

	result, err := renderer.RenderTemplate("debug", map[string]any{})
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

	result, err := renderer.RenderTemplate("complex", map[string]any{})
	require.NoError(t, err)
	t.Logf("Actual result: %q", result)
	require.Equal(t, "User: GlobalUser, Items: item1, item2, item3, ", result)
}

func TestEngine_GlobalContext(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create template that uses global data
	globalTemplate := `Name: {{ name }}, Version: {{ version }}`
	err := os.WriteFile(filepath.Join(dir, "global.tpl"), []byte(globalTemplate), fs.ModePerm)
	require.NoError(t, err)

	// Initialize engine without global data
	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// First render should show empty values
	result, err := renderer.RenderTemplate("global", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "Name: , Version: ", result)

	// Add global data after initialization
	err = renderer.GlobalContext(map[string]any{
		"name":    "MyApp",
		"version": "1.0.0",
	})
	require.NoError(t, err)

	// Second render should show global data
	result, err = renderer.RenderTemplate("global", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "Name: MyApp, Version: 1.0.0", result)

	// Update global data with new values
	err = renderer.GlobalContext(map[string]any{
		"name":    "UpdatedApp",
		"version": "2.0.0",
		"author":  "Developer",
	})
	require.NoError(t, err)

	// Create template that uses new global data
	extendedTemplate := `{{ name }} v{{ version }} by {{ author }}`
	err = os.WriteFile(filepath.Join(dir, "extended.tpl"), []byte(extendedTemplate), fs.ModePerm)
	require.NoError(t, err)

	result, err = renderer.RenderTemplate("extended", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "UpdatedApp v2.0.0 by Developer", result)
}

func TestEngine_GlobalContext_WithStructs(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create template that uses struct global data
	structTemplate := `User: {{ user.name }} ({{ user.age }})`
	err := os.WriteFile(filepath.Join(dir, "struct.tpl"), []byte(structTemplate), fs.ModePerm)
	require.NoError(t, err)

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Add struct global data after initialization
	err = renderer.GlobalContext(map[string]any{
		"user": User{Name: "John", Age: 30},
	})
	require.NoError(t, err)

	result, err := renderer.RenderTemplate("struct", map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "User: John (30.000000)", result)
}

func TestEngine_RegisterFilter(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Create template that uses custom filters
	filterTemplate := `{{ text|reverse }}, {{ number|double }}`
	err := os.WriteFile(filepath.Join(dir, "filter.tpl"), []byte(filterTemplate), fs.ModePerm)
	require.NoError(t, err)

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Register custom filters after initialization
	err = renderer.RegisterFilter("reverse", func(input any, param any) (any, error) {
		str := fmt.Sprintf("%v", input)
		runes := []rune(str)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	})
	require.NoError(t, err)

	err = renderer.RegisterFilter("double", func(input any, param any) (any, error) {
		if num, ok := input.(float64); ok {
			return num * 2, nil
		}
		if num, ok := input.(int); ok {
			return num * 2, nil
		}
		return input, nil
	})
	require.NoError(t, err)

	// Test the custom filters
	result, err := renderer.RenderTemplate("filter", map[string]any{
		"text":   "hello",
		"number": 5,
	})
	require.NoError(t, err)
	require.Equal(t, "olleh, 10.000000", result)
}

func TestEngine_RegisterFilter_ErrorHandling(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Register a filter that can return an error
	err = renderer.RegisterFilter("validate", func(input any, param any) (any, error) {
		str := fmt.Sprintf("%v", input)
		if len(str) < 3 {
			return nil, fmt.Errorf("input must be at least 3 characters")
		}
		return str, nil
	})
	require.NoError(t, err)

	// Test successful case
	validateTemplate := `{{ text|validate }}`
	err = os.WriteFile(filepath.Join(dir, "validate.tpl"), []byte(validateTemplate), fs.ModePerm)
	require.NoError(t, err)

	result, err := renderer.RenderTemplate("validate", map[string]any{
		"text": "hello",
	})
	require.NoError(t, err)
	require.Equal(t, "hello", result)

	// Test error case
	_, err = renderer.RenderTemplate("validate", map[string]any{
		"text": "hi",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "input must be at least 3 characters")
}

func TestEngine_RegisterFilter_DuplicateFilter(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Register a filter
	err = renderer.RegisterFilter("test", func(input any, param any) (any, error) {
		return input, nil
	})
	require.NoError(t, err)

	// Try to register the same filter again
	err = renderer.RegisterFilter("test", func(input any, param any) (any, error) {
		return input, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "filter test already exists")
}

func TestEngine_DynamicUpdates_IntegrationTest(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	// Initialize engine
	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Step 1: Register custom filters first
	err = renderer.RegisterFilter("uppercase", func(input any, param any) (any, error) {
		return strings.ToUpper(fmt.Sprintf("%v", input)), nil
	})
	require.NoError(t, err)

	err = renderer.RegisterFilter("exclaim", func(input any, param any) (any, error) {
		return fmt.Sprintf("%v!", input), nil
	})
	require.NoError(t, err)

	// Step 2: Create a complex template using both global data and custom filters
	complexTemplate := `{{ app_name }} v{{ version }}: {{ message|uppercase|exclaim }}`
	err = os.WriteFile(filepath.Join(dir, "complex.tpl"), []byte(complexTemplate), fs.ModePerm)
	require.NoError(t, err)

	// Step 3: RenderTemplate with no global data but with custom filters
	result, err := renderer.RenderTemplate("complex", map[string]any{
		"message": "hello world",
	})
	require.NoError(t, err)
	require.Equal(t, " v: HELLO WORLD!", result) // Global data empty, but filters work

	// Step 4: Add global data
	err = renderer.GlobalContext(map[string]any{
		"app_name": "TestApp",
		"version":  "1.0",
	})
	require.NoError(t, err)

	// Step 5: RenderTemplate with all features
	result, err = renderer.RenderTemplate("complex", map[string]any{
		"message": "hello world",
	})
	require.NoError(t, err)
	require.Equal(t, "TestApp v1.0: HELLO WORLD!", result)

	// Step 6: Update global data and test again
	err = renderer.GlobalContext(map[string]any{
		"app_name": "UpdatedApp",
		"version":  "2.0",
	})
	require.NoError(t, err)

	result, err = renderer.RenderTemplate("complex", map[string]any{
		"message": "goodbye world",
	})
	require.NoError(t, err)
	require.Equal(t, "UpdatedApp v2.0: GOODBYE WORLD!", result)
}

func TestEngine_RenderString_Basic(t *testing.T) {
	// Initialize engine (no templates directory needed for RenderString)
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	// Simple template string
	templateContent := `Hello, {{ name }}! You are {{ age }} years old.`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"name": "Alice",
		"age":  25,
	})
	require.NoError(t, err)
	require.Equal(t, "Hello, Alice! You are 25.000000 years old.", result)
}

func TestEngine_RenderString_WithMultipleWriters(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	templateContent := `{{ greeting }}, {{ name }}!`

	var buf1, buf2 bytes.Buffer
	result, err := renderer.RenderString(templateContent, map[string]any{
		"greeting": "Hello",
		"name":     "World",
	}, &buf1, &buf2)

	require.NoError(t, err)
	expected := "Hello, World!"
	require.Equal(t, expected, result)
	require.Equal(t, expected, buf1.String())
	require.Equal(t, expected, buf2.String())
}

func TestEngine_RenderString_WithGlobalData(t *testing.T) {
	renderer, err := template.NewRenderer(
		template.WithBaseDir("/tmp"),
		template.WithGlobalData(map[string]any{
			"app_name": "TestApp",
			"version":  "1.0",
		}),
	)
	require.NoError(t, err)

	templateContent := `{{ app_name }} v{{ version }}: {{ message }}`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"message": "Hello from template string",
	})
	require.NoError(t, err)
	require.Equal(t, "TestApp v1.0: Hello from template string", result)
}

func TestEngine_RenderString_WithCustomFilters(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	// Register custom filter
	err = renderer.RegisterFilter("shout", func(input any, param any) (any, error) {
		return fmt.Sprintf("%v!", strings.ToUpper(fmt.Sprintf("%v", input))), nil
	})
	require.NoError(t, err)

	templateContent := `{{ message|shout }}`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"message": "hello world",
	})
	require.NoError(t, err)
	require.Equal(t, "HELLO WORLD!", result)
}

func TestEngine_RenderString_WithBuiltinFilters(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	templateContent := `{{ text|trim|lowerfirst }}`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"text": "  HELLO WORLD  ",
	})
	require.NoError(t, err)
	require.Equal(t, "hELLO WORLD", result)
}

func TestEngine_RenderString_WithStructData(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	templateContent := `User: {{ user.name }} ({{ user.age }})`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"user": User{Name: "Bob", Age: 35},
	})
	require.NoError(t, err)
	require.Equal(t, "User: Bob (35.000000)", result)
}

func TestEngine_RenderString_ComplexTemplate(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	templateContent := `
{%- for item in items -%}
{{ item.name }}: ${{ item.price }}
{% endfor -%}
Total: ${{ total }}
`

	result, err := renderer.RenderString(templateContent, map[string]any{
		"items": []map[string]any{
			{"name": "Apple", "price": 1.50},
			{"name": "Banana", "price": 0.75},
		},
		"total": 2.25,
	})
	require.NoError(t, err)
	expected := "Apple: $1.500000\nBanana: $0.750000\nTotal: $2.250000\n"
	require.Equal(t, expected, result)
}

func TestEngine_RenderString_ErrorHandling(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	// Test invalid template syntax
	invalidTemplate := `{{ unclosed_tag`
	_, err = renderer.RenderString(invalidTemplate, map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse template string")

	// Test missing variable (should not error, just render empty)
	validTemplate := `Hello, {{ missing_var }}!`
	result, err := renderer.RenderString(validTemplate, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "Hello, !", result)
}

func TestEngine_RenderString_DynamicGlobalData(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	templateContent := `{{ app_name }}: {{ message }}`

	// First render without global data
	result, err := renderer.RenderString(templateContent, map[string]any{
		"message": "test message",
	})
	require.NoError(t, err)
	require.Equal(t, ": test message", result)

	// Add global data
	err = renderer.GlobalContext(map[string]any{
		"app_name": "DynamicApp",
	})
	require.NoError(t, err)

	// Second render with global data
	result, err = renderer.RenderString(templateContent, map[string]any{
		"message": "test message",
	})
	require.NoError(t, err)
	require.Equal(t, "DynamicApp: test message", result)
}

func TestEngine_Render_AutoDetection(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	// Test 1: Template filename (should call RenderTemplate)
	result, err := renderer.Render("hello", map[string]any{
		"name":  "Alice",
		"count": 3,
	})
	require.NoError(t, err)
	require.Equal(t, "Hello, Alice! You have 3 items.\n", result)

	// Test 2: Template string with {{ (should call RenderString)
	result, err = renderer.Render("Hello, {{ name }}!", map[string]any{
		"name": "Bob",
	})
	require.NoError(t, err)
	require.Equal(t, "Hello, Bob!", result)

	// Test 3: Template string with {% (should call RenderString)
	result, err = renderer.Render("{% if name %}Hello, {{ name }}!{% endif %}", map[string]any{
		"name": "Charlie",
	})
	require.NoError(t, err)
	require.Equal(t, "Hello, Charlie!", result)

	// Test 4: Plain text without template syntax (should call RenderTemplate and fail)
	_, err = renderer.Render("plaintext", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load template plaintext.tpl")
}

func TestEngine_Render_EdgeCases(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
	require.NoError(t, err)

	// Test with mixed content that looks like filename but has template syntax
	result, err := renderer.Render("file.txt with {{ content }}", map[string]any{
		"content": "dynamic data",
	})
	require.NoError(t, err)
	require.Equal(t, "file.txt with dynamic data", result)

	// Test with filename that could be mistaken for template
	// This should try to load as file (and fail since file doesn't exist)
	_, err = renderer.Render("template-name", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load template template-name.tpl")
}

func TestEngine_Render_WithGlobalDataAndFilters(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(
		template.WithBaseDir(dir),
		template.WithGlobalData(map[string]any{
			"app_name": "TestApp",
		}),
	)
	require.NoError(t, err)

	// Register a custom filter
	err = renderer.RegisterFilter("excited", func(input any, param any) (any, error) {
		return fmt.Sprintf("%v!!", input), nil
	})
	require.NoError(t, err)

	// Test 1: File template with global data and filters
	fileTemplate := `Welcome to {{ app_name }}, {{ name|excited }}`
	err = os.WriteFile(filepath.Join(dir, "welcome.tpl"), []byte(fileTemplate), fs.ModePerm)
	require.NoError(t, err)

	result, err := renderer.Render("welcome", map[string]any{
		"name": "Alice",
	})
	require.NoError(t, err)
	require.Equal(t, "Welcome to TestApp, Alice!!", result)

	// Test 2: String template with same global data and filters
	result, err = renderer.Render("Welcome to {{ app_name }}, {{ name|excited }}", map[string]any{
		"name": "Bob",
	})
	require.NoError(t, err)
	require.Equal(t, "Welcome to TestApp, Bob!!", result)
}

func TestEngine_Render_MultipleWriters(t *testing.T) {
	dir, cleanup := createTempTemplates(t)
	defer cleanup()

	renderer, err := template.NewRenderer(template.WithBaseDir(dir))
	require.NoError(t, err)

	var buf1, buf2 bytes.Buffer

	// Test with file template
	result, err := renderer.Render("hello", map[string]any{
		"name":  "File",
		"count": 1,
	}, &buf1, &buf2)
	require.NoError(t, err)
	expected := "Hello, File! You have 1 items.\n"
	require.Equal(t, expected, result)
	require.Equal(t, expected, buf1.String())
	require.Equal(t, expected, buf2.String())

	// Clear buffers
	buf1.Reset()
	buf2.Reset()

	// Test with string template
	result, err = renderer.Render("Hello, {{ name }}!", map[string]any{
		"name": "String",
	}, &buf1, &buf2)
	require.NoError(t, err)
	expected = "Hello, String!"
	require.Equal(t, expected, result)
	require.Equal(t, expected, buf1.String())
	require.Equal(t, expected, buf2.String())
}

func TestEngine_Render_DetectionLogic(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isTemplate  bool
		description string
	}{
		{
			name:        "simple_variable",
			input:       "{{ name }}",
			isTemplate:  true,
			description: "Simple variable should be detected as template",
		},
		{
			name:        "if_statement",
			input:       "{% if condition %}text{% endif %}",
			isTemplate:  true,
			description: "If statement should be detected as template",
		},
		{
			name:        "filename_only",
			input:       "template-name",
			isTemplate:  false,
			description: "Plain filename should be detected as file",
		},
		{
			name:        "filename_with_extension",
			input:       "template.html",
			isTemplate:  false,
			description: "Filename with extension should be detected as file",
		},
		{
			name:        "mixed_content",
			input:       "Hello {{ name }}, welcome!",
			isTemplate:  true,
			description: "Mixed content with variables should be detected as template",
		},
		{
			name:        "empty_string",
			input:       "",
			isTemplate:  false,
			description: "Empty string should be detected as filename",
		},
		{
			name:        "special_filename",
			input:       "my-template_v2.tpl",
			isTemplate:  false,
			description: "Complex filename should be detected as file",
		},
		{
			name:        "template_with_filename_like_text",
			input:       "Loading template.html: {{ status }}",
			isTemplate:  true,
			description: "Text that mentions filename but has template syntax should be template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test the detection logic indirectly through Render method
			renderer, err := template.NewRenderer(template.WithBaseDir("/tmp"))
			require.NoError(t, err)

			if tc.isTemplate {
				// For template content, we expect successful parsing (even if variables are missing)
				result, err := renderer.Render(tc.input, map[string]any{})
				if err == nil || !strings.Contains(err.Error(), "failed to load template") {
					// Either successful or a template execution error (not file loading error)
					t.Logf("Correctly detected as template content: %q -> %q", tc.input, result)
				} else {
					t.Errorf("Expected template content but got file loading error: %v", err)
				}
			} else {
				// For filenames, we expect a file loading error
				_, err := renderer.Render(tc.input, map[string]any{})
				if err != nil && strings.Contains(err.Error(), "failed to load template") {
					t.Logf("Correctly detected as filename: %q", tc.input)
				} else {
					t.Errorf("Expected filename but was treated as template content")
				}
			}
		})
	}
}
