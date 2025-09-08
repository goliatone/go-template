package template_test

import (
	"bytes"
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
