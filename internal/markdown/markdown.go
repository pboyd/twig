package markdown

import (
	"image/color"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Theme holds styling colors for markdown rendering.
type Theme struct {
	Accent color.Color
	Dim    color.Color
	CodeBg color.Color
	Text   color.Color
}

// Options controls how markdown is rendered.
type Options struct {
	Width  int
	Styled bool
}

// cacheKey is used to deduplicate rendered output.
type cacheKey struct {
	text    string
	width   int
	styled  bool
	inline  bool
}

// Renderer renders markdown text to terminal-friendly strings.
type Renderer struct {
	md    goldmark.Markdown
	theme Theme
	cache map[cacheKey]string
}

// NewRenderer constructs a Renderer with GFM support and an empty cache.
func NewRenderer(theme Theme) *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	return &Renderer{
		md:    md,
		theme: theme,
		cache: make(map[cacheKey]string),
	}
}

// Render converts markdown text to a terminal string. Currently a stub that
// returns the input unchanged.
func (r *Renderer) Render(text string, opts Options) string {
	return text
}

// RenderInline converts inline markdown to a terminal string. Currently a stub
// that returns the input unchanged.
func (r *Renderer) RenderInline(text string, opts Options) string {
	return text
}
