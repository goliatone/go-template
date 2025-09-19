package main

import (
	"fmt"
	"log"

	"github.com/goliatone/go-template"
)

func main() {
	fmt.Println("=== Example: Common Hooks ===")
	demoCommonHooks()

	fmt.Println("\n=== Example: Custom Hook ===")
	demoCustomHook()
}

func demoCommonHooks() {
	// Create renderer
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	if err != nil {
		log.Fatal(err)
	}

	// Create common hooks instance
	hooks := template.NewCommonHooks()

	// Register pre-hooks for data validation and defaults
	renderer.RegisterPreHook(hooks.ValidateDataHook([]string{"package_name"}))
	renderer.RegisterPreHook(hooks.SetDefaultsHook(map[string]any{
		"version": "1.0.0",
		"name":    "DefaultName",
	}))

	// Register post-hooks for code generation
	renderer.RegisterPostHook(hooks.AddGeneratedWarningHook())
	renderer.RegisterPostHook(hooks.AddTimestampHook())
	renderer.RegisterPostHook(hooks.AddCopyrightHook("Copyright 2024 MyCompany"))
	renderer.RegisterPostHook(hooks.RemoveTrailingWhitespaceHook())

	// Render a Go template
	result, err := renderer.RenderTemplate("code.go", map[string]any{
		"package_name": "main",
		"struct_name":  "Config",
		// version and name will use defaults
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}

func demoCustomHook() {
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	if err != nil {
		log.Fatal(err)
	}

	// Custom post-hook that adds a build tag
	buildTagHook := func(ctx *template.HookContext) (string, error) {
		if ctx.TemplateName == "code.go" {
			return "//go:build !ignore\n\n" + ctx.Output, nil
		}
		return ctx.Output, nil
	}

	renderer.RegisterPostHook(buildTagHook)

	result, err := renderer.RenderTemplate("code.go", map[string]any{
		"package_name": "main",
		"struct_name":  "Config",
		"name":         "TestApp",
		"version":      "1.0.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated with custom build tag:")
	fmt.Println(result)
}
