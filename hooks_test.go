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

func TestHookChain_Constructor(t *testing.T) {
	// Test empty constructor
	chain := template.NewHookChain()
	require.NotNil(t, chain)

	// Test with pre-hooks option
	preHook1 := func(ctx *template.HookContext) error {
		return nil
	}
	preHook2 := func(ctx *template.HookContext) error {
		return nil
	}

	chain = template.NewHookChain(template.WithPreHooksChain(preHook1, preHook2))
	require.NotNil(t, chain)

	// Test with post-hooks option
	postHook1 := func(ctx *template.HookContext) (string, error) {
		return ctx.Output, nil
	}
	postHook2 := func(ctx *template.HookContext) (string, error) {
		return ctx.Output, nil
	}

	chain = template.NewHookChain(template.WithPostHooksChain(postHook1, postHook2))
	require.NotNil(t, chain)

	// Test with both options
	chain = template.NewHookChain(
		template.WithPreHooksChain(preHook1),
		template.WithPostHooksChain(postHook1),
	)
	require.NotNil(t, chain)
}

func TestHookChain_AddHooks(t *testing.T) {
	chain := template.NewHookChain()

	var executionOrder []string

	// Add pre-hooks using method chaining
	preHook1 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "pre1")
		return nil
	}
	preHook2 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "pre2")
		return nil
	}

	chain.AddPreHook(preHook1).AddPreHook(preHook2)

	// Add post-hooks using method chaining
	postHook1 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "post1")
		return "post1-" + ctx.Output, nil
	}
	postHook2 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "post2")
		return "post2-" + ctx.Output, nil
	}

	chain.AddPostHook(postHook1).AddPostHook(postHook2)

	// Execute and verify chaining worked
	ctx := &template.HookContext{
		Data:     map[string]any{},
		Metadata: make(map[string]any),
		Output:   "original",
	}

	err := chain.ExecutePreHooks(ctx)
	require.NoError(t, err)

	result, err := chain.ExecutePostHooks(ctx)
	require.NoError(t, err)

	// Verify execution order and result
	expectedOrder := []string{"pre1", "pre2", "post1", "post2"}
	require.Equal(t, expectedOrder, executionOrder)
	require.Equal(t, "post2-post1-original", result)
}

func TestHookChain_ExecutePreHooks(t *testing.T) {
	var executionOrder []string
	var receivedData []any

	hook1 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook1")
		receivedData = append(receivedData, ctx.Data)
		// Modify data
		if data, ok := ctx.Data.(map[string]any); ok {
			data["hook1"] = "executed"
		}
		return nil
	}

	hook2 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook2")
		receivedData = append(receivedData, ctx.Data)
		// Modify metadata
		ctx.Metadata["hook2"] = "executed"
		return nil
	}

	hook3 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook3")
		receivedData = append(receivedData, ctx.Data)
		return nil
	}

	chain := template.NewHookChain(template.WithPreHooksChain(hook1, hook2, hook3))

	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{"initial": "value"},
		Metadata:     make(map[string]any),
		IsPreHook:    true,
	}

	err := chain.ExecutePreHooks(ctx)
	require.NoError(t, err)

	// Verify execution order
	expectedOrder := []string{"hook1", "hook2", "hook3"}
	require.Equal(t, expectedOrder, executionOrder)

	// Verify data modifications were applied
	data := ctx.Data.(map[string]any)
	require.Equal(t, "executed", data["hook1"])
	require.Equal(t, "executed", ctx.Metadata["hook2"])

	// Verify all hooks received the same data reference (mutations persist)
	require.Len(t, receivedData, 3)
}

func TestHookChain_ExecutePreHooks_ErrorHandling(t *testing.T) {
	var executionOrder []string

	hook1 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook1")
		return nil
	}

	hook2 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook2")
		return fmt.Errorf("hook2 failed")
	}

	hook3 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "hook3")
		return nil
	}

	chain := template.NewHookChain(template.WithPreHooksChain(hook1, hook2, hook3))

	ctx := &template.HookContext{
		Data:      map[string]any{},
		Metadata:  make(map[string]any),
		IsPreHook: true,
	}

	err := chain.ExecutePreHooks(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hook2 failed")

	// Verify execution stopped at the failing hook
	expectedOrder := []string{"hook1", "hook2"}
	require.Equal(t, expectedOrder, executionOrder)
}

func TestHookChain_ExecutePostHooks(t *testing.T) {
	var executionOrder []string

	hook1 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook1")
		return "hook1-" + ctx.Output, nil
	}

	hook2 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook2")
		// Verify ctx.Output was updated by previous hook
		require.Contains(t, ctx.Output, "hook1-")
		return "hook2-" + ctx.Output, nil
	}

	hook3 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook3")
		require.Contains(t, ctx.Output, "hook2-hook1-")
		return "hook3-" + ctx.Output, nil
	}

	chain := template.NewHookChain(template.WithPostHooksChain(hook1, hook2, hook3))

	ctx := &template.HookContext{
		TemplateName: "test",
		Data:         map[string]any{},
		Metadata:     make(map[string]any),
		Output:       "original",
		IsPreHook:    false,
	}

	result, err := chain.ExecutePostHooks(ctx)
	require.NoError(t, err)

	// Verify execution order
	expectedOrder := []string{"hook1", "hook2", "hook3"}
	require.Equal(t, expectedOrder, executionOrder)

	// Verify output transformations
	require.Equal(t, "hook3-hook2-hook1-original", result)
	require.Equal(t, "hook3-hook2-hook1-original", ctx.Output)
}

func TestHookChain_ExecutePostHooks_ErrorHandling(t *testing.T) {
	var executionOrder []string

	hook1 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook1")
		return "hook1-" + ctx.Output, nil
	}

	hook2 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook2")
		return "", fmt.Errorf("hook2 failed")
	}

	hook3 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "hook3")
		return "hook3-" + ctx.Output, nil
	}

	chain := template.NewHookChain(template.WithPostHooksChain(hook1, hook2, hook3))

	ctx := &template.HookContext{
		Data:     map[string]any{},
		Metadata: make(map[string]any),
		Output:   "original",
	}

	result, err := chain.ExecutePostHooks(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hook2 failed")
	require.Empty(t, result)

	// Verify execution stopped at the failing hook
	expectedOrder := []string{"hook1", "hook2"}
	require.Equal(t, expectedOrder, executionOrder)
}

func TestHookChain_AsPreHook(t *testing.T) {
	var executionOrder []string

	hook1 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "chain-hook1")
		return nil
	}

	hook2 := func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "chain-hook2")
		return nil
	}

	chain := template.NewHookChain(template.WithPreHooksChain(hook1, hook2))

	// Convert chain to a single PreHook
	combinedPreHook := chain.AsPreHook()

	// Use it in a hook manager
	manager := template.NewHooksManager()
	manager.AddPreHook(combinedPreHook, 5)

	// Add individual hook for comparison
	manager.AddPreHook(func(ctx *template.HookContext) error {
		executionOrder = append(executionOrder, "individual-hook")
		return nil
	}, 10)

	// Execute all hooks
	hooks := manager.PreHooks()
	ctx := &template.HookContext{
		Data:      map[string]any{},
		Metadata:  make(map[string]any),
		IsPreHook: true,
	}

	for _, hook := range hooks {
		err := hook(ctx)
		require.NoError(t, err)
	}

	// Verify both chain hooks executed before individual hook (due to priority)
	expectedOrder := []string{"chain-hook1", "chain-hook2", "individual-hook"}
	require.Equal(t, expectedOrder, executionOrder)
}

func TestHookChain_AsPostHook(t *testing.T) {
	var executionOrder []string

	hook1 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "chain-hook1")
		return "chain1-" + ctx.Output, nil
	}

	hook2 := func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "chain-hook2")
		return "chain2-" + ctx.Output, nil
	}

	chain := template.NewHookChain(template.WithPostHooksChain(hook1, hook2))

	// Convert chain to a single PostHook
	combinedPostHook := chain.AsPostHook()

	// Use it in a hook manager
	manager := template.NewHooksManager()
	manager.AddPostHook(combinedPostHook, 5)

	// Add individual hook for comparison
	manager.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionOrder = append(executionOrder, "individual-hook")
		return "individual-" + ctx.Output, nil
	}, 10)

	// Execute all hooks
	hooks := manager.PostHooks()
	ctx := &template.HookContext{
		Data:     map[string]any{},
		Metadata: make(map[string]any),
		Output:   "original",
	}

	output := ctx.Output
	for _, hook := range hooks {
		var err error
		output, err = hook(&template.HookContext{
			Data:     ctx.Data,
			Metadata: ctx.Metadata,
			Output:   output,
		})
		require.NoError(t, err)
	}

	// Verify execution order and output transformation
	expectedOrder := []string{"chain-hook1", "chain-hook2", "individual-hook"}
	require.Equal(t, expectedOrder, executionOrder)
	require.Equal(t, "individual-chain2-chain1-original", output)
}

func TestHookChain_EmptyChain(t *testing.T) {
	chain := template.NewHookChain()

	ctx := &template.HookContext{
		Data:     map[string]any{"test": "data"},
		Metadata: make(map[string]any),
		Output:   "original",
	}

	// Test empty pre-hook chain
	err := chain.ExecutePreHooks(ctx)
	require.NoError(t, err)

	// Test empty post-hook chain
	result, err := chain.ExecutePostHooks(ctx)
	require.NoError(t, err)
	require.Equal(t, "original", result)

	// Test as converted hooks
	preHook := chain.AsPreHook()
	err = preHook(ctx)
	require.NoError(t, err)

	postHook := chain.AsPostHook()
	result, err = postHook(ctx)
	require.NoError(t, err)
	require.Equal(t, "original", result)
}

func TestHookChain_WithRenderer_Integration(t *testing.T) {
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	require.NoError(t, err)

	var executionMarkers []string

	// Create a pre-hook chain
	preChain := template.NewHookChain()
	preChain.AddPreHook(func(ctx *template.HookContext) error {
		executionMarkers = append(executionMarkers, "pre-chain-1")
		return nil
	})
	preChain.AddPreHook(func(ctx *template.HookContext) error {
		executionMarkers = append(executionMarkers, "pre-chain-2")
		// Add default data
		if data, ok := ctx.Data.(map[string]any); ok {
			if _, exists := data["app_name"]; !exists {
				data["app_name"] = "ChainApp"
			}
		}
		return nil
	})

	// Create a post-hook chain
	postChain := template.NewHookChain()
	postChain.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "post-chain-1")
		return "// Chain Header 1\n" + ctx.Output, nil
	})
	postChain.AddPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "post-chain-2")
		return "// Chain Header 2\n" + ctx.Output, nil
	})

	// Register chains as hooks
	renderer.RegisterPreHook(preChain.AsPreHook())
	renderer.RegisterPostHook(postChain.AsPostHook())

	// Add an individual hook for comparison
	renderer.RegisterPostHook(func(ctx *template.HookContext) (string, error) {
		executionMarkers = append(executionMarkers, "individual-post")
		return "// Individual Header\n" + ctx.Output, nil
	})

	result, err := renderer.RenderTemplate("simple", map[string]any{
		"name": "Alice",
	})
	require.NoError(t, err)

	// Verify execution order
	expectedOrder := []string{
		"pre-chain-1", "pre-chain-2",
		"post-chain-1", "post-chain-2",
		"individual-post",
	}
	require.Equal(t, expectedOrder, executionMarkers)

	// Verify output contains chain transformations
	require.Contains(t, result, "// Individual Header")
	require.Contains(t, result, "// Chain Header 2")
	require.Contains(t, result, "// Chain Header 1")
	require.Contains(t, result, "Hello, Alice! Welcome to ChainApp.")
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
