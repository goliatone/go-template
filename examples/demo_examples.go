//go:build skip

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/goliatone/go-template"
	"github.com/goliatone/go-template/templatehooks"
)

func main() {
	fmt.Println("=== Example: Common Hooks ===")
	demoCommonHooks()

	fmt.Println("\n=== Example: Custom Hook ===")
	demoCustomHook()

	fmt.Println("\n=== Example: Template Helpers & Filters ===")
	demoTemplateHelpersAndFilters()
}

func demoCommonHooks() {
	// Create renderer
	renderer, err := template.NewRenderer(template.WithBaseDir("testdata"))
	if err != nil {
		log.Fatal(err)
	}

	// Create common hooks instance
	hooks := templatehooks.NewCommonHooks()

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

func demoTemplateHelpersAndFilters() {
	renderer, err := template.NewRenderer(
		template.WithBaseDir("testdata"),
		template.WithTemplateFunc(map[string]any{
			"is_even": func(v any) bool {
				if n, ok := v.(int); ok {
					return n%2 == 0
				}
				return false
			},
			"double": pongo2.FilterFunction(func(in *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
				n, _ := in.Interface().(int)
				return pongo2.AsValue(n * 2), nil
			}),
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := renderer.GlobalContext(map[string]any{
		"shout": func(v any) string {
			return strings.ToUpper(fmt.Sprint(v))
		},
	}); err != nil {
		log.Fatal(err)
	}

	tmpl := "" +
		"Value: {{ value }}\n" +
		"Even? {{ is_even(value) }}\n" +
		"Double: {{ value|double }}\n" +
		"Shout: {{ shout(name) }}\n"

	output, err := renderer.Render(tmpl, map[string]any{
		"value": 3,
		"name":  "codex",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
}
