package template_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goliatone/go-template"
	"github.com/goliatone/go-template/templatehooks"
	"github.com/stretchr/testify/require"
)

func TestHooks_ErrorHandling(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	require.NoError(t, err)

	// Register a pre-hook that always fails
	renderer.RegisterPreHook(func(ctx *template.HookContext) error {
		return fmt.Errorf("validation failed: test error")
	})

	_, err = renderer.RenderTemplate("simple", map[string]any{
		"name":     "Alice",
		"app_name": "TestApp",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-hook failed")
	require.Contains(t, err.Error(), "validation failed: test error")
}

func TestHooks_PostHook_ErrorHandling(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	require.NoError(t, err)

	// Register a post-hook that always fails
	renderer.RegisterPostHook(func(ctx *template.HookContext) (string, error) {
		return "", fmt.Errorf("post-processing failed: test error")
	})

	_, err = renderer.RenderTemplate("simple", map[string]any{
		"name":     "Alice",
		"app_name": "TestApp",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "post-hook failed")
	require.Contains(t, err.Error(), "post-processing failed: test error")
}

func TestHooks_ConcurrentAccess(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	require.NoError(t, err)

	hooks := templatehooks.NewCommonHooks()

	// Register hooks from multiple goroutines
	done := make(chan bool, 3)

	go func() {
		renderer.RegisterPreHook(hooks.AddMetadataHook())
		done <- true
	}()

	go func() {
		renderer.RegisterPostHook(hooks.AddTimestampHook())
		done <- true
	}()

	go func() {
		renderer.RegisterPostHook(hooks.RemoveTrailingWhitespaceHook())
		done <- true
	}()

	// Wait for all goroutines to complete
	for range 3 {
		<-done
	}

	// Verify hooks were registered correctly by using them
	result, err := renderer.RenderTemplate("code.go", map[string]any{
		"package_name": "main",
		"struct_name":  "Config",
		"name":         "test",
		"version":      "1.0.0",
	})
	require.NoError(t, err)
	require.Contains(t, result, "Generated on")
	require.Contains(t, result, "package main")
}

func TestHooks_PrioritySorting_PreHooks(t *testing.T) {
	manager := template.NewHooksManager()

	var executionOrder []int

	// Register hooks with different priorities
	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 10)
		return nil
	}, 10)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 5)
		return nil
	}, 5)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 1)
		return nil
	}, 1)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 0)
		return nil
	})

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, -5)
		return nil
	}, -5)

	hooks := manager.PreHooks()
	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{},
		Metadata:     make(map[string]any),
		IsPreHook:    true,
	}

	for _, hook := range hooks {
		err := hook(ctx)
		require.NoError(t, err)
	}

	// Hooks should execute in ascending priority order: -5, 0, 1, 5, 10
	expected := []int{-5, 0, 1, 5, 10}
	require.Equal(t, expected, executionOrder)
}

func TestHooks_PrioritySorting_PostHooks(t *testing.T) {
	manager := template.NewHooksManager()

	var executionOrder []int

	// Register hooks with different priorities
	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, 100)
		return ctx.Output, nil
	}, 100)

	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, 20)
		return ctx.Output, nil
	}, 20)

	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, 0)
		return ctx.Output, nil
	}) // Default priority 0

	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, -10)
		return ctx.Output, nil
	}, -10)

	// Execute hooks and verify order
	hooks := manager.PostHooks()
	ctx := &template.HookContext{
		TemplateName: "test",
		Output:       "test output",
		Metadata:     make(map[string]any),
		IsPreHook:    false,
	}

	for _, hook := range hooks {
		_, err := hook(ctx)
		require.NoError(t, err)
	}

	// Hooks should execute in ascending priority order: -10, 0, 20, 100
	expected := []int{-10, 0, 20, 100}
	require.Equal(t, expected, executionOrder)
}

func TestHooks_PrioritySorting_SamePriority(t *testing.T) {
	manager := template.NewHooksManager()

	var executionOrder []string

	// Register multiple hooks with same priority
	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "first")
		return nil
	}, 5)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "second")
		return nil
	}, 5)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "third")
		return nil
	}, 5)

	// Execute hooks
	hooks := manager.PreHooks()
	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{},
		Metadata:     make(map[string]any),
		IsPreHook:    true,
	}

	for _, hook := range hooks {
		err := hook(ctx)
		require.NoError(t, err)
	}

	// Hooks with same priority should execute in registration order
	expected := []string{"first", "second", "third"}
	require.Equal(t, expected, executionOrder)
}

func TestHooks_PrioritySorting_IntegrationWithHookManager(t *testing.T) {
	manager := template.NewHooksManager()

	var executionMarkers []string

	// Register pre-hooks with different priorities
	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionMarkers = append(executionMarkers, "pre-priority-5")
		return nil
	}, 5)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionMarkers = append(executionMarkers, "pre-priority-1")
		return nil
	}, 1)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionMarkers = append(executionMarkers, "pre-priority-0")
		return nil
	}) // Default priority 0

	// Register post-hooks with different priorities
	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "post-priority-10")
		return "// Priority 10\n" + ctx.Output, nil
	}, 10)

	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "post-priority-3")
		return "// Priority 3\n" + ctx.Output, nil
	}, 3)

	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "post-priority-0")
		return "// Priority 0\n" + ctx.Output, nil
	}) // Default priority 0

	// Simulate pre-hook execution
	preHooks := manager.PreHooks()
	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{},
		Metadata:     make(map[string]any),
		IsPreHook:    true,
	}

	for _, hook := range preHooks {
		err := hook(ctx)
		require.NoError(t, err)
	}

	// Simulate post-hook execution
	postHooks := manager.PostHooks()
	ctx.IsPreHook = false
	ctx.Output = "Original content"

	output := ctx.Output
	for _, hook := range postHooks {
		var err error
		output, err = hook(&template.HookContext{
			TemplateName: ctx.TemplateName,
			Output:       output,
			Metadata:     ctx.Metadata,
			IsPreHook:    false,
		})
		require.NoError(t, err)
	}

	// Verify execution order: pre-hooks (0, 1, 5) then post-hooks (0, 3, 10)
	expectedOrder := []string{
		"pre-priority-0", "pre-priority-1", "pre-priority-5",
		"post-priority-0", "post-priority-3", "post-priority-10",
	}
	require.Equal(t, expectedOrder, executionMarkers)

	// Verify post-hook content transformations are applied in order
	require.Contains(t, output, "Priority 10")
	require.Contains(t, output, "Priority 3")
	require.Contains(t, output, "Priority 0")
	require.Contains(t, output, "Original content")
}

func TestHooks_PrioritySorting_NegativePriorities(t *testing.T) {
	manager := template.NewHooksManager()

	var executionOrder []int

	// Test with negative priorities
	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, -100)
		return nil
	}, -100)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, -1)
		return nil
	}, -1)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 0)
		return nil
	}, 0)

	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, 1)
		return nil
	}, 1)

	// Execute hooks
	hooks := manager.PreHooks()
	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{},
		Metadata:     make(map[string]any),
		IsPreHook:    true,
	}

	for _, hook := range hooks {
		err := hook(ctx)
		require.NoError(t, err)
	}

	// Should execute in ascending order including negatives
	expected := []int{-100, -1, 0, 1}
	require.Equal(t, expected, executionOrder)
}

func TestHooks_PrioritySorting_EmptyHooks(t *testing.T) {
	manager := template.NewHooksManager()

	// Test with no hooks registered
	preHooks := manager.PreHooks()
	postHooks := manager.PostHooks()

	require.Empty(t, preHooks)
	require.Empty(t, postHooks)
}

func TestHooks_HelperFunctions(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		templateFile string
		output       string
		expectGo     bool
		expectCode   bool
	}{
		{
			name:         "go_file_by_extension",
			templateName: "test.go",
			templateFile: "test.go.tpl",
			output:       "func main() {}",
			expectGo:     true,
			expectCode:   true,
		},
		{
			name:         "go_template_by_extension",
			templateName: "test.go",
			templateFile: "test.go.tpl",
			output:       "func main() {}",
			expectGo:     true,
			expectCode:   true,
		},
		{
			name:         "go_file_by_content",
			templateName: "test.txt",
			templateFile: "test.txt.tpl",
			output:       "package main\n\nfunc main() {}",
			expectGo:     true,
			expectCode:   true,
		},
		{
			name:         "javascript_file",
			templateName: "test.js",
			templateFile: "test.js.tpl",
			output:       "function main() {}",
			expectGo:     false,
			expectCode:   true,
		},
		{
			name:         "yaml_file",
			templateName: "config.yaml",
			templateFile: "config.yaml.tpl",
			output:       "name: test\nversion: 1.0",
			expectGo:     false,
			expectCode:   false,
		},
		{
			name:         "python_by_content",
			templateName: "script.txt",
			templateFile: "script.txt.tpl",
			output:       "def main():\n    pass",
			expectGo:     false,
			expectCode:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary template file to test template name detection
			dir := t.TempDir()
			templatePath := filepath.Join(dir, tc.templateFile)
			err := os.WriteFile(templatePath, []byte(tc.output), 0644)
			require.NoError(t, err)

			renderer, err := template.NewRenderer(template.WithBaseDir(dir))
			require.NoError(t, err)

			hooks := templatehooks.NewCommonHooks()
			renderer.RegisterPostHook(hooks.AddTimestampHook())       // Go files only
			renderer.RegisterPostHook(hooks.AddCopyrightHook("Test")) // Code files only

			result, err := renderer.RenderTemplate(tc.templateName, map[string]any{})
			require.NoError(t, err)

			if tc.expectGo {
				require.Contains(t, result, "Generated on", "Expected Go file detection for %s", tc.templateName)
			} else {
				require.NotContains(t, result, "Generated on", "Did not expect Go file detection for %s", tc.templateName)
			}

			if tc.expectCode {
				require.Contains(t, result, "// Test", "Expected code file detection for %s", tc.templateName)
			} else {
				require.NotContains(t, result, "// Test", "Did not expect code file detection for %s", tc.templateName)
			}
		})
	}
}
