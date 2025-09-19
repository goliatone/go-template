package templatehooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/goliatone/go-template"
)

////////////////////////////////////////////////////////////////////////////////

// CommonHooks provides a collection of commonly used hooks
type CommonHooks struct{}

// NewCommonHooks creates a new instance of CommonHooks
func NewCommonHooks() *CommonHooks {
	return &CommonHooks{}
}

// TimestampHookOption configures the behaviour of AddTimestampHook.
type TimestampHookOption func(*TimestampHookConfig)

// TimestampHookConfig captures configuration for AddTimestampHook.
type TimestampHookConfig struct {
	CommentPrefix string
	Format        string
	Location      *time.Location
	Condition     template.HookCondition
	MessageFormat string
}

// WithTimestampCommentPrefix overrides the default line comment prefix.
func WithTimestampCommentPrefix(prefix string) TimestampHookOption {
	return func(cfg *TimestampHookConfig) {
		cfg.CommentPrefix = prefix
	}
}

// WithTimestampFormat changes the time formatting layout.
func WithTimestampFormat(layout string) TimestampHookOption {
	return func(cfg *TimestampHookConfig) {
		cfg.Format = layout
	}
}

// WithTimestampLocation changes the time zone/location used for timestamps.
func WithTimestampLocation(loc *time.Location) TimestampHookOption {
	return func(cfg *TimestampHookConfig) {
		cfg.Location = loc
	}
}

// WithTimestampCondition sets a predicate governing when the hook executes.
func WithTimestampCondition(condition template.HookCondition) TimestampHookOption {
	return func(cfg *TimestampHookConfig) {
		cfg.Condition = condition
	}
}

// WithTimestampMessageFormat sets the message template for the comment line.
// The format string must contain a single %s placeholder for the formatted timestamp.
func WithTimestampMessageFormat(message string) TimestampHookOption {
	return func(cfg *TimestampHookConfig) {
		cfg.MessageFormat = message
	}
}

// AddTimestampHook adds a timestamp to generated files.
func (h *CommonHooks) AddTimestampHook(opts ...TimestampHookOption) template.PostHook {
	cfg := TimestampHookConfig{
		CommentPrefix: "// ",
		Format:        "2006-01-02 15:04:05",
		Location:      time.Local,
		Condition: func(ctx *template.HookContext) bool {
			return isGoFile(ctx.TemplateName, ctx.Output)
		},
		MessageFormat: "Generated on %s",
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx *template.HookContext) (string, error) {
		if cfg.Condition != nil && !cfg.Condition(ctx) {
			return ctx.Output, nil
		}

		timestamp := time.Now()
		if cfg.Location != nil {
			timestamp = timestamp.In(cfg.Location)
		}

		formatted := fmt.Sprintf(cfg.MessageFormat, timestamp.Format(cfg.Format))
		header := BuildLineComment(cfg.CommentPrefix, formatted)
		return header + "\n" + ctx.Output, nil
	}
}

// CopyrightHookOption configures AddCopyrightHook behaviour.
type CopyrightHookOption func(*CopyrightHookConfig)

// CopyrightHookConfig captures settings for AddCopyrightHook.
type CopyrightHookConfig struct {
	CommentPrefix string
	Condition     template.HookCondition
}

// WithCopyrightCommentPrefix overrides the default prefix ("// ").
func WithCopyrightCommentPrefix(prefix string) CopyrightHookOption {
	return func(cfg *CopyrightHookConfig) {
		cfg.CommentPrefix = prefix
	}
}

// WithCopyrightCondition sets a predicate governing when the hook executes.
func WithCopyrightCondition(condition template.HookCondition) CopyrightHookOption {
	return func(cfg *CopyrightHookConfig) {
		cfg.Condition = condition
	}
}

// AddCopyrightHook adds a copyright header.
func (h *CommonHooks) AddCopyrightHook(copyright string, opts ...CopyrightHookOption) template.PostHook {
	cfg := CopyrightHookConfig{
		CommentPrefix: "// ",
		Condition: func(ctx *template.HookContext) bool {
			return isCodeFile(ctx.TemplateName, ctx.Output)
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx *template.HookContext) (string, error) {
		if copyright == "" {
			return ctx.Output, nil
		}

		if cfg.Condition != nil && !cfg.Condition(ctx) {
			return ctx.Output, nil
		}

		header := BuildLineComment(cfg.CommentPrefix, copyright)
		return header + "\n" + ctx.Output, nil
	}
}

// CommentBlockStyle describes how to render a block comment.
type CommentBlockStyle struct {
	Start      string
	LinePrefix string
	End        string
}

// BuildLineComment joins a prefix and content into a single comment line.
func BuildLineComment(prefix, content string) string {
	return prefix + content
}

// BuildCommentBlock renders a comment block using the provided style and lines.
func BuildCommentBlock(style CommentBlockStyle, lines []string) string {
	var b strings.Builder
	if style.Start != "" {
		b.WriteString(style.Start)
		b.WriteString("\n")
	}
	for _, line := range lines {
		if line != "" && style.LinePrefix != "" {
			b.WriteString(style.LinePrefix)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if style.End != "" {
		b.WriteString(style.End)
		b.WriteString("\n")
	}
	return b.String()
}

// LicenseHookOption configures AddLicenseHook behaviour.
type LicenseHookOption func(*LicenseHookConfig)

// LicenseHookConfig captures settings for AddLicenseHook.
type LicenseHookConfig struct {
	Style     CommentBlockStyle
	Condition template.HookCondition
}

// WithLicenseCommentStyle overrides the block comment style used for the license header.
func WithLicenseCommentStyle(style CommentBlockStyle) LicenseHookOption {
	return func(cfg *LicenseHookConfig) {
		cfg.Style = style
	}
}

// WithLicenseCondition sets a predicate governing when the hook executes.
func WithLicenseCondition(condition template.HookCondition) LicenseHookOption {
	return func(cfg *LicenseHookConfig) {
		cfg.Condition = condition
	}
}

var defaultLicenseStyle = CommentBlockStyle{
	Start:      "/*",
	LinePrefix: " * ",
	End:        " */",
}

// AddLicenseHook adds a license header
func (h *CommonHooks) AddLicenseHook(license string, opts ...LicenseHookOption) template.PostHook {
	cfg := LicenseHookConfig{
		Style: defaultLicenseStyle,
		Condition: func(ctx *template.HookContext) bool {
			return isCodeFile(ctx.TemplateName, ctx.Output)
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx *template.HookContext) (string, error) {
		if license == "" {
			return ctx.Output, nil
		}

		if cfg.Condition != nil && !cfg.Condition(ctx) {
			return ctx.Output, nil
		}

		lines := strings.Split(license, "\n")
		header := BuildCommentBlock(cfg.Style, lines) + "\n"
		return header + ctx.Output, nil
	}
}

// GeneratedWarningHookOption configures AddGeneratedWarningHook behaviour.
type GeneratedWarningHookOption func(*GeneratedWarningHookConfig)

// GeneratedWarningHookConfig captures settings for AddGeneratedWarningHook.
type GeneratedWarningHookConfig struct {
	CommentPrefix string
	Message       string
	Condition     template.HookCondition
}

// WithGeneratedWarningCommentPrefix overrides the default prefix ("// ").
func WithGeneratedWarningCommentPrefix(prefix string) GeneratedWarningHookOption {
	return func(cfg *GeneratedWarningHookConfig) {
		cfg.CommentPrefix = prefix
	}
}

// WithGeneratedWarningMessage overrides the default warning message.
func WithGeneratedWarningMessage(message string) GeneratedWarningHookOption {
	return func(cfg *GeneratedWarningHookConfig) {
		cfg.Message = message
	}
}

// WithGeneratedWarningCondition sets a predicate governing when the hook executes.
func WithGeneratedWarningCondition(condition template.HookCondition) GeneratedWarningHookOption {
	return func(cfg *GeneratedWarningHookConfig) {
		cfg.Condition = condition
	}
}

// AddGeneratedWarningHook adds a warning that the file is generated
func (h *CommonHooks) AddGeneratedWarningHook(opts ...GeneratedWarningHookOption) template.PostHook {
	cfg := GeneratedWarningHookConfig{
		CommentPrefix: "// ",
		Message:       "Code generated by go-template. DO NOT EDIT.",
		Condition: func(ctx *template.HookContext) bool {
			return isGoFile(ctx.TemplateName, ctx.Output)
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx *template.HookContext) (string, error) {
		if cfg.Condition != nil && !cfg.Condition(ctx) {
			return ctx.Output, nil
		}

		header := BuildLineComment(cfg.CommentPrefix, cfg.Message)
		return header + "\n" + ctx.Output, nil
	}
}

// RemoveTrailingWhitespaceHook removes trailing whitespace from lines
func (h *CommonHooks) RemoveTrailingWhitespaceHook() template.PostHook {
	return func(ctx *template.HookContext) (string, error) {
		lines := strings.Split(ctx.Output, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " \t")
		}
		return strings.Join(lines, "\n"), nil
	}
}

// AddMetadataHook adds template metadata for tracking
// By default, template_content is only populated by RenderString
func (h *CommonHooks) AddMetadataHook() template.PreHook {
	return func(ctx *template.HookContext) error {
		ctx.Metadata["processed_at"] = time.Now()
		ctx.Metadata["template_name"] = ctx.TemplateName
		ctx.Metadata["template_content"] = ctx.Template
		return nil
	}
}

// ValidateDataHook validates required data fields
func (h *CommonHooks) ValidateDataHook(requiredFields []string) template.PreHook {
	return func(ctx *template.HookContext) error {
		data, ok := ctx.Data.(map[string]any)
		if !ok {
			newData, err := template.ConvertToContext(ctx.Data)
			if err != nil {
				return fmt.Errorf("unable to coerce data to map: %w", err)
			}
			data = newData
			ctx.Data = newData
		}

		for _, field := range requiredFields {
			if _, exists := data[field]; !exists {
				return fmt.Errorf("required field '%s' is missing", field)
			}
		}
		return nil
	}
}

// SetDefaultsHook sets default values for missing data fields
func (h *CommonHooks) SetDefaultsHook(defaults map[string]any) template.PreHook {
	return func(ctx *template.HookContext) error {
		data, ok := ctx.Data.(map[string]any)
		if !ok {
			// Convert to map if it's not already
			newData, err := template.ConvertToContext(ctx.Data)
			if err != nil {
				return fmt.Errorf("unable to coerce data to map: %w", err)
			}
			data = newData
			ctx.Data = newData
		}

		for key, value := range defaults {
			if _, exists := data[key]; !exists {
				data[key] = value
			}
		}
		return nil
	}
}

// Helper functions

// isGoFile checks if the output is for a Go file
func isGoFile(templateName, output string) bool {
	return strings.HasSuffix(templateName, ".go") ||
		strings.HasSuffix(templateName, ".go.tpl") ||
		strings.Contains(output, "package ")
}

// isCodeFile checks if the output is for a code file
func isCodeFile(templateName, output string) bool {
	// Check common code file extensions
	codeExtensions := []string{".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp"}
	for _, ext := range codeExtensions {
		if strings.HasSuffix(templateName, ext) || strings.HasSuffix(templateName, ext+".tpl") {
			return true
		}
	}

	// Check for common code patterns in output
	codePatterns := []string{"package ", "import ", "func ", "class ", "def ", "#include"}
	for _, pattern := range codePatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}

	return false
}
